package d2dsoakstat

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
)

// AreaRow is one area's totals from the minute lines (docs/areametrics.md). XP and kills are the differences between
// consecutive minute lines, booked to the area of the later line (S: the minute that crosses an area border goes to
// the new area). GoldPickup sums the `goldPickup` key; HasGold is false when no line of the area had it (old logs).
type AreaRow struct {
	Area       int
	Minutes    float64
	XP, Kills  int64
	GoldPickup int64
	HasGold    bool
}

// MonsterRow is the mean xp share of `combat kill` lines per monster name.
type MonsterRow struct {
	Name  string
	Class string // "" when the line has no class key (old logs)
	Kills int
	XP    int64
}

// AreaStats are the per-area and per-monster tables of one log.
type AreaStats struct {
	Areas    []AreaRow
	Monsters []MonsterRow
	Summary  int // `area summary` lines seen
}

// maxGap caps the time credited to one minute line: a pause in the log must not inflate the hours.
const maxGap = 2 * time.Minute

// ParseAreas reads slog JSON or text lines; lines it does not know are skipped.
func ParseAreas(r io.Reader) (AreaStats, error) {
	var st AreaStats
	rows := map[int]*AreaRow{}
	mons := map[string]*MonsterRow{}
	var prev time.Time
	var prevXP, prevKills int64
	have := false
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		var kv map[string]string
		switch {
		case strings.HasPrefix(line, "{"):
			kv = jsonKV(line)
		case strings.Contains(line, "msg="):
			kv = textKV(line)
		}
		switch kv["msg"] {
		case "area summary":
			st.Summary++
		case "combat kill":
			name := kv["monster"]
			if name == "" {
				continue
			}
			xp, err := strconv.ParseInt(kv["xp"], 10, 64)
			if err != nil {
				continue
			}
			key := name + "/" + kv["class"]
			m := mons[key]
			if m == nil {
				m = &MonsterRow{Name: name, Class: kv["class"]}
				mons[key] = m
			}
			m.Kills++
			m.XP += xp
		case "minute":
			m, ok := minuteFromKV(kv)
			if !ok {
				continue
			}
			kills := int64(m.Kills)
			row := rows[m.Area]
			if row == nil {
				row = &AreaRow{Area: m.Area}
				rows[m.Area] = row
			}
			if g, has := kv["goldPickup"]; has {
				if n, err := strconv.ParseInt(g, 10, 64); err == nil {
					row.GoldPickup += n
					row.HasGold = true
				}
			}
			if have {
				dt := min(m.At.Sub(prev), maxGap)
				if dt > 0 {
					row.Minutes += dt.Minutes()
				}
				if d := int64(m.XP) - prevXP; d > 0 {
					row.XP += d
				}
				if d := kills - prevKills; d > 0 {
					row.Kills += d
				}
			}
			prev, prevXP, prevKills, have = m.At, int64(m.XP), kills, true
		}
	}
	for _, r := range rows {
		st.Areas = append(st.Areas, *r)
	}
	sort.Slice(st.Areas, func(i, j int) bool { return st.Areas[i].Area < st.Areas[j].Area })
	for _, m := range mons {
		st.Monsters = append(st.Monsters, *m)
	}
	sort.Slice(st.Monsters, func(i, j int) bool {
		if st.Monsters[i].Kills != st.Monsters[j].Kills {
			return st.Monsters[i].Kills > st.Monsters[j].Kills
		}
		return st.Monsters[i].Name < st.Monsters[j].Name
	})
	return st, sc.Err()
}

// Text prints the per-area table: XP/h, XP/kill and goldPickup/h; n/a where the log has no data.
func (s AreaStats) Text(name string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "== %s: per area (docs/areametrics.md) ==\n", name)
	if len(s.Areas) == 0 {
		b.WriteString("no minute lines: n/a\n")
	}
	fmt.Fprintf(&b, "%-6s %8s %10s %8s %9s %10s\n", "area", "minutes", "xp/h", "kills", "xp/kill", "goldPick/h")
	for _, a := range s.Areas {
		h := a.Minutes / 60
		xph, xpk, gph := "n/a", "n/a", "n/a"
		if h > 0 {
			xph = fmt.Sprintf("%.0f", float64(a.XP)/h)
			if a.HasGold {
				gph = fmt.Sprintf("%.0f", float64(a.GoldPickup)/h)
			}
		}
		if a.Kills > 0 {
			xpk = fmt.Sprintf("%.1f", float64(a.XP)/float64(a.Kills))
		}
		fmt.Fprintf(&b, "%-6d %8.0f %10s %8d %9s %10s\n", a.Area, a.Minutes, xph, a.Kills, xpk, gph)
	}
	fmt.Fprintf(&b, "area summary lines: %d\n", s.Summary)
	if len(s.Monsters) == 0 {
		b.WriteString("xp per kill by monster: n/a (no combat kill lines)\n")
		return b.String()
	}
	b.WriteString("xp per kill by monster (share of the pooled xp gain, S):\n")
	for _, m := range s.Monsters {
		c := m.Class
		if c == "" {
			c = "n/a"
		}
		fmt.Fprintf(&b, "  %-24s class=%-5s kills=%-5d xp/kill=%.1f\n", m.Name, c, m.Kills, float64(m.XP)/float64(m.Kills))
	}
	return b.String()
}

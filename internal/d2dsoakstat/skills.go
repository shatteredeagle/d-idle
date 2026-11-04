package d2dsoakstat

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// SkillStats are the per-skill fields of docs/research/sorc-act-gate.md section 2 that the bot log can supply
// (V by unit test on fixture lines; the message names are read from internal/combat: "combat decision" act=cast|nova
// at debug level, "combat buff", "combat buff cut by a hit, will retry"). Skills are keyed by the name the log
// carries (from the skill data), never by id.
type SkillStats struct {
	Casts          map[string]int // casts started per skill (combat decision act=cast)
	Buffs          map[string]int // buff casts per skill (combat buff)
	BuffRetries    map[string]int // buff casts cut by a hit that will be recast
	Novas          int            // crowd novas (act=nova)
	NovaMonsters   int            // sum of the monsters counted per nova
	Kills          int            // killtime lines (all skills together)
	Branch, Commit string         // d2d build, from any line that carries the keys; "" = not logged
}

func newSkillStats() SkillStats {
	return SkillStats{Casts: map[string]int{}, Buffs: map[string]int{}, BuffRetries: map[string]int{}}
}

// ParseSkills reads slog JSON or text lines.
func ParseSkills(r io.Reader) (SkillStats, error) {
	s := newSkillStats()
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
		if kv == nil {
			continue
		}
		if s.Branch == "" {
			s.Branch = kv["branch"]
		}
		if s.Commit == "" {
			s.Commit = kv["commit"]
		}
		switch kv["msg"] {
		case "combat decision":
			switch kv["act"] {
			case "cast":
				if kv["skill"] != "" {
					s.Casts[kv["skill"]]++
				}
			case "nova":
				n, err := strconv.Atoi(kv["n"])
				if err == nil && n >= 0 {
					s.Novas++
					s.NovaMonsters += n
				}
			}
		case "combat buff":
			if kv["skill"] != "" {
				s.Buffs[kv["skill"]]++
			}
		case "combat buff cut by a hit, will retry":
			if kv["skill"] != "" {
				s.BuffRetries[kv["skill"]]++
			}
		case "killtime":
			s.Kills++
		}
	}
	return s, sc.Err()
}

const na = "n/a (not logged)"

// Text is the per-skill block of the report. melee is the name of the basic-attack skill in the data ("" = unknown):
// its casts are the staff swings. Fields the log cannot supply print n/a; docs/soakskills.md lists them.
func (s SkillStats) Text(name, melee string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "skills %s: kills %d\n", name, s.Kills)
	names := map[string]bool{}
	for k := range s.Casts {
		names[k] = true
	}
	for k := range s.Buffs {
		names[k] = true
	}
	list := make([]string, 0, len(names))
	for k := range names {
		list = append(list, k)
	}
	sort.Strings(list)
	if len(list) == 0 {
		b.WriteString("  no skill lines in the log (debug level off, or no cast)\n")
	}
	for _, k := range list {
		kind := "attack"
		if s.Buffs[k] > 0 && s.Casts[k] == 0 {
			kind = "buff"
		}
		fmt.Fprintf(&b, "  %s (%s): casts started %d, made effect %s, cut before action frame %s, mana spent %s, kills %s, casts/kill %s, hits/cast %s",
			k, kind, s.Casts[k]+s.Buffs[k], na, na, na, na, s.perKill(s.Casts[k]), na)
		if s.Buffs[k] > 0 {
			fmt.Fprintf(&b, "; buff recasts after a cut %d, seconds absent %s", s.BuffRetries[k], na)
		}
		b.WriteString("\n")
	}
	total := 0
	for _, n := range s.Casts {
		total += n
	}
	fmt.Fprintf(&b, "  all attack casts %d, casts/kill %s\n", total, s.perKill(total))
	if s.Novas > 0 {
		fmt.Fprintf(&b, "  novas %d, monsters per nova %.1f\n", s.Novas, float64(s.NovaMonsters)/float64(s.Novas))
	} else {
		b.WriteString("  novas 0\n")
	}
	if melee == "" {
		fmt.Fprintf(&b, "  staff swings %s (melee skill name not given)\n", na)
	} else {
		fmt.Fprintf(&b, "  staff swings %d\n", s.Casts[melee])
	}
	build := na
	if s.Branch != "" || s.Commit != "" {
		build = s.Branch + " " + s.Commit
	}
	fmt.Fprintf(&b, "  d2d build %s; Lightning cast length ticks %s\n", build, na)
	return b.String()
}

func (s SkillStats) perKill(casts int) string {
	if s.Kills == 0 {
		return "n/a (no kills)"
	}
	return fmt.Sprintf("%.1f", float64(casts)/float64(s.Kills))
}

package d2dsoakstat

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// LifeRow is one area's life statistics from the `zone entry`, `safety decision` and `character died` lines
// (docs/lifelog.md). MinRatio is the lowest life/maxLife seen on any of those lines; HasRatio is false when none had
// both numbers (old logs).
type LifeRow struct {
	Area             int
	Entries          int
	Chickens, Deaths int
	MinRatio         float64
	HasRatio         bool
}

func (r *LifeRow) ratio(life, max int) {
	if max <= 0 {
		return
	}
	v := float64(life) / float64(max)
	if !r.HasRatio || v < r.MinRatio {
		r.MinRatio, r.HasRatio = v, true
	}
}

// ParseLife reads slog JSON or text lines; lines it does not know are skipped.
func ParseLife(r io.Reader) ([]LifeRow, error) {
	rows := map[int]*LifeRow{}
	get := func(a int) *LifeRow {
		if rows[a] == nil {
			rows[a] = &LifeRow{Area: a}
		}
		return rows[a]
	}
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
		area, aerr := strconv.Atoi(kv["area"])
		if aerr != nil {
			continue
		}
		switch kv["msg"] {
		case "zone entry":
			row := get(area)
			row.Entries++
			if l, m, ok := strings.Cut(kv["life"], "/"); ok {
				a, e1 := strconv.Atoi(l)
				b, e2 := strconv.Atoi(m)
				if e1 == nil && e2 == nil {
					row.ratio(a, b)
				}
			}
		case "safety decision":
			row := get(area)
			if kv["action"] == "chicken" {
				row.Chickens++
			}
			row.ratio(atoi(kv["life"]), atoi(kv["maxLife"]))
		case "character died":
			row := get(area)
			row.Deaths++
			// life is 0 on the death line; only the other lines give a ratio
		}
	}
	out := make([]LifeRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Area < out[j].Area })
	return out, sc.Err()
}

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }

// LifeText prints the per-area life table; n/a where the log has no such lines (old logs).
func LifeText(name string, rows []LifeRow) string {
	var b strings.Builder
	fmt.Fprintf(&b, "== %s: life per area (docs/lifelog.md) ==\n", name)
	if len(rows) == 0 {
		b.WriteString("no zone entry lines: n/a\n")
		return b.String()
	}
	fmt.Fprintf(&b, "%-6s %8s %10s %9s %7s\n", "area", "entries", "minLife%", "chickens", "deaths")
	for _, r := range rows {
		ml := "n/a"
		if r.HasRatio {
			ml = fmt.Sprintf("%.0f", r.MinRatio*100)
		}
		fmt.Fprintf(&b, "%-6d %8d %10s %9d %7d\n", r.Area, r.Entries, ml, r.Chickens, r.Deaths)
	}
	return b.String()
}

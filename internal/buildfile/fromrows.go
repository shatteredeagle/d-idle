package buildfile

import (
	"fmt"

	"d2lod-bot/internal/leveler"
)

// FromRows compresses a per-level plan table into goals. Every goal carries when.clvl = the row the table spent it
// in, so Expand reproduces the table exactly as long as it spends every point it has (see notes). Consecutive rows
// that put their single point into the same skill collapse into one goal; rows that bank or mix stay one goal per
// skill per row. Quest points are inferred as the smallest schedule that makes the table affordable.
// notes lists what could not be represented (a finding, never silent).
func FromRows(rows []leveler.Row, class string, rules Rules) (*Build, []string) {
	b := &Build{Schema: Schema, Name: "from-rows", Class: class}
	var notes []string
	start, ok := rules.StartStats(class)
	if !ok {
		return b, []string{"unknown class " + class}
	}
	byClvl := map[int]leveler.Row{}
	lastClvl := 0
	for _, r := range rows {
		byClvl[r.Clvl] = r
		lastClvl = max(lastClvl, r.Clvl)
	}

	// skills
	bank := 0
	cum := map[string]int{}
	lastGoal, lastRow, lastSingle := -1, 0, false
	for L := 2; L <= lastClvl; L++ {
		r := byClvl[L]
		spent, tagged := 0, 0
		for _, s := range r.Skills {
			spent += s.Points
			if s.Quest != "" {
				tagged += s.Points
			}
		}
		if tagged > 0 {
			b.Quest = append(b.Quest, QuestPoint{Clvl: L, Skill: tagged})
		}
		avail := bank + 1 + tagged
		if spent > avail {
			b.Quest = append(b.Quest, QuestPoint{Clvl: L, Skill: spent - avail})
			avail = spent
		}
		bank = avail - spent
		single := len(r.Skills) == 1 && bank == 0
		for _, s := range r.Skills {
			cum[s.Skill] += s.Points
			if single && lastSingle && lastRow == L-1 && b.Skills[lastGoal].Skill == s.Skill {
				b.Skills[lastGoal].To = cum[s.Skill]
				lastRow = L
				continue
			}
			b.Skills = append(b.Skills, Goal{Skill: s.Skill, To: cum[s.Skill], When: &When{Clvl: L}})
			lastGoal, lastRow, lastSingle = len(b.Skills)-1, L, single
		}
		if len(r.Skills) == 0 {
			lastSingle = false
		}
	}

	// stats: runs of identical rows become one goal per stat, capped by per when the run splits the 5 points
	var runVec [4]int
	runStart, runEnd := 0, 0
	have := start
	flush := func() {
		if runStart == 0 {
			return
		}
		total := 0
		for _, v := range runVec {
			total += v
		}
		n := runEnd - runStart + 1
		for i, v := range runVec {
			if v == 0 {
				continue
			}
			have[i] += v * n
			g := StatGoal{Stat: StatNames[i], To: StatTarget{N: have[i]}, When: &When{Clvl: runStart}}
			if v < total {
				g.Per = v
			}
			b.Stats = append(b.Stats, g)
		}
	}
	sbank := 0
	for L := 2; L <= lastClvl; L++ {
		r := byClvl[L]
		sum := 0
		for _, v := range r.Attr {
			sum += v
		}
		avail := sbank + StatPerLevel
		if sum > avail {
			b.Quest = addStat(b.Quest, L, sum-avail)
			avail = sum
		}
		if sum < avail {
			sbank = avail - sum
			notes = append(notes, fmt.Sprintf("clvl %d spends %d of %d stat points: banked stat points are not representable, later rows will differ", L, sum, avail))
		} else {
			sbank = 0
		}
		if runStart != 0 && r.Attr == runVec && runEnd == L-1 {
			runEnd = L
			continue
		}
		flush()
		runVec, runStart, runEnd = r.Attr, L, L
		if sum == 0 {
			runStart = 0
		}
	}
	flush()
	return b, notes
}

func addStat(q []QuestPoint, clvl, n int) []QuestPoint {
	for i := range q {
		if q[i].Clvl == clvl {
			q[i].Stat += n
			return q
		}
	}
	return append(q, QuestPoint{Clvl: clvl, Stat: n})
}

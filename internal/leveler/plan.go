package leveler

// The pure part of point spending: which stat and skill points to buy next. No memory, no clicks.

// Kind says what an Action buys.
type Kind int

const (
	KindStat Kind = iota
	KindSkill
)

// Action is one point to spend.
type Action struct {
	Kind  Kind
	Attr  int    // KindStat: Str, Dex, Vit, Ene
	Skill string // KindSkill: skills.txt name
}

func (a Action) String() string {
	if a.Kind == KindStat {
		return "+1 " + attrNames[a.Attr]
	}
	return "+1 " + a.Skill
}

// Have is what the character owns now.
type Have struct {
	Attr     [numAttr]int   // base attribute values (not counting items)
	Start    [numAttr]int   // class starting values (charstats.txt)
	Skills   map[string]int // base skill levels by skills.txt name
	Level    int
	StatPts  int // unspent
	SkillPts int // unspent
	// Floor is the minimum base value per attribute the character should reach before the table gets its points
	// (gear requirements, see Leveler.NeedStats and classFloor). Zero entries mean none.
	Floor [numAttr]int
}

// Rule says whether the game would accept one more point in skill name given the levels the character will have
// by then (planned points included). Nil means "always".
type Rule func(name string, level int, levels map[string]int) bool

// Plan returns the points to spend now, in order: stat points first-come per table order, skills likewise.
// It counts what the character already owns against the table (so a rerun after a partial spend, a quest point,
// or a hand-spent point continues where the table says) and walks the table past the current level when more points
// are unspent than the level grants (quest points pull future rows forward, data/quest-points.json policy).
// A skill the Rule refuses is skipped together with its later points; skip lists skills refused by the game earlier.
func (b *Build) Plan(h Have, rule Rule, skip map[string]bool) []Action {
	var out []Action
	remain := [numAttr]int{}
	for a := range remain {
		remain[a] = max(0, h.Attr[a]-h.Start[a])
	}
	levels := map[string]int{}
	owned := map[string]int{}
	for k, v := range h.Skills {
		levels[k], owned[k] = v, v
	}
	stat, skill := 0, 0
	blocked := map[string]bool{}
	// Floors first: a missing requirement costs the table nothing it would not lose anyway, and no gear can be worn without it.
	for a := 0; a < numAttr; a++ {
		for n := h.Floor[a] - h.Attr[a]; n > 0 && stat < h.StatPts; n-- {
			out = append(out, Action{Kind: KindStat, Attr: a})
			stat++
		}
	}
	for _, r := range b.Rows {
		for a := 0; a < numAttr; a++ {
			for i := 0; i < r.Attr[a]; i++ {
				if remain[a] > 0 {
					remain[a]--
					continue
				}
				if stat < h.StatPts {
					out = append(out, Action{Kind: KindStat, Attr: a})
					stat++
				}
			}
		}
		for _, sp := range r.Skills {
			for i := 0; i < sp.Points; i++ {
				if owned[sp.Skill] > 0 {
					owned[sp.Skill]--
					continue
				}
				if skill >= h.SkillPts || blocked[sp.Skill] {
					continue
				}
				if skip[sp.Skill] || (rule != nil && !rule(sp.Skill, h.Level, levels)) {
					blocked[sp.Skill] = true
					continue
				}
				out = append(out, Action{Kind: KindSkill, Skill: sp.Skill})
				levels[sp.Skill]++
				skill++
			}
		}
		if stat >= h.StatPts && skill >= h.SkillPts {
			break
		}
	}
	// stat actions first (character panel), then skills (skill tree); each group keeps table order
	var stats, skills []Action
	for _, a := range out {
		if a.Kind == KindStat {
			stats = append(stats, a)
		} else {
			skills = append(skills, a)
		}
	}
	return append(stats, skills...)
}

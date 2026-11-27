package leveler

import "strconv"

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
	// Prereq is true for a skill bought because a planned skill needs it (skills.txt reqskill1-3), not for its own
	// sake: the allocator spends it even when the fighter does not use it (Sacrifice before Zeal).
	Prereq bool
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
	// QuestPts is how many skill points the character holds beyond what its levels grant (hard points in the skill
	// tree plus unspent, minus clvl-1): the quest bonus points received so far (QuestPoints).
	QuestPts int
	// Req lists the prerequisites of a skill (skills.txt reqskill1-3 by name, DBReq). Nil: no prerequisite handling.
	Req func(name string) []string
}

// Rule says whether the game would accept one more point in skill name given the levels the character will have
// by then (planned points included). Nil means "always".
type Rule func(name string, level int, levels map[string]int) bool

// Plan returns the points to spend now, in order: stat points first-come per table order, skills likewise.
// It counts what the character already owns against the table (so a rerun after a partial spend, a quest point,
// or a hand-spent point continues where the table says) and walks the table past the current level when more points
// are unspent than the level grants (quest points pull future rows forward, data/quest-points.json policy).
// A skill the Rule refuses is skipped together with its later points; skip lists skills refused by the game earlier.
// Points tagged with a quest count only once the character holds that many quest points (the first h.QuestPts tagged
// points in table order): the rows count level points, and a bonus point is spent as soon as it is granted.
//
// Prerequisites (h.Req): a planned skill whose prerequisites are not owned gets them bought first, ahead of it, so a
// missing prerequisite never stalls the table; every skill that is a prerequisite of a table skill is marked Prereq.
func (b *Build) Plan(h Have, rule Rule, skip map[string]bool) []Action {
	acts, _ := b.PlanWhy(h, rule, skip)
	return acts
}

// PlanWhy is Plan plus the planned skills it could not buy now, each with the blocker (a level, a refused
// prerequisite, a skill the game refused earlier), for one log line per blocker.
func (b *Build) PlanWhy(h Have, rule Rule, skip map[string]bool) ([]Action, map[string]string) {
	var out []Action
	why := map[string]string{}
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
	questLeft := h.QuestPts
	needed := b.prereqSet(h.Req)
	ok := func(name string) bool { return !skip[name] && (rule == nil || rule(name, h.Level, levels)) }
	// buyReqs buys the missing prerequisites of name (theirs first), as far as points and the rule allow.
	var buyReqs func(name string, depth int)
	buyReqs = func(name string, depth int) {
		if h.Req == nil || depth > 4 {
			return
		}
		for _, rq := range h.Req(name) {
			if levels[rq] > 0 || skill >= h.SkillPts {
				continue
			}
			buyReqs(rq, depth+1)
			if skill < h.SkillPts && levels[rq] == 0 && ok(rq) {
				out = append(out, Action{Kind: KindSkill, Skill: rq, Prereq: true})
				levels[rq]++
				owned[rq]++ // a later table row for rq counts this point as owned
				skill++
			}
		}
	}
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
				if sp.Quest != "" {
					if questLeft <= 0 {
						continue
					}
					questLeft--
				}
				if owned[sp.Skill] > 0 {
					owned[sp.Skill]--
					continue
				}
				if skill >= h.SkillPts || blocked[sp.Skill] {
					continue
				}
				if !ok(sp.Skill) {
					buyReqs(sp.Skill, 0)
				}
				if skill >= h.SkillPts {
					continue
				}
				if !ok(sp.Skill) {
					blocked[sp.Skill] = true
					why[sp.Skill] = b.blocker(h, sp.Skill, levels, skip)
					continue
				}
				out = append(out, Action{Kind: KindSkill, Skill: sp.Skill, Prereq: needed[sp.Skill]})
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
	return append(stats, skills...), why
}

// prereqSet is every skill that is a prerequisite (directly or through others) of a skill in the table.
func (b *Build) prereqSet(req func(string) []string) map[string]bool {
	set := map[string]bool{}
	if req == nil {
		return set
	}
	var add func(name string, depth int)
	add = func(name string, depth int) {
		for _, rq := range req(name) {
			if !set[rq] && depth < 8 {
				set[rq] = true
				add(rq, depth+1)
			}
		}
	}
	for _, r := range b.Rows {
		for _, sp := range r.Skills {
			add(sp.Skill, 0)
		}
	}
	return set
}

// blocker names why a planned skill cannot be bought now.
func (b *Build) blocker(h Have, name string, levels map[string]int, skip map[string]bool) string {
	if skip[name] {
		return "refused by the game earlier"
	}
	if h.Req != nil {
		for _, rq := range h.Req(name) {
			if levels[rq] == 0 {
				return "prerequisite " + rq + " cannot be bought now"
			}
		}
	}
	return "the game rule refuses it at clvl " + strconv.Itoa(h.Level) + " (required level or max level)"
}

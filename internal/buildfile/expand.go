package buildfile

import (
	"fmt"
	"strings"

	"d2lod-bot/internal/leveler"
)

// StatPerLevel is the stat points per level up (V: every row of the repo's plan tables sums to 5).
const StatPerLevel = 5

// Expansion is the result of Expand. Rows are leveler.Row values for clvl 2..99. Without a respec Pre holds all of
// them and Respec is 0. With one, Pre covers clvl 2..Respec (the pre-respec path), Pool is the whole post-respec
// skill allocation at the respec level (everything refunded and respent), and Post covers Respec+1..99.
type Expansion struct {
	Pre    []leveler.Row
	Respec int
	Pool   []leveler.SkillPts
	Post   []leveler.Row
	// Unplaced lists "skill n" steps never affordable or unlocked by clvl 99; Validate turns them into problems.
	Unplaced    []string
	StatUnspent int
}

// ToLeveler adapts an Expansion to a leveler.Build (Rows = Pre then Post; the respec fields as the table format has them).
func (e *Expansion) ToLeveler(b *Build) *leveler.Build {
	rows := append(append([]leveler.Row{}, e.Pre...), e.Post...)
	return &leveler.Build{Name: b.Name, Class: b.Class, Rows: rows, Respec: e.Respec, RespecPool: e.Pool}
}

type step struct {
	skill string
	n     int // hard points after this step
	min   int // not before this clvl
	until int // dropped after this clvl (0 = never)
	hold  bool
	after *After
	info  SkillInfo
	done  bool
}

type goalCtx struct {
	Goal
	min, until int
}

// Expand turns a build into per-level rows. Deterministic; assumes Validate found nothing (unknown names are errors).
// Quest points come from b.Quest only.
func Expand(b *Build, rules Rules) (*Expansion, error) {
	start, ok := rules.StartStats(b.Class)
	if !ok {
		return nil, fmt.Errorf("buildfile: unknown class %q", b.Class)
	}
	statRows, unspent := expandStats(b, start)
	out := &Expansion{StatUnspent: unspent}

	var pre []goalCtx
	for _, p := range b.Phases {
		for _, g := range p.Skills {
			c := goalCtx{Goal: g, min: max(p.FromClvl, whenClvl(g)), until: p.UntilClvl}
			pre = append(pre, c)
		}
	}
	var main []goalCtx
	for _, g := range b.Skills {
		main = append(main, goalCtx{Goal: g, min: whenClvl(g)})
	}

	last := MaxClvl
	first := pre
	if b.RespecAt != nil {
		last = b.RespecAt.Clvl
	} else {
		first = append(first, main...)
	}
	e, err := newEngine(first, rules, b.Quest)
	if err != nil {
		return nil, err
	}
	rows := e.run(2, last, 0, statRows)
	out.Pre = rows
	out.Unplaced = e.unplaced()
	if b.RespecAt == nil {
		return out, nil
	}
	out.Respec = last
	e2, err := newEngine(main, rules, b.Quest)
	if err != nil {
		return nil, err
	}
	// the respec level's row holds the whole refunded pool; later rows continue from it
	post := e2.run(last, MaxClvl, earnedSkill(b.Quest, last), statRows)
	out.Pool = post[0].Skills
	out.Post = post[1:]
	out.Unplaced = e2.unplaced()
	return out, nil
}

func whenClvl(g Goal) int {
	if g.When == nil {
		return 0
	}
	return g.When.Clvl
}

type engine struct {
	steps  []*step
	levels map[string]int
	quest  []QuestPoint
}

func newEngine(goals []goalCtx, rules Rules, quest []QuestPoint) (*engine, error) {
	e := &engine{levels: map[string]int{}, quest: quest}
	planned := map[string]int{}
	var addPre func(name string, c goalCtx, eff int, seen map[string]bool) error
	addPre = func(name string, c goalCtx, eff int, seen map[string]bool) error {
		in, ok := rules.Skill(name)
		if !ok {
			return fmt.Errorf("buildfile: unknown skill %q", name)
		}
		if planned[in.Name] > 0 {
			return nil
		}
		if seen[in.Name] {
			return fmt.Errorf("buildfile: prerequisite cycle at %q", in.Name)
		}
		seen[in.Name] = true
		for _, r := range in.ReqSkills {
			if err := addPre(r, c, max(eff, in.ReqLevel), seen); err != nil {
				return err
			}
		}
		// as late as possible: not before the dependent can be spent itself
		e.steps = append(e.steps, &step{skill: in.Name, n: 1, min: eff, until: c.until, hold: c.Hold, info: in})
		planned[in.Name] = 1
		return nil
	}
	for _, c := range goals {
		in, ok := rules.Skill(c.Skill)
		if !ok {
			return nil, fmt.Errorf("buildfile: unknown skill %q", c.Skill)
		}
		for n := planned[in.Name] + 1; n <= min(c.To, in.maxLevel()); n++ {
			if n == 1 {
				for _, r := range in.ReqSkills {
					if err := addPre(r, c, max(c.min, in.ReqLevel), map[string]bool{in.Name: true}); err != nil {
						return nil, err
					}
				}
			}
			var af *After
			if c.When != nil {
				af = c.When.After
			}
			e.steps = append(e.steps, &step{skill: in.Name, n: n, min: c.min, until: c.until, hold: c.Hold, after: af, info: in})
			planned[in.Name] = n
		}
	}
	return e, nil
}

func (e *engine) ready(s *step, clvl int) bool {
	if clvl < s.min || clvl < s.info.ReqLevel || e.levels[s.skill] != s.n-1 {
		return false
	}
	if s.after != nil && e.levels[canon(e.steps, s.after.Skill)] < s.after.Level {
		return false
	}
	for _, r := range s.info.ReqSkills {
		if e.levels[canon(e.steps, r)] < 1 {
			return false
		}
	}
	return true
}

// canon maps a case-variant name to the canonical one used by the steps (names in After/ReqSkills may differ in case).
func canon(steps []*step, name string) string {
	for _, s := range steps {
		if strings.EqualFold(s.skill, name) {
			return s.skill
		}
	}
	return name
}

// run spends points for clvl from..to. bank0, when > 0, replaces the normal gain at clvl from (a respec refunds
// everything earned so far). Level points are spent before quest points, and a point paid by a quest carries that
// quest's id (questIDs), so a row never holds more untagged points than the levels grant.
func (e *engine) run(from, to, bank0 int, stats map[int][4]int) []leveler.Row {
	var rows []leveler.Row
	bank, issued := 0, 0
	var quests []string // pending quest points, oldest first
	for L := from; L <= to; L++ {
		if L == from && bank0 > 0 {
			bank += bank0
			issued = earnedSkill(e.quest, L) - (L - 1)
		} else {
			q, _ := questAt(e.quest, L)
			bank++
			for ; q > 0; q-- {
				quests = append(quests, questID(issued))
				issued++
			}
		}
		row := leveler.Row{Clvl: L, Attr: stats[L]}
		for _, s := range e.steps {
			if !s.done && s.until > 0 && L > s.until {
				s.done = true // dropped, not spent
			}
		}
		for bank > 0 || len(quests) > 0 {
			var pick *step
			for _, s := range e.steps {
				if s.done {
					continue
				}
				if e.ready(s, L) {
					pick = s
					break
				}
				if s.hold {
					break // an unfinished hold goal blocks everything after it
				}
			}
			if pick == nil {
				break
			}
			pick.done = true
			e.levels[pick.skill]++
			tag := ""
			if bank > 0 {
				bank--
			} else {
				tag, quests = quests[0], quests[1:]
			}
			if n := len(row.Skills); n > 0 && row.Skills[n-1].Skill == pick.skill && row.Skills[n-1].Quest == tag {
				row.Skills[n-1].Points++
			} else {
				row.Skills = append(row.Skills, leveler.SkillPts{Skill: pick.skill, Points: 1, Quest: tag})
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func (e *engine) unplaced() []string {
	var out []string
	for _, s := range e.steps {
		if !s.done {
			out = append(out, fmt.Sprintf("%s %d", s.skill, s.n))
		}
	}
	return out
}

// expandStats walks the ordered stat goals for clvl 2..99. Points no open goal takes stay banked; the leftover is
// returned (nonzero means the file has no "rest" goal and ends below budget).
func expandStats(b *Build, start [4]int) (map[int][4]int, int) {
	rows := map[int][4]int{}
	cur := start
	bank := 0
	for L := 2; L <= MaxClvl; L++ {
		_, sq := questAt(b.Quest, L)
		bank += StatPerLevel + sq
		var add [4]int
		for _, g := range b.Stats {
			i := statIndex(g.Stat)
			if i < 0 || bank == 0 || (g.When != nil && g.When.Clvl > L) {
				continue
			}
			take := bank
			if g.Per > 0 {
				take = min(take, g.Per)
			}
			if !g.To.Rest {
				take = min(take, g.To.N-cur[i])
			}
			if take <= 0 {
				continue
			}
			add[i] += take
			cur[i] += take
			bank -= take
		}
		rows[L] = add
	}
	return rows, bank
}

// Cumulative returns the hard points per skill and the stat points added through clvl.
func Cumulative(rows []leveler.Row, clvl int) (map[string]int, [4]int) {
	sk := map[string]int{}
	var at [4]int
	for _, r := range rows {
		if r.Clvl > clvl {
			break
		}
		for _, s := range r.Skills {
			sk[s.Skill] += s.Points
		}
		for i := range at {
			at[i] += r.Attr[i]
		}
	}
	return sk, at
}

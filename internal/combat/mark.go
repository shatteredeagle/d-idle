package combat

import (
	"context"
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/input"
)

// Mark driver (card dlv-mark; docs/design/fable-review-2026-11-21/combat-engine-d-cards.md, wave 1). A mark is a skill
// that puts a state on enemies: the ten curses, Taunt, Howl, Battle Cry, Cloak of Shadows. The rows are profile data
// (Profile.Marks); a Necromancer profile without marks rows reads its curse rules as rows of group "curse". Go names
// no skill and no class:
//   - target choice by the row's When ("any", "pack", "eliteOrBoss", "magicResist"; "" = never cast).
//   - area cover from the row's Radius: the target with the most unmarked hostiles inside it wins, and a hostile
//     inside the radius of a recent cast of the same group counts as covered.
//   - state observed on targets: the state id is the skill's Skills.txt aurastate (data.Skill.StateID, else the row's
//     State). It is trusted only after it was seen on a unit the driver marked; until then the cast memory decides.
//   - one mark group per target: a state of any row of the same group on the unit counts as marked (a second curse
//     replaces the first in the game, S: docs/summon.md curses).
//   - Self rows (war cries, Cloak of Shadows; U which rows the owner adds) are cast at the own feet and cover the
//     hostiles within Radius of the hero.
//
// Knob combat.deliveryUpkeep (the delivery driver knob shared with upkeep.go): 0 the old Necromancer curse path, 1
// shadow (the old path acts, the driver's choice is logged), 2 the driver acts and the old curse path stands down.
// No uptime number is claimed before the fidelity A2 verdict; on d2d the put rule deviates (the card's risk).

// MarkRule is one mark row of a profile.
type MarkRule struct {
	Name  string `json:"name"`
	When  string `json:"when"`
	Group string `json:"group,omitempty"` // rows of one group replace each other on a unit; "" = the row's own name
	// Radius is the area a cast covers, in subtiles (0: markCoverDefault); Range the farthest target (0: curseRange).
	Radius int  `json:"radius,omitempty"`
	Range  int  `json:"range,omitempty"`
	Self   bool `json:"self,omitempty"` // cast at the own feet
	// State is the monster state the mark sets when the skill table has none (0: unknown).
	State int `json:"state,omitempty"`
	// Source says where Radius, Range and State come from (V, S or U); required when one of them is set.
	Source string `json:"source,omitempty"`
}

const (
	markCoverDefault = curseCover // U radius, V live 2026-10-29 that one cast per unit burns mana (necro.go)
	markTriesMax     = 3          // casts at one unit with nothing registering before the driver gives up on it
	markStateGrace   = 2 * time.Second
)

var markWhen = map[string]bool{"": true, "any": true, "pack": true, "eliteOrBoss": true, "magicResist": true}

func (r MarkRule) group() string {
	if r.Group != "" {
		return r.Group
	}
	return r.Name
}

func (r MarkRule) radius() float64 { return float64(rule(r.Radius, markCoverDefault)) }
func (r MarkRule) reach() float64  { return float64(rule(r.Range, curseRange)) }

// markRows are the profile's mark rows: Marks, else the Necromancer curse rules as rows of group "curse".
func (p *Profile) markRows() []MarkRule {
	if p == nil {
		return nil
	}
	if len(p.Marks) > 0 || p.Necro == nil {
		return p.Marks
	}
	nd := p.Necro
	out := make([]MarkRule, 0, len(nd.Curses))
	for _, c := range nd.Curses {
		out = append(out, MarkRule{Name: c.Name, When: c.When, Group: "curse", State: c.State,
			Radius: int(nd.curseCover()), Range: int(nd.curseRange())})
	}
	return out
}

// isMarkRow says the name is a mark row of the profile (the buff pass never casts it at the own feet in mode 2).
func (p *Profile) isMarkRow(name string) bool {
	for _, r := range p.markRows() {
		if r.Name == name {
			return true
		}
	}
	return false
}

// validateMarks checks the marks rows at load time: an error names the profile and the row.
func validateMarks(ps []Profile) error {
	for _, p := range ps {
		for i, r := range p.Marks {
			where := fmt.Sprintf("profile %s/%s marks[%d]", p.Class, p.Build, i)
			switch {
			case r.Name == "":
				return fmt.Errorf("%s: no name", where)
			case !markWhen[r.When]:
				return fmt.Errorf("%s (%s): unknown when %q", where, r.Name, r.When)
			case r.Radius < 0 || r.Range < 0 || r.State < 0:
				return fmt.Errorf("%s (%s): negative number", where, r.Name)
			case (r.Radius > 0 || r.Range > 0 || r.State > 0) && r.Source == "":
				return fmt.Errorf("%s (%s): radius, range or state set without a source tag", where, r.Name)
			}
		}
	}
	return nil
}

// markState is the driver's memory: learned state ids, recent casts for the area cover, tries per unit and group.
type markState struct {
	learned     map[int]bool
	casts       []markCast
	tries       map[markKey]int
	shadowDiffs int
	shadowLast  string
}

type markKey struct {
	unit  uint32
	group string
}

type markCast struct {
	group  string
	state  int
	x, y   int
	radius float64
	at     time.Time
	life   time.Duration
}

func (m *markState) init() {
	if m.learned == nil {
		m.learned, m.tries = map[int]bool{}, map[markKey]int{}
	}
}

// markChoice is what the driver would cast now.
type markChoice struct {
	row    MarkRule
	sp     SkillSpec
	at     input.Point
	unit   uint32 // 0 for a self cast
	name   string
	covers int // unmarked hostiles inside the radius
}

// markStateID is the monster state the row's skill sets: the table first, the row second (0: unknown).
func (f *Fighter) markStateID(r MarkRule, sp SkillSpec) int {
	if f.cat != nil {
		if s := f.cat.Skill(sp.ID); s != nil && s.StateID > 0 {
			return s.StateID
		}
	}
	return r.State
}

// markLife is how long a cast is remembered when the state is not readable: the skill's state duration at its level
// from the table (U whether curses fill auralencalc), else the curse memory default.
func (f *Fighter) markLife(sp SkillSpec) time.Duration {
	if f.cat != nil {
		if s := f.cat.Skill(sp.ID); s != nil {
			if fr, ok := s.AuraFrames(max(sp.Level, 1)); ok {
				return time.Duration(fr) * time.Second / 25
			}
		}
	}
	return curseLifeSec * time.Second
}

// marked says whether the unit already carries a mark of the row's group: a learned (or own) state seen on it, the
// driver gave up on it, or a recent cast of the group covers it.
func (f *Fighter) marked(rows []MarkRule, known map[string]SkillSpec, u d2.Unit, group string) bool {
	for _, r := range rows {
		if r.group() != group {
			continue
		}
		sp, ok := known[r.Name]
		if !ok {
			continue
		}
		if st := f.markStateID(r, sp); st > 0 && u.Monster.HasState(st) && f.mark.learned[st] {
			return true
		}
	}
	if f.mark.tries[markKey{u.ID, group}] >= markTriesMax {
		return true // the cast does not register on it (no line, immune, too close): do not burn mana
	}
	now := f.now()
	for _, c := range f.mark.casts {
		if c.group != group || dist(u.X, u.Y, c.x, c.y) > c.radius {
			continue
		}
		age := now.Sub(c.at)
		if c.state > 0 && f.mark.learned[c.state] {
			if age < markStateGrace {
				return true // just cast: give the state time to show
			}
			continue
		}
		if age < c.life {
			return true
		}
	}
	return false
}

func markWhenHolds(when string, t Target) bool {
	switch when {
	case "any", "pack":
		return true
	case "eliteOrBoss":
		return t.Boss || t.Elite
	case "magicResist":
		return t.HasRes && t.Res[data.Physical] < 50 && t.Res[data.Magic] >= 50
	}
	return false
}

func markable(t Target) bool {
	return t.LifePct >= 25 && t.Unit.Mode != d2.MonModeDead && t.Unit.Mode != d2.MonModeDeath
}

// markChoose walks the rows in priority order and returns the first usable row with a target worth a cast.
func (f *Fighter) markChoose(p *d2.Player, rows []MarkRule, known map[string]SkillSpec, hs []Target) (markChoice, bool) {
	f.mark.init()
	for _, r := range rows {
		sp, ok := known[r.Name]
		if r.When == "" || !ok || f.expectDemoted(r.Name) || (sp.Key == "" && p.RightSkill != sp.ID) || sp.Cost*2 > p.Mana {
			continue
		}
		if c, ok := f.markTarget(p, rows, known, r, hs); ok {
			c.row, c.sp = r, sp
			return c, true
		}
	}
	return markChoice{}, false
}

// markTarget scores the targets of one row. A cast must cover a boss or elite, or at least two unmarked hostiles
// ("pack" and every area row; a lone trash monster is not worth the mana, V live 2026-10-29).
func (f *Fighter) markTarget(p *d2.Player, rows []MarkRule, known map[string]SkillSpec, r MarkRule, hs []Target) (markChoice, bool) {
	g, rad := r.group(), r.radius()
	open := make([]Target, 0, len(hs))
	for _, t := range hs {
		if markable(t) && !f.marked(rows, known, t.Unit, g) {
			open = append(open, t)
		}
	}
	around := func(x, y int) (n int, worth bool) {
		for _, o := range open {
			if dist(x, y, o.Unit.X, o.Unit.Y) <= rad {
				n++
				worth = worth || markWhenHolds(r.When, o)
			}
		}
		return n, worth
	}
	if r.Self {
		n, worth := around(p.X, p.Y)
		elite := false
		for _, o := range open {
			elite = elite || (o.Boss || o.Elite) && dist(p.X, p.Y, o.Unit.X, o.Unit.Y) <= rad
		}
		if !worth || (n < 2 && !elite) {
			return markChoice{}, false
		}
		return markChoice{at: input.Point{X: p.X, Y: p.Y}, name: "self", covers: n}, true
	}
	var best markChoice
	bestScore, found := 0, false
	for _, t := range open {
		if t.Dist > r.reach() || !markWhenHolds(r.When, t) {
			continue
		}
		n, _ := around(t.Unit.X, t.Unit.Y)
		if n < 2 && !t.Boss && !t.Elite {
			continue
		}
		score := n*10 - int(t.Dist/4)
		if t.Boss || t.Elite {
			score += 100
		}
		if !found || score > bestScore {
			best = markChoice{at: input.Point{X: t.Unit.X, Y: t.Unit.Y}, unit: t.Unit.ID, name: t.Name, covers: n}
			bestScore, found = score, true
		}
	}
	return best, found
}

// markTick is the fight-loop hook of the driver. true = a mark was cast this tick.
func (f *Fighter) markTick(ctx context.Context, snap *d2.Snapshot) (bool, error) {
	mode := f.upkeepMode()
	p := snap.Player
	if mode == 0 || p == nil || p.InTown() {
		return false, nil
	}
	rows := f.profileFor(p).markRows()
	if len(rows) == 0 {
		return false, nil
	}
	known := KnownSkills(p, f.cat, f.keys)
	c, ok := f.markChoose(p, rows, known, Hostiles(snap, f.hostileCtx(p)))
	if mode == 1 {
		f.markShadow(c, ok)
		return false, nil
	}
	if !ok {
		return false, nil
	}
	return true, f.markCast(ctx, snap, c)
}

// markShadow logs the driver's choice once per change (mode 1); the old path acts.
func (f *Fighter) markShadow(c markChoice, ok bool) {
	key := ""
	if ok {
		key = fmt.Sprint(c.sp.ID, c.unit)
	}
	if key == f.mark.shadowLast {
		return
	}
	f.mark.shadowLast = key
	if ok {
		f.mark.shadowDiffs++
		f.info("mark shadow", "skill", c.sp.Name, "target", c.name, "covers", c.covers)
	}
}

// MarkShadowChoices is how many distinct choices mode 1 logged; it is for telemetry and tests.
func (f *Fighter) MarkShadowChoices() int { return f.mark.shadowDiffs }

// markCast casts the chosen mark, remembers it for the area cover and re-reads the target to learn whether the state
// id is readable.
func (f *Fighter) markCast(ctx context.Context, snap *d2.Snapshot, c markChoice) error {
	if err := f.cast(snap, c.sp, c.at); err != nil {
		return err
	}
	g, st := c.row.group(), f.markStateID(c.row, c.sp)
	f.mark.casts = append(f.markLive(), markCast{group: g, state: st, x: c.at.X, y: c.at.Y, radius: c.row.radius(), at: f.now(), life: f.markLife(c.sp)})
	if c.unit != 0 {
		f.mark.tries[markKey{c.unit, g}]++
	}
	f.info("mark", "skill", c.sp.Name, "target", c.name, "covers", c.covers, "stateReadable", st > 0 && f.mark.learned[st])
	if err := f.sleep(ctx, ms(f.tune().CastGapMs)+ms(300)); err != nil {
		return err
	}
	after, err := f.read(ctx)
	if err != nil || after == nil || st == 0 {
		return nil
	}
	for _, u := range after.Monsters {
		if dist(u.X, u.Y, c.at.X, c.at.Y) <= c.row.radius() && u.Monster.HasState(st) {
			f.mark.learned[st] = true
			delete(f.mark.tries, markKey{u.ID, g})
		}
	}
	return nil
}

// markLive drops casts whose memory ran out.
func (f *Fighter) markLive() []markCast {
	now, out := f.now(), f.mark.casts[:0]
	for _, c := range f.mark.casts {
		if now.Sub(c.at) < c.life {
			out = append(out, c)
		}
	}
	return out
}

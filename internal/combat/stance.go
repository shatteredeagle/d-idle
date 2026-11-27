package combat

import (
	"context"
	"fmt"
	"math"

	"d2lod-bot/internal/arbiter"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// Army stance (card dlv-stance-army; docs/design/fable-review-2026-11-21/combat-engine-d-cards.md, S8 and S11 of the
// Necromancer note). A profile with a stance block of kind behindArmy keeps its hero on the far side of its own army:
// the spot lies Behind subtiles past the army's centroid on the line from the hostiles' centroid. In a boss fight a
// census under BossMin turns the step into a retreat from the boss. The army is the minion driver's census (minions.go),
// so the stance and the arbiter facts read one source.
//
// Knob combat.deliveryUpkeep (shared with the upkeep and minion drivers): 0 no stance, 1 shadow (the step the stance
// would take is logged and counted, nothing moves), 2 the stance steps. A stance step is a non-cast movement, so it only
// takes a tick right after a cast (evadeOK) and the cast-first watchdog bounds how long the hero can hide.

// Pack stance (card dlv-stance-pack): kind insidePack stands on a free cell StandOff subtiles from the hostiles'
// centroid and casts in place (a self-centred area skill such as a hammer or a nova), and leaves when LeaveSurround
// hostiles close within stancePackContact or life falls under LeaveLifePct. It is the dangerous stance: knob 2 waits
// for the survival model's contact archetype (roadmap R6) to judge it; the knob, not the data, is the rollback. H1 (does
// the clicked point of Blessed Hammer matter) is a live look, so the cast aim stays the target.

// StanceBehindArmy is the stance kind of a summoner; StanceInsidePack of an area caster standing in the pack.
const (
	StanceBehindArmy = "behindArmy"
	StanceInsidePack = "insidePack"
)

const (
	stanceSlackDefault = 3  // subtiles the hero may stray from its spot
	stanceThreatRange  = 30 // hostiles farther than this do not shape the line
	stancePackContact  = 3  // hostiles this close count as surrounding the hero (U, the profile's surround rule uses 3)
	stancePackMin      = 2  // fewer hostiles than this are no pack: the fight puts the hero where it likes
)

// ArmyStance is a profile's stance block.
type ArmyStance struct {
	Kind   string `json:"kind"`
	Behind int    `json:"behind"`          // subtiles behind the army centroid, away from the hostiles
	Slack  int    `json:"slack,omitempty"` // stray allowed before the hero steps back (0: stanceSlackDefault)
	// BossMin: with a boss in the fight, retreat from it while fewer minions stand than this (0: never).
	BossMin int `json:"bossMin,omitempty"`
	// insidePack: StandOff subtiles from the hostiles' centroid; leave with LeaveSurround hostiles within
	// stancePackContact or life under LeaveLifePct percent; MinPack hostiles make a pack (0: stancePackMin).
	StandOff      int    `json:"standOff,omitempty"`
	LeaveSurround int    `json:"leaveSurround,omitempty"`
	LeaveLifePct  int    `json:"leaveLifePct,omitempty"`
	MinPack       int    `json:"minPack,omitempty"`
	Source        string `json:"source"` // V, S or U first: where the numbers come from
}

func (s *ArmyStance) slack() float64 {
	if s.Slack > 0 {
		return float64(s.Slack)
	}
	return stanceSlackDefault
}

// validateStance checks the stance blocks at load time: an error names the profile.
func validateStance(ps []Profile) error {
	for i := range ps {
		p := &ps[i]
		s := p.Stance
		if s == nil {
			continue
		}
		switch {
		case s.Kind == StanceInsidePack:
			if s.StandOff < 1 || s.LeaveSurround < 1 || s.LeaveLifePct < 1 || s.LeaveLifePct > 100 || s.MinPack < 0 || !validSource(s.Source) {
				return fmt.Errorf("profile %q: stance %s needs standOff, leaveSurround >= 1, leaveLifePct 1..100, minPack >= 0 and a source tag (V, S or U)", p.ProfileID(), s.Kind)
			}
		case s.Kind != StanceBehindArmy:
			return fmt.Errorf("profile %q: stance kind %q unknown", p.ProfileID(), s.Kind)
		case s.Behind < 1 || s.Slack < 0 || s.BossMin < 0 || !validSource(s.Source):
			return fmt.Errorf("profile %q: stance needs behind >= 1, slack and bossMin >= 0 and a source tag (V, S or U)", p.ProfileID())
		case p.minionParams() == nil:
			return fmt.Errorf("profile %q: stance %s without minion groups", p.ProfileID(), s.Kind)
		}
	}
	return nil
}

// Army is the hero's army as the census sees it.
type Army struct {
	Count, Want int
	// Strength is the standing minions' life in percent of one full minion (three unhurt skeletons: 300).
	Strength  int
	Centroid  input.Point
	Front     input.Point // the minion nearest the hostiles' centroid; the centroid with no hostile near
	FrontDist int         // subtiles from the hero to Front
	UpkeepDue bool        // the minion driver would summon now (after-fight rules, no walk)
}

// BelowWant says fewer minions stand than the groups want.
func (a Army) BelowWant() bool { return a.Count < a.Want }

// army reads the hero's army. ok is false when the profile has no minion groups (a class without an army).
func (f *Fighter) army(snap *d2.Snapshot) (Army, bool) {
	if snap == nil || snap.Player == nil {
		return Army{}, false
	}
	p := snap.Player
	pr := f.profileFor(p)
	mp := pr.minionParams()
	if mp == nil || len(mp.Groups) == 0 {
		return Army{}, false
	}
	f.necro.init()
	var a Army
	a.Count = censusTotal(f.minionCensus(snap, mp.Groups))
	known := KnownSkills(p, f.cat, f.keys)
	for _, g := range mp.Groups {
		want := 0
		for _, name := range g.Skills {
			if sp, ok := known[name]; ok {
				want = 1
				if !g.Exclusive {
					want = min(sp.Level, g.Max)
				}
				break
			}
		}
		if cap, ok := f.necro.capSeen[g.Name]; ok && cap < want {
			want = cap
		}
		a.Want += want
	}
	units := f.minionUnits(snap, mp.Groups)
	var cx, cy float64
	for _, u := range units {
		cx, cy = cx+float64(u.X), cy+float64(u.Y)
		lp := 100
		if u.Monster != nil && u.Monster.LifePct > 0 {
			lp = min(u.Monster.LifePct, 100)
		}
		a.Strength += lp
	}
	me := input.Point{X: p.X, Y: p.Y}
	a.Centroid, a.Front = me, me
	if len(units) > 0 {
		n := float64(len(units))
		a.Centroid = input.Point{X: int(math.Round(cx / n)), Y: int(math.Round(cy / n))}
		a.Front = a.Centroid
		if th, ok := f.threatCentroid(snap); ok {
			best := math.MaxFloat64
			for _, u := range units {
				if d := pdist(input.Point{X: u.X, Y: u.Y}, th); d < best {
					best, a.Front = d, input.Point{X: u.X, Y: u.Y}
				}
			}
		}
	}
	a.FrontDist = int(math.Round(pdist(me, a.Front)))
	if !p.InTown() {
		_, a.UpkeepDue = f.planMinion(snap, pr, false, false)
	}
	return a, true
}

// minionUnits lists the standing minions of the groups with their positions (snapshot only: the engine read has none).
func (f *Fighter) minionUnits(snap *d2.Snapshot, groups []MinionGroup) []d2.Unit {
	byMon := map[string]bool{}
	for _, g := range groups {
		for _, name := range g.Skills {
			if s := f.cat.SkillByName(name); s != nil && s.Summon != "" {
				byMon[s.Summon] = true
			}
		}
	}
	var out []d2.Unit
	for _, u := range friendlyUnits(snap, f.cat) {
		key := ""
		if m := f.cat.Monster(u.Class); m != nil {
			key = m.Key
		}
		if byMon[key] || u.Monster.HasState(d2.StateMinion) {
			out = append(out, u)
		}
	}
	return out
}

// threatCentroid is the centroid of the hostiles within stanceThreatRange of the hero.
func (f *Fighter) threatCentroid(snap *d2.Snapshot) (input.Point, bool) {
	p := snap.Player
	me := input.Point{X: p.X, Y: p.Y}
	var cx, cy float64
	n := 0
	for _, h := range Hostiles(snap, f.hostileCtx(p)) {
		if pdist(me, input.Point{X: h.Unit.X, Y: h.Unit.Y}) <= stanceThreatRange {
			cx, cy, n = cx+float64(h.Unit.X), cy+float64(h.Unit.Y), n+1
		}
	}
	if n == 0 {
		return input.Point{}, false
	}
	return input.Point{X: int(math.Round(cx / float64(n))), Y: int(math.Round(cy / float64(n)))}, true
}

// behindPoint is the spot behind subtiles past army on the line from threat through army. With the army on the
// threat itself the hero keeps its own side.
func behindPoint(me, army, threat input.Point, behind int) input.Point {
	from := threat
	if pdist(army, threat) < 0.5 {
		from = approachPoint(me, threat, -1) // a point just past the threat on the far side from the hero
	}
	d := pdist(army, from)
	if d < 0.5 {
		return me
	}
	k := float64(behind) / d
	return input.Point{X: army.X + int(math.Round(float64(army.X-from.X)*k)), Y: army.Y + int(math.Round(float64(army.Y-from.Y)*k))}
}

// stancePlan is the stance's step for this tick: act "" (hold), "behind" or "retreat", and where to.
func (f *Fighter) stancePlan(snap *d2.Snapshot) (act string, to input.Point, why string) {
	p := snap.Player
	if p == nil || p.InTown() {
		return "", input.Point{}, ""
	}
	s := f.profileFor(p).Stance
	if s != nil && s.Kind == StanceInsidePack {
		return f.packPlan(snap, s)
	}
	if s == nil || s.Kind != StanceBehindArmy {
		return "", input.Point{}, ""
	}
	a, ok := f.army(snap)
	if !ok {
		return "", input.Point{}, ""
	}
	me := input.Point{X: p.X, Y: p.Y}
	if s.BossMin > 0 && a.Count < s.BossMin {
		for _, h := range Hostiles(snap, f.hostileCtx(p)) {
			if h.Boss && h.Dist <= stanceThreatRange {
				boss := input.Point{X: h.Unit.X, Y: h.Unit.Y}
				if pt, ok := awayPoint(me, []input.Point{boss}, nil, s.Behind+4, f.valid); ok {
					return "retreat", pt, fmt.Sprintf("boss %s with army %d under %d", h.Name, a.Count, s.BossMin)
				}
				return "", input.Point{}, ""
			}
		}
	}
	if a.Count == 0 {
		return "", input.Point{}, ""
	}
	th, ok := f.threatCentroid(snap)
	if !ok {
		return "", input.Point{}, ""
	}
	spot := behindPoint(me, a.Centroid, th, s.Behind)
	if pdist(me, spot) <= s.slack() {
		return "", input.Point{}, ""
	}
	return "behind", spot, fmt.Sprintf("army of %d at %v, hostiles at %v", a.Count, a.Centroid, th)
}

// stanceState is the shadow memory: the last step logged, so a lasting one is logged once.
type stanceState struct {
	last  string
	diffs int
}

// StanceShadowSteps is the number of stance steps the shadow (combat.deliveryUpkeep 1) would have taken.
func (f *Fighter) StanceShadowSteps() int { return f.stance.diffs }

// stanceTick is the fight-loop hook after the class tick. true = the tick was spent on a step.
func (f *Fighter) stanceTick(ctx context.Context, snap *d2.Snapshot, st *fightState) (bool, error) {
	mode := f.upkeepMode()
	if mode == 0 {
		return false, nil
	}
	act, to, why := f.stancePlan(snap)
	if p := snap.Player; mode == 1 && p != nil && !p.InTown() {
		if s := f.profileFor(p).Stance; s != nil {
			f.upkCmp.note(cmpStance, act == "")
		}
	}
	if act == "" {
		f.stance.last = ""
		return false, nil
	}
	if mode == 1 {
		if key := act + "|" + why; key != f.stance.last {
			f.stance.last = key
			f.stance.diffs++
			f.info("stance shadow", "act", act, "to", to, "why", why)
		}
		return false, nil
	}
	if !f.evadeOK(st) {
		return false, nil
	}
	f.decision(st, "stance", act+": "+why)
	f.info("stance", "act", act, "to", to, "why", why)
	p := snap.Player
	return true, f.evade(ctx, input.Point{X: p.X, Y: p.Y}, to)
}

// ArmyFacts are the arbiter's army facts (arbiter.FactArmy...) from the same census the minion driver and the stance
// read. A hero without an army reports none, so a fact reads 0.
func (f *Fighter) ArmyFacts(snap *d2.Snapshot) map[string]int {
	if f.upkeepMode() == 0 {
		return nil // dark launch: the facts appear with the driver knob, so tick logs stay as they were at 0
	}
	a, ok := f.army(snap)
	if !ok {
		return nil
	}
	b := func(x bool) int {
		if x {
			return 1
		}
		return 0
	}
	return map[string]int{
		arbiter.FactArmy:          a.Count,
		arbiter.FactArmyWant:      a.Want,
		arbiter.FactArmyBelowWant: b(a.BelowWant()),
		arbiter.FactUpkeepDue:     b(a.UpkeepDue),
		arbiter.FactArmyStrength:  a.Strength,
		arbiter.FactArmyFront:     a.FrontDist,
	}
}

// packPlan is the insidePack stance's step: act "" (hold and cast in place), "inside" (walk to a free cell next to
// the centroid) or "leave" (back out of a surround or a low-life fight).
func (f *Fighter) packPlan(snap *d2.Snapshot, s *ArmyStance) (act string, to input.Point, why string) {
	p := snap.Player
	me := input.Point{X: p.X, Y: p.Y}
	var near []input.Point
	var cx, cy float64
	contact := 0
	for _, h := range Hostiles(snap, f.hostileCtx(p)) {
		hp := input.Point{X: h.Unit.X, Y: h.Unit.Y}
		d := pdist(me, hp)
		if d > stanceThreatRange {
			continue
		}
		near = append(near, hp)
		cx, cy = cx+float64(hp.X), cy+float64(hp.Y)
		if d <= stancePackContact {
			contact++
		}
	}
	minPack := s.MinPack
	if minPack == 0 {
		minPack = stancePackMin
	}
	if len(near) < minPack {
		return "", input.Point{}, ""
	}
	if contact >= s.LeaveSurround || (contact > 0 && p.MaxLife > 0 && p.Life*100 < p.MaxLife*s.LeaveLifePct) {
		if pt, ok := awayPoint(me, near, nil, s.StandOff+4, f.valid); ok {
			return "leave", pt, fmt.Sprintf("%d hostiles in contact, life %d of %d", contact, p.Life, p.MaxLife)
		}
		return "", input.Point{}, ""
	}
	n := float64(len(near))
	th := input.Point{X: int(math.Round(cx / n)), Y: int(math.Round(cy / n))}
	if pdist(me, th) <= float64(s.StandOff)+stanceSlackDefault {
		return "", input.Point{}, ""
	}
	spot := approachPoint(me, th, float64(s.StandOff))
	if !f.valid(spot) {
		return "", input.Point{}, ""
	}
	return "inside", spot, fmt.Sprintf("pack of %d at %v", len(near), th)
}

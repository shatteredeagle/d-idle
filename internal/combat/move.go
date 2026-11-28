package combat

import (
	"fmt"
	"math"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/nav"
)

// Move driver (cards dlv-move-tele and dlv-move-melee; docs/design/fable-review-2026-11-21/combat-engine-d-cards.md).
// A move skill is a profile row (Profile.Moves): Teleport hops, Charge and Leap Attack close a gap. The driver is the
// only caller of the cast; navigation keeps the route and asks the driver for what a hop is (HopInfo: key, range, cost,
// refusal), so no range or mana constant lives in nav and a character without the skill, a demoted skill (expect.go) or
// too little mana is a refusal with a reason, not a failed cast. A row names its uses:
//   - travel: navigation may hop along its route (HopInfo).
//   - escape: out of a surround, in the fight loop (moveTick): EscapeSurround hostiles within stancePackContact and life
//     under EscapeLifePct percent hop to the reachable cell inside the row's range that is farthest from the pack.
//   - gap: close on a target beyond MinGap and inside the range (gapClose); Through aims past the target (Leap Attack).
//   - whirl: a held spin across the pack (whirl.go, card dlv-whirl): legs between pack positions, each leg one press slice.
//
// Knob combat.deliveryUpkeep (the delivery driver knob): 0 nothing here acts (navigation hops through its own key as
// before), 1 shadow (the hop or gap cast the driver would make is counted and logged, no cast), 2 the driver casts. The
// cast-first watchdog and the pinned-evade memory bound a repeated hop.

// Move uses.
const (
	MoveTravel = "travel"
	MoveEscape = "escape"
	MoveGap    = "gap"
	MoveWhirl  = "whirl"
)

// MoveRule is one move row of a profile.
type MoveRule struct {
	Name string   `json:"name"`
	Uses []string `json:"uses"`
	// Range is the farthest landing in subtiles one use covers; MinHop the nearest worth casting for.
	Range  int `json:"range"`
	MinHop int `json:"minHop,omitempty"`
	// Escape: hostiles in contact and life percent under which the hop is taken.
	EscapeSurround int `json:"escapeSurround,omitempty"`
	EscapeLifePct  int `json:"escapeLifePct,omitempty"`
	// Gap: the target must be farther than MinGap (and within Range); Through aims MoveThrough subtiles past it.
	MinGap  int `json:"minGap,omitempty"`
	Through int `json:"through,omitempty"`
	// Whirl (whirl.go): Range is the longest leg, MinHop the shortest; a leg starts only against WhirlPack hostiles within
	// WhirlRadius subtiles of its aim cell and while life is at or above WhirlAbortLifePct (the life floor).
	WhirlPack         int    `json:"whirlPack,omitempty"`
	WhirlRadius       int    `json:"whirlRadius,omitempty"`
	WhirlAbortLifePct int    `json:"whirlAbortLifePct,omitempty"`
	Source            string `json:"source"` // V, S or U first: where Range and the triggers come from
}

func (r *MoveRule) uses(u string) bool {
	for _, x := range r.Uses {
		if x == u {
			return true
		}
	}
	return false
}

// validateMoves checks the move rows at load time: an error names the profile and the row.
func validateMoves(ps []Profile) error {
	for i := range ps {
		p := &ps[i]
		seen := map[string]bool{}
		for _, r := range p.Moves {
			switch {
			case r.Name == "" || seen[r.Name]:
				return fmt.Errorf("profile %q: move row %q missing or twice", p.ProfileID(), r.Name)
			case len(r.Uses) == 0:
				return fmt.Errorf("profile %q: move row %q has no use", p.ProfileID(), r.Name)
			case r.Range < 2 || r.MinHop < 0 || r.MinHop >= r.Range || r.MinGap < 0 || r.Through < 0:
				return fmt.Errorf("profile %q: move row %q: range >= 2, minHop below range, minGap and through >= 0", p.ProfileID(), r.Name)
			case !validSource(r.Source):
				return fmt.Errorf("profile %q: move row %q needs a source tag (V, S or U)", p.ProfileID(), r.Name)
			}
			for _, u := range r.Uses {
				if u != MoveTravel && u != MoveEscape && u != MoveGap && u != MoveWhirl {
					return fmt.Errorf("profile %q: move row %q: use %q unknown", p.ProfileID(), r.Name, u)
				}
			}
			if r.uses(MoveEscape) && (r.EscapeSurround < 1 || r.EscapeLifePct < 1 || r.EscapeLifePct > 100) {
				return fmt.Errorf("profile %q: move row %q: escape needs escapeSurround >= 1 and escapeLifePct 1..100", p.ProfileID(), r.Name)
			}
			if r.uses(MoveGap) && r.MinGap < 1 {
				return fmt.Errorf("profile %q: move row %q: gap needs minGap >= 1", p.ProfileID(), r.Name)
			}
			if r.uses(MoveWhirl) && (r.WhirlPack < 2 || r.WhirlRadius < 1 || r.WhirlAbortLifePct < 1 || r.WhirlAbortLifePct > 100) {
				return fmt.Errorf("profile %q: move row %q: whirl needs whirlPack >= 2, whirlRadius >= 1 and whirlAbortLifePct 1..100", p.ProfileID(), r.Name)
			}
			seen[r.Name] = true
		}
	}
	return nil
}

// Move refusal reasons (HopInfo.Refused).
const (
	MoveNoRow    = "no move row"
	MoveUnknown  = "skill not known or not bound"
	MoveDemoted  = "demoted by the expectation ledger"
	MoveNoMana   = "mana"
	MoveNoSelect = "left-hand skill on a backend that cannot cast it by id"
	MoveNotUsing = "row has no such use"
)

// HopInfo is what navigation asks for: the cast key, the range and mana cost of one hop, and why there is none.
type HopInfo struct {
	Name    string
	Key     string
	Range   float64
	MinHop  float64
	Cost    int
	Refused string
}

// HopInfo reports the travel row of the character: the first row with the travel use. Refused is set (and Key empty)
// when there is none, the skill is unknown, unbound, demoted or unaffordable.
func (f *Fighter) HopInfo(p *d2.Player) HopInfo {
	if p == nil {
		return HopInfo{Refused: MoveNoRow}
	}
	for i := range f.profileFor(p).Moves {
		r := &f.profileFor(p).Moves[i]
		if !r.uses(MoveTravel) {
			continue
		}
		hi := HopInfo{Name: r.Name, Range: float64(r.Range), MinHop: float64(r.MinHop)}
		sp, why := f.moveSpec(p, r)
		hi.Cost = sp.Cost
		if why != "" {
			hi.Refused = why
			return hi
		}
		hi.Key = sp.Key
		return hi
	}
	return HopInfo{Refused: MoveNoRow}
}

// moveSpec is the castable spec of a move row, or the refusal why not.
func (f *Fighter) moveSpec(p *d2.Player, r *MoveRule) (SkillSpec, string) {
	sp, ok := KnownSkills(p, f.cat, f.keys)[r.Name]
	switch {
	case !ok:
		return sp, MoveUnknown
	case sp.Hand == HandLeft && !f.swings():
		return sp, MoveNoSelect // a hotkey selects only the right-hand skill
	case sp.Hand != HandLeft && sp.Key == "" && !f.castsByID():
		return sp, MoveUnknown
	case f.expectDemoted(r.Name):
		return sp, MoveDemoted
	case p.MaxMana > 0 && p.Mana < sp.Cost:
		return sp, MoveNoMana
	}
	return sp, ""
}

// moveRow is the first row of the character's profile with the use, and its castable spec.
func (f *Fighter) moveRow(p *d2.Player, use string) (*MoveRule, SkillSpec, string) {
	pr := f.profileFor(p)
	for i := range pr.Moves {
		if r := &pr.Moves[i]; r.uses(use) {
			sp, why := f.moveSpec(p, r)
			return r, sp, why
		}
	}
	return nil, SkillSpec{}, MoveNoRow
}

// escapeCell is the reachable cell inside [MinHop, Range] subtiles that is farthest from the nearest hostile, at least
// moveGain subtiles farther than the hero stands now; ok false when none.
func escapeCell(me input.Point, hostiles []input.Point, r *MoveRule, valid func(input.Point) bool) (input.Point, bool) {
	cur := minDist(me, hostiles)
	best, bestD, found := me, cur+moveGain, false
	for dy := -r.Range; dy <= r.Range; dy++ {
		for dx := -r.Range; dx <= r.Range; dx++ {
			l := math.Hypot(float64(dx), float64(dy))
			if l > float64(r.Range) || l < float64(max(r.MinHop, 2)) {
				continue
			}
			c := input.Point{X: me.X + dx, Y: me.Y + dy}
			if d := minDist(c, hostiles); d > bestD && valid(c) {
				best, bestD, found = c, d, true
			}
		}
	}
	return best, found
}

// moveGain is how many subtiles farther from the pack an escape must land (U).
const moveGain = 3

// escapePlan is the escape hop for this tick: the row, its spec, the landing cell and why; ok false holds.
func (f *Fighter) escapePlan(snap *d2.Snapshot) (r *MoveRule, sp SkillSpec, to input.Point, why string, ok bool) {
	p := snap.Player
	if p == nil || p.InTown() {
		return
	}
	var why0 string
	if r, sp, why0 = f.moveRow(p, MoveEscape); r == nil || why0 != "" {
		if r != nil && f.learned(p, r.Name) {
			f.hold(holdEscape, why0)
		}
		return nil, sp, to, why0, false
	}
	me := input.Point{X: p.X, Y: p.Y}
	var near []input.Point
	contact := 0
	for _, h := range Hostiles(snap, f.hostileCtx(p)) {
		hp := input.Point{X: h.Unit.X, Y: h.Unit.Y}
		if h.Dist <= stanceThreatRange {
			near = append(near, hp)
		}
		if h.Dist <= stancePackContact {
			contact++
		}
	}
	if contact < r.EscapeSurround || pct(p.Life, p.MaxLife) >= r.EscapeLifePct {
		f.hold(holdEscape, holdNoTrigger)
		return nil, sp, to, "", false
	}
	to, found := escapeCell(me, near, r, f.valid)
	if !found {
		f.hold(holdEscape, holdNoCell)
		return nil, sp, to, "", false
	}
	return r, sp, to, fmt.Sprintf("%d in contact, life %d%%, %s to %v", contact, pct(p.Life, p.MaxLife), r.Name, to), true
}

// moveState is the shadow memory and counters of the move driver.
type moveState struct {
	last           string
	shadow, hops   int
	gaps, refusals int
	legs, aborts   int
	legEnd         *input.Point // where the last whirl leg ended: the next leg prefers another pack
}

// MoveShadowCasts, MoveHops, MoveGaps and MoveRefusals are the telemetry of the move driver.
func (f *Fighter) MoveShadowCasts() int { return f.moves.shadow }
func (f *Fighter) MoveHops() int        { return f.moves.hops }
func (f *Fighter) MoveGaps() int        { return f.moves.gaps }
func (f *Fighter) MoveRefusals() int    { return f.moves.refusals }

// moveTick is the fight-loop hook after the stance: an escape hop out of a surround. true = the tick was spent on it.
func (f *Fighter) moveTick(snap *d2.Snapshot, st *fightState) (bool, error) {
	mode := f.upkeepMode()
	if mode == 0 || !f.mayEvade() {
		return false, nil
	}
	r, sp, to, why, ok := f.escapePlan(snap)
	if !ok {
		f.moves.last = ""
		return false, nil
	}
	if mode == 1 {
		if key := r.Name + "|" + why; key != f.moves.last {
			f.moves.last = key
			f.moves.shadow++
			f.info("move shadow", "use", MoveEscape, "why", why)
		}
		return false, nil
	}
	f.decision(st, "move", MoveEscape+": "+why)
	f.info("move", "use", MoveEscape, "skill", r.Name, "to", to, "why", why)
	if err := f.cast(snap, sp, to); err != nil {
		f.moves.refusals++
		return true, err
	}
	f.moves.hops++
	return true, nil
}

// gapClose casts the gap row at a target beyond MinGap and inside the row's range, with a clear line. aimed false: no
// cast (mode 0, no row, refused or out of band); the caller walks as before.
//
// The landing is a movement like a walk: a point in a cave mouth's trigger area is refused as f.move refuses it
// (fable review 3 finding 5; escape and whirl landings pass f.valid, which asks the same gate). A gap cast and a whirl
// leg are attacks and count as casts of the fight, so the cast watchdog does not force casts into a fight made of them.
func (f *Fighter) gapClose(snap *d2.Snapshot, tg Target, st *fightState) (aimed bool, err error) {
	mode := f.upkeepMode()
	p := snap.Player
	if mode == 0 || p == nil || p.InTown() {
		return false, nil
	}
	r, sp, why := f.moveRow(p, MoveGap)
	if r == nil {
		return false, nil
	}
	if why != "" {
		f.moves.refusals++
		return false, nil
	}
	if tg.Dist <= float64(r.MinGap) || tg.Dist > float64(r.Range) {
		return false, nil
	}
	me := input.Point{X: p.X, Y: p.Y}
	at := input.Point{X: tg.Unit.X, Y: tg.Unit.Y}
	if !f.los(me, at) {
		return false, nil
	}
	if r.Through > 0 {
		at = approachPoint(me, at, -float64(r.Through)) // past the target on the line from the hero
	}
	if f.nearMouth(at) {
		return false, nil
	}
	if mode == 1 {
		f.moves.shadow++
		f.info("move shadow", "use", MoveGap, "skill", r.Name, "target", tg.Name, "dist", int(tg.Dist))
		return false, nil
	}
	f.info("move", "use", MoveGap, "skill", r.Name, "target", tg.Name, "dist", int(tg.Dist))
	if err := f.cast(snap, sp, at); err != nil {
		f.moves.refusals++
		return true, err
	}
	f.moves.gaps++
	if st != nil {
		st.noteCast(f.now())
	}
	return true, nil
}

// NavHop is HopInfo in the navigator's own type: the bot sets nav.Navigator.TeleportHopFn to a closure over it.
func (f *Fighter) NavHop(p *d2.Player) nav.TeleportHop {
	h := f.HopInfo(p)
	return nav.TeleportHop{Key: h.Key, Range: h.Range, MinHop: h.MinHop, Cost: h.Cost, Refused: h.Refused}
}

func (f *Fighter) swings() bool { _, ok := f.e.In.(swinger); return ok }

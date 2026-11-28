package combat

import (
	"fmt"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// Whirlwind driver (card dlv-whirl; docs/design/fable-review-2026-11-21/combat-engine-d-cards.md). A move row with the
// whirl use (move.go) makes a held spin a path across the pack: each fight tick is one leg, aimed at the densest pack
// within the row's range and past it (Through), so the spin cuts through the monsters instead of ending on the first. The
// leg is one press slice (press.go: the cast carries its release, so no spin outlasts the slice); the fight loop between
// legs is where the potion and the evade rules run, so there is no drink during a spin. The life floor
// (WhirlAbortLifePct) is checked before each leg: under it no leg starts and the fight falls through to its normal slot
// (potion, retreat, the other attacks). Consecutive legs prefer a pack other than the one the last leg ended in, so the
// hero moves between pack positions.
//
// Knob combat.deliveryUpkeep (the delivery driver knob): 0 nothing here acts, 1 shadow (the leg is counted and logged,
// no cast), 2 the driver casts. U: the spin radius and the real damage rate wait for fidelity A6 (radius) and A1
// (attack-sequence rate); the numbers in the rows are planning values.

// whirlLeg picks the end of the next leg: the hostile with the biggest pack (hostiles within r.WhirlRadius of it, at
// least r.WhirlPack), inside r.Range, reachable and in line of sight; a pack near the last leg's end counts last. The
// end is r.Through subtiles past that hostile and at least r.MinHop away from the hero. ok false: no pack worth a leg.
func whirlLeg(me input.Point, hostiles []input.Point, r *MoveRule, last *input.Point, valid, los func(input.Point) bool) (to input.Point, pack int, ok bool) {
	best := -1 << 30
	for _, h := range hostiles {
		d := pdist(me, h)
		if d > float64(r.Range) {
			continue
		}
		n := 0
		for _, o := range hostiles {
			if pdist(h, o) <= float64(r.WhirlRadius) {
				n++
			}
		}
		if n < r.WhirlPack || !los(h) {
			continue
		}
		score := n*100 - int(d)
		if last != nil && pdist(*last, h) <= float64(r.WhirlRadius) {
			score -= 1000
		}
		if score <= best {
			continue
		}
		reach := max(d+float64(r.Through), float64(max(r.MinHop, 1)))
		reach = min(reach, float64(r.Range))
		end := approachPoint(me, h, -(reach - d))
		if !valid(end) {
			if end = h; !valid(end) {
				continue
			}
		}
		best, to, pack, ok = score, end, n, true
	}
	return
}

// whirlPlan is the leg for this tick: the row, its spec, the end, why; ok false holds.
func (f *Fighter) whirlPlan(snap *d2.Snapshot) (r *MoveRule, sp SkillSpec, to input.Point, why string, ok bool) {
	p := snap.Player
	if p == nil || p.InTown() {
		return
	}
	var why0 string
	if r, sp, why0 = f.moveRow(p, MoveWhirl); r == nil || why0 != "" {
		if r != nil && f.learned(p, r.Name) {
			f.hold(holdWhirl, why0)
		}
		return nil, sp, to, why0, false
	}
	if pct(p.Life, p.MaxLife) < r.WhirlAbortLifePct {
		f.moves.aborts++
		f.hold(holdWhirl, holdLifeFloor)
		return nil, sp, to, "", false
	}
	if !f.pressMayStart(p, sp) {
		f.hold(holdWhirl, holdCommit)
		return nil, sp, to, "", false
	}
	me := input.Point{X: p.X, Y: p.Y}
	var pts []input.Point
	for _, h := range Hostiles(snap, f.hostileCtx(p)) {
		pts = append(pts, input.Point{X: h.Unit.X, Y: h.Unit.Y})
	}
	to, pack, found := whirlLeg(me, pts, r, f.moves.legEnd, f.valid, func(h input.Point) bool { return f.los(me, h) })
	if !found {
		f.hold(holdWhirl, holdNoPack)
		return nil, sp, to, "", false
	}
	return r, sp, to, fmt.Sprintf("%d in the pack, life %d%%, %s leg to %v", pack, pct(p.Life, p.MaxLife), r.Name, to), true
}

// whirlTick is the fight-loop hook after moveTick: one whirl leg. true = the tick was spent on it.
func (f *Fighter) whirlTick(snap *d2.Snapshot, st *fightState) (bool, error) {
	mode := f.upkeepMode()
	if mode == 0 {
		return false, nil
	}
	r, sp, to, why, ok := f.whirlPlan(snap)
	if !ok {
		return false, nil
	}
	if mode == 1 {
		f.moves.shadow++
		f.info("move shadow", "use", MoveWhirl, "why", why)
		return false, nil
	}
	f.decision(st, "move", MoveWhirl+": "+why)
	f.info("move", "use", MoveWhirl, "skill", r.Name, "to", to, "why", why)
	f.moves.legEnd = &to
	if err := f.cast(snap, sp, to); err != nil {
		f.moves.refusals++
		return true, err
	}
	f.moves.legs++
	st.noteCast(f.now())
	return true, nil
}

// WhirlLegs and WhirlAborts are the telemetry of the whirl driver: legs cast, legs held back by the life floor.
func (f *Fighter) WhirlLegs() int   { return f.moves.legs }
func (f *Fighter) WhirlAborts() int { return f.moves.aborts }

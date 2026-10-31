package combat

import (
	"context"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// Crowd rule (SME decision EXP, docs/combat.md "Crowd rule"): Soaky died three times at one spot with 22 hostiles near,
// life 50 percent to 0 in about 3 s. The rule is ADDITIVE: it only looks at hostiles within crowdRadius, so the normal
// 55 percent drink and 25 percent chicken rules still decide ordinary fights alone. Thresholds are knobs
// (combat.crowdN, crowdPortalPct, crowdDrinkPct, crowdAbandonN); the radii below are U (judgment, not measured).
const (
	crowdRadius      = 8  // subtiles: a hostile this close counts to the crowd
	crowdAdjacent    = 3  // subtiles: a hostile this close is in melee with us
	crowdAdjacentMax = 3  // this many adjacent monsters or more: a 2 s portal cast fails (U)
	crowdRetreatDist = 20 // subtiles an abandon retreat goes before the portal
	crowdMaxSteps    = 8  // retreat steps of one abandon
)

// Crowd counts the hostiles within crowdRadius of the player and those adjacent.
func Crowd(p *d2.Player, hs []Target) (crowd, adjacent int) {
	for _, h := range hs {
		if h.Dist <= crowdRadius {
			crowd++
		}
		if h.Dist <= crowdAdjacent {
			adjacent++
		}
	}
	return crowd, adjacent
}

// crowdDecide is the crowd part of Decide: ok is false when the rule has nothing to say and the normal rules go on.
func crowdDecide(in SafetyInput, lp int, heal, rejuv bool) (SafetyAction, bool) {
	t := in.Tune
	if t.CrowdN <= 0 || in.Crowd < t.CrowdN {
		return SafetyNone, false
	}
	surrounded := in.Adjacent >= crowdAdjacentMax
	drink := func() (SafetyAction, bool) {
		switch {
		case heal && in.HealReady:
			return DrinkLife, true
		case rejuv && in.RejuvReady:
			return DrinkRejuv, true
		}
		return SafetyNone, false
	}
	if t.CrowdAbandonN > 0 && in.Crowd >= t.CrowdAbandonN && (in.Walking || lp <= t.CrowdPortalPct) {
		if surrounded && lp < t.CrowdDrinkPct {
			if a, ok := drink(); ok {
				return a, true // cannot walk away from three attackers at this life: drink first
			}
		}
		if in.HasPortal {
			return CrowdAbandon, true
		}
		if in.CanRetreat {
			return CrowdRetreat, true
		}
	}
	if lp <= t.CrowdPortalPct && in.HasPortal && !surrounded {
		return TownPortal, true
	}
	if lp < t.CrowdDrinkPct {
		if a, ok := drink(); ok {
			return a, true
		}
	}
	if lp <= t.CrowdPortalPct && surrounded && in.CanRetreat {
		return CrowdRetreat, true
	}
	return SafetyNone, false
}

// SetWalking tells the fighter whether the bot is on a travel or quest walk (the crowd-abandon rule is for walks; in a
// fight the same crowd is fought). Safe from any goroutine.
func (f *Fighter) SetWalking(on bool) { f.walking.Store(on) }

// crowdPoints lists the positions of the hostiles within crowdRadius+4 (the crowd to get away from).
func crowdPoints(hs []Target) []input.Point {
	var pts []input.Point
	for _, h := range hs {
		if h.Dist <= crowdRadius+4 {
			pts = append(pts, input.Point{X: h.Unit.X, Y: h.Unit.Y})
		}
	}
	return pts
}

// canRetreat reports whether a step away from the crowd exists (so Decide never asks for a retreat that cannot happen).
func (f *Fighter) canRetreat(snap *d2.Snapshot, hs []Target) bool {
	if f.mv == nil {
		return false
	}
	p := snap.Player
	_, ok := awayPoint(input.Point{X: p.X, Y: p.Y}, crowdPoints(hs), hostileMissiles(snap), f.tune().KiteRetreat, f.valid)
	return ok
}

// retreatStep walks one step away from the crowd; it reports whether a step was taken.
func (f *Fighter) retreatStep(ctx context.Context, snap *d2.Snapshot, hs []Target) (bool, error) {
	p := snap.Player
	me := input.Point{X: p.X, Y: p.Y}
	pt, ok := awayPoint(me, crowdPoints(hs), hostileMissiles(snap), f.tune().KiteRetreat, f.valid)
	if !ok {
		return false, nil
	}
	f.info("crowd: stepping away", "crowd", len(crowdPoints(hs)), "life", pct(p.Life, p.MaxLife), "to", pt)
	return true, f.evade(ctx, me, pt)
}

// crowdAct carries out CrowdRetreat (one step) or CrowdAbandon (retreat 20+ subtiles, then a portal).
func (f *Fighter) crowdAct(ctx context.Context, a SafetyAction) error {
	snap, err := f.read(ctx)
	if err != nil {
		return err
	}
	hs := Hostiles(snap, f.hostileCtx(snap.Player))
	if a == CrowdRetreat {
		_, err := f.retreatStep(ctx, snap, hs)
		return err
	}
	start := input.Point{X: snap.Player.X, Y: snap.Player.Y}
	f.info("crowd: abandoning the walk", "crowd", len(crowdPoints(hs)), "life", pct(snap.Player.Life, snap.Player.MaxLife))
	for i := 0; i < crowdMaxSteps && f.mv != nil; i++ {
		me := input.Point{X: snap.Player.X, Y: snap.Player.Y}
		if pdist(start, me) >= crowdRetreatDist {
			break
		}
		moved, err := f.retreatStep(ctx, snap, hs)
		if err != nil || !moved {
			break
		}
		if snap, err = f.read(ctx); err != nil {
			return err
		}
		hs = Hostiles(snap, f.hostileCtx(snap.Player))
	}
	if f.Portal != nil {
		return f.Portal(ctx)
	}
	return f.openPortal(ctx)
}

// PortalCast is the last Town Portal cast the fighter made: when it started, how long it took and whether it opened.
type PortalCast struct {
	Started bool
	Start   time.Time
	Dur     time.Duration
	OK      bool
}

// LastPortalCast reports the last portal cast (Started false when there was none), for the death autopsy.
func (f *Fighter) LastPortalCast() PortalCast {
	f.diagMu.Lock()
	defer f.diagMu.Unlock()
	return f.portalCast
}

type castRec struct {
	at    time.Time
	skill int
}

const castRingMax = 64

func (f *Fighter) noteCast(skill int, at time.Time) {
	f.diagMu.Lock()
	f.casts = append(f.casts, castRec{at, skill})
	if n := len(f.casts) - castRingMax; n > 0 {
		f.casts = append([]castRec(nil), f.casts[n:]...)
	}
	f.diagMu.Unlock()
}

// RecentCasts counts the skill casts in the last window, in total and per skill id, for the death autopsy.
func (f *Fighter) RecentCasts(window time.Duration) (int, map[int]int) {
	cut := f.now().Add(-window)
	by := map[int]int{}
	n := 0
	f.diagMu.Lock()
	defer f.diagMu.Unlock()
	for _, c := range f.casts {
		if !c.at.Before(cut) {
			by[c.skill]++
			n++
		}
	}
	return n, by
}

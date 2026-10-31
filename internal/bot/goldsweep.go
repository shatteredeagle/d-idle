package bot

import (
	"context"
	"fmt"
	"math"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// Gold sweep (docs/deathspiral.md "Gold on death"). SECONDARY source (d2decomped notes, docs/sme/play-economy.md): on
// death the purse is cut by min(clvl,20) percent and the rest drops as a ground pile at the death spot, the corpse holds
// only worn items, the stash is safe. UNVERIFIED live: our logs read gold 0 after every death.
const (
	goldSweepMin    = 100 // expected pile below which the trip is not worth it
	goldSweepRadius = 40  // subtiles around the death position in which piles are taken
	goldSweepPiles  = 4
)

// expectedPile is the gold a death left on the ground, from the carried gold known at the last living snapshot.
func expectedPile(d DeathRecord) int {
	if d.Autopsy == nil || d.Autopsy.GoldCarried <= 0 {
		return 0
	}
	pct := min(max(d.Level, 0), 20)
	return d.Autopsy.GoldCarried * (100 - pct) / 100
}

// corpseWorth is what a fresh death left behind: the worn items plus the expected gold pile.
func corpseWorth(d DeathRecord) int { return d.CorpseValue + expectedPile(d) }

// sweepDecision says whether to walk back to the death spot for the gold pile when no corpse run is under way. It never
// goes back to a banned area (spiral or repeat site), after a second death on the way, or into an unfavorable one.
func (r *Runner) sweepDecision(rec *recovery, snap *d2.Snapshot) (bool, string) {
	t := r.e.Tune.Bot
	d := rec.rec
	pile := expectedPile(d)
	switch {
	case r.cfg.Hardcore:
		return false, "hardcore"
	case rec.tries >= 1<<20:
		return false, "second death on the way"
	case d.Area == 0 || r.d.Plan.IsTown(d.Area):
		return false, "death in town"
	case pile < goldSweepMin:
		return false, fmt.Sprintf("expected pile %d < %d", pile, goldSweepMin)
	case d.Autopsy != nil && d.Autopsy.HostilesNear >= max(r.e.Tune.Combat.CrowdN, 1):
		// the pack that killed us still stands at the spot, and the town portal leads right into it (Fresha i5
		// 2026-10-30: back through the portal into 25 Carvers for a 644 gold pile, dead in 6 s, docs/potion-bug.md)
		return false, fmt.Sprintf("%d hostiles were at the death spot", d.Autopsy.HostilesNear)
	case d.KillerBoss:
		return false, "killed by a boss"
	case r.banned(d.Area):
		return false, "area banned"
	case snap == nil || snap.Player == nil:
		return false, "no snapshot"
	}
	if m := r.areaMLvl(d.Area, snap.Player.Difficulty); m > 0 && m-snap.Player.Level >= 3 {
		return false, fmt.Sprintf("monster level %d at clvl %d", m, snap.Player.Level)
	}
	r.obMu.Lock()
	recent := r.dlog.since(r.e.Clock.Now().Add(-time.Duration(t.SpiralMinutes)*time.Minute), d.Area)
	r.obMu.Unlock()
	if recent >= t.SpiralDeaths-1 {
		return false, fmt.Sprintf("%d recent deaths in this area", recent)
	}
	return true, fmt.Sprintf("expected pile %d", pile)
}

// goldSweep walks onto the gold piles near the death position (walking onto a pile takes it, VERIFIED LIVE for the
// corpse pile, docs/combat.md "Death"), then lets the looter take what is left. there says the character already stands
// in the death area (a corpse run just ended).
func (r *Runner) goldSweep(ctx context.Context, area int, at input.Point, there bool) error {
	if !there {
		if err := r.gotoDeathArea(ctx, area); err != nil {
			return err
		}
	}
	tried := map[input.Point]bool{}
	for i := 0; i < goldSweepPiles; i++ {
		s := r.poll()
		if s == nil || s.Player == nil {
			break
		}
		var near []d2.Unit
		for _, u := range s.Ground {
			p := input.Point{X: u.X, Y: u.Y}
			if !tried[p] && math.Hypot(float64(u.X-at.X), float64(u.Y-at.Y)) <= goldSweepRadius {
				near = append(near, u)
			}
		}
		pile, ok := combat.GoldPile(near, at, r.isGoldClass)
		if !ok {
			break
		}
		tried[pile] = true
		if err := r.d.Mover.GoTo(ctx, pile); err != nil {
			return fmt.Errorf("walk to gold: %w", err)
		}
	}
	if r.d.Loot != nil {
		if _, err := r.d.Loot.PickUp(ctx, PickOpts{Radius: 30}); err != nil {
			return fmt.Errorf("pickup: %w", err)
		}
	}
	return nil
}

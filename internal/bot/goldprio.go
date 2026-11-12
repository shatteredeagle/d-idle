package bot

import (
	"context"
	"errors"
	"time"

	"d2lod-bot/internal/d2"
)

// Gold priority (docs/goldprio.md, knob loot.goldPriority): below survival, above the next fight target, the next roam
// leg and leaving an area, the hero takes the gold piles it knows about.

// goldLooter is the optional Looter capability behind it (loot.Looter through the adapter).
type goldLooter interface {
	GoldNear(s *d2.Snapshot, dist int) bool
	GoldLedger() (seen, picked, gold, abandoned int)
}

const goldIdleFor = 20 * time.Second // after a gold stop that picked nothing, the roam does not stop for gold again this long

func (r *Runner) goldLooter() (goldLooter, bool) {
	if r.d.Loot == nil || r.e.Tune.Loot.GoldPriority == 0 {
		return nil, false
	}
	gl, ok := r.d.Loot.(goldLooter)
	return gl, ok
}

// goldGrab is the Clear interlude: a pile within GoldGrabDist is taken in the middle of a fight when the hero is
// healthy, no hostile is closer than GoldSafeDist and the pack is not more dangerous than GoldMaxDanger. Survival
// (potions, chicken, retreat) runs before the interlude in the fight loop, so it always wins.
func (r *Runner) goldGrab(ctx context.Context, s *d2.Snapshot, nearest float64, danger int) bool {
	gl, ok := r.goldLooter()
	t := &r.e.Tune.Loot
	if !ok || s == nil || s.Player == nil || ctx.Err() != nil {
		return false
	}
	now := r.e.Clock.Now()
	switch {
	case s.Player.LifePct() < t.GoldMinLifePct, nearest < float64(t.GoldSafeDist), danger > t.GoldMaxDanger:
		return false
	case now.Before(r.loop.goldIdleUntil):
		return false
	case !gl.GoldNear(s, t.GoldGrabDist):
		return false
	}
	rep, err := r.d.Loot.PickUp(ctx, PickOpts{Radius: t.GoldGrabDist, GoldOnly: true, Max: 1, Abort: func() bool {
		cur := r.latest()
		return cur != nil && r.hostileNear(cur, t.GoldSafeDist) > 0
	}})
	if err != nil {
		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			r.e.Log.Warn("gold grab failed", "err", err)
		}
		return false
	}
	if rep.Picked == 0 {
		// nothing taken (aborted by a hostile, or a pile the click could not reach): do not spin on it
		r.loop.goldIdleUntil = now.Add(2 * time.Second)
	} else {
		r.e.Log.Info("gold grab: took a pile mid-fight", "gold", rep.Gold)
	}
	return true
}

// goldInReach tells the roam to stop: a pile within radius that the pickit wants and the hostile-free sweep can fetch.
func (r *Runner) goldInReach(radius int) bool {
	gl, ok := r.goldLooter()
	if !ok || r.e.Clock.Now().Before(r.loop.goldIdleUntil) {
		return false
	}
	return gl.GoldNear(r.latest(), radius)
}

// goldBeforeLeaving takes what is in reach and walks back for remembered piles before the sweep gives an area up
// (empty-area leave, explored enough, explore gave up, cleared). hostile is the sweep's own test.
func (r *Runner) goldBeforeLeaving(ctx context.Context, area, radius int, hostile func() bool) {
	if _, ok := r.goldLooter(); !ok || r.d.Loot == nil || ctx.Err() != nil || hostile() {
		return
	}
	if _, err := r.d.Loot.PickUp(ctx, PickOpts{Radius: radius, Abort: hostile}); err != nil {
		return
	}
	r.walkBackLoot(ctx, area, hostile)
	if gl, ok := r.goldLooter(); ok {
		if seen, picked, gold, left := gl.GoldLedger(); seen > 0 {
			r.e.Log.Info("gold ledger", "area", r.areaName(area), "seen", seen, "picked", picked, "gold", gold, "abandoned", left)
		}
	}
}

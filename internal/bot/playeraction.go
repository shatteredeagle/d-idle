package bot

import (
	"context"
	"time"

	"d2lod-bot/internal/d2"
)

// playerActionGap is the shortest time between two attempts of the player's queued item actions after one was
// postponed (a fight started under the click path).
const playerActionGap = 10 * time.Second

// playerActionStep performs the item actions the player queued on the watch page (docs/inventory-actions.md): equip a
// backpack item, trash a backpack item. The OPEN OPPORTUNITY is the same predicate as the points step: in town, or in
// the field while no hostile is within the clear radius (hostileNear; no fight is on). The click path checks again
// (leveler.busy) and postpones the request when a monster is close. It never undoes anything: a worn item stays pinned,
// a trashed one stays dropped. It reports whether it acted.
func (r *Runner) playerActionStep(ctx context.Context, snap *d2.Snapshot, c control) bool {
	pg := r.d.PlayerGear
	if pg == nil || snap == nil || snap.Player == nil || pg.Pending() == 0 {
		return false
	}
	l := &r.loop
	if l.trip != nil {
		return false // a town trip in progress finishes first
	}
	now := r.e.Clock.Now()
	if !l.playerActAt.IsZero() && now.Sub(l.playerActAt) < playerActionGap {
		return false
	}
	if r.hostileNear(snap, r.e.Tune.Bot.ClearRadius) > 0 {
		return false
	}
	l.playerActAt = now
	r.setWhy(WhyPlayerItems)
	r.do(ctx, c, StateInTown, "Item actions from the watch page", sec(r.cfg.TownTimeoutSec), "player item actions", func(ctx context.Context) error {
		for _, line := range pg.RunPending(ctx) {
			r.e.Log.Info("player " + line)
		}
		return ctx.Err()
	})
	return true
}

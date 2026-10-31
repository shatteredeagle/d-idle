package bot

import (
	"context"
	"errors"
	"time"

	"d2lod-bot/internal/d2"
)

// YieldDeps wires a bounded combat clear for nav.Navigator.Yield. The bot package does not import nav or combat.
type YieldDeps struct {
	Snapshot func() (*d2.Snapshot, error)
	// Friendly reports minions, mercenaries, pets and townspeople (combat.IsFriendly); they are never fought.
	Friendly func(*d2.Snapshot, d2.Unit) bool
	// Clear kills hostiles within radius of the player, giving up after timeout.
	Clear   func(ctx context.Context, radius int, timeout time.Duration) error
	Radius  int           // 0 = 12 subtiles
	Timeout time.Duration // 0 = 20s
}

// PackYield returns the nav Yield hook: a short clear when a walk is stalled by a pack. It does nothing in town or
// when only friendlies are near. Combat results are not errors for the walk (nil = replan), except a cancelled ctx
// and any other Clear error (leave game, death, town portal), which ends the walk and propagates.
func PackYield(d YieldDeps) func(ctx context.Context, near []d2.Unit) error {
	if d.Radius <= 0 {
		d.Radius = 12
	}
	if d.Timeout <= 0 {
		d.Timeout = 20 * time.Second
	}
	return func(ctx context.Context, near []d2.Unit) error {
		snap, err := d.Snapshot()
		if err != nil || snap == nil || snap.Player == nil || snap.Player.InTown() {
			return nil
		}
		hostile := false
		for _, u := range near {
			if !d.Friendly(snap, u) {
				hostile = true
				break
			}
		}
		if !hostile {
			return nil
		}
		err = d.Clear(ctx, d.Radius, d.Timeout)
		if cerr := ctx.Err(); cerr != nil {
			return cerr
		}
		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return nil
	}
}

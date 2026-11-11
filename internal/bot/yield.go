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

// yieldMaxRounds bounds one hand-over: rounds of d.Timeout each (4 s in the live wiring, so about 40 s).
const yieldMaxRounds = 10

// remaining counts the living, non-friendly monsters within the clear radius of the player.
func (d YieldDeps) remaining(s *d2.Snapshot) int {
	n := 0
	for _, u := range s.Monsters {
		if !u.Alive() || u.Type != d2.TypeMonster || d.Friendly(s, u) {
			continue
		}
		dx, dy := float64(u.X-s.Player.X), float64(u.Y-s.Player.Y)
		if dx*dx+dy*dy <= float64(d.Radius*d.Radius) {
			n++
		}
	}
	return n
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
		// One call is short on purpose (the unpinned guard needs to see a stuck character), but a pack is not done after
		// one: live d2play-20261108 had 48 of 108 big fights end at exactly the 4 s timeout with the rest of the pack
		// alive, and the walk then ran on ("kills two or three, then runs away"). Go on while hostiles stay in reach
		// and the count keeps falling; two rounds without a kill, or yieldMaxRounds, end it.
		last, stale := d.remaining(snap), 0
		for round := 0; round < yieldMaxRounds; round++ {
			err = d.Clear(ctx, d.Radius, d.Timeout)
			if cerr := ctx.Err(); cerr != nil {
				return cerr
			}
			if err != nil && !errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			s, serr := d.Snapshot()
			if serr != nil || s == nil || s.Player == nil || s.Player.InTown() {
				return nil
			}
			left := d.remaining(s)
			if left == 0 {
				return nil
			}
			if left >= last {
				if stale++; stale >= 2 {
					return nil
				}
			} else {
				stale = 0
			}
			last = left
		}
		return nil
	}
}

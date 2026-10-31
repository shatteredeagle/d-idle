package main

import (
	"context"
	"errors"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/town"
)

// toTown is a copy of cmd/d2bot townAdapter.ToTown: reuse a portal the fighter already opened next to the player,
// else let the Townsman open one.
func toTown(e *env.Env, t *town.Town) func(context.Context) error {
	return func(ctx context.Context) error {
		s, err := e.Snapshot()
		if err == nil && s.Player != nil && !s.Player.InTown() {
			for _, o := range s.Objects {
				if o.Class != town.ObjPortal || max(abs(o.X-s.Player.X), abs(o.Y-s.Player.Y)) > 25 {
					continue
				}
				if err := e.In.Interact(input.Target{Type: town.TypeObject, ID: o.ID, Pos: input.Point{X: o.X, Y: o.Y}}); err != nil {
					break
				}
				if waitTown(ctx, e, 15*time.Second) == nil {
					return nil
				}
				break
			}
		}
		return t.ToTown(ctx)
	}
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func waitTown(ctx context.Context, e *env.Env, d time.Duration) error {
	deadline := e.Clock.Now().Add(d)
	for e.Clock.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if s, err := e.Snapshot(); err == nil && s.State == d2.StateInGame && s.Player != nil && s.Player.InTown() {
			return nil
		}
		e.Clock.Sleep(150 * time.Millisecond)
	}
	return errors.New("portal did not lead to town")
}

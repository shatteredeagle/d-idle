package bot

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
)

// slowBack is a Townsman whose way back to the field lands in a field with hostiles and then keeps walking, until its
// context is cancelled: the 54 s and 124 s of "Returning to the field" in the Doofer log of 2026-11-27 (problem 4).
type slowBack struct {
	*fakeTown
	g *rig
}

func (s *slowBack) BackToField(ctx context.Context, l int) error {
	s.g.w.set(func(w *world) { w.area = 3 })
	s.g.w.addMonster(900, 1, 4, 0)
	s.g.w.addMonster(901, 1, -5, 3)
	for ctx.Err() == nil {
		s.g.clock.Sleep(100 * time.Millisecond)
	}
	return ctx.Err()
}

// With hostiles near on the way back the return leg ends within seconds of virtual time, the trip is over and the
// fighter has the hero. Two classes: the rule is about the place, not the build.
func TestReturnLegHandsOverToTheFighter(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		t.Run(class.String(), func(t *testing.T) {
			g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) {
				d.Town = &slowBack{fakeTown: &fakeTown{}, g: nil}
			})
			st := g.r.d.Town.(*slowBack)
			st.g, st.fakeTown.w = g, g.w
			g.w.set(func(w *world) { w.class = class; w.state = d2.StateInGame; w.area = 1; w.life, w.maxL = 100, 100 })
			snap := g.r.poll()
			g.r.loop.trip = &trip{plan: g.r.fullChores("kit: test"), returnTo: 3, stage: 3}
			start := g.clock.Now()
			done := make(chan bool, 1)
			go func() { done <- g.r.advanceTrip(context.Background(), snap, g.r.control()) }()
			select {
			case <-done:
			case <-time.After(20 * time.Second):
				t.Fatal("the return leg stood in the field with hostiles and was never handed over")
			}
			if d := g.clock.Now().Sub(start); d > 10*time.Second {
				t.Fatalf("handed over after %v of virtual time", d)
			}
			if g.r.loop.trip != nil || g.r.loop.failures != 0 {
				t.Fatalf("trip=%v failures=%d, want the trip ended without a failure", g.r.loop.trip, g.r.loop.failures)
			}
		})
	}
}

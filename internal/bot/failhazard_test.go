package bot

import (
	"context"
	"errors"
	"testing"

	"d2lod-bot/internal/d2"
)

// A failed or refused step ran no action, so no guard watched the hero (lab night 2026-11-28, drill.act2.cube.stepexec:
// potion-not-drunk, life 99 to 18 percent between two refused steps with four potions in the belt). The failure path asks
// the safety check before it waits. Must fail without that: no "act" call. In town nothing is asked. Two classes.
func TestFailedStepAsksTheSafetyCheck(t *testing.T) {
	for _, cl := range []d2.Class{d2.Paladin, d2.Sorceress} {
		g := newRig(t)
		g.w.class = cl
		g.w.set(func(w *world) { w.area, w.life = 48, 35 })
		g.fight.safety = func(s *d2.Snapshot) SafetyAction {
			if s.Player.Life*2 < s.Player.MaxLife {
				return SafetyDrinkLife
			}
			return SafetyNone
		}
		g.r.poll()
		g.r.fail(context.Background(), control{}, "quest board step", errors.New("no eligible reachable leveling area"))
		if g.w.count("act "+SafetyDrinkLife.String()) != 1 {
			t.Fatalf("class %v: no drink after a failed step in the field: %v", cl, g.w.log())
		}
		g = newRig(t)
		g.w.class = cl
		g.w.set(func(w *world) { w.life = 35 }) // area 1: town
		g.fight.safety = func(*d2.Snapshot) SafetyAction { return SafetyDrinkLife }
		g.r.poll()
		g.r.fail(context.Background(), control{}, "quest board step", errors.New("x"))
		if g.w.count("act ") != 0 {
			t.Fatalf("class %v: the safety check drank in town", cl)
		}
	}
}

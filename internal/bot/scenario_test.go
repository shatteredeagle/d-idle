package bot

// Scenario corpus, bot half (docs/scenarios.md): recorded real failure situations replayed through the Runner as
// end-state tests. Each test is named TestScenario_<Name>, states the situation and the evidence, and has a knob-off
// subtest that EXPECTS the old failure, so the corpus proves it can see the bug it pins.

import (
	"fmt"
	"testing"
)

// visited lists the areas the fake mover was sent to (exits and waypoints) in call order.
func (w *world) visited() map[int]int {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := map[int]int{}
	for _, c := range w.calls {
		var n int
		if _, err := fmt.Sscanf(c, "exit %d", &n); err == nil {
			out[n]++
		} else if _, err := fmt.Sscanf(c, "wp %d", &n); err == nil {
			out[n]++
		}
	}
	return out
}

// Situation: a clvl 7 caster in recovery mode, 0 healing potions, no town portal scroll, 35 gold, no merc, Blood Moor
// and Cold Plains spent (cleared in this game). Evidence (docs/kitgate.md, soak seed 5 of the d2d livestream
// run, VERIFIED from the log): it walked to Burial Grounds, died there at 35 gold with 49 x "no town portal
// available", and had near-deaths in the Crypt and Mausoleum. The fall-through to the quest ladder was INFERRED by
// reading target0 (not replayed live). End state: no dungeon mouth is ever entered, the goal waits at home.
func TestScenario_KitlessClvl7PackDeath(t *testing.T) {
	setup := func(gate int) *rig {
		g := newRig(t)
		g.e.Tune.Resource.RecoveryKitGate = gate
		g.mover.exploreFn = blockExplore
		g.w.set(func(w *world) { w.level = 7; w.gold0 = 35; w.area = 1 })
		g.w.belt() // 0 healing potions; the fake pack holds no town portal scroll
		g.town.choresErr = errBoom
		g.r.loop.cleared = map[int]bool{2: true, 3: true} // Blood Moor and Cold Plains spent in this game
		return g
	}
	mouths := func(g *rig) []int {
		var in []int
		for a := range g.w.visited() {
			if dungeonMouths[a] {
				in = append(in, a)
			}
		}
		return in
	}
	steps := func(g *rig, n int64) {
		g.t.Helper()
		start := g.w.snaps.Load()
		g.waitFor("loop steps", func() bool { return g.w.snaps.Load() >= start+n })
	}

	t.Run("gate on: waits at home", func(t *testing.T) {
		g := setup(1)
		g.run()
		defer g.finish()
		g.r.Start(reach(15))
		steps(g, 400)
		if in := mouths(g); len(in) != 0 {
			t.Fatalf("entered dungeon mouths %v; calls %s", in, g.w.log())
		}
		if a := g.r.Status().Character.AreaID; a == 17 || a == 18 || a == 19 {
			t.Fatalf("standing in area %d", a)
		}
		for a := range g.w.visited() {
			if a != 1 && a != 2 && a != 3 {
				t.Fatalf("left the safe fields for area %d", a)
			}
		}
	})

	// Knob off (resource.recoveryKitGate 0): the old fall-through to the quest ladder returns.
	t.Run("gate off: old failure reproduces", func(t *testing.T) {
		g := setup(0)
		g.run()
		defer g.finish()
		g.r.Start(reach(15))
		g.waitFor("old behaviour walks into a dungeon area", func() bool { return len(mouths(g)) > 0 })
	})
}

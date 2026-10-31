package bot

// Scenario corpus, bot half (docs/scenarios.md): recorded real failure situations replayed through the Runner as
// end-state tests. Each test is named TestScenario_<Name>, states the situation and the evidence, and has a knob-off
// subtest that EXPECTS the old failure, so the corpus proves it can see the bug it pins.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"d2lod-bot/internal/goals"
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

// Situation: a kit-less clvl 7 recovery character at home, both safe fields banned for hours (a frozen ban set) and
// the virtual clock advancing: target0 answers "no eligible area" every minute, the wall. Evidence
// (docs/probes/navsoak-compare.md, VERIFIED from the logs of two soaks): 51.9k and 173k "no eligible reachable
// leveling area" lines after the town wedge, 0 "Entered" lines, the clock advanced so the loop watchdog never fired;
// 60 and 167 wall minutes lost. End state: after bot.stallWarnMin virtual minutes without progress a fresh game
// starts, after bot.stallExitMin more Run would return ErrCampaignStalled (exit status 5, supervisor restart).
// Driven step by step (one virtual minute per step: the planner verdict, then the loop's stall hook) because the real
// loop farms a banned area anyway after its idle cap (areaskip.go), so the live wall needs the nav wedge that
// TestScenario_WedgePocket pins in the nav package; this test pins the safety net behind it.
func TestScenario_CampaignStall(t *testing.T) {
	setup := func(warn int) *rig {
		g := newRig(t)
		g.r.d.Cycler = fakeCycler{g.w}
		g.e.Tune.Bot.StallWarnMin, g.e.Tune.Bot.StallExitMin = warn, 30
		g.w.set(func(w *world) { w.level = 7; w.area = 1; w.gold0 = 35 })
		g.w.belt()
		g.r.loop.broke.on = true
		for _, a := range []int{2, 3} {
			g.r.avoid[a] = g.clock.Now().Add(1000 * time.Hour)
		}
		g.r.poll()
		g.r.campaignStall(context.Background(), g.r.latest(), true) // arms the window at now
		return g
	}
	// minute advances one virtual minute and reports the wall and whether the ladder acted.
	minute := func(g *rig) (wall, acted bool) {
		g.clock.Sleep(time.Minute)
		g.r.poll()
		s := g.r.latest()
		_, ok := g.r.target0(s, control{spec: goalSpec{kind: goals.ReachLevel}})
		return !ok, g.r.campaignStall(context.Background(), s, true)
	}

	t.Run("ladder on: warns, fresh game, exit 5", func(t *testing.T) {
		g := setup(30)
		warnedAt, exitAt := 0, 0
		for m := 1; m <= 90 && exitAt == 0; m++ {
			wall, acted := minute(g)
			if !wall {
				t.Fatalf("minute %d: the planner found an area, the wall is gone", m)
			}
			switch {
			case acted && warnedAt == 0:
				warnedAt = m
			case acted && g.stalled() != nil:
				exitAt = m
			}
		}
		if warnedAt < 30 || warnedAt > 32 {
			t.Fatalf("first window at minute %d, want 30", warnedAt)
		}
		if exitAt < 60 || exitAt > 63 || !errors.Is(g.stalled(), ErrCampaignStalled) {
			t.Fatalf("exit at minute %d with %v, want ErrCampaignStalled at 60", exitAt, g.stalled())
		}
		g.r.obMu.Lock()
		rec := g.r.ob.recycles
		g.r.obMu.Unlock()
		if rec != 1 {
			t.Fatalf("fresh games %d, want exactly 1 before the exit", rec)
		}
	})

	// Knob off (bot.stallWarnMin 0): the old wall reproduces, 5 virtual hours pass with no fresh game and no exit.
	t.Run("ladder off: old wall reproduces", func(t *testing.T) {
		g := setup(0)
		for m := 1; m <= 300; m++ {
			if wall, acted := minute(g); !wall || acted {
				t.Fatalf("minute %d: wall %v acted %v", m, wall, acted)
			}
		}
		g.r.obMu.Lock()
		rec := g.r.ob.recycles
		g.r.obMu.Unlock()
		if rec != 0 || g.stalled() != nil {
			t.Fatalf("ladder off but recycles %d crash %v", rec, g.stalled())
		}
	})
}

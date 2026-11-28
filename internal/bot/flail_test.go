package bot

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
)

// flailFixture is the cycle from the sandbox log of 2026-11-18 18:13:40 to 18:14:20 (seconds since the first line,
// area id): Dark Wood 5, Rogue Encampment 1, Cold Plains 3. V: copied from the d2play docker log, "Entered ..." lines.
var flailFixture = []struct {
	dt   int
	area int
}{
	{0, 5}, {2, 1}, {1, 5}, {2, 3}, {2, 5}, {2, 1}, {1, 5}, {2, 3}, {2, 5}, {2, 1}, {1, 5}, {2, 3}, {2, 5}, {2, 1}, {1, 5},
}

func feedFlailFixture(g *rig) {
	for _, m := range flailFixture {
		g.clock.Sleep(time.Duration(m.dt) * time.Second)
		g.r.loop.flail.noteMove(g.clock.Now(), m.area, 0)
	}
}

func TestFlailHoldsInsteadOfGoingHome(t *testing.T) {
	g := newRig(t)
	g.r.e.Tune.Bot.FlailMoves = env.DefaultTuning().Bot.FlailMoves
	g.w.set(func(w *world) { w.area = 5 })
	feedFlailFixture(g)
	g.r.poll()
	g.r.flailHold(context.Background(), g.r.latest(), levelGoalControl())
	if g.r.loop.flail.holdUntil.IsZero() {
		t.Fatal("the observed cycle did not start a hold")
	}
	if n := g.w.count("totown"); n != 0 {
		t.Fatalf("flailing bot went home %d times", n)
	}
	// still held a minute later, free after the hold
	g.clock.Sleep(time.Minute)
	g.r.poll()
	if _, _, holding := g.r.loop.flail.check(g.clock.Now(), 0, 6, 10*time.Minute, 5*time.Minute, g.r.areaName); !holding {
		t.Fatal("hold ended early")
	}
	g.clock.Sleep(5 * time.Minute)
	if _, _, holding := g.r.loop.flail.check(g.clock.Now(), 0, 6, 10*time.Minute, 5*time.Minute, g.r.areaName); holding {
		t.Fatal("hold never ended")
	}
}

// The control: with the guard off the same cycle sends the hero home (the loop of the log).
func TestFlailGuardOffGoesHome(t *testing.T) {
	g := newRig(t)
	g.r.e.Tune.Bot.FlailMoves = 0
	g.w.set(func(w *world) { w.area = 5 })
	feedFlailFixture(g)
	g.r.poll()
	g.r.noTarget(context.Background(), g.r.latest(), levelGoalControl())
	if n := g.w.count("totown"); n != 1 {
		t.Fatalf("control: %d trips", n)
	}
}

func TestFlailNeedsNoProgress(t *testing.T) {
	var g flailGuard
	now := time.Unix(1000, 0)
	nm := func(a int) string { return "a" }
	for i := 0; i < 8; i++ {
		g.noteMove(now.Add(time.Duration(i)*time.Second), i%2, uint64(i)) // a kill between every move
	}
	if _, started, _ := g.check(now.Add(9*time.Second), 8, 6, 10*time.Minute, 5*time.Minute, nm); started {
		t.Fatal("moves with kills in between are not flailing")
	}
	// slow moves outside the window are not flailing either
	var h flailGuard
	for i := 0; i < 8; i++ {
		h.noteMove(now.Add(time.Duration(i)*3*time.Minute), i%2, 0)
	}
	if _, started, _ := h.check(now.Add(24*time.Minute), 0, 6, 10*time.Minute, 5*time.Minute, nm); started {
		t.Fatal("slow moves are not flailing")
	}
}

func TestFlailIgnoresLinearTravel(t *testing.T) {
	var g flailGuard
	now := time.Unix(1000, 0)
	for i := 0; i < 8; i++ {
		g.noteMove(now.Add(time.Duration(i)*time.Second), i+1, 0) // eight different areas
	}
	if _, started, _ := g.check(now.Add(10*time.Second), 0, 6, 10*time.Minute, 5*time.Minute, func(int) string { return "a" }); started {
		t.Fatal("a walk through new areas is not a loop")
	}
}

// cainspin (TASK-1103, Proving Ground 20261127-232846): a flail hold that started in town carried on in Tower Cellar 3
// after the sweep had cleared it. Every hold tick swept the cleared area again ("roam: done", "area cleared", "run
// finished" each ~4 ms) and none of it slept, so the lockstep clock stood at 1719 s for 33 wall minutes and the
// 5 minute hold could never end. A hold in a cleared field must let virtual time pass.
func TestFlailHoldInClearedAreaLetsTimePass(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		t.Run(class.String(), func(t *testing.T) {
			g := newRig(t)
			g.r.e.Tune.Bot.FlailMoves = env.DefaultTuning().Bot.FlailMoves
			g.w.set(func(w *world) { w.area, w.class = 5, class })
			feedFlailFixture(g)
			g.r.poll()
			c := levelGoalControl()
			g.r.flailHold(context.Background(), g.r.latest(), c)
			if g.r.loop.flail.holdUntil.IsZero() {
				t.Fatal("the observed cycle did not start a hold")
			}
			g.r.loop.cleared[5] = true // the sweep emptied the area in this game
			start := g.clock.Now()
			for i := 0; i < 500; i++ {
				g.r.poll()
				if !g.r.flailHold(context.Background(), g.r.latest(), c) {
					if g.clock.Now().Sub(start) < 4*time.Minute {
						t.Fatalf("hold released after %v virtual, before it ran out", g.clock.Now().Sub(start))
					}
					return
				}
			}
			t.Fatalf("hold still running after 500 ticks; virtual time moved %v (the spin of cain.chain seed 11)", g.clock.Now().Sub(start))
		})
	}
}

// The same freeze through any other caller: a sweep pass in an area already cleared in this game that kills nothing
// must take at least one tick of virtual time, so every hold measured in virtual time (area done gap, board hold) ends.
func TestSweepPassOnClearedAreaTakesATick(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		t.Run(class.String(), func(t *testing.T) {
			g := newRig(t)
			g.w.set(func(w *world) { w.area, w.class = 5, class })
			g.r.poll()
			g.r.loop.cleared[5] = true
			start := g.clock.Now()
			if err := g.r.sweepPass(context.Background(), 5); err != nil {
				t.Fatal(err)
			}
			if g.clock.Now().Sub(start) < time.Duration(g.r.e.Tune.Bot.TickMs)*time.Millisecond {
				t.Fatalf("a pass over a cleared area took %v of virtual time", g.clock.Now().Sub(start))
			}
		})
	}
}

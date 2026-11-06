package bot

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town"
)

// F2 (g2e seed 11): low life, no potion and no town portal in the field leaves the game after bot.noPortalWaitSec
// instead of waiting lowLifeRetry again and again.
func TestNoPortalLowLifeLeavesGame(t *testing.T) {
	for _, cl := range nokillClasses {
		t.Run(cl.String(), func(t *testing.T) {
			g := newRig(t)
			g.r.d.Cycler = fakeCycler{g.w}
			g.mover.exploreFn = blockExplore
			g.town.toTownErr = fmt.Errorf("heal: %w", town.ErrNoPortal)
			g.w.set(func(w *world) { w.class = cl; w.level = 3; w.area = 4; w.life = 20 })
			g.w.belt()
			g.run()
			defer g.finish()
			g.r.Start(farm("Stony Field"))
			g.waitFor("fresh game instead of waiting", func() bool { return g.w.count("newgame") > 0 })
			if strings.Contains(g.r.Status().Activity, "retrying in") && g.w.count("newgame") == 0 {
				t.Fatal("still waiting")
			}
		})
	}
}

// With the knob at 240 the old wait is back: no game is left.
func TestNoPortalWaitKnobKeepsOldWait(t *testing.T) {
	g := newRig(t)
	g.r.d.Cycler = fakeCycler{g.w}
	g.r.e.Tune.Bot.NoPortalWaitSec = 240
	g.mover.exploreFn = blockExplore
	g.town.toTownErr = town.ErrNoPortal
	g.w.set(func(w *world) { w.level = 3; w.area = 4; w.life = 20 })
	g.w.belt()
	g.run()
	defer g.finish()
	g.r.Start(farm("Stony Field"))
	g.waitFor("waiting", func() bool { return strings.Contains(g.r.Status().Activity, "way to town failed") })
	if n := g.w.count("newgame"); n != 0 {
		t.Fatalf("%d fresh games with the knob at 240", n)
	}
}

// F3: stage 2 follows stage 1 after bot.droughtStage2Min, not after another bot.droughtMin.
func TestDroughtStage2Soon(t *testing.T) {
	for _, cl := range nokillClasses {
		t.Run(cl.String(), func(t *testing.T) {
			g := droughtRig(t, cl, true)
			g.r.obMu.Lock()
			g.r.ob.targetArea = 2
			g.r.obMu.Unlock()
			if !g.droughtStep(9 * time.Minute) {
				t.Fatal("stage 1 must act")
			}
			if g.droughtStep(time.Minute) || strings.Contains(g.w.log(), "newgame") {
				t.Fatal("one minute after stage 1 is too early")
			}
			if !g.droughtStep(90*time.Second) || !strings.Contains(g.w.log(), "newgame") {
				t.Fatal("2 minutes after stage 1 must start a fresh game")
			}
		})
	}
}

// F3: knob 0 restores the old second-stage timing (a full droughtMin).
func TestDroughtStage2KnobZero(t *testing.T) {
	g := droughtRig(t, d2.Sorceress, true)
	g.r.e.Tune.Bot.DroughtStage2Min = 0
	g.r.obMu.Lock()
	g.r.ob.targetArea = 2
	g.r.obMu.Unlock()
	g.droughtStep(9 * time.Minute)
	if g.droughtStep(3 * time.Minute) {
		t.Fatal("stage 2 must wait the full limit with the knob at 0")
	}
}

// F4: no unplanned fresh game within bot.freshAfterRecoverySec of a lost-game recovery, except at low life.
func TestFreshHoldAfterRecovery(t *testing.T) {
	for _, cl := range nokillClasses {
		t.Run(cl.String(), func(t *testing.T) {
			g := droughtRig(t, cl, false) // a farm goal cannot skip: the drought goes straight to a fresh game
			g.r.loop.recoveredAt = g.r.e.Clock.Now()
			if !g.droughtStep(9*time.Minute) || !strings.Contains(g.w.log(), "newgame") {
				t.Fatal("the recovery was 9 minutes ago: the fresh game is allowed")
			}
		})
	}
	g := droughtRig(t, d2.Sorceress, false)
	g.clock.Sleep(9 * time.Minute)
	g.r.loop.recoveredAt = g.r.e.Clock.Now().Add(-10 * time.Second)
	g.r.poll()
	if h := g.r.freshHold(g.r.latest().Player); h < 100*time.Second || h > 180*time.Second {
		t.Fatalf("hold %v", h)
	}
	if g.r.droughtCheck(t.Context(), g.r.latest(), g.r.control()) || strings.Contains(g.w.log(), "newgame") {
		t.Fatal("a fresh game 10 s after a recovery")
	}
	g.w.set(func(w *world) { w.life = 30 })
	g.r.poll()
	if h := g.r.freshHold(g.r.latest().Player); h != 0 {
		t.Fatalf("low life must lift the hold, got %v", h)
	}
	g.r.e.Tune.Bot.FreshAfterRecoverySec = 0
	g.w.set(func(w *world) { w.life = 100 })
	g.r.poll()
	if h := g.r.freshHold(g.r.latest().Player); h != 0 {
		t.Fatalf("knob 0 is off, got %v", h)
	}
}

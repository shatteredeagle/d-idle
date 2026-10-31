package bot

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/launch"
)

// Failsafe layer 1 (PLAN.md "Failsafes"): each test injects one stuck state with fakes and proves the ladder
// escalates in order and ends in a visible pause, never a hang.

func failsafeRig(t *testing.T, mutate ...func(*Config, *Deps, *env.Tuning)) *rig {
	t.Helper()
	g := newRig(t, append([]func(*Config, *Deps, *env.Tuning){func(c *Config, d *Deps, tu *env.Tuning) {
		c.StuckAfterSec, c.TownTimeoutSec, c.RecoverTimeoutSec = 30, 20, 100000
		c.MaxFailsafeRestarts = 1
		tu.Bot.CorpseRetries = 0
	}}, mutate...)...)
	g.r.d.Cycler = fakeCycler{g.w}
	g.r.d.Restart = func(ctx context.Context) error { g.w.call("restart"); return nil }
	return g
}

// ordered asserts the calls appear in this order in the world log.
func ordered(t *testing.T, g *rig, parts ...string) {
	t.Helper()
	log, at := g.w.log(), 0
	for _, p := range parts {
		i := strings.Index(log[at:], p)
		if i < 0 {
			t.Fatalf("%q missing or out of order in: %s", p, log)
		}
		at += i + len(p)
	}
}

func assertPausedWithReason(t *testing.T, g *rig) {
	t.Helper()
	g.waitFor("paused itself", func() bool { return g.r.Status().State == goals.RunnerPaused })
	if m := g.r.Status().Message; !strings.Contains(m, "Failsafe") {
		t.Fatalf("message %q", m)
	}
	// the event reaches the collector goroutine after the state flips: wait for it instead of racing it
	g.waitFor("the failsafe error event", func() bool { return len(g.events(goals.EvError)) > 0 })
	errs := g.events(goals.EvError)
	if len(errs) == 0 || errs[len(errs)-1].Fatal || !strings.Contains(errs[len(errs)-1].Message, "Failsafe") {
		t.Fatalf("want a visible non-fatal failsafe error, got %+v", errs)
	}
	h := g.r.Health()
	if !h.SelfPaused || h.LastError == "" || h.Status != "paused" {
		t.Fatalf("health %+v", h)
	}
}

func TestFailsafeTravelWithNoProgress(t *testing.T) {
	g := failsafeRig(t)
	g.mover.exitBlock = true // the exit never leads anywhere
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	assertPausedWithReason(t, g)
	ordered(t, g, "exit 2", "closeall", "newgame", "restart") // town is skipped: already there
	if h := g.r.Health(); h.StuckReason == "" {
		t.Fatalf("health has no stuck reason: %+v", h)
	}
}

func TestFailsafeClearingWithNoProgress(t *testing.T) {
	g := failsafeRig(t)
	g.mover.exploreFn = func(ctx context.Context, done func() bool) error { <-ctx.Done(); return ctx.Err() }
	g.w.set(func(w *world) { w.area = 2 })
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	assertPausedWithReason(t, g)
	ordered(t, g, "closeall", "totown", "newgame", "restart")
}

func TestFailsafeTownTripThatNeverReturns(t *testing.T) {
	g := failsafeRig(t)
	g.town.toTownBlock = true
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.area = 2 })
	g.w.belt() // no potions: a trip is due at once
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	assertPausedWithReason(t, g)
	if g.w.count("totown") == 0 {
		t.Fatal("no trip was attempted")
	}
}

func TestFailsafePanelLeftOpen(t *testing.T) {
	g := failsafeRig(t)
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.area = 2 })
	g.run()
	defer g.finish()
	g.r.d.Loot = fakeLoot{w: g.w, err: errors.New("hand: panels still open")}
	g.r.Start(farm("Blood Moor"))
	assertPausedWithReason(t, g)
	ordered(t, g, "pickup", "closeall")
}

func TestFailsafeDeathOverlayNeverDismissed(t *testing.T) {
	g := failsafeRig(t)
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.area = 2 })
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("in field", func() bool { return g.w.count("explore") > 0 })
	die(g) // and nobody (not even our Esc) ever brings it back
	assertPausedWithReason(t, g)
	if g.w.count("key esc") < 5 {
		t.Fatalf("Esc pressed only %d times", g.w.count("key esc"))
	}
	ordered(t, g, "key esc", "newgame", "restart")
	if g.r.State() != StateRecovering {
		t.Fatalf("state %s", g.r.State())
	}
}

func TestFailsafeLadderRestartsAfterRealProgress(t *testing.T) {
	g := failsafeRig(t)
	g.mover.exploreFn = func(ctx context.Context, done func() bool) error { <-ctx.Done(); return ctx.Err() }
	g.w.set(func(w *world) { w.area = 2 })
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("rung 1", func() bool { return g.r.Health().LadderStep >= 1 })
	g.w.set(func(w *world) { w.exp += 100 }) // experience arrives: the ladder starts over
	g.waitFor("ladder reset", func() bool {
		h := g.r.Health()
		return h.ProgressCounter > 0 && h.StuckReason == ""
	})
}

func TestFailsafeGameGoneStaysFatal(t *testing.T) {
	g := failsafeRig(t)
	g.r.d.Restart = nil // unmanaged: nothing can restart the game
	g.mover.exploreFn = blockExplore
	g.run()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("working", func() bool { return g.w.count("explore") > 0 })
	g.health.mu.Lock()
	g.health.ev = launch.EventCrashed
	g.health.mu.Unlock()
	g.waitFor("fatal", func() bool { return g.r.Health().Fatal != "" })
	if err := g.finish(); err == nil {
		t.Fatal("no fatal error")
	}
	if g.r.Health().Fatal == "" {
		t.Fatal("health does not report the fatal error")
	}
}

func TestHealthReportsProgress(t *testing.T) {
	g := newRig(t)
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.area = 2 })
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("working", func() bool { return g.w.count("explore") > 0 })
	h0 := g.r.Health()
	g.w.set(func(w *world) { w.x += 20 })
	g.waitFor("counter moves with the character", func() bool { return g.r.Health().ProgressCounter > h0.ProgressCounter })
	h := g.r.Health()
	if h.State != StateClearing || h.Goal != "g1" || h.SinceSeconds < 0 || h.LadderStep != 0 {
		t.Fatalf("health %+v", h)
	}
}

// The restart rung kills the game on purpose: the exit must not be reported as "the game is gone", the run continues.
func TestFailsafeRestartIsNotFatalAndGoalContinues(t *testing.T) {
	g := failsafeRig(t, func(c *Config, d *Deps, tu *env.Tuning) { c.MaxFailsafeRestarts = 3 })
	var restarted atomic.Bool
	g.r.d.Restart = func(ctx context.Context) error {
		g.w.call("restart")
		g.health.mu.Lock()
		g.health.ev = launch.EventExited // what the supervisor sees while the recycler kills the process
		g.health.mu.Unlock()
		g.e.Clock.Sleep(3 * time.Second) // the guard polls the health meanwhile
		restarted.Store(true)
		return nil
	}
	g.mover.exploreFn = func(ctx context.Context, done func() bool) error {
		if restarted.Load() {
			g.w.set(func(w *world) { w.x += 50; w.exp += 10 })
			g.e.Clock.Sleep(time.Second)
			return nil
		}
		<-ctx.Done()
		return ctx.Err()
	}
	g.w.set(func(w *world) { w.area = 2 })
	g.run()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("restart rung", func() bool { return restarted.Load() })
	g.waitFor("working again", func() bool {
		return g.r.Health().StuckReason == "" && g.r.Health().ProgressCounter > 0 && g.w.count("explore") > 5
	})
	if f := g.r.Health().Fatal; f != "" {
		t.Fatalf("fatal after a deliberate restart: %s", f)
	}
	if err := g.finish(); err != nil {
		t.Fatalf("run ended with %v", err)
	}
}

// A real exit that is not the ladder's own stays fatal even with restarts allowed (TestFailsafeGameGoneStaysFatal).

// Clearing has its own, shorter idle limit; a character that keeps moving is never flagged by it.
func TestClearIdleCatchesStallEarly(t *testing.T) {
	g := failsafeRig(t, func(c *Config, d *Deps, tu *env.Tuning) { c.StuckAfterSec, c.ClearIdleSec = 1000, 20 })
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.area = 2 })
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("rung 1 within the idle limit", func() bool { return g.r.Health().LadderStep >= 1 })
}

func TestClearIdleQuietWhileMoving(t *testing.T) {
	g := failsafeRig(t, func(c *Config, d *Deps, tu *env.Tuning) { c.StuckAfterSec, c.ClearIdleSec = 1000, 20 })
	g.mover.exploreFn = func(ctx context.Context, done func() bool) error {
		g.w.set(func(w *world) { w.x += 20 })
		g.e.Clock.Sleep(time.Second)
		return nil
	}
	g.w.set(func(w *world) { w.area = 2 })
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("exploring", func() bool { return g.w.count("explore") > 3 })
	for i := 0; i < 100; i++ {
		g.e.Clock.Sleep(time.Second)
	}
	if h := g.r.Health(); h.LadderStep != 0 {
		t.Fatalf("ladder %d while moving", h.LadderStep)
	}
}

// The game process dies by itself (soak 20261029-035908: after a chicken Save and Exit): in managed mode the run restarts
// it and the goal continues instead of ending with "the game is gone".
func TestGameLostAfterChickenLeaveRestartsAndContinues(t *testing.T) {
	g := failsafeRig(t, func(c *Config, d *Deps, tu *env.Tuning) { c.MaxFailsafeRestarts = 3 })
	var restarted atomic.Bool
	g.r.d.Restart = func(ctx context.Context) error {
		g.w.call("restart")
		restarted.Store(true)
		return nil
	}
	g.mover.exploreFn = func(ctx context.Context, done func() bool) error {
		g.w.set(func(w *world) { w.x += 50; w.exp += 10 })
		g.e.Clock.Sleep(time.Second)
		return nil
	}
	g.w.set(func(w *world) { w.area = 2 })
	g.run()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("exploring", func() bool { return g.w.count("explore") > 2 })
	g.health.mu.Lock()
	g.health.ev = launch.EventExited
	g.health.mu.Unlock()
	g.waitFor("restart", func() bool { return restarted.Load() })
	g.health.mu.Lock()
	g.health.ev = launch.EventNone
	g.health.mu.Unlock()
	n := g.w.count("explore")
	g.waitFor("working again", func() bool { return g.w.count("explore") > n+3 })
	if f := g.r.Health().Fatal; f != "" {
		t.Fatalf("fatal: %s", f)
	}
	if err := g.finish(); err != nil {
		t.Fatalf("run ended with %v", err)
	}
}

// A restart that fails, or a game that keeps dying without progress, still ends the run.
func TestGameLostRestartFailureIsFatal(t *testing.T) {
	g := failsafeRig(t)
	g.r.d.Restart = func(ctx context.Context) error { return errors.New("launch failed") }
	g.mover.exploreFn = blockExplore
	g.run()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("working", func() bool { return g.w.count("explore") > 0 })
	g.health.mu.Lock()
	g.health.ev = launch.EventExited
	g.health.mu.Unlock()
	g.waitFor("fatal", func() bool { return g.r.Health().Fatal != "" })
	g.finish()
}

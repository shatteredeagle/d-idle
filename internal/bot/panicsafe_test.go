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
	"d2lod-bot/internal/input"
)

// panicMover panics in its movement calls while n > 0, then behaves like the fake.
type panicMover struct {
	*fakeMover
	n atomic.Int32
}

func (m *panicMover) hit() {
	if m.n.Add(-1) >= 0 {
		var p *struct{ x int }
		_ = p.x // the incident: a nil dereference deep inside an action
	}
}
func (m *panicMover) ExitTo(ctx context.Context, l int) error {
	m.hit()
	return m.fakeMover.ExitTo(ctx, l)
}
func (m *panicMover) Explore(ctx context.Context, done func() bool) error {
	m.hit()
	return m.fakeMover.Explore(ctx, done)
}
func (m *panicMover) GoTo(ctx context.Context, p input.Point) error {
	m.hit()
	return m.fakeMover.GoTo(ctx, p)
}

func panicRig(t *testing.T, panics int32) (*rig, *panicMover) {
	t.Helper()
	var pm *panicMover
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) {
		pm = &panicMover{fakeMover: d.Mover.(*fakeMover)}
		pm.n.Store(panics)
		d.Mover = pm
	})
	return g, pm
}

func TestPanicIsRecoveredAndRunnerKeepsGoing(t *testing.T) {
	g, pm := panicRig(t, 2)
	g.run()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("panic events", func() bool { return len(g.events(goals.EvError)) >= 2 })
	g.waitFor("calls after the panics", func() bool { return pm.n.Load() < 0 })
	for _, ev := range g.events(goals.EvError) {
		if ev.Fatal || !strings.Contains(ev.Message, "panic") {
			t.Fatalf("unexpected error event %+v", ev)
		}
	}
	if h := g.r.Health(); h.Panics != 2 || h.Fatal != "" || h.SelfPaused {
		t.Fatalf("health %+v", h)
	}
	if err := g.finish(); err != nil {
		t.Fatalf("Run returned %v", err)
	}
}

func TestPanicBudgetSelfPausesAndResumes(t *testing.T) {
	g, _ := panicRig(t, 1000)
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("self pause", func() bool { return g.r.Status().State == goals.RunnerPaused })
	if m := g.r.Status().Message; !strings.Contains(m, "internal errors") {
		t.Fatalf("message %q", m)
	}
	if h := g.r.Health(); !h.SelfPaused || h.Panics <= PanicBudget {
		t.Fatalf("health %+v", h)
	}
	g.r.mu.Lock()
	at := g.r.autoResumeAt
	g.r.mu.Unlock()
	if at.IsZero() {
		t.Fatal("the self pause has no auto resume")
	}
	g.clock.Sleep(time.Hour) // virtual time: the pause ends by itself
	g.waitFor("auto resume", func() bool { return !g.r.Health().SelfPaused })
}

func TestPanicWindowForgets(t *testing.T) {
	g, _ := panicRig(t, 0)
	for i := 0; i < PanicBudget; i++ {
		if g.r.notePanic("x", nil) {
			t.Fatalf("over budget at %d", i)
		}
	}
	g.clock.Sleep(PanicWindow + time.Minute)
	if g.r.notePanic("x", nil) {
		t.Fatal("old panics still counted")
	}
}

func TestPanicStormWithoutGoalEndsRun(t *testing.T) {
	g, _ := panicRig(t, 0)
	for i := 0; i <= PanicBudget; i++ {
		g.r.onPanic(context.Background(), "boom", nil)
	}
	g.r.mu.Lock()
	crash := g.r.crash
	g.r.mu.Unlock()
	if !errors.Is(crash, ErrPanicStorm) {
		t.Fatalf("crash = %v, want ErrPanicStorm (no goal: pausing cannot help)", crash)
	}
	if err := g.r.Run(context.Background()); !errors.Is(err, ErrPanicStorm) {
		t.Fatalf("Run returned %v", err)
	}
}

func TestPanicSecondOverrunEnds(t *testing.T) {
	g, _ := panicRig(t, 1000)
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("self pause", func() bool { return g.r.Status().State == goals.RunnerPaused })
	g.r.mu.Lock()
	first := g.r.crash
	g.r.mu.Unlock()
	if first != nil {
		t.Fatalf("first overrun must only pause, got %v", first)
	}
	g.r.mu.Lock()
	g.r.selfStop, g.r.paused = false, false // the auto resume
	g.r.mu.Unlock()
	g.waitFor("second overrun ends the run", func() bool {
		g.r.mu.Lock()
		defer g.r.mu.Unlock()
		return errors.Is(g.r.crash, ErrPanicStorm)
	})
}

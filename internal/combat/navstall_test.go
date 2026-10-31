package combat

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/input"
)

// emptyPathMover plans fine (PathTo) but every walk fails at once with the nav error seen live ("no path: empty
// path"), without letting game time pass. It cancels the run after a bounded number of walks so a spin ends the test.
type emptyPathMover struct {
	Mover
	cancel context.CancelFunc
	walks  int
	limit  int
}

func (m *emptyPathMover) GoTo(ctx context.Context, p input.Point) error {
	if m.walks++; m.walks >= m.limit {
		m.cancel()
	}
	return errors.New("nav: no path: empty path")
}

// Live (soakfull seed 11, runab seed 42): a crowd, life low, no portal and no potion: Safety asks for a crowd
// retreat, the step-away walk fails at once, the pin rule switches evading off for pinHold of lockstep time that never
// passes, and the fight loop spun forever without a tick. A step-away that cannot move must hand over to the
// normal fight (cast) so the game clock moves on.
func TestCrowdRetreatEmptyPathDoesNotSpin(t *testing.T) {
	h := newHarness(t)
	h.s.noTP = true
	h.s.life = 34
	h.s.healAmount = 0
	for i := 0; i < 10; i++ {
		h.s.addMon(uint32(10+i), clsFallen, 102+float64(i%3), 98+float64(i%4), 4000, 0, 0)
	}
	ctx, cancel := context.WithCancel(context.Background())
	mv := &emptyPathMover{Mover: h.s, cancel: cancel, limit: 500}
	h.f.mv = mv
	t0 := h.s.Now()
	_, _ = h.f.Clear(ctx, ClearOpts{Timeout: 20 * time.Second})
	if mv.walks >= mv.limit {
		t.Fatalf("spin: %d failed walks, game time moved %v", mv.walks, h.s.Now().Sub(t0))
	}
	if !h.s.Now().After(t0) {
		t.Fatal("game time never moved")
	}
}

// The step-away is not offered while the fighter is pinned (it would only fail again).
func TestCanRetreatOffWhilePinned(t *testing.T) {
	h := newHarness(t)
	for i := 0; i < 10; i++ {
		h.s.addMon(uint32(10+i), clsFallen, 105+float64(i%3), 98+float64(i%4), 4000, 0, 0)
	}
	snap, _ := h.s.snapshot()
	hs := Hostiles(snap, h.f.hostileCtx(snap.Player))
	if !h.f.canRetreat(snap, hs) {
		t.Fatal("a step away should exist when not pinned")
	}
	h.f.noEvadeUntil = h.s.Now().Add(pinHold)
	if h.f.canRetreat(snap, hs) {
		t.Fatal("pinned: no step away")
	}
}

// A walk that fails at once still spends game time (the general guard of the spin pattern: every failed move is a tick).
func TestFailedMoveAdvancesTime(t *testing.T) {
	h := newHarness(t)
	h.f.mv = &emptyPathMover{Mover: h.s, cancel: func() {}, limit: 1 << 30}
	t0 := h.s.Now()
	for i := 0; i < 10; i++ {
		if err := h.f.move(context.Background(), input.Point{X: 90, Y: 90}); err != nil {
			t.Fatal(err)
		}
	}
	if got := h.s.Now().Sub(t0); got < 10*ms(h.f.tune().PollMs) {
		t.Fatalf("10 failed walks advanced only %v", got)
	}
}

// frozenClock never advances: Sleep only counts and cancels the run after a few calls (a clock that is not stepped).
type frozenClock struct {
	t      time.Time
	sleeps int
	cancel context.CancelFunc
}

func (c *frozenClock) Now() time.Time { return c.t }
func (c *frozenClock) Sleep(time.Duration) {
	if c.sleeps++; c.sleeps >= 400 {
		c.cancel()
	}
}

// The no-progress watchdog: with a clock that never moves and walks that fail, the fight loop must reach a sleep on its
// own (bounded iterations between sleeps) and say so once.
func TestFightWatchdogLogs(t *testing.T) {
	var buf bytes.Buffer
	h := newHarness(t)
	h.s.noTP = true
	h.s.life = 34
	h.s.healAmount = 0
	for i := 0; i < 10; i++ {
		h.s.addMon(uint32(10+i), clsFallen, 102+float64(i%3), 98+float64(i%4), 4000, 0, 0)
	}
	ctx, cancel := context.WithCancel(context.Background())
	fc := &frozenClock{t: h.s.Now(), cancel: cancel}
	h.f.e.Clock = fc
	h.f.e.Log = slog.New(slog.NewTextHandler(&buf, nil))
	h.f.mv = &emptyPathMover{Mover: h.s, cancel: cancel, limit: 1 << 30}
	_, _ = h.f.Clear(ctx, ClearOpts{Timeout: time.Hour})
	if fc.sleeps < 400 {
		t.Fatalf("only %d sleeps: the loop spun", fc.sleeps)
	}
	if n := strings.Count(buf.String(), "combat no progress, holding"); n != 1 {
		t.Fatalf("watchdog log lines = %d, want 1", n)
	}
}

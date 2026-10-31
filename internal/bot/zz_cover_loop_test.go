package bot

import (
	"context"
	"d2lod-bot/internal/combat"
	"errors"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

func TestCoverWaitAreaTimesOutOnVirtualClock(t *testing.T) {
	g := newRig(t)
	g.r.poll()
	err := g.r.waitArea(context.Background(), 5, 3*time.Second)
	if err == nil || !strings.Contains(err.Error(), "did not arrive") {
		t.Fatalf("got %v", err)
	}
}

func TestCoverWaitAreaCancelled(t *testing.T) {
	g := newRig(t)
	cause := errors.New("stop requested")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	if err := g.r.waitArea(ctx, 5, time.Hour); !errors.Is(err, cause) {
		t.Fatalf("cause lost: %v", err)
	}
}

func TestCoverWaitAreaWaitsThroughLoadingScreen(t *testing.T) {
	g := newRig(t)
	g.w.set(func(w *world) { w.state = d2.StateMenu })
	calls := 0
	g.r.d.Snapshot = func() (*d2.Snapshot, error) {
		calls++
		if calls == 3 {
			g.w.set(func(w *world) { w.state = d2.StateInGame; w.area = 5 })
		}
		return g.w.snapshot()
	}
	if err := g.r.waitArea(context.Background(), 5, time.Minute); err != nil {
		t.Fatal(err)
	}
}

func TestCoverWaitAreaDeadPlayerDoesNotCountAsArrived(t *testing.T) {
	g := newRig(t)
	g.w.set(func(w *world) { w.area = 5; w.mode = combat.ModeDead })
	if err := g.r.waitArea(context.Background(), 5, 2*time.Second); err == nil {
		t.Fatal("a corpse in the target area is not an arrival")
	}
}

func TestCoverTravelToNoRouteAndCancel(t *testing.T) {
	g := newRig(t)
	g.r.poll()
	g.mover.routeFn = func(from, to int) ([]int, error) { return nil, errors.New("unknown level") }
	if err := g.r.travelTo(context.Background(), 6); err == nil || !strings.Contains(err.Error(), "no route") {
		t.Fatalf("got %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// no snapshot at all (area 0): the loop must still end through ctx
	g.w.set(func(w *world) { w.state = d2.StateMenu })
	g.r.obMu.Lock()
	g.r.ob.lastSnap = nil
	g.r.obMu.Unlock()
	done := make(chan error, 1)
	go func() { done <- g.r.travelTo(ctx, 6) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("travel without a game reported success")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("travelTo spins without a game")
	}
}

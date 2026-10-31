package input

import (
	"context"
	"errors"
	"testing"
	"time"
)

type testClock struct{ now time.Time }

func (c *testClock) Now() time.Time        { return c.now }
func (c *testClock) Sleep(d time.Duration) { c.now = c.now.Add(d) }

func TestGameUsesInjectedClockAndCtx(t *testing.T) {
	clk := &testClock{now: time.Unix(0, 0)}
	g := &Game{Clock: clk, T: Instant()}
	g.sleep(3 * time.Second)
	if clk.now != time.Unix(3, 0) {
		t.Fatalf("sleep did not use the injected clock: %v", clk.now)
	}
	ctx, cancel := context.WithCancel(context.Background())
	g.Ctx = ctx
	g.UIOpen = func() (bool, error) { return true, nil }
	cancel()
	if err := g.CloseAll(); !errors.Is(err, context.Canceled) {
		t.Fatalf("CloseAll after cancel: %v", err)
	}
	if err := g.MoveTo(Point{1, 1}, MoveOpts{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("MoveTo after cancel: %v", err)
	}
}

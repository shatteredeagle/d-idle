package combat

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

// The Clear interlude runs once per tick between survival and the next target, gets the nearest hostile's distance, and
// a true answer makes the loop look again; with no interlude (or Kill) nothing changes.
func TestClearInterludeSeesNearestHostile(t *testing.T) {
	h := newHarness(t)
	h.s.addMon(10, clsFallen, 112, 100, 40, 1, 2)
	var calls int
	var nearest float64
	_, _ = h.f.Clear(context.Background(), ClearOpts{Timeout: 5 * time.Second, Interlude: func(_ context.Context, s *d2.Snapshot, n float64, danger int) bool {
		if calls == 0 {
			nearest = n
		}
		calls++
		return false
	}})
	if calls == 0 {
		t.Fatal("interlude never called")
	}
	if nearest < 10 || nearest > 14 {
		t.Fatalf("nearest hostile %.1f, want about 12", nearest)
	}
}

// An interlude that acted (true) and spends a tick is called again; it must not starve the fight forever: here it acts
// three times and the monster still dies afterwards.
func TestClearInterludeActsThenFightContinues(t *testing.T) {
	h := newHarness(t)
	h.s.addMon(10, clsFallen, 112, 100, 40, 1, 2)
	acts := 0
	res, err := h.f.Clear(context.Background(), ClearOpts{Timeout: 20 * time.Second, Interlude: func(ctx context.Context, _ *d2.Snapshot, _ float64, _ int) bool {
		if acts >= 3 {
			return false
		}
		acts++
		h.s.Sleep(100 * time.Millisecond)
		return true
	}})
	if err != nil {
		t.Fatal(err)
	}
	if acts != 3 || res.Kills != 1 {
		t.Fatalf("acts %d kills %d", acts, res.Kills)
	}
}

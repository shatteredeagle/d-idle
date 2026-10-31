package bot

import (
	"context"
	"errors"
	"testing"

	"d2lod-bot/internal/goals"
)

// A waypoint trip to an area the reader says is not activated costs a panel round trip for nothing (V live soak
// 2026-10-29: about 20 s per try): it is skipped, the walk goes exit by exit.
func TestTravelSkipsWaypointKnownInactive(t *testing.T) {
	q := &fakeQuests{}
	q.v.WPs = map[int]bool{4: true}
	g := campaignRig(t, q, 12)
	g.w.set(func(w *world) { w.area = 4 })
	g.r.poll()
	if err := g.r.travelTo(context.Background(), 6); err != nil {
		t.Fatal(err)
	}
	if n := g.w.count("wp "); n != 0 {
		t.Fatalf("waypoint tried %d times: %v", n, g.w.log())
	}
	if g.w.count("exit 5") != 1 || g.w.count("exit 6") != 1 {
		t.Fatalf("calls: %v", g.w.log())
	}
}

func TestTravelUsesActivatedWaypointAndRemembersIt(t *testing.T) {
	q := &fakeQuests{}
	q.v.WPs = map[int]bool{4: true, 6: true}
	g := campaignRig(t, q, 12)
	g.w.set(func(w *world) { w.area = 4 })
	s := g.r.poll()
	if err := g.r.travelTo(context.Background(), 6); err != nil {
		t.Fatal(err)
	}
	if g.w.count("wp 6") != 1 || g.w.count("exit") != 0 {
		t.Fatalf("calls: %v", g.w.log())
	}
	if a, _ := g.r.activeWaypoints(s); !a[4] || !a[6] {
		t.Fatalf("active: %v", a)
	}
}

// Without a reader nothing is known: the trip is tried once, and a failure is not repeated.
func TestFailedWaypointTripIsNotRepeated(t *testing.T) {
	g := newRig(t)
	g.w.set(func(w *world) { w.area = 4 })
	g.mover.wpErr = errors.New("level 6 not reached")
	g.r.poll()
	for i := 0; i < 2; i++ {
		g.w.set(func(w *world) { w.area = 4 })
		g.r.poll()
		if err := g.r.travelTo(context.Background(), 6); err != nil {
			t.Fatal(err)
		}
	}
	if n := g.w.count("wp 6"); n != 1 {
		t.Fatalf("wp 6 tried %d times: %v", n, g.w.log())
	}
}

type toucherMover struct {
	*fakeMover
	touched bool
	calls   int
}

func (m *toucherMover) ActivateWaypointNear(ctx context.Context, d float64) (bool, error) {
	m.calls++
	return m.touched, nil
}

func TestTouchWaypointInPassingOnce(t *testing.T) {
	q := &fakeQuests{}
	q.v.WPs = map[int]bool{}
	g := campaignRig(t, q, 12)
	tm := &toucherMover{fakeMover: g.mover, touched: true}
	g.r.d.Mover = tm
	g.w.set(func(w *world) { w.area = 4 })
	g.r.poll()
	g.r.touchWaypointHere(context.Background())
	if tm.calls != 1 {
		t.Fatalf("calls %d", tm.calls)
	}
	g.r.touchWaypointHere(context.Background()) // known active now: no second look
	if tm.calls != 1 {
		t.Fatalf("calls %d after activation", tm.calls)
	}
	g.w.set(func(w *world) { w.area = 3 })
	g.r.poll()
	tm.touched = false
	g.r.touchWaypointHere(context.Background())
	g.r.touchWaypointHere(context.Background()) // rate limited
	if tm.calls != 2 {
		t.Fatalf("calls %d, want 2 (one per area)", tm.calls)
	}
}

// LevelTo walks to its target with the target hidden from the guard: in the clearing state the guard cancels every
// action that runs outside the published target area (LIVE soak 2026-10-29).
func TestLevelToTravelsWithoutGuardTarget(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 12)
	var seen []int
	g.r.d.Mover = &exitSpy{fakeMover: g.mover, r: g.r, seen: &seen}
	g.w.set(func(w *world) { w.area = 1 })
	g.r.poll()
	_ = (taskActs{r: g.r, c: control{goal: &goals.Goal{}}}).LevelTo(context.Background(), 14)
	if len(seen) == 0 {
		t.Fatalf("no exit taken: %v", g.w.log())
	}
	for _, v := range seen {
		if v != 0 {
			t.Fatalf("guard saw target %d during the walk", v)
		}
	}
}

type exitSpy struct {
	*fakeMover
	r    *Runner
	seen *[]int
}

func (m *exitSpy) ExitTo(ctx context.Context, l int) error {
	m.r.obMu.Lock()
	*m.seen = append(*m.seen, m.r.ob.targetArea)
	m.r.obMu.Unlock()
	return m.fakeMover.ExitTo(ctx, l)
}

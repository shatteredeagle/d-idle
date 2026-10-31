package nav

import (
	"context"
	"errors"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

// wpWorld: Rogue Encampment (1) with a waypoint object and Cold Plains (3). The panel opens after `needClicks`
// interactions (a lit waypoint: 1, a new one: 2, D2D-V) and a row click only travels once `ready` says so.
type wpWorld struct {
	s          *sim
	interacts  int
	needClicks int
	rowClicks  int
	arrived    time.Time // virtual time the character entered the level
	cooldown   time.Duration
}

func newWPWorld(t *testing.T, needClicks int) (*wpWorld, *Navigator) {
	g := openField(1000, 1000, 400, 100)
	l1 := &simLevel{no: 1, rect: d2map.Rect{X: 1000, Y: 1000, W: 200, H: 100}}
	l3 := &simLevel{no: 3, rect: d2map.Rect{X: 1200, Y: 1000, W: 200, H: 100}}
	s := newSim(g, input.Point{X: 1020, Y: 1050}, l1, l3)
	s.objects = []d2.Unit{d2Unit(11, 119, 1060, 1050)}
	w := &wpWorld{s: s, needClicks: needClicks, cooldown: 10 * time.Second, arrived: s.now}
	s.onInteract = func(s *sim, _ input.Target) error {
		w.interacts++
		if w.interacts >= w.needClicks {
			s.ui[0x14] = 1
		}
		return nil
	}
	s.onClick = func(s *sim, x, y int) {
		if y != wpPanel.RowY0+wpPanel.RowDY {
			return
		}
		w.rowClicks++
		if s.now.Sub(w.arrived) >= w.cooldown { // the game eats a click inside the cooldown
			s.px, s.py = 1300.5, 1050.5
			s.ui[0x14] = 0
		}
	}
	return w, s.navigator()
}

func TestWaypointWaitsOutCooldown(t *testing.T) {
	w, n := newWPWorld(t, 1)
	start := w.s.now
	if err := n.TakeWaypoint(ctxT(t), 3); err != nil {
		t.Fatal(err)
	}
	if got := w.s.now.Sub(start); got < 10*time.Second {
		t.Fatalf("clicked after %v, the cooldown is 10 s", got)
	}
	if w.rowClicks != 1 {
		t.Fatalf("row clicks = %d, want 1 (waited instead of clicking into the cooldown)", w.rowClicks)
	}
}

func TestWaypointSecondClickActivates(t *testing.T) {
	w, n := newWPWorld(t, 2)
	if err := n.TakeWaypoint(ctxT(t), 3); err != nil {
		t.Fatal(err)
	}
	if w.interacts != 2 || n.wpClicks < 3 {
		t.Fatalf("interacts %d, clicks %d: want 2 interactions then the row click", w.interacts, n.wpClicks)
	}
}

// A row click the game ate (our clock started late) is retried while the arrival is recent.
func TestWaypointRetriesEatenClick(t *testing.T) {
	w, n := newWPWorld(t, 1)
	w.cooldown = 22 * time.Second // the game's clock is ahead of the margin we assume
	if err := n.TakeWaypoint(ctxT(t), 3); err != nil {
		t.Fatal(err)
	}
	if w.rowClicks < 2 {
		t.Fatalf("row clicks = %d, want a retry round", w.rowClicks)
	}
}

func TestWaypointNoRetryForOldArrival(t *testing.T) {
	w, n := newWPWorld(t, 1)
	w.cooldown = time.Hour // an inactive waypoint: never travels
	n.wpLevel, n.wpSince = 1, w.s.now.Add(-time.Hour)
	w.s.now = w.s.now.Add(time.Minute)
	if err := n.TakeWaypoint(ctxT(t), 3); !errors.Is(err, ErrNoWaypoint) {
		t.Fatalf("err = %v", err)
	}
	if w.rowClicks > 2 {
		t.Fatalf("row clicks = %d: a long-standing character has no cooldown excuse, one round only", w.rowClicks)
	}
}

func TestWaypointCursorItem(t *testing.T) {
	w, n := newWPWorld(t, 1)
	w.s.cursor = &d2.Unit{ID: 5}
	if err := n.TakeWaypoint(ctxT(t), 3); !errors.Is(err, ErrCursorItem) {
		t.Fatalf("no hook: err = %v, want ErrCursorItem", err)
	}
	if w.interacts != 0 {
		t.Fatal("the waypoint was clicked with an item on the cursor")
	}
	cleared := 0
	n.ClearCursor = func(context.Context) error { cleared++; w.s.cursor = nil; return nil }
	if err := n.TakeWaypoint(ctxT(t), 3); err != nil || cleared != 1 {
		t.Fatalf("err %v cleared %d", err, cleared)
	}
}

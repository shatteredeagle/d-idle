package nav

import (
	"context"
	"errors"
	"testing"
	"time"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

// fkSim is a sim whose world hands out whole levels, like d2dworld.
type fkSim struct{ *sim }

func (fkSim) FullyKnown() bool { return true }

// townPocket models the soak state: the town (level 1) touches the Blood Moor (level 2) on its east edge, and the
// character stands in a small island of the known grid (a wall ring that the real game does not have).
func townPocket(physical bool) (*sim, *Navigator) {
	truth := openField(1000, 1000, 200, 100)
	known := openField(1000, 1000, 200, 100)
	ring := func(g *d2map.Grid) {
		for _, r := range []d2map.Rect{{X: 1030, Y: 1030, W: 14, H: 2}, {X: 1030, Y: 1048, W: 14, H: 2}, {X: 1030, Y: 1030, W: 2, H: 20}, {X: 1042, Y: 1030, W: 2, H: 20}} {
			blockRect(g, r)
		}
	}
	ring(known)
	if physical {
		ring(truth)
	}
	la := &simLevel{no: 1, rect: d2map.Rect{X: 1000, Y: 1000, W: 100, H: 100}}
	lb := &simLevel{no: 2, rect: d2map.Rect{X: 1100, Y: 1000, W: 100, H: 100}}
	s := newSim(truth, input.Point{X: 1036, Y: 1040}, la, lb)
	s.knownSrc = known
	n := s.navigator()
	n.w = fkSim{s}
	n.exploreFn = func(context.Context, func() bool, *input.Point) error { return ErrExplored }
	return s, n
}

// Before the fix the pocket ended in ErrNoExit after an explore that cannot find anything on a fully known map, and
// ToArea marked town->Blood Moor bad. Now the character is walked to the nearest connected cell and leaves.
func TestExitToLeavesPocketOfKnownGrid(t *testing.T) {
	s, n := townPocket(false)
	if err := n.ToArea(ctxT(t), 2); err != nil {
		t.Fatalf("ToArea: %v", err)
	}
	if s.cur().no != 2 {
		t.Fatalf("ended in level %d", s.cur().no)
	}
	if n.isBad(1, 2) {
		t.Fatal("edge 1->2 marked bad")
	}
}

// A character that really cannot get out gets a plain error, never ErrNoExit, and the edge the map shows stays good.
func TestPocketThatCannotBeLeftDoesNotCondemnTheEdge(t *testing.T) {
	_, n := townPocket(true)
	err := n.ToArea(ctxT(t), 2)
	if err == nil || errors.Is(err, ErrNoExit) {
		t.Fatalf("err = %v, want a non-ErrNoExit failure", err)
	}
	if n.isBad(1, 2) {
		t.Fatal("edge 1->2 marked bad")
	}
	if _, rerr := n.Route(1, 3); rerr != nil {
		t.Fatalf("route from the town is gone: %v", rerr)
	}
}

// The old behaviour behind the knob: the same pocket marks the link bad for good.
func TestPocketKnobOffKeepsOldBehaviour(t *testing.T) {
	s, n := townPocket(true)
	tuningOf(n).Nav.BadEdgeTTLSec = 0
	err := n.ToArea(ctxT(t), 2)
	if !errors.Is(err, ErrNoExit) {
		t.Fatalf("err = %v, want ErrNoExit", err)
	}
	if !n.isBad(1, 2) {
		t.Fatal("old behaviour: 1->2 should be marked bad")
	}
	s.Sleep(24 * time.Hour)
	if _, rerr := n.Route(1, 3); rerr == nil {
		t.Fatal("old behaviour: the mark never expires")
	}
}

// The soak wedge: with 1->2 marked bad, every route from the town fails. It recovers when the mark expires.
func TestBadEdgeExpires(t *testing.T) {
	s, n := townPocket(false)
	n.markBad(1, 2)
	if _, err := n.Route(1, 3); !errors.Is(err, ErrNoExit) {
		t.Fatalf("with 1->2 bad: err = %v, want no route", err)
	}
	s.Sleep(4 * time.Minute)
	if _, err := n.Route(1, 3); !errors.Is(err, ErrNoExit) {
		t.Fatal("expired too early")
	}
	s.Sleep(2 * time.Minute)
	r, err := n.Route(1, 3)
	if err != nil || len(r) != 2 || r[0] != 2 || r[1] != 3 {
		t.Fatalf("after the TTL: route %v err %v", r, err)
	}
}

// Re-entering the game replaces the level maps; a bad mark made in the old game is gone at once.
func TestBadEdgeClearedByGameReentry(t *testing.T) {
	s, n := townPocket(false)
	n.markBad(1, 2)
	n.markBadGate(1, 2, input.Point{X: 1099, Y: 1040})
	if !n.isBad(1, 2) {
		t.Fatal("mark not set")
	}
	s.levels[0].lm = &d2map.LevelMap{No: 1, Rect: s.levels[0].rect, Grid: s.levels[0].known}
	if n.isBad(1, 2) {
		t.Fatal("mark survived the re-entry")
	}
	if n.gateIsBad(1, 2, input.Point{X: 1099, Y: 1040}) {
		t.Fatal("bad gate survived the re-entry")
	}
}

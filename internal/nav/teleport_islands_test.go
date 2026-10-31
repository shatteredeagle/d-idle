package nav

import (
	"errors"
	"testing"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

// islandMap lays square islands over a void. Cells outside the rectangles are not in the truth grid at all, like
// the Arcane Sanctuary's void (no room data).
func islandMap(rects ...d2map.Rect) *d2map.Grid {
	g := d2map.NewGrid()
	for _, r := range rects {
		g.SetBlock(r, make([]uint16, r.W*r.H))
	}
	return g
}

func islandNav(t *testing.T, g *d2map.Grid, start input.Point, reveal int) (*sim, *Navigator) {
	t.Helper()
	s := newSim(g, start)
	s.reveal = reveal
	s.voidUnknown = true
	n := s.navigator()
	n.TeleportKey = "f2"
	return s, n
}

func TestTeleportToCrossesVoidGapsOverKnownIslands(t *testing.T) {
	// A -(12 void)- B -(12 void)- C -(12 void)- D going east; every island is revealed from the start.
	g := islandMap(
		d2map.Rect{X: 2000, Y: 2000, W: 30, H: 30}, d2map.Rect{X: 2042, Y: 2000, W: 30, H: 30},
		d2map.Rect{X: 2084, Y: 2000, W: 30, H: 30}, d2map.Rect{X: 2126, Y: 2000, W: 30, H: 30})
	s, n := islandNav(t, g, input.Point{X: 2010, Y: 2015}, 200)
	if err := n.TeleportTo(ctxT(t), 2145, 2015); err != nil {
		t.Fatal(err)
	}
	if d := dist(s.pos(), input.Point{X: 2145, Y: 2015}); d > 5 {
		t.Fatalf("ended %.0f away", d)
	}
	if s.travelled != 0 {
		t.Fatalf("walked %.0f", s.travelled)
	}
	for _, c := range s.casts {
		if !g.Walkable(c.X, c.Y) {
			t.Fatalf("blind cast %v although every island was known", c)
		}
	}
}

func TestTeleportToPlansAroundADeadEnd(t *testing.T) {
	// From A the island nearest the target (X, east) is a dead end; the way east leads through B (south) and C.
	g := islandMap(
		d2map.Rect{X: 2000, Y: 2000, W: 30, H: 30},
		d2map.Rect{X: 2044, Y: 2000, W: 20, H: 30}, // X: dead end, closest to the target
		d2map.Rect{X: 2000, Y: 2044, W: 30, H: 30}, // B
		d2map.Rect{X: 2040, Y: 2084, W: 60, H: 30}, // C
		d2map.Rect{X: 2112, Y: 2060, W: 30, H: 30}) // T
	s, n := islandNav(t, g, input.Point{X: 2015, Y: 2015}, 400)
	if err := n.TeleportTo(ctxT(t), 2127, 2075); err != nil {
		t.Fatal(err)
	}
	if d := dist(s.pos(), input.Point{X: 2127, Y: 2075}); d > 5 {
		t.Fatalf("ended %.0f away after %d casts %v", d, len(s.casts), s.casts)
	}
}

func TestTeleportToBlindCrossesUnknownGapsAndStreamsRooms(t *testing.T) {
	// Rooms load only within 22 subtiles, so the next island is unknown until a blind cast reaches it.
	g := islandMap(
		d2map.Rect{X: 2000, Y: 2000, W: 40, H: 40}, d2map.Rect{X: 2054, Y: 2000, W: 40, H: 40},
		d2map.Rect{X: 2108, Y: 2000, W: 40, H: 40}, d2map.Rect{X: 2162, Y: 2000, W: 40, H: 40})
	s, n := islandNav(t, g, input.Point{X: 2030, Y: 2020}, 22)
	if err := n.TeleportTo(ctxT(t), 2190, 2020); err != nil {
		t.Fatalf("%v; casts %v", err, s.casts)
	}
	if d := dist(s.pos(), input.Point{X: 2190, Y: 2020}); d > 5 {
		t.Fatalf("ended %.0f away", d)
	}
}

func TestTeleportToGivesUpWhenNothingLiesThere(t *testing.T) {
	g := islandMap(d2map.Rect{X: 2000, Y: 2000, W: 40, H: 40})
	s, n := islandNav(t, g, input.Point{X: 2020, Y: 2020}, 22)
	err := n.TeleportTo(ctxT(t), 2300, 2020)
	if !errors.Is(err, ErrNoPath) {
		t.Fatalf("err = %v", err)
	}
	if len(s.casts) > 60 {
		t.Fatalf("%d casts: the retry budget did not bound it", len(s.casts))
	}
}

func TestExploreTowardHopsBetweenIslands(t *testing.T) {
	g := islandMap(
		d2map.Rect{X: 2000, Y: 2000, W: 40, H: 40}, d2map.Rect{X: 2054, Y: 2000, W: 40, H: 40},
		d2map.Rect{X: 2108, Y: 2000, W: 40, H: 40}, d2map.Rect{X: 2162, Y: 2000, W: 40, H: 40})
	s, n := islandNav(t, g, input.Point{X: 2020, Y: 2020}, 22)
	tuningOf(n).Nav.ExploreTimeoutSec = 2000
	if err := n.ExploreToward(ctxT(t), input.Point{X: 2190, Y: 2020}, func() bool { return s.pos().X > 2170 }); err != nil {
		t.Fatalf("%v; casts %v", err, s.casts)
	}
}

package nav

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

// wallMaze: a 300x300 level split by a solid wall from y=2000 to y=2290 at x=2100; the only walking gap is at
// the bottom, so walking to the east side is a long detour while a hop crosses the wall.
func wallMaze() *d2map.Grid {
	g := openField(2000, 2000, 300, 300)
	blockRect(g, d2map.Rect{X: 2100, Y: 2000, W: 6, H: 290})
	return g
}

func TestTeleportExploreCrossesWallsAndLandsOnWalkableCells(t *testing.T) {
	g := wallMaze()
	s := newSim(g, input.Point{X: 2010, Y: 2010})
	n := s.navigator()
	n.TeleportKey = "f2"
	tuningOf(n).Nav.ExploreTimeoutSec = 2000
	target := d2map.Point{X: 2290, Y: 2010}
	done := func() bool { return s.Current().Grid.Known(target.X, target.Y) }
	if err := n.Explore(ctxT(t), done); err != nil {
		t.Fatal(err)
	}
	if len(s.casts) == 0 {
		t.Fatal("no teleport casts")
	}
	for _, c := range s.casts {
		if !g.Walkable(c.X, c.Y) {
			t.Fatalf("cast at non-walkable cell %v", c)
		}
	}
	if s.travelled > 150 {
		t.Fatalf("walked %.0f subtiles, hopping should do most of it", s.travelled)
	}
}

func TestTeleportExploreTowardHintHeadsThatWay(t *testing.T) {
	g := openField(2000, 2000, 400, 400)
	s := newSim(g, input.Point{X: 2200, Y: 2200})
	n := s.navigator()
	n.TeleportKey = "f2"
	tuningOf(n).Nav.ExploreTimeoutSec = 2000
	hint := input.Point{X: 2390, Y: 2200}
	done := func() bool { return s.pos().X > 2330 }
	if err := n.ExploreToward(ctxT(t), hint, done); err != nil {
		t.Fatal(err)
	}
	if len(s.casts) > 12 {
		t.Fatalf("%d casts to cross 130 subtiles toward the hint", len(s.casts))
	}
}

func TestTeleportFallsBackToWalkingWhenCastsDoNothing(t *testing.T) {
	g := openField(2000, 2000, 200, 200)
	s := newSim(g, input.Point{X: 2010, Y: 2010})
	s.noTeleport = true
	n := s.navigator()
	n.TeleportKey = "f2"
	tuningOf(n).Nav.ExploreTimeoutSec = 5000
	if err := n.Explore(ctxT(t), nil); err != ErrExplored {
		t.Fatalf("err = %v, want ErrExplored", err)
	}
	if len(s.casts) > 3*teleportBadHops+3 || len(s.clicks) == 0 {
		t.Fatalf("%d casts, %d walk clicks: the walk fallback did not take over", len(s.casts), len(s.clicks))
	}
	if got := s.Current().Grid.KnownCount(); got != 200*200 {
		t.Fatalf("known %d", got)
	}
}

func TestTeleportWaitsForMana(t *testing.T) {
	g := openField(2000, 2000, 400, 100)
	s := newSim(g, input.Point{X: 2010, Y: 2050})
	s.maxMana, s.mana, s.castCost, s.manaRegen = 100, 0, 24, 4
	n := s.navigator()
	n.TeleportKey = "f2"
	tuningOf(n).Nav.ExploreTimeoutSec = 2000
	start := s.now
	done := func() bool { return s.pos().X > 2060 }
	if err := n.ExploreToward(ctxT(t), input.Point{X: 2390, Y: 2050}, done); err != nil {
		t.Fatal(err)
	}
	if s.now.Sub(start) < 5*time.Second {
		t.Fatalf("took %v: the cast did not wait for mana", s.now.Sub(start))
	}
	for _, c := range s.casts {
		if c == (input.Point{}) {
			t.Fatal("bad cast")
		}
	}
}

func TestTeleportRefillHookIsCalledWhenDry(t *testing.T) {
	g := openField(2000, 2000, 400, 100)
	s := newSim(g, input.Point{X: 2010, Y: 2050})
	s.maxMana, s.mana, s.castCost = 100, 0, 24
	n := s.navigator()
	n.TeleportKey = "f2"
	calls := 0
	n.Refill = func(_ context.Context, mana, need int) error { calls++; s.mana = 100; return nil }
	done := func() bool { return s.pos().X > 2060 }
	if err := n.ExploreToward(ctxT(t), input.Point{X: 2390, Y: 2050}, done); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("Refill called %d times", calls)
	}
}

func TestTeleportTowardStopsNearTarget(t *testing.T) {
	g := wallMaze()
	s := newSim(g, input.Point{X: 2010, Y: 2010})
	n := s.navigator()
	n.TeleportKey = "f2"
	// the east side must be known to hop there: reveal it first
	s.px, s.py = 2200, 2010
	_, _ = s.Update()
	s.px, s.py = 2010, 2010
	_, _ = s.Update()
	if err := n.teleportToward(ctxT(t), input.Point{X: 2200, Y: 2010}, 10); err != nil {
		t.Fatal(err)
	}
	if d := dist(s.pos(), input.Point{X: 2200, Y: 2010}); d > 10 {
		t.Fatalf("ended %.0f away", d)
	}
}

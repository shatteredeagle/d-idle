package nav

import (
	"errors"
	"testing"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

// bigMaze: 400x400 level with a few walls and an opening pattern so exploring has to go around.
func bigLevel() *d2map.Grid {
	g := openField(2000, 2000, 400, 400)
	blockRect(g, d2map.Rect{X: 2100, Y: 2000, W: 6, H: 300})
	blockRect(g, d2map.Rect{X: 2250, Y: 2100, W: 6, H: 300})
	return g
}

func TestExploreRevealsLevelUntilDone(t *testing.T) {
	g := bigLevel()
	s := newSim(g, input.Point{X: 2010, Y: 2010})
	n := s.navigator()
	tuningOf(n).Nav.ExploreTimeoutSec = 2000
	tuningOf(n).Nav.WalkTimeoutSec = 600
	target := d2map.Point{X: 2390, Y: 2390}
	done := func() bool { return s.Current().Grid.Known(target.X, target.Y) }
	if err := n.Explore(ctxT(t), done); err != nil {
		t.Fatal(err)
	}
	if !done() {
		t.Fatal("done() never true")
	}
}

func TestExploreExhaustsAndReportsErrExplored(t *testing.T) {
	g := openField(2000, 2000, 200, 200)
	s := newSim(g, input.Point{X: 2010, Y: 2010})
	n := s.navigator()
	tuningOf(n).Nav.ExploreTimeoutSec = 5000
	err := n.Explore(ctxT(t), nil)
	if !errors.Is(err, ErrExplored) {
		t.Fatalf("err = %v, want ErrExplored", err)
	}
	if got := s.Current().Grid.KnownCount(); got != 200*200 {
		t.Fatalf("known %d of %d cells", got, 200*200)
	}
}

func TestExploreDeadEndIsNotRevisited(t *testing.T) {
	g := openField(2000, 2000, 200, 120)
	s := newSim(g, input.Point{X: 2010, Y: 2060})
	s.neverReveal = []d2map.Rect{{X: 2120, Y: 2000, W: 80, H: 120}} // the east part never loads
	n := s.navigator()
	tuningOf(n).Nav.ExploreTimeoutSec = 5000
	err := n.Explore(ctxT(t), nil)
	if !errors.Is(err, ErrExplored) {
		t.Fatalf("err = %v, want ErrExplored after the dead end", err)
	}
	if len(s.clicks) > 400 {
		t.Fatalf("%d clicks: the dead end was revisited", len(s.clicks))
	}
}

func TestExploreGreedyAlsoFinishes(t *testing.T) {
	g := bigLevel()
	var clicks [2]int
	for i, informed := range []bool{true, false} {
		s := newSim(g, input.Point{X: 2010, Y: 2010})
		n := s.navigator()
		tuningOf(n).Nav.ExploreInformed = informed
		tuningOf(n).Nav.ExploreTimeoutSec = 5000
		tuningOf(n).Nav.WalkTimeoutSec = 600
		if err := n.Explore(ctxT(t), nil); !errors.Is(err, ErrExplored) {
			t.Fatalf("informed=%v: %v", informed, err)
		}
		clicks[i] = len(s.clicks)
	}
	t.Logf("clicks to explore a 400x400 level: informed=%d nearest=%d", clicks[0], clicks[1])
}

package nav

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
)

func ctxT(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// walledField is a 200x120 field with a wall at x=100 that has a gap at y=80..89.
func walledField() *d2map.Grid {
	g := openField(1000, 1000, 200, 120)
	blockRect(g, d2map.Rect{X: 1100, Y: 1000, W: 4, H: 80})
	blockRect(g, d2map.Rect{X: 1100, Y: 1090, W: 4, H: 30})
	return g
}

func TestGoToOpenField(t *testing.T) {
	g := openField(1000, 1000, 200, 60)
	s := newSim(g, input.Point{X: 1010, Y: 1030})
	n := s.navigator()
	if err := n.GoTo(ctxT(t), input.Point{X: 1150, Y: 1030}); err != nil {
		t.Fatal(err)
	}
	if d := dist(s.pos(), input.Point{X: 1150, Y: 1030}); d > 2.5 {
		t.Fatalf("stopped %v away", d)
	}
	if len(s.clicks) < 10 || len(s.clicks) > 40 {
		t.Fatalf("%d clicks for 140 subtiles", len(s.clicks))
	}
	if s.travelled > 150 {
		t.Fatalf("walked %.0f for a 140 subtile straight line", s.travelled)
	}
}

func TestGoToAroundWallThroughGap(t *testing.T) {
	s := newSim(walledField(), input.Point{X: 1020, Y: 1020})
	n := s.navigator()
	if err := n.GoTo(ctxT(t), input.Point{X: 1180, Y: 1020}); err != nil {
		t.Fatal(err)
	}
	if dist(s.pos(), input.Point{X: 1180, Y: 1020}) > 2.5 {
		t.Fatalf("at %v", s.pos())
	}
}

func TestGoToDiscoversMapAsItWalks(t *testing.T) {
	// the goal is far outside what the sim has revealed at the start: planning is partial until rooms appear
	s := newSim(walledField(), input.Point{X: 1005, Y: 1005})
	n := s.navigator()
	if err := n.GoTo(ctxT(t), input.Point{X: 1190, Y: 1110}); err != nil {
		t.Fatal(err)
	}
	if dist(s.pos(), input.Point{X: 1190, Y: 1110}) > 2.5 {
		t.Fatalf("at %v", s.pos())
	}
}

func TestStuckOnGhostPillarRecovers(t *testing.T) {
	// the known grid says the pillar cells are walkable; only the truth blocks them (a monster, a door...)
	g := openField(1000, 1000, 200, 60)
	s := newSim(g, input.Point{X: 1010, Y: 1030})
	n := s.navigator()
	truth := g.Clone()
	blockRect(truth, d2map.Rect{X: 1080, Y: 1025, W: 4, H: 10})
	s.truth = truth
	// reveal the OPEN version to the navigator: ghost obstacle
	s.knownSrc = g
	if err := n.GoTo(ctxT(t), input.Point{X: 1150, Y: 1030}); err != nil {
		t.Fatal(err)
	}
	if dist(s.pos(), input.Point{X: 1150, Y: 1030}) > 2.5 {
		t.Fatalf("at %v", s.pos())
	}
}

func TestStuckGivesUp(t *testing.T) {
	g := openField(1000, 1000, 200, 60)
	s := newSim(g, input.Point{X: 1010, Y: 1030})
	n := s.navigator()
	truth := g.Clone()
	blockRect(truth, d2map.Rect{X: 1080, Y: 1000, W: 4, H: 60}) // a complete wall nobody knows about
	s.truth = truth
	s.knownSrc = g
	err := n.GoTo(ctxT(t), input.Point{X: 1150, Y: 1030})
	if !errors.Is(err, ErrStuck) {
		t.Fatalf("err = %v, want ErrStuck", err)
	}
}

func TestNoPathWhenWalledOff(t *testing.T) {
	g := openField(1000, 1000, 200, 60)
	blockRect(g, d2map.Rect{X: 1080, Y: 1000, W: 4, H: 60})
	s := newSim(g, input.Point{X: 1010, Y: 1030})
	n := s.navigator()
	err := n.GoTo(ctxT(t), input.Point{X: 1150, Y: 1030})
	if !errors.Is(err, ErrNoPath) {
		t.Fatalf("err = %v, want ErrNoPath", err)
	}
}

func TestWalksAroundStandingMonster(t *testing.T) {
	g := openField(1000, 1000, 200, 60)
	s := newSim(g, input.Point{X: 1010, Y: 1030})
	s.units = []d2.Unit{{ID: 5, Type: d2.TypeMonster, X: 1080, Y: 1030, Mode: d2.MonModeStand}}
	s.solidUnits = true
	n := s.navigator()
	minD := 1e9
	s.onStep = func(s *sim) { minD = math.Min(minD, math.Hypot(s.px-1080, s.py-1030)) }
	if err := n.GoTo(ctxT(t), input.Point{X: 1150, Y: 1030}); err != nil {
		t.Fatal(err)
	}
	if minD < 3 {
		t.Fatalf("came within %.1f of the monster", minD)
	}
}

func TestClosesPanelBeforeWalking(t *testing.T) {
	g := openField(1000, 1000, 100, 40)
	s := newSim(g, input.Point{X: 1010, Y: 1020})
	s.ui[d2.UIInventory] = 1
	n := s.navigator()
	if err := n.GoTo(ctxT(t), input.Point{X: 1060, Y: 1020}); err != nil {
		t.Fatal(err)
	}
	if s.closeAlls != 1 {
		t.Fatalf("CloseAll called %d times", s.closeAlls)
	}
}

func TestGuardAbortsWalk(t *testing.T) {
	g := openField(1000, 1000, 300, 40)
	s := newSim(g, input.Point{X: 1010, Y: 1020})
	n := s.navigator()
	errLow := errors.New("life low")
	n.Guard = func(sn *d2.Snapshot) error {
		if sn.Player.X > 1100 {
			return errLow
		}
		return nil
	}
	err := n.GoTo(ctxT(t), input.Point{X: 1290, Y: 1020})
	if !errors.Is(err, ErrAborted) || !errors.Is(err, errLow) {
		t.Fatalf("err = %v", err)
	}
}

func TestWalkWaitsThroughLoading(t *testing.T) {
	g := openField(1000, 1000, 100, 40)
	s := newSim(g, input.Point{X: 1010, Y: 1020})
	s.loadTicks = 5
	n := s.navigator()
	if err := n.GoTo(ctxT(t), input.Point{X: 1060, Y: 1020}); err != nil {
		t.Fatal(err)
	}
}

func TestGoToUnitFollowsMovingUnit(t *testing.T) {
	g := openField(1000, 1000, 300, 60)
	s := newSim(g, input.Point{X: 1010, Y: 1030})
	s.units = []d2.Unit{{ID: 9, Type: d2.TypeMonster, X: 1100, Y: 1030, Mode: d2.MonModeStand}}
	s.onStep = func(s *sim) {
		if len(s.units) > 0 && s.units[0].X < 1200 {
			s.units[0].X++ // creeps east
		}
	}
	n := s.navigator()
	if err := n.GoToUnit(ctxT(t), 9); err != nil {
		t.Fatal(err)
	}
	if d := dist(s.pos(), input.Point{X: s.units[0].X, Y: s.units[0].Y}); d > unitTolerance+1 {
		t.Fatalf("%v away from the unit", d)
	}
	s.units = nil
	if err := n.GoToUnit(ctxT(t), 9); !errors.Is(err, ErrUnitGone) {
		t.Fatalf("err = %v, want ErrUnitGone", err)
	}
}

func TestGoToBlockedGoalStopsNextToIt(t *testing.T) {
	g := openField(1000, 1000, 100, 40)
	blockRect(g, d2map.Rect{X: 1050, Y: 1018, W: 6, H: 6})
	s := newSim(g, input.Point{X: 1010, Y: 1020})
	n := s.navigator()
	if err := n.GoTo(ctxT(t), input.Point{X: 1052, Y: 1020}); err != nil {
		t.Fatal(err)
	}
	if d := dist(s.pos(), input.Point{X: 1052, Y: 1020}); d > 8 {
		t.Fatalf("%v from the blocked goal", d)
	}
}

func TestLowerQualityWalksLongerButArrives(t *testing.T) {
	var travelled []float64
	for _, q := range []float64{1, 0.6, 0.15} {
		sum := 0.0
		for seed := int64(1); seed <= 4; seed++ {
			g := randomMap(300, 300, 50, seed)
			// the random map's origin is 5000,5000; keep ends free
			blockClear(g, 5000, 5000, 12)
			blockClear(g, 5288, 5288, 12)
			s := newSim(g, input.Point{X: 5006, Y: 5006})
			n := s.navigator()
			tuningOf(n).Nav.PathQuality = q
			tuningOf(n).Nav.WalkTimeoutSec = 900
			if err := n.GoTo(ctxT(t), input.Point{X: 5290, Y: 5290}); err != nil {
				t.Fatalf("q=%v seed=%d: %v", q, seed, err)
			}
			sum += s.travelled
		}
		travelled = append(travelled, sum)
	}
	if !(travelled[0] < travelled[1] && travelled[1] < travelled[2]) {
		t.Fatalf("distance walked by quality 1/.6/.15 = %v: not increasing", travelled)
	}
	t.Logf("walked totals q1=%.0f q0.6=%.0f q0.15=%.0f", travelled[0], travelled[1], travelled[2])
}

func blockClear(g *d2map.Grid, x0, y0, r int) {
	g.SetBlock(d2map.Rect{X: x0, Y: y0, W: r, H: r}, make([]uint16, r*r))
}

var _ = env.DefaultTuning

func TestDeadIsAHardStop(t *testing.T) {
	g := openField(1000, 1000, 300, 40)
	s := newSim(g, input.Point{X: 1010, Y: 1020})
	n := s.navigator()
	s.onStep = func(s *sim) {
		if s.px > 1060 {
			s.mode, s.life = 17, 0
		}
	}
	err := n.GoTo(ctxT(t), input.Point{X: 1290, Y: 1020})
	if !errors.Is(err, ErrDead) {
		t.Fatalf("err = %v, want ErrDead", err)
	}
	clicks := len(s.clicks)
	if err := n.GoTo(ctxT(t), input.Point{X: 1290, Y: 1020}); !errors.Is(err, ErrDead) || len(s.clicks) != clicks {
		t.Fatalf("a dead character was clicked again (%v)", err)
	}
}

func TestCorpseApproachStopsNearMonsters(t *testing.T) {
	g := openField(1000, 1000, 300, 60)
	s := newSim(g, input.Point{X: 1010, Y: 1030})
	n := s.navigator()
	if err := n.CorpseApproach(ctxT(t), input.Point{X: 1200, Y: 1030}); err != nil {
		t.Fatal(err)
	}
	s2 := newSim(g, input.Point{X: 1010, Y: 1030})
	s2.units = []d2.Unit{{ID: 3, Class: 20, Type: d2.TypeMonster, X: 1190, Y: 1040, Mode: d2.MonModeStand}}
	err := s2.navigator().CorpseApproach(ctxT(t), input.Point{X: 1200, Y: 1030})
	if !errors.Is(err, ErrDanger) {
		t.Fatalf("err = %v, want ErrDanger", err)
	}
	if len(s2.clicks) != 0 {
		t.Fatal("walked toward a corpse guarded by a monster")
	}
}

func TestSpacingKnobChangesClickCountNotArrival(t *testing.T) {
	var counts []int
	for _, sp := range []float64{4, 8, 24} {
		s := newSim(walledField(), input.Point{X: 1020, Y: 1020})
		n := s.navigator()
		tuningOf(n).Nav.WaypointSpacing = sp
		if err := n.GoTo(ctxT(t), input.Point{X: 1180, Y: 1020}); err != nil {
			t.Fatalf("spacing %v: %v", sp, err)
		}
		counts = append(counts, len(s.clicks))
	}
	if !(counts[0] > counts[1] && counts[1] > counts[2]) {
		t.Fatalf("clicks by spacing 4/8/24 = %v", counts)
	}
}

func TestSlowReactionStillArrives(t *testing.T) {
	s := newSim(walledField(), input.Point{X: 1020, Y: 1020})
	n := s.navigator()
	tuningOf(n).Nav.PollMs = 400
	tuningOf(n).Nav.StuckMs = 4000
	tuningOf(n).Nav.ReplanMs = 8000
	if err := n.GoTo(ctxT(t), input.Point{X: 1180, Y: 1020}); err != nil {
		t.Fatal(err)
	}
}

func TestUseWorldReplacesSource(t *testing.T) {
	n := New(&env.Env{})
	if n.w != nil {
		t.Fatal("no world expected")
	}
	n.UseWorld(&d2map.Tracker{})
	if n.w == nil {
		t.Fatal("UseWorld did not set the source")
	}
}

package nav

import (
	"errors"
	"math"
	"testing"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

// Tower Cellar 5 walk (cards countess3b, countessnav). V (job 20261121-005917, docs/probes/gridwhole-2026-11-22.md): level 25 is
// 150 x 150 subtiles, whole on arrival; stairs up at (130,60), hero arrives at (125,76), the Countess stands at (21,70). The
// recorded cell grid of the level is NOT in the repo (docs/probes/countess-nav-fixture.md has the capture command), so these
// two maps are SYNTHETIC corridor mazes with the logged key points: a wall everywhere except a corridor that passes the stairs
// at about 13.5 subtiles (as the logged corridor does, mouthCoreR comment) and a short branch up to the stairs.
//
//	map A: corridor y 66..73 west of the arrival, the stairs branch goes north from x 126..134.
//	map B: the same corridor mirrored (y 106..113 below, stairs at (130,120)), the branch goes south.
type cellarMap struct {
	name          string
	corridor      d2map.Rect
	branch        d2map.Rect
	stairs, start input.Point
	goal          input.Point
	beyond        input.Point // a goal only reachable through the stairs branch
}

var cellarMaps = []cellarMap{
	{"A", d2map.Rect{X: 20, Y: 66, W: 116, H: 8}, d2map.Rect{X: 126, Y: 40, W: 9, H: 27},
		input.Point{X: 130, Y: 60}, input.Point{X: 125, Y: 72}, input.Point{X: 21, Y: 69}, input.Point{X: 130, Y: 45}},
	{"B", d2map.Rect{X: 20, Y: 106, W: 116, H: 8}, d2map.Rect{X: 126, Y: 113, W: 9, H: 27},
		input.Point{X: 130, Y: 120}, input.Point{X: 125, Y: 108}, input.Point{X: 21, Y: 111}, input.Point{X: 130, Y: 135}},
}

func (c cellarMap) build(t *testing.T) *sim {
	t.Helper()
	g := d2map.NewGrid()
	lvl := d2map.Rect{X: 0, Y: 0, W: 150, H: 150}
	cells := make([]uint16, lvl.W*lvl.H)
	for i := range cells {
		cells[i] = d2map.BlockWalk
	}
	g.SetBlock(lvl, cells)
	for _, r := range []d2map.Rect{c.corridor, c.branch} {
		g.SetBlock(r, make([]uint16, r.W*r.H))
	}
	l := &simLevel{no: 25, rect: lvl}
	s := newSim(g, c.start, l)
	s.reveal = 1000
	s.Update()
	s.reveal = 45
	l.lm.Exits = []d2map.Exit{{Dest: 24, Pos: d2map.Point{X: c.stairs.X, Y: c.stairs.Y}}}
	return s
}

func (c cellarMap) nav(s *sim) *Navigator {
	n := s.navigator()
	tuningOf(n).Nav.ExploreTimeoutSec = 5000
	tuningOf(n).Nav.WalkTimeoutSec = 600
	n.w = knownSim{s}
	n.stayLevel, n.arrival = 25, true
	return n
}

// The walk from the arrival toward the Countess preset is planned and sent, the hero reaches her and never enters the
// core of the stairs disc. Both maps.
func TestCellar5WalkToThePresetIsPlanned(t *testing.T) {
	for _, c := range cellarMaps {
		t.Run(c.name, func(t *testing.T) {
			s := c.build(t)
			n := c.nav(s)
			minD := math.MaxFloat64
			s.onStep = func(s *sim) { minD = math.Min(minD, dist(s.pos(), c.stairs)) }
			err := n.GoTo(ctxT(t), c.goal)
			if errors.Is(err, ErrMouthGate) {
				t.Fatalf("gate refused the walk: %v (clicks %v)", err, s.clicks)
			}
			if len(s.clicks) < 3 {
				t.Fatalf("the walk sent %d clicks", len(s.clicks))
			}
			t.Logf("clicks %v, closest %.1f to the stairs", s.clicks, minD)
			if err != nil || dist(s.pos(), c.goal) > 6 {
				t.Fatalf("GoTo %v, at %v, clicks %v", err, s.pos(), s.clicks)
			}
			if minD < mouthCoreR {
				t.Fatalf("hero came within %.1f of the stairs (core %v)", minD, mouthCoreR)
			}
		})
	}
}

// Must-fail pair: a goal whose only way runs through the stairs branch crosses the unwanted exit; the gate keeps
// refusing it (the hero never gets inside the core and the walk does not report success).
func TestCellar5WalkThroughTheStairsIsRefused(t *testing.T) {
	for _, c := range cellarMaps {
		t.Run(c.name, func(t *testing.T) {
			s := c.build(t)
			n := c.nav(s)
			minD := math.MaxFloat64
			s.onStep = func(s *sim) { minD = math.Min(minD, dist(s.pos(), c.stairs)) }
			err := n.GoTo(ctxT(t), c.beyond)
			if err == nil {
				t.Fatalf("walk through the stairs succeeded, at %v", s.pos())
			}
			if minD < mouthCoreR {
				t.Fatalf("hero came within %.1f of the stairs (core %v); err %v", minD, mouthCoreR, err)
			}
		})
	}
}

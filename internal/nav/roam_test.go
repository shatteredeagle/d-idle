package nav

import (
	"errors"
	"math"
	"testing"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

// knownSim is a sim whose whole level is known from the start, like a d2dworld level.
type knownSim struct{ *sim }

func (knownSim) FullyKnown() bool { return true }

func newKnown(t *testing.T, g *d2map.Grid, start input.Point, full bool) (*sim, *Navigator) {
	t.Helper()
	s := newSim(g, start)
	s.reveal = 1000
	s.Update()
	s.reveal = 45
	n := s.navigator()
	tuningOf(n).Nav.ExploreTimeoutSec = 5000
	tuningOf(n).Nav.WalkTimeoutSec = 600
	if full {
		n.w = knownSim{s}
	}
	return s, n
}

// The failing case that motivated roam: with every cell known, plain Explore says "explored" at once.
func TestFullyKnownMapWithoutRoamIsExploredAtOnce(t *testing.T) {
	_, n := newKnown(t, openField(2000, 2000, 120, 120), input.Point{X: 2010, Y: 2010}, false)
	if err := n.Explore(ctxT(t), nil); !errors.Is(err, ErrExplored) {
		t.Fatalf("err = %v, want ErrExplored (Game.exe path must stay as it was)", err)
	}
	if v, m := n.Coverage(); v != 0 || m != 0 {
		t.Fatalf("coverage %d/%d on the unknown-cell path", v, m)
	}
}

// A bot-like loop: explore until something hostile is near, kill it, go on until Explore reports ErrExplored.
func TestRoamVisitsCoverageAndFightsOnFullyKnownMap(t *testing.T) {
	g := openField(2000, 2000, 120, 120)
	blockRect(g, d2map.Rect{X: 2050, Y: 2000, W: 4, H: 90}) // a wall with a gap at the south
	s, n := newKnown(t, g, input.Point{X: 2010, Y: 2010}, true)
	mons := []input.Point{{X: 2100, Y: 2015}, {X: 2030, Y: 2100}, {X: 2110, Y: 2110}, {X: 2080, Y: 2060}}
	for i, m := range mons {
		s.units = append(s.units, d2Unit(uint32(i+1), 100, m.X, m.Y))
	}
	hostile := func() bool {
		for _, u := range s.units {
			if math.Hypot(float64(u.X)-s.px, float64(u.Y)-s.py) < 12 {
				return true
			}
		}
		return false
	}
	kills := 0
	for i := 0; i < 200; i++ {
		err := n.Explore(ctxT(t), hostile)
		if errors.Is(err, ErrExplored) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		for j := 0; j < len(s.units); {
			u := s.units[j]
			if math.Hypot(float64(u.X)-s.px, float64(u.Y)-s.py) < 12 {
				s.units = append(s.units[:j], s.units[j+1:]...)
				kills++
				continue
			}
			j++
		}
		if i == 199 {
			t.Fatal("roam did not terminate")
		}
	}
	v, m := n.Coverage()
	t.Logf("sweep: coverage %d/%d points, kills %d, clicks %d", v, m, kills, len(s.clicks))
	if m < 40 || v < m*9/10 {
		t.Fatalf("coverage %d/%d", v, m)
	}
	if kills != len(mons) || len(s.units) != 0 {
		t.Fatalf("kills %d of %d, %d left", kills, len(mons), len(s.units))
	}
}

// Points behind a wall that cannot be walked to are skipped, not retried forever.
func TestRoamSkipsUnreachableGround(t *testing.T) {
	g := openField(2000, 2000, 100, 60)
	blockRect(g, d2map.Rect{X: 2060, Y: 2000, W: 4, H: 60}) // seals the east half
	_, n := newKnown(t, g, input.Point{X: 2010, Y: 2030}, true)
	if err := n.Explore(ctxT(t), nil); !errors.Is(err, ErrExplored) {
		t.Fatalf("err = %v", err)
	}
	v, m := n.Coverage()
	if m == 0 || v != m {
		t.Fatalf("coverage %d/%d, want all reachable points", v, m)
	}
	if m > 12*9 { // 60x60 reachable ground at spacing 7, not the sealed half
		t.Fatalf("%d points: unreachable ground was sampled", m)
	}
}

func TestDropNearExits(t *testing.T) {
	pts := []input.Point{{X: 10, Y: 10}, {X: 40, Y: 10}, {X: 14, Y: 16}}
	got := dropNearExits(pts, []d2map.Exit{{Dest: 8, Pos: d2map.Point{X: 12, Y: 12}}})
	if len(got) != 1 || got[0].X != 40 {
		t.Fatalf("%v", got)
	}
	if len(dropNearExits(pts, nil)) != 3 {
		t.Fatal("no exits must keep all points")
	}
}

func TestMouthStepWalksAwayFromTheExit(t *testing.T) {
	_, n := newKnown(t, openField(2000, 2000, 120, 120), input.Point{X: 2050, Y: 2050}, true)
	o, err := n.observe()
	if err != nil {
		t.Fatal(err)
	}
	o.lm.Exits = []d2map.Exit{{Dest: 8, Pos: d2map.Point{X: 2048, Y: 2050}}}
	to, ok := n.mouthStep(o)
	if !ok || to.X <= 2060 {
		t.Fatalf("mouth step %v %v: want a point well east of the pos, away from the exit", to, ok)
	}
	n.wantExit = 8
	if _, ok := n.mouthStep(o); ok {
		t.Fatal("the exit we want is not a mouth to flee from")
	}
}

// A wall right behind the character (cave corridor): the straight line away is blocked, the flood finds a way that
// ends farther from the warp (KILLFIX: no step in a cave left the walk heading into the warp and the cave shuttled).
func TestMouthStepFindsAWayAroundAWall(t *testing.T) {
	g := openField(2000, 2000, 120, 120)
	blockRect(g, d2map.Rect{X: 2054, Y: 2030, W: 40, H: 40}) // solid block east of the character and the warp's line
	_, n := newKnown(t, g, input.Point{X: 2050, Y: 2050}, true)
	o, err := n.observe()
	if err != nil {
		t.Fatal(err)
	}
	o.lm.Exits = []d2map.Exit{{Dest: 8, Pos: d2map.Point{X: 2048, Y: 2050}}}
	to, ok := n.mouthStep(o)
	if !ok {
		t.Fatal("no step found")
	}
	if math.Hypot(float64(to.X-2048), float64(to.Y-2050)) < 2+8 {
		t.Fatalf("step %v is not away from the warp", to)
	}
}

func TestNearMouthOnlyWhenGuardedAndNotWanted(t *testing.T) {
	_, n := newKnown(t, openField(2000, 2000, 120, 120), input.Point{X: 2050, Y: 2050}, true)
	o, err := n.observe()
	if err != nil {
		t.Fatal(err)
	}
	o.lm.Exits = []d2map.Exit{{Dest: 8, Pos: d2map.Point{X: 2048, Y: 2050}}}
	p := input.Point{X: 2053, Y: 2050}
	if n.NearMouth(p) {
		t.Fatal("off by default")
	}
	n.SetMouthGuard(true)
	if !n.NearMouth(p) || n.NearMouth(input.Point{X: 2070, Y: 2050}) {
		t.Fatal("within mouthR only")
	}
	n.wantExit = 8
	if n.NearMouth(p) {
		t.Fatal("the wanted exit is not a mouth")
	}
}

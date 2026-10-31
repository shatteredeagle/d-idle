package nav

import (
	"errors"
	"math"
	"math/rand"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
)

// planOn runs the planner on a fully known grid with default tuning.
func planOn(t testing.TB, g *d2map.Grid, from, to input.Point, mod func(*env.NavTuning)) (planResult, error) {
	t.Helper()
	s := newSim(g, from)
	n := s.navigator()
	if mod != nil {
		mod(&tuningOf(n).Nav)
	}
	lm := &d2map.LevelMap{No: 2, Rect: g.Bounds(), Grid: g}
	return n.plan(lm, from, to, n.tune().ArriveTolerance, nil, 0)
}

func TestPlanOpenFieldIsStraight(t *testing.T) {
	g := openField(100, 100, 80, 80)
	res, err := planOn(t, g, input.Point{X: 110, Y: 110}, input.Point{X: 170, Y: 170}, nil)
	if err != nil || res.partial {
		t.Fatal(err, res.partial)
	}
	got := pathLength(input.Point{X: 110, Y: 110}, res.wps)
	want := math.Hypot(60, 60)
	if got > want*1.02 {
		t.Fatalf("length %.1f, straight line %.1f (%v)", got, want, res.wps)
	}
	if last := res.wps[len(res.wps)-1]; last != (input.Point{X: 170, Y: 170}) {
		t.Fatalf("last waypoint %v", last)
	}
}

func TestPlanWaypointSpacing(t *testing.T) {
	g := openField(0, 0, 200, 40)
	res, _ := planOn(t, g, input.Point{X: 10, Y: 20}, input.Point{X: 190, Y: 20}, nil)
	for i, p := range res.wps[:len(res.wps)-1] {
		prev := input.Point{X: 10, Y: 20}
		if i > 0 {
			prev = res.wps[i-1]
		}
		if d := dist(prev, p); math.Abs(d-8) > 1.5 {
			t.Fatalf("spacing %v between %v and %v", d, prev, p)
		}
	}
}

func TestPlanThroughGapNoCornerCutting(t *testing.T) {
	g := asciiTruth(0, 0,
		"..................",
		"..................",
		"########..########",
		"########..########",
		"..................",
		"..................")
	// diagonal squeeze: the two cells (8,2),(9,2) are the only gap, 2 wide
	res, err := planOn(t, g, input.Point{X: 1, Y: 0}, input.Point{X: 16, Y: 5}, func(t *env.NavTuning) { t.WallClearance = 0 })
	if err != nil || res.partial {
		t.Fatal(err, res.partial)
	}
	f := buildField(g, g.Bounds(), 0, 0)
	prev := input.Point{X: 1, Y: 0}
	for _, p := range res.wps {
		if !f.los(prev.X, prev.Y, p.X, p.Y, 0) {
			t.Fatalf("segment %v -> %v crosses a wall", prev, p)
		}
		prev = p
	}
}

func TestPlanDiagonalCornerIsNotCut(t *testing.T) {
	// two blocks touching only at a corner: a diagonal step between them is forbidden
	g := asciiTruth(0, 0,
		"......",
		"..#...",
		"...#..",
		"......")
	f := buildField(g, g.Bounds(), 0, 0)
	if f.los(1, 2, 3, 1, 0) {
		t.Fatal("line of sight cuts the corner between (2,1) and (3,2)")
	}
	var p Planner
	path, _ := p.Plan(f, 2, 2, 3, 1, 0, 1)
	for i := 1; i < len(path); i++ {
		ax, ay := f.pt(path[i-1])
		bx, by := f.pt(path[i])
		if ax != bx && ay != by && (!f.Walkable(ax, by) || !f.Walkable(bx, ay)) {
			t.Fatalf("diagonal %d,%d -> %d,%d cuts a corner", ax, ay, bx, by)
		}
	}
}

func TestPlanUnknownIsNotWalkableAndPartial(t *testing.T) {
	g := asciiTruth(0, 0,
		"......   ......",
		"......   ......",
		"......   ......")
	res, err := planOn(t, g, input.Point{X: 1, Y: 1}, input.Point{X: 13, Y: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.optimistic {
		t.Fatal("goal behind unknown cells is only reachable optimistically")
	}
	// PathTo is pure planning on known cells: it refuses
	s := newSim(g, input.Point{X: 1, Y: 1})
	s.knownSrc = g
	s.levels[0].known = g
	s.levels[0].lm.Grid = g
	if _, err := s.navigator().PathTo(input.Point{X: 13, Y: 1}); !errors.Is(err, ErrNoPath) {
		t.Fatalf("PathTo through unknown cells: %v", err)
	}
}

func TestPlanGoalOnWallSnaps(t *testing.T) {
	g := asciiTruth(0, 0,
		"..........",
		"..........",
		"....##....",
		"..........")
	res, err := planOn(t, g, input.Point{X: 0, Y: 0}, input.Point{X: 4, Y: 2}, nil)
	if err != nil || !res.snapped {
		t.Fatal(err, res.snapped)
	}
}

func TestPlanKeepsOffWalls(t *testing.T) {
	// 9-wide corridor, start and goal on the wall side: the path should move toward the middle
	rows := []string{}
	for y := 0; y < 11; y++ {
		switch {
		case y == 0 || y == 10:
			rows = append(rows, "####################################")
		default:
			rows = append(rows, "#..................................#")
		}
	}
	g := asciiTruth(0, 0, rows...)
	res, err := planOn(t, g, input.Point{X: 2, Y: 1}, input.Point{X: 33, Y: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	f := buildField(g, g.Bounds(), 3, 0.75)
	minC := 99.0
	for _, p := range res.wps[:len(res.wps)-1] {
		minC = math.Min(minC, f.Clearance(p.X, p.Y))
	}
	if minC < 1.5 {
		t.Fatalf("path hugs the wall, min clearance %.1f: %v", minC, res.wps)
	}
}

func TestPlanAvoidsMonster(t *testing.T) {
	g := openField(0, 0, 100, 60)
	s := newSim(g, input.Point{X: 5, Y: 30})
	n := s.navigator()
	lm := &d2map.LevelMap{No: 2, Rect: g.Bounds(), Grid: g}
	mon := []d2.Unit{{ID: 7, Type: d2.TypeMonster, X: 50, Y: 30, Mode: d2.MonModeStand}}
	res, err := n.plan(lm, input.Point{X: 5, Y: 30}, input.Point{X: 95, Y: 30}, 2, mon, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range res.wps {
		if math.Hypot(float64(p.X-50), float64(p.Y-30)) < 4 {
			t.Fatalf("waypoint %v is on top of the monster", p)
		}
	}
	res2, _ := n.plan(lm, input.Point{X: 5, Y: 30}, input.Point{X: 95, Y: 30}, 2, mon, 7)
	if len(res2.wps) == 0 {
		t.Fatal("ignored unit should not matter")
	}
}

// randomMap builds a w x h map with n random rectangular obstacles, keeping a margin free of them.
func randomMap(w, h, n int, seed int64) *d2map.Grid {
	rng := rand.New(rand.NewSource(seed))
	cells := make([]uint16, w*h)
	for i := 0; i < n; i++ {
		bw, bh := 5+rng.Intn(60), 5+rng.Intn(60)
		x0, y0 := rng.Intn(w-bw), rng.Intn(h-bh)
		for y := y0; y < y0+bh; y++ {
			for x := x0; x < x0+bw; x++ {
				cells[y*w+x] = d2map.BlockWalk
			}
		}
	}
	g := d2map.NewGrid()
	g.SetBlock(d2map.Rect{X: 5000, Y: 5000, W: w, H: h}, cells)
	return g
}

func TestPathQualityLengthensButReaches(t *testing.T) {
	var sum [3]float64
	qs := []float64{1, 0.5, 0.1}
	for seed := int64(1); seed <= 6; seed++ {
		g := randomMap(300, 300, 60, seed)
		from, to := input.Point{X: 5005, Y: 5005}, input.Point{X: 5295, Y: 5295}
		// make sure both ends are free and connected
		ok := true
		for i, q := range qs {
			res, err := planOn(t, g, from, to, func(t *env.NavTuning) { t.PathQuality = q })
			if err != nil || res.partial {
				ok = false
				break
			}
			if last := res.wps[len(res.wps)-1]; dist(last, to) > 3 {
				t.Fatalf("q=%v ends at %v, not %v", q, last, to)
			}
			sum[i] += pathLength(from, res.wps)
		}
		_ = ok
	}
	if !(sum[0] < sum[1] && sum[1] < sum[2]) {
		t.Fatalf("total path length by quality 1, .5, .1 = %.0f, %.0f, %.0f: not increasing", sum[0], sum[1], sum[2])
	}
	t.Logf("path length totals: q1=%.0f q0.5=%.0f q0.1=%.0f", sum[0], sum[1], sum[2])
}

func TestPlanLargeLevelSpeed(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	g := randomMap(1500, 1500, 500, 42)
	from, to := input.Point{X: 5003, Y: 5003}, input.Point{X: 6495, Y: 6495}
	s := newSim(g, from)
	n := s.navigator()
	lm := &d2map.LevelMap{No: 2, Rect: g.Bounds(), Grid: g}
	for i := 0; i < 2; i++ { // second run reuses the field
		start := time.Now()
		res, err := n.plan(lm, from, to, 2, nil, 0)
		el := time.Since(start)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("1500x1500 plan #%d: %v, partial=%v coarse=%v expanded=%d cells=%d wps=%d", i, el, res.partial, n.pl.Coarse, n.pl.Expanded, res.cells, len(res.wps))
		if el > 10*time.Second {
			t.Fatalf("too slow: %v", el)
		}
	}
}

func benchPlan(b *testing.B, coarse bool, seed int64, density int) {
	g := randomMap(1500, 1500, density, seed)
	from, to := input.Point{X: 5003, Y: 5003}, input.Point{X: 6495, Y: 6495}
	f := buildField(g, g.Bounds(), 3, 0.75)
	var p Planner
	sx, sy, _ := f.nearestWalkable(from.X, from.Y, 10)
	gx, gy, _ := f.nearestWalkable(to.X, to.Y, 10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if coarse {
			p.Plan(f, sx, sy, gx, gy, 1.6, 1)
		} else {
			p.search(f, sx, sy, gx, gy, 1.6, 1, nil)
		}
	}
	b.ReportMetric(float64(p.Expanded), "expanded")
}

func BenchmarkPlan1500Fine(b *testing.B)       { benchPlan(b, false, 42, 500) }
func BenchmarkPlan1500Coarse(b *testing.B)     { benchPlan(b, true, 42, 500) }
func BenchmarkPlan1500OpenFine(b *testing.B)   { benchPlan(b, false, 42, 0) }
func BenchmarkPlan1500OpenCoarse(b *testing.B) { benchPlan(b, true, 42, 0) }

func BenchmarkBuildField1500(b *testing.B) {
	g := randomMap(1500, 1500, 500, 42)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildField(g, g.Bounds(), 3, 0.75)
	}
}

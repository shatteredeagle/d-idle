package nav

import (
	"runtime"
	"testing"
	"time"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

// bogusLevelMap is a small known map whose level rectangle is absurd (a torn or uninitialised Level struct passes
// the per-side check of the reader). The goal sits in a walled pocket, so the planner searches everything it may.
func bogusLevelMap(side int) (*d2map.LevelMap, input.Point, input.Point) {
	rows := make([]string, 40)
	for y := range rows {
		b := make([]byte, 60)
		for x := range b {
			b[x] = '.'
			if y == 0 || y == 39 || x == 0 || x == 59 {
				b[x] = '#'
			}
			if x >= 40 && x <= 46 && y >= 10 && y <= 16 && (x == 40 || x == 46 || y == 10 || y == 16) {
				b[x] = '#' // closed pocket
			}
		}
		rows[y] = string(b)
	}
	g := d2map.ParseASCII(5000, 5000, rows...)
	lm := &d2map.LevelMap{No: 17, Rect: d2map.Rect{X: 5000 - side/2, Y: 5000 - side/2, W: side, H: side}, Grid: g}
	return lm, input.Point{X: 5010, Y: 5020}, input.Point{X: 5043, Y: 5013}
}

// TestPlanBogusLevelRectIsBounded is the OOM reproducer (G1 attempt 16: heap 28 GB in 30 s inside a PathTo). With
// a level rectangle of side*side subtiles the field and the planner scratch used to be allocated for all of it.
func TestPlanBogusLevelRectIsBounded(t *testing.T) {
	const side = 6000 // 36M cells: 13 bytes each unbounded, about 470 MB; a real level is under 2000 per side
	lm, from, to := bogusLevelMap(side)
	s := newSim(lm.Grid, from)
	n := s.navigator()
	var m0, m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m0)
	start := time.Now()
	_, err := n.plan(lm, from, to, 2, nil, 0)
	el := time.Since(start)
	runtime.ReadMemStats(&m1)
	f := n.fieldFor(lm)
	t.Logf("plan err=%v in %v, field %dx%d, allocated %d MB, sys %d MB", err, el, f.W, f.H, (m1.TotalAlloc-m0.TotalAlloc)>>20, m1.Sys>>20)
	if cells := f.W * f.H; cells > maxFieldCells {
		t.Fatalf("field has %d cells, cap %d", cells, maxFieldCells)
	}
	if alloc := (m1.TotalAlloc - m0.TotalAlloc) >> 20; alloc > 200 {
		t.Fatalf("a single plan allocated %d MB", alloc)
	}
}

func TestSearchAbsoluteCapReturnsPartial(t *testing.T) {
	old := maxExpandAbs
	maxExpandAbs = 50
	defer func() { maxExpandAbs = old }()
	g := randomMap(200, 200, 0, 1)
	f := buildField(g, g.Bounds(), 1, 0.5)
	var p Planner
	sx, sy, _ := f.nearestWalkable(5003, 5003, 5)
	path, partial := p.search(f, sx, sy, 5190, 5190, 0, 1, nil)
	if !p.Truncated || !partial || len(path) == 0 || p.Expanded > 51 {
		t.Fatalf("truncated=%v partial=%v len=%d expanded=%d", p.Truncated, partial, len(path), p.Expanded)
	}
}

func TestTraceStopsOnParentCycle(t *testing.T) {
	f := &Field{W: 3, H: 1}
	p := Planner{from: []uint8{1, 0, 1}} // 0: parent x-1... dirs[1] is (-1,0): cells 0 and 1 name each other
	if got := p.trace(f, 0); len(got) > 4 {
		t.Fatalf("trace did not stop: %d cells", len(got))
	}
}

package senses

import (
	"testing"
	"time"

	"d2lod-bot/internal/d2map"
)

var t0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func TestTrailSamplingAndStall(t *testing.T) {
	var tr Trail
	if !tr.Add(t0, 1, 10, 10) || tr.Add(t0.Add(100*time.Millisecond), 1, 11, 10) {
		t.Fatal("first sample stored, a sample inside the min gap dropped")
	}
	if tr.Add(t0.Add(time.Second), 1, 10, 10) {
		t.Fatal("unchanged position must not add a sample")
	}
	if !tr.Add(t0.Add(5*time.Second), 1, 12, 10) {
		t.Fatal("moved position adds a sample")
	}
	if got := tr.Area(t0.Add(5*time.Second), 1); len(got) != 2 || got[1].At.Sub(got[0].At) != 5*time.Second {
		t.Fatalf("a stall is the gap between samples: %+v", got)
	}
}

func TestTrailRetentionAndCap(t *testing.T) {
	var tr Trail
	for i := 0; i < 5; i++ {
		tr.Add(t0.Add(time.Duration(i)*time.Minute), 1, i, 0)
	}
	// samples at 0..4 min; at 6 min only those within 5 min remain (1..4)
	if got := tr.Area(t0.Add(6*time.Minute), 1); len(got) != 4 || got[0].X != 1 {
		t.Fatalf("retention: %+v", got)
	}
	var big Trail
	for i := 0; i < TrailCap*2; i++ {
		big.Add(t0.Add(time.Duration(i)*TrailMinGap), 1, i, 0)
	}
	if len(big.pts) != TrailCap || big.pts[len(big.pts)-1].X != TrailCap*2-1 {
		t.Fatalf("cap keeps the newest: %d", len(big.pts))
	}
}

func TestTrailPerAreaAndEntry(t *testing.T) {
	var tr Trail
	tr.Add(t0, 1, 1, 1)
	tr.Add(t0.Add(time.Second), 1, 2, 1)
	tr.Add(t0.Add(2*time.Second), 2, 50, 50)
	tr.Add(t0.Add(3*time.Second), 1, 3, 1) // back in area 1, same coordinates as before are not required
	now := t0.Add(4 * time.Second)
	a1, a2 := tr.Area(now, 1), tr.Area(now, 2)
	if len(a1) != 3 || len(a2) != 1 {
		t.Fatalf("areas %d %d", len(a1), len(a2))
	}
	if !a1[0].Entry || a1[1].Entry || !a1[2].Entry || !a2[0].Entry {
		t.Fatalf("entry marks: %+v", a1)
	}
	if len(tr.Area(now, 9)) != 0 {
		t.Fatal("unknown area has no trail")
	}
}

func TestPublisherRecordsTrail(t *testing.T) {
	clock := t0
	p := &Publisher{now: func() time.Time { return clock }}
	lm := &d2map.LevelMap{No: 1, Grid: d2map.ParseASCII(0, 0, "#####", "#...#", "#####")}
	for i, x := range []int{1, 2, 3} {
		clock = t0.Add(time.Duration(i) * time.Second)
		if err := p.Observe(lm, "T", d2map.Point{X: x, Y: 1}); err != nil {
			t.Fatal(err)
		}
	}
	pts, area, _, ok := p.TrailOf()
	if !ok || area != 1 || len(pts) != 3 {
		t.Fatalf("%v %d %+v", ok, area, pts)
	}
}

func TestCoverageOnlyForCurrentArea(t *testing.T) {
	p := &Publisher{now: func() time.Time { return t0 }}
	lm := &d2map.LevelMap{No: 1, Grid: d2map.ParseASCII(0, 0, "#####", "#...#", "#####")}
	if err := p.Observe(lm, "T", d2map.Point{X: 1, Y: 1}); err != nil {
		t.Fatal(err)
	}
	cells := make([][3]int, MaxCoverCells+5)
	p.SetCoverage(Coverage{Area: 1, Cells: cells})
	c, ok := p.CoverageOf()
	if !ok || len(c.Cells) != MaxCoverCells {
		t.Fatalf("cap: %v %d", ok, len(c.Cells))
	}
	p.SetCoverage(Coverage{Area: 2, Cells: cells})
	if _, ok := p.CoverageOf(); ok {
		t.Fatal("coverage of another area must not show")
	}
	p.SetCoverage(Coverage{})
	if _, ok := p.CoverageOf(); ok {
		t.Fatal("empty clears")
	}
}

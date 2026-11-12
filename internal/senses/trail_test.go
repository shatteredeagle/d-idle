package senses

import (
	"testing"
	"time"

	"d2lod-bot/internal/d2map"
)

var t0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func TestTrailSamplingAndStall(t *testing.T) {
	var tr Trail
	if !tr.Add(t0, 1, 10, 10, 0) || tr.Add(t0.Add(100*time.Millisecond), 1, 11, 10, 0) {
		t.Fatal("first sample stored, a sample inside the min gap dropped")
	}
	if tr.Add(t0.Add(time.Second), 1, 10, 10, 0) {
		t.Fatal("unchanged position must not add a sample")
	}
	if !tr.Add(t0.Add(5*time.Second), 1, 12, 10, 0) {
		t.Fatal("moved position adds a sample")
	}
	if got := tr.Area(t0.Add(5*time.Second), 1); len(got) != 2 || got[1].At.Sub(got[0].At) != 5*time.Second {
		t.Fatalf("a stall is the gap between samples: %+v", got)
	}
}

func TestTrailRetentionAndCap(t *testing.T) {
	var tr Trail
	for i := 0; i < 5; i++ {
		tr.Add(t0.Add(time.Duration(i)*time.Minute), 1, i, 0, 0)
	}
	// samples at 0..4 min; at 33 min only those within 30 min remain (3, 4)
	if got := tr.Area(t0.Add(33*time.Minute), 1); len(got) != 2 || got[0].X != 3 {
		t.Fatalf("retention: %+v", got)
	}
	var big Trail
	for i := 0; i < TrailCap+1000; i++ {
		big.Add(t0.Add(time.Duration(i)*TrailMinGap), 1, i, 0, 0)
	}
	if len(big.pts) > TrailCap || big.pts[len(big.pts)-1].X != TrailCap+999 || big.pts[0].Entry == false {
		t.Fatalf("cap thins the old half, keeps the newest and the entry: %d", len(big.pts))
	}
	// the newest half keeps every sample
	n := len(big.pts)
	if big.pts[n-1].X-big.pts[n-2].X != 1 {
		t.Fatal("newest samples are not thinned")
	}
}

func TestTrailPerAreaAndEntry(t *testing.T) {
	var tr Trail
	tr.Add(t0, 1, 1, 1, 0)
	tr.Add(t0.Add(time.Second), 1, 2, 1, 0)
	tr.Add(t0.Add(2*time.Second), 2, 50, 50, 0)
	tr.Add(t0.Add(3*time.Second), 1, 3, 1, 0) // back in area 1, same coordinates as before are not required
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
	p.SetCoverage(Coverage{Area: 1, Cell: 8, Cells: cells})
	c, ok := p.CoverageOf()
	if !ok || len(c.Cells) != MaxCoverCells || c.Total != len(cells) || c.Drawn != MaxCoverCells || c.Cell != 16 {
		t.Fatalf("cap: %v %d drawn %d total %d cell %d", ok, len(c.Cells), c.Drawn, c.Total, c.Cell)
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

// Regression (live Blood Moor, 1593 roam points sorted left to right): the grid must span the whole zone, and a
// downsample must keep the full extent instead of truncating to the west strip.
func TestCoverageSpansWholeZone(t *testing.T) {
	p := &Publisher{now: func() time.Time { return t0 }}
	lm := &d2map.LevelMap{No: 1, Grid: d2map.ParseASCII(0, 0, "#####", "#...#", "#####")}
	if err := p.Observe(lm, "T", d2map.Point{X: 1, Y: 1}); err != nil {
		t.Fatal(err)
	}
	build := func(n int) [][3]int { // a 40-column synthetic level, points sorted by x then y like nav.samplePoints
		var cs [][3]int
		for i := 0; i < n; i++ {
			cs = append(cs, [3]int{i / 40 * 7, i % 40 * 7, i % 3 % 2})
		}
		return cs
	}
	for _, n := range []int{1593, 5000} {
		cs := build(n)
		p.SetCoverage(Coverage{Area: 1, Cell: 7, Cells: cs})
		c, _ := p.CoverageOf()
		maxX := cs[len(cs)-1][0]
		got := 0
		for _, q := range c.Cells {
			got = max(got, q[0])
		}
		if got < maxX-14 || c.Total != n || c.Drawn != len(c.Cells) || c.Drawn > MaxCoverCells {
			t.Fatalf("n=%d: east edge %d of %d, %+v", n, got, maxX, c.Drawn)
		}
		if n == 1593 && (c.Drawn != n || c.Cell != 7) {
			t.Fatalf("a level under the cap is published whole: %d cell %d", c.Drawn, c.Cell)
		}
		seen := 0
		for _, q := range cs {
			if q[2] == CoverSeen {
				seen++
			}
		}
		if c.Seen != seen {
			t.Fatalf("seen counts all points: %d vs %d", c.Seen, seen)
		}
	}
}

func TestTrailActivityAndEvents(t *testing.T) {
	clock := t0
	p := &Publisher{now: func() time.Time { return clock }}
	lm := &d2map.LevelMap{No: 1, Grid: d2map.ParseASCII(0, 0, "#####", "#...#", "#####")}
	p.SetActivity(ActCombat)
	p.Observe(lm, "T", d2map.Point{X: 1, Y: 1})
	p.SetActivity(99) // out of range is ignored
	clock = t0.Add(time.Second)
	p.Observe(lm, "T", d2map.Point{X: 2, Y: 1})
	pts, _, _, _ := p.TrailOf()
	if len(pts) != 2 || pts[0].Act != ActCombat || pts[1].Act != ActCombat {
		t.Fatalf("%+v", pts)
	}
	p.AddEvent(EvKill, "Fallen killed", false, nil)
	clock = t0.Add(2 * time.Second)
	p.AddEvent(EvFight, "fight", false, &FightInfo{End: "clear", Kills: 1, Sec: 1.5})
	evs := p.EventsOf()
	if len(evs) != 2 || evs[0].X != 2 || evs[0].Fight == nil || evs[0].Fight.End != "clear" {
		t.Fatalf("a kill during a fight gets its summary: %+v", evs)
	}
}

func TestEventsCapAndRetention(t *testing.T) {
	var r Events
	for i := 0; i < EventCap+50; i++ {
		r.Add(Event{At: t0.Add(time.Duration(i) * time.Second), Area: 1, Kind: EvKill})
	}
	now := t0.Add(time.Duration(EventCap+50) * time.Second)
	if got := r.Area(now, 1); len(got) != EventCap {
		t.Fatalf("cap: %d", len(got))
	}
	r.Add(Event{At: now.Add(EventRetention + time.Minute), Area: 1, Kind: EvDeath})
	if got := r.Area(now.Add(EventRetention+time.Minute), 1); len(got) != 1 {
		t.Fatalf("retention: %d", len(got))
	}
}

func TestSamplesEveryAndCap(t *testing.T) {
	var r Samples
	if !r.Add(Sample{At: t0}) || r.Add(Sample{At: t0.Add(time.Second)}) || !r.Add(Sample{At: t0.Add(2 * time.Second)}) {
		t.Fatal("one sample per SampleEvery")
	}
	for i := 0; i < SampleCap+10; i++ {
		r.Add(Sample{At: t0.Add(time.Duration(10+i*2) * time.Second), Life: i})
	}
	if len(r.s) != SampleCap || r.s[len(r.s)-1].Life != SampleCap+9 {
		t.Fatalf("cap keeps the newest: %d", len(r.s))
	}
}

func TestHeatKillsDwellDamageAndReset(t *testing.T) {
	clock := t0
	p := &Publisher{now: func() time.Time { return clock }}
	grid := func() *d2map.LevelMap {
		return &d2map.LevelMap{No: 1, Grid: d2map.ParseASCII(0, 0, "#####", "#...#", "#####")}
	}
	lm := grid()
	for i := 0; i < 5; i++ { // 5 observations 500 ms apart on one cell
		clock = t0.Add(time.Duration(i) * 500 * time.Millisecond)
		p.Observe(lm, "T", d2map.Point{X: 2, Y: 1})
	}
	p.AddEvent(EvKill, "x killed", false, nil)
	p.AddDamage(20, "hurt")
	pts, area := p.HeatOf()
	if area != 1 || len(pts) != 1 || pts[0].Kills != 1 || pts[0].Dmg != 20 || pts[0].DwellS != 2 {
		t.Fatalf("%+v", pts)
	}
	if pts[0].X != 0 || pts[0].Y != 0 {
		t.Fatalf("cells are HeatCell-aligned: %+v", pts[0])
	}
	p.Observe(grid(), "T", d2map.Point{X: 2, Y: 1}) // a new level object is a new game
	if pts, _ := p.HeatOf(); len(pts) != 0 {
		t.Fatalf("heat of the area resets with its level object: %+v", pts)
	}
}

func TestHeatCap(t *testing.T) {
	var h Heat
	for i := 0; i < MaxHeatCells+50; i++ {
		h.cell(1, (i%100)*HeatCell, (i/100)*HeatCell).kills = 1
	}
	h.cell(1, 0, 0).kills = 99
	got := h.Area(1)
	if len(got) != MaxHeatCells || got[0].Kills != 99 {
		t.Fatalf("the busiest cells survive the cap: %d", len(got))
	}
}

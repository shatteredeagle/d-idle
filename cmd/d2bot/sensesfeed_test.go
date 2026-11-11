package main

import (
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/nav"
	"d2lod-bot/internal/senses"
	"d2lod-bot/internal/watch"
)

// strip is a level-relative corridor w subtiles long (a fake world, not a recording).
func strip(no, w int) *d2map.LevelMap {
	row := strings.Repeat(".", w)
	wall := strings.Repeat("#", w)
	// Origin is the act-absolute origin d2dworld sets on a level-relative map; it must NOT shift the panel's space.
	return &d2map.LevelMap{No: no, Origin: d2map.Point{X: 5000, Y: 5000}, Grid: d2map.ParseASCII(0, 0, wall, row, row, wall)}
}

func revealed(t *testing.T, m *senses.Map) int {
	t.Helper()
	cells, err := senses.Decode(m.RLE, m.W*m.H)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, c := range cells {
		if c != senses.Unknown {
			n++
		}
	}
	return n
}

func TestSensesFeedTwoAreas(t *testing.T) {
	pub := &senses.Publisher{}
	panel := &watch.Senses{Src: pub, Lock: watch.FixedTier(1)}
	clk := time.Unix(0, 0)
	now := func() time.Time { return clk }
	e := &env.Env{}
	useSenses(&ladderRig{Senses: pub}, e, nil, slog.Default())
	if e.OnObserve == nil {
		t.Fatal("useSenses did not set the hook")
	}
	hook := sensesFeed(pub, now, slog.Default())
	if r := panel.Reply(0); !r.Waiting {
		t.Fatalf("before the first observation: %+v", r)
	}
	a, b := strip(1, 120), strip(2, 60)

	step := func(lm *d2map.LevelMap, x int) watch.SensesReply {
		clk = clk.Add(time.Second)
		hook(lm, d2map.Point{X: x, Y: 1})
		return panel.Reply(0)
	}
	r1 := step(a, 6)
	if r1.Map == nil || r1.Hero == nil || r1.Hero.X != 6 || r1.Map.AreaID != 1 || r1.Map.X0 != 0 {
		t.Fatalf("first: %+v", r1)
	}
	n1 := revealed(t, r1.Map)

	// Same blocks, hero moves one subtile: the hero follows, the version does not change.
	r2 := step(a, 7)
	if r2.Version != r1.Version || r2.Hero.X != 7 {
		t.Fatalf("small move: version %d -> %d hero %+v", r1.Version, r2.Version, r2.Hero)
	}
	// Walking on reveals more of the same area: new version, more cells.
	r3 := step(a, 40)
	if r3.Version == r2.Version || revealed(t, r3.Map) <= n1 {
		t.Fatalf("reveal: version %d -> %d, cells %d -> %d", r2.Version, r3.Version, n1, revealed(t, r3.Map))
	}
	if x := r3.Map.W; x != 120 { // no Origin shift: the map spans the grid's own coordinates
		t.Fatalf("map width %d", x)
	}
	// Another area: own map and fog.
	r4 := step(b, 5)
	if r4.Map == nil || r4.Map.AreaID != 2 || r4.Version == r3.Version {
		t.Fatalf("second area: %+v", r4)
	}
	// Back: the first area keeps its fog (more cells than a fresh visit would show).
	r5 := step(a, 6)
	if r5.Map.AreaID != 1 || revealed(t, r5.Map) <= n1 {
		t.Fatalf("return: cells %d (fresh %d)", revealed(t, r5.Map), n1)
	}
	// A caller holding the current version gets no map.
	if r := panel.Reply(r5.Version); r.Map != nil || r.Hero == nil {
		t.Fatalf("cached: %+v", r)
	}
}

// The hook is throttled: unchanged area and grid inside sensesEvery are skipped, a level change is not.
func TestSensesFeedThrottle(t *testing.T) {
	pub := &senses.Publisher{}
	clk := time.Unix(0, 0)
	hook := sensesFeed(pub, func() time.Time { return clk }, slog.Default())
	a := strip(1, 80)
	hook(a, d2map.Point{X: 3, Y: 1})
	clk = clk.Add(sensesEvery / 5)
	hook(a, d2map.Point{X: 9, Y: 1})
	if s, _ := pub.Snapshot(); s.Hero.X != 3 {
		t.Fatalf("throttled call went through: %+v", s.Hero)
	}
	hook(strip(2, 80), d2map.Point{X: 4, Y: 1})
	if s, _ := pub.Snapshot(); s.Map.AreaID != 2 {
		t.Fatal("area change was throttled")
	}
}

// Producer on one goroutine (the nav goroutine), pollers on others: run with -race.
func TestSensesFeedRace(t *testing.T) {
	pub := &senses.Publisher{}
	panel := &watch.Senses{Src: pub, Lock: watch.FixedTier(1)}
	var clk time.Time
	hook := sensesFeed(pub, func() time.Time { return clk }, slog.Default())
	stop := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					panel.Reply(0)
				}
			}
		}()
	}
	lms := []*d2map.LevelMap{strip(1, 200), strip(2, 200)}
	for i := 0; i < 400; i++ {
		clk = clk.Add(time.Second)
		hook(lms[i/50%2], d2map.Point{X: i % 190, Y: 1})
	}
	close(stop)
	wg.Wait()
}

type fakeCover struct{ v nav.CoverageView }

func (f fakeCover) CoverageView() (nav.CoverageView, bool) { return f.v, len(f.v.Pts) > 0 }

func TestCoverageFor(t *testing.T) {
	c := coverageFor(fakeCover{nav.CoverageView{Level: 2, Spacing: 8, Cover: 5, Legs: 3, Held: 1,
		Pts:    []nav.CoverPoint{{X: 1, Y: 2, State: nav.CoverSeen}, {X: 9, Y: 2, State: nav.CoverFailed}, {X: 17, Y: 2, State: nav.CoverUnseen}},
		Target: &nav.CoverPoint{X: 17, Y: 2}}})
	if c.Area != 2 || c.Total != 3 || c.Cells[0] != [3]int{1, 2, senses.CoverSeen} || c.Cells[1][2] != senses.CoverFailed || c.Cells[2][2] != senses.CoverUnseen || c.Target.X != 17 || c.Legs != 3 {
		t.Fatalf("%+v", c)
	}
	if got := coverageFor(fakeCover{}); len(got.Cells) != 0 {
		t.Fatal("no roam, no cells")
	}
}

package d2dworld

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dctl/fake"
	"d2lod-bot/internal/d2dsrc"
	"d2lod-bot/internal/d2map"
)

// Compile-time check against nav's unexported interface (kept in sync by hand with internal/nav/nav.go).
var _ interface {
	Update() (d2map.State, error)
	Current() *d2map.LevelMap
	Level(no int) *d2map.LevelMap
} = (*World)(nil)

// stub is a Verbs with settable replies. All data is synthetic, in the formats of docs/d2dctl.md.
type stub struct {
	level               string // rest of "debug level"
	levelErr, gridErr   error
	rows                []string
	warps, presets, npc []string
	calls               map[string]int
}

func (s *stub) n(k string) { s.calls[k]++ }

func (s *stub) Level(context.Context) (d2dctl.LevelInfo, error) {
	s.n("level")
	if s.levelErr != nil {
		return d2dctl.LevelInfo{}, s.levelErr
	}
	var li d2dctl.LevelInfo
	_, err := fmt.Sscanf(s.level, "%d %g %g %d %d %d %d", &li.ID, &li.X, &li.Y, &li.W, &li.H, &li.OriginX, &li.OriginY)
	return li, err
}
func (s *stub) Grid(context.Context, int) ([]string, error) { s.n("grid"); return s.rows, s.gridErr }

func list[T any](lines []string, parse func(string) (T, error)) (d2dctl.List[T], error) {
	var l d2dctl.List[T]
	for _, x := range lines {
		v, err := parse(x)
		if err != nil {
			return l, err
		}
		l.Items = append(l.Items, v)
	}
	return l, nil
}
func (s *stub) Warps(context.Context) (d2dctl.List[d2dctl.Warp], error) {
	return list(s.warps, d2dctl.ParseWarp)
}
func (s *stub) Presets(context.Context) (d2dctl.List[d2dctl.Preset], error) {
	return list(s.presets, d2dctl.ParsePreset)
}
func (s *stub) NPCs(context.Context) (d2dctl.List[d2dctl.NPC], error) {
	return list(s.npc, d2dctl.ParseNPC)
}

// A 3x2-cell level (15x10 subtiles): a wall ring with a gap at the east side and a 2x2 pillar.
var synth = []string{
	"###############",
	"#.............#",
	"#....##.......#",
	"#....##........",
	"#.............#",
	"###############",
}

func newStub() *stub {
	return &stub{level: "1 1.0 0.6 3 2 10 20", rows: synth, calls: map[string]int{},
		warps:   []string{"0\t2.8\t0.6\t2"},
		presets: []string{"0\t119\t0.4\t0.4\t0\tWaypoint", "1\t12\t1.0\t0.2\t0\tWooden Door"},
		npc:     []string{"Akara 100 100 1 3 1.0 0.8"}}
}

func TestGridMatchesSyntheticMap(t *testing.T) {
	s := newStub()
	// 3x2 cells need 15x10 subtiles: stack the map twice and cut it.
	s.rows = append(append([]string(nil), synth...), synth...)[:10]
	w := New(s, Options{})
	st, err := w.Update()
	if err != nil {
		t.Fatal(err)
	}
	if !st.LevelChanged || st.LevelNo != 1 || st.Act != d2dsrc.ActOf(1) || st.Pos != (d2map.Point{X: 5, Y: 3}) {
		t.Fatalf("state %+v", st)
	}
	lm := w.Current()
	if lm.Rect != (d2map.Rect{X: 0, Y: 0, W: 15, H: 10}) {
		t.Fatalf("rect %+v", lm.Rect)
	}
	for y, row := range s.rows {
		for x, ch := range row {
			if got := lm.Grid.Walkable(x, y); got != (ch == '.') || !lm.Grid.Known(x, y) {
				t.Fatalf("cell %d,%d: %q walkable=%v known=%v", x, y, ch, got, lm.Grid.Known(x, y))
			}
		}
	}
	if lm.Grid.Known(15, 0) || lm.Grid.Walkable(-1, 3) {
		t.Fatal("outside the map must be unknown and not walkable")
	}
}

func TestExitsPresetsNPCs(t *testing.T) {
	s := newStub()
	w := New(s, Options{})
	if _, err := w.Update(); err != nil {
		t.Fatal(err)
	}
	lm := w.Current()
	if len(lm.Exits) != 1 || lm.Exits[0].Dest != 2 || lm.Exits[0].Pos != (d2map.Point{X: 14, Y: 3}) {
		t.Fatalf("exits %+v", lm.Exits)
	}
	if wp := w.Waypoints(1); len(wp) != 1 || wp[0] != (d2map.Point{X: 2, Y: 2}) {
		t.Fatalf("waypoints %v", wp)
	}
	if got := lm.Waypoints(); len(got) != 1 {
		t.Fatalf("LevelMap.Waypoints %v", got)
	}
	if d := w.Doors(1); len(d) != 1 || d[0] != (d2map.Point{X: 5, Y: 1}) {
		t.Fatalf("doors %v", d)
	}
	n, err := w.NPCs(context.Background())
	if err != nil || len(n) != 1 || n[0].Name != "Akara" || n[0].Index != 3 || n[0].Pos != (d2map.Point{X: 5, Y: 4}) {
		t.Fatalf("npcs %+v %v", n, err)
	}
}

func TestOriginModes(t *testing.T) {
	cases := []struct {
		mode OriginMode
		want d2map.Point
	}{{OriginNone, d2map.Point{X: 14, Y: 3}}, {OriginTiles, d2map.Point{X: 64, Y: 103}}, {OriginSubtiles, d2map.Point{X: 24, Y: 23}}}
	for _, c := range cases {
		w := New(newStub(), Options{Origin: c.mode})
		st, _ := w.Update()
		if e := w.Current().Exits[0].Pos; e != c.want {
			t.Errorf("mode %d exit %v want %v", c.mode, e, c.want)
		}
		if !w.Current().Rect.Contains(w.Current().Exits[0].Pos.X-1, w.Current().Exits[0].Pos.Y) {
			t.Errorf("mode %d: exit outside rect %+v", c.mode, w.Current().Rect)
		}
		ox, _ := originSub(c.mode, 10, 20)
		if st.Pos.X != ox+5 {
			t.Errorf("mode %d pos %v", c.mode, st.Pos)
		}
	}
}

func TestLevelChangeInvalidates(t *testing.T) {
	s := newStub()
	clock := time.Unix(0, 0)
	w := New(s, Options{Now: func() time.Time { return clock }})
	w.Update()
	first := w.Current()
	if s.calls["grid"] != 1 {
		t.Fatalf("fetches %d", s.calls["grid"])
	}
	// same level, player barely moved: no refetch
	w.Update()
	if s.calls["grid"] != 1 {
		t.Fatal("refetched without a reason")
	}
	// another level: fetched at once, old map stays reachable through Level
	s.level = "2 0.4 0.4 3 2 0 0"
	s.rows = []string{"..........", "..........", "..........", "..........", "..........", "###", "", "", "", ""}
	st, err := w.Update()
	if err != nil || !st.LevelChanged || st.LevelNo != 2 || s.calls["grid"] != 2 {
		t.Fatalf("st %+v err %v fetches %d", st, err, s.calls["grid"])
	}
	if w.Level(1) != first || w.Current() == first || w.Level(2) != w.Current() {
		t.Fatal("level cache wrong")
	}
	// back to level 1: stale data is refreshed, the same LevelMap (and grid pointer) is reused
	s.level = "1 1.0 0.6 3 2 10 20"
	s.rows = synth
	st, _ = w.Update()
	if !st.LevelChanged || s.calls["grid"] != 3 || w.Current() != first {
		t.Fatalf("return visit: %+v fetches %d", st, s.calls["grid"])
	}
	w.Reset()
	if w.Current() != nil || w.Level(1) != nil {
		t.Fatal("Reset keeps data")
	}
}

func TestRefetchOnMovementAndRate(t *testing.T) {
	s := newStub()
	clock := time.Unix(0, 0)
	w := New(s, Options{RefetchDist: 10, Now: func() time.Time { return clock }})
	w.Update()
	s.level = "1 2.8 0.6 3 2 10 20" // 9 subtiles east
	w.Update()
	if s.calls["grid"] != 1 {
		t.Fatal("moved less than the threshold")
	}
	s.level = "1 2.8 1.6 3 2 10 20" // 14 subtiles away
	w.Update()
	if s.calls["grid"] != 1 {
		t.Fatal("rate limit ignored")
	}
	clock = clock.Add(3 * time.Second)
	w.Update()
	if s.calls["grid"] != 2 {
		t.Fatal("no refetch after moving far and waiting")
	}
	if _, err := w.Refresh(); err != nil || s.calls["grid"] != 3 {
		t.Fatal("Refresh must refetch")
	}
}

func TestGridReplacedOnlyOnChange(t *testing.T) {
	s := newStub()
	clock := time.Unix(0, 0)
	w := New(s, Options{Now: func() time.Time { return clock }})
	w.Update()
	g := w.Current().Grid
	clock = clock.Add(time.Hour)
	w.Refresh()
	if w.Current().Grid != g {
		t.Fatal("identical refetch replaced the grid")
	}
	s.rows = append([]string(nil), synth...)
	s.rows[1] = "#######.......#" // a door opened or a room relaid
	if _, err := w.Refresh(); err != nil {
		t.Fatal(err)
	}
	ng := w.Current().Grid
	// nav keys its field cache on the pointer and KnownCount: a changed grid must be a new pointer
	if ng == g || ng.Walkable(2, 1) || !ng.Walkable(8, 1) || !g.Walkable(2, 1) || w.Level(1).Grid != ng {
		t.Fatal("changed refetch must hand out a new grid")
	}
}

func TestUnknownCellsAreNeverOpen(t *testing.T) {
	s := newStub()
	s.rows = []string{"..??#", "?.. #", "ab..."}
	w := New(s, Options{})
	w.Update()
	g := w.Current().Grid
	for _, p := range [][2]int{{2, 0}, {3, 0}, {0, 1}, {3, 1}, {0, 2}, {1, 2}, {20, 20}} {
		if g.Known(p[0], p[1]) || g.Walkable(p[0], p[1]) {
			t.Errorf("%v must be unknown and blocked", p)
		}
	}
	if !g.Walkable(0, 0) || !g.Known(4, 0) || g.Walkable(4, 0) {
		t.Fatal("known cells wrong")
	}
}

func TestGridCap(t *testing.T) {
	s := newStub()
	w := New(s, Options{MaxCells: 50}) // synth is 15x6
	w.Update()
	if w.Current().Grid.KnownCount() != 0 || !strings.Contains(w.LastGridNote, "cap") {
		t.Fatalf("cap ignored: known %d note %q", w.Current().Grid.KnownCount(), w.LastGridNote)
	}
	// a hostile row count must not be merged (and must not allocate) either
	s.rows = make([]string, 5000)
	for i := range s.rows {
		s.rows[i] = strings.Repeat(".", 5000)
	}
	w = New(s, Options{})
	w.Update()
	if w.Current().Grid.KnownCount() != 0 {
		t.Fatal("25M cells accepted over the 4096*4096 cap")
	}
}

func TestErrorsDegrade(t *testing.T) {
	s := newStub()
	s.levelErr = &d2dctl.CmdError{Msg: "not in game"}
	w := New(s, Options{})
	if _, err := w.Update(); !errors.Is(err, d2map.ErrNotInGame) || w.Current() != nil {
		t.Fatalf("level err: %v", err)
	}
	s.levelErr = nil
	s.level = "garbage"
	if _, err := w.Update(); !errors.Is(err, d2map.ErrNotInGame) {
		t.Fatalf("bad level reply: %v", err)
	}
	s.level = "1 1.0 0.6 0 0 0 0"
	if _, err := w.Update(); !errors.Is(err, d2map.ErrNotInGame) {
		t.Fatalf("zero-size level: %v", err)
	}
	s.level = "1 1.0 0.6 3 2 0 0"
	s.gridErr = &d2dctl.CmdError{Msg: "no grid"}
	st, err := w.Update()
	if err != nil || st.LevelNo != 1 || w.Current().Grid.KnownCount() != 0 || w.Current().Grid.Walkable(3, 3) {
		t.Fatalf("grid err must leave an empty grid, got %v known %d", err, w.Current().Grid.KnownCount())
	}
	s.gridErr = nil
	s.rows = nil
	s.warps = []string{"bad"}
	s.presets = []string{"x"}
	if _, err := w.Refresh(); err != nil || w.Current().Grid.KnownCount() != 0 {
		t.Fatalf("empty reply: %v", err)
	}
	if n, err := New(s, Options{}).NPCs(context.Background()); err == nil || n != nil {
		t.Fatal("NPCs before any Update must fail")
	}
}

// The fake server speaks the real wire format: a full round trip through d2dctl.Client.
func TestFakeServer(t *testing.T) {
	path, cleanup, err := fake.TempSocket()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	fix := fake.Sample()
	fix.Level = "1 1.0 0.6 3 2 10 20"
	fix.Grid = synth[:5]
	fix.Warps = []string{"0\t2.8\t0.6\t2"}
	srv, err := fake.New(path, fix, fake.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	c := d2dctl.New(path, d2dctl.Options{Timeout: 2 * time.Second})
	defer c.Close()
	w := New(Client{Client: c}, Options{}) // the fake has no "debug layout": borders degrade to none
	if _, err := w.Update(); err != nil {
		t.Fatal(err)
	}
	lm := w.Current()
	if len(lm.Adjacent) != 0 || !strings.Contains(w.LastBorderNote, "no layout") {
		t.Fatalf("adjacent %v note %q", lm.Adjacent, w.LastBorderNote)
	}
	if lm.Grid.KnownCount() != 15*5 || !lm.Grid.Walkable(1, 1) || lm.Grid.Walkable(0, 0) || lm.Grid.Known(0, 5) {
		t.Fatalf("grid known %d", lm.Grid.KnownCount())
	}
	if len(lm.Exits) != 1 || lm.Exits[0].Dest != 2 {
		t.Fatalf("exits %+v", lm.Exits)
	}
	if _, err := w.NPCs(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(srv.Lines(), "|"), "debug grid 5") {
		t.Fatalf("lines %v", srv.Lines())
	}
}

func TestASCII(t *testing.T) {
	w := New(newStub(), Options{})
	w.Update()
	out := ASCII(w.Current(), 100, 100)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 10 || len(lines[0]) != 15 || lines[3][14] != 'E' || lines[2][2] != 'W' || lines[0][0] != '#' || lines[1][1] != '.' {
		t.Fatalf("\n%s", out)
	}
	if small := ASCII(w.Current(), 5, 5); strings.Count(small, "\n") > 5 {
		t.Fatalf("not scaled:\n%s", small)
	}
}

// border stub: level 1 at tile (10,20) 3x2 tiles; level 2 abuts on the east, level 3 below, level 4 is far away.
type borderStub struct {
	*stub
	places  []Placement
	near    []Near
	openRow func(cy float64) bool // collision of the east neighbour, by cell y
	probes  int
}

func (b *borderStub) Layout(context.Context) ([]Placement, error) { return b.places, nil }
func (b *borderStub) Nearby(context.Context) ([]Near, error)      { return b.near, nil }
func (b *borderStub) Blocked(_ context.Context, x, y float64) (bool, error) {
	b.probes++
	if x >= 3 { // east of the level
		return !b.openRow(y), nil
	}
	return y < 2, nil // south neighbour: open past the bottom edge only from x in [0,3)
}

func newBorderStub() *borderStub {
	s := newStub()
	s.level = "1 1.0 0.6 3 2 10 20"
	rows := make([]string, 10)
	for y := range rows {
		rows[y] = "#" + strings.Repeat(".", 13) + "."
		if y < 5 {
			rows[y] = "#" + strings.Repeat(".", 13) + "#" // east edge open only on rows 5..9
		}
	}
	s.rows = rows
	return &borderStub{stub: s,
		places:  []Placement{{1, 10, 20, 3, 2}, {2, 13, 20, 3, 2}, {3, 10, 22, 3, 2}, {4, 40, 40, 3, 2}},
		near:    []Near{{2, 3, 0}},
		openRow: func(cy float64) bool { return cy >= 1.0 }} // rows 5..9 open across
}

func TestAdjacentFromLayout(t *testing.T) {
	b := newBorderStub()
	clock := time.Unix(0, 0)
	w := New(b, Options{Now: func() time.Time { return clock }})
	if _, err := w.Update(); err != nil {
		t.Fatal(err)
	}
	lm := w.Current()
	if len(lm.Adjacent) != 1 {
		t.Fatalf("adjacent %+v note %q", lm.Adjacent, w.LastBorderNote)
	}
	a := lm.Adjacent[0]
	if a.LevelNo != 2 || a.Contact != (d2map.Rect{X: 14, Y: 5, W: 1, H: 5}) || a.Rect != (d2map.Rect{X: 15, Y: 0, W: 15, H: 10}) {
		t.Fatalf("adjacent %+v", a)
	}
	if cells := lm.BorderCells(2); len(cells) != 5 || cells[0] != (d2map.Point{X: 14, Y: 5}) {
		t.Fatalf("border cells %v", cells)
	}
	if lm.BorderCells(3) != nil || lm.BorderCells(4) != nil {
		t.Fatal("unlinked or distant level must not produce border cells")
	}
	if w.cur.pending != 1 || b.probes == 0 || b.probes > 10 {
		t.Fatalf("pending %d probes %d", w.cur.pending, b.probes)
	}
	// d2d finished building level 3: the pending refetch finds it (no movement needed), after the rate limit
	b.near = append(b.near, Near{3, 0, 2})
	w.Update()
	if len(w.Current().Adjacent) != 1 {
		t.Fatal("refetched before MinRefetch")
	}
	clock = clock.Add(3 * time.Second)
	w.Update()
	if got := w.Current().Adjacent; len(got) < 2 || got[len(got)-1].LevelNo != 3 || w.cur.pending != 0 {
		t.Fatalf("after linking: %+v", got)
	}
}

func TestAdjacentNeedsLayoutVerb(t *testing.T) {
	w := New(newStub(), Options{}) // stub has no Layouter
	w.Update()
	if len(w.Current().Adjacent) != 0 {
		t.Fatal("adjacent without layout")
	}
}

func TestAbutGeometry(t *testing.T) {
	own := Placement{1, 100, 100, 10, 10}
	all := []Placement{own, {2, 110, 105, 10, 10}, {3, 90, 95, 10, 10}, {4, 95, 110, 10, 10}, {5, 100, 85, 10, 15}, {6, 110, 110, 5, 5}, {7, 111, 100, 5, 5}}
	got := map[int]abutting{}
	for _, a := range abut(own, all) {
		got[a.p.Level] = a
	}
	want := map[int][3]int{2: {3, 105, 110}, 3: {1, 100, 105}, 4: {0, 100, 105}, 5: {2, 100, 110}}
	for lv, w := range want {
		a, ok := got[lv]
		if !ok || a.side != w[0] || a.lo != w[1] || a.hi != w[2] {
			t.Errorf("level %d: %+v want %v", lv, a, w)
		}
	}
	if _, ok := got[6]; ok {
		t.Error("corner touch is not a border")
	}
	if _, ok := got[7]; ok {
		t.Error("gap of one tile is not a border")
	}
}

// d2d's grid is a tile larger than the layout and the margin is open beside the town wall (V live, Rogue
// Encampment): the gate is the gap in the layout-edge column, not the whole margin.
func TestGateIsInLayoutEdgeNotMargin(t *testing.T) {
	b := newBorderStub()
	b.level = "1 1.0 0.6 4 3 10 20" // grid 4x3 tiles, layout 3x2
	rows := make([]string, 15)
	for y := range rows {
		rows[y] = strings.Repeat(".", 14) + "#" + strings.Repeat(".", 5) // wall column x=14, open margin 15..19
		if y >= 5 && y < 10 {
			rows[y] = strings.Repeat(".", 20)
		}
	}
	b.rows = rows
	w := New(b, Options{})
	w.Update()
	lm := w.Current()
	if len(lm.Adjacent) != 1 || lm.Adjacent[0].Contact != (d2map.Rect{X: 14, Y: 5, W: 1, H: 5}) {
		t.Fatalf("%+v note %q", lm.Adjacent, w.LastBorderNote)
	}
}

func TestLinkWaitOnArrival(t *testing.T) {
	b := newBorderStub()
	polls := 0
	w := New(b, Options{LinkWait: 5 * time.Second, Sleep: func(time.Duration) {
		polls++
		if polls == 3 {
			b.near = append(b.near, Near{3, 0, 2}) // d2d's worker finished level 3
		}
	}})
	if _, err := w.Update(); err != nil {
		t.Fatal(err)
	}
	if w.cur.pending != 0 || polls != 3 || w.LinkPolls != 3 {
		t.Fatalf("pending %d polls %d", w.cur.pending, polls)
	}
	if got := w.Current().Adjacent; len(got) < 2 {
		t.Fatalf("level 3 border missing after the wait: %+v", got)
	}
	// without LinkWait the arrival does not wait
	b2 := newBorderStub()
	w2 := New(b2, Options{Sleep: func(time.Duration) { t.Fatal("slept") }})
	w2.Update()
}

// The link wait must not be cut by the per-verb Timeout: the neighbour here appears after 8 polls (160 ms real) while
// the Timeout is 50 ms. Before the fix the wait ended with the context and nav saw no exit (d2denv, default 10 s).
func TestLinkWaitOutlivesTimeout(t *testing.T) {
	b := newBorderStub()
	polls := 0
	w := New(b, Options{Timeout: 50 * time.Millisecond, LinkWait: 5 * time.Second, Sleep: func(time.Duration) {
		time.Sleep(20 * time.Millisecond)
		if polls++; polls == 8 {
			b.near = append(b.near, Near{3, 0, 2})
		}
	}})
	if _, err := w.Update(); err != nil {
		t.Fatal(err)
	}
	if w.cur.pending != 0 || polls < 8 || len(w.Current().Adjacent) < 2 {
		t.Fatalf("pending %d polls %d adjacent %+v", w.cur.pending, polls, w.Current().Adjacent)
	}
}

func TestLinkedButNothingOpenIsPending(t *testing.T) {
	b := newBorderStub()
	b.openRow = func(float64) bool { return false } // neighbour listed, collision not built yet
	w := New(b, Options{})
	w.Update()
	if w.cur.pending != 2 || len(w.Current().Adjacent) != 0 { // level 2 (nothing open) and level 3 (not linked)
		t.Fatalf("pending %d adjacent %+v", w.cur.pending, w.Current().Adjacent)
	}
}

package senses

import (
	"os"
	"strings"
	"testing"

	"d2lod-bot/internal/d2map"
)

func shipped(t *testing.T) *Adjacency {
	t.Helper()
	a, err := ShippedAdjacency()
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func recordedTown(t *testing.T) (*d2map.LevelMap, d2map.Point) {
	t.Helper()
	f, err := os.Open("../d2map/testdata/rogue_encampment.d2mi")
	if err != nil {
		t.Skip("fixture absent: ", err)
	}
	defer f.Close()
	img, err := d2map.LoadImage(f)
	if err != nil {
		t.Fatal(err)
	}
	tr := d2map.NewTracker(img)
	st, err := tr.Update()
	if err != nil {
		t.Fatal(err)
	}
	return tr.Current(), st.Pos
}

// bloodMoor is a SYNTHETIC grid (no Blood Moor image was recorded) placed where the recorded town image says the
// Blood Moor lies: Rect and the shared edge come from the town's Adjacent entry for level 2 (V by the image), its
// own Adjacent entry for the town is the town rect. All of it in Game.exe's absolute frame.
func bloodMoor(town *d2map.LevelMap, shift d2map.Point) *d2map.LevelMap {
	var rect d2map.Rect
	for _, a := range town.Adjacent {
		if a.LevelNo == 2 {
			rect = a.Rect
		}
	}
	g := d2map.ParseASCII(rect.X-shift.X, rect.Y-shift.Y, "####################", "#..................#", "#..................#", "####################")
	r := rect
	r.X -= shift.X
	r.Y -= shift.Y
	tr := town.Rect
	tr.X -= shift.X
	tr.Y -= shift.Y
	return &d2map.LevelMap{No: 2, Rect: r, Grid: g, Origin: shift,
		Adjacent: []d2map.Adjacent{{LevelNo: 1, Rect: tr}}}
}

func TestTownThenBloodMoorStitched(t *testing.T) {
	town, pos := recordedTown(t)
	var p Publisher
	p.SetAdjacency(shipped(t))
	if err := p.Observe(town, "Rogue Encampment", pos); err != nil {
		t.Fatal(err)
	}
	if s, _ := p.Snapshot(); len(s.Neighbours) != 0 {
		t.Fatalf("first area has neighbours: %+v", s.Neighbours)
	}
	bm := bloodMoor(town, d2map.Point{})
	hero := d2map.Point{X: bm.Rect.X + 5, Y: bm.Rect.Y + 1}
	if err := p.Observe(bm, "Blood Moor", hero); err != nil {
		t.Fatal(err)
	}
	s, _ := p.Snapshot()
	if len(s.Neighbours) != 1 {
		t.Fatalf("neighbours %+v skipped %v", s.Neighbours, p.Skipped())
	}
	n := s.Neighbours[0]
	if n.AreaID != 1 || n.DX != 0 || n.DY != 0 || n.Map.AreaID != 1 || n.Version != n.Map.Version {
		t.Fatalf("neighbour %+v", n)
	}
	// fog per area: the town was only revealed around where the hero stood, not whole
	full, err := Build(town, "x", 0)
	if err != nil {
		t.Fatal(err)
	}
	if known(t, Snapshot{Map: n.Map}) >= known(t, Snapshot{Map: full}) || known(t, Snapshot{Map: n.Map}) == 0 {
		t.Fatalf("town fog: %d of %d known", known(t, Snapshot{Map: n.Map}), known(t, Snapshot{Map: full}))
	}
	// the stitched maps abut: the town's west edge is the Blood Moor's east edge
	if n.Map.X0 != bm.Rect.X+bm.Rect.W {
		t.Fatalf("town x0 %d, blood moor east edge %d", n.Map.X0, bm.Rect.X+bm.Rect.W)
	}
	// back in town: the Blood Moor is now the neighbour, with the fog it had
	if err := p.Observe(town, "Rogue Encampment", pos); err != nil {
		t.Fatal(err)
	}
	s, _ = p.Snapshot()
	if len(s.Neighbours) != 1 || s.Neighbours[0].AreaID != 2 {
		t.Fatalf("back in town: %+v (skipped %v)", s.Neighbours, p.Skipped())
	}
}

// A level-relative frame (d2d without AddOrigin): the offset comes from the two rectangles, not from the table.
func TestRelativeFrameOffsetDerived(t *testing.T) {
	town, pos := recordedTown(t)
	var p Publisher
	p.SetAdjacency(shipped(t))
	p.Observe(town, "Rogue Encampment", pos)
	shift := d2map.Point{X: 5480, Y: 4800} // Blood Moor's own frame starts at 0,0
	bm := bloodMoor(town, shift)
	p.Observe(bm, "Blood Moor", d2map.Point{X: 5, Y: 1})
	s, _ := p.Snapshot()
	if len(s.Neighbours) != 1 {
		t.Fatalf("%+v %v", s.Neighbours, p.Skipped())
	}
	if n := s.Neighbours[0]; n.DX != 0 || n.DY != 0 {
		// town recorded in the absolute frame, Blood Moor's Adjacent entry for it was shifted by shift: dx = -shift
		if n.DX != -shift.X || n.DY != -shift.Y {
			t.Fatalf("offset %d,%d", n.DX, n.DY)
		}
	} else {
		t.Fatal("expected a nonzero offset")
	}
}

func TestStitchFallbacks(t *testing.T) {
	town, pos := recordedTown(t)
	hero := func(bm *d2map.LevelMap) d2map.Point { return d2map.Point{X: bm.Rect.X + 5, Y: bm.Rect.Y + 1} }

	// no table: never stitched
	var off Publisher
	off.Observe(town, "Rogue Encampment", pos)
	bm := bloodMoor(town, d2map.Point{})
	off.Observe(bm, "Blood Moor", hero(bm))
	if s, _ := off.Snapshot(); len(s.Neighbours) != 0 {
		t.Fatalf("stitched without a table: %+v", s.Neighbours)
	}

	// pair not in the table
	a, err := ParseAdjacency([]byte(`{"pairs":[{"a":2,"b":3,"evidence":"S"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	var p Publisher
	p.SetAdjacency(a)
	p.Observe(town, "Rogue Encampment", pos)
	p.Observe(bm, "Blood Moor", hero(bm))
	if s, _ := p.Snapshot(); len(s.Neighbours) != 0 || len(p.Skipped()) != 0 {
		t.Fatalf("unknown pair: %+v %v", s.Neighbours, p.Skipped())
	}

	// pair known, but the level reports no such neighbour: skipped, error names the pair
	var q Publisher
	q.SetAdjacency(shipped(t))
	q.Observe(town, "Rogue Encampment", pos)
	bare := bloodMoor(town, d2map.Point{})
	bare.Adjacent = nil
	q.Observe(bare, "Blood Moor", hero(bare))
	if s, _ := q.Snapshot(); len(s.Neighbours) != 0 {
		t.Fatalf("stitched without alignment: %+v", s.Neighbours)
	}
	if sk := q.Skipped(); len(sk) != 1 || !strings.Contains(sk[0].Error(), "2<->1") {
		t.Fatalf("skipped %v", sk)
	}

	// size mismatch
	var r Publisher
	r.SetAdjacency(shipped(t))
	r.Observe(town, "Rogue Encampment", pos)
	bad := bloodMoor(town, d2map.Point{})
	bad.Adjacent[0].Rect.W++
	r.Observe(bad, "Blood Moor", hero(bad))
	if sk := r.Skipped(); len(sk) != 1 || !strings.Contains(sk[0].Error(), "2<->1") || !strings.Contains(sk[0].Error(), "recorded") {
		t.Fatalf("skipped %v", sk)
	}
}

func TestAdjacencyTableErrors(t *testing.T) {
	for in, want := range map[string]string{
		`{"pairs":[{"a":1,"b":2},{"a":2,"b":1}]}`: "1<->2",
		`{"pairs":[{"a":3,"b":3}]}`:               "3<->3",
		`{`:                                       "adjacency table",
	} {
		if _, err := ParseAdjacency([]byte(in)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v", in, err)
		}
	}
	a := shipped(t)
	if !a.Has(1, 2) || !a.Has(2, 1) || a.Has(1, 3) {
		t.Fatal("shipped table pairs")
	}
	var nilA *Adjacency
	if nilA.Has(1, 2) {
		t.Fatal("nil table")
	}
}

func TestNeighbourCap(t *testing.T) {
	cur := &d2map.LevelMap{No: 3, Rect: d2map.Rect{X: 100, Y: 0, W: 10, H: 10}, Adjacent: []d2map.Adjacent{
		{LevelNo: 2, Rect: d2map.Rect{X: 0, Y: 0, W: 10, H: 10}}, {LevelNo: 4, Rect: d2map.Rect{X: 200, Y: 0, W: 10, H: 10}}}}
	adj, _ := ParseAdjacency([]byte(`{"pairs":[{"a":2,"b":3},{"a":3,"b":4}]}`))
	seen := map[int]*visit{
		2: {m: Map{AreaID: 2, Version: 1, RLE: make([]int, MaxNeighbourRLE+2)}, rect: d2map.Rect{X: 0, Y: 0, W: 10, H: 10}, seq: 1},
		4: {m: Map{AreaID: 4, Version: 2, RLE: []int{1, 4}}, rect: d2map.Rect{X: 200, Y: 0, W: 10, H: 10}, seq: 2},
	}
	ns, sk := neighbours(adj, cur, seen)
	if len(ns) != 1 || ns[0].AreaID != 4 || len(sk) != 1 || !strings.Contains(sk[0].Error(), "3<->2") {
		t.Fatalf("%+v %v", ns, sk)
	}
}

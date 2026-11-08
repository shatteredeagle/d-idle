package senses

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"d2lod-bot/internal/d2map"
)

// tiny returns a synthesized area (not a game recording): a corridor with a wall, an exit and a waypoint.
func tiny(no int) *d2map.LevelMap {
	return &d2map.LevelMap{No: no, Grid: d2map.ParseASCII(100, 200,
		"##########",
		"#........#",
		"#.######.#",
		"#........#",
		"##########"),
		Exits:   []d2map.Exit{{Dest: 3, Pos: d2map.Point{X: 101, Y: 201}}},
		Presets: []d2map.Preset{{Kind: d2map.PresetObject, ID: 119, Pos: d2map.Point{X: 108, Y: 203}}}}
}

func TestRLERoundTrip(t *testing.T) {
	for _, c := range [][]int{{}, {0}, {1, 1, 2, 2, 0, 0, 1}, make([]int, 1000)} {
		got, err := Decode(Encode(c), len(c))
		if err != nil || !reflect.DeepEqual(append([]int{}, got...), append([]int{}, c...)) {
			t.Fatalf("%v -> %v %v", c, got, err)
		}
	}
	for _, bad := range [][]int{{1}, {5, 1}, {1, 0}, {1, 9}} {
		if _, err := Decode(bad, 3); err == nil {
			t.Fatalf("%v accepted", bad)
		}
	}
	if _, err := Decode([]int{1, 2}, 3); err == nil {
		t.Fatal("short stream accepted")
	}
}

func TestBuildGolden(t *testing.T) {
	m, err := Build(tiny(1), "Rogue Encampment", 1)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := json.MarshalIndent(m, "", " ")
	const path = "testdata/tiny.golden.json"
	if os.Getenv("UPDATE_GOLDEN") != "" {
		os.WriteFile(path, append(got, '\n'), 0o644)
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(want)) != string(got) {
		t.Fatalf("golden differs:\n%s", got)
	}
	cells, err := Decode(m.RLE, m.W*m.H)
	if err != nil || cells[1*m.W+1] != Walkable || cells[0] != Blocked {
		t.Fatalf("decode %v %v", cells, err)
	}
}

func TestDownsample(t *testing.T) {
	rows := make([]string, 600)
	for i := range rows {
		rows[i] = strings.Repeat(".", 600)
	}
	m, err := Build(&d2map.LevelMap{No: 2, Grid: d2map.ParseASCII(0, 0, rows...)}, "", 1)
	if err != nil || m.Step != 3 || m.W != 200 || m.H != 200 {
		t.Fatalf("%+v %v", m, err)
	}
}

func TestUnknownAreaErrorNamesIt(t *testing.T) {
	_, err := Build(&d2map.LevelMap{No: 77, Grid: d2map.NewGrid()}, "", 1)
	if err == nil || !strings.Contains(err.Error(), "77") {
		t.Fatalf("err %v", err)
	}
	var p Publisher
	if err := p.Observe(&d2map.LevelMap{No: 78}, "", d2map.Point{}); err == nil || !strings.Contains(err.Error(), "78") {
		t.Fatalf("err %v", err)
	}
	if _, ok := p.Snapshot(); ok {
		t.Fatal("snapshot after error")
	}
}

func known(t *testing.T, s Snapshot) int {
	t.Helper()
	c, err := Decode(s.Map.RLE, s.Map.W*s.Map.H)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, v := range c {
		if v != Unknown {
			n++
		}
	}
	return n
}

func TestFogRevealsAsHeroWalks(t *testing.T) {
	rows := make([]string, 40)
	for i := range rows {
		rows[i] = strings.Repeat(".", 200)
	}
	lm := &d2map.LevelMap{No: 1, Grid: d2map.ParseASCII(0, 0, rows...),
		Exits: []d2map.Exit{{Dest: 2, Pos: d2map.Point{X: 190, Y: 20}}}}
	var p Publisher
	if err := p.Observe(lm, "A", d2map.Point{X: 10, Y: 20}); err != nil {
		t.Fatal(err)
	}
	a, _ := p.Snapshot()
	if len(a.Map.Exits) != 0 || known(t, a) == 0 || known(t, a) >= 200*40 {
		t.Fatalf("start: exits %v known %d", a.Map.Exits, known(t, a))
	}
	// same spot: same version, hero moves inside revealed cells
	p.Observe(lm, "A", d2map.Point{X: 10, Y: 20})
	if b, _ := p.Snapshot(); b.Map.Version != a.Map.Version {
		t.Fatal("version moved without a reveal")
	}
	for x := 10; x <= 190; x += 10 {
		p.Observe(lm, "A", d2map.Point{X: x, Y: 20})
	}
	c, _ := p.Snapshot()
	if c.Map.Version <= a.Map.Version || known(t, c) <= known(t, a) || len(c.Map.Exits) != 1 {
		t.Fatalf("after walk: version %d known %d exits %v", c.Map.Version, known(t, c), c.Map.Exits)
	}
	if c.Hero != (Point{190, 20}) {
		t.Fatalf("hero %v", c.Hero)
	}
}

func TestTwoAreasKeepOwnFog(t *testing.T) {
	var p Publisher
	p.Observe(tiny(1), "One", d2map.Point{X: 101, Y: 201})
	a, _ := p.Snapshot()
	p.Observe(tiny(5), "Two", d2map.Point{X: 108, Y: 203})
	b, _ := p.Snapshot()
	if b.Map.AreaID != 5 || b.Map.Version <= a.Map.Version {
		t.Fatalf("area switch: %+v", b.Map)
	}
	p.Observe(tiny(1), "One", d2map.Point{X: 101, Y: 201})
	c, _ := p.Snapshot()
	if c.Map.AreaID != 1 || known(t, c) != known(t, a) {
		t.Fatalf("fog of area 1 lost: %d vs %d", known(t, c), known(t, a))
	}
}

// The recorded Rogue Encampment of internal/d2map (recorded live 2026-10-28); skipped where the file is absent.
func TestRecordedTown(t *testing.T) {
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
	var p Publisher
	if err := p.Observe(tr.Current(), "Rogue Encampment", st.Pos); err != nil {
		t.Fatal(err)
	}
	s, _ := p.Snapshot()
	if s.Map.AreaID != 1 || s.Map.Step != 2 || s.Map.W != 140 || known(t, s) == 0 || s.Hero != (Point{st.Pos.X, st.Pos.Y}) {
		t.Fatalf("%+v", s.Map)
	}
}

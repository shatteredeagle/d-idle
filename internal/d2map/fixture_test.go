package d2map

import (
	"os"
	"testing"
)

func loadFixture(t *testing.T) *Image {
	t.Helper()
	f, err := os.Open("testdata/rogue_encampment.d2mi")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := LoadImage(f)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

// The fixture was recorded from the live game (Rogue Encampment, 2026-10-28) with
// `d2mapdump record`; the expected values below were checked live (docs/map.md).
func TestRecordedRogueEncampment(t *testing.T) {
	tr := NewTracker(loadFixture(t))
	st, err := tr.Update()
	if err != nil {
		t.Fatal(err)
	}
	if st.LevelNo != 1 || st.Act != 0 || st.ActiveRooms != 36 || st.NewRooms != 36 {
		t.Fatalf("state %+v", st)
	}
	lm := tr.Current()
	if lm.Rect != (Rect{5960, 4840, 280, 200}) || len(lm.Rooms) != 35 {
		t.Fatalf("level %+v rooms %d", lm.Rect, len(lm.Rooms))
	}
	g := lm.Grid
	if g.KnownCount() != 48000 {
		t.Fatalf("known %d", g.KnownCount())
	}
	// the four rooms at x=6160 had no room data in the client: they stay unknown
	if g.Known(6180, 4900) || !g.Known(6140, 4900) || !g.Known(6200, 4900) {
		t.Fatal("unknown strip wrong")
	}
	// player stands on walkable ground and the Player bit marks the cell
	if !st.Room.Rect.Contains(st.Pos.X, st.Pos.Y) {
		t.Fatalf("player %v outside room %+v", st.Pos, st.Room.Rect)
	}
	if f, _ := g.Flags(st.Pos.X, st.Pos.Y); f&Player == 0 || !g.Walkable(st.Pos.X, st.Pos.Y) {
		t.Fatalf("player cell flags %#x", f)
	}
	// town fence objects (classes 40/41/42/65 at x=6120..6140) are solid
	for _, p := range lm.Presets {
		if p.Kind == PresetObject && p.ID >= 40 && p.ID <= 42 && g.Walkable(p.Pos.X, p.Pos.Y) {
			t.Fatalf("fence object at %v is walkable", p.Pos)
		}
	}
	// well-known presets, positions cross-checked against the live unit table
	has := map[[3]int]bool{}
	for _, p := range lm.Presets {
		has[[3]int{p.Kind, p.ID, p.Pos.X*10000 + p.Pos.Y}] = true
	}
	for _, w := range []struct{ kind, id, x, y int }{
		{PresetObject, 119, 6044, 4909}, // waypoint
		{PresetObject, 267, 6016, 4934}, // stash
		{PresetNPC, 148, 6082, 4911},    // Akara
		{PresetNPC, 147, 5985, 4983},    // Gheed
		{PresetNPC, 154, 5984, 4922},    // Charsi
	} {
		if !has[[3]int{w.kind, w.id, w.x*10000 + w.y}] {
			t.Errorf("missing preset %+v", w)
		}
	}
	if wp := lm.Waypoints(); len(wp) != 1 || wp[0].Pos != (Point{6044, 4909}) {
		t.Fatalf("waypoints %+v", wp)
	}
	// town connects to the Blood Moor by walking across its west edge, not through a tile
	if len(lm.Exits) != 0 {
		t.Fatalf("unexpected tile exits %+v", lm.Exits)
	}
	if len(lm.Adjacent) != 1 || lm.Adjacent[0].LevelNo != 2 || lm.Adjacent[0].Rect != (Rect{5480, 4800, 480, 280}) {
		t.Fatalf("adjacent %+v", lm.Adjacent)
	}
	// the only gap in the town's west wall is the gate to the Blood Moor: 15 cells at x=5960,
	// y=4945..4959. One more isolated walkable cell (5960,4867) is walled in on every side.
	gate := 0
	for _, p := range lm.BorderCells(2) {
		if p.X == 5960 && p.Y >= 4945 && p.Y <= 4959 {
			gate++
		}
	}
	if gate != 15 || len(lm.BorderCells(2)) != 16 {
		t.Fatalf("gate cells %d of %d", gate, len(lm.BorderCells(2)))
	}
	// rooms of the Blood Moor that were loaded were routed into their own level
	if bm := tr.Level(2); bm == nil || bm.Grid.KnownCount() == 0 || bm.Grid.Known(6000, 4900) {
		t.Fatal("Blood Moor cache")
	}
}

// A second recorded game (different seed, mirrored town): everything the bot needs must come from the level data.
func TestRecordedRogueEncampmentSecondSeed(t *testing.T) {
	f, err := os.Open("testdata/rogue_encampment_b.d2mi")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := LoadImage(f)
	if err != nil {
		t.Fatal(err)
	}
	tr := NewTracker(img)
	st, err := tr.Update()
	if err != nil || st.LevelNo != 1 {
		t.Fatalf("%+v %v", st, err)
	}
	lm := tr.Current()
	a := loadFixtureTracker(t)
	if lm.Rect == a.Current().Rect {
		t.Fatal("both fixtures have the same origin; the second one proves nothing")
	}
	if wp := lm.Waypoints(); len(wp) != 1 || !lm.Grid.Known(wp[0].Pos.X, wp[0].Pos.Y) {
		t.Fatalf("waypoints %+v", wp)
	}
	if len(lm.Adjacent) != 1 || lm.Adjacent[0].LevelNo != 2 || len(lm.BorderCells(2)) < 10 {
		t.Fatalf("adjacent %+v border %d", lm.Adjacent, len(lm.BorderCells(2)))
	}
	// the Blood Moor lies east of this town, west of the other one
	if lm.Adjacent[0].Rect.X < lm.Rect.X || a.Current().Adjacent[0].Rect.X > a.Current().Rect.X {
		t.Fatal("expected a mirrored town")
	}
}

func loadFixtureTracker(t *testing.T) *Tracker {
	tr := NewTracker(loadFixture(t))
	if _, err := tr.Update(); err != nil {
		t.Fatal(err)
	}
	return tr
}

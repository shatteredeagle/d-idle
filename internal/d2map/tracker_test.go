package d2map

import (
	"errors"
	"testing"
	"time"

	"d2lod-bot/internal/mem"
)

// twoRoomTown builds level 1 (tile rect 100,100 4x2 => subtiles 500,500 20x10) with two 10x10
// rooms side by side. Room A is fully walkable with a wall column at local x=3, room B is open.
func twoRoomTown(t *testing.T) (w *world, lvl, r2a, r2b, r1a, r1b uint32) {
	w = newWorld(t)
	lvl = w.level(1, 100, 100, 4, 2)
	r2a = w.room2(lvl, 100, 100, 2, 2)
	r2b = w.room2(lvl, 102, 100, 2, 2)
	a := fill(10, 10, 0)
	for y := 0; y < 10; y++ {
		a[y*10+3] = BlockWalk | Wall
	}
	r1a = w.room1(r2a, Rect{500, 500, 10, 10}, a)
	r1b = w.room1(r2b, Rect{510, 500, 10, 10}, fill(10, 10, 0))
	w.player(505, 505, r1a)
	return
}

func TestTrackerMergesRoomsAndFlags(t *testing.T) {
	w, _, _, _, _, _ := twoRoomTown(t)
	tr := NewTracker(w.f)
	st, err := tr.Update()
	if err != nil {
		t.Fatal(err)
	}
	if st.LevelNo != 1 || st.Pos != (Point{505, 505}) || st.ActiveRooms != 2 || st.NewRooms != 2 || !st.LevelChanged {
		t.Fatalf("state %+v", st)
	}
	g := tr.Current().Grid
	if got := g.Bounds(); got != (Rect{500, 500, 20, 10}) {
		t.Fatalf("bounds %+v", got)
	}
	if g.Walkable(503, 500) || !g.Walkable(504, 500) || !g.Walkable(515, 509) {
		t.Fatal("walkability wrong")
	}
	if f, ok := g.Flags(503, 509); !ok || f != BlockWalk|Wall {
		t.Fatalf("flags %x %v", f, ok)
	}
	if g.Known(499, 500) || g.Walkable(499, 500) || g.Known(520, 500) {
		t.Fatal("outside cells must be unknown")
	}
	if st2, _ := tr.Update(); st2.LevelChanged || st2.NewRooms != 0 {
		t.Fatalf("second update %+v", st2)
	}
}

func TestTrackerAccumulatesRoomsAsTheyLoad(t *testing.T) {
	w := newWorld(t)
	lvl := w.level(1, 100, 100, 4, 2)
	r2a := w.room2(lvl, 100, 100, 2, 2)
	r2b := w.room2(lvl, 102, 100, 2, 2)
	r1a := w.room1(r2a, Rect{500, 500, 10, 10}, fill(10, 10, 0))
	w.player(505, 505, r1a)
	tr := NewTracker(w.f)
	tr.RescanEvery = 1
	if _, err := tr.Update(); err != nil {
		t.Fatal(err)
	}
	g := tr.Current().Grid
	if g.Known(515, 505) {
		t.Fatal("room B not loaded yet")
	}
	w.room1(r2b, Rect{510, 500, 10, 10}, fill(10, 10, BlockWalk))
	st, _ := tr.Update()
	if st.NewRooms != 1 || !g.Known(515, 505) || g.Walkable(515, 505) {
		t.Fatalf("room B not merged: %+v", st)
	}
	// B unloads (removed from the active list) but stays cached.
	w.set(w.act, offActRoom1, w.get(w.act, offActRoom1))
	w.set(r2b, offR2Room1, 0)
	if _, err := tr.Update(); err != nil || !g.Known(515, 505) {
		t.Fatalf("cache lost: %v", err)
	}
}

func TestOverlappingRoomsLatestWins(t *testing.T) {
	w := newWorld(t)
	lvl := w.level(1, 100, 100, 4, 2)
	r2a := w.room2(lvl, 100, 100, 2, 2)
	r2b := w.room2(lvl, 101, 100, 2, 2)
	r1a := w.room1(r2a, Rect{500, 500, 10, 10}, fill(10, 10, BlockWalk))
	w.room1(r2b, Rect{505, 500, 10, 10}, fill(10, 10, 0))
	w.player(501, 501, r1a)
	tr := NewTracker(w.f)
	if _, err := tr.Update(); err != nil {
		t.Fatal(err)
	}
	g := tr.Current().Grid
	if g.Walkable(502, 502) || !g.Walkable(507, 502) || !g.Walkable(512, 502) {
		t.Fatal("overlap should take the later room")
	}
	if g.KnownCount() != 15*10 {
		t.Fatalf("known %d", g.KnownCount())
	}
}

func TestDynamicRefreshOfPlayerRoom(t *testing.T) {
	w, _, _, _, r1a, _ := twoRoomTown(t)
	tr := NewTracker(w.f)
	tr.RescanEvery = 1000
	tr.Update()
	g := tr.Current().Grid
	// a monster steps onto (506,500) in room A: rewrite the room's cells
	cells := fill(10, 10, 0)
	cells[6] = NPCCollision
	coll := w.get(r1a, offR1Coll)
	data := w.get(coll, offCollStart)
	for i, v := range cells {
		w.set16(data, 2*i, v)
	}
	tr.Update()
	if f, _ := g.Flags(506, 500); f != NPCCollision {
		t.Fatalf("player's room not refreshed: %x", f)
	}
}

func TestNotInGameAndNullPointers(t *testing.T) {
	f := mem.NewFake()
	f.PutU32(mem.Base+rvaPlayerUnit, 0)
	tr := NewTracker(f)
	if _, err := tr.Update(); !errors.Is(err, ErrNotInGame) {
		t.Fatalf("want ErrNotInGame, got %v", err)
	}
	// valid player but null room1: still ErrNotInGame, no panic
	w := newWorld(t)
	w.level(1, 100, 100, 4, 2)
	w.player(505, 505, 0)
	tr = NewTracker(w.f)
	if _, err := tr.Update(); !errors.Is(err, ErrNotInGame) {
		t.Fatalf("want ErrNotInGame, got %v", err)
	}
}

func TestRoomWithNullCollIsSkipped(t *testing.T) {
	w, _, _, _, r1a, r1b := twoRoomTown(t)
	w.set(r1b, offR1Coll, 0)
	tr := NewTracker(w.f)
	st, err := tr.Update()
	if err != nil || st.NewRooms != 1 {
		t.Fatalf("%+v %v", st, err)
	}
	_ = r1a
	if tr.Current().Grid.Known(515, 505) {
		t.Fatal("null coll must not produce cells")
	}
}

func TestCollDisagreeingWithRoom1IsRejected(t *testing.T) {
	w, _, _, _, _, r1b := twoRoomTown(t)
	w.set(w.get(r1b, offR1Coll), offCollX, 9999) // stale/recycled collmap
	tr := NewTracker(w.f)
	tr.Update()
	if tr.Current().Grid.Known(515, 505) {
		t.Fatal("mismatching collmap accepted")
	}
}

func TestCyclesTerminate(t *testing.T) {
	w, lvl, r2a, r2b, r1a, r1b := twoRoomTown(t)
	// Room1 list loops, near arrays point at themselves, Room2 list loops, levels loop,
	// presets and tiles loop.
	w.set(r1b, offR1Next, r1a)
	w.set(r1a, offR1Next, r1b)
	w.set(r2b, offR2Next, r2a)
	w.set(lvl, offLvlNext, lvl)
	w.near2(r2a, r2a, r2b, r2a)
	p := w.preset(r2a, PresetNPC, 1, 1, 1)
	w.set(p, offPreNext, p)
	w.tile(r2a, r2b, 7)
	tl := w.get(r2a, offR2Tiles)
	w.set(tl, offTileNext, tl)
	arr := w.alloc(8)
	w.set(arr, 0, r1a)
	w.set(arr, 4, r1b)
	w.set(r1a, offR1Near, arr)
	w.set(r1a, offR1NearCount, 2)

	done := make(chan struct{})
	go func() {
		defer close(done)
		tr := NewTracker(w.f)
		for i := 0; i < 3; i++ {
			if _, err := tr.Update(); err != nil {
				t.Error(err)
			}
		}
		if n := len(tr.Current().Rooms); n != 2 {
			t.Errorf("rooms %d", n)
		}
		r := Reader{w.f}
		a, _ := r.Act(w.act)
		if len(r.Levels(a)) != 1 || len(r.ActiveRoom1s(a, r1a)) != 2 {
			t.Error("cycle handling")
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("did not terminate on cyclic structures")
	}
}

func TestGarbageStructsDoNotPanic(t *testing.T) {
	w, lvl, r2a, _, r1a, _ := twoRoomTown(t)
	w.set(lvl, offLvlSizeX, 0x7fffffff)
	w.set(r2a, offR2SizeY, 0xfffffff0)
	w.set(r1a, offR1XSize, 100000)
	tr := NewTracker(w.f)
	tr.Update() // errors are fine, panics and hangs are not
	tr.Update()
}

func TestTornCollmapRead(t *testing.T) {
	w, _, _, _, r1a, _ := twoRoomTown(t)
	coll := w.get(r1a, offR1Coll)
	torn := &tornSource{Source: w.f, addr: coll}
	r := Reader{torn}
	rm, err := r.Room1(r1a)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Coll(rm); err == nil {
		t.Fatal("torn read accepted")
	}
}

// tornSource returns different header bytes on every read of addr, like a room being freed and
// reused while we read it.
type tornSource struct {
	mem.Source
	addr uint32
	n    byte
}

func (s *tornSource) ReadAt(p []byte, addr uint32) error {
	if err := s.Source.ReadAt(p, addr); err != nil {
		return err
	}
	if addr == s.addr {
		s.n++
		p[offCollRoomX] = s.n
	}
	return nil
}

func TestLevelChangeAndCaching(t *testing.T) {
	w, _, _, _, r1a, _ := twoRoomTown(t)
	lvl2 := w.level(2, 200, 100, 2, 2)
	r2c := w.room2(lvl2, 200, 100, 2, 2)
	r1c := w.room1(r2c, Rect{1000, 500, 10, 10}, fill(10, 10, 0))
	w.player(505, 505, r1a)
	tr := NewTracker(w.f)
	st, _ := tr.Update()
	if st.LevelNo != 1 || !st.LevelChanged {
		t.Fatalf("%+v", st)
	}
	// walk into level 2
	w.movePlayer(1004, 504, r1c)
	st, err := tr.Update()
	if err != nil || st.LevelNo != 2 || !st.LevelChanged {
		t.Fatalf("%+v %v", st, err)
	}
	if tr.Level(1) == nil || tr.Level(1).Grid.KnownCount() != 200 {
		t.Fatal("level 1 cache lost")
	}
	if tr.Current().Grid.KnownCount() != 100 {
		t.Fatalf("level 2 grid has %d cells", tr.Current().Grid.KnownCount())
	}
	// loading screen: chain broken
	w.set(w.pathAddr, offPathRoom1, 0)
	if _, err := tr.Update(); !errors.Is(err, ErrNotInGame) {
		t.Fatalf("%v", err)
	}
	if tr.Current().No != 2 {
		t.Fatal("state must survive a loading screen")
	}
	if got := tr.LevelNos(); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("levels %v", got)
	}
}

func TestNewGameClearsCache(t *testing.T) {
	w, _, _, _, _, _ := twoRoomTown(t)
	tr := NewTracker(w.f)
	tr.Update()
	w.set(w.act, offActSeed, 0x9999)
	tr.Update()
	if tr.Level(1).Grid.KnownCount() != 200 {
		t.Fatal("rebuilt cache should hold the new game's rooms")
	}
	if n := len(tr.LevelNos()); n != 1 {
		t.Fatalf("levels %d", n)
	}
}

func TestPresetsExitsWaypointsAdjacency(t *testing.T) {
	w, lvl, r2a, r2b, _, _ := twoRoomTown(t)
	lvl2 := w.level(2, 98, 100, 2, 2) // west neighbour: subtiles 490..500
	r2w := w.room2(lvl2, 98, 100, 2, 2)
	lvl3 := w.level(8, 300, 300, 2, 2)
	r2cave := w.room2(lvl3, 300, 300, 2, 2)

	w.preset(r2a, PresetNPC, 148, 3, 4)
	w.preset(r2b, PresetObject, 119, 5, 6) // waypoint
	w.preset(r2b, PresetObject, 267, 1, 1)
	w.preset(r2b, PresetTile, 60, 8, 2)
	w.tile(r2b, r2cave, 60)
	w.tile(r2b, r2a, 61) // unmatched
	w.near2(r2a, r2w, r2b)

	tr := NewTracker(w.f)
	if _, err := tr.Update(); err != nil {
		t.Fatal(err)
	}
	lm := tr.Current()
	_ = lvl
	want := []Preset{
		{PresetNPC, 148, Point{503, 504}, r2a},
		{PresetObject, 119, Point{515, 506}, r2b},
		{PresetObject, 267, Point{511, 501}, r2b},
		{PresetTile, 60, Point{518, 502}, r2b},
	}
	if len(lm.Presets) != len(want) {
		t.Fatalf("presets %+v", lm.Presets)
	}
	for _, p := range want {
		found := false
		for _, q := range lm.Presets {
			found = found || p == q
		}
		if !found {
			t.Errorf("missing preset %+v in %+v", p, lm.Presets)
		}
	}
	if len(lm.Exits) != 1 || lm.Exits[0].Dest != 8 || lm.Exits[0].Pos != (Point{518, 502}) {
		t.Fatalf("exits %+v", lm.Exits)
	}
	if wp := lm.Waypoints(); len(wp) != 1 || wp[0].Pos != (Point{515, 506}) {
		t.Fatalf("waypoints %+v", wp)
	}
	if len(lm.Adjacent) != 1 || lm.Adjacent[0].LevelNo != 2 {
		t.Fatalf("adjacent %+v", lm.Adjacent)
	}
	if c := lm.Adjacent[0].Contact; c != (Rect{500, 500, 1, 10}) {
		t.Fatalf("contact %+v", c)
	}
	// the wall column is at x=503, the border column x=500 is open
	if cells := lm.BorderCells(2); len(cells) != 10 || cells[0] != (Point{500, 500}) {
		t.Fatalf("border cells %v", cells)
	}
	if lm.BorderCells(99) != nil {
		t.Fatal("unknown neighbour")
	}
}

func TestPresetsAppearWhenRoomDataLoads(t *testing.T) {
	w, _, r2a, _, _, _ := twoRoomTown(t)
	tr := NewTracker(w.f)
	tr.Update()
	if len(tr.Current().Presets) != 0 {
		t.Fatal("no presets yet")
	}
	w.preset(r2a, PresetNPC, 150, 0, 0)
	tr.Update()
	if len(tr.Current().Presets) != 1 {
		t.Fatal("late preset not picked up")
	}
}

// NAVFIX: the client frees a Room2's preset, tile and near data when the player walks away. Exits and borders seen
// earlier must stay known (live on i1: back in the Blood Moor the Den entrance and the town border had vanished,
// nav said "no exit" and the fully explored grid had no frontier).
func TestStaticDataSurvivesRoomDataBeingFreed(t *testing.T) {
	w, _, r2a, r2b, _, _ := twoRoomTown(t)
	lvl2 := w.level(2, 98, 100, 2, 2)
	r2w := w.room2(lvl2, 98, 100, 2, 2)
	lvl3 := w.level(8, 300, 300, 2, 2)
	r2cave := w.room2(lvl3, 300, 300, 2, 2)
	w.preset(r2b, PresetObject, 119, 5, 6)
	w.preset(r2b, PresetTile, 60, 8, 2)
	w.tile(r2b, r2cave, 60)
	w.near2(r2a, r2w, r2b)
	tr := NewTracker(w.f)
	if _, err := tr.Update(); err != nil {
		t.Fatal(err)
	}
	if lm := tr.Current(); len(lm.Exits) != 1 || len(lm.Adjacent) != 1 || len(lm.Presets) != 2 {
		t.Fatalf("setup: exits %+v adjacent %+v presets %+v", lm.Exits, lm.Adjacent, lm.Presets)
	}
	// the client drops the room data of r2b and the near links of r2a
	w.set(r2b, offR2Preset, 0)
	w.set(r2b, offR2Tiles, 0)
	w.set(r2a, offR2Near, 0)
	w.set(r2a, offR2NearCount, 0)
	if _, err := tr.Update(); err != nil {
		t.Fatal(err)
	}
	lm := tr.Current()
	if len(lm.Exits) != 1 || lm.Exits[0].Dest != 8 {
		t.Fatalf("exit forgotten: %+v", lm.Exits)
	}
	if len(lm.Adjacent) != 1 || len(lm.BorderCells(2)) == 0 {
		t.Fatalf("border forgotten: %+v", lm.Adjacent)
	}
	if len(lm.Waypoints()) != 1 {
		t.Fatalf("waypoint forgotten: %+v", lm.Presets)
	}
}

func TestMergePresetsDropsDuplicates(t *testing.T) {
	a := []Preset{{PresetObject, 119, Point{1, 2}, 10}}
	a = mergePresets(a, []Preset{{PresetObject, 119, Point{1, 2}, 20}, {PresetNPC, 148, Point{3, 4}, 20}})
	if len(a) != 2 || a[0].Room2 != 20 {
		t.Fatalf("%+v", a)
	}
}

func TestLevelRectFilledInLate(t *testing.T) {
	w, _, _, _, _, _ := twoRoomTown(t)
	lvl2 := w.level(2, 0, 0, 0, 0) // not initialised yet
	_ = w.room2(lvl2, 98, 100, 2, 2)
	tr := NewTracker(w.f)
	if lm := tr.levelMap(mustLevel(t, tr, lvl2)); !lm.Rect.Empty() {
		t.Fatal("rect should still be empty")
	}
	w.set(lvl2, offLvlSizeX, 2)
	w.set(lvl2, offLvlSizeY, 2)
	if lm := tr.levelMap(mustLevel(t, tr, lvl2)); lm.Rect.Empty() {
		t.Fatal("rect not refreshed")
	}
}

func mustLevel(t *testing.T, tr *Tracker, addr uint32) Level {
	t.Helper()
	l, err := tr.R.Level(addr)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

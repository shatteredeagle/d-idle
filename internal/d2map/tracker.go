package d2map

import (
	"errors"
	"fmt"
	"sort"
	"sync/atomic"

	"d2lod-bot/internal/mem"
)

// LevelMap is everything known about one level in the current game.
type LevelMap struct {
	No   int
	Rect Rect  // level rectangle from the Level struct (the Grid fills in as rooms load)
	Grid *Grid // merged collision of every room of this level seen so far

	Presets  []Preset // all preset units of all Room2s, sorted
	Exits    []Exit
	Adjacent []Adjacent
	Warps    []Point // Level.WarpX/Y, UNVERIFIED semantics
	Rooms    []Room2 // static Room2 list

	room2Sig map[uint32][3]uint32 // Room2 -> (pRoom1, pPreset, pRoomTiles) when last scanned
}

// Waypoints returns the waypoint objects among the presets.
func (l *LevelMap) Waypoints() []Preset {
	var out []Preset
	for _, p := range l.Presets {
		if p.Kind == PresetObject && IsWaypointObject(p.ID) {
			out = append(out, p)
		}
	}
	return out
}

// State is what one Update observed about the player.
type State struct {
	Pos          Point
	LevelNo      int
	Act          int
	Room         Room2   // the player's current Room2
	NearRooms    []Room2 // Room2s linked from it (pRoom2Near), incl. neighbouring levels
	LevelChanged bool    // true on the first Update in a level different from the previous one
	NewRooms     int     // collision rooms merged this tick
	ActiveRooms  int     // Room1s the client has active right now
}

// ErrNotInGame means the player/room chain is not valid (menu, loading screen, level change).
var ErrNotInGame = errors.New("d2map: player not in a room")

// Tracker accumulates rooms as the client loads them. Call Update about once per tick from a
// single goroutine; Current/Level return live objects, Clone the Grid to share it.
type Tracker struct {
	R           Reader
	RescanEvery int // re-read already merged rooms every N updates (default 4; 1 = every tick)

	tick     int
	seed     uint32
	haveSeed bool
	levels   map[int]*LevelMap
	byAddr   map[uint32]Level
	loaded   map[uint32]loadedRoom
	cur      *LevelMap
	curNo    int
	resetReq atomic.Bool
}

type loadedRoom struct {
	rect  Rect
	level int
}

func NewTracker(src mem.Source) *Tracker {
	t := &Tracker{R: Reader{src}, RescanEvery: 4}
	t.reset()
	return t
}

// Reset asks for a forgotten map at the next Update (safe from any goroutine). The bot calls it after it re-entered a game
// through the menus: the next game may reuse the act seed but not the heap addresses the caches are keyed by
// (found live on i4, 2026-10-28: after a Save and Exit the town reported "no exit to level 2" forever).
func (t *Tracker) Reset() { t.resetReq.Store(true) }

func (t *Tracker) reset() {
	t.levels = map[int]*LevelMap{}
	t.byAddr = map[uint32]Level{}
	t.loaded = map[uint32]loadedRoom{}
	t.cur, t.curNo = nil, -1
}

// Current is the LevelMap of the level the player is in (nil before the first good Update).
func (t *Tracker) Current() *LevelMap { return t.cur }

// Level returns the cached map for a level id, if any room of it has been seen this game.
func (t *Tracker) Level(no int) *LevelMap { return t.levels[no] }

// LevelNos lists cached level ids in ascending order.
func (t *Tracker) LevelNos() []int {
	var out []int
	for n := range t.levels {
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}

func (t *Tracker) levelAt(addr uint32) (Level, error) {
	if l, ok := t.byAddr[addr]; ok {
		return l, nil
	}
	l, err := t.R.Level(addr)
	if err != nil {
		return l, err
	}
	t.byAddr[addr] = l
	return l, nil
}

func (t *Tracker) levelMap(l Level) *LevelMap {
	lm := t.levels[l.No]
	if lm != nil && lm.Rect.Empty() && !l.Rect.Empty() {
		lm.Rect = l.Rect // first read came before the level was initialised
	}
	if lm == nil {
		lm = &LevelMap{No: l.No, Rect: l.Rect, Grid: NewGrid(), Warps: l.Warps, room2Sig: map[uint32][3]uint32{}}
		t.levels[l.No] = lm
	}
	return lm
}

// Update reads the player's position and room chain, merges any newly loaded (or, every
// RescanEvery updates, all) active rooms into their levels' grids and refreshes static data.
func (t *Tracker) Update() (State, error) {
	var st State
	t.tick++
	if t.resetReq.Swap(false) {
		t.reset()
	}
	pc, err := t.R.Player()
	if err != nil {
		return st, fmt.Errorf("%w: %v", ErrNotInGame, err)
	}
	act, err := t.R.Act(pc.Act)
	if err != nil {
		return st, fmt.Errorf("%w: %v", ErrNotInGame, err)
	}
	if t.haveSeed && act.Seed != t.seed {
		t.reset() // new game: rooms of the old map are meaningless
	}
	t.seed, t.haveSeed = act.Seed, true
	st.Pos, st.Act = pc.Pos, act.Index

	room1, err := t.R.Room1(pc.Room1)
	if err != nil {
		return st, fmt.Errorf("%w: %v", ErrNotInGame, err)
	}
	room2, err := t.R.Room2(room1.Room2)
	if err != nil {
		return st, fmt.Errorf("%w: %v", ErrNotInGame, err)
	}
	lvl, err := t.R.Level(room2.Level)
	if err != nil {
		return st, fmt.Errorf("%w: %v", ErrNotInGame, err)
	}
	t.byAddr[lvl.Addr] = lvl
	lm := t.levelMap(lvl)
	st.LevelNo, st.Room = lvl.No, room2
	st.LevelChanged = lvl.No != t.curNo
	t.cur, t.curNo = lm, lvl.No
	for _, n := range room2.Near {
		if rm, err := t.R.Room2(n); err == nil {
			st.NearRooms = append(st.NearRooms, rm)
		}
	}

	rescan := t.tick%max(t.RescanEvery, 1) == 0
	active := t.R.ActiveRoom1s(act, pc.Room1)
	st.ActiveRooms = len(active)
	fresh := map[uint32]bool{room1.Addr: true} // the player's room and its neighbours are re-read every tick
	for _, n := range room1.Near {
		fresh[n] = true
	}
	for _, rm := range active {
		if t.mergeRoom(rm, rescan || fresh[rm.Addr]) {
			st.NewRooms++
		}
	}
	t.scanStatic(lm, lvl)
	// Neighbouring levels that received rooms: keep their static data fresh once.
	for _, ln := range t.neighbourLevels(active) {
		t.scanStatic(t.levelMap(ln), ln)
	}
	return st, nil
}

func (t *Tracker) neighbourLevels(active []Room1) []Level {
	seen := map[int]bool{t.curNo: true}
	var out []Level
	for _, rm := range active {
		lr, ok := t.loaded[rm.Addr]
		if !ok || seen[lr.level] {
			continue
		}
		seen[lr.level] = true
		for _, l := range t.byAddr {
			if l.No == lr.level {
				out = append(out, l)
				break
			}
		}
	}
	return out
}

// mergeRoom reads one Room1's collision into its level's grid. Returns true when the room was
// newly merged (as opposed to skipped or merely refreshed).
func (t *Tracker) mergeRoom(rm Room1, rescan bool) bool {
	prev, had := t.loaded[rm.Addr]
	if had && prev.rect == rm.Rect && !rescan {
		return false
	}
	if !validPtr(rm.Coll) {
		return false
	}
	r2, err := t.R.Room2(rm.Room2)
	if err != nil {
		return false
	}
	lvl, err := t.levelAt(r2.Level)
	if err != nil {
		return false
	}
	cm, err := t.R.Coll(rm)
	if err != nil {
		return false
	}
	lm := t.levelMap(lvl)
	lm.Grid.SetBlock(cm.Rect, cm.Cells)
	t.loaded[rm.Addr] = loadedRoom{rm.Rect, lvl.No}
	return !had || prev.rect != rm.Rect
}

// scanStatic walks the level's Room2 list and (re)reads presets, exits and adjacency for rooms
// that are new or whose Room1 just appeared.
func (t *Tracker) scanStatic(lm *LevelMap, lvl Level) {
	rooms := t.R.LevelRoom2s(lvl)
	if len(rooms) == 0 {
		return
	}
	lm.Rooms = rooms
	dirty := false
	for _, rm := range rooms {
		// Presets and tiles only exist while the client has room data for the Room2, so
		// rescan whenever one of these pointers changes.
		sig := [3]uint32{rm.Room1, rm.Preset, rm.Tiles}
		if old, ok := lm.room2Sig[rm.Addr]; ok && old == sig {
			continue
		}
		lm.room2Sig[rm.Addr] = sig
		dirty = true
		// Static data only accumulates: the client frees a Room2's preset/tile/near data when the player moves away
		// and gives Room2s new addresses when a level is entered again, so "replace what this room lists" would
		// forget exits and borders seen earlier (found live on i1, 2026-10-28: back in the Blood Moor the Den
		// entrance and the town border were gone, nav said "no exit" and the explored grid had no frontier left).
		lm.Presets = mergePresets(lm.Presets, t.R.Presets(rm))
	}
	if !dirty {
		return
	}
	sortPresets(lm.Presets)
	adj := map[int]Rect{}
	for _, rm := range rooms {
		tiles := t.R.RoomTiles(rm)
		for _, p := range lm.Presets {
			if p.Kind != PresetTile || p.Room2 != rm.Addr {
				continue
			}
			for _, tl := range tiles {
				if int(tl.Num) != p.ID {
					continue
				}
				if d, err := t.R.Room2(tl.Dest); err == nil {
					if dl, err := t.levelAt(d.Level); err == nil {
						x := Exit{Dest: dl.No, Pos: p.Pos, Room2: rm.Addr}
						dup := false
						for i, o := range lm.Exits {
							if o.Dest == x.Dest && o.Pos == x.Pos {
								dup, lm.Exits[i].Room2 = true, x.Room2
							}
						}
						if !dup {
							lm.Exits = append(lm.Exits, x)
						}
					}
				}
				break
			}
		}
		for _, n := range rm.Near {
			if nr, err := t.R.Room2(n); err == nil && nr.Level != lvl.Addr {
				if nl, err := t.levelAt(nr.Level); err == nil && nl.No != lvl.No {
					adj[nl.No] = nl.Rect
				}
			}
		}
	}
	for no, rc := range adj {
		a := Adjacent{no, rc, contact(lvl.Rect, rc)}
		found := false
		for i := range lm.Adjacent {
			if lm.Adjacent[i].LevelNo == no {
				lm.Adjacent[i], found = a, true
			}
		}
		if !found {
			lm.Adjacent = append(lm.Adjacent, a)
		}
	}
	sort.Slice(lm.Adjacent, func(i, j int) bool { return lm.Adjacent[i].LevelNo < lm.Adjacent[j].LevelNo })
}

// contact is the strip of subtiles inside a, one subtile deep, that touches level rectangle b.
func contact(a, b Rect) Rect {
	x0, y0 := max(a.X, b.X-1), max(a.Y, b.Y-1)
	x1, y1 := min(a.Right(), b.Right()+1), min(a.Bottom(), b.Bottom()+1)
	if x1 <= x0 || y1 <= y0 {
		return Rect{}
	}
	return Rect{x0, y0, x1 - x0, y1 - y0}
}

// BorderCells returns the known walkable cells of this level that touch the given neighbouring
// level. Outdoor levels connect by walking across this border, not through a tile preset.
func (l *LevelMap) BorderCells(levelNo int) []Point {
	var out []Point
	for _, a := range l.Adjacent {
		if a.LevelNo != levelNo {
			continue
		}
		for y := a.Contact.Y; y < a.Contact.Bottom(); y++ {
			for x := a.Contact.X; x < a.Contact.Right(); x++ {
				if l.Rect.Contains(x, y) && l.Grid.Walkable(x, y) {
					out = append(out, Point{x, y})
				}
			}
		}
	}
	return out
}

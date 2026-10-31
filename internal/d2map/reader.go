package d2map

import (
	"encoding/binary"
	"errors"
	"fmt"

	"d2lod-bot/internal/mem"
)

// Rect is a half-open rectangle in absolute subtiles.
type Rect struct{ X, Y, W, H int }

func (r Rect) Right() int  { return r.X + r.W }
func (r Rect) Bottom() int { return r.Y + r.H }
func (r Rect) Empty() bool { return r.W <= 0 || r.H <= 0 }
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && y >= r.Y && x < r.Right() && y < r.Bottom()
}

// Point is an absolute subtile position.
type Point struct{ X, Y int }

type Act struct {
	Addr       uint32
	Seed       uint32
	Index      int
	Room1First uint32
	Misc       uint32
	LevelFirst uint32
}

type Level struct {
	Addr       uint32
	No         int
	Rect       Rect
	Next       uint32
	Room2First uint32
	Warps      []Point // UNVERIFIED semantics, see docs/map.md
}

type Room2 struct {
	Addr   uint32
	Rect   Rect
	Room1  uint32
	Level  uint32
	Next   uint32
	Near   []uint32
	Preset uint32
	Tiles  uint32
	Flags  uint32
}

type Room1 struct {
	Addr      uint32
	Room2     uint32
	Coll      uint32
	Rect      Rect
	Near      []uint32
	UnitFirst uint32
	Next      uint32
}

// CollMap is one room's collision grid: Cells[(y-Rect.Y)*Rect.W + (x-Rect.X)].
type CollMap struct {
	Addr  uint32
	Rect  Rect
	Cells []uint16
}

// Reader decodes the map structures. All methods are all-or-nothing and tolerate garbage:
// pointers are range-checked, counts are capped and results are consistency-checked.
type Reader struct{ S mem.Source }

var ErrBadPointer = errors.New("d2map: invalid pointer")

func (r Reader) block(addr uint32, n int) ([]byte, error) {
	if !validPtr(addr) {
		return nil, fmt.Errorf("%w 0x%x", ErrBadPointer, addr)
	}
	b := make([]byte, n)
	if err := r.S.ReadAt(b, addr); err != nil {
		return nil, err
	}
	return b, nil
}

func le32(b []byte, off int) uint32 { return binary.LittleEndian.Uint32(b[off:]) }
func ti(b []byte, off int) int      { return int(int32(le32(b, off))) }

// PlayerChain follows UnitAny -> pAct / pPath -> Path.pRoom1 and also returns the position.
type PlayerChain struct {
	Unit, Act, Path, Room1 uint32
	Pos                    Point
}

// Player reads the player unit and the pointers the map reader starts from.
func (r Reader) Player() (PlayerChain, error) {
	var c PlayerChain
	var err error
	if c.Unit, err = mem.U32(r.S, mem.Base+rvaPlayerUnit); err != nil {
		return c, err
	}
	if !validPtr(c.Unit) {
		return c, fmt.Errorf("%w: no player unit (menu or loading)", ErrBadPointer)
	}
	if c.Act, err = mem.U32(r.S, c.Unit+offUnitAct); err != nil {
		return c, err
	}
	if c.Path, err = mem.U32(r.S, c.Unit+offUnitPath); err != nil {
		return c, err
	}
	b, err := r.block(c.Path, 0x20)
	if err != nil {
		return c, err
	}
	c.Pos = Point{int(binary.LittleEndian.Uint16(b[offPathX:])), int(binary.LittleEndian.Uint16(b[offPathY:]))}
	c.Room1 = le32(b, offPathRoom1)
	return c, nil
}

func (r Reader) Act(addr uint32) (Act, error) {
	b, err := r.block(addr, actSize)
	if err != nil {
		return Act{}, err
	}
	a := Act{Addr: addr, Seed: le32(b, offActSeed), Index: ti(b, offActIndex), Room1First: le32(b, offActRoom1), Misc: le32(b, offActMisc)}
	if validPtr(a.Misc) {
		if m, err := r.block(a.Misc+offMiscLevel1, 4); err == nil {
			a.LevelFirst = le32(m, 0)
		}
	}
	return a, nil
}

func (r Reader) Level(addr uint32) (Level, error) {
	b, err := r.block(addr, levelSize)
	if err != nil {
		return Level{}, err
	}
	l := Level{Addr: addr, No: ti(b, offLvlNo), Next: le32(b, offLvlNext), Room2First: le32(b, offLvlRoom2First)}
	w, h := ti(b, offLvlSizeX), ti(b, offLvlSizeY)
	if l.No < 0 || l.No > 200 || w < 0 || h < 0 || w > maxLevelTiles || h > maxLevelTiles {
		return Level{}, fmt.Errorf("d2map: implausible level at 0x%x (no=%d size=%dx%d)", addr, l.No, w, h)
	}
	l.Rect = Rect{ti(b, offLvlPosX) * 5, ti(b, offLvlPosY) * 5, w * 5, h * 5}
	for i := 0; i < 9; i++ {
		x, y := ti(b, offLvlWarpX+4*i), ti(b, offLvlWarpY+4*i)
		if x != 0 || y != 0 {
			l.Warps = append(l.Warps, Point{x, y})
		}
	}
	return l, nil
}

// Levels walks Act -> ActMisc -> Level list (cycle-safe).
func (r Reader) Levels(a Act) []Level {
	var out []Level
	seen := map[uint32]bool{}
	for p := a.LevelFirst; validPtr(p) && !seen[p] && len(out) < maxList; {
		seen[p] = true
		l, err := r.Level(p)
		if err != nil {
			break
		}
		out = append(out, l)
		p = l.Next
	}
	return out
}

func (r Reader) ptrArray(addr uint32, n int) []uint32 {
	if n <= 0 || n > maxNear || !validPtr(addr) {
		return nil
	}
	b, err := r.block(addr, 4*n)
	if err != nil {
		return nil
	}
	out := make([]uint32, 0, n)
	for i := 0; i < n; i++ {
		if p := le32(b, 4*i); validPtr(p) {
			out = append(out, p)
		}
	}
	return out
}

func (r Reader) Room2(addr uint32) (Room2, error) {
	b, err := r.block(addr, room2Size)
	if err != nil {
		return Room2{}, err
	}
	w, h := ti(b, offR2SizeX), ti(b, offR2SizeY)
	if w <= 0 || h <= 0 || w > maxLevelDim || h > maxLevelDim {
		return Room2{}, fmt.Errorf("d2map: implausible room2 at 0x%x (size %dx%d)", addr, w, h)
	}
	return Room2{
		Addr:   addr,
		Rect:   Rect{ti(b, offR2PosX) * 5, ti(b, offR2PosY) * 5, w * 5, h * 5},
		Room1:  le32(b, offR2Room1),
		Level:  le32(b, offR2Level),
		Next:   le32(b, offR2Next),
		Near:   r.ptrArray(le32(b, offR2Near), ti(b, offR2NearCount)),
		Preset: le32(b, offR2Preset),
		Tiles:  le32(b, offR2Tiles),
		Flags:  le32(b, offR2Flags),
	}, nil
}

// LevelRoom2s walks a level's static Room2 list (cycle-safe).
func (r Reader) LevelRoom2s(l Level) []Room2 {
	var out []Room2
	seen := map[uint32]bool{}
	for p := l.Room2First; validPtr(p) && !seen[p] && len(out) < maxList; {
		seen[p] = true
		rm, err := r.Room2(p)
		if err != nil {
			break
		}
		out = append(out, rm)
		p = rm.Next
	}
	return out
}

func (r Reader) Room1(addr uint32) (Room1, error) {
	b, err := r.block(addr, room1Size)
	if err != nil {
		return Room1{}, err
	}
	w, h := ti(b, offR1XSize), ti(b, offR1YSize)
	if w < 0 || h < 0 || w > maxCollDim || h > maxCollDim {
		return Room1{}, fmt.Errorf("d2map: implausible room1 at 0x%x (size %dx%d)", addr, w, h)
	}
	return Room1{
		Addr:      addr,
		Room2:     le32(b, offR1Room2),
		Coll:      le32(b, offR1Coll),
		Rect:      Rect{ti(b, offR1XStart), ti(b, offR1YStart), w, h},
		Near:      r.ptrArray(le32(b, offR1Near), ti(b, offR1NearCount)),
		UnitFirst: le32(b, offR1UnitFirst),
		Next:      le32(b, offR1Next),
	}, nil
}

// ActiveRoom1s returns every room the client currently has active: the Act's Room1 list, the
// given extra rooms (e.g. the player's) and their pRoomsNear neighbours, de-duplicated.
// Cycle-safe and bounded; unreadable rooms are skipped.
func (r Reader) ActiveRoom1s(a Act, extra ...uint32) []Room1 {
	var out []Room1
	seen := map[uint32]bool{}
	add := func(p uint32) (Room1, bool) {
		if !validPtr(p) || seen[p] || len(out) >= maxList {
			return Room1{}, false
		}
		seen[p] = true
		rm, err := r.Room1(p)
		if err != nil {
			return Room1{}, false
		}
		out = append(out, rm)
		return rm, true
	}
	for p := a.Room1First; ; {
		rm, ok := add(p)
		if !ok {
			break
		}
		p = rm.Next
	}
	for _, p := range extra {
		for {
			rm, ok := add(p)
			if !ok {
				break
			}
			p = rm.Next
		}
	}
	for i := 0; i < len(out); i++ {
		for _, n := range out[i].Near {
			add(n)
		}
	}
	return out
}

// Coll reads a Room1's collision map. The header is read before and after the cells and must
// agree (torn-read guard), and it must match the Room1's own rectangle.
func (r Reader) Coll(rm Room1) (CollMap, error) {
	hdr := func() ([]byte, error) { return r.block(rm.Coll, collSize) }
	b, err := hdr()
	if err != nil {
		return CollMap{}, err
	}
	rect := Rect{ti(b, offCollX), ti(b, offCollY), ti(b, offCollW), ti(b, offCollH)}
	if rect.W <= 0 || rect.H <= 0 || rect.W > maxCollDim || rect.H > maxCollDim {
		return CollMap{}, fmt.Errorf("d2map: implausible collmap %+v", rect)
	}
	if rect != rm.Rect {
		return CollMap{}, fmt.Errorf("d2map: collmap %+v disagrees with room1 %+v", rect, rm.Rect)
	}
	n := rect.W * rect.H
	raw, err := r.block(le32(b, offCollStart), 2*n)
	if err != nil {
		return CollMap{}, err
	}
	b2, err := hdr()
	if err != nil {
		return CollMap{}, err
	}
	if string(b) != string(b2) {
		return CollMap{}, errors.New("d2map: torn collmap read")
	}
	cells := make([]uint16, n)
	for i := range cells {
		cells[i] = binary.LittleEndian.Uint16(raw[2*i:])
	}
	return CollMap{Addr: rm.Coll, Rect: rect, Cells: cells}, nil
}

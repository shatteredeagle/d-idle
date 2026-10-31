package d2map

import (
	"encoding/binary"
	"testing"

	"d2lod-bot/internal/mem"
)

// world builds a fake client memory image with the real struct layout.
type world struct {
	t    *testing.T
	f    *mem.Fake
	next uint32
	act  uint32
	misc uint32
	lvls []uint32

	blocks   map[uint32][]byte
	pathAddr uint32
}

func newWorld(t *testing.T) *world {
	w := &world{t: t, f: mem.NewFake(), next: 0x01000000, blocks: map[uint32][]byte{}}
	w.act = w.alloc(actSize)
	w.misc = w.alloc(miscSize)
	w.set(w.act, offActSeed, 0x1234)
	w.set(w.act, offActMisc, w.misc)
	return w
}

func (w *world) alloc(n int) uint32 {
	a := w.next
	w.next += uint32(n+0xF) &^ 0xF
	w.blocks[a] = make([]byte, n)
	w.f.Put(a, w.blocks[a])
	return a
}

// set patches one dword inside an allocated block (Fake only serves whole-segment reads, so the
// block is re-Put after every change).
func (w *world) set(base uint32, off int, v uint32) {
	binary.LittleEndian.PutUint32(w.blocks[base][off:], v)
	w.f.Put(base, w.blocks[base])
}

func (w *world) set16(base uint32, off int, v uint16) {
	binary.LittleEndian.PutUint16(w.blocks[base][off:], v)
	w.f.Put(base, w.blocks[base])
}

func (w *world) get(base uint32, off int) uint32 {
	return binary.LittleEndian.Uint32(w.blocks[base][off:])
}

func (w *world) level(no int, tx, ty, tw, th int) uint32 {
	a := w.alloc(levelSize)
	w.set(a, offLvlNo, uint32(no))
	w.set(a, offLvlPosX, uint32(tx))
	w.set(a, offLvlPosY, uint32(ty))
	w.set(a, offLvlSizeX, uint32(tw))
	w.set(a, offLvlSizeY, uint32(th))
	if n := len(w.lvls); n > 0 {
		w.set(w.lvls[n-1], offLvlNext, a)
	} else {
		w.set(w.misc, offMiscLevel1, a)
	}
	w.lvls = append(w.lvls, a)
	return a
}

// room2 appends a Room2 (tile units) to the level's list.
func (w *world) room2(level uint32, tx, ty, tw, th int) uint32 {
	a := w.alloc(room2Size)
	w.set(a, offR2PosX, uint32(tx))
	w.set(a, offR2PosY, uint32(ty))
	w.set(a, offR2SizeX, uint32(tw))
	w.set(a, offR2SizeY, uint32(th))
	w.set(a, offR2Level, level)
	if first := w.get(level, offLvlRoom2First); first == 0 {
		w.set(level, offLvlRoom2First, a)
	} else {
		last := first
		for n := w.get(last, offR2Next); n != 0; n = w.get(last, offR2Next) {
			last = n
		}
		w.set(last, offR2Next, a)
	}
	return a
}

// room1 creates an active room for room2 with the given collision rows (subtile rect r).
func (w *world) room1(r2 uint32, r Rect, cells []uint16) uint32 {
	a := w.alloc(room1Size)
	c := w.alloc(collSize)
	w.set(a, offR1Room2, r2)
	w.set(a, offR1Coll, c)
	w.set(a, offR1XStart, uint32(r.X))
	w.set(a, offR1YStart, uint32(r.Y))
	w.set(a, offR1XSize, uint32(r.W))
	w.set(a, offR1YSize, uint32(r.H))
	w.set(r2, offR2Room1, a)
	data := w.alloc(2 * len(cells))
	for i, v := range cells {
		binary.LittleEndian.PutUint16(w.blocks[data][2*i:], v)
	}
	w.f.Put(data, w.blocks[data])
	w.set(c, offCollX, uint32(r.X))
	w.set(c, offCollY, uint32(r.Y))
	w.set(c, offCollW, uint32(r.W))
	w.set(c, offCollH, uint32(r.H))
	w.set(c, offCollStart, data)
	// append to the act's active list
	if head := w.get(w.act, offActRoom1); head == 0 {
		w.set(w.act, offActRoom1, a)
	} else {
		last := head
		for n := w.get(last, offR1Next); n != 0; n = w.get(last, offR1Next) {
			last = n
		}
		w.set(last, offR1Next, a)
	}
	return a
}

func (w *world) player(x, y int, room1 uint32) {
	unit := w.alloc(0x40)
	path := w.alloc(0x30)
	w.set(unit, offUnitAct, w.act)
	w.set(unit, offUnitPath, path)
	w.set16(path, offPathX, uint16(x))
	w.set16(path, offPathY, uint16(y))
	w.set(path, offPathRoom1, room1)
	w.f.PutU32(mem.Base+rvaPlayerUnit, unit)
	w.pathAddr = path
}

func (w *world) movePlayer(x, y int, room1 uint32) {
	w.set16(w.pathAddr, offPathX, uint16(x))
	w.set16(w.pathAddr, offPathY, uint16(y))
	w.set(w.pathAddr, offPathRoom1, room1)
}

func (w *world) preset(r2 uint32, kind, id, px, py int) uint32 {
	a := w.alloc(presetSize)
	w.set(a, offPreType, uint32(kind))
	w.set(a, offPreTxt, uint32(id))
	w.set(a, offPrePosX, uint32(px))
	w.set(a, offPrePosY, uint32(py))
	w.set(a, offPreNext, w.get(r2, offR2Preset))
	w.set(r2, offR2Preset, a)
	return a
}

func (w *world) tile(r2, dest2 uint32, num int) {
	a := w.alloc(tileSize)
	n := w.alloc(4)
	w.set(n, 0, uint32(num))
	w.set(a, offTileRoom2, dest2)
	w.set(a, offTileNum, n)
	w.set(a, offTileNext, w.get(r2, offR2Tiles))
	w.set(r2, offR2Tiles, a)
}

func (w *world) near2(r2 uint32, others ...uint32) {
	arr := w.alloc(4 * len(others))
	for i, o := range others {
		binary.LittleEndian.PutUint32(w.blocks[arr][4*i:], o)
	}
	w.f.Put(arr, w.blocks[arr])
	w.set(r2, offR2Near, arr)
	w.set(r2, offR2NearCount, uint32(len(others)))
}

func fill(w, h int, v uint16) []uint16 {
	c := make([]uint16, w*h)
	for i := range c {
		c[i] = v
	}
	return c
}

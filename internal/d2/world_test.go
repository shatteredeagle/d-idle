package d2

import (
	"encoding/binary"

	"d2lod-bot/internal/mem"
)

// world builds a synthetic game image in a mem.Fake using the layouts documented in docs/mem.md.
type world struct {
	f    *mem.Fake
	next uint32
	mem  map[uint32][]byte
}

func newWorld() *world {
	w := &world{f: mem.NewFake(), next: 0x03000000, mem: map[uint32][]byte{}}
	for _, r := range []struct{ a, n uint32 }{
		{mem.Base + rvaServerUnits, 0xC00}, {mem.Base + rvaClientUnits, 0xC00}, {mem.Base + rvaUIFlags, uiCount * 4},
	} {
		w.mem[r.a] = make([]byte, r.n)
		w.f.Put(r.a, w.mem[r.a])
	}
	return w
}

// alloc returns a zeroed block registered with the fake; later writes go through set.
func (w *world) alloc(n int) uint32 {
	a := w.next
	w.next += uint32(n+0xF) &^ 0xF
	w.mem[a] = make([]byte, n)
	w.f.Put(a, w.mem[a])
	return a
}

func (w *world) set(addr uint32, off uint32, vals ...uint32) {
	a := addr + off
	for blk, b := range w.mem {
		if a >= blk && a+uint32(len(vals)*4) <= blk+uint32(len(b)) {
			for i, v := range vals {
				binary.LittleEndian.PutUint32(b[a-blk+uint32(i*4):], v)
			}
			w.f.Put(blk, b)
			return
		}
	}
	// static RVA area: allocate on demand
	b := make([]byte, len(vals)*4)
	for i, v := range vals {
		binary.LittleEndian.PutUint32(b[i*4:], v)
	}
	w.f.Put(a, b)
}

func (w *world) set16(addr, off uint32, vals ...uint16) {
	a := addr + off
	for blk, b := range w.mem {
		if a >= blk && a+uint32(len(vals)*2) <= blk+uint32(len(b)) {
			for i, v := range vals {
				binary.LittleEndian.PutUint16(b[a-blk+uint32(i*2):], v)
			}
			w.f.Put(blk, b)
			return
		}
	}
	panic("set16 outside block")
}

func (w *world) setBytes(addr, off uint32, bs []byte) {
	a := addr + off
	for blk, b := range w.mem {
		if a >= blk && a+uint32(len(bs)) <= blk+uint32(len(b)) {
			copy(b[a-blk:], bs)
			w.f.Put(blk, b)
			return
		}
	}
	panic("setBytes outside block")
}

type statKV struct {
	id  uint32 // sub<<16 | stat id
	val int32
}

// statList builds a StatList with a base vector and, if ex, an effective vector (flag 0x80000000).
func (w *world) statList(base, eff []statKV, ex bool) uint32 {
	mk := func(kv []statKV) (uint32, uint16) {
		if len(kv) == 0 {
			return w.alloc(8), 0
		}
		v := w.alloc(len(kv) * 8)
		for i, s := range kv {
			w.set16(v, uint32(i*8), uint16(s.id>>16), uint16(s.id))
			w.set(v, uint32(i*8+4), uint32(s.val))
		}
		return v, uint16(len(kv))
	}
	sl := w.alloc(0x60)
	bv, bc := mk(base)
	w.set(sl, slVec, bv)
	w.set16(sl, slCount, bc, bc)
	if ex {
		w.set(sl, slFlags, slFlagEx)
		ev, ec := mk(eff)
		w.set(sl, slVecEx, ev)
		w.set16(sl, slCountEx, ec, ec)
	}
	return sl
}

// unit allocates a UnitAny with a Path.
func (w *world) unit(typ, class, id, mode uint32, x, y int) uint32 {
	u := w.alloc(0x100)
	w.set(u, uoType, typ, class)
	w.set(u, uoID, id, mode)
	path := w.alloc(0x40)
	w.set(u, uoPath, path)
	if typ == TypeObject || typ == TypeItem || typ == TypeTile {
		w.set(path, 0x0C, uint32(x), uint32(y))
	} else {
		w.set16(path, 2, uint16(x))
		w.set16(path, 6, uint16(y))
	}
	return u
}

func (w *world) addToTable(typ int, base uint32, u uint32) {
	id := binary.LittleEndian.Uint32(w.mem[u][uoID:])
	slot := base + uint32(typ)*0x200 + (id&0x7f)*4
	head, _ := mem.U32(w.f, slot)
	w.set(u, uoListNext, head)
	w.set(slot, 0, u)
}

// areaChain wires path -> room1 -> room2 -> level for a unit.
func (w *world) areaChain(u uint32, level uint32) {
	path, _ := mem.U32(w.f, u+uoPath)
	room1, room2, lvl := w.alloc(0x80), w.alloc(0x80), w.alloc(0x200)
	w.set(path, 0x1C, room1)
	w.set(room1, 0x10, room2)
	w.set(room2, 0x58, lvl)
	w.set(lvl, 0x1D0, level)
}

type fixture struct {
	*world
	player uint32
	inv    uint32
}

const (
	absPlayer = mem.Base + rvaPlayerUnit
	absFirst  = mem.Base + rvaFirstControl
	absSrv    = mem.Base + rvaServerUnits
	absUI     = mem.Base + rvaUIFlags
)

// buildInGame makes a level-1 Sorceress "Testy" in the Rogue Encampment, mirroring the live recording.
func buildInGame() *fixture {
	w := newWorld()
	fx := &fixture{world: w}
	p := w.unit(TypePlayer, 1, 1, 5, 6053, 4963)
	fx.player = p
	w.areaChain(p, 1)
	w.set(p, uoAct, 0)
	pd := w.alloc(0x80)
	w.setBytes(pd, 0, []byte("Testy"))
	w.set(p, uoData, pd)
	base := []statKV{{0, 10}, {1, 35}, {2, 25}, {3, 10}, {6, 10240}, {7, 10240}, {8, 8960}, {9, 8960}, {10, 18944}, {11, 18944}, {12, 1}}
	eff := append(append([]statKV{}, base...), statKV{67, 100}, statKV{68, 110}, statKV{69, 100}, statKV{36<<16 | 107, 1})
	w.set(p, uoStats, w.statList(base, eff, true))
	// skills: attack(0) selected left, fire bolt (36) right
	info := w.alloc(0x20)
	mkSkill := func(id uint16, lvl uint32) uint32 {
		sk, si := w.alloc(0x40), w.alloc(0x20)
		w.set16(si, 0, id)
		w.set(sk, 0, si)
		w.set(sk, skLevel, lvl)
		return sk
	}
	s0, s1, s36 := mkSkill(0, 1), mkSkill(1, 1), mkSkill(36, 0)
	w.set(s0, skNext, s1)
	w.set(s1, skNext, s36)
	w.set(info, infoFirst, s0, s0, s36)
	w.set(p, uoInfo, info)
	// inventory: staff equipped, 2 belt potions, tome in inventory
	inv := w.alloc(0x40)
	fx.inv = inv
	w.set(inv, 0x08, p)
	w.set(p, uoInventory, inv)
	item := func(class, id, mode, quality, flags uint32, body, page, node byte, gx, gy int) uint32 {
		u := w.unit(TypeItem, class, id, mode, gx, gy)
		d := w.alloc(0x90)
		w.set(d, idQuality, quality)
		w.set(d, idFlags, flags)
		w.set(d, idILvl, 1)
		w.setBytes(d, idBody, []byte{body, page})
		w.setBytes(d, idNodePage, []byte{node})
		w.set(u, uoData, d)
		w.addToTable(TypeItem, absSrv, u)
		return u
	}
	staff := item(63, 1, ItemModeEquipped, QualityNormal, 0x822011, 4, 0xff, 3, 0, 0)
	pot1 := item(587, 2, ItemModeBelt, QualityNormal, 0xa22010, 0, 0xff, 2, 0, 0)
	pot2 := item(587, 3, ItemModeBelt, QualityNormal, 0xa22010, 0, 0xff, 2, 1, 0)
	tome := item(529, 6, ItemModeStored, QualityNormal, 0xa22010, 0, 0, 1, 9, 3)
	chain := []uint32{staff, pot1, pot2, tome}
	for i := 0; i+1 < len(chain); i++ {
		d, _ := mem.U32(w.f, chain[i]+uoData)
		w.set(d, idNextInv, chain[i+1])
	}
	w.set(inv, 0x0C, staff, tome)
	w.set(inv, 0x28, 4)
	// NPCs / objects
	for i, c := range []struct {
		class, id uint32
		x, y      int
	}{{152, 2, 6068, 4968}, {150, 9, 6048, 4942}} {
		m := w.unit(TypeMonster, c.class, c.id, 1, c.x, c.y)
		md := w.alloc(0x60)
		w.set(m, uoData, md)
		w.set(m, uoStats, w.statList([]statKV{{6, 0x8000}, {7, 0x8000}}, []statKV{{6, 0x8000}, {7, 0x8000}, {67, 75}}, true))
		w.addToTable(TypeMonster, absSrv, m)
		_ = i
	}
	o := w.unit(TypeObject, 37, 2, 2, 6052, 4964)
	w.addToTable(TypeObject, absSrv, o)
	// UI: only the always-on flags (live baseline)
	for _, i := range []uint32{0, 21, 34, 35} {
		w.set(absUI, i*4, 1)
	}
	w.set(absPlayer, 0, p)
	w.set(absFirst, 0, 0)
	w.set(mem.Base+rvaScreenSizeX, 0, 800, 600)
	w.set(mem.Base+rvaMouseY, 0, 164)
	w.set(mem.Base+rvaMouseX, 0, 304)
	w.set(mem.Base+rvaViewportX, 0, 17216)
	w.set(mem.Base+rvaViewportY, 0, 88004)
	w.set(mem.Base+rvaSelActive, 0, 0)
	w.set(mem.Base+rvaSelID, 0, 9)
	w.set(mem.Base+rvaSelType, 0, 1)
	w.set(mem.Base+rvaDifficulty, 0, 0)
	return fx
}

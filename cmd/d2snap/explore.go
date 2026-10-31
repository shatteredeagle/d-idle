package main

import (
	"encoding/binary"
	"fmt"

	"d2lod-bot/internal/mem"
)

// Exploration helpers: print raw structures so layouts can be checked by eye (docs/mem.md).

func u32(s mem.Source, a uint32) uint32 { v, _ := mem.U32(s, a); return v }
func u16(s mem.Source, a uint32) uint16 { v, _ := mem.U16(s, a); return v }

func hexdump(s mem.Source, a uint32, n int) {
	b := make([]byte, n)
	if err := s.ReadAt(b, a); err != nil {
		fmt.Printf("  %08x: ERR %v\n", a, err)
		return
	}
	for i := 0; i < n; i += 16 {
		fmt.Printf("  %08x:", a+uint32(i))
		for j := i; j < i+16 && j < n; j += 4 {
			fmt.Printf(" %08x", binary.LittleEndian.Uint32(b[j:]))
		}
		fmt.Println()
	}
}

func dumpStatList(s mem.Source, sl uint32, tag string) {
	fmt.Printf("%s statlist 0x%x\n", tag, sl)
	hexdump(s, sl, 0x68)
	vec := u32(s, sl+0x24)
	cnt := int(u16(s, sl+0x28))
	for i := 0; i < cnt && i < 200; i++ {
		sub := u16(s, vec+uint32(i)*8)
		idx := u16(s, vec+uint32(i)*8+2)
		val := int32(u32(s, vec+uint32(i)*8+4))
		fmt.Printf("    stat %3d sub %d = %d (0x%x)\n", idx, sub, val, uint32(val))
	}
}

func dumpMerged(s mem.Source, sl uint32) {
	if u32(s, sl+0x10)&0x80000000 == 0 {
		return
	}
	vec := u32(s, sl+0x48)
	cnt := int(u16(s, sl+0x4C))
	fmt.Printf("  merged vec 0x%x count %d\n", vec, cnt)
	for i := 0; i < cnt && i < 200; i++ {
		fmt.Printf("    m-stat %3d sub %d = %d\n", u16(s, vec+uint32(i)*8+2), u16(s, vec+uint32(i)*8), int32(u32(s, vec+uint32(i)*8+4)))
	}
}

func exploreItemTxt(s mem.Source, classes ...uint32) {
	cnt := u32(s, 0x96ca58)
	tbl := u32(s, 0x96ca5c)
	fmt.Printf("itemtxt count=%d table=0x%x\n", cnt, tbl)
	for _, c := range classes {
		row := tbl + c*0x1A8
		code, _ := s2(s, row+0x40, 4)
		b := make([]byte, 2)
		_ = s.ReadAt(b, row+0xFC)
		fmt.Printf("  class %d code=%q size=%dx%d\n", c, code, b[0], b[1])
	}
}

func s2(s mem.Source, a uint32, n int) (string, error) {
	b := make([]byte, n)
	err := s.ReadAt(b, a)
	return string(b), err
}

func exploreStats(s mem.Source) {
	pl := u32(s, mem.Base+0x3A6A70)
	sl := u32(s, pl+0x5C)
	dumpStatList(s, sl, "player")
	dumpMerged(s, sl)
	for _, off := range []uint32{0x2C, 0x34, 0x3C, 0x40, 0x44} {
		p := u32(s, sl+off)
		if p > 0x10000 {
			dumpStatList(s, p, fmt.Sprintf("player+0x%x", off))
		}
	}
}

func exploreInv(s mem.Source) {
	pl := u32(s, mem.Base+0x3A6A70)
	inv := u32(s, pl+0x60)
	fmt.Printf("inventory 0x%x\n", inv)
	hexdump(s, inv, 0x40)
	it := u32(s, inv+0x0C)
	for n := 0; it > 0x10000 && n < 100; n++ {
		fmt.Printf("ITEM unit 0x%x type=%d class=%d id=0x%x mode=%d\n", it, u32(s, it), u32(s, it+4), u32(s, it+0xC), u32(s, it+0x10))
		hexdump(s, it, 0x100)
		id := u32(s, it+0x14)
		fmt.Printf("  itemdata 0x%x\n", id)
		hexdump(s, id, 0x90)
		path := u32(s, it+0x2C)
		fmt.Printf("  path 0x%x\n", path)
		hexdump(s, path, 0x40)
		it = u32(s, u32(s, it+0x14)+0x64)
	}
	fmt.Println("cursor item:", u32(s, inv+0x20), "count:", u32(s, inv+0x28))
}

func exploreSkills(s mem.Source) {
	pl := u32(s, mem.Base+0x3A6A70)
	info := u32(s, pl+0xA8)
	fmt.Printf("info 0x%x\n", info)
	hexdump(s, info, 0x20)
	for sk := u32(s, info+4); sk > 0x10000; sk = u32(s, sk+4) {
		si := u32(s, sk)
		fmt.Printf("skill 0x%x id=%d lvl=%d (selected? left=%v right=%v)\n", sk, u16(s, si), u32(s, sk+0x28), sk == u32(s, info+8), sk == u32(s, info+0xC))
	}
}

func explorePlayer(s mem.Source) {
	pl := u32(s, mem.Base+0x3A6A70)
	fmt.Printf("player 0x%x\n", pl)
	hexdump(s, pl, 0x100)
	pd := u32(s, pl+0x14)
	fmt.Printf("playerdata 0x%x\n", pd)
	hexdump(s, pd, 0x80)
	path := u32(s, pl+0x2C)
	fmt.Printf("path 0x%x\n", path)
	hexdump(s, path, 0x80)
	info := u32(s, pl+0xA8)
	for sk := u32(s, info+4); sk > 0x10000; sk = u32(s, sk+4) {
		fmt.Printf("skill 0x%x id=%d\n", sk, u16(s, u32(s, sk)))
		hexdump(s, sk, 0x40)
	}
}

// exploreUnits dumps raw UnitAny + type data + stat lists for every unit of one type.
func exploreUnits(s mem.Source, typ uint32, max int) {
	tbl := mem.Base + 0x3A5E70 + typ*0x200
	n := 0
	for i := uint32(0); i < 128; i++ {
		for u := u32(s, tbl+i*4); u > 0x10000 && n < max; u = u32(s, u+0xE4) {
			n++
			fmt.Printf("UNIT type=%d class=%d id=0x%x mode=%d at 0x%x\n", u32(s, u), u32(s, u+4), u32(s, u+0xC), u32(s, u+0x10), u)
			hexdump(s, u, 0xF0)
			d := u32(s, u+0x14)
			fmt.Printf("  data 0x%x\n", d)
			hexdump(s, d, 0x80)
			path := u32(s, u+0x2C)
			fmt.Printf("  path 0x%x\n", path)
			hexdump(s, path, 0x30)
			if sl := u32(s, u+0x5C); sl > 0x10000 {
				dumpStatList(s, sl, "  unit")
				dumpMerged(s, sl)
			}
		}
	}
}

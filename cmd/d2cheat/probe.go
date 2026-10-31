package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
	"strings"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/mem"
)

// probe is read-only research: it finds every player-like UnitAny in the process (the client copy and,
// in single player, the game server's copy) so the cheat package knows which one to write.
func probe(p *mem.Proc) {
	r := d2.New(p)
	pl, err := r.Player()
	if err != nil {
		fmt.Println("player:", err)
		return
	}
	fmt.Printf("client player 0x%x id=%d name=%s class=%v lvl=%d\n", pl.Addr, pl.ID, pl.Name, pl.Class, pl.Level)
	// server table bucket
	tbl := mem.Base + 0x3A5E70
	for t := uint32(0); t < 1; t++ {
		for b := uint32(0); b < 128; b++ {
			u, _ := mem.U32(p, tbl+t*0x200+b*4)
			for n := 0; u > 0x10000 && n < 8; n++ {
				id, _ := mem.U32(p, u+0xC)
				ty, _ := mem.U32(p, u)
				fmt.Printf("  table[%d][%d] unit 0x%x type=%d id=%d stats=0x%x\n", t, b, u, ty, id, u32(p, u+0x5C))
				u, _ = mem.U32(p, u+0xE4)
			}
		}
	}
	f, _ := os.Open(fmt.Sprintf("/proc/%d/maps", p.PID))
	sc := bufio.NewScanner(f)
	var found []uint32
	for sc.Scan() {
		fs := strings.Fields(sc.Text())
		if len(fs) < 2 || !strings.HasPrefix(fs[1], "rw") {
			continue
		}
		var lo, hi uint64
		fmt.Sscanf(fs[0], "%x-%x", &lo, &hi)
		if hi > 0x100000000 || lo < 0x10000 || hi-lo > 0x20000000 {
			continue
		}
		buf := make([]byte, hi-lo)
		if err := p.ReadAt(buf, uint32(lo)); err != nil {
			continue
		}
		for o := 0; o+0xF0 <= len(buf); o += 4 {
			if binary.LittleEndian.Uint32(buf[o:]) == 0 && binary.LittleEndian.Uint32(buf[o+4:]) == uint32(pl.Class) &&
				binary.LittleEndian.Uint32(buf[o+0xC:]) == pl.ID && binary.LittleEndian.Uint32(buf[o+0x5C:]) > 0x10000 && binary.LittleEndian.Uint32(buf[o+0x10:]) < 20 {
				found = append(found, uint32(lo)+uint32(o))
			}
		}
	}
	for _, a := range found {
		sl := u32(p, a+0x5C)
		if u32(p, sl+0x10)&0x80000000 == 0 {
			continue
		}
		if u32(p, u32(p, a+0x5C)+0x10)&0x80000000 != 0 {
			dumpVec(p, fmt.Sprintf("unit 0x%x base", a), u32(p, a+0x5C)+0x24, u32(p, a+0x5C)+0x28)
			dumpVec(p, fmt.Sprintf("unit 0x%x merged", a), u32(p, a+0x5C)+0x48, u32(p, a+0x5C)+0x4C)
		}
		fmt.Printf("candidate unit 0x%x statlist=0x%x flags=0x%x vec24=0x%x/%d vec48=0x%x/%d data=0x%x\n", a, sl, u32(p, sl+0x10), u32(p, sl+0x24), u16(p, sl+0x28), u32(p, sl+0x48), u16(p, sl+0x4C), u32(p, a+0x14))
	}
}

func u32(s mem.Source, a uint32) uint32 { v, _ := mem.U32(s, a); return v }
func u16(s mem.Source, a uint32) uint16 { v, _ := mem.U16(s, a); return v }

// refs lists every aligned dword in rw mappings equal to any of the targets.
func refs(p *mem.Proc, targets ...uint32) {
	f, _ := os.Open(fmt.Sprintf("/proc/%d/maps", p.PID))
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fs := strings.Fields(sc.Text())
		if len(fs) < 2 || !strings.HasPrefix(fs[1], "rw") {
			continue
		}
		var lo, hi uint64
		fmt.Sscanf(fs[0], "%x-%x", &lo, &hi)
		if hi > 0x100000000 || lo < 0x10000 || hi-lo > 0x20000000 {
			continue
		}
		buf := make([]byte, hi-lo)
		if p.ReadAt(buf, uint32(lo)) != nil {
			continue
		}
		for o := 0; o+4 <= len(buf); o += 4 {
			v := binary.LittleEndian.Uint32(buf[o:])
			for _, t := range targets {
				if v == t {
					fmt.Printf("ref to 0x%x at 0x%x (map %s)\n", t, uint32(lo)+uint32(o), fs[0])
				}
			}
		}
	}
}

func dumpVec(p *mem.Proc, tag string, vecAt, cntAt uint32) {
	vec := u32(p, vecAt)
	cnt := int(u16(p, cntAt))
	fmt.Printf("%s vec=0x%x count=%d cap=%d\n", tag, vec, cnt, u16(p, cntAt+2))
	for i := 0; i < cnt; i++ {
		fmt.Printf("   [%02d] @0x%x id=%d sub=%d val=%d\n", i, vec+uint32(i)*8, u16(p, vec+uint32(i)*8+2), u16(p, vec+uint32(i)*8), int32(u32(p, vec+uint32(i)*8+4)))
	}
}

func dumpBoth(p *mem.Proc, client, server uint32) {
	for _, u := range []struct {
		n string
		a uint32
	}{{"client", client}, {"server", server}} {
		sl := u32(p, u.a+0x5C)
		fmt.Printf("== %s unit 0x%x statlist 0x%x\n", u.n, u.a, sl)
		dumpVec(p, u.n+" base", sl+0x24, sl+0x28)
		dumpVec(p, u.n+" merged", sl+0x48, sl+0x4C)
	}
}

func refsCmd(p *mem.Proc, args []string) {
	var ts []uint32
	for _, a := range args {
		var v uint32
		fmt.Sscanf(a, "%x", &v)
		ts = append(ts, v)
	}
	refs(p, ts...)
}

// seqCmd finds a run of consecutive dwords (read-only research: locates game tables in memory).
func seqCmd(p *mem.Proc, args []string) {
	var want []uint32
	for _, a := range args {
		var v uint32
		fmt.Sscanf(a, "%d", &v)
		want = append(want, v)
	}
	f, _ := os.Open(fmt.Sprintf("/proc/%d/maps", p.PID))
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fs := strings.Fields(sc.Text())
		if len(fs) < 2 || !strings.HasPrefix(fs[1], "r") {
			continue
		}
		var lo, hi uint64
		fmt.Sscanf(fs[0], "%x-%x", &lo, &hi)
		if hi > 0x100000000 || lo < 0x10000 || hi-lo > 0x20000000 {
			continue
		}
		buf := make([]byte, hi-lo)
		if p.ReadAt(buf, uint32(lo)) != nil {
			continue
		}
	next:
		for o := 0; o+4*len(want) <= len(buf); o += 4 {
			for i, w := range want {
				if binary.LittleEndian.Uint32(buf[o+4*i:]) != w {
					continue next
				}
			}
			fmt.Printf("sequence at 0x%x (map %s)\n", uint32(lo)+uint32(o), fs[0])
		}
	}
}

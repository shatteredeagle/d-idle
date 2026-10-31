package main

import (
	"fmt"

	"d2lod-bot/internal/mem"
)

// Memory layout used below: the roster (PlayerUnitList) at RVA 0x3BB5C0, struct from D2BS D2Structs.h (R);
// every field printed here is recorded in docs/net/multiplayer.md with its live reading.
const rvaRoster = 0x3BB5C0

// players prints the player units of the server and client hash tables and the roster (the party panel's
// list), so two clients in one game can be compared.
func players(src mem.Source) error {
	if src == nil {
		return fmt.Errorf("no game")
	}
	for _, t := range []struct {
		name string
		base uint32
	}{{"server", 0x3A5E70}, {"client", 0x3A5270}} {
		n, shown := 0, ""
		tbl := uint32(mem.Base) + t.base
		for i := uint32(0); i < 128; i++ {
			p, err := mem.U32(src, tbl+4*i)
			for k := 0; p != 0 && err == nil && k < 64; k++ {
				var b [0xE8]byte
				if err = src.ReadAt(b[:], p); err != nil {
					break
				}
				typ := le(b[:], 0)
				id := le(b[:], 0x0C)
				if typ == 0 {
					n++
					nm := ""
					if d, e := mem.U32(src, p+0x14); e == nil && d != 0 {
						nm = cstr(src, d)
					}
					shown += fmt.Sprintf(" [id=%08x class=%d mode=%d name=%q]", id, le(b[:], 4), le(b[:], 0x10), nm)
				}
				p = le(b[:], 0xE4)
			}
		}
		fmt.Printf("  %s table players=%d%s\n", t.name, n, shown)
	}
	p, err := mem.U32(src, mem.Base+rvaRoster)
	fmt.Printf("  roster head=%08x\n", p)
	for n := 0; p != 0 && err == nil && n < 16; n++ {
		var b [0x84]byte
		if err = src.ReadAt(b[:], p); err != nil {
			break
		}
		fmt.Printf("  roster %q id=%08x class=%d level=%d partyid=%d levelid=%d x=%d y=%d partyflag=%d\n",
			cstrB(b[:16]), le(b[:], 0x10), le(b[:], 0x1C), le(b[:], 0x20)&0xFFFF, le(b[:], 0x20)>>16,
			le(b[:], 0x24), le(b[:], 0x28), le(b[:], 0x2C), le(b[:], 0x30))
		fmt.Printf("         raw % x\n", b[:0x40])
		p = le(b[:], 0x80)
	}
	return err
}

func le(b []byte, o int) uint32 {
	return uint32(b[o]) | uint32(b[o+1])<<8 | uint32(b[o+2])<<16 | uint32(b[o+3])<<24
}

func cstrB(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

func cstr(src mem.Source, a uint32) string {
	var b [16]byte
	_ = src.ReadAt(b[:], a)
	return cstrB(b[:])
}

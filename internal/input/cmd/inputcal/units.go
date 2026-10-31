package main

import (
	"fmt"
	"math"

	"d2lod-bot/internal/input"
	"d2lod-bot/internal/mem"
)

const rvaServerUnits = 0x3A5E70

type unit struct {
	Typ, Class, ID, Mode uint32
	X, Y                 int
	Addr                 uint32
}

// units walks the server-side hash table (VERIFIED LIVE in the removed cmd/spike) for one unit type.
func (e *env) units(typ int) []unit {
	var out []unit
	src := e.r.Src
	tbl := mem.Base + rvaServerUnits + uint32(typ)*0x200
	for i := uint32(0); i < 128; i++ {
		head, err := mem.U32(src, tbl+i*4)
		for n := 0; err == nil && head > 0x10000 && n < 64; n++ {
			u := unit{Addr: head}
			u.Typ, _ = mem.U32(src, head)
			u.Class, _ = mem.U32(src, head+4)
			u.ID, _ = mem.U32(src, head+0xC)
			u.Mode, _ = mem.U32(src, head+0x10)
			if p, _ := mem.U32(src, head+0x2C); p > 0x10000 {
				x, _ := mem.U16(src, p+2)
				y, _ := mem.U16(src, p+6)
				if typ == 2 || typ == 4 {
					a, _ := mem.U32(src, p+0xC)
					b, _ := mem.U32(src, p+0x10)
					x, y = uint16(a), uint16(b)
				}
				u.X, u.Y = int(x), int(y)
			}
			out = append(out, u)
			head, err = mem.U32(src, head+0xE4)
		}
	}
	return out
}

func (e *env) npcs() []unit { return e.units(1) }

func (e *env) nearby() {
	p, _ := e.r.Player()
	fmt.Printf("player (%.1f,%.1f)\n", p.X, p.Y)
	for _, typ := range []int{1, 2} {
		for _, u := range e.units(typ) {
			sx, sy := input.Calibrated.WorldToScreenF([2]float64{p.X, p.Y}, float64(u.X), float64(u.Y))
			if math.Hypot(float64(u.X)-p.X, float64(u.Y)-p.Y) < 40 {
				fmt.Printf("type=%d class=%d id=%d mode=%d pos=(%d,%d) screen=(%.0f,%.0f)\n", u.Typ, u.Class, u.ID, u.Mode, u.X, u.Y, sx, sy)
			}
		}
	}
}

package main

import (
	"fmt"
	"sort"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/mem"
)

const (
	rvaServerUnits = 0x3A5E70 // VERIFIED LIVE by cmd/spike
	rvaClientUnits = 0x3A5270 // docs 2.5, checked here
)

type unit struct {
	typ, class, id, mode uint32
	client               bool
	x, y                 int
}

func okPtr(v uint32) bool { return v >= 0x10000 && v < 0x7fff0000 }

// collectUnits reads server-side NPCs (type 1) and objects (type 2) from the hash table.
func collectUnits(s mem.Source) []unit {
	return append(collectTable(s, rvaServerUnits, false), collectTable(s, rvaClientUnits, true)...)
}

func collectTable(s mem.Source, rva uint32, client bool) []unit {
	var out []unit
	for typ := 1; typ <= 2; typ++ {
		tbl := mem.Base + rva + uint32(typ)*0x200
		for i := uint32(0); i < 128; i++ {
			head, err := mem.U32(s, tbl+i*4)
			if err != nil {
				continue
			}
			for n := 0; okPtr(head) && n < 64; n++ {
				u := unit{typ: uint32(typ), client: client}
				u.class, _ = mem.U32(s, head+4)
				u.id, _ = mem.U32(s, head+0xC)
				u.mode, _ = mem.U32(s, head+0x10)
				if path, err := mem.U32(s, head+0x2C); err == nil && okPtr(path) {
					if typ == 1 {
						x, _ := mem.U16(s, path+2)
						y, _ := mem.U16(s, path+6)
						u.x, u.y = int(x), int(y)
					} else {
						x, _ := mem.U32(s, path+0xC)
						y, _ := mem.U32(s, path+0x10)
						u.x, u.y = int(x), int(y)
					}
				}
				out = append(out, u)
				head, _ = mem.U32(s, head+0xE4)
			}
		}
	}
	return out
}

func units(s mem.Source, t *d2map.Tracker, st d2map.State) {
	us := collectUnits(s)
	sort.Slice(us, func(i, j int) bool {
		if us[i].typ != us[j].typ {
			return us[i].typ < us[j].typ
		}
		return us[i].class < us[j].class
	})
	lm := t.Current()
	fmt.Println("units (server table) vs nearest preset of same kind/id:")
	wallMatch := 0
	defer func() { fmt.Printf("  (%d wall/fence objects 40/41/42/65 matched their preset exactly)\n", wallMatch) }()
	for _, u := range us {
		kind := d2map.PresetNPC
		if u.typ == 2 {
			kind = d2map.PresetObject
		}
		best, bd := d2map.Preset{}, 1<<30
		for _, p := range lm.Presets {
			if p.Kind == kind && uint32(p.ID) == u.class {
				d := abs(p.Pos.X-u.x) + abs(p.Pos.Y-u.y)
				if d < bd {
					best, bd = p, d
				}
			}
		}
		f, ok := lm.Grid.Flags(u.x, u.y)
		note := "no preset"
		if bd < 1<<30 {
			note = fmt.Sprintf("preset (%d,%d) d=%d", best.Pos.X, best.Pos.Y, bd)
		}
		if u.typ == 2 && bd == 0 && (u.class >= 40 && u.class <= 42 || u.class == 65) {
			wallMatch++
			continue
		}
		fmt.Printf("  client=%v type=%d class=%-4d mode=%-2d pos=(%d,%d) cell=0x%04x known=%v  %s\n", u.client, u.typ, u.class, u.mode, u.x, u.y, f, ok, note)
	}
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

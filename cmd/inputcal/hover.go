package main

import (
	"fmt"
	"math"
	"sort"
	"time"

	"d2lod-bot/internal/input"
	"d2lod-bot/internal/mem"
)

const varBlock = 0x3A6800 // RVA scanned for the hover variables around GetSelectedUnit's operands

func (e *env) block() [0x2D0 / 4]uint32 {
	var out [0x2D0 / 4]uint32
	for i := range out {
		out[i], _ = mem.U32(e.r.Src, mem.Base+varBlock+uint32(i*4))
	}
	return out
}

// hover walks near each distinct NPC and sweeps the pointer over it (never clicks), printing
// which variables in varBlock change while the pointer is on the NPC.
func (e *env) hover(class int) {
	e.g.T = input.Instant()
	e.g.MoveMouse(790, 8)
	time.Sleep(300 * time.Millisecond)
	base := e.block()
	fmt.Printf("baseline @%#x nonzero:", varBlock)
	for i, v := range base {
		if v != 0 {
			fmt.Printf(" +%#x=%#x", i*4, v)
		}
	}
	fmt.Println()
	npcs := e.npcs()
	p, _ := e.r.Player()
	sort.Slice(npcs, func(i, j int) bool {
		return math.Hypot(float64(npcs[i].X)-p.X, float64(npcs[i].Y)-p.Y) < math.Hypot(float64(npcs[j].X)-p.X, float64(npcs[j].Y)-p.Y)
	})
	done := 0
	for _, n := range npcs {
		if done >= 1 {
			break
		}
		if int(n.Class) != class {
			continue
		}
		done++
		fmt.Printf("\nNPC class=%d id=%d pos=(%d,%d) mode=%d\n", n.Class, n.ID, n.X, n.Y, n.Mode)
		if err := e.g.MoveTo(input.Point{X: n.X, Y: n.Y}, input.MoveOpts{Tolerance: 9, Timeout: 15 * time.Second}); err != nil {
			fmt.Println("  moveto:", err)
			continue
		}
		time.Sleep(400 * time.Millisecond)
		pl, _ := e.r.Player()
		sx, sy := input.Calibrated.WorldToScreenF([2]float64{pl.X, pl.Y}, float64(n.X), float64(n.Y))
		fmt.Printf("  player (%.1f,%.1f) npc feet at screen (%.0f,%.0f)\n", pl.X, pl.Y, sx, sy)
		probe := func(x, y int, tag string) {
			e.g.MoveMouse(x, y)
			time.Sleep(150 * time.Millisecond)
			b := e.block()
			var diff string
			for i := range b {
				if b[i] != base[i] && varBlock+i*4 != 0x3A6AAC && varBlock+i*4 != 0x3A6AB0 {
					diff += fmt.Sprintf(" +%#x=%#x", i*4, b[i])
				}
			}
			fmt.Printf("  %s mouse(%d,%d): id@78=%d type@8c=%d f@94=%d f@98=%d |%s\n", tag, x, y, b[0x278/4], b[0x28c/4], b[0x294/4], b[0x298/4], diff)
		}
		probe(700, 100, "control")
		probe(100, 400, "control")
		probe(790, 8, "corner")
		for dy := 10; dy >= -70; dy -= 40 {
			for _, dx := range []int{0, -8, 8} {
				x, y := int(sx)+dx, int(sy)+dy
				if x < 0 || y < 0 || x > 799 || y > 520 {
					continue
				}
				e.g.MoveMouse(x, y)
				time.Sleep(120 * time.Millisecond)
				b := e.block()
				var diff string
				for i := range b {
					if b[i] != base[i] && varBlock+i*4 != 0x3A6AAC && varBlock+i*4 != 0x3A6AB0 {
						diff += fmt.Sprintf(" +%#x=%#x", i*4, b[i])
					}
				}
				fmt.Printf("  mouse(%d,%d) feet%+d,%+d:%s\n", x, y, dx, dy, diff)
			}
		}
		e.g.MoveMouse(790, 8)
	}
}

func (e *env) hoverTest(class int) {
	e.g.T = input.Instant()
	e.g.MoveMouse(790, 8)
	time.Sleep(300 * time.Millisecond)
	show := func(tag string) {
		t, id, err := e.r.Hovered()
		f, _ := mem.U32(e.r.Src, mem.Base+0x3A6A94)
		g, _ := mem.U32(e.r.Src, mem.Base+0x3A6A98)
		fmt.Printf("  %s hovered typ=%d id=%d err=%v f94=%d f98=%d\n", tag, t, id, err, f, g)
	}
	show("corner")
	for _, n := range e.npcs() {
		if int(n.Class) != class {
			continue
		}
		p, _ := e.r.Player()
		sx, sy := input.Calibrated.WorldToScreenF([2]float64{p.X, p.Y}, float64(n.X)+0.5, float64(n.Y)+0.5)
		fmt.Printf("npc id=%d pos=(%d,%d) feet screen (%.0f,%.0f)\n", n.ID, n.X, n.Y, sx, sy)
		for dy := 0; dy >= -50; dy -= 10 {
			e.g.MoveMouse(int(sx), int(sy)+dy)
			time.Sleep(200 * time.Millisecond)
			show(fmt.Sprintf("feet%+d", dy))
		}
		e.g.MoveMouse(790, 8)
		time.Sleep(200 * time.Millisecond)
		show("corner")
		return
	}
	fmt.Println("npc not found")
}

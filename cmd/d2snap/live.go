package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"d2lod-bot/internal/mem"
)

// Live experiments that need the d2input helper (UI panel toggling, hover). Always restore state.

type helper struct {
	c net.Conn
	r *bufio.Reader
}

func dialHelper() (*helper, error) {
	c, err := net.DialTimeout("tcp", "127.0.0.1:47012", 2*time.Second)
	if err != nil {
		return nil, err
	}
	return &helper{c: c, r: bufio.NewReader(c)}, nil
}

func (h *helper) send(cmd string) string {
	fmt.Fprintln(h.c, cmd)
	h.c.SetReadDeadline(time.Now().Add(3 * time.Second))
	l, err := h.r.ReadString('\n')
	if err != nil {
		return "ERR " + err.Error()
	}
	return strings.TrimSpace(l)
}

func uiArray(s mem.Source) [38]uint32 {
	var a [38]uint32
	for i := range a {
		a[i] = u32(s, mem.Base+0x3A27C0+uint32(i)*4)
	}
	return a
}

func diffUI(a, b [38]uint32) string {
	var out []string
	for i := range a {
		if a[i] != b[i] {
			out = append(out, fmt.Sprintf("[%d(0x%x)] %d->%d", i, i, a[i], b[i]))
		}
	}
	if len(out) == 0 {
		return "none"
	}
	return strings.Join(out, " ")
}

func liveUI(s mem.Source, keys []string) {
	h, err := dialHelper()
	if err != nil {
		fmt.Println("helper:", err)
		return
	}
	defer h.c.Close()
	fmt.Println("info:", h.send("info"))
	base := uiArray(s)
	fmt.Println("baseline nonzero:", nonzero(base))
	for _, k := range keys {
		h.send("key press " + k)
		time.Sleep(500 * time.Millisecond)
		open := uiArray(s)
		fmt.Printf("key %-4s opened: diff vs baseline = %s\n", k, diffUI(base, open))
		h.send("key press " + k)
		time.Sleep(500 * time.Millisecond)
		after := uiArray(s)
		if after != base {
			fmt.Printf("  toggle-off did not restore (%s); pressing esc\n", diffUI(base, after))
			for i := 0; i < 3 && uiArray(s) != base; i++ {
				h.send("key press esc")
				time.Sleep(500 * time.Millisecond)
			}
		}
		fmt.Printf("  restored=%v\n", uiArray(s) == base)
	}
	h.send("key press esc")
	time.Sleep(500 * time.Millisecond)
	esc := uiArray(s)
	fmt.Println("esc alone: diff vs baseline =", diffUI(base, esc))
	if esc != base {
		h.send("key press esc")
		time.Sleep(500 * time.Millisecond)
	}
	fmt.Println("final restored =", uiArray(s) == base)
}

func nonzero(a [38]uint32) string {
	var out []string
	for i, v := range a {
		if v != 0 {
			out = append(out, fmt.Sprintf("[%d]=%d", i, v))
		}
	}
	return strings.Join(out, " ")
}

func selVars(s mem.Source) string {
	var sb strings.Builder
	for a := uint32(0x3A6A78); a <= 0x3A6AA8; a += 4 {
		fmt.Fprintf(&sb, "%x ", u32(s, mem.Base+a))
	}
	return sb.String()
}

func liveHover(s mem.Source) {
	h, err := dialHelper()
	if err != nil {
		fmt.Println("helper:", err)
		return
	}
	defer h.c.Close()
	pl := u32(s, mem.Base+0x3A6A70)
	path := u32(s, pl+0x2C)
	px, py := int(u16(s, path+2)), int(u16(s, path+6))
	vpx := int(int32(u32(s, mem.Base+0x3A520C)))
	vpy := int(int32(u32(s, mem.Base+0x3A5208)))
	fmt.Printf("player (%d,%d) viewport (%d,%d) => player screen (%d,%d)\n", px, py, vpx, vpy, (px-py)*16-vpx, (px+py)*8-vpy)
	tbl := mem.Base + 0x3A5E70 + 1*0x200
	n := 0
	for i := uint32(0); i < 128 && n < 3; i++ {
		for u := u32(s, tbl+i*4); u > 0x10000 && n < 3; u = u32(s, u+0xE4) {
			p := u32(s, u+0x2C)
			x, y := int(u16(s, p+2)), int(u16(s, p+6))
			sx, sy := (x-y)*16-vpx, (x+y)*8-vpy
			if sx < 20 || sx > 780 || sy < 40 || sy > 450 {
				continue
			}
			n++
			fmt.Printf("monster class=%d id=%d pos=(%d,%d) feet screen=(%d,%d)\n", u32(s, u+4), u32(s, u+0xC), x, y, sx, sy)
			for _, dy := range []int{0, -10, -20, -30, -45} {
				h.send(fmt.Sprintf("move %d %d", sx, sy+dy))
				time.Sleep(250 * time.Millisecond)
				fmt.Printf("  dy=%d mouse=(%d,%d) vars: %s\n", dy, int32(u32(s, mem.Base+0x3A6AB0)), int32(u32(s, mem.Base+0x3A6AAC)), selVars(s))
			}
		}
	}
	h.send("move 400 560")
	time.Sleep(250 * time.Millisecond)
	fmt.Println("neutral:", selVars(s))
}

// liveSweep moves the pointer over a grid and reports every distinct state of the memory block that
// contains the selected-unit variables, to find which dwords change when a unit is hovered.
func liveSweep(s mem.Source) {
	h, err := dialHelper()
	if err != nil {
		fmt.Println("helper:", err)
		return
	}
	defer h.c.Close()
	blk := func() string {
		b := make([]byte, 0x100)
		_ = s.ReadAt(b, mem.Base+0x3A6A00)
		return string(b)
	}
	h.send("move 400 570")
	time.Sleep(300 * time.Millisecond)
	neutral := blk()
	seen := map[string]bool{}
	for y := 60; y <= 460; y += 20 {
		for x := 60; x <= 740; x += 20 {
			h.send(fmt.Sprintf("move %d %d", x, y))
			time.Sleep(120 * time.Millisecond)
			cur := blk()
			var d []string
			for i := 0; i+4 <= len(cur); i += 4 {
				if cur[i:i+4] != neutral[i:i+4] {
					a := uint32(0x3A6A00 + i)
					if a == 0x3A6AAC || a == 0x3A6AB0 {
						continue
					}
					d = append(d, fmt.Sprintf("%x=%x", a, le(cur[i:])))
				}
			}
			k := strings.Join(d, " ")
			if k != "" && !seen[k] {
				seen[k] = true
				fmt.Printf("mouse (%d,%d): %s\n", x, y, k)
			}
		}
	}
	h.send("move 400 570")
}

func le(s string) uint32 {
	return uint32(s[0]) | uint32(s[1])<<8 | uint32(s[2])<<16 | uint32(s[3])<<24
}

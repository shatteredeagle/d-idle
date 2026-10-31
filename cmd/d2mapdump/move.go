package main

import (
	"bufio"
	"fmt"
	"net"
	"time"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/mem"
)

func helper(cmd string) string {
	c, err := net.DialTimeout("tcp", "127.0.0.1:47012", 2*time.Second)
	if err != nil {
		return err.Error()
	}
	defer c.Close()
	fmt.Fprintln(c, cmd)
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	line, _ := bufio.NewReader(c).ReadString('\n')
	return line
}

func around(g *d2map.Grid, p d2map.Point, rad int) string {
	return g.ASCII(d2map.Rect{X: p.X - rad, Y: p.Y - rad, W: 2*rad + 1, H: 2*rad + 1}, map[d2map.Point]rune{p: '@'})
}

// movetest clicks at (400+dx, 300+dy), watches the player walk and shows the grid around it.
func movetest(s mem.Source, dx, dy int) {
	t := d2map.NewTracker(s)
	t.RescanEvery = 1
	st, err := t.Update()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("before: pos=%v\n%s", st.Pos, around(t.Current().Grid, st.Pos, 8))
	fmt.Println("click:", helper(fmt.Sprintf("click l %d %d", 400+dx, 300+dy)))
	for i := 0; i < 16; i++ {
		time.Sleep(250 * time.Millisecond)
		st, _ = t.Update()
		f, _ := t.Current().Grid.Flags(st.Pos.X, st.Pos.Y)
		fmt.Printf("  t=%.2fs pos=%v flags@pos=0x%04x\n", float64(i+1)*0.25, st.Pos, f)
	}
	fmt.Printf("after: pos=%v\n%s", st.Pos, around(t.Current().Grid, st.Pos, 8))
}

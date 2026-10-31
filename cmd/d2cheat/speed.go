package main

import (
	"fmt"
	"math"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/mem"
)

// speedtest is read/input research (no memory writes): click to move to a spot a few tiles away and
// measure the walked distance over a fixed window, to compare movement stat values.
func speedtest(p *mem.Proc, args []string) {
	r := d2.New(p)
	g := input.NewGame(input.NewClient(""), input.Reader{Src: p})
	for _, dir := range [][2]int{{-14, 0}, {14, 0}, {0, -14}, {0, 14}, {10, 10}, {-10, -10}, {10, -10}, {-10, 10}} {
		pl, err := r.Player()
		if err != nil {
			fatal(err)
		}
		before := input.Point{X: pl.X, Y: pl.Y}
		if err := g.ClickWorld(input.Point{X: pl.X + dir[0], Y: pl.Y + dir[1]}); err != nil {
			fatal(err)
		}
		var path, lx, ly, tt float64
		lx, ly = float64(pl.X), float64(pl.Y)
		start := time.Now()
		for time.Since(start) < 1500*time.Millisecond {
			time.Sleep(40 * time.Millisecond)
			cur, err := r.Player()
			if err != nil {
				continue
			}
			path += math.Hypot(float64(cur.X)-lx, float64(cur.Y)-ly)
			lx, ly = float64(cur.X), float64(cur.Y)
			if tt == 0 && math.Hypot(lx-float64(before.X), ly-float64(before.Y)) >= 9 {
				tt = time.Since(start).Seconds()
			}
		}
		moved := math.Hypot(lx-float64(before.X), ly-float64(before.Y))
		fmt.Printf("dir %v: moved %.1f subtiles in 1.5 s (path %.1f), 9 subtiles reached after %.2f s\n", dir, moved, path, tt)
		if moved > 4 {
			return
		}
	}
}

// Command kickrepro drives a THROWAWAY headless d2d through the bot's own d2dinput.Controller to reproduce the warp
// arrival bounce (docs/bounce3.md): for each (from, to) pair it takes the warp with "debug enter", then sends the
// bot's first move after a level change (Controller.Move: the priming click, then "cmd move") to a cell 4 cells
// straight away from the return warp, and reports whether the hero went back. Exit 1 when any arrival bounced.
// Usage: kickrepro -sock PATH (tools/bounce3/repro.sh starts the d2d and passes the socket).
package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dinput"
	"d2lod-bot/internal/input"
)

func main() {
	sock := flag.String("sock", "", "d2d control socket")
	flag.Parse()
	ctx := context.Background()
	c := d2dctl.New(*sock, d2dctl.Options{Timeout: 10 * time.Second, NoAutoKick: true})
	defer c.Close()
	runs, bounces := 0, 0
	for _, p := range [][2]int{{3, 9}, {9, 3}, {2, 8}, {8, 2}} {
		b, err := trial(ctx, c, p[0], p[1])
		if err != nil {
			fmt.Printf("%d->%d: %v\n", p[0], p[1], err)
			continue
		}
		runs++
		if b {
			bounces++
		}
	}
	fmt.Printf("RESULT runs=%d bounces=%d\n", runs, bounces)
	if bounces > 0 || runs == 0 {
		os.Exit(1)
	}
}

func waitLevel(ctx context.Context, c *d2dctl.Client, id int, d time.Duration) (d2dctl.LevelInfo, error) {
	end := time.Now().Add(d)
	for {
		l, err := c.Level(ctx)
		if err != nil || l.ID == id || time.Now().After(end) {
			return l, err
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func trial(ctx context.Context, c *d2dctl.Client, src, dst int) (bool, error) {
	if _, err := c.Debug(ctx, "goto", strconv.Itoa(src)); err != nil {
		return false, err
	}
	time.Sleep(500 * time.Millisecond)
	ws, err := c.Warps(ctx)
	if err != nil {
		return false, err
	}
	idx := -1
	for _, w := range ws.Items {
		if w.To == dst {
			idx = w.Index
		}
	}
	if idx < 0 {
		return false, fmt.Errorf("no warp to %d", dst)
	}
	if _, err := c.Debug(ctx, "enter", strconv.Itoa(idx)); err != nil {
		return false, err
	}
	l, err := waitLevel(ctx, c, dst, 6*time.Second)
	if err != nil || l.ID != dst {
		return false, fmt.Errorf("arrival in %d failed (level %d, %v)", dst, l.ID, err)
	}
	time.Sleep(time.Second)
	if l, err = c.Level(ctx); err != nil {
		return false, err
	}
	back, err := c.Warps(ctx)
	if err != nil {
		return false, err
	}
	bx, by := l.X, l.Y-1
	for _, w := range back.Items {
		if w.To == src {
			bx, by = w.X+0.5, w.Y+0.5
		}
	}
	ax, ay := l.X-bx, l.Y-by
	d := math.Max(math.Hypot(ax, ay), 1e-6)
	tx, ty := l.X+ax/d*4, l.Y+ay/d*4
	ctl := d2dinput.New(c)
	ctl.Ctx = ctx
	ctl.Trace = func(msg string, kv ...any) { fmt.Println("  ", msg, kv) }
	if err := ctl.Move(input.Point{X: int(tx * 5), Y: int(ty * 5)}); err != nil {
		return false, err
	}
	time.Sleep(1500 * time.Millisecond)
	after, err := c.Level(ctx)
	if err != nil {
		return false, err
	}
	bounced := after.ID == src
	fmt.Printf("%d->%d arrive=(%.1f,%.1f) returnWarp=(%.1f,%.1f) dist=%.2f -> level %d bounced=%v\n", src, dst, l.X, l.Y, bx, by, math.Hypot(ax, ay), after.ID, bounced)
	return bounced, nil
}

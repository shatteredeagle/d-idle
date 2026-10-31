// d2multi: live isolation check for multi-instance (docs/multi.md). Run with tools/live.sh, instance from
// D2BOT_INSTANCE. Usage: d2multi [all-instances...]  e.g. d2multi i0 i1 i2 i3 i4 i5
// It reads the player position of every named instance, drives its OWN game with a posted click-to-move,
// reads all positions again, and prints who moved. It also runs input.Bind against its own memory
// (must pass) and against every other instance's memory (must fail).
package main

import (
	"d2lod-bot/internal/buildinfo"
	"fmt"
	"os"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/mem"
)

func main() {
	buildinfo.Exit("d2multi")
	me := os.Getenv(mem.InstanceEnv)
	ids := os.Args[1:]
	procs := map[string]*mem.Proc{}
	for _, id := range ids {
		p, err := mem.OpenInstance(id)
		if err != nil {
			fmt.Printf("%s: open: %v\n", id, err)
			continue
		}
		defer p.Close()
		procs[id] = p
	}
	pos := func() map[string]string {
		out := map[string]string{}
		for id, p := range procs {
			s, err := d2.New(p).Snapshot()
			if err != nil {
				out[id] = "err " + err.Error()
				continue
			}
			out[id] = fmt.Sprintf("%s area=%d state=%v pos=(%d,%d)", s.Player.Name, s.Player.Area, s.State, s.Player.X, s.Player.Y)
		}
		return out
	}
	cli := input.NewClient("")
	fmt.Printf("me=%s helper=%s\n", me, cli.Addr)
	w, err := cli.Info()
	fmt.Printf("window: %+v err=%v\n", w, err)
	g := input.NewGame(cli, input.Reader{Src: procs[me]})
	for id, p := range procs {
		err := input.Bind(g, p)
		fmt.Printf("bind own window vs memory of %s: %v\n", id, err)
	}
	before := pos()
	p0, err := g.Pos()
	if err != nil {
		fmt.Println("pos:", err)
		os.Exit(1)
	}
	target := input.Point{X: p0.X + 5, Y: p0.Y + 3}
	err = g.MoveTo(target, input.MoveOpts{Timeout: 15 * time.Second})
	fmt.Printf("moveto %v from %v: err=%v\n", target, p0, err)
	after := pos()
	for _, id := range ids {
		fmt.Printf("%s: before %s | after %s | moved=%v\n", id, before[id], after[id], before[id] != after[id])
	}
	g.CloseAll()
}

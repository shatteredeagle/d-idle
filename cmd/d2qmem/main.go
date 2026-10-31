// d2qmem: read-only verification tool for in-memory quest and waypoint data (docs/questmem.md).
// It prints the quest words and waypoint masks the game holds, for comparison with `d2cheat qdump` of the save.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/questmem"
)

func main() {
	p, err := mem.OpenGame()
	if err != nil {
		fmt.Println("open:", err)
		os.Exit(1)
	}
	defer p.Close()
	r := d2.New(p)
	if len(os.Args) == 2 && strings.HasPrefix(os.Args[1], "record=") {
		record(p, os.Args[1][7:])
		return
	}
	if len(os.Args) == 3 && os.Args[1] == "watch" {
		watch(r, os.Args[2])
		return
	}
	// "do CMD" arguments post d2input commands (e.g. "click l 349 238") before the read.
	if len(os.Args) > 1 {
		c := input.NewClient("")
		for _, a := range os.Args[1:] {
			out, err := c.Do(a, 2*time.Second)
			fmt.Println(a, "->", out, err)
			time.Sleep(1200 * time.Millisecond)
		}
	}
	pl, err := r.Player()
	if err != nil {
		fmt.Println("state", r.State(), "player:", err)
		return
	}
	t0 := time.Now()
	su, err := r.FindServerPlayer(pl)
	fmt.Printf("%s area %d client=%#x server=%#x err=%v scan=%v\n", pl.Name, pl.Area, pl.Addr, su, err, time.Since(t0))
	if err != nil {
		return
	}
	pp, err := r.ReadProgress(su)
	if err != nil {
		fmt.Println("progress:", err)
		return
	}
	for d := 0; d < 3; d++ {
		fmt.Printf("D%d wp=%#x quests:", d, pp.Waypoints[d])
		for i, v := range pp.Quest[d] {
			if v != 0 {
				fmt.Printf(" %d=%#04x", i, v)
			}
		}
		fmt.Println()
	}
}

// record writes a replay fixture holding only the pages a quest read touches: the in-game canary chain, the
// client player, the server unit and the pointer chain to the quest buffers and waypoint structs. The test
// supplies the region list for the scan (the server unit's page), so no heap sweep is recorded.
func record(p *mem.Proc, path string) {
	r := d2.New(p)
	pl, err := r.PlayerIdent()
	if err != nil {
		fmt.Println("player:", err)
		return
	}
	su, err := r.FindServerPlayer(pl)
	if err != nil {
		fmt.Println("scan:", err)
		return
	}
	rec := mem.NewRecording(p)
	rr := d2.New(rec)
	st := rr.State()
	pl2, err := rr.PlayerIdent()
	if err != nil || !rr.ServerPlayerOK(su, pl2) {
		fmt.Println("validate:", err)
		return
	}
	pp, err := rr.ReadProgress(su)
	if err != nil {
		fmt.Println("progress:", err)
		return
	}
	snap := rec.Snapshot()
	if err := snap.Save(path); err != nil {
		fmt.Println("save:", err)
		return
	}
	fmt.Printf("recorded %s: %d pages, state %v, server unit %#x, player %+v\nprogress %+v\n", path, snap.NumPages(), st, su, *pl2, *pp)
}

// watch polls a questmem.Source for SECS seconds and prints every change, scan and error.
func watch(r *d2.Reader, secs string) {
	n, _ := strconv.Atoi(secs)
	src := questmem.New(r)
	src.MaxAge = 0
	last, lastScans := "", 0
	for end := time.Now().Add(time.Duration(n) * time.Second); time.Now().Before(end); time.Sleep(100 * time.Millisecond) {
		pp, err := src.Progress()
		cur := fmt.Sprint(err)
		if err == nil {
			cur = fmt.Sprintf("%v wp=%#x/%#x/%#x state=%v", pp.Quest[0][:8], pp.Waypoints[0], pp.Waypoints[1], pp.Waypoints[2], r.State())
		}
		if cur != last || src.Scans() != lastScans {
			fmt.Printf("%s scans=%d unit=%v %s\n", time.Now().Format("15:04:05.000"), src.Scans(), pp != nil, cur)
			last, lastScans = cur, src.Scans()
		}
	}
}

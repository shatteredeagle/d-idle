//go:build rigs

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/leveler/townvendor"
	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/nav"
	"d2lod-bot/internal/town"
	"d2lod-bot/internal/town/hand"
)

// amaBuy: `classsmoke ama buy NPC TABX TABY GX GY H`: open vendor NPC (class id), click the tab at TABX,TABY, right-click
// the stock cell GX,GY (H cells tall) and report the gold. levelerlive trybuy only knows the tabs of pages 0, 1 and 3,
// so Weapons 2 (bows) could not be reached with it (docs/classes/smoke-amazon.md finding 5).
func amaBuy(a []string) {
	n := func(i int) int { v, _ := strconv.Atoi(a[i]); return v }
	src, err := mem.OpenGame()
	die(err)
	defer src.Close()
	home, _ := os.UserHomeDir()
	dir := home + "/Games/d2lod-inst/" + os.Getenv("D2BOT_INSTANCE") + "/proton/pfx/drive_c/Program Files (x86)/Diablo II"
	db, err := data.Open(dir, home+"/d2lod-live/classama/data")
	die(err)
	tune := env.DefaultTuning()
	e := &env.Env{ID: "classama", Log: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})),
		Mem: src, Read: d2.New(src), World: d2map.NewTracker(src), In: input.NewGame(input.NewClient(""), input.Reader{Src: src}), Data: db, Tune: &tune,
		Clock: env.RealClock{}, Emit: func(goals.Event) {}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Second)
	defer cancel()
	tv := townvendor.Vendor{T: town.New(e, nav.New(e))}
	npc, err := tv.OpenTrade(ctx, n(0))
	fmt.Println("open:", npc.Class, err)
	if err != nil {
		return
	}
	s, _ := e.Snapshot()
	g0 := s.Player.Gold
	if len(a) > 6 { // hold the open vendor panel for a screenshot
		hold, _ := strconv.Atoi(a[6])
		fmt.Println("holding", hold, "s")
		time.Sleep(time.Duration(hold) * time.Second)
	}
	_ = e.In.MoveMouse(n(1), n(2)) // hover first: the game ignores a click that comes right after the move
	time.Sleep(400 * time.Millisecond)
	fmt.Println("tab click", e.In.Click(input.Left, n(1), n(2)))
	time.Sleep(3 * time.Second)
	lay, err := hand.New(e).Layout(s)
	die(err)
	pt := lay.Vendor.ItemCenter(n(3), n(4), 1, n(5))
	_ = e.In.MoveMouse(pt.X, pt.Y)
	time.Sleep(400 * time.Millisecond)
	fmt.Println("right click", pt, e.In.Click(input.Right, pt.X, pt.Y))
	time.Sleep(3 * time.Second)
	s, _ = e.Snapshot()
	fmt.Println("gold", g0, "->", s.Player.Gold)
	fmt.Println("closeall:", e.In.CloseAll())
}

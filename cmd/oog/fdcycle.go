package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/mem"
)

func inst() string {
	if v := os.Getenv("D2BOT_INSTANCE"); v != "" {
		return v
	}
	return "i0"
}

func fdCycle(flow *launch.Flow, n int, name string) error {
	procs := instProcs(inst())
	var first map[string]map[string]string
	snap := func() map[string]map[string]string {
		m := map[string]map[string]string{}
		for _, p := range instProcs(inst()) {
			if p.name == "Game.exe" || p.name == "wineserver" || p.name == "d2input.exe" {
				m[p.name] = fdTargets(p.pid)
			}
		}
		return m
	}
	line, _ := fdLine(procs)
	fmt.Printf("FD cycle=0 t=%s %s\n", time.Now().Format("15:04:05"), line)
	first = snap()
	for i := 1; i <= n; i++ {
		t0 := time.Now()
		if err := flow.LeaveGame(context.Background(), 20*time.Second); err != nil {
			return fmt.Errorf("cycle %d leave: %w", i, err)
		}
		if os.Getenv("FD_SPLIT") != "" {
			l, _ := fdLine(instProcs(inst()))
			fmt.Printf("FD   after-leave %s\n", l)
		}
		if err := flow.Start(context.Background(), launch.CharSpec{Name: name, Difficulty: "normal"}); err != nil {
			return fmt.Errorf("cycle %d start: %w", i, err)
		}
		time.Sleep(2 * time.Second)
		line, _ := fdLine(instProcs(inst()))
		fmt.Printf("FD cycle=%d took=%s t=%s %s\n", i, time.Since(t0).Round(100*time.Millisecond), time.Now().Format("15:04:05"), line)
	}
	last := snap()
	for nm := range last {
		d := fdDiff(first[nm], last[nm])
		fmt.Printf("DIFF %s +%d new targets vs cycle 0:\n", nm, len(d))
		cnt := map[string]int{}
		for _, v := range d {
			cnt[v]++
		}
		for v, c := range cnt {
			fmt.Printf("  %dx %s\n", c, v)
		}
	}
	return nil
}

func fdIdle(n int, every time.Duration) {
	game := func() (int, map[string]string) {
		for _, p := range instProcs(inst()) {
			if p.name == "Game.exe" {
				return p.pid, fdTargets(p.pid)
			}
		}
		return 0, nil
	}
	line, _ := fdLine(instProcs(inst()))
	fmt.Printf("FD idle=0 t=%s %s\n", time.Now().Format("15:04:05"), line)
	_, prev := game()
	for i := 1; i <= n; i++ {
		time.Sleep(every)
		line, _ := fdLine(instProcs(inst()))
		fmt.Printf("FD idle=%d t=%s %s\n", i, time.Now().Format("15:04:05"), line)
		_, cur := game()
		if len(cur) != len(prev) {
			for _, v := range fdDiff(prev, cur) {
				fmt.Printf("  game +%s\n", v)
			}
			prev = cur
		}
	}
}

// recycleNow runs one launch.Recycler pass for this instance and reports the fd counts before and after.
func recycleNow(flow *launch.Flow, name string, src *mem.Source) error {
	life := launch.NewLife(os.Stdout)
	id := inst()
	gamefds := func() int {
		for _, p := range instProcs(id) {
			if p.name == "Game.exe" {
				n, _ := launch.CountFDs(p.pid)
				return n
			}
		}
		return 0
	}
	fmt.Println("  game fds before:", gamefds())
	r := &launch.Recycler{
		Leave: func(ctx context.Context) error { return flow.LeaveGame(ctx, 20*time.Second) },
		Stop:  func(ctx context.Context) error { return life.Stop(ctx, id) },
		Start: func(ctx context.Context) error {
			if err := life.Start(ctx, id); err != nil {
				return err
			}
			return life.EnsureHelper(ctx, id)
		},
		Login: func(ctx context.Context) error {
			t0 := time.Now()
			for {
				p, err := mem.OpenGame()
				if err == nil {
					*src, flow.Mem = p, p
					break
				}
				if time.Since(t0) > 60*time.Second {
					return err
				}
				time.Sleep(200 * time.Millisecond)
			}
			return flow.Start(ctx, launch.CharSpec{Name: name, Difficulty: "normal"})
		},
		Log: func(s string) { fmt.Println("  recycle:", s) },
	}
	t0 := time.Now()
	err := r.Recycle(context.Background(), "manual")
	fmt.Printf("  recycle took %v, game fds after: %d\n", time.Since(t0).Round(100*time.Millisecond), gamefds())
	return err
}

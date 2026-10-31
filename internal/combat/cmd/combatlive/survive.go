package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/nav"
)

type typer struct {
	*input.OOG
	c *input.Client
}

func (t typer) TypeChar(r rune) error { return t.c.TypeChar(r) }

// survive runs one real fight with survival thresholds chosen to provoke a potion or a chicken exit:
//
//	combatlive survive potions|chicken [spot X Y] [seconds]
//
// potions: drink at 70% (chicken far below); chicken: no potions at all (drink 5%), leave at 55%. After a chicken exit the
// character re-enters through launch.Flow.Start and the tool prints where the game put it.
func survive(ctx context.Context, e *env.Env, f *combat.Fighter, nv *nav.Navigator, src mem.Source, db *data.DB, args []string) {
	mode := "potions"
	if len(args) > 0 {
		mode = args[0]
	}
	spotX, spotY, secs, haveSpot := 0, 0, 90, false
	if len(args) > 3 {
		fmt.Sscan(args[1], &spotX)
		fmt.Sscan(args[2], &spotY)
		fmt.Sscan(args[3], &secs)
		haveSpot = true
	}
	c := &e.Tune.Combat
	switch mode {
	case "potions":
		c.DrinkLifePct, c.DrinkRejuvPct, c.ChickenPct, c.TownPotionMin = 70, 5, 10, 0
	case "chicken":
		c.DrinkLifePct, c.DrinkRejuvPct, c.ChickenPct, c.TownPotionMin = 5, 3, 100, 0 // 100: full life already counts, so the exit path runs at once
	}
	cli := input.NewClient("")
	flow := &launch.Flow{Mem: src, In: typer{input.NewOOG(cli), cli}, Log: func(s string) { fmt.Println("  flow:", s) }}
	f.Leave = func(ctx context.Context) error { return flow.LeaveGame(ctx, 20*time.Second) }

	s, err := e.Snapshot()
	die(err)
	cat := combat.NewCatalog(db)
	fmt.Printf("belt before: %+v life %d/%d\n", combat.BeltOf(s.Inventory, cat), s.Player.Life, s.Player.MaxLife)
	if s.Player.InTown() {
		fmt.Println("ExitTo(2):", nv.ExitTo(ctx, 2))
	}
	if haveSpot {
		fmt.Println("GoTo spot:", nv.GoTo(ctx, input.Point{X: spotX, Y: spotY}))
	} else { // explore until something hostile is within 45 subtiles; the layout differs per map seed
		hostile := func() bool {
			cur, err := e.Snapshot()
			if err != nil || cur.Player == nil {
				return false
			}
			hs := combat.Hostiles(cur, combat.HostileCtx{Cat: cat, Diff: cur.Player.Difficulty, ClassLvl: cur.Player.Level})
			return len(hs) > 0 && hs[0].Dist < 45
		}
		xctx, xc := context.WithTimeout(ctx, 60*time.Second)
		fmt.Println("Explore:", nv.Explore(xctx, hostile))
		xc()
	}
	cctx, cancel := context.WithTimeout(ctx, time.Duration(secs)*time.Second)
	defer cancel()
	t0 := time.Now()
	res, err := f.Clear(cctx, combat.ClearOpts{Radius: 60, Timeout: time.Duration(secs) * time.Second})
	fmt.Printf("Clear after %v: %v err=%v\n", time.Since(t0).Round(time.Second), res, err)
	for _, ig := range res.Ignored {
		fmt.Println("  ignored:", ig)
	}
	s, _ = e.Snapshot()
	if s != nil && s.Player != nil {
		fmt.Printf("after: life %d/%d mode %d area %d belt %+v\n", s.Player.Life, s.Player.MaxLife, s.Player.Mode, s.Player.Area, combat.BeltOf(s.Inventory, cat))
	} else if s != nil {
		fmt.Println("after: state", s.State)
	}
	if errors.Is(err, combat.ErrChicken) {
		fmt.Println("chicken exit done; re-entering through the OOG flow")
		t1 := time.Now()
		serr := flow.Start(ctx, launch.CharSpec{Name: "Combi"})
		fmt.Println("Start:", serr, "in", time.Since(t1).Round(time.Second))
		for i := 0; i < 20; i++ {
			s, _ = e.Snapshot()
			if s != nil && s.State == d2.StateInGame && s.Player != nil {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}
		if s != nil && s.Player != nil {
			p := s.Player
			fmt.Printf("re-entered: %s area %d pos (%d,%d) mode %d life %d/%d; monsters near: %d\n", p.Name, p.Area, p.X, p.Y, p.Mode, p.Life, p.MaxLife,
				len(combat.Hostiles(s, combat.HostileCtx{Cat: cat, Diff: p.Difficulty, ClassLvl: p.Level})))
		}
	}
}

// farm explores and clears for a fixed time and prints kills per minute and XP per hour (cheated sessions are
// flagged in the ledger): combatlive farm SECONDS [AREA]
func farm(ctx context.Context, e *env.Env, f *combat.Fighter, nv *nav.Navigator, db *data.DB, args []string) {
	secs, area := 240, 2
	if len(args) > 0 {
		fmt.Sscan(args[0], &secs)
	}
	if len(args) > 1 {
		fmt.Sscan(args[1], &area)
	}
	cat := combat.NewCatalog(db)
	s, err := e.Snapshot()
	die(err)
	if s.Player.InTown() {
		fmt.Println("ExitTo(2):", nv.ExitTo(ctx, 2))
	}
	if area != 2 {
		fmt.Println("ExitTo area", area, nv.ExitTo(ctx, area))
	}
	s, _ = e.Snapshot()
	startExp, startLvl := s.Player.Exp, s.Player.Level
	t0 := time.Now()
	var tot combat.ClearResult
	skills := map[int]int{}
	hostile := func() bool {
		cur, err := e.Snapshot()
		if err != nil || cur.Player == nil {
			return false
		}
		for _, h := range combat.Hostiles(cur, combat.HostileCtx{Cat: cat, Diff: cur.Player.Difficulty, ClassLvl: cur.Player.Level}) {
			if h.Dist < 45 && !f.IsIgnored(h.Unit.ID, h.Dist) {
				return true
			}
		}
		return false
	}
	dead := false
	seenU := map[uint32]bool{}
	go func() { // affix census: every elite/boss/avoided/dangerous unit once, with what the danger table does with it
		for ctx.Err() == nil {
			if cur, err := e.Snapshot(); err == nil && cur.Player != nil {
				for _, h := range combat.Hostiles(cur, combat.HostileCtx{Cat: cat, Diff: cur.Player.Difficulty, ClassLvl: cur.Player.Level}) {
					if (h.Elite || h.Boss || h.Affixes != 0 || h.Avoid) && !seenU[h.Unit.ID] {
						seenU[h.Unit.ID] = true
						fmt.Printf("AFFIX %q class %d elite=%v boss=%v affixes=[%s] avoid=%v danger=%.0f killFast=%v dist=%.0f flags=%#x\n",
							h.Name, h.Unit.Class, h.Elite, h.Boss, h.Affixes, h.Avoid, h.Danger, h.KillFast, h.Dist, h.Unit.Monster.Flags)
					}
				}
			}
			time.Sleep(400 * time.Millisecond)
		}
	}()
	for time.Since(t0) < time.Duration(secs)*time.Second && ctx.Err() == nil {
		xctx, xc := context.WithTimeout(ctx, 40*time.Second)
		xerr := nv.Explore(xctx, hostile)
		xc()
		if errors.Is(xerr, nav.ErrExplored) {
			fmt.Println("level explored")
			break
		}
		res, err := f.Clear(ctx, combat.ClearOpts{Radius: 50, Timeout: 60 * time.Second})
		if res.Kills == 0 && len(res.Ignored) == 0 {
			time.Sleep(500 * time.Millisecond) // nothing to do yet: do not spin
			continue
		}
		tot.Kills += res.Kills
		tot.XP += res.XP
		tot.Potions += res.Potions
		tot.Ignored = append(tot.Ignored, res.Ignored...)
		fmt.Printf("[%3.0fs] %v err=%v\n", time.Since(t0).Seconds(), res, err)
		if errors.Is(err, combat.ErrDead) {
			dead = true
			break
		}
		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			break
		}
	}
	_ = skills
	s, _ = e.Snapshot()
	el := time.Since(t0)
	if s != nil && s.Player != nil {
		fmt.Printf("FARM area %d: %v elapsed, %d kills = %.1f/min, exp %d -> %d (+%d = %.0f XP/h), level %d -> %d, potions %d, ignored %d, dead=%v, life %d/%d mana %d/%d\n",
			area, el.Round(time.Second), tot.Kills, float64(tot.Kills)/el.Minutes(), startExp, s.Player.Exp, s.Player.Exp-startExp,
			float64(s.Player.Exp-startExp)/el.Hours(), startLvl, s.Player.Level, tot.Potions, len(tot.Ignored), dead, s.Player.Life, s.Player.MaxLife, s.Player.Mana, s.Player.MaxMana)
	}
}

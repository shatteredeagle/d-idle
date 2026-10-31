//go:build rigs

// combatclass: live verification of the class-agnostic combat (agent COMBATCLASS, run through
// tools/live.sh combatclass combatclass <sub>). Subcommands:
//
//	state                    hands, active aura, profile, hotkeys
//	bind                     EnsureHotkeys for the profile layout, KeepAura, print the result
//	setleft NAME             select a left skill through the left popup and verify
//	auras NAME1 NAME2 N      switch between two auras N times by hotkey, time each switch (needs bind first)
//	farm SECS [AREA]         leave town, explore and Clear for SECS seconds with the profile; kills/min, deaths
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/nav"
)

const install = "Games/d2lod/proton/pfx/drive_c/Program Files (x86)/Diablo II"

func die(err error) {
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
}

func main() {
	src, err := mem.OpenGame()
	die(err)
	defer src.Close()
	game := input.NewGame(input.NewClient(""), input.Reader{Src: src})
	home, _ := os.UserHomeDir()
	instDir := install
	if id := os.Getenv(mem.InstanceEnv); id != "" && id != "i0" {
		instDir = "Games/d2lod-inst/" + id + "/proton/pfx/drive_c/Program Files (x86)/Diablo II"
	}
	db, err := data.Open(home+"/"+instDir, home+"/d2lod-live/combatclass/data")
	die(err)
	tune := env.DefaultTuning()
	kills := 0
	e := &env.Env{ID: "combatclass", Log: slog.New(slog.NewTextHandler(os.Stdout, nil)), Mem: src, Read: d2.New(src), In: game, Data: db,
		Tune: &tune, Clock: env.RealClock{}, Emit: func(ev goals.Event) {
			if ev.Type == goals.EvKill {
				kills++
			}
		}}
	e.World = d2map.NewTracker(src)
	nv := nav.New(e)
	f := combat.New(e, nv)
	rows, err := combat.LoadListRows(home+"/"+instDir, db)
	die(err)
	f.SetListRows(rows)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	sub := "state"
	if len(os.Args) > 1 {
		sub = os.Args[1]
	}
	switch sub {
	case "state":
		state(e, f)
	case "bind":
		err := f.EnsureHotkeys(ctx)
		fmt.Println("EnsureHotkeys:", err, f.Hotkeys())
		s, _ := e.Snapshot()
		ok, err := f.KeepAura(ctx, s, nil, true)
		fmt.Println("KeepAura:", ok, err)
		state(e, f)
	case "setleft":
		cat := combat.NewCatalog(db)
		sk := cat.SkillByName(os.Args[2])
		if sk == nil {
			die(fmt.Errorf("unknown skill %s", os.Args[2]))
		}
		t0 := time.Now()
		fmt.Println("SetLeft:", f.SetLeft(ctx, sk.ID), time.Since(t0).Round(10*time.Millisecond))
		state(e, f)
	case "home": // walk back to the Rogue Encampment, close panels
		fmt.Println("ToArea(1):", nv.ToArea(ctx, 1))
		die(game.CloseAll())
		state(e, f)
	case "key": // key K...: press keys (potions are 1-4)
		for _, k := range os.Args[2:] {
			die(game.Key(k))
			time.Sleep(700 * time.Millisecond)
		}
		state(e, f)
	case "auras":
		auras(ctx, e, f, db)
	case "farm":
		secs, area := 300, 2
		if len(os.Args) > 2 {
			secs, _ = strconv.Atoi(os.Args[2])
		}
		if len(os.Args) > 3 {
			area, _ = strconv.Atoi(os.Args[3])
		}
		farm(ctx, e, f, nv, db, secs, area, &kills)
	default:
		fmt.Println("unknown subcommand", sub)
	}
}

func state(e *env.Env, f *combat.Fighter) {
	s, err := e.Snapshot()
	die(err)
	if s.Player == nil {
		fmt.Println("not in game", s.State)
		return
	}
	p := s.Player
	aura, ok := f.ActiveAura(p)
	fmt.Printf("player %s class %v lvl %d area %d pos (%d,%d) mode %d life %d/%d mana %d/%d left %d right %d activeAura %d (readable=%v)\n",
		p.Name, p.Class, p.Level, p.Area, p.X, p.Y, p.Mode, p.Life, p.MaxLife, p.Mana, p.MaxMana, p.LeftSkill, p.RightSkill, aura, ok)
	cat := combat.NewCatalog(e.Data)
	pr := combat.SelectProfile(p, cat, f.Build)
	fmt.Println("profile:", pr.Class, "melee:", pr.MeleeClass(), "attack:", pr.AttackNames(p.Level, func(string) bool { return true }))
	fmt.Println("hotkeys:", f.Hotkeys())
}

func auras(ctx context.Context, e *env.Env, f *combat.Fighter, db *data.DB) {
	if len(os.Args) < 5 {
		die(errors.New("usage: auras NAME1 NAME2 N"))
	}
	if err := f.EnsureHotkeys(ctx); err != nil {
		fmt.Println("EnsureHotkeys:", err)
	}
	cat := combat.NewCatalog(db)
	ids := [2]int{cat.SkillByName(os.Args[2]).ID, cat.SkillByName(os.Args[3]).ID}
	n, _ := strconv.Atoi(os.Args[4])
	s, _ := e.Snapshot()
	m0 := s.Player.Mana
	for i := 0; i < n && ctx.Err() == nil; i++ {
		id := ids[i%2]
		key := f.Hotkeys()[id]
		t0 := time.Now()
		die(e.In.Key(key))
		tk := time.Since(t0)
		var seen time.Duration
		for time.Since(t0) < 2*time.Second {
			cs, _ := e.Snapshot()
			if a, ok := f.ActiveAura(cs.Player); ok && a == id {
				seen = time.Since(t0)
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		fmt.Printf("switch %d to %d via %s: key call %v, aura visible after %v\n", i, id, key, tk.Round(time.Millisecond), seen.Round(time.Millisecond))
	}
	s, _ = e.Snapshot()
	fmt.Printf("mana before %d after %d\n", m0, s.Player.Mana)
}

func farm(ctx context.Context, e *env.Env, f *combat.Fighter, nv *nav.Navigator, db *data.DB, secs, area int, kills *int) {
	cat := combat.NewCatalog(db)
	s, err := e.Snapshot()
	die(err)
	if err := f.EnsureHotkeys(ctx); err != nil {
		fmt.Println("EnsureHotkeys:", err)
	}
	startExp, startLvl := s.Player.Exp, s.Player.Level
	if s.Player.InTown() {
		fmt.Println("ExitTo(2):", nv.ExitTo(ctx, 2))
	}
	if s, _ = e.Snapshot(); s.Player.Area != area {
		fmt.Println("ExitTo", area, nv.ExitTo(ctx, area))
	}
	t0 := time.Now()
	deaths, ignored := 0, 0
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
	for time.Since(t0) < time.Duration(secs)*time.Second && ctx.Err() == nil {
		xctx, xc := context.WithTimeout(ctx, 30*time.Second)
		xerr := nv.Explore(xctx, hostile)
		xc()
		if errors.Is(xerr, nav.ErrExplored) {
			fmt.Println("level explored")
			break
		}
		res, err := f.Clear(ctx, combat.ClearOpts{Radius: 50, Timeout: 60 * time.Second})
		ignored += len(res.Ignored)
		if res.Kills > 0 || len(res.Ignored) > 0 {
			fmt.Printf("[%3.0fs] %v err=%v\n", time.Since(t0).Seconds(), res, err)
		}
		if errors.Is(err, combat.ErrDead) {
			deaths++
			fmt.Println("DEAD, respawn:", f.Respawn(ctx, 20*time.Second))
			break // the bot walks the corpse run; this tool stops at the first death
		}
		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			break
		}
		if res.Kills == 0 && len(res.Ignored) == 0 {
			time.Sleep(500 * time.Millisecond)
		}
	}
	el := time.Since(t0)
	s, _ = e.Snapshot()
	if s != nil && s.Player != nil {
		p := s.Player
		fmt.Printf("FARM area %d: %v elapsed, %d kills = %.1f/min, exp %d -> %d (+%d), level %d -> %d, deaths %d, ignored %d, life %d/%d mana %d/%d left %d right %d\n",
			p.Area, el.Round(time.Second), *kills, float64(*kills)/el.Minutes(), startExp, p.Exp, int(p.Exp)-int(startExp), startLvl, p.Level, deaths, ignored,
			p.Life, p.MaxLife, p.Mana, p.MaxMana, p.LeftSkill, p.RightSkill)
	}
}

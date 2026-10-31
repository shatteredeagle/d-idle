package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
)

const chargedBolt = 38 // Skills.txt id

const castHeader = "area,visit,cb_level,clvl,monster,boss,xp,cast_cmds,clicks"

// counting wraps the bot's input so every skill-cast command of the real combat.Fighter is counted. A "cast
// command" is CastAt (hotkey cast); plain left/right Clicks (movement, melee, right-button casts) are counted
// apart so the two cannot be confused. Same definition as docs/d2d-casts.md (commands, not missile spawns).
type counting struct {
	input.Controller
	casts, clicks int
}

func (c *counting) CastAt(key string, p input.Point) error {
	c.casts++
	return c.Controller.CastAt(key, p)
}

func (c *counting) Click(b input.Button, x, y int, m ...input.Mod) error {
	c.clicks++
	return c.Controller.Click(b, x, y, m...)
}

// casts fights in one area with the bot's real Fighter for d and prints one row per kill (EvKill), the d2d columns.
// The character is prepared by the caller (save edit: clvl and Charged Bolt level, game restarted); god mode keeps
// life, mana and stamina full like the d2d probe's top-up.
func (r *rig) casts(ctx context.Context, area, cb, clvl int, d time.Duration) {
	stop := r.god(ctx, "casts per kill probe: keep life and mana full like the d2d probe's top-up (throwaway Testy)")
	defer func() { fmt.Fprintln(os.Stderr, "# god stop:", stop()) }()
	cc := &counting{Controller: r.e.In}
	r.e.In = cc
	r.nav = nil
	nv := newNav(r)
	f := combat.New(r.e, nv)
	rows, err := combat.LoadListRows(installDir(), r.db)
	if err != nil {
		die(err)
	}
	f.SetListRows(rows)
	fmt.Fprintln(os.Stderr, "# EnsureHotkeys:", f.EnsureHotkeys(ctx), f.Hotkeys())
	if s, err := r.e.Snapshot(); err == nil && s.Player != nil {
		fmt.Fprintf(os.Stderr, "# player lvl %d skills %v life %d/%d mana %d/%d\n", s.Player.Level, s.Player.Skills, s.Player.Life, s.Player.MaxLife, s.Player.Mana, s.Player.MaxMana)
		if s.Player.Level != clvl {
			fmt.Fprintf(os.Stderr, "# WARNING clvl %d, expected %d\n", s.Player.Level, clvl)
		}
	}
	if err := nv.ToArea(ctx, area); err != nil {
		die(err)
	}
	prev, prevClicks := 0, 0
	r.e.Emit = func(ev goals.Event) {
		if ev.Type != goals.EvKill {
			return
		}
		fmt.Printf("%d,%s,%d,%d,%q,%t,%d,%d,%d\n", area, os.Getenv("VISIT"), cb, clvl, ev.Monster, ev.Boss, ev.XP, cc.casts-prev, cc.clicks-prevClicks)
		prev, prevClicks = cc.casts, cc.clicks
	}
	fmt.Println(castHeader)
	end := time.Now().Add(d)
	for k := 0; ctx.Err() == nil && time.Now().Before(end); k++ {
		res, err := f.Clear(ctx, combat.ClearOpts{Radius: 60, Timeout: 30 * time.Second})
		fmt.Fprintf(os.Stderr, "# clear %d: %v err=%v casts=%d clicks=%d\n", k, res, err, cc.casts, cc.clicks)
		if res.Kills == 0 {
			if werr := r.wander(ctx, nv, k); werr != nil {
				fmt.Fprintf(os.Stderr, "# wander %d: %v\n", k, werr)
			}
		}
	}
}

func (r *rig) wander(ctx context.Context, nv navigator, k int) error {
	snap, err := r.e.Snapshot()
	if err != nil || snap.Player == nil {
		return err
	}
	p := snap.Player
	dirs := [][2]int{{1, 0}, {0, 1}, {-1, 0}, {0, -1}, {1, 1}, {-1, 1}, {-1, -1}, {1, -1}}
	for i := range dirs {
		d := dirs[(k+i)%len(dirs)]
		for _, dist := range []int{110, 90, 70} {
			to := input.Point{X: p.X + d[0]*dist, Y: p.Y + d[1]*dist}
			if _, perr := nv.PathTo(to); perr != nil {
				continue
			}
			wctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()
			return nv.GoTo(wctx, to)
		}
	}
	return fmt.Errorf("no reachable point around (%d,%d)", p.X, p.Y)
}

// manaRegen estimates the frame rate of the loaded Game.exe from mana regeneration (UNVERIFIED method: no frame
// counter is known). Mana comes back at max/3000 per FRAME (docs/d2d-tick-rate.md: all mana in 3000 frames, the
// original's rule, the premise there too), so measured gain per wall second divided by max/3000 is frames per
// wall second. No god mode (it refills mana). Spends mana in the Blood Moor with the real fighter, then stands still.
func (r *rig) manaRegen(ctx context.Context, d time.Duration) {
	cc := &counting{Controller: r.e.In}
	r.e.In = cc
	nv := newNav(r)
	f := combat.New(r.e, nv)
	rows, err := combat.LoadListRows(installDir(), r.db)
	if err != nil {
		die(err)
	}
	f.SetListRows(rows)
	fmt.Fprintln(os.Stderr, "# EnsureHotkeys:", f.EnsureHotkeys(ctx), f.Hotkeys())
	if err := nv.ToArea(ctx, 2); err != nil {
		die(err)
	}
	for k := 0; k < 12 && ctx.Err() == nil; k++ {
		s, err := r.e.Snapshot()
		if err != nil || s.Player == nil {
			die(fmt.Errorf("snapshot: %v", err))
		}
		if s.Player.Life*2 < s.Player.MaxLife {
			die(fmt.Errorf("life low (%d/%d), abort", s.Player.Life, s.Player.MaxLife))
		}
		if s.Player.Mana*2 <= s.Player.MaxMana {
			break
		}
		res, _ := f.Clear(ctx, combat.ClearOpts{Radius: 60, Timeout: 20 * time.Second})
		if res.Kills == 0 {
			_ = r.wander(ctx, nv, k)
		}
	}
	t0 := time.Now()
	type smp struct {
		t    float64
		mana int
	}
	var ss []smp
	for time.Since(t0) < d && ctx.Err() == nil {
		s, err := r.e.Snapshot()
		if err == nil && s.Player != nil {
			ss = append(ss, smp{time.Since(t0).Seconds(), s.Player.Mana})
			fmt.Printf("sample t=%.1f mana=%d/%d life=%d/%d\n", time.Since(t0).Seconds(), s.Player.Mana, s.Player.MaxMana, s.Player.Life, s.Player.MaxLife)
			if s.Player.MaxMana > 0 {
				maxMana = s.Player.MaxMana
			}
		}
		time.Sleep(time.Second)
	}
	if len(ss) < 2 || maxMana == 0 {
		return
	}
	a, b := ss[0], ss[len(ss)-1]
	rate := float64(b.mana-a.mana) / (b.t - a.t)
	exp := float64(maxMana) / 3000 * 25 // mana per second at exactly 25 frames per second
	fmt.Printf("RESULT mana %d -> %d in %.1fs: %.3f/s; expected at 25 fps %.3f/s; frames per wall second about %.1f (integer mana: +-1 point = +-%.1f)\n",
		a.mana, b.mana, b.t-a.t, rate, exp, rate/exp*25, 25/(rate*(b.t-a.t))*1)
}

var maxMana int

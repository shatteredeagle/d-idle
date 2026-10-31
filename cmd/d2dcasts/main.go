// Command d2dcasts measures casts per kill on a private lockstep d2d (docs/d2d-casts.md): a throwaway Sorceress whose
// only skill is Charged Bolt fights in one area with the bot's real combat.Fighter, and every kill event records the
// monster name and the number of cast commands and mana spent since the previous kill. Attach mode: the wrapper
// docs/probes/run-d2d.sh starts d2d (seed, --no-save) under the lab lock.
//
//	d2dcasts -sock S -tables DIR [-area 2] [-cb 1] [-clvl 3] [-seed N] [-virtual 20m] [-wall 6m]
//
// Cheats are the fork's logged `debug` verbs (level, skill level, life and mana), throwaway character only. Life and
// mana are topped up between Clear calls so every cast is affordable and no death ends a run.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2dcombat"
	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
)

const chargedBolt = 38 // Skills.txt id (V-d2d: docs/d2d-fidelity.md section 3, distinct mana cost 4 proves it)

func main() {
	sock := flag.String("sock", os.Getenv("D2D_SOCK"), "control socket")
	tables := flag.String("tables", "", "extracted game table dir (data.Load)")
	area := flag.Int("area", 2, "Levels.txt id to fight in (2 Blood Moor)")
	cb := flag.Int("cb", 1, "Charged Bolt skill level (debug points)")
	clvl := flag.Int("clvl", 3, "character level")
	seed := flag.Int("seed", 0, "seed label for the CSV rows")
	virt := flag.Duration("virtual", 20*time.Minute, "virtual time limit")
	wall := flag.Duration("wall", 6*time.Minute, "wall limit")
	flag.Parse()
	if *sock == "" || *tables == "" {
		fmt.Fprintln(os.Stderr, "d2dcasts: need -sock and -tables")
		os.Exit(2)
	}
	if err := run(*sock, *tables, *area, *cb, *clvl, *seed, *virt, *wall); err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		os.Exit(1)
	}
}

func topUp(ctx context.Context, c *d2dctl.Client, clvl int) error {
	life, mana := 300, 100
	for _, kv := range [][2]int{{6, life * 256}, {7, life * 256}, {8, mana * 256}, {9, mana * 256}} {
		if err := c.SetStat(ctx, kv[0], kv[1]); err != nil {
			return err
		}
	}
	return nil
}

func run(sock, tables string, area, cb, clvl, seed int, virt, wall time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), wall)
	defer cancel()
	db, err := data.Load(tables)
	if err != nil {
		return err
	}
	c := d2dctl.New(sock, d2dctl.Options{Timeout: 20 * time.Second})
	defer c.Close()
	if ci, err := c.Clock(ctx); err != nil || !ci.Lockstep {
		return fmt.Errorf("lockstep clock not available: %v", err)
	}
	if err := c.SetStat(ctx, 12, clvl); err != nil {
		return err
	}
	if err := c.Points(ctx, chargedBolt, cb); err != nil {
		return err
	}
	// a fresh Sorceress may already know a starting skill (V-d2d pilot: Fire Bolt 36 was bound): clear it so only Charged Bolt exists
	if err := c.Points(ctx, 36, 0); err != nil {
		fmt.Fprintln(os.Stderr, "# points 36 0:", err)
	}
	if err := topUp(ctx, c, clvl); err != nil {
		return err
	}
	if _, err := c.Step(ctx, 5); err != nil {
		return err
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	rig, err := d2dcombat.New(d2dcombat.Config{Ctl: c, Ctx: ctx, Tables: db, Log: log, LinkWait: 3 * time.Minute})
	if err != nil {
		return err
	}
	if n, err := rig.BindSkills(); err != nil || n == 0 {
		return fmt.Errorf("bind skills: n=%d err=%v", n, err)
	}
	fmt.Fprintf(os.Stderr, "# hotkeys %v\n", rig.Fighter.Hotkeys())
	if snap, err := rig.Env.Snapshot(); err == nil && snap.Player != nil {
		fmt.Fprintf(os.Stderr, "# skills %+v\n", snap.Player.Skills)
	}
	if err := c.Goto(ctx, area); err != nil {
		return err
	}
	rig.Pace.Sleep(3 * time.Second)
	rig.World.Update()
	rig.Meter.Reset()

	prevCasts, prevMana := 0, 0
	rig.Env.Emit = func(ev goals.Event) {
		if ev.Type != goals.EvKill {
			return
		}
		st, tot := rig.In.Stats(), rig.Meter.Totals()
		casts := st.Casts[chargedBolt]
		fmt.Printf("%d,%d,%d,%d,%q,%t,%d,%d,%d\n", area, seed, cb, clvl, ev.Monster, ev.Boss, ev.XP, casts-prevCasts, tot.ManaSpent-prevMana)
		prevCasts, prevMana = casts, tot.ManaSpent
	}
	fmt.Println(CSVHeader)
	_, _, v0 := rig.Pace.Stats()
	for k := 0; ctx.Err() == nil; k++ {
		if _, _, v := rig.Pace.Stats(); v-v0 > virt {
			break
		}
		topUp(ctx, c, clvl)
		rig.Meter.Reset()
		res, err := rig.Fighter.Clear(ctx, combat.ClearOpts{Radius: 60, Timeout: 30 * time.Second})
		fmt.Fprintf(os.Stderr, "# clear %d: %v err=%v\n", k, res, err)
		if res.Kills == 0 {
			if werr := wander(ctx, rig, k); werr != nil {
				fmt.Fprintf(os.Stderr, "# wander %d: %v\n", k, werr)
			}
		}
	}
	return nil
}

const CSVHeader = "area,seed,cb_level,clvl,monster,boss,xp,cast_cmds,mana_spent"

func wander(ctx context.Context, rig *d2dcombat.Rig, k int) error {
	snap, err := rig.Env.Snapshot()
	if err != nil || snap.Player == nil {
		return err
	}
	p := snap.Player
	dirs := [][2]int{{1, 0}, {0, 1}, {-1, 0}, {0, -1}, {1, 1}, {-1, 1}, {-1, -1}, {1, -1}}
	for i := range dirs {
		d := dirs[(k+i)%len(dirs)]
		for _, dist := range []int{110, 90, 70} {
			to := input.Point{X: p.X + d[0]*dist, Y: p.Y + d[1]*dist}
			if _, perr := rig.Nav.PathTo(to); perr != nil {
				continue
			}
			wctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()
			return rig.Nav.GoTo(wctx, to)
		}
	}
	return fmt.Errorf("no reachable point around (%d,%d)", p.X, p.Y)
}

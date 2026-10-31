// Command d2dfight is the smoke test of the bot's real combat layer on the d2d backend (docs/d2d-combat.md): it wires
// internal/d2dcombat to a private lockstep fork, optionally levels a THROWAWAY character with fork cheats, walks
// town -> Blood Moor -> Cold Plains and back with nav's combat Yield wired, and clears the packs it meets with
// combat.Fighter.Clear. It prints kills, damage taken, casts, mana use, deaths, virtual versus wall time.
//
//	d2dfight -sock ~/d2d-lab/fork/d2d.sock -tables ~/d2lod-live/combat/data [-clvl 12] [-route 2,3,2] [-wall 8m]
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2dcombat"
	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/input"
)

func main() {
	sock := flag.String("sock", os.Getenv("D2D_SOCK"), "d2d control socket (default $D2D_SOCK)")
	tables := flag.String("tables", "", "extracted game table cache dir (data.Load), required")
	route := flag.String("route", "2,3,2", "level ids to ExitTo in order after starting in town (1 town, 2 Blood Moor, 3 Cold Plains)")
	wall := flag.Duration("wall", 8*time.Minute, "wall-clock limit for the whole run")
	clear := flag.Duration("clear", 60*time.Second, "virtual time limit of the Clear on arrival in each level")
	radius := flag.Int("radius", 40, "Clear radius in subtiles on arrival")
	linkWait := flag.Duration("linkwait", 3*time.Minute, "wall time to wait on arrival for the neighbour levels")
	clvl := flag.Int("clvl", 0, "PRIVATE FORK ONLY, throwaway character: cheat the level, skill and stat points up to this clvl (Sorceress build)")
	tele := flag.Bool("goto", false, "PRIVATE FORK ONLY: enter each route level with `debug goto` instead of nav ExitTo (needs no layout verb), then wander with nav.GoTo")
	wander := flag.Int("wander", 4, "with -goto: walks of 70+ subtiles per level (nav combat Yield is what clears the way)")
	verbose := flag.Bool("v", false, "debug logs")
	flag.Parse()
	if *sock == "" || *tables == "" {
		fmt.Fprintln(os.Stderr, "d2dfight: need -sock and -tables")
		os.Exit(2)
	}
	var legs []int
	for _, f := range strings.Split(*route, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil {
			fmt.Fprintln(os.Stderr, "d2dfight: bad -route:", err)
			os.Exit(2)
		}
		legs = append(legs, n)
	}
	os.Exit(run(*sock, *tables, legs, *wall, *clear, *radius, *linkWait, *clvl, *tele, *wander, *verbose))
}

func run(sock, tables string, legs []int, wall, clearFor time.Duration, radius int, linkWait time.Duration, clvl int, tele bool, wander int, verbose bool) int {
	lvl := slog.LevelInfo
	if verbose {
		lvl = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
	ctx, cancel := context.WithTimeout(context.Background(), wall)
	defer cancel()
	db, err := data.Load(tables)
	if err != nil {
		fmt.Println("FAIL tables:", err)
		return 1
	}
	c := d2dctl.New(sock, d2dctl.Options{Timeout: 20 * time.Second})
	defer c.Close()
	if ci, err := c.Clock(ctx); err != nil || !ci.Lockstep {
		fmt.Printf("FAIL lockstep clock not available (err=%v): start the fork with --lockstep\n", err)
		return 1
	}
	if clvl > 0 {
		if err := cheatSorc(ctx, c, clvl); err != nil {
			fmt.Println("FAIL cheat:", err)
			return 1
		}
	}
	rig, err := d2dcombat.New(d2dcombat.Config{Ctl: c, Ctx: ctx, Tables: db, Log: log, LinkWait: linkWait})
	if err != nil {
		fmt.Println("FAIL rig:", err)
		return 1
	}
	rig.Meter.Reset()
	n, err := rig.BindSkills()
	fmt.Printf("== bound %d skills (err=%v) hotkeys=%v\n", n, err, rig.Fighter.Hotkeys())
	if err != nil || n == 0 {
		return 1
	}
	rig.Pace.Sleep(2 * time.Second) // neighbour levels only link once the game has stepped (docs/d2d-walk.md)
	if _, err := rig.World.Update(); err != nil {
		fmt.Println("  world.Update:", err)
	} else if lm := rig.World.Current(); lm != nil {
		fmt.Printf("  map: rect %+v exits %d adjacent %d; grid: %s; borders: %s\n", lm.Rect, len(lm.Exits), len(lm.Adjacent), rig.World.LastGridNote, rig.World.LastBorderNote)
	}
	start := time.Now()
	kills, bosses, potionsSeen := 0, 0, 0
	ok := true
	for i, to := range legs {
		_, _, v0 := rig.Pace.Stats()
		w0 := time.Now()
		fmt.Printf("== leg %d: ExitTo(%d)\n", i+1, to)
		var err error
		if tele {
			if err = c.Goto(ctx, to); err == nil {
				rig.Meter.Reset()
				rig.Pace.Sleep(3 * time.Second)
				_, err = rig.World.Update()
			}
		} else {
			err = rig.Nav.ExitTo(ctx, to)
		}
		snap, _ := rig.Env.Snapshot()
		area := 0
		if snap != nil && snap.Player != nil {
			area = snap.Player.Area
		}
		_, _, v1 := rig.Pace.Stats()
		fmt.Printf("  leg %d: err=%v area=%d (want %d) virtual=%v wall=%v\n", i+1, err, area, to, (v1 - v0).Round(time.Millisecond), time.Since(w0).Round(time.Millisecond))
		if err != nil || area != to {
			ok = false
			break
		}
		// on arrival: fight what is near with the real Clear
		res, err := rig.Fighter.Clear(ctx, combat.ClearOpts{Radius: radius, Timeout: clearFor})
		fmt.Printf("  clear in area %d: %v err=%v\n", area, res, err)
		kills += res.Kills
		bosses += res.Bosses
		potionsSeen += res.Potions
		if err != nil {
			ok = false
			break
		}
		if tele {
			for k := 0; k < wander && ctx.Err() == nil; k++ {
				werr := wanderOnce(ctx, rig, k)
				fmt.Printf("  wander %d in area %d: err=%v\n", k+1, area, werr)
				if res, cerr := rig.Fighter.Clear(ctx, combat.ClearOpts{Radius: radius, Timeout: clearFor}); cerr == nil || res.Kills > 0 {
					kills += res.Kills
					bosses += res.Bosses
					fmt.Printf("    clear: %v err=%v\n", res, cerr)
				}
			}
		}
	}
	steps, ticks, virt := rig.Pace.Stats()
	tot := rig.Meter.Totals()
	st := rig.In.Stats()
	fmt.Printf("== summary: ok=%v kills=%d bosses=%d potions(result)=%d potions(belt verbs)=%d deaths=%d\n", ok, kills, bosses, potionsSeen, st.Potions, tot.Deaths)
	fmt.Printf("   damage taken=%d min life=%d mana spent=%d (samples %d)\n", tot.Damage, tot.MinLife, tot.ManaSpent, tot.Samples)
	var ids []int
	for id := range st.Casts {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	var cs []string
	total := 0
	for _, id := range ids {
		cs = append(cs, fmt.Sprintf("%d:%d", id, st.Casts[id]))
		total += st.Casts[id]
	}
	fmt.Printf("   casts=%d by skill id %s, clicks (melee/right) %d\n", total, strings.Join(cs, " "), st.Clicks)
	fmt.Printf("   virtual=%v wall=%v step calls=%d ticks=%d pacer err=%v\n", virt.Round(time.Millisecond), time.Since(start).Round(time.Millisecond), steps, ticks, rig.Pace.Err())
	if ok {
		return 0
	}
	return 1
}

// cheatSorc sets level, stat and skill points of a throwaway private-fork Sorceress (cold/fire bolt build): every call is
// a fork debug verb, nothing touches Game.exe saves.
func cheatSorc(ctx context.Context, c *d2dctl.Client, clvl int) error {
	if err := c.SetStat(ctx, 12, clvl); err != nil {
		return err
	}
	pts := clvl - 1 // skill points of a leveled sorceress (quests ignored)
	// Fire Bolt 36 (main), Ice Bolt 39, Frost Nova 44, Static Field 42, Frozen Armor 40
	plan := [][2]int{{36, 6}, {39, 3}, {44, 1}, {42, 1}}
	spent := 0
	for _, p := range plan {
		if spent+p[1] > pts {
			continue
		}
		if err := c.Points(ctx, p[0], p[1]); err != nil {
			fmt.Printf("  cheat: points %d: %v (continuing)\n", p[0], err)
			continue
		}
		spent += p[1]
	}
	// life and mana as if 4 stat points per level went to vitality/energy; max and current together (x256 stat units)
	life, mana := 40+clvl*10, 35+clvl*8
	for _, kv := range [][2]int{{6, life * 256}, {7, life * 256}, {8, mana * 256}, {9, mana * 256}} {
		if err := c.SetStat(ctx, kv[0], kv[1]); err != nil {
			return err
		}
	}
	fmt.Printf("  cheat: clvl %d, %d skill points spent, life %d mana %d (throwaway private-fork character)\n", clvl, spent, life, mana)
	return nil
}

// wanderOnce walks to the first reachable point 70 to 110 subtiles from the player, trying eight directions starting at k.
func wanderOnce(ctx context.Context, rig *d2dcombat.Rig, k int) error {
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
			err := rig.Nav.GoTo(wctx, to)
			cancel()
			return err
		}
	}
	return fmt.Errorf("no reachable point around (%d,%d)", p.X, p.Y)
}

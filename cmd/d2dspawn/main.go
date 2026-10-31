// Command d2dspawn is an attach-mode probe for a private lockstep d2d (docs/d2d-tick-rate.md, docs/d2d-spawns.md).
// It never starts a game: the wrapper docs/probes/run-spawns.sh starts d2d for one seed under the lab lock.
//
//	d2dspawn -sock S ticks              measure ticks per game second and the mana regen formula
//	d2dspawn -sock S -seed N spawns     count monsters per area (CSV rows on stdout)
//
// Cheats are the fork's logged `debug` verbs on a throwaway character (ticks sets level and mana only).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/d2dctl"
)

func main() {
	sock := flag.String("sock", os.Getenv("D2D_SOCK"), "control socket")
	seed := flag.Int("seed", 0, "seed label for the CSV rows")
	diag := flag.Bool("diag", false, "spawns: print per-warp list sizes to stderr")
	areas := flag.String("areas", "2,3,4,5,8", "Levels.txt ids (2 Blood Moor, 3 Cold Plains, 4 Stony Field, 5 Dark Wood, 8 Den of Evil)")
	flag.Parse()
	if *sock == "" || flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: d2dspawn -sock S [-seed N] ticks|spawns")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	c := d2dctl.New(*sock, d2dctl.Options{Timeout: 30 * time.Second})
	defer c.Close()
	var err error
	switch flag.Arg(0) {
	case "ticks":
		err = ticks(ctx, c)
	case "spawns":
		verbose = *diag
		err = spawns(ctx, c, *seed, *areas)
	default:
		err = fmt.Errorf("unknown mode %q", flag.Arg(0))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		os.Exit(1)
	}
}

var verbose bool

func rawStats(ctx context.Context, c *d2dctl.Client) ([]int, error) {
	r, err := c.Do(ctx, "dump stats")
	if err != nil {
		return nil, err
	}
	var d struct {
		Stats struct {
			Raw []int `json:"raw"`
		} `json:"stats"`
	}
	if err := json.Unmarshal([]byte(strings.Join(append(append([]string{}, r.Lines...), r.Rest), "")), &d); err != nil {
		return nil, err
	}
	return d.Stats.Raw, nil
}

// ticks: step 250 and 1500 ticks, report clock deltas, and compare the mana gain with the original formula
// (maxmana*256/3000 per frame, floored, in 1/256 units: all mana back in 3000 frames = 120 s at 25 fps; S, docs/d2d-fidelity.md 1a).
func ticks(ctx context.Context, c *d2dctl.Client) error {
	if _, err := c.Step(ctx, 2); err != nil {
		return err
	}
	for _, kv := range [][2]int{{12, 12}, {9, 250 * 256}, {8, 0}} {
		if err := c.SetStat(ctx, kv[0], kv[1]); err != nil {
			return err
		}
	}
	c.Step(ctx, 2)
	for _, n := range []int{250, 1500} {
		r0, err := rawStats(ctx, c)
		if err != nil {
			return err
		}
		a, err := c.Clock(ctx)
		if err != nil {
			return err
		}
		t0 := time.Now()
		b, err := c.Step(ctx, n)
		if err != nil {
			return err
		}
		wall := time.Since(t0)
		r1, err := rawStats(ctx, c)
		if err != nil {
			return err
		}
		gain := float64(r1[8]-r0[8]) / 256
		sec := float64(b.GameMS-a.GameMS) / 1000
		want := float64(250*256/3000) / 256 * float64(n)
		fmt.Printf("step %d: tick %d -> %d (+%d) game_ms %d -> %d (+%d = %.2f game s, %.1f ticks/s) mana +%.3f (formula at 25 ticks/s: %.3f, per game s %.3f) wall %v\n",
			n, a.Tick, b.Tick, b.Tick-a.Tick, a.GameMS, b.GameMS, b.GameMS-a.GameMS, sec, float64(b.Tick-a.Tick)/sec, gain, want, gain/sec, wall.Round(time.Millisecond))
	}
	return nil
}

// Row is one area of one seed.
type Row struct {
	Area, Seed                                    int
	Normal, Minion, Champion, Unique, SuperUnique int
	Names, BossNames                              string
}

func (r Row) CSV() string {
	return strings.Join([]string{strconv.Itoa(r.Area), strconv.Itoa(r.Seed), strconv.Itoa(r.Normal), strconv.Itoa(r.Minion),
		strconv.Itoa(r.Champion), strconv.Itoa(r.Unique), strconv.Itoa(r.SuperUnique), strconv.Itoa(r.Normal + r.Minion + r.Champion + r.Unique + r.SuperUnique),
		strconv.Quote(r.Names), strconv.Quote(r.BossNames)}, ",")
}

const CSVHeader = "area,seed,normal,minion,champion,unique,superunique,total,names,boss_names"

// Count classifies a de-duplicated monster set. The fork's boss column: "-" normal, minion, champion, unique (rare), superunique.
func Count(area, seed int, mons map[int]d2dctl.Monster) Row {
	r := Row{Area: area, Seed: seed}
	names := map[string]int{}
	var boss []string
	for _, m := range mons {
		switch m.Boss {
		case "", "-":
			r.Normal++
			names[m.Name]++
		case "minion":
			r.Minion++
			names[m.Name]++
		case "champion":
			r.Champion++
			boss = append(boss, "champion:"+m.Name)
		case "unique":
			r.Unique++
			boss = append(boss, "unique:"+m.Name)
		default:
			r.SuperUnique++
			boss = append(boss, m.Boss+":"+m.Name)
		}
	}
	var ns []string
	for n, k := range names {
		ns = append(ns, fmt.Sprintf("%s x%d", n, k))
	}
	sort.Strings(ns)
	sort.Strings(boss)
	r.Names, r.BossNames = strings.Join(ns, ";"), strings.Join(boss, ";")
	return r
}

func spawns(ctx context.Context, c *d2dctl.Client, seed int, areas string) error {
	if _, err := c.Step(ctx, 5); err != nil {
		return err
	}
	for _, f := range strings.Split(areas, ",") {
		a, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil {
			return err
		}
		mons, err := sweep(ctx, c, a)
		if err != nil {
			fmt.Fprintf(os.Stderr, "area %d seed %d: %v\n", a, seed, err)
			continue
		}
		fmt.Println(Count(a, seed, mons).CSV())
	}
	return nil
}

// enter arrives in area a: debug goto, except the Den (goto renders it black, docs/d2dctl.md): Blood Moor warp to 8.
func enter(ctx context.Context, c *d2dctl.Client, a int) error {
	if a != 8 {
		return c.Goto(ctx, a)
	}
	if err := c.Goto(ctx, 2); err != nil {
		return err
	}
	if _, err := c.Step(ctx, 50); err != nil {
		return err
	}
	ws, err := c.Warps(ctx)
	if err != nil {
		return err
	}
	for _, w := range ws.Items {
		if w.To == 8 {
			return c.Enter(ctx, w.Index)
		}
	}
	return fmt.Errorf("no warp to level 8 in the Blood Moor (warps %d)", len(ws.Items))
}

// sweep enters the area, then visits a grid of points over the level, collecting every monster once by unit id
// (rooms and their monsters only exist near the player, UNVERIFIED until the first run: the output states the coverage).
func sweep(ctx context.Context, c *d2dctl.Client, a int) (map[int]d2dctl.Monster, error) {
	if err := enter(ctx, c, a); err != nil {
		return nil, err
	}
	mons := map[int]d2dctl.Monster{}
	collect := func() error {
		ml, err := c.Monsters(ctx)
		if err != nil {
			return err
		}
		fresh := 0
		for _, m := range ml.Items {
			if _, ok := mons[m.Unit]; !ok && !m.Dead() {
				fresh++
			}
			if !m.Dead() {
				mons[m.Unit] = m
			}
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "  list %d new %d total %d\n", len(ml.Items), fresh, len(mons))
		}
		return nil
	}
	if _, err := c.Step(ctx, 60); err != nil {
		return nil, err
	}
	if err := collect(); err != nil {
		return nil, err
	}
	lv, err := c.Level(ctx)
	if err != nil {
		return nil, err
	}
	const gap = 30.0
	for y := gap / 2; y < float64(lv.H); y += gap {
		for x := gap / 2; x < float64(lv.W); x += gap {
			if c.Warp(ctx, x, y) != nil {
				continue
			}
			if verbose {
				l2, _ := c.Level(ctx)
				fmt.Fprintf(os.Stderr, "warp %.0f,%.0f -> level %d at %.0f,%.0f of %dx%d\n", x, y, l2.ID, l2.X, l2.Y, l2.W, l2.H)
			}
			if _, err := c.Step(ctx, 12); err != nil {
				return nil, err
			}
			if err := collect(); err != nil {
				return nil, err
			}
		}
	}
	return mons, nil
}

// Command d2dwalk is the first end-to-end smoke test of the d2d backend: it builds an env.Env from d2dsrc (reads),
// d2dworld (level maps) and d2dinput (writes) over one control socket, constructs the bot's real navigator and walks
// the character through level exits, stepping a lockstep d2d instead of sleeping (d2dinput.Pacer). It needs the
// private fork running with --lockstep and a character in town. See docs/d2d-walk.md.
//
//	d2dwalk -sock ~/d2d-lab/fork/d2d.sock [-route 2,3] [-wall 3m] [-virtual 10m] [-v]
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dinput"
	"d2lod-bot/internal/d2dsrc"
	"d2lod-bot/internal/d2dworld"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/nav"
)

// traced logs the verbs the navigator uses and the player position after each move command.
type traced struct {
	input.Controller
	pace  *d2dinput.Pacer
	pos   func() input.Point
	log   *slog.Logger
	n     int
	kill  func()
	trail []input.Point
}

func (t *traced) ClickWorld(w input.Point) error {
	if t.kill != nil {
		t.kill() // monsters spawn as rooms come alive near the player, so a one-off kill per leg is not enough
	}
	err := t.Controller.ClickWorld(w)
	t.n++
	p := t.pos()
	t.trail = append(t.trail, p)
	t.log.Info("input ClickWorld", "target", w, "player", p, "vt", t.vt(), "err", err)
	return err
}

func (t *traced) MoveTo(w input.Point, o input.MoveOpts) error {
	err := t.Controller.MoveTo(w, o)
	t.log.Info("input MoveTo", "target", w, "player", t.pos(), "vt", t.vt(), "err", err)
	return err
}

func (t *traced) Click(b input.Button, x, y int, m ...input.Mod) error {
	err := t.Controller.Click(b, x, y, m...)
	t.log.Info("input Click", "button", b, "x", x, "y", y, "err", err)
	return err
}

func (t *traced) Key(name string) error {
	err := t.Controller.Key(name)
	t.log.Info("input Key", "key", name, "err", err)
	return err
}

func (t *traced) vt() time.Duration { _, _, v := t.pace.Stats(); return v }

func main() {
	sock := flag.String("sock", os.Getenv("D2D_SOCK"), "d2d control socket (default $D2D_SOCK)")
	route := flag.String("route", "2,3", "level ids to ExitTo in order (1 town, 2 Blood Moor, 3 Cold Plains)")
	wall := flag.Duration("wall", 3*time.Minute, "wall-clock limit for the whole run")
	virt := flag.Duration("virtual", 10*time.Minute, "virtual (game) time limit per leg")
	chunk := flag.Int("chunk", 0, "max ticks per step call (0: 50)")
	cheat := flag.Int("life", 0, "PRIVATE FORK ONLY: set clvl 30 and max life and life to this value (x256 stat units) before the walk, so wandering monsters do not end the run")
	linkWait := flag.Duration("linkwait", 4*time.Minute, "wall time to wait on arrival for d2d to link the neighbour levels (V: 30 s to over 100 s on a loaded box)")
	clear := flag.Bool("clear", false, "PRIVATE FORK ONLY: `debug kill` the current level's monsters before every move command (nav has no combat here: a Fallen standing in a corridor stalls it)")
	verbose := flag.Bool("v", false, "debug-level nav logs")
	flag.Parse()
	if *sock == "" {
		fmt.Fprintln(os.Stderr, "d2dwalk: need -sock or $D2D_SOCK")
		os.Exit(2)
	}
	var legs []int
	for _, f := range strings.Split(*route, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil {
			fmt.Fprintln(os.Stderr, "d2dwalk: bad -route:", err)
			os.Exit(2)
		}
		legs = append(legs, n)
	}
	os.Exit(run(*sock, legs, *wall, *virt, *chunk, *verbose, *cheat, *linkWait, *clear))
}

func run(sock string, legs []int, wall, virt time.Duration, chunk int, verbose bool, life int, linkWait time.Duration, clearMobs bool) int {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	ctx, cancel := context.WithTimeout(context.Background(), wall)
	defer cancel()
	start := time.Now()

	c := d2dctl.New(sock, d2dctl.Options{Timeout: 20 * time.Second})
	defer c.Close()
	if life > 0 {
		for _, kv := range [][2]int{{12, 30}, {7, life * 256}, {6, life * 256}} {
			if err := c.SetStat(ctx, kv[0], kv[1]); err != nil {
				fmt.Println("FAIL setup: cheat SetStat:", err)
				return 1
			}
		}
		fmt.Printf("  cheat: clvl 30, life %d (throwaway private-fork character)\n", life)
	}
	pace := &d2dinput.Pacer{C: c, Ctx: ctx, Chunk: chunk}
	if ci, err := c.Clock(ctx); err != nil || !ci.Lockstep {
		fmt.Printf("FAIL setup: lockstep clock not available (err=%v clock=%+v): start the fork with --lockstep\n", err, ci)
		return 1
	}
	src := d2dsrc.New(c, d2dsrc.Options{Logf: func(string, ...any) {}})
	lp := 0
	var world *d2dworld.World
	world = d2dworld.New(d2dworld.Client{Client: c}, d2dworld.Options{Timeout: linkWait + 30*time.Second, LinkWait: linkWait,
		Sleep: func(d time.Duration) {
			time.Sleep(50 * time.Millisecond)
			pace.Sleep(d)
			if lp++; lp%40 == 0 {
				log.Info("waiting for neighbour levels", "polls", lp, "linked", world.LastLinked, "note", world.LastBorderNote)
			}
		}})
	ctl := d2dinput.New(c)
	ctl.Ctx, ctl.Clock = ctx, pace
	tr := &traced{Controller: ctl, pace: pace, log: log}
	tr.pos = func() input.Point { p, _ := ctl.Pos(); return p }

	tune := env.DefaultTuning()
	if clearMobs {
		tr.kill = func() { _, _ = c.Debug(ctx, "kill") }
	}
	e := &env.Env{ID: "d2dwalk", Log: log, Read: src, In: tr, Clock: pace, Tune: &tune}
	n := nav.New(e)
	n.UseWorld(world)

	report := func(what string) *d2.Snapshot {
		s, err := src.Snapshot()
		if err != nil || s.Player == nil {
			fmt.Printf("  %s: snapshot error %v\n", what, err)
			return nil
		}
		fmt.Printf("  %s: area %d pos (%d,%d) life %d/%d state %v clock %v\n", what, s.Player.Area, s.Player.X, s.Player.Y,
			s.Player.Life, s.Player.MaxLife, s.State, func() time.Duration { _, _, v := pace.Stats(); return v }())
		return s
	}
	fmt.Println("== d2dwalk start")
	s := report("start")
	if s == nil {
		return 1
	}
	if _, err := world.Update(); err == nil {
		if lm := world.Current(); lm != nil {
			fmt.Printf("  map: level rect %+v exits %d adjacent %d border-cells-by-level:", lm.Rect, len(lm.Exits), len(lm.Adjacent))
			for _, a := range lm.Adjacent {
				fmt.Printf(" %d(%d cells)", a.LevelNo, len(lm.BorderCells(a.LevelNo)))
			}
			fmt.Printf("\n  grid: %s; borders: %s\n", world.LastGridNote, world.LastBorderNote)
		}
	} else {
		fmt.Println("  world.Update:", err)
	}

	ok := true
	for i, to := range legs {
		legCtx, lc := context.WithTimeout(ctx, wall)
		_, _, v0 := pace.Stats()
		w0 := time.Now()
		fmt.Printf("== leg %d: ExitTo(%d) (map: %d border cells to it; %s)\n", i+1, to, func() int {
			if lm := world.Current(); lm != nil {
				return len(lm.BorderCells(to))
			}
			return -1
		}(), world.LastBorderNote)
		done := make(chan error, 1)
		go func() { done <- n.ExitTo(legCtx, to) }()
		var err error
		tick := time.NewTicker(200 * time.Millisecond)
	wait:
		for {
			select {
			case err = <-done:
				break wait
			case <-tick.C:
				if _, _, v := pace.Stats(); v-v0 > virt {
					lc()
				}
			}
		}
		tick.Stop()
		lc()
		_, _, v1 := pace.Stats()
		after := report("after")
		got := 0
		if after != nil {
			got = after.Player.Area
		}
		fmt.Printf("  leg %d result: err=%v area=%d (want %d) wall=%v virtual=%v\n", i+1, err, got, to, time.Since(w0).Round(time.Millisecond), (v1 - v0).Round(time.Millisecond))
		if err != nil || got != to {
			ok = false
			fmt.Printf("FAIL leg %d: %s\n", i+1, reason(err, got, to))
			break
		}
	}
	steps, ticks, v := pace.Stats()
	fmt.Printf("   world: grid fetches %d, link polls %d\n", world.GridFetches, world.LinkPolls)
	fmt.Printf("== summary: ok=%v ClickWorld calls=%d step calls=%d ticks=%d virtual=%v wall=%v pacer err=%v\n", ok, tr.n, steps, ticks, v.Round(time.Millisecond), time.Since(start).Round(time.Millisecond), pace.Err())
	fmt.Printf("   unknown snapshot fields: %v\n", src.UnknownCounts())
	if len(tr.trail) > 0 {
		fmt.Printf("   trail (every move command): %v\n", tr.trail)
	}
	if ok {
		return 0
	}
	return 1
}

func reason(err error, got, want int) string {
	switch {
	case err != nil:
		return "navigator error: " + err.Error()
	default:
		return fmt.Sprintf("navigator returned nil but the player is in area %d, not %d", got, want)
	}
}

// bossrun runs ONE boss script (docs/boss.md section 8) against the game instance picked by D2BOT_INSTANCE, from
// wherever the character stands, and prints the Result. Run it through tools/live.sh <agent> bossrun smith.
//
//	bossrun [--dry] [--timeout 15m] [--gamedir DIR] [--cache DIR] BOSS
//
// BOSS is countess, smith, andariel or bloodraven (or the exact script name). --dry opens the game read-only, resolves
// the script and prints the character and the script's gates; it sends no input.
// Exit codes: 0 killed, 1 run ended without a kill (error printed), 2 usage, 3 setup failed, 4 timed out, 5 died.
package main

import (
	"context"
	"d2lod-bot/internal/cheat"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"d2lod-bot/internal/boss"
	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/nav"
	"d2lod-bot/internal/town"
)

const (
	exitKilled  = 0
	exitFailed  = 1
	exitUsage   = 2
	exitSetup   = 3
	exitTimeout = 4
	exitDied    = 5
)

// aliases maps the short names of the command line to the script names.
var aliases = map[string]string{
	"countess": boss.NameCountess, "smith": boss.NameSmith, "andariel": boss.NameAndariel, "bloodraven": boss.NameBloodRaven,
}

func resolveName(arg string) (string, bool) {
	a := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(arg), " ", ""))
	if n, ok := aliases[a]; ok {
		return n, true
	}
	for _, n := range aliases {
		if strings.EqualFold(n, strings.TrimSpace(arg)) {
			return n, true
		}
	}
	return "", false
}

// leaveToTown is set by setup: with --god the run ends by portalling home while god mode still runs (V live: stopping god
// in the Barracks crowd killed Combi).
var leaveToTown func(context.Context) error

// yieldPinned counts nav-yield Clear calls that ended on the starting tile (see unpinned).
var yieldPinned pinState

func main() { os.Exit(realMain()) }

func realMain() int {
	dry := flag.Bool("dry", false, "read-only: print the character and the script gates, send no input")
	timeout := flag.Duration("timeout", 15*time.Minute, "hard limit for the whole run")
	gameDir := flag.String("gamedir", os.Getenv("D2BOT_GAMEDIR"), "Diablo II install (default: the instance's prefix)")
	cache := flag.String("cache", "/tmp/d2lod-bossrun-cache", "table cache directory")
	god := flag.String("god", "", "REASON: run under the logged god-mode cheat (throwaway characters only; masks the escape logic)")
	nokite := flag.Bool("nokite", false, "experiment: combat kiting and missile dodging off (Tune.Combat)")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: bossrun [--dry] [--timeout D] BOSS   (countess|smith|andariel|bloodraven)")
		return exitUsage
	}
	name, ok := resolveName(flag.Arg(0))
	if !ok {
		fmt.Fprintf(os.Stderr, "bossrun: unknown boss %q\n", flag.Arg(0))
		return exitUsage
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	e, r, err := setup(ctx, log, *gameDir, *cache, *dry)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bossrun: setup:", err)
		return exitSetup
	}
	if *nokite {
		e.Tune.Combat.KiteEnabled, e.Tune.Combat.DodgeRadius = 0, 0
	}
	if c, ok := e.Mem.(io.Closer); ok {
		defer c.Close()
	}
	if *dry {
		return dryRun(os.Stdout, e, r, name)
	}
	if *god != "" {
		gate := cheat.Gate{Allow: true, Env: cheat.DefaultEnv(), Actor: "BOSSLIVE2"}
		stop := gate.God(context.Background(), *god)
		defer func() { fmt.Fprintln(os.Stderr, "god stop:", stop()) }()
		time.Sleep(1500 * time.Millisecond)
	}
	code := execute(ctx, os.Stdout, r, name)
	if *god != "" && leaveToTown != nil {
		tctx, tc := context.WithTimeout(context.Background(), 45*time.Second)
		fmt.Fprintln(os.Stderr, "to town:", leaveToTown(tctx))
		tc()
	}
	return code
}

// setup builds the env and the runner the way d2bot's buildBehaviors does for the boss task (nav, combat, loot, town).
// d2bot's class setup (applyClass) is not repeated: combat's own skill choice and hotkey binding are used as they are.
func setup(ctx context.Context, log *slog.Logger, gameDir, cache string, dry bool) (*env.Env, *boss.Runner, error) {
	src, err := mem.OpenGame() // honors D2BOT_INSTANCE
	if err != nil {
		return nil, nil, err
	}
	if gameDir == "" {
		lc := launch.Defaults()
		if inst := os.Getenv("D2BOT_INSTANCE"); inst != "" && inst != "i0" {
			if home, err := os.UserHomeDir(); err == nil {
				if dir := filepath.Join(home, "Games", "d2lod-inst", inst); dirExists(dir) {
					lc.Root = dir
				}
			}
		}
		lc.Resolve()
		gameDir = lc.GameDir
	}
	db, err := data.Open(gameDir, filepath.Join(cache, "tables"))
	if err != nil {
		src.Close()
		return nil, nil, fmt.Errorf("game tables (gamedir %s): %w", gameDir, err)
	}
	tune := env.DefaultTuning()
	e := &env.Env{
		ID: env.InstanceID(os.Getenv("D2BOT_INSTANCE")), Log: log, Mem: src, Read: d2.New(src), World: d2map.NewTracker(src),
		Data: db, Tune: &tune, Clock: env.RealClock{}, Emit: func(goals.Event) {},
	}
	if dry {
		e.In = input.NewFake(input.Point{})
	} else {
		e.In = input.NewGame(input.NewClient(""), input.Reader{Src: src})
	}
	nv := nav.New(e)
	cb := combat.New(e, nv)
	cat := combat.NewCatalog(e.Data)
	nv.Yield = bot.PackYield(bot.YieldDeps{
		Timeout:  4 * time.Second, // short calls so unpinned can see a stuck character (a 20 s Clear is one long dodge loop)
		Snapshot: e.Snapshot,
		Friendly: func(s *d2.Snapshot, u d2.Unit) bool { return combat.IsFriendly(s, u, cat) },
		Clear: func(ctx context.Context, radius int, timeout time.Duration) error {
			return unpinned(e, &yieldPinned, func() error {
				_, err := cb.Clear(ctx, combat.ClearOpts{Radius: radius, Timeout: timeout, Yield: true})
				return err
			})
		},
	})
	if rows, err := combat.LoadListRows(gameDir, db); err != nil {
		log.Warn("skill list rows unavailable: hotkeys cannot be bound", "err", err)
	} else {
		cb.SetListRows(rows)
	}
	lt := loot.New(e, nv)
	tw := town.New(e, nv)
	tw.SetJudge(lt)
	if !dry {
		cli := input.NewClient("")
		flow := &launch.Flow{Mem: src, In: typer{input.NewOOG(cli), cli}, Log: func(s string) { log.Info("oog", "msg", s) }}
		cb.Leave = func(ctx context.Context) error { return flow.LeaveGame(ctx, 20*time.Second) }
		if err := cb.EnsureHotkeys(ctx); err != nil {
			log.Warn("hotkeys not ensured", "err", err)
		}
	}
	leaveToTown = toTown(e, tw)
	return e, newBossRunner(newBossGame(e, db, cb, nv, lt, leaveToTown)), nil
}

type typer struct {
	*input.OOG
	c *input.Client
}

func (t typer) TypeChar(r rune) error { return t.c.TypeChar(r) }

func dirExists(p string) bool { st, err := os.Stat(p); return err == nil && st.IsDir() }

// killer is the part of boss.Runner the run needs (a fake in the tests).
type killer interface {
	KillBoss(ctx context.Context, name string) error
}

// exitCode maps a KillBoss result to the process exit code.
func exitCode(res boss.Result, err error) int {
	switch {
	case err == nil && res.Killed:
		return exitKilled
	case errors.Is(err, boss.ErrTimeout), errors.Is(err, context.DeadlineExceeded):
		return exitTimeout
	case errors.Is(err, boss.ErrDead):
		return exitDied
	}
	return exitFailed
}

// execute runs the boss once and prints the Result; w gets the human summary.
func execute(ctx context.Context, w io.Writer, r *boss.Runner, name string) int {
	return runKill(ctx, w, r, func() boss.Result { return r.Last }, name)
}

func runKill(ctx context.Context, w io.Writer, k killer, last func() boss.Result, name string) int {
	err := k.KillBoss(ctx, name)
	res := last()
	fmt.Fprintf(w, "boss=%q killed=%v loot=%d phases=%s elapsed=%s\n", name, res.Killed, res.Loot, strings.Join(res.Phases, ","), res.Elapsed.Round(time.Second))
	for _, n := range res.Notes {
		fmt.Fprintln(w, "note:", n)
	}
	if err != nil {
		fmt.Fprintln(w, "err:", err)
	}
	code := exitCode(res, err)
	fmt.Fprintln(w, "exit:", code)
	return code
}

// dryRun prints the character and the script's gates without touching the game.
func dryRun(w io.Writer, e *env.Env, r *boss.Runner, name string) int {
	sc, ok := r.Scripts[name]
	if !ok {
		fmt.Fprintf(w, "no script %q\n", name)
		return exitFailed
	}
	fmt.Fprintf(w, "script %q: act %d, waypoint area %d, route %v, arena %d, timeout %s, min potions %d, min level %v\n",
		sc.Name, sc.Act, sc.WP, sc.Route, sc.Arena, sc.Timeout, sc.MinPotions, sc.MinLevel)
	s, err := e.Snapshot()
	if err != nil || s.Player == nil {
		fmt.Fprintln(w, "snapshot:", err)
		return exitFailed
	}
	fmt.Fprintf(w, "state=%v area=%d level=%d life=%d/%d town=%v\n", s.State, s.Player.Area, s.Player.Level, s.Player.Life, s.Player.MaxLife, s.Player.InTown())
	return exitKilled
}

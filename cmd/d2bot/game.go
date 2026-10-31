package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/mem"
)

// gameConn is a connection to a running game: its memory, a liveness probe and the health state machine.
type gameConn struct {
	src    mem.Source
	alive  func() bool
	pid    func() int // current game pid, 0 when gone (metrics)
	health *launch.Monitor
	game   *launch.Game // managed mode only
	helper *launch.Helper
	swap   *swapSource // managed mode: the memory source the bot reads through; Swap re-attaches after a recycle
	close  func()
}

// openGame attaches to a running game (--attach) or launches/attaches through internal/launch (--managed).
// It never launches anything in attach mode.
func openGame(ctx context.Context, o options, log *slog.Logger) (*gameConn, error) {
	if o.attach {
		p, err := mem.OpenGame()
		if err != nil {
			return nil, fmt.Errorf("attach: %w (is the game running, and are we allowed to read it? see docs/launch.md)", err)
		}
		log.Info("attached to the game", "pid", p.PID)
		mon := &launch.Monitor{P: launch.Probes{
			Alive: func() (int, bool) {
				if pid, err := mem.FindGame(); err == nil {
					return pid, true
				}
				return 0, false
			},
			Heartbeat: launch.CPUTicks,
			CrashDialog: func() bool {
				ps, err := launch.ListProcs()
				return err == nil && len(launch.FindImage(ps, "blizzarderror.exe")) > 0
			},
		}}
		pidf := func() int { pid, _ := mem.FindGame(); return pid }
		return &gameConn{src: p, alive: p.Alive, pid: pidf, health: mon, close: func() { p.Close() }}, nil
	}

	// managed: this process becomes the subreaper so the game it starts stays readable without capabilities.
	launch.CloseOnExecInherited()
	if err := launch.BecomeSubreaper(); err != nil {
		return nil, err
	}
	cfg, err := managedConfig(ctx, o)
	if err != nil {
		return nil, err
	}
	sp := launch.NewSpawner()
	helper := launch.NewHelper(cfg, sp)
	game := launch.NewGame(cfg, sp)
	game.WindowUp = func() bool { _, up, _, _ := helper.Ping(time.Second); return up }
	if _, ok := game.Find(); ok && o.cleanStart {
		// the previous d2bot died (a restart by --supervise): its game may sit at the death screen with a stale state
		log.Warn("clean start: stopping the game left over from the previous d2bot")
		if err := cleanStop(ctx, game, cfg, log); err != nil {
			return nil, fmt.Errorf("clean start: %w", err)
		}
	}
	// No game of this instance: whatever is left in its prefix (lingering wineserver, orphaned winedevice) is stale. This
	// also ends the helper, hence before helper.Ensure.
	if err := game.PrepareLaunch(); err != nil {
		log.Warn("prefix cleanup skipped", "err", err)
	}
	if err := helper.Ensure(ctx, 60*time.Second); err != nil {
		return nil, fmt.Errorf("input helper: %w", err)
	}
	if _, ok := game.Find(); !ok {
		log.Info("launching the game")
		if _, err := game.Launch(ctx, 90*time.Second); err != nil {
			return nil, fmt.Errorf("launch: %w", err)
		}
	}
	pid, _, rd, err := game.Attach()
	if err != nil {
		return nil, err
	}
	log.Info("managed game attached", "pid", pid)
	go func() { // reap re-parented zombies while we are the subreaper
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				launch.Reap()
			}
		}
	}()
	swap := &swapSource{s: rd, pid: func() int { p, _ := game.Find(); return p.PID }}
	return &gameConn{
		src: swap, swap: swap, helper: helper, game: game, health: &launch.Monitor{P: launch.DefaultProbes(game)},
		alive: func() bool { _, ok := game.Find(); return ok },
		pid:   func() int { p, _ := game.Find(); return p.PID },
		close: func() { swap.Close() },
	}, nil
}

// managedConfig is the launch config of --managed. For a dev instance id (i1..i9, docs/multi.md) it is that
// instance's own prefix, helper port and private Xvfb display (created if missing), so `D2BOT_INSTANCE=i4
// d2bot --managed` drives the same game i4 that tools/game.sh would start. Any other id (the default
// "main") uses --data as the Wine root, as before.
func managedConfig(ctx context.Context, o options) (launch.Config, error) {
	cfg := launch.Defaults()
	if in := (launch.Instance{ID: o.instance}); launch.ValidID(o.instance) && in.Num() > 0 {
		in.Display = fmt.Sprintf(":%d", 100+in.Num())
		cfg = in.Config(cfg)
		xv := launch.XvfbDisplay{Sys: launch.OSSystem{}, Num: 100 + in.Num(), ID: o.instance}
		// children must not inherit the desktop session (same as Life.childEnv)
		cfg.ExtraEnv = append(cfg.ExtraEnv, "WAYLAND_DISPLAY=", "XAUTHORITY=")
		if err := cfg.Resolve(); err != nil {
			return cfg, err
		}
		xv.LogDir = cfg.LogDir
		if err := xv.Ensure(ctx); err != nil {
			return cfg, err
		}
	} else {
		cfg.Root = o.data
	}
	if o.helper != "" {
		cfg.HelperAddr = o.helper
	}
	return cfg, cfg.Resolve()
}

// cleanStopWait bounds each phase of the clean start: Wine often needs more than the 4 s of a normal shutdown, and a
// relaunch while the old wineserver is still winding down fails or adopts the old game.
const cleanStopWait = 20 * time.Second

// cleanStop stops the game left over by a crashed d2bot, waits for the prefix's wineserver to exit (bounded) and fails
// when the old game is still there afterwards.
func cleanStop(ctx context.Context, game *launch.Game, cfg launch.Config, log *slog.Logger) error {
	if err := game.Shutdown(cleanStopWait); err != nil {
		return err
	}
	wctx, cancel := context.WithTimeout(ctx, cleanStopWait)
	defer cancel()
	if err := cfg.WineserverWait(wctx); err != nil {
		log.Warn("clean start: wineserver did not exit in time, killing it", "err", err)
		_ = cfg.WineserverKill()
	}
	if _, ok := game.Find(); ok {
		return errors.New("the old game process is still running after the stop")
	}
	return nil
}

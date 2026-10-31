package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/metrics"
)

// swapSource is the game memory the bot reads through. A recycle starts a new Game.exe: the bot keeps its readers and
// only the file behind them changes.
type swapSource struct {
	mu  sync.RWMutex
	s   mem.Source
	pid func() int // the game's pid (managed mode): the server-side quest data is found by scanning its mappings
}

// Regions implements d2.RegionLister: the quest memory source (internal/questmem) scans the game's private mappings, and a
// plain file reader cannot say where they are.
func (w *swapSource) Regions() []d2.Region {
	if w.pid == nil {
		return nil
	}
	rs, _ := d2.RegionsOfPID(w.pid())
	return rs
}

func (w *swapSource) ReadAt(p []byte, addr uint32) error {
	w.mu.RLock()
	s := w.s
	w.mu.RUnlock()
	return s.ReadAt(p, addr)
}

// Swap installs the new source and closes the old one when it is closable.
func (w *swapSource) Swap(n mem.Source) {
	w.mu.Lock()
	old := w.s
	w.s = n
	w.mu.Unlock()
	if c, ok := old.(io.Closer); ok {
		c.Close()
	}
	if m, ok := old.(launch.MemReader); ok && m.F != nil {
		m.F.Close()
	}
}

func (w *swapSource) Close() {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if m, ok := w.s.(launch.MemReader); ok && m.F != nil {
		m.F.Close()
	}
}

// newRecycler builds the per-instance Recycler for a managed game (docs/soak.md): Leave = Save and Exit, Stop/Start = the
// Game, Login = re-attach the memory reader and walk the menus back into the character's game. Nil in attach mode: the
// game is not ours to restart.
func newRecycler(g *gameConn, cy cycler, e *env.Env, log *slog.Logger, instance string, ref *runnerRef) *launch.Recycler {
	if g.game == nil || g.swap == nil {
		return nil
	}
	rc := launch.NewRecycler()
	rc.Log = func(s string) { log.Info("recycle", "msg", s) }
	rc.Progress = func(string) { ref.beat() }
	rc.Guard = instanceGuard(instance)
	rc.FDs = func() (int, error) {
		pid := g.pid()
		if pid <= 0 {
			return -1, fmt.Errorf("no game pid")
		}
		return launch.CountFDs(pid)
	}
	diff := "normal"
	rc.Leave = func(ctx context.Context) error {
		if s, err := e.Snapshot(); err == nil && s.Player != nil {
			diff = difficultyName(s.Player.Difficulty)
		}
		return cy.flow.LeaveGame(ctx, 30*time.Second)
	}
	rc.Stop = func(ctx context.Context) error { return g.game.Shutdown(15 * time.Second) }
	rc.Start = func(ctx context.Context) error {
		if err := g.game.PrepareLaunch(); err != nil { // also ends the helper: ensure it afterwards
			log.Warn("recycle: prefix cleanup skipped", "err", err)
		}
		if err := g.helper.Ensure(ctx, 60*time.Second); err != nil {
			return fmt.Errorf("input helper: %w", err)
		}
		if _, err := g.game.Launch(ctx, 90*time.Second); err != nil {
			return err
		}
		_, _, rd, err := g.game.Attach()
		if err != nil {
			return err
		}
		g.swap.Swap(rd)
		return nil
	}
	rc.Login = func(ctx context.Context) error { return cy.EnterGame(ctx, diff) }
	return rc
}

// difficultyName is the OOG flow's difficulty word for a d2 difficulty.
func difficultyName(d d2.Difficulty) string {
	return [...]string{"normal", "nightmare", "hell"}[min(max(int(d), 0), 2)]
}

// recycleDue adapts a Recycler to bot.Deps.RecycleDue.
func recycleDue(rc *launch.Recycler) func() (bool, string) {
	if rc == nil {
		return nil
	}
	return rc.Due
}

// recycleDep adapts a Recycler to bot.Deps.Recycle.
func recycleDep(rc *launch.Recycler) func(ctx context.Context) (bool, string, error) {
	if rc == nil {
		return nil
	}
	return func(ctx context.Context) (bool, string, error) {
		_, why := rc.Due()
		did, err := rc.Maybe(ctx)
		return did, why, err
	}
}

// registerGameMetrics exports the game's open descriptor count (the soak charts it without /proc access) and the
// recycler's counters.
func registerGameMetrics(reg *metrics.Registry, pid func() int, rc *launch.Recycler) {
	reg.GaugeFunc("game_open_fds", "Open file descriptors of Game.exe (-1 when not running).", func() float64 {
		p := pid()
		if p <= 0 {
			return -1
		}
		n, err := launch.CountFDs(p)
		if err != nil {
			return -1
		}
		return float64(n)
	})
	if rc != nil {
		reg.GaugeFunc("game_recycles", "Game restarts by the recycler.", func() float64 { return float64(rc.Recycles()) })
	}
}

// instanceGuard returns the Recycler.Guard that takes the per-instance flock (the one tools/live.sh and the sentinel
// use) for the duration of a recycle, so a sentinel with --act and the bot never both launch Game.exe (S1). Non-blocking.
// When the lock is already held at startup, a wrapper (tools/live.sh) holds it for our whole life: our own
// open file description would conflict with it, so the guard is off (nil) in that case.
func instanceGuard(instance string) func() (func(), error) {
	if launch.LockHeld(instance) {
		return nil
	}
	return func() (func(), error) { return launch.LockInstance(instance, 0) }
}

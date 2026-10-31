package launch

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	denv "d2lod-bot/internal/env"
)

// Game manages Diablo II.exe / Game.exe. Launch from the directory GameDir (D2 loads its MPQs relative
// to the cwd) with `-w` (windowed). VERIFIED manually with tools/d2-prefix.sh game.
type Game struct {
	Cfg Config
	m   Managed
	// Reap overrides the winedevice reaping done after Shutdown (tests); the zero value is the real system.
	Reap ReapOpts
	// WindowUp reports whether the Diablo II window exists (the input helper's `info`); WindowWait bounds how long
	// Launch waits for it (default 45 s, U: typical time to window not measured). Nil WindowUp skips the check.
	WindowUp   func() bool
	WindowWait time.Duration
	Mounts     MountOpts           // network mount probe for the failure reason (tests); zero = real
	find       func() (Proc, bool) // test hook, default g.m.Find
}

// NewGame returns a Game for cfg (already Resolved). spawn may be nil.
func NewGame(cfg Config, spawn *Spawner) *Game {
	return &Game{Cfg: cfg, m: Managed{
		Cfg: cfg, Spawn: spawn, Exe: cfg.GameExe(), Args: cfg.GameArgs, Cwd: cfg.GameDir,
		Image: "game.exe", Log: "game.out",
	}}
}

// Launch starts the game and waits for the Game.exe process (not the window) to exist.
// NEVER call this while another game instance is running; it does not check beyond Find.
func (g *Game) Launch(ctx context.Context, timeout time.Duration) (Proc, error) {
	if p, ok := g.m.Find(); ok {
		return p, fmt.Errorf("game already running (pid %d)", p.PID)
	}
	if why := hungReason(ctx, g.Mounts); why != "" {
		if !g.Cfg.SandboxActive() {
			return Proc{}, errors.New("not launching: " + why)
		}
		slog.Warn("network mount hung, launching anyway: the Wine sandbox hides it", "mount", why)
	}
	if err := g.m.Start(ctx); err != nil {
		return Proc{}, err
	}
	p, err := g.m.WaitUp(ctx, timeout)
	if err != nil {
		return p, err
	}
	// A process is not a game: Game.exe can start and never create its window (V: G1 attempts 2/3, a hung
	// network mount; a stale prefix is another U cause). Treat that as a failed start and stop it, so the caller does not "attach" to it.
	if err := waitWindow(ctx, g.WindowUp, g.windowWait(), time.Sleep); err != nil {
		if why := hungReason(ctx, g.Mounts); why != "" && !g.Cfg.SandboxActive() {
			err = fmt.Errorf("%w: %s", err, why)
		}
		err = fmt.Errorf("%w (%s)", err, g.Cfg.SandboxStatus())
		g.Shutdown(5 * time.Second)
		return Proc{}, err
	}
	return p, nil
}

func (g *Game) windowWait() time.Duration {
	if g.WindowWait > 0 {
		return g.WindowWait
	}
	return 45 * time.Second
}

// waitWindow polls up until the Diablo II window exists; a nil up passes at once.
func waitWindow(ctx context.Context, up func() bool, wait time.Duration, sleep func(time.Duration)) error {
	if up == nil {
		return nil
	}
	for waited := time.Duration(0); ; waited += 500 * time.Millisecond {
		if up() {
			return nil
		}
		if waited >= wait {
			return fmt.Errorf("game started but no Diablo II window within %s", wait)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		sleep(500 * time.Millisecond)
	}
}

// PrepareLaunch cleans the prefix of an extra instance (i1..i9; never i0, never an unset instance) before a launch
// when no Game.exe of the instance runs: see CleanPrefix. It kills the instance's helper too. Off with D2BOT_REAP=0.
func (g *Game) PrepareLaunch() error {
	cfg := g.Cfg
	if !denv.ReapEnabled() || !ValidID(cfg.Instance) || cfg.Instance == "i0" || cfg.PrefixDir == "" {
		return nil
	}
	find := g.find
	if find == nil {
		find = g.m.Find
	}
	if _, ok := find(); ok {
		return fmt.Errorf("a game of %s is running: not cleaning its prefix", cfg.Instance)
	}
	o := g.Reap
	if o.Log == nil {
		o.Log = func(s string) { slog.Info(s) }
	}
	if o.ServerKill == nil {
		o.ServerKill = cfg.WineserverKill
	}
	o.GameRunning = func() bool { _, ok := find(); return ok }
	CleanPrefix(filepath.Join(cfg.PrefixDir, "pfx"), o)
	return nil
}

// Find returns the running Game.exe.
func (g *Game) Find() (Proc, bool) { return g.m.Find() }

// Attach finds Game.exe and opens its memory. It fails with a clear message if the process exists but is
// not a descendant of this process (the game was started by something else, or before the app became a
// subreaper), because /proc/pid/mem would be denied by Yama.
func (g *Game) Attach() (pid int, base uint32, mem MemReader, err error) {
	p, ok := g.m.Find()
	if !ok {
		return 0, 0, MemReader{}, fmt.Errorf("Game.exe not running")
	}
	b, err := ImageBase(p.PID, "Game.exe")
	if err != nil {
		return p.PID, 0, MemReader{}, err
	}
	f, err := OpenMem(p.PID)
	if err != nil {
		if ps, lerr := ListProcs(); lerr == nil && !IsDescendant(ps, p.PID, os.Getpid()) {
			return p.PID, 0, MemReader{}, fmt.Errorf("Game.exe pid %d is not our descendant (parent %d): it was started outside this app; stop it and let the app launch it: %w", p.PID, p.PPID, err)
		}
		return p.PID, 0, MemReader{}, err
	}
	return p.PID, uint32(b), MemReader{F: f}, nil
}

// Shutdown ends the game process: SIGTERM, then SIGKILL after grace. For a CLEAN exit that saves the
// character first leave the game through the OOG flow (SaveAndExit), then call this.
func (g *Game) Shutdown(grace time.Duration) error {
	err := g.m.Stop(grace)
	if err == nil {
		reapAfterStop(g.Cfg, g.Reap)
	}
	return err
}

// CPUTicks returns utime+stime (clock ticks, 100 Hz) of pid, summed over all its threads, from
// /proc/pid/stat. A running D2 client burns CPU constantly (VERIFIED: ~11 ticks/s idle in town),
// so a flat counter is a freeze signal that needs no game knowledge.
func CPUTicks(pid int) (uint64, error) {
	b, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return 0, err
	}
	s := string(b)
	k := strings.LastIndexByte(s, ')')
	if k < 0 {
		return 0, fmt.Errorf("bad stat")
	}
	f := strings.Fields(s[k+1:])
	if len(f) < 13 {
		return 0, fmt.Errorf("short stat")
	}
	u, _ := strconv.ParseUint(f[11], 10, 64)
	sy, _ := strconv.ParseUint(f[12], 10, 64)
	return u + sy, nil
}

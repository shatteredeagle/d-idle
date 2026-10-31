// Package d2denv builds the per-instance dependencies of the bot for the d2d backend (docs/d2d-bot.md): the reader
// (d2dsrc plus unit-id stats), input (d2dinput), the nav world (d2dworld), a clock that is virtual under d2d's lockstep
// fork and real otherwise, the process lifecycle, the quest source and the seams for the combat and town packages.
// Nothing here needs process memory or the Wine game.
package d2denv

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dinput"
	"d2lod-bot/internal/d2dsrc"
	"d2lod-bot/internal/d2dworld"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/nav"
)

// DefaultLinkWait is the arrival wait for neighbour levels. d2dworld's own default is no wait, which made the bot's
// nav report "no exit" on arrival (V live, docs/d2d-exit.md); d2dwalk crossed with 150 s, a few hundred stepped ticks
// are what linking needs.
const DefaultLinkWait = 90 * time.Second

// Config is everything Build needs. Zero values are sensible; Sock is required.
type Config struct {
	ID   env.InstanceID
	Log  *slog.Logger
	Tune *env.Tuning
	Data *data.DB // game tables; nil: item rows and monster classes stay unknown

	Sock string // control socket path (the one d2d listens on)
	// Dial, when set, is the socket the client connects to instead of Sock (a tap such as cmd/d2dview in front of d2d).
	Dial string
	// Process: Bin == "" attaches to a d2d that already listens on Sock (nothing is started or killed).
	Bin, DataDir, Home string
	Nice               int
	ExtraArgs          []string // --toggle autoloot=off,autoparty=off, --lockstep ...
	Spec               d2dinput.Spec
	LogFile            string // d2d stdout/stderr in managed mode ("" discards)

	LinkWait time.Duration // d2dworld neighbour-link wait (0 = DefaultLinkWait)
}

// Backend is the built set. Env is ready to hand to nav, combat, town and the bot; nothing talks to d2d before Start
// (the client dials lazily).
type Backend struct {
	Env    *env.Env
	Client *d2dctl.Client
	Source *d2dsrc.Source
	Reader *Reader
	World  *d2dworld.World
	Ctl    *d2dinput.Controller
	Proc   *Proc
	Cycler *Cycler
	Quests *QuestSource
	Clock  *LateClock
	log    *slog.Logger
	Beat   *Beat // liveness heartbeat for launch.Monitor (Probes.Heartbeat = func(int)(uint64, error){ return Beat.Next() })
}

// Build assembles the backend.
func Build(ctx context.Context, cfg Config) (*Backend, error) {
	if cfg.Sock == "" {
		return nil, errNoSock
	}
	if cfg.Log == nil {
		cfg.Log = slog.Default()
	}
	dial := cfg.Sock
	if cfg.Dial != "" {
		dial = cfg.Dial
	}
	c := d2dctl.New(dial, d2dctl.Options{Timeout: 20 * time.Second})
	var tables d2dsrc.Tables
	if cfg.Data != nil {
		tables = d2dsrc.NewDBTables(cfg.Data)
	}
	src := d2dsrc.New(c, d2dsrc.Options{Tables: tables, Logf: func(f string, a ...any) { cfg.Log.Info("d2dsrc", "msg", sprintf(f, a...)) }})
	rd := NewReader(src)
	clk := &LateClock{c: c, ctx: ctx}
	ctl := d2dinput.New(c)
	ctl.Ctx, ctl.Clock = ctx, clk
	ctl.Run = d2dinput.RunPolicyFromEnv() // run while stamina lasts; D2D_RUN=off restores the walk-only bot
	if cfg.LinkWait == 0 {
		cfg.LinkWait = DefaultLinkWait
	}
	w := d2dworld.New(d2dworld.Client{Client: c}, d2dworld.Options{LinkWait: cfg.LinkWait, Sleep: clk.Poll, Now: clk.Now,
		Logf: func(f string, a ...any) { cfg.Log.Info(sprintf(f, a...)) }})
	p := &Proc{C: c, Bin: cfg.Bin, Data: cfg.DataDir, Sock: cfg.Sock, Home: cfg.Home, Nice: cfg.Nice, NoSave: cfg.Spec.NoSave, ExtraArgs: cfg.ExtraArgs}
	if cfg.LogFile != "" && cfg.Bin != "" {
		p.LogPath = cfg.LogFile // opened (and the previous one rotated) at every start
	}
	p.OnInGame = ctl.ResetRun
	p.Logf = func(f string, a ...any) { cfg.Log.Warn(sprintf(f, a...)) }
	go traceLoop(ctx, c, clk, cfg.Log)
	openJournal(c, cfg.Log)
	e := &env.Env{ID: cfg.ID, Log: cfg.Log, Mem: noMem{}, Read: rd, In: ctl, Data: cfg.Data, Tune: cfg.Tune, Clock: clk}
	return &Backend{Env: e, Client: c, Source: src, Reader: rd, World: w, Ctl: ctl, Proc: p,
		Cycler: &Cycler{P: p, Spec: cfg.Spec, Reenter: true, OnCycle: w.ResetNext}, Quests: &QuestSource{Src: src, Now: clk.Now}, Clock: clk, Beat: &Beat{C: c}, log: cfg.Log}, nil
}

// Navigator is nav.New over the backend's Env with the d2d world installed.
func (b *Backend) Navigator() *nav.Navigator {
	n := nav.New(b.Env)
	n.UseWorld(b.World)
	return n
}

// Start brings d2d up (managed) or waits for it (attach), waits until a character is in a game and picks the clock.
func (b *Backend) Start(ctx context.Context, timeout time.Duration) error {
	if err := b.Cycler.EnterGame(ctx, ""); err != nil {
		return err
	}
	b.Clock.Detect(ctx)
	b.checkVerbs(ctx)
	return nil
}

// checkVerbs warns when the d2d binary lacks the fork's layout verbs: without them the world has no outdoor borders and
// nav reports "no exit" in every field (V live, D2DEXIT: a rebuild of the fork from a branch without the P3 patch did
// exactly that, and the only trace was an info-level note).
func (b *Backend) checkVerbs(ctx context.Context) {
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if _, err := (d2dworld.Client{Client: b.Client}).Layout(cctx); err != nil {
		b.log.Warn("d2d has no 'debug layout' verb (fork patch P3 missing): outdoor exits cannot be found, nav will say 'no exit'", "err", err)
	}
}

// Close ends the process side per mode and closes the channel. Save and Exit first with Cycler.LeaveGame.
func (b *Backend) Close() {
	_ = b.Proc.Stop()
	_ = b.Client.CloseJournal()
	_ = b.Client.Close()
}

// openJournal starts the command journal of the determinism finder when $D2DCTL_JOURNAL names a file; checkpoints every
// $D2DCTL_JOURNAL_EVERY ticks (default 250). docs/d2d-determinism.md.
func openJournal(c *d2dctl.Client, log *slog.Logger) {
	path := os.Getenv("D2DCTL_JOURNAL")
	if path == "" {
		return
	}
	every := 250
	if n, err := strconv.Atoi(os.Getenv("D2DCTL_JOURNAL_EVERY")); err == nil && n >= 0 {
		every = n
	}
	if err := c.OpenJournal(path, every); err != nil {
		log.Warn("d2dctl journal not opened", "path", path, "err", err)
		return
	}
	if raw := os.Getenv("D2DCTL_JOURNAL_RAW"); raw != "" {
		if f, err := os.Create(path + ".raw"); err == nil {
			c.RawReplies(f, strings.Split(raw, ",")...)
		}
	}
	log.Info("d2dctl journal on", "path", path, "every", every)
}

// LateClock is env.Clock and input.Clock whose kind is decided after d2d is up: with the lockstep fork Sleep steps the
// game (a Pacer, virtual time), on a free-running d2d it is the wall clock. Before Detect it is the wall clock.
type LateClock struct {
	c   *d2dctl.Client
	ctx context.Context
	p   atomic.Pointer[d2dinput.Pacer]
}

// Detect looks at "clock": lockstep gives the Pacer. Returns whether lockstep is on.
func (l *LateClock) Detect(ctx context.Context) bool {
	ci, err := l.c.Clock(ctx)
	if err != nil || !ci.Lockstep {
		l.p.Store(nil)
		return false
	}
	l.p.Store(&d2dinput.Pacer{C: l.c, Ctx: l.ctx})
	return true
}

// Pace reports the Pacer's step calls, stepped ticks and virtual time; ok is false on a wall clock.
func (l *LateClock) Pace() (steps, ticks int, virtual time.Duration, ok bool) {
	p := l.p.Load()
	if p == nil {
		return 0, 0, 0, false
	}
	steps, ticks, virtual = p.Stats()
	return steps, ticks, virtual, true
}

// LastStep is the wall time of the last successful lockstep step (zero on a wall clock or before the first).
func (l *LateClock) LastStep() time.Time {
	if p := l.p.Load(); p != nil {
		return p.LastStep()
	}
	return time.Time{}
}

// Lockstep reports whether the Pacer is active.
func (l *LateClock) Lockstep() bool { return l.p.Load() != nil }

func (l *LateClock) Now() time.Time {
	if p := l.p.Load(); p != nil {
		return p.Now()
	}
	return time.Now()
}

// Watch is the Pacer's Watch under lockstep (the runner's guard runs on the acting goroutine, docs/d2d-determinism.md);
// on a wall clock it returns nil and the caller keeps its own goroutine.
func (l *LateClock) Watch(every time.Duration, fn func()) (stop func()) {
	if p := l.p.Load(); p != nil {
		return p.Watch(every, fn)
	}
	return nil
}

func (l *LateClock) Sleep(d time.Duration) {
	if p := l.p.Load(); p != nil {
		p.Sleep(d)
		return
	}
	time.Sleep(d)
}

// linkPollStep and linkPollWall: one neighbour-link poll under lockstep. d2d links neighbour levels on a worker thread
// that runs in WALL time and hands its result to the main loop on a stepped frame (docs/d2d-walk.md section on
// debug nearby). The old poll stepped the full 500 ms (12 ticks) and returned after about a millisecond of wall time, so
// the worker's 100 to 300 ms of work cost 80 to 300 virtual SECONDS per arrival (V d2d logs m11 to m14: 11 to 16
// waits per 2 h run, 15 to 25 % of the virtual time, and the 60 s clearing watchdog fired during it). Two ticks plus
// 10 ms of real time per poll keep the same wall bound and cost about a virtual second (docs/d2d-economy.md).
const (
	linkPollStep = 80 * time.Millisecond
	linkPollWall = 10 * time.Millisecond
)

// Poll is the world source's wait between neighbour-link polls (see linkPollStep).
func (l *LateClock) Poll(d time.Duration) {
	if p := l.p.Load(); p != nil {
		p.Sleep(min(d, linkPollStep))
		time.Sleep(linkPollWall)
		return
	}
	time.Sleep(d)
}

// traceLoop logs the per-verb round-trip table and the pacer counters every $D2DCTL_TRACE (a Go duration, e.g. 60s);
// unset or invalid means off. It is the profiler of docs/d2d-pace.md.
func traceLoop(ctx context.Context, c *d2dctl.Client, clk *LateClock, log *slog.Logger) {
	cpuProfile(ctx, log)
	every, err := time.ParseDuration(os.Getenv("D2DCTL_TRACE"))
	if err != nil || every <= 0 {
		return
	}
	tk := time.NewTicker(every)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			steps, ticks, v, _ := clk.Pace()
			log.Info("d2dtrace", "steps", steps, "ticks", ticks, "virtual", v.Round(time.Second), "verbs", "\n"+c.TraceReport())
		}
	}
}

// cpuProfile writes a 40 s CPU profile of the bot to $D2DCTL_PPROF once the first minute has passed (go tool pprof -top).
func cpuProfile(ctx context.Context, log *slog.Logger) {
	path := os.Getenv("D2DCTL_PPROF")
	if path == "" {
		return
	}
	go wallStacks(ctx, path+".stacks")
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(20 * time.Second):
		}
		f, err := os.Create(path)
		if err != nil {
			return
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			return
		}
		select {
		case <-ctx.Done():
		case <-time.After(30 * time.Second):
		}
		pprof.StopCPUProfile()
		log.Info("d2dtrace: cpu profile written", "path", path)
	}()
}

// wallStacks samples every goroutine's stack every 50 ms for 40 s after the first 45 s: a wall-clock profile (where does
// the bot WAIT; the CPU profile only sees what it computes). One sample per blank-line separated block.
func wallStacks(ctx context.Context, path string) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(20 * time.Second):
	}
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()
	buf := make([]byte, 1<<20)
	end := time.Now().Add(30 * time.Second)
	for time.Now().Before(end) && ctx.Err() == nil {
		n := runtime.Stack(buf, true)
		f.Write(buf[:n])
		f.WriteString("\n=====\n")
		time.Sleep(50 * time.Millisecond)
	}
}

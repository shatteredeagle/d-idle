package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/api"
	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/event"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/server"
)

var errGameGone = errors.New("the game process is gone")

func serveHealth(ctx context.Context, log *slog.Logger, o options, h *health) {
	mux := http.NewServeMux()
	mux.Handle("/healthz", h.handler())
	hs := &http.Server{Addr: o.addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { <-ctx.Done(); hs.Close() }()
	if err := hs.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Warn("health endpoint", "err", err)
	}
}

// runLive is the real bot: game, env, behaviors, runner, goal manager and API.
func runLive(ctx context.Context, o options, log *slog.Logger) error {
	cfg, err := bot.LoadConfig(o.config)
	if err != nil {
		return err
	}
	ip, err := instancePaths(o, log)
	if err != nil {
		return err
	}
	if cfg.DeathLogPath == "" {
		cfg.DeathLogPath = ip.Deaths()
	}
	if !o.stubs && len(cfg.DisableObserve) == 0 {
		// combat emits Kill and Potion, loot emits Drop through env.Emit (docs/combat.md, docs/town.md): do not count twice
		cfg.DisableObserve = []string{"kill", "drop", "potion"}
	}
	tune, warns, err := cfg.BuildTuning()
	if err != nil {
		return err
	}
	for _, w := range warns {
		log.Warn("tuning knob skipped", "detail", w)
	}
	tuner, err := bot.NewTuner(&tune, cfg, filepath.Join(ip.State(), "tuning.json"))
	if err != nil {
		return err
	}
	g, err := openGame(ctx, o, log)
	if err != nil {
		return err
	}
	defer g.close()

	e := &env.Env{
		ID: env.InstanceID(o.instance), Log: log, Mem: g.src, Read: d2.New(g.src), World: d2map.NewTracker(g.src),
		Tune: &tune, Clock: env.RealClock{},
	}
	if o.stubs {
		e.In = input.NewFake(input.Point{}) // nothing reaches the game
	} else {
		e.In = input.NewGame(input.NewClient(o.helper), input.Reader{Src: g.src})
	}
	var db *data.DB
	gameDir := o.gameDir
	if gameDir == "" {
		lc := launch.Defaults()
		if o.managed {
			if mc, err := managedConfig(ctx, o); err == nil {
				lc = mc
			}
		} else if home, err := os.UserHomeDir(); err == nil {
			// attach mode: the instance's own prefix (docs/multi.md) when it exists, else the default install
			if dir := filepath.Join(home, "Games", "d2lod-inst", o.instance); o.instance != "i0" && dirExists(dir) {
				lc.Root = dir
			}
		}
		lc.Resolve()
		gameDir = lc.GameDir
	}
	if db, err = data.Open(gameDir, filepath.Join(o.data, "cache", "tables")); err != nil {
		log.Warn("game tables unavailable; running without a catalog (no kill/item names, no potion or inventory checks)", "gamedir", gameDir, "err", err)
		db = nil
	}
	e.Data = db
	var cat bot.Catalog
	if db != nil {
		cat = bot.NewDataCatalog(db)
	}

	lastName := o.char // the name is only readable while in a game: remember it
	var nameMu sync.Mutex
	charName := func() string {
		nameMu.Lock()
		defer nameMu.Unlock()
		if o.char == "" {
			if p, err := e.Read.Player(); err == nil && p.Name != "" {
				lastName = p.Name
			}
		}
		return lastName
	}
	charName() // read it now, while the game is up
	disp := newDisplay(cat)
	bus := event.NewBus(instanceID(o), nil)
	ref := &runnerRef{}
	bh, err := buildBehaviors(ctx, e, o, gameDir, db, charName, log, cfg, ref)
	if err != nil {
		return err
	}
	var rc *launch.Recycler
	var restart func(context.Context) error
	if cy, ok := bh.Cycler.(cycler); ok && !o.stubs {
		if rc = newRecycler(g, cy, e, log, instanceID(o), ref); rc != nil {
			cy.onCycle = rc.NoteCycle
			bh.Cycler = cy
			restart = func(ctx context.Context) error { return rc.Recycle(ctx, "failsafe restart") }
		}
	}
	runner, err := bot.New(e, bot.Deps{Recycle: recycleDep(rc), RecycleDue: recycleDue(rc), Restart: restart, Mover: bh.Mover, Fighter: bh.Fighter, Town: bh.Town, Loot: bh.Loot, Cycler: bh.Cycler, Leveler: bh.Leveler, Gear: bh.Gear, Quester: bh.Quester, Task: bh.Task, SavedExp: savedExpDep(saveDirOf(ctx, o), charName), Prepare: bh.prepare, Catalog: cat,
		Health: g.health, OnSnapshot: disp.OnSnapshot, Bus: bus}, cfg)
	if err != nil {
		return err
	}
	disp.run = runner
	ref.p.Store(runner)
	m, err := goals.New(goals.Options{Runner: runner, Path: ip.Goals(), Chain: taskChain(runner.LastSnapshot),
		Logf: func(f string, a ...any) { log.Info("goals", "msg", sprintf(f, a...)) }})
	if err != nil {
		return err
	}

	if err := enqueueGoals(m, o.goalSpec); err != nil {
		return err
	}
	if o.goalSpec != "" {
		log.Info("goals enqueued") // d2fleet marks the goals of a slot as seeded on this line
	}
	if o.goalSpec != "" || o.autostart || cfg.Autostart {
		m.Start()
	}
	if o.runFor > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.runFor)
		defer cancel()
	}

	if o.probeStates {
		go goSafe(ctx, log, "probe-states", func() { probeStates(ctx, e, log) })
	}
	var ready atomic.Bool
	ready.Store(true)
	inst := &server.Instance{
		ID: instanceID(o), Manager: m, Bus: bus, Hub: disp.hub,
		Health: func() (any, bool) {
			rep := runner.Health()
			return rep, g.alive() && rep.Fatal == "" && rep.Status != string(goals.RunnerError)
		},
		Ready:  ready.Load,
		Deaths: func() any { return runner.DeathReport() },
		State: func() any {
			m := map[string]any{"state": runner.State(), "status": runner.Status(),
				"panics": runner.Health().Panics, "goPanics": goPanics.Load()}
			if rc != nil {
				m["recycle"] = rc.Status()
			}
			return m
		},
		Mounts: map[string]http.Handler{"knobs": api.KnobsHandler(tuner), "upgrades": api.UpgradesHandler(tuner)},
	}
	reg := newRegistry()
	reg.RegisterProcs(map[string]func() int{"game": g.pid, "bot": os.Getpid})
	registerGameMetrics(reg, g.pid, rc)

	runCtx, stopRunner := context.WithCancel(context.Background())
	defer stopRunner()
	runnerDone := make(chan error, 1)
	if lim := watchdogLimit(); lim > 0 {
		go goSafe(runCtx, log, "watchdog", func() { newWatchdog(log, runner.LastStep, lim).run(runCtx) })
	}
	go goSafe(runCtx, log, "heapguard", func() {
		newHeapGuard(log, func() int { return tune.Bot.HeapLimitMB }, filepath.Join(ip.State(), "heapguard")).run(runCtx)
	})
	runnerFinished := make(chan struct{})
	go func() {
		var err error
		if !goSafe(runCtx, log, "runner", func() { err = runner.Run(runCtx) }) {
			err = &exitError{exitStorm, errors.New("runner goroutine panicked too often")}
		}
		runnerDone <- err
		close(runnerFinished)
	}()
	mgrCtx, stopMgr := context.WithCancel(context.Background())
	defer stopMgr()
	gaveUp := make(chan string, 4)
	crit := func(name string, f func()) { // the process cannot do without these: giving up ends it, the supervisor restarts it
		go func() {
			if !goSafe(mgrCtx, log, name, f) {
				gaveUp <- name
			}
		}()
	}
	crit("manager", func() { m.Run(mgrCtx) })
	go goSafe(mgrCtx, log, "log-events", func() { logEvents(mgrCtx, log, m) })
	go goSafe(mgrCtx, log, "bus-bridge", func() { busBridge(mgrCtx, m, bus) })
	go goSafe(mgrCtx, log, "display", func() { disp.feed(mgrCtx, m, bus) })

	apiCtx, stopAPI := context.WithCancel(context.Background())
	apiDone := make(chan error, 1)
	go func() { // the listener is bound once: a panic ends the API with an error (the supervisor restarts the process)
		var err error
		if v, st := runRecovered(func() { err = serve(apiCtx, log, o, inst, reg, "d2bot "+version) }); v != nil {
			log.Error("api server panic", "panic", fmt.Sprint(v), "stack", string(st))
			err = fmt.Errorf("panic: %v", v)
		}
		apiDone <- err
	}()

	var result error
	select {
	case <-ctx.Done():
		log.Info("shutting down: finishing in town")
	case err := <-runnerDone:
		result = haltError(err)
	case name := <-gaveUp:
		result = &exitError{exitStorm, fmt.Errorf("goroutine %s gave up after repeated panics", name)}
	case err := <-apiDone:
		result = fmt.Errorf("api server: %w", err)
	}
	// Graceful stop: ask the bot to finish in town, wait for it, save and exit, then close everything.
	sd := shutdownSeq{
		stopBot:    func() { m.Stop() },
		settled:    runner.Settled,
		settleWait: bot.SignalSettleWait,
		summary:    func() { logSummary(log, m, runner) },
		stopRunner: stopRunner,
		runnerDone: func() { waitClosed(runnerFinished, bot.RunnerDoneWait) },
		quiesce:    func() { waitClosed(runnerFinished, bot.RunnerQuiesce) },
		closeBot:   func() { stopMgr(); runner.Close(); m.Close(); stopAPI() },
		apiDone:    func() { waitDone(apiDone, bot.APIDoneWait) },
		shutdownGame: func() {
			log.Info("stopping the game")
			g.game.Shutdown(bot.GameShutdownWait)
		},
	}
	if o.stopGame && g.game != nil {
		if cy, ok := bh.Cycler.(cycler); ok {
			sd.leave = func() {
				lctx, lcancel := context.WithTimeout(context.Background(), bot.LeaveTimeout)
				defer lcancel()
				if err := cy.flow.LeaveGame(lctx, 25*time.Second); err != nil {
					log.Warn("save and exit before stopping the game failed", "err", err)
				}
			}
		}
		sd.stopGame = true
	}
	sd.run()
	return result
}

func waitClosed(ch <-chan struct{}, d time.Duration) {
	select {
	case <-ch:
	case <-time.After(d):
	}
}

func waitDone(ch <-chan error, d time.Duration) {
	select {
	case <-ch:
	case <-time.After(d):
	}
}

// shutdownSeq is the SIGTERM shutdown order. The save (.d2s: quests, waypoints) is
// known to be written on Save and Exit (V: area changes do not, docs/soak-analysis.md 6c), so it runs right after the runner settles and the
// runner is cancelled (it must not react to the menus), before the slow waits on the
// runner and API goroutines; the fleet SIGKILLs d2bot after its 70 s stop grace (budget: bot.ShutdownWorstCase).
type shutdownSeq struct {
	stopBot      func()
	settled      func() bool
	settleWait   time.Duration
	summary      func()
	stopRunner   func()
	leave        func() // Save and Exit; nil when the game is not ours
	quiesce      func() // waits (bounded) until the cancelled runner stopped sending input; may be nil
	runnerDone   func()
	closeBot     func()
	apiDone      func()
	stopGame     bool
	shutdownGame func()
}

func (s shutdownSeq) run() {
	s.stopBot()
	deadline := time.Now().Add(s.settleWait)
	for time.Now().Before(deadline) { // the runner walks home first (Stop = finish in town), then reports settled
		if s.settled() {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	s.summary()
	s.stopRunner()
	if s.leave != nil {
		if s.quiesce != nil {
			s.quiesce()
		}
		s.leave()
	}
	s.runnerDone()
	s.closeBot()
	s.apiDone()
	if s.stopGame && s.shutdownGame != nil {
		s.shutdownGame()
	}
}

func dirExists(p string) bool { st, err := os.Stat(p); return err == nil && st.IsDir() }

// haltError turns what ended the runner into the process result: a storm of panics is a crash (restart), anything else
// the runner returns is a deliberate fatal (do not restart).
func haltError(err error) error {
	var ee *exitError
	switch {
	case err == nil:
		return &exitError{exitStorm, errors.New("bot halted without a reason")}
	case errors.As(err, &ee):
		return err
	case errors.Is(err, bot.ErrPanicStorm):
		return &exitError{exitStorm, fmt.Errorf("bot halted: %w", err)}
	}
	return &exitError{exitFatal, fmt.Errorf("bot halted: %w", err)}
}

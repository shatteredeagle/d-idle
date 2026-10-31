package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/api"
	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2denv"
	"d2lod-bot/internal/d2dinput"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/event"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/questmem"
	"d2lod-bot/internal/server"
)

// The d2d backend (docs/d2d-bot.md): `d2bot --backend d2d` runs the same goal manager, planner, dashboard/API and
// shutdown sequence against a d2d process instead of Game.exe. --attach means a d2d already listens on the socket,
// --managed means d2bot starts (and on stop saves and quits) one.

type d2dOptions struct {
	backend   string // "" or "game": Game.exe memory backend, "d2d"
	sock      string
	dial      string // --d2d-dial: connect here instead of sock (a tap in front of d2d; --managed still launches d2d on sock)
	bin       string
	data      string
	class     int // d2d UI-order class of a NEW character; -1 loads the saved character --char from the select screen
	seed      int64
	noSave    bool
	toggle    string
	extraArgs string
	nice      int
	linkWait  time.Duration
	attach    bool // --d2d-attach: the confirmation that a d2d this bot did not start may be driven
}

func (d d2dOptions) on() bool { return d.backend == "d2d" }

func d2dFlags(fs *flag.FlagSet, d *d2dOptions) {
	fs.StringVar(&d.backend, "backend", envOr("D2BOT_BACKEND", "game"), "game backend: game (Game.exe memory, default) or d2d (native d2d over its control channel, docs/d2d-bot.md) (env D2BOT_BACKEND)")
	fs.StringVar(&d.sock, "d2d-sock", envOr("D2D_SOCK", ""), "d2d backend: control socket (env D2D_SOCK; default <run dir>/d2d-<instance>.sock)")
	fs.StringVar(&d.dial, "d2d-dial", envOr("D2D_DIAL", ""), "d2d backend: dial this socket instead of --d2d-sock (a tap such as cmd/d2dview listens here and forwards to d2d, which --managed still launches on --d2d-sock) (env D2D_DIAL)")
	fs.StringVar(&d.bin, "d2d-bin", envOr("D2D_BIN", ""), "d2d backend, --managed: path of the private d2d fork binary (env D2D_BIN)")
	fs.StringVar(&d.data, "d2d-data", envOr("D2D_DATA", ""), "d2d backend, --managed: d2d's --data dir (MPQs) (env D2D_DATA)")
	fs.IntVar(&d.class, "d2d-class", -1, "d2d backend, --managed: class index (d2d UI order, 1 = Necromancer) of a NEW character; -1 loads the saved --char")
	fs.Int64Var(&d.seed, "seed", 0, "d2d backend, --managed: map seed passed to d2d (0 = random layout); with --sim-seed-free runs this makes the world deterministic")
	fs.BoolVar(&d.noSave, "no-save", false, "d2d backend, --managed: run d2d with --no-save (a throwaway character that is never written)")
	fs.StringVar(&d.toggle, "toggle", "", "d2d backend, --managed: d2d QoL toggles, e.g. autoloot=off,autoparty=off (passed as --toggle)")
	fs.StringVar(&d.extraArgs, "d2d-args", "", "d2d backend, --managed: extra d2d arguments, space separated (e.g. --lockstep)")
	fs.IntVar(&d.nice, "d2d-nice", 10, "d2d backend, --managed: run d2d under nice -n (0 = off)")
	fs.BoolVar(&d.attach, "d2d-attach", envBool("D2D_ATTACH"), "d2d backend, --attach: confirm that the d2d on the socket is yours; --attach alone is refused (one control client per d2d, a stranger's d2d would be driven) (env D2D_ATTACH)")
	fs.DurationVar(&d.linkWait, "d2d-linkwait", 0, "d2d backend: how long arriving in a level waits for d2d to link the neighbour levels (0 = default)")
}

func (d *d2dOptions) validate(o options) error {
	switch d.backend {
	case "", "game":
		d.backend = ""
		return nil
	case "d2d":
	default:
		return fmt.Errorf("--backend %q: want game or d2d", d.backend)
	}
	if o.sim {
		return errors.New("--backend d2d does not combine with --sim")
	}
	if o.dry {
		return errors.New("--dry is not available on the d2d backend yet (it reads game memory)")
	}
	if o.attach && !d.attach {
		return errors.New("--backend d2d --attach drives a d2d this bot did not start: add --d2d-attach to confirm the socket is yours (docs/d2d-wire.md)")
	}
	if o.managed && d.bin == "" {
		return errors.New("--backend d2d --managed needs the d2d binary (--d2d-bin or D2D_BIN)")
	}
	return nil
}

// d2dSock is the control socket: the flag/env, else one per instance in the run dir.
func d2dSock(o options) string {
	if o.d2d.sock != "" {
		return o.d2d.sock
	}
	run := o.layout.Run()
	if run == "" {
		run = os.TempDir()
	}
	return filepath.Join(run, "d2d-"+instanceID(o)+".sock")
}

// answers pings a few times: one lost ping on a loaded box must not make a stranger's d2d look absent (the managed
// launch would then remove its socket file).
func answers(ctx context.Context, p *d2denv.Proc) bool {
	for i := 0; i < 3; i++ {
		if p.Alive(ctx) {
			return true
		}
		time.Sleep(300 * time.Millisecond)
	}
	return false
}

// ownerFile marks a socket whose d2d a managed d2bot started: only such a d2d is quit by --clean-start. The d2d control
// socket takes one client and has no authentication, so a bot must never drive or kill a d2d it cannot show it owns
// (a bot once connected to another agent's d2d on the shared fork socket).
func ownerFile(sock string) string { return sock + ".owner" }

func markOwned(sock string, pid int) {
	_ = os.WriteFile(ownerFile(sock), []byte(fmt.Sprintf("d2bot %d d2d %d\n", os.Getpid(), pid)), 0o600)
}

func ownedSocket(sock string) bool {
	_, err := os.Stat(ownerFile(sock))
	return err == nil
}

// openTables is data.Open; tests replace it with a synthetic table set.
var openTables = data.Open

// d2dSeams presets single seams (tests); runD2D fills the nil ones with the real stack (d2dwire.go). Nil fields would run the
// "not yet" stubs.
var d2dSeams d2denv.Seams

func d2dConfig(o options, log *slog.Logger, tune *env.Tuning, db *data.DB, home string) d2denv.Config {
	name := o.char
	if name == "" {
		name = "D2dbot"
	}
	cfg := d2denv.Config{
		ID: env.InstanceID(o.instance), Log: log, Tune: tune, Data: db, Sock: d2dSock(o), Dial: o.d2d.dial, LinkWait: o.d2d.linkWait,
		Spec: d2dinput.Spec{Name: name, Class: o.d2d.class, Seed: o.d2d.seed, NoSave: o.d2d.noSave},
	}
	if o.managed {
		cfg.Bin, cfg.DataDir, cfg.Nice, cfg.Home = o.d2d.bin, o.d2d.data, o.d2d.nice, home
		cfg.LogFile = filepath.Join(home, "d2d.out")
		if o.d2d.toggle != "" {
			cfg.ExtraArgs = append(cfg.ExtraArgs, "--toggle", o.d2d.toggle)
		}
		cfg.ExtraArgs = append(cfg.ExtraArgs, strings.Fields(o.d2d.extraArgs)...)
	}
	return cfg
}

// memQuestSource is the in-game quest words source of this env's reader (task.go): questmem for the memory reader,
// the "dump quests" source for d2d, none otherwise. A nil result means the save file is all there is.
func memQuestSource(e *env.Env) memQuests {
	switch r := e.Read.(type) {
	case *d2.Reader:
		if r != nil {
			return questmemFor(r)
		}
	case *d2denv.Reader:
		qs := &d2denv.QuestSource{Src: r.Source}
		if e.Clock != nil {
			qs.Now = e.Clock.Now // virtual under lockstep: a wall 200 ms cache outlives many stepped seconds
		}
		return qs
	}
	return nil
}

// runD2D is runLive for the d2d backend: same runner, goal manager, API and shutdown order, no Game.exe, no
// recycler (a d2d restart is Cycler.NewGame, not memory heap hygiene).
func runD2D(ctx context.Context, o options, log *slog.Logger) error {
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
	if len(cfg.DisableObserve) == 0 {
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
	var db *data.DB
	if o.gameDir != "" {
		if db, err = openTables(o.gameDir, filepath.Join(o.data, "cache", "tables")); err != nil {
			log.Warn("game tables unavailable; running without a catalog", "gamedir", o.gameDir, "err", err)
			db = nil
		}
	}
	if o.managed {
		if err := os.MkdirAll(filepath.Dir(d2dSock(o)), 0o700); err != nil {
			return err
		}
	}
	be, err := d2denv.Build(ctx, d2dConfig(o, log, &tune, db, filepath.Join(ip.Dir(), "d2d")))
	if err != nil {
		return err
	}
	defer be.Close()
	if o.managed && answers(ctx, be.Proc) {
		if !o.cleanStart {
			return fmt.Errorf("a d2d already answers on %s: use --attach --d2d-attach, or --clean-start to replace one this bot started", d2dSock(o))
		}
		if !ownedSocket(d2dSock(o)) {
			return fmt.Errorf("a d2d answers on %s but this bot did not start it (no %s): refusing to quit it", d2dSock(o), ownerFile(d2dSock(o)))
		}
		log.Warn("clean start: quitting the d2d left over from the previous d2bot")
		qctx, qc := context.WithTimeout(ctx, 10*time.Second)
		_ = be.Client.Quit(qctx)
		qc()
		for i := 0; i < 40 && be.Proc.Alive(ctx); i++ {
			time.Sleep(250 * time.Millisecond)
		}
	}
	e := be.Env
	if err := be.Start(ctx, 0); err != nil {
		return fmt.Errorf("d2d: %w", err)
	}
	if o.managed {
		markOwned(d2dSock(o), be.Proc.PID())
		defer os.Remove(ownerFile(d2dSock(o)))
	}
	log.Info("d2d backend up", "sock", d2dSock(o), "managed", o.managed, "lockstep", be.Clock.Lockstep())

	var cat bot.Catalog
	if db != nil {
		cat = bot.NewDataCatalog(db)
	}
	disp := newDisplay(cat)
	bus := event.NewBus(instanceID(o), nil)
	nv := be.Navigator()
	ref := &runnerRef{}
	seams := d2dSeams // tests may preset single seams; the real stack fills the rest
	deps := bot.Deps{Mover: moverAdapter{nv}, Cycler: be.Cycler, Catalog: cat, Health: d2dHealth(be.Proc, be.Beat, be.Clock.Lockstep, be.Clock.LastStep, log),
		OnSnapshot: disp.OnSnapshot, Bus: bus, Restart: func(ctx context.Context) error { return be.Cycler.NewGame(ctx, "") }}
	// Rig and town copy the Env by value, before bot.New installs Env.Emit: they would emit into nothing (kills, potions,
	// drops never reached the goal manager, summary 0 kills). Give them a forwarder bound to the runner later.
	emit := &lateEmit{}
	e.Emit = emit.send
	if db != nil { // combat and town need the game tables; without them the "not yet" stubs stay
		st := buildD2DStack(ctx, o, be, nv, db, cfg, ref, log)
		if seams.Fighter == nil {
			seams.Fighter = st.seams().Fighter
		}
		if seams.Town == nil {
			seams.Town = st.seams().Town
		}
		if seams.Loot == nil { // D2DLOOP: the seam existed but was never copied here, so no pickup ever ran (gold stayed 0)
			seams.Loot = st.seams().Loot
		}
		deps.Leveler, deps.Gear, deps.Quester, deps.Task, deps.Prepare = st.leveler, st.gear, st.quester, st.task, st.prepare
	} else {
		log.Warn("no game tables (--gamedir): combat, town and campaign goals stay off on d2d")
	}
	installMercSeam(&seams)
	fighter, town, loot := be.Behaviors(seams, nv)
	deps.Fighter, deps.Town = fighter, town
	if loot != nil {
		deps.Loot = loot
	}
	cy := be.Cycler
	e.Emit = nil // bot.New installs the runner's own and lateEmit forwards to it
	runner, err := bot.New(e, deps, cfg)
	if err == nil {
		ref.p.Store(runner)
		emit.bind(e.Emit)
		e.Emit = emit.send
	}
	if err != nil {
		return err
	}
	disp.run = runner
	m, err := goals.New(goals.Options{Runner: runner, Path: ip.Goals(), Chain: taskChain(runner.LastSnapshot),
		Logf: func(f string, a ...any) { log.Info("goals", "msg", sprintf(f, a...)) }})
	if err != nil {
		return err
	}
	if err := enqueueGoals(m, o.goalSpec); err != nil {
		return err
	}
	if o.goalSpec != "" {
		log.Info("goals enqueued")
	}
	if o.goalSpec != "" || o.autostart || cfg.Autostart {
		m.Start()
	}
	if o.runFor > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.runFor)
		defer cancel()
	}

	if be.Clock.Lockstep() {
		go logPace(ctx, be.Clock, log)
	}
	if h := soakHours(); h > 0 && be.Clock.Lockstep() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()
		go stopAtVirtual(ctx, cancel, e.Clock, h, log)
	}

	var ready atomic.Bool
	ready.Store(true)
	inst := &server.Instance{
		ID: instanceID(o), Manager: m, Bus: bus, Hub: disp.hub,
		Health: func() (any, bool) {
			rep := runner.Health()
			return rep, be.Proc.Alive(context.Background()) && rep.Fatal == "" && rep.Status != string(goals.RunnerError)
		},
		Ready:  ready.Load,
		Deaths: func() any { return runner.DeathReport() },
		State: func() any {
			return map[string]any{"state": runner.State(), "status": runner.Status(), "backend": "d2d",
				"panics": runner.Health().Panics, "goPanics": goPanics.Load(), "lockstep": be.Clock.Lockstep()}
		},
		Mounts: map[string]http.Handler{"knobs": api.KnobsHandler(tuner), "upgrades": api.UpgradesHandler(tuner), "merc": mercMount()},
	}
	reg := newRegistry()
	reg.RegisterProcs(map[string]func() int{"game": be.Proc.PID, "bot": os.Getpid})

	runCtx, stopRunner := context.WithCancel(context.Background())
	defer stopRunner()
	runnerDone := make(chan error, 1)
	if lim := watchdogLimit(); lim > 0 {
		go goSafe(runCtx, log, "watchdog", func() { newWatchdog(log, runner.LastStep, lim).run(runCtx) })
	}
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
	go func() {
		if !goSafe(mgrCtx, log, "manager", func() { m.Run(mgrCtx) }) {
			gaveUp <- "manager"
		}
	}()
	go goSafe(mgrCtx, log, "log-events", func() { logEvents(mgrCtx, log, m) })
	go goSafe(mgrCtx, log, "bus-bridge", func() { busBridge(mgrCtx, m, bus) })
	go goSafe(mgrCtx, log, "display", func() { disp.feed(mgrCtx, m, bus) })

	apiCtx, stopAPI := context.WithCancel(context.Background())
	apiDone := make(chan error, 1)
	go func() {
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
	}
	if o.managed { // we own this d2d: Save and Exit (the process ends with it)
		sd.leave = func() {
			lctx, lcancel := context.WithTimeout(context.Background(), bot.LeaveTimeout)
			defer lcancel()
			log.Info("stopping d2d (save and quit)")
			if err := cy.LeaveGame(lctx); err != nil {
				log.Warn("save and quit failed", "err", err)
				_ = be.Proc.Stop()
			}
		}
	}
	sd.run()
	return result
}

// questmemFor is the memory backend's quest source (unchanged behaviour of newTaskDeps).
func questmemFor(r *d2.Reader) memQuests { return questmem.New(r) }

// soakHours is the virtual run length cmd/d2dsoak asks for in -clock bot mode ($D2DSOAK_HOURS), 0 when unset.
func soakHours() time.Duration {
	f, err := strconv.ParseFloat(os.Getenv("D2DSOAK_HOURS"), 64)
	if err != nil || f <= 0 {
		return 0
	}
	return time.Duration(f * float64(time.Hour))
}

// stopAtVirtual cancels the run once the env clock (virtual under lockstep) has advanced by d. --run-for stays wall
// clock; this is the soak runner's hour count, which only means something in game time.
func stopAtVirtual(ctx context.Context, cancel context.CancelFunc, clk env.Clock, d time.Duration, log *slog.Logger) {
	t0 := clk.Now()
	tk := time.NewTicker(time.Second)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			if clk.Now().Sub(t0) >= d {
				log.Info("virtual run length reached", "virtual", d)
				cancel()
				return
			}
		}
	}
}

// logPace writes the virtual-to-wall ratio once a wall minute: the loaded box runs lockstep far slower than real time
// and every virtual-time knob (minute line, 240 s no-progress, soak hours) is in the slow unit.
func logPace(ctx context.Context, c *d2denv.LateClock, log *slog.Logger) {
	w0 := time.Now()
	tk := time.NewTicker(time.Minute)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			steps, ticks, v, ok := c.Pace()
			if !ok {
				return
			}
			wall := time.Since(w0)
			log.Info("pace", "wall", wall.Round(time.Second), "virtual", v.Round(time.Second), "ratio", float64(v)/float64(wall), "steps", steps, "ticks", ticks)
		}
	}
}

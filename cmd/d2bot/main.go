// Command d2bot is the single d2lod-bot binary: it wires the game (attached or launched), the input helper, the
// game tables, the goal manager, the HTTP API with the dashboard and the bot runner, or the simulator (--sim) when
// there is no game. Logs are structured (slog) on stdout; SIGINT/SIGTERM stop gracefully. See docs/bot.md.
package main

import (
	"context"
	"d2lod-bot/internal/gear"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"d2lod-bot/internal/buildinfo"
	"d2lod-bot/internal/logx"
	"d2lod-bot/internal/paths"
)

// version is stamped by CI with -ldflags "-X main.version=<tag>".
var version = "dev" // the stamp; buildinfo.Current() adds the VCS fallback

// options is everything configurable by flag or environment (container friendly: every flag has a D2BOT_* variable).
type options struct {
	data, config, addr, token, helper, gameDir, instance string
	layout                                               paths.Layout // resolved by parseFlags from --data/--config and the environment
	logLevel                                             string
	sim, attach, managed, dry, once                      bool
	dryFor                                               time.Duration
	simSpeed                                             float64
	simSeed                                              int64
	allowHost                                            string
	stopGame, stubs, autostart                           bool
	char                                                 string
	goalSpec                                             string
	runFor                                               time.Duration
	showVersion                                          bool
	probeStates                                          bool
	soakDir                                              string
	supervise, cleanStart                                bool
	superviseMax, testCrashAfter                         int
	d2d                                                  d2dOptions
}

func envOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func envBool(name string) bool {
	switch os.Getenv(name) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func parseFlags(args []string) (options, error) {
	var o options
	fs := flag.NewFlagSet("d2bot", flag.ContinueOnError)
	fs.StringVar(&o.data, "data", envOr("D2BOT_DATA", ""), "data dir: Wine prefix, extracted tables, per-instance state (env D2BOT_DATA; default ~/.local/share/d2bot)")
	fs.StringVar(&o.config, "config", envOr("D2BOT_CONFIG", ""), "config JSON file (env D2BOT_CONFIG; default <data>/config.json, optional)")
	fs.StringVar(&o.addr, "addr", envOr("D2BOT_ADDR", "127.0.0.1:8420"), "listen address (env D2BOT_ADDR); non-loopback requires a token")
	fs.StringVar(&o.token, "token", envOr("D2BOT_TOKEN", ""), "API token (env D2BOT_TOKEN)")
	fs.StringVar(&o.helper, "helper", envOr("D2BOT_HELPER", ""), "d2input helper address (env D2BOT_HELPER; default: derived from the instance, 127.0.0.1:47012+N)")
	fs.StringVar(&o.gameDir, "gamedir", envOr("D2BOT_GAMEDIR", ""), "Diablo II install dir for the game tables (env D2BOT_GAMEDIR; default <data>/proton/pfx/.../Diablo II)")
	fs.StringVar(&o.instance, "instance", envOr("D2BOT_INSTANCE", "main"), "instance id used in logs, state paths and API routes (env D2BOT_INSTANCE)")
	fs.StringVar(&o.logLevel, "log-level", envOr("D2BOT_LOG_LEVEL", "info"), "debug, info, warn or error (env D2BOT_LOG_LEVEL)")
	fs.StringVar(&o.allowHost, "allow-host", envOr("D2BOT_ALLOW_HOST", ""), "extra Host name accepted on a loopback listener (proxy, ssh tunnel)")
	fs.BoolVar(&o.sim, "sim", envBool("D2BOT_SIM"), "run the simulated runner: no game, no input (env D2BOT_SIM)")
	fs.BoolVar(&o.attach, "attach", envBool("D2BOT_ATTACH"), "dev mode: attach to an already running game (mem.FindGame) (env D2BOT_ATTACH)")
	fs.BoolVar(&o.managed, "managed", envBool("D2BOT_MANAGED"), "managed mode: this process launches the game and the helper through internal/launch (env D2BOT_MANAGED)")
	fs.BoolVar(&o.dry, "dry", envBool("D2BOT_DRY"), "read-only: attach and print what the bot sees; send no input, run no goals (env D2BOT_DRY)")
	fs.BoolVar(&o.once, "once", false, "with --dry: print one observation and exit")
	fs.DurationVar(&o.dryFor, "dry-for", 0, "with --dry: stop after this long (0 = until interrupted)")
	fs.StringVar(&o.char, "char", envOr("D2BOT_CHAR", ""), "character name for re-entering games through the menus (env D2BOT_CHAR; default: the name read from the game)")
	fs.StringVar(&o.goalSpec, "goals", envOr("D2BOT_GOALS", ""), "enqueue goals at start and start playing: 'farm:Blood Moor;reach:3;idle' (env D2BOT_GOALS)")
	fs.BoolVar(&o.autostart, "autostart", envBool("D2BOT_AUTOSTART"), "start playing the saved goals at startup, without a /control/start (env D2BOT_AUTOSTART; config key autostart)")
	fs.DurationVar(&o.runFor, "run-for", 0, "stop gracefully after this long (0 = until interrupted)")
	fs.BoolVar(&o.stubs, "stub-behaviors", false, "use failing combat/town stubs and a fake input controller: nothing is sent to the game")
	fs.BoolVar(&o.stopGame, "stop-game", false, "managed mode: also stop the game on shutdown")
	fs.Float64Var(&o.simSpeed, "sim-speed", 4, "with --sim: virtual seconds per real second")
	fs.Int64Var(&o.simSeed, "sim-seed", 1, "with --sim: random seed")
	fs.BoolVar(&o.probeStates, "probe-states", envBool("D2BOT_PROBE_STATES"), "debug: log changes in the player StatList struct (locating the state bit array, docs/boss.md)")
	fs.StringVar(&o.soakDir, "soak-dir", envOr("D2BOT_SOAK_DIR", ""), "directory of soak runs (~/d2lod-live/soak): the dashboard shows the newest run's alerts and status, read-only (env D2BOT_SOAK_DIR; default off)")
	fs.BoolVar(&o.supervise, "supervise", envBool("D2BOT_SUPERVISE"), "run d2bot as a child and restart it (backoff, capped) when it exits non-zero or crashes; a restart cleans up the old game first (env D2BOT_SUPERVISE)")
	fs.IntVar(&o.superviseMax, "supervise-max", 10, "with --supervise: give up after this many restarts within one hour")
	fs.IntVar(&o.testCrashAfter, "test-crash-after", -1, "TEST ONLY (crash drill): panic unrecovered after N seconds; same as D2BOT_TEST=1 D2BOT_TEST_CRASH_AFTER=N (tools/live.sh cannot pass env)")
	fs.BoolVar(&o.cleanStart, "clean-start", false, "managed mode: stop a game left over from a crashed d2bot before launching a fresh one (set by --supervise on restarts)")
	d2dFlags(fs, &o.d2d)
	fs.BoolVar(&o.showVersion, "version", false, "print the version and exit")
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	if o.showVersion {
		return o, nil
	}
	n := 0
	for _, b := range []bool{o.sim, o.attach, o.managed} {
		if b {
			n++
		}
	}
	if n != 1 {
		return o, errors.New("choose exactly one of --sim, --attach, --managed (see --help)")
	}
	if err := o.d2d.validate(o); err != nil {
		return o, err
	}
	if o.dry && o.sim {
		return o, errors.New("--dry needs a game: use --attach or --managed")
	}
	// paths.FromEnv owns the defaults (D2BOT_DATA, XDG_DATA_HOME, ~/.local/share/d2bot; D2BOT_CONFIG; XDG_RUNTIME_DIR).
	l, err := paths.FromEnv()
	if err != nil && o.data == "" {
		return o, err
	}
	if o.data != "" {
		l.Root = o.data
	}
	if o.config != "" {
		l.ConfigFile = o.config
	}
	o.data, o.config, o.layout = l.Root, l.Config(), l
	if _, err := l.Instance(o.instance); err != nil {
		return o, err
	}
	return o, nil
}

// logFlush writes the pending "last line repeated" line; set by newLogger.
var logFlush = func() {}

func newLogger(level, instance string) (*slog.Logger, error) {
	var l slog.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("log level %q: %w", level, err)
	}
	// Tight retry loops log the same line hundreds of times a second: fold identical runs (internal/logx).
	h := logx.NewDedup(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l}), 0)
	logFlush = h.Flush
	return slog.New(h).With("inst", instance), nil
}

func init() {
	if version == "dev" {
		version = buildinfo.Current()
	}
}

func main() {
	o, err := parseFlags(os.Args[1:])
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "d2bot:", err)
		os.Exit(2)
	}
	if o.showVersion {
		fmt.Println("d2bot", version)
		return
	}
	log, err := newLogger(o.logLevel, o.instance)
	if err != nil {
		fmt.Fprintln(os.Stderr, "d2bot:", err)
		os.Exit(2)
	}
	log.Info("d2bot start", "version", version)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if os.Getenv(superviseMarker) != "" {
		o.supervise = false // a supervised child never supervises (D2BOT_SUPERVISE leaking in would fork-bomb)
	}
	if o.supervise && underFleet() {
		log.Warn("--supervise ignored: d2fleet already restarts d2bot (nesting supervisors would double the restarts)")
		o.supervise = false
	}
	if o.supervise {
		code := superviseProcess(ctx, o, log, os.Args[1:])
		stop()
		os.Exit(code)
	}
	// The gear tables come from the user's own MPQs (cmd/d2tables); fail here, not mid-run inside a recovered panic.
	if os.Getenv("D2_DATA_DIR") == "" && o.d2d.data != "" {
		os.Setenv("D2_DATA_DIR", o.d2d.data)
	}
	if err := gear.Available(); err != nil {
		fmt.Fprintln(os.Stderr, "d2bot:", err)
		os.Exit(2)
	}
	armTestCrash(log, o.testCrashAfter)
	err = run(ctx, o, log)
	if err != nil {
		log.Error("exit", "err", err)
	}
	logFlush()
	if err != nil {
		var ee *exitError
		if errors.As(err, &ee) {
			os.Exit(ee.code)
		}
		os.Exit(1)
	}
}

func run(ctx context.Context, o options, log *slog.Logger) error {
	switch {
	case o.d2d.on():
		return runD2D(ctx, o, log)
	case o.sim:
		return runSim(ctx, o, log)
	case o.dry:
		return runDry(ctx, o, log)
	default:
		return runLive(ctx, o, log)
	}
}

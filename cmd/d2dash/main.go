// Command d2dash runs the goal manager with the simulated runner and serves the dashboard, so the UI and
// API can be tried without the game. The real binary (cmd/d2bot) wires the same manager and api to the bot.
package main

import (
	"context"
	"d2lod-bot/internal/buildinfo"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/goals/sim"
	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/server"
	"d2lod-bot/internal/setup"
	"d2lod-bot/web"
)

func main() {
	buildinfo.Exit("d2dash")
	setup.MaybeProbe()
	addr := flag.String("addr", "127.0.0.1:8420", "listen address (non-loopback requires -token)")
	token := flag.String("token", "", "require this token (Bearer, cookie, or ?token= once)")
	state := flag.String("state", "", "state file (default: in memory, nothing is written)")
	speed := flag.Float64("speed", 4, "simulation speed: virtual seconds per real second")
	seed := flag.Int64("seed", 1, "simulation seed")
	warm := flag.Duration("warm", 90*time.Minute, "pre-simulate this much history so charts are not empty")
	empty := flag.Bool("empty", false, "start with an empty queue and a stopped bot")
	n := flag.Int("instances", 2, "number of simulated instances (ids i1..iN; /api/v1/instances/{id}/...)")
	allow := flag.String("allow-host", "", "extra Host name accepted on a loopback listener")
	setupData := flag.String("setup-data", "", "serve the first-run wizard at /setup.html with its state under this data dir (the simulated bots are not gated)")
	soak := flag.String("soak-dir", os.Getenv("D2BOT_SOAK_DIR"), "directory of soak runs (~/d2lod-live/soak): show the newest run's alerts and status in the dashboard, read-only (default off)")
	flag.Parse()

	if *n < 1 || *n > 16 {
		log.Fatal("-instances must be 1..16")
	}
	reg := &server.StaticRegistry{}
	var sims []*sim.Runner
	var mgrs []*goals.Manager
	for k := 1; k <= *n; k++ {
		path := *state
		if path != "" && *n > 1 {
			path = fmt.Sprintf("%s.i%d", path, k)
		}
		runner, m := newInstance(*seed+int64(k-1), path, *warm, *empty)
		sims, mgrs = append(sims, runner), append(mgrs, m)
		if err := reg.Add(&server.Instance{ID: fmt.Sprintf("i%d", k), Title: fmt.Sprintf("Sim %d", k), Manager: m}); err != nil {
			log.Fatal(err)
		}
	}

	cfg := server.Config{Instances: reg, Addr: *addr, Token: *token, Assets: web.FS,
		Version: "d2dash (simulated runner)", AllowedHosts: splitNonEmpty(*allow), SoakDir: *soak}
	if *setupData != "" {
		if err := launch.BecomeSubreaper(); err != nil {
			log.Fatal(err)
		}
		go (&launch.ZombieReaper{}).Run(context.Background(), 0)
		wz, err := setup.NewWeb(setup.WebConfig{DataDir: *setupData, Display: os.Getenv("DISPLAY")})
		if err != nil {
			log.Fatal(err)
		}
		cfg.Setup = wz.Handler()
	}
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	ln, err := srv.Listen()
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	for k := range sims {
		go sims[k].Run(ctx, *speed)
		go mgrs[k].Run(ctx)
	}

	u := "http://" + ln.Addr().String() + "/"
	if *token != "" {
		u += "?token=" + *token
	}
	fmt.Printf("d2dash: dashboard at %s  (%d simulated instances i1..i%d, speed x%g; Ctrl-C to stop)\n", u, *n, *n, *speed)
	if err := srv.Serve(ctx, ln); err != nil {
		log.Fatal(err)
	}
}

func splitNonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	return []string{s}
}

// newInstance builds a simulated runner and its manager, pre-simulating warm of virtual history on a private clock
// that switches to wall time afterwards.
func newInstance(seed int64, state string, warm time.Duration, empty bool) (*sim.Runner, *goals.Manager) {
	runner := sim.New(sim.Options{Seed: seed})
	var live atomic.Bool
	var virt time.Time
	clock := func() time.Time {
		if live.Load() {
			return time.Now()
		}
		return virt
	}
	virt = time.Now().Add(-warm)

	m, err := goals.New(goals.Options{Runner: runner, Path: state, Clock: clock, Logf: log.Printf})
	if err != nil {
		log.Fatal(err)
	}
	if !empty && len(m.Goals()) == 0 {
		for _, g := range []goals.Goal{
			{Type: goals.ReachLevel, Params: goals.Params{Level: 10}},
			{Type: goals.FarmArea, Params: goals.Params{Area: "Cold Plains", UntilKills: 150}},
			{Type: goals.KillBoss, Params: goals.Params{Boss: "Mephisto", Count: 2}},
			{Type: goals.CollectItem, Params: goals.Params{Count: 2, Item: goals.ItemFilter{Type: "rune"}}},
			{Type: goals.FarmArea, Params: goals.Params{Area: "Pit Level 1"}},
		} {
			if _, err := m.Add(g, -1); err != nil {
				log.Fatal(err)
			}
		}
		m.Start()
		for end := virt.Add(warm); virt.Before(end); {
			virt = virt.Add(time.Second)
			runner.Advance(time.Second)
			for len(runner.Events()) > 0 {
				m.Handle(<-runner.Events())
			}
			m.Tick(virt)
		}
		for _, g := range []goals.Goal{
			{Type: goals.ReachLevel, Params: goals.Params{Level: 30}, Priority: 1},
			{Type: goals.Idle, Priority: -1, Schedule: []goals.Window{{Start: "03:00", End: "05:00"}}},
		} {
			m.Add(g, -1)
		}
	}
	live.Store(true)
	return runner, m
}

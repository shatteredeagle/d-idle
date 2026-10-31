package main

import (
	"context"
	"d2lod-bot/internal/fleet"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// slotInfo is what the sidecar needs of one fleet slot (fleet.View).
type slotInfo struct {
	ID, Prefix string
	HelperPort int
	Display    string
	BotAddr    string // d2bot's own API (127.0.0.1:port), "" while it is not up
	BotLog     string
}

// slotProcs are the --proc specs of one slot, matched by the slot's own identity so several slots never mix:
// the bot by "--instance iN", the Wine processes by their WINEPREFIX (<prefix>/pfx, the fleet's convention,
// internal/launch: WINEPREFIX=<PrefixDir>/pfx), the helper by its port and the Xvfb by its display.
func slotProcs(s slotInfo) []ProcSource {
	pfx := filepath.Join(s.Prefix, "pfx")
	ps := []ProcSource{
		{"bot", "find:comm=d2bot;cmd=--instance " + s.ID},
		{"game", "find:comm=Game.exe;env=WINEPREFIX=" + pfx},
		{"wineserver", "find:comm=wineserver;env=WINEPREFIX=" + pfx},
	}
	if s.HelperPort > 0 {
		ps = append(ps, ProcSource{"helper", fmt.Sprintf("find:comm=d2input.exe;cmd=-addr 127.0.0.1:%d", s.HelperPort)})
	}
	if s.Display != "" {
		ps = append(ps, ProcSource{"xvfb", "find:comm=Xvfb;cmd=Xvfb " + s.Display + " "})
	}
	return ps
}

func slotsFromView(vs []fleet.View) []slotInfo {
	var out []slotInfo
	for _, v := range vs {
		s := slotInfo{ID: v.ID, Prefix: v.Alloc.Prefix, HelperPort: v.Alloc.HelperPort, Display: v.Alloc.Display}
		if v.Bot != nil {
			s.BotAddr, s.BotLog = v.Bot.Addr, v.Bot.Log
		}
		out = append(out, s)
	}
	return out
}

// fleetSlots lists the slots through the fleet admin API.
func fleetSlots(ctx context.Context, cl *http.Client, base, token string) ([]slotInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/api/v1/fleet-admin/slots", nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := cl.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("fleet slots: HTTP %d", resp.StatusCode)
	}
	var vs []fleet.View
	if err := json.NewDecoder(resp.Body).Decode(&vs); err != nil {
		return nil, err
	}
	return slotsFromView(vs), nil
}

type sidecarCfg struct {
	Slots  func(ctx context.Context) ([]slotInfo, error)
	Out    string
	Poll   time.Duration
	Every  time.Duration
	For    time.Duration
	Limits Limits
	Rules  AlertRules
	// AlertCmd is run on every new fail alert (D2SOAK_ALERT_CMD, alertcmd.go); "" = none.
	AlertCmd string
	Now      func() time.Time
	Log      func(string, ...any)
	// Alive, when set, ends the sidecar once it returns false (the container's main process is gone).
	Alive func() bool
}

// runSidecar starts one soak run per slot as soon as the slot's bot answers and follows the bot over restarts
// (its address changes each time) until ctx ends or every run has finished. A slot whose run finished is not
// started again by this sidecar; a new sidecar (container restart) starts new runs.
func runSidecar(ctx context.Context, cfg sidecarCfg) error {
	var mu sync.Mutex
	addr := map[string]string{}
	running := map[string]bool{}
	var wg sync.WaitGroup
	defer wg.Wait()
	ctx, cancel := context.WithCancel(ctx) // runs before wg.Wait: Alive ending the sidecar also ends the runs
	defer cancel()
	tick := time.NewTicker(cfg.Poll)
	defer tick.Stop()
	for {
		if cfg.Alive != nil && !cfg.Alive() {
			return nil
		}
		slots, err := cfg.Slots(ctx)
		if err != nil {
			cfg.Log("d2soak sidecar: slots: %v", err)
		}
		for _, s := range slots {
			mu.Lock()
			if s.BotAddr != "" {
				addr[s.ID] = s.BotAddr
			}
			start := !running[s.ID] && s.BotAddr != ""
			if start {
				running[s.ID] = true
			}
			mu.Unlock()
			if !start {
				continue
			}
			wg.Add(1)
			go func(s slotInfo) {
				defer wg.Done()
				runSlot(ctx, cfg, s, func() string { mu.Lock(); defer mu.Unlock(); return addr[s.ID] })
			}(s)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
		}
	}
}

// runSlot is one slot's soak run. The slot's processes and log are fixed for the run; only the bot address moves.
func runSlot(ctx context.Context, cfg sidecarCfg, s slotInfo, current func() string) {
	run := filepath.Join(cfg.Out, s.ID+"-"+cfg.Now().Format("20060102-150405"))
	root := func(a string) string { return "http://" + a }
	cl := &http.Client{Timeout: 10 * time.Second}
	c, err := Resolve(ctx, cl, "", root(current()), "")
	if err != nil {
		c = &Collector{HTTP: cl, Base: root(current()) + "/api/v1", Flat: true} // bot gone again: the first samples show it
	}
	c.Procs = slotProcs(s)
	c.MetricsURL = root(current()) + "/metrics"
	if s.Prefix != "" {
		c.WinePrefixes = []string{filepath.Join(s.Prefix, "pfx")}
	}
	last := current()
	c.Retarget = func(ctx context.Context, c *Collector) {
		a := current()
		if a == last {
			return
		}
		if n, err := Resolve(ctx, cl, "", root(a), ""); err == nil {
			c.Base, c.Flat, c.MetricsURL = n.Base, n.Flat, root(a)+"/metrics"
			last = a
			cfg.Log("d2soak sidecar: %s bot moved to %s", s.ID, a)
		}
	}
	rules := cfg.Rules
	rules.BotLog = s.BotLog
	cfg.Log("d2soak sidecar: %s run %s (bot %s)", s.ID, run, last)
	// The effective limits (limits.json of the run, written by runSoak) never depend on a limits block being present.
	lim := withProfileDefaults(cfg.Limits)
	r, err := runSoak(ctx, runCfg{Collector: c, Every: cfg.Every, For: cfg.For, Out: run, Limits: lim, Rules: rules,
		AlertCmd: cfg.AlertCmd, Now: cfg.Now, Log: cfg.Log})
	if err != nil {
		cfg.Log("d2soak sidecar: %s: %v", s.ID, err)
		return
	}
	cfg.Log("d2soak sidecar: %s finished: pass=%v", s.ID, r.Pass)
}

func cmdSidecar(args []string) int {
	fs := flag.NewFlagSet("sidecar", flag.ContinueOnError)
	fleetURL := fs.String("fleet", "http://127.0.0.1:8431", "fleet server root (the slots come from its admin API)")
	tokenFile := fs.String("token-file", "", "file with the fleet bearer token (default $D2BOT_DATA/token when it exists)")
	out := fs.String("out", "/data/soak", "directory of runs; one <slot>-<time> directory per slot")
	dur := fs.Duration("for", 24*time.Hour, "length of each slot's run")
	every := fs.Duration("every", 30*time.Second, "sample interval")
	poll := fs.Duration("poll", 10*time.Second, "how often the slot list is read")
	parent := fs.Int("parent", 0, "exit when this pid is gone (the container's main process)")
	lim, finish := limitFlags(fs)
	rules := alertFlags(fs)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if !flagSet(fs, "profile") {
		_ = fs.Set("profile", "m24")
	}
	if err := finish(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	tf := *tokenFile
	if tf == "" {
		tf = filepath.Join(envOr("D2BOT_DATA", "/data"), "token")
	}
	token := ""
	if b, err := os.ReadFile(tf); err == nil {
		token = strings.TrimSpace(string(b))
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cl := &http.Client{Timeout: 10 * time.Second}
	cfg := sidecarCfg{
		Slots: func(ctx context.Context) ([]slotInfo, error) { return fleetSlots(ctx, cl, *fleetURL, token) },
		Out:   *out, Poll: *poll, Every: *every, For: *dur, Limits: *lim, Rules: *rules, AlertCmd: os.Getenv(alertCmdEnv), Now: time.Now,
		Log: func(f string, a ...any) { fmt.Fprintf(os.Stderr, f+"\n", a...) },
	}
	if *parent > 0 {
		cfg.Alive = func() bool { return syscall.Kill(*parent, 0) == nil }
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "d2soak:", err)
		return 1
	}
	if err := runSidecar(ctx, cfg); err != nil {
		fmt.Fprintln(os.Stderr, "d2soak:", err)
		return 1
	}
	return 0
}

func flagSet(fs *flag.FlagSet, name string) bool {
	set := false
	fs.Visit(func(f *flag.Flag) { set = set || f.Name == name })
	return set
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

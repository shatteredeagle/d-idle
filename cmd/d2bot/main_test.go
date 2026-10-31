package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/goals/sim"
)

func TestParseFlags(t *testing.T) {
	for _, bad := range [][]string{{}, {"--sim", "--attach"}, {"--dry", "--sim"}} {
		if _, err := parseFlags(bad); err == nil {
			t.Errorf("%v accepted", bad)
		}
	}
	t.Setenv("D2BOT_TOKEN", "secret")
	t.Setenv("D2BOT_DATA", "/tmp/x")
	o, err := parseFlags([]string{"--attach", "--dry"})
	if err != nil || o.token != "secret" || o.data != "/tmp/x" || o.config != "/tmp/x/config.json" || !o.dry {
		t.Fatalf("%+v %v", o, err)
	}
}

func get(t *testing.T, url string) (int, string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestSimServesDashboardHealthAndStopsGracefully(t *testing.T) {
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := l.Addr().String()
	l.Close()
	o := options{sim: true, data: t.TempDir(), addr: addr, instance: "t", simSpeed: 50, simSeed: 1}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runSim(ctx, o, slog.New(slog.NewTextHandler(io.Discard, nil))) }()
	var code int
	var body string
	for i := 0; i < 100; i++ {
		func() {
			defer func() { recover() }()
			resp, err := http.Get("http://" + addr + "/healthz")
			if err == nil {
				b, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				code, body = resp.StatusCode, string(b)
			}
		}()
		if code != 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if code != 200 || !strings.Contains(body, `"ok":true`) {
		t.Fatalf("healthz %d %s", code, body)
	}
	base := "http://" + addr
	if c, b := get(t, base+"/readyz"); c != 200 || !strings.Contains(b, `"t":true`) {
		t.Fatalf("readyz %d %s", c, b)
	}
	if c, b := get(t, base+"/api/v1/instances"); c != 200 || !strings.Contains(b, `"id":"t"`) {
		t.Fatalf("instances %d %s", c, b)
	}
	if c, b := get(t, base+"/api/v1/instances/t/status"); c != 200 || !strings.Contains(b, "runner") {
		t.Fatalf("instance status %d %s", c, b)
	}
	if c, b := get(t, base+"/api/v1/bot/health"); c != 200 || !strings.Contains(b, `"ok":true`) {
		t.Fatalf("compat bot/health %d %s", c, b)
	}
	if c, b := get(t, base+"/api/v1/instances/t/knobs"); c != 200 || !strings.Contains(b, `"knobs"`) || !strings.Contains(b, `"profile":"optimal"`) {
		t.Fatalf("knobs %d %.200s", c, b)
	}
	if c, b := get(t, base+"/metrics/"); c != 200 || !strings.Contains(b, "go_goroutines") {
		t.Fatalf("metrics %d %.200s", c, b)
	}
	if c, b := get(t, "http://"+addr+"/api/v1/status"); c != 200 || !strings.Contains(b, "runner") {
		t.Fatalf("status %d %s", c, b)
	}
	if c, b := get(t, "http://"+addr+"/"); c != 200 || !strings.Contains(b, "<") {
		t.Fatalf("dashboard %d", c)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no graceful shutdown")
	}
}

func TestVersionFlagNeedsNoMode(t *testing.T) {
	o, err := parseFlags([]string{"--version"})
	if err != nil || !o.showVersion {
		t.Fatalf("%+v %v", o, err)
	}
}

func TestParseFlagsUsesPathsLayout(t *testing.T) {
	t.Setenv("D2BOT_DATA", "")
	t.Setenv("D2BOT_CONFIG", "")
	t.Setenv("XDG_DATA_HOME", "/xdg")
	o, err := parseFlags([]string{"--sim"})
	if err != nil || o.data != "/xdg/d2bot" || o.config != "/xdg/d2bot/config.json" || o.instance != "main" {
		t.Fatalf("%+v %v", o, err)
	}
	t.Setenv("D2BOT_CONFIG", "/etc/d2bot.json")
	if o, err = parseFlags([]string{"--sim", "--data", "/d"}); err != nil || o.data != "/d" || o.config != "/etc/d2bot.json" {
		t.Fatalf("%+v %v", o, err)
	}
	if _, err := parseFlags([]string{"--sim", "--instance", "../x"}); err == nil {
		t.Fatal("bad instance id accepted")
	}
}

func TestLegacyStateMoves(t *testing.T) {
	dir := t.TempDir()
	o := options{data: dir, instance: "i2"}
	old := filepath.Join(dir, "state", "instances", "i2")
	os.MkdirAll(old, 0o755)
	os.WriteFile(filepath.Join(old, "goals.json"), []byte("{}"), 0o644)
	ip, err := instancePaths(o, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ip.Goals()); err != nil || ip.Goals() != filepath.Join(dir, "instances", "i2", "goals.json") {
		t.Fatalf("goals not moved to %s: %v", ip.Goals(), err)
	}
	if _, err := os.Stat(filepath.Join(old, "goals.json")); err == nil {
		t.Fatal("old file still there")
	}
}

func TestAutostartFlagAndCampaignGoal(t *testing.T) {
	o, err := parseFlags([]string{"--sim", "--autostart", "--goals", "campaign"})
	if err != nil || !o.autostart {
		t.Fatalf("%+v %v", o, err)
	}
	m, err := goals.New(goals.Options{Runner: sim.New(sim.Options{Seed: 1})})
	if err != nil {
		t.Fatal(err)
	}
	if err := enqueueGoals(m, "campaign;campaign:normal_act2"); err != nil {
		t.Fatal(err)
	}
	if n := len(m.Goals()); n != 2 {
		t.Fatalf("%d goals", n)
	}
}

func TestEnqueueRunsGoal(t *testing.T) {
	m, err := goals.New(goals.Options{Runner: sim.New(sim.Options{Seed: 1})})
	if err != nil {
		t.Fatal(err)
	}
	if err := enqueueGoals(m, "runs:Eldritch_Shenk:5;runs:auto:3"); err != nil {
		t.Fatal(err)
	}
	gs := m.Goals()
	if len(gs) != 2 || gs[0].Type != goals.Runs || gs[0].Params.Area != "Eldritch Shenk" || gs[0].Params.Count != 5 {
		t.Fatalf("%+v", gs)
	}
	if enqueueGoals(m, "runs:") == nil {
		t.Fatal("empty run name accepted")
	}
}

func TestSoakDirFlagAndEnv(t *testing.T) {
	t.Setenv("D2BOT_DATA", t.TempDir())
	o, err := parseFlags([]string{"--sim"})
	if err != nil || o.soakDir != "" {
		t.Fatalf("default must be off: %+v %v", o, err)
	}
	t.Setenv("D2BOT_SOAK_DIR", "/soak/env")
	if o, _ = parseFlags([]string{"--sim"}); o.soakDir != "/soak/env" {
		t.Fatalf("env: %q", o.soakDir)
	}
	if o, _ = parseFlags([]string{"--sim", "--soak-dir", "/soak/flag"}); o.soakDir != "/soak/flag" {
		t.Fatalf("flag: %q", o.soakDir)
	}
}

func TestSimServesSoakStrip(t *testing.T) {
	root := t.TempDir()
	run := filepath.Join(root, "run1")
	os.MkdirAll(run, 0o755)
	os.WriteFile(filepath.Join(run, "alerts.json"), []byte(`{"level":"ok","alerts":[]}`), 0o644)
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := l.Addr().String()
	l.Close()
	o := options{sim: true, data: t.TempDir(), addr: addr, instance: "t", simSpeed: 50, simSeed: 1, soakDir: root}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runSim(ctx, o, slog.New(slog.NewTextHandler(io.Discard, nil))) }()
	var code int
	var body string
	for i := 0; i < 100; i++ {
		if resp, err := http.Get("http://" + addr + "/api/v1/soak"); err == nil {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			code, body = resp.StatusCode, string(b)
		}
		if code == 200 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	cancel()
	<-done
	if code != 200 || !strings.Contains(body, `"run":"run1"`) {
		t.Fatalf("%d %s", code, body)
	}
}

func TestEnqueueGoalsIdempotentAcrossRestarts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "goals.json")
	open := func() *goals.Manager {
		m, err := goals.New(goals.Options{Runner: sim.New(sim.Options{Seed: 1}), Path: path})
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	for i := 0; i < 5; i++ {
		if err := enqueueGoals(open(), "campaign;reach:5"); err != nil {
			t.Fatal(err)
		}
	}
	m := open()
	if n := len(m.Goals()); n != 2 {
		t.Fatalf("%d goals after 5 restarts", n)
	}
	// changed string: only the new entry is added
	if err := enqueueGoals(m, "campaign;reach:5;boss:Andariel"); err != nil {
		t.Fatal(err)
	}
	if n := len(m.Goals()); n != 3 {
		t.Fatalf("%d goals after changing --goals", n)
	}
	// a finished one is not queued again
	for _, g := range m.Goals() {
		if g.Type == goals.ReachLevel {
			if err := m.Skip(g.ID); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := enqueueGoals(open(), "campaign;reach:5;boss:Andariel"); err != nil {
		t.Fatal(err)
	}
	if n := len(open().Goals()); n != 3 {
		t.Fatalf("%d goals, finished goal was re-queued", n)
	}
	// a dashboard add stays free to duplicate
	if _, err := m.Add(goals.Goal{Type: goals.Campaign, Params: goals.Params{Template: "normal_act1"}}, -1); err != nil {
		t.Fatal(err)
	}
	if n := len(m.Goals()); n != 4 {
		t.Fatalf("%d goals", n)
	}
}

func TestEnqueueCampaignIsIdempotent(t *testing.T) {
	m, err := goals.New(goals.Options{Runner: sim.New(sim.Options{Seed: 1})})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ { // three starts of the same command line
		if err := enqueueGoals(m, "campaign;campaign:normal_act1:nochain"); err != nil {
			t.Fatal(err)
		}
	}
	if n := len(m.Goals()); n != 2 {
		t.Fatalf("%d goals after repeated enqueue, want 2", n)
	}
}

func TestSuperviseEnvDoesNotMakeAChildASupervisor(t *testing.T) {
	t.Setenv("D2BOT_SUPERVISE", "1")
	o, err := parseFlags([]string{"--sim"})
	if err != nil || !o.supervise {
		t.Fatalf("precondition: %+v %v", o, err)
	}
	for _, kv := range childEnv([]string{"D2BOT_SUPERVISE=1"}, false) {
		if strings.HasPrefix(kv, "D2BOT_SUPERVISE=") {
			t.Fatal("D2BOT_SUPERVISE leaks into the child")
		}
	}
}

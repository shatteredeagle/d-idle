package main

import (
	"context"
	"flag"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/goals/sim"
	"d2lod-bot/internal/metrics"
	"d2lod-bot/internal/server"
)

// bot is a simulated bot behind the real unified server: manager + sim runner, virtual clock.
type bot struct {
	t     *testing.T
	m     *goals.Manager
	r     *sim.Runner
	virt  time.Time
	srv   *httptest.Server
	reg   *metrics.Registry
	met   *httptest.Server
	gor   *metrics.Gauge
	rssKB int64
	fds   int
	c     *Collector
}

func newBot(t *testing.T) *bot {
	t.Helper()
	b := &bot{t: t, virt: time.Date(2026, 10, 28, 8, 0, 0, 0, time.UTC), rssKB: 400_000, fds: 40}
	b.r = sim.New(sim.Options{Seed: 7, DeathRate: 0.2})
	var err error
	b.m, err = goals.New(goals.Options{Runner: b.r, Clock: func() time.Time { return b.virt }})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.m.Add(goals.Goal{Type: goals.FarmArea, Params: goals.Params{Area: "Cold Plains"}}, -1); err != nil {
		t.Fatal(err)
	}
	if err := b.m.Start(); err != nil {
		t.Fatal(err)
	}
	reg := &server.StaticRegistry{}
	if err := reg.Add(&server.Instance{ID: "i4", Manager: b.m}); err != nil {
		t.Fatal(err)
	}
	s, err := server.New(server.Config{Instances: reg})
	if err != nil {
		t.Fatal(err)
	}
	b.srv = httptest.NewServer(s.Handler())
	b.reg = metrics.NewRegistry()
	b.gor = b.reg.Gauge("go_goroutines", "g")
	b.gor.Set(30)
	b.met = httptest.NewServer(b.reg.Handler())
	t.Cleanup(func() { b.srv.Close(); b.met.Close(); s.Close() })

	b.c, err = Resolve(context.Background(), b.srv.Client(), "", b.srv.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	b.c.MetricsURL = b.met.URL + "/metrics"
	b.c.ProcOverride = func() map[string]ProcSample {
		return map[string]ProcSample{"game": {PID: 100, Start: 5, State: "S", RSSKB: b.rssKB, FDs: b.fds, Threads: 20}}
	}
	return b
}

// run advances the simulation (when play is true) and samples every 30 virtual seconds for d.
func (b *bot) run(d time.Duration, play bool, each func(i int)) []Sample {
	var out []Sample
	for i := 0; time.Duration(i)*30*time.Second < d; i++ {
		if play {
			for s := 0; s < 30; s++ {
				b.virt = b.virt.Add(time.Second)
				b.r.Advance(time.Second)
				for len(b.r.Events()) > 0 {
					b.m.Handle(<-b.r.Events())
				}
				b.m.Tick(b.virt)
			}
		} else {
			b.virt = b.virt.Add(30 * time.Second)
		}
		if each != nil {
			each(i)
		}
		out = append(out, b.c.Sample(context.Background(), b.virt))
	}
	return out
}

func lim() Limits {
	l := DefaultLimits()
	l.MaxDeathsPerHour = 100 // the sim dies on purpose; deaths have their own test
	return l
}

func code(r Report) string {
	var c []string
	for _, f := range r.Findings {
		if f.Level == "fail" {
			c = append(c, f.Code)
		}
	}
	return strings.Join(c, ",")
}

func TestSimSoakPasses(t *testing.T) {
	b := newBot(t)
	ss := b.run(3*time.Hour, true, nil)
	r := Analyze(ss, lim())
	if !r.Pass {
		t.Fatalf("expected pass:\n%s", r.Text())
	}
	if r.XPPerHour <= 0 || r.KillsPerHour <= 0 {
		t.Errorf("no progress measured: %+v", r)
	}
	if !ss[0].HealthOK || ss[0].State == "" {
		t.Errorf("health not read: %+v", ss[0])
	}
	if ss[len(ss)-1].Seq == 0 {
		t.Error("no events seen")
	}
	if ss[len(ss)-1].Gauges["go_goroutines"] != 30 {
		t.Errorf("metrics not scraped: %v", ss[len(ss)-1].Gauges)
	}
	t.Log("\n" + r.Text())
}

func TestSimDeathsCounted(t *testing.T) {
	ss := newBot(t).run(6*time.Hour, true, nil)
	r := Analyze(ss, lim())
	if r.Deaths == 0 {
		t.Skip("sim produced no deaths with this seed")
	}
	l := lim()
	l.MaxDeathsPerHour = r.DeathsPerHour / 2
	if r2 := Analyze(ss, l); !strings.Contains(code(r2), "deaths") {
		t.Fatalf("deaths limit not enforced:\n%s", r2.Text())
	}
}

func TestLeakDetected(t *testing.T) {
	b := newBot(t)
	ss := b.run(3*time.Hour, true, func(i int) {
		b.rssKB += 300 * 1024 / 120 // 300 MB per hour at 120 samples/h
		b.fds++                     // 120 fds/h
		b.gor.Add(1)                // 120 goroutines/h
	})
	r := Analyze(ss, lim())
	got := code(r)
	if r.Pass || strings.Count(got, "leak") < 3 {
		t.Fatalf("want three leak findings, got %q:\n%s", got, r.Text())
	}
}

func TestStallDetected(t *testing.T) {
	b := newBot(t)
	ss := b.run(30*time.Minute, true, nil)
	ss = append(ss, b.run(25*time.Minute, false, nil)...) // alive and answering, but nothing happens
	r := Analyze(ss, lim())
	if !strings.Contains(code(r), "no-progress") {
		t.Fatalf("stall not detected:\n%s", r.Text())
	}
	// Pausing is exempt.
	b2 := newBot(t)
	ss = b2.run(30*time.Minute, true, nil)
	b2.r.Pause()
	b2.m.Tick(b2.virt)
	ss = append(ss, b2.run(25*time.Minute, false, nil)...)
	for i := range ss[len(ss)-50:] {
		ss[len(ss)-50+i].Paused = true
	}
	if r := Analyze(ss, lim()); strings.Contains(code(r), "no-progress") {
		t.Fatalf("paused time must be exempt:\n%s", r.Text())
	}
}

func TestDeathOfTargetIsUnhealthy(t *testing.T) {
	b := newBot(t)
	ss := b.run(10*time.Minute, true, nil)
	b.srv.CloseClientConnections()
	b.srv.Close()
	ss = append(ss, b.run(10*time.Minute, false, nil)...)
	r := Analyze(ss, lim())
	if !strings.Contains(code(r), "unhealthy") || ss[len(ss)-1].HTTP != 0 {
		t.Fatalf("unreachable target not flagged:\n%s", r.Text())
	}
}

func TestZombieAndRestart(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var ss []Sample
	for i := 0; i < 8; i++ {
		p := ProcSample{PID: 10, Start: 1, State: "S"}
		if i >= 2 && i < 7 {
			p.State = "Z"
		}
		if i == 7 {
			p.PID = 11 // restarted
		}
		ss = append(ss, Sample{T: t0.Add(time.Duration(i) * time.Minute), HealthOK: true, Progress: int64(i), Procs: map[string]ProcSample{"game": p}})
	}
	r := Analyze(ss, DefaultLimits())
	if !strings.Contains(code(r), "zombie") || r.Restarts["game"] != 1 {
		t.Fatalf("%s", r.Text())
	}
}

func TestSentinelCounter(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.jsonl")
	os.WriteFile(p, []byte(`{"kind":"acted"}`+"\n"), 0o644) // predates the soak
	c := &Collector{SentinelLog: p}
	if n := c.sentinelActed(); n != 0 {
		t.Fatalf("old lines counted: %d", n)
	}
	f, _ := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o644)
	f.WriteString(`{"kind":"would"}` + "\n" + `{"kind":"acted"}` + "\n" + `{"kind":"acted"` + "\n"[:0]) // last line partial
	if n := c.sentinelActed(); n != 1 {
		t.Fatalf("got %d, want 1", n)
	}
	f.WriteString("}\n")
	if n := c.sentinelActed(); n != 1 {
		t.Fatalf("completed line: got %d, want 1", n)
	}
	f.Close()
	os.WriteFile(p, []byte(`{"kind":"acted"}`+"\n"), 0o644) // rotated, shorter
	if n := c.sentinelActed(); n != 1 {
		t.Fatalf("rotated: got %d", n)
	}
}

func TestRunSoakAndReport(t *testing.T) {
	b := newBot(t)
	dir := t.TempDir()
	cfg := runCfg{Collector: b.c, Every: 5 * time.Millisecond, For: 100 * time.Millisecond, Out: dir, Limits: lim(),
		Now: time.Now, Log: func(string, ...any) {}}
	r, err := runSoak(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if r.Samples < 5 {
		t.Fatalf("samples = %d", r.Samples)
	}
	for _, f := range []string{"samples.jsonl", "report.json", "report.txt"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Error(err)
		}
	}
	ss, err := readSamples(filepath.Join(dir, "samples.jsonl"))
	if err != nil || len(ss) != r.Samples {
		t.Fatalf("readSamples: %d %v", len(ss), err)
	}
	// Short run: too little data for slopes, so a warning, and exit code 0 or 1 never 2.
	if c := realMain([]string{"report", "-in", filepath.Join(dir, "samples.jsonl")}); c == 2 {
		t.Fatal("report usage error")
	}
}

func TestParseGauges(t *testing.T) {
	g := parseGauges("# TYPE x gauge\ngo_goroutines 12\nother 3\nprocess_open_fds 7\n")
	if len(g) != 2 || g["go_goroutines"] != 12 || g["process_open_fds"] != 7 {
		t.Fatal(g)
	}
}

func TestProfileKeepsExplicitFlags(t *testing.T) {
	fs := flag.NewFlagSet("x", flag.ContinueOnError)
	lim, finish := limitFlags(fs)
	if err := fs.Parse([]string{"-profile", "m1", "-max-sentinel-per-hour", "9"}); err != nil {
		t.Fatal(err)
	}
	if err := finish(); err != nil {
		t.Fatal(err)
	}
	// m1 judges deaths with the death rule, not deaths per hour (plan-v2 10.2)
	if lim.MaxDeathsPerHour != 0 || !lim.Death.Enabled || lim.MaxNoProgress != 5*time.Minute || lim.MaxSentinelPerHr != 9 || lim.MinXPPerHour != 15000 {
		t.Fatalf("%+v", lim)
	}
}

// -max-deaths-per-hour stays an optional extra next to the death rule.
func TestProfileDeathsPerHourOptionalExtra(t *testing.T) {
	fs := flag.NewFlagSet("x", flag.ContinueOnError)
	lim, finish := limitFlags(fs)
	if err := fs.Parse([]string{"-profile", "m24", "-max-deaths-per-hour", "3"}); err != nil {
		t.Fatal(err)
	}
	if err := finish(); err != nil {
		t.Fatal(err)
	}
	if lim.MaxDeathsPerHour != 3 || !lim.Death.Enabled {
		t.Fatalf("%+v", lim)
	}
}

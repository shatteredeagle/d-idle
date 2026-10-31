package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 29, 8, 0, 0, 0, time.UTC)

// synth builds n samples 30 s apart; f may edit each one.
func synth(n int, f func(i int, s *Sample)) []Sample {
	var out []Sample
	for i := 0; i < n; i++ {
		s := Sample{T: t0.Add(time.Duration(i) * 30 * time.Second), HTTP: 200, HealthOK: true, Progress: -1, HasStats: true,
			Level: 4, XP: int64(i) * 50, Kills: int64(i), State: "clearing",
			Procs: map[string]ProcSample{"helper": {PID: 7, Start: 1, FDs: 59, Threads: 8}, "game": {PID: 9, Start: 2, FDs: 190}}}
		if f != nil {
			f(i, &s)
		}
		out = append(out, s)
	}
	return out
}

func TestLevelSkipsSamplesWithoutLevel(t *testing.T) {
	ss := synth(40, func(i int, s *Sample) {
		if i == 0 || i == 39 {
			s.Level = 0 // /status failed at the edges
		}
	})
	r := Analyze(ss, DefaultLimits())
	if r.LevelStart != 4 || r.LevelEnd != 4 || !strings.Contains(r.Text(), "level 4 -> 4") {
		t.Fatalf("level %d -> %d\n%s", r.LevelStart, r.LevelEnd, r.Text())
	}
	ss = synth(40, func(i int, s *Sample) { s.Level = 0 })
	if txt := Analyze(ss, DefaultLimits()).Text(); !strings.Contains(txt, "level ? -> ?") {
		t.Fatalf("unknown level must print ?\n%s", txt)
	}
}

func TestNoSamplesIsNoDataNotFail(t *testing.T) {
	for _, n := range []int{0, 1} {
		r := Analyze(synth(n, nil), DefaultLimits())
		if r.Verdict != "NO-DATA" || r.Pass || !strings.Contains(r.Text(), "NO-DATA") || strings.Contains(r.Text(), "FAIL") {
			t.Fatalf("n=%d: %q\n%s", n, r.Verdict, r.Text())
		}
		if exitCode(r) != 3 {
			t.Errorf("exit %d", exitCode(r))
		}
	}
	if r := Analyze(synth(20, nil), DefaultLimits()); r.Verdict != "PASS" || exitCode(r) != 0 {
		t.Fatalf("%q\n%s", r.Verdict, r.Text())
	}
}

func TestStatusLine(t *testing.T) {
	areas := []string{"Rogue Encampment", "Blood Moor", "Cold Plains", "Stony Field", "Cold Plains"}
	ss := synth(120, func(i int, s *Sample) {
		s.Area = areas[i/25%len(areas)]
		if i >= 60 {
			s.Procs["game"] = ProcSample{PID: 10, Start: 3, FDs: 190} // one restart
		}
		if i == 80 {
			s.Events = map[string]int{"failsafe": 1}
		}
		if i > 100 { // progress stops
			s.XP, s.Kills = ss0(i)
		}
	})
	got := StatusLine(ss, ss[len(ss)-1].T)
	for _, want := range []string{"lvl 4", "xp/h", "deaths 0", "failsafe rungs 1", "restarts 1.0/h", "helper 59 fds/8 thr",
		"Cold Plains > Stony Field > Cold Plains", "10m since progress"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
	t.Log(got)
	if got := StatusLine(nil, t0); !strings.Contains(got, "no samples") {
		t.Error(got)
	}
	if got := StatusLine(ss, ss[len(ss)-1].T.Add(10*time.Minute)); !strings.Contains(got, "LAST SAMPLE 10m AGO") {
		t.Error(got)
	}
}

func ss0(i int) (int64, int64) { return 100 * 50, 100 }

func TestAlertStall(t *testing.T) {
	ss := synth(60, func(i int, s *Sample) {
		if i >= 20 {
			s.XP = 20 * 50 // 20 min flat at the end
		}
	})
	a := EvalAlerts(ss, DefaultLimits(), DefaultAlertRules(), ss[len(ss)-1].T)
	if a.Level != "crit" || !hasCode(a, "stall-xp") {
		t.Fatalf("%+v", a)
	}
	// paused time is exempt
	ss = synth(60, func(i int, s *Sample) {
		if i >= 20 {
			s.XP, s.Paused = 20*50, true
		}
	})
	if a := EvalAlerts(ss, DefaultLimits(), DefaultAlertRules(), ss[len(ss)-1].T); hasCode(a, "stall-xp") {
		t.Fatalf("paused must not alert: %+v", a)
	}
}

func hasCode(a Alerts, c string) bool {
	for _, x := range a.Alerts {
		if x.Code == c {
			return true
		}
	}
	return false
}

func TestAlertRestarts(t *testing.T) {
	ss := synth(120, func(i int, s *Sample) {
		s.Procs["helper"] = ProcSample{PID: 100 + i/10, Start: uint64(i / 10), FDs: 59, Threads: 8}
	})
	a := EvalAlerts(ss, DefaultLimits(), DefaultAlertRules(), ss[len(ss)-1].T) // 11 restarts in 1 h
	if !hasCode(a, "restarts") {
		t.Fatalf("%+v", a)
	}
	r := DefaultAlertRules()
	r.RestartsPerHour = 20
	if a := EvalAlerts(ss, DefaultLimits(), r, ss[len(ss)-1].T); hasCode(a, "restarts") {
		t.Fatal("limit 20 must not alert")
	}
}

func TestAlertFDSlope(t *testing.T) {
	leak := synth(240, func(i int, s *Sample) { // 2 h, +60 fds/h
		s.Procs["game"] = ProcSample{PID: 9, Start: 2, FDs: 190 + i/2}
	})
	l := DefaultLimits()
	l.MaxFDsPerHour = 8
	a := EvalAlerts(leak, l, DefaultAlertRules(), leak[len(leak)-1].T)
	if !hasCode(a, "fd-slope") || a.Level != "warn" {
		t.Fatalf("%+v", a)
	}
	flat := synth(240, nil)
	if a := EvalAlerts(flat, l, DefaultAlertRules(), flat[len(flat)-1].T); a.Level != "ok" || len(a.Alerts) != 0 {
		t.Fatalf("%+v", a)
	}
}

func TestAlertStaleSampler(t *testing.T) {
	ss := synth(20, nil)
	if a := EvalAlerts(ss, DefaultLimits(), DefaultAlertRules(), ss[19].T.Add(20*time.Minute)); !hasCode(a, "sampler-stale") {
		t.Fatalf("%+v", a)
	}
}

func TestWriteLiveAndLimits(t *testing.T) {
	dir := t.TempDir()
	ss := synth(30, nil)
	if err := writeLive(dir, ss, DefaultLimits(), DefaultAlertRules(), ss[29].T); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"alerts.json", "status.txt"} {
		if b, err := os.ReadFile(filepath.Join(dir, f)); err != nil || len(b) == 0 {
			t.Fatalf("%s: %v", f, err)
		}
	}
	if err := writeJSONAtomic(filepath.Join(dir, "limits.json"), Limits{MaxFDsPerHour: 3}); err != nil {
		t.Fatal(err)
	}
	if l, ok := loadLimits(dir); !ok || l.MaxFDsPerHour != 3 {
		t.Fatalf("%+v %v", l, ok)
	}
}

func TestPanicsAlert(t *testing.T) {
	t0 := time.Unix(1000, 0)
	var ss []Sample
	for i, p := range []int{0, 1, 2, 0, 3, 4} { // 0 after 2 is a bot restart
		ss = append(ss, Sample{T: t0.Add(time.Duration(i) * 5 * time.Minute), HealthOK: true, Panics: p, XP: int64(i)})
	}
	if n := panicsSince(ss); n != 6 {
		t.Fatalf("panicsSince = %d, want 6", n)
	}
	a := EvalAlerts(ss, Limits{}, AlertRules{PanicsPerHour: 3}, ss[len(ss)-1].T)
	found := false
	for _, al := range a.Alerts {
		found = found || al.Code == "panics"
	}
	if !found {
		t.Fatalf("no panics alert: %+v", a)
	}
}

func TestAlertHeapRunaway(t *testing.T) {
	rss := func(mb float64) func(int, *Sample) {
		return func(i int, s *Sample) { s.Gauges = map[string]float64{"process_resident_memory_bytes": mb * (1 << 20)} }
	}
	// the G1 attempt 16 shape: 66 MB, then 32 GB at the next 30 s sample
	ss := synth(20, func(i int, s *Sample) {
		mb := 66.0
		if i == 19 {
			mb = 32505
		}
		rss(mb)(i, s)
	})
	a := EvalAlerts(ss, DefaultLimits(), DefaultAlertRules(), ss[len(ss)-1].T)
	if a.Level != "crit" || !hasCode(a, "heap_runaway") {
		t.Fatalf("%+v", a)
	}
	// steady 70 MB: quiet
	ss = synth(20, rss(70))
	if a := EvalAlerts(ss, DefaultLimits(), DefaultAlertRules(), ss[len(ss)-1].T); hasCode(a, "heap_runaway") {
		t.Fatalf("steady heap alerted: %+v", a)
	}
	// +1.2 GB inside the window, far below the absolute limit: growth rule
	ss = synth(20, func(i int, s *Sample) { rss(100+float64(i)*130)(i, s) })
	if a := EvalAlerts(ss, DefaultLimits(), DefaultAlertRules(), ss[len(ss)-1].T); !hasCode(a, "heap_runaway") {
		t.Fatalf("growth missed: %+v", a)
	}
	// slow growth (3 MB per sample) stays under 1 GB in 5 min
	ss = synth(20, func(i int, s *Sample) { rss(100+float64(i)*3)(i, s) })
	if a := EvalAlerts(ss, DefaultLimits(), DefaultAlertRules(), ss[len(ss)-1].T); hasCode(a, "heap_runaway") {
		t.Fatalf("slow growth alerted: %+v", a)
	}
	// a restart after the spike: the new process is small, no alert
	ss = synth(20, func(i int, s *Sample) {
		mb := 40.0
		if i < 15 {
			mb = 20000
		}
		rss(mb)(i, s)
	})
	if a := EvalAlerts(ss, DefaultLimits(), DefaultAlertRules(), ss[len(ss)-1].T); hasCode(a, "heap_runaway") {
		t.Fatalf("restarted bot alerted: %+v", a)
	}
}

package main

import (
	"flag"
	"strings"
	"testing"
	"time"
)

// leakhuntShape builds 30 s samples of a healthy run with the measured fd shape (docs/leakhunt.md, VERIFIED LIVE):
// 54 -> 126 in the first minutes, +14 and +24 at later area changes, flat otherwise. leakPerHour adds a genuine
// leak on top. A progress counter keeps the stall check quiet.
func leakhuntShape(d time.Duration, leakPerHour float64) []Sample {
	t0 := time.Date(2026, 10, 28, 8, 0, 0, 0, time.UTC)
	var out []Sample
	for i := 0; time.Duration(i)*30*time.Second <= d; i++ {
		el := time.Duration(i) * 30 * time.Second
		fds := 54.0
		ev := map[string]int{}
		step := func(at time.Duration, n float64) {
			if el >= at {
				fds += n
			}
			if el == at {
				ev["area_change"] = 1
			}
		}
		step(90*time.Second, 40)
		step(150*time.Second, 32)
		step(25*time.Minute, 14)
		step(47*time.Minute, 24)
		fds += leakPerHour * el.Hours()
		out = append(out, Sample{T: t0.Add(el), HealthOK: true, HTTP: 200, Progress: int64(i), Events: ev,
			Procs: map[string]ProcSample{"game": {PID: 1, Start: 1, State: "S", RSSKB: 400_000, FDs: int(fds), Threads: 20}}})
	}
	return out
}

func fdLeak(r Report) bool {
	for _, f := range r.Findings {
		if f.Code == "leak" && strings.Contains(f.Msg, "fds") {
			return true
		}
	}
	return false
}

func TestPlateauAfterWarmUpPasses(t *testing.T) {
	l := DefaultLimits()
	r := Analyze(leakhuntShape(2*time.Hour, 0), l)
	if fdLeak(r) || !r.Pass {
		t.Fatalf("plateau judged as leak:\n%s", r.Text())
	}
	if len(r.FDPlateau) != 1 || r.FDPlateau[0].Steps != 2 || r.FDPlateau[0].PerHour > 1 {
		t.Fatalf("plateau: %+v", r.FDPlateau)
	}
	if txt := r.Text(); !strings.Contains(txt, "fd plateau") || !strings.Contains(txt, "warm-up      10m0s") {
		t.Fatalf("report lacks plateau/warm-up lines:\n%s", txt)
	}
}

func TestPlateauWithoutHandlingWouldFail(t *testing.T) {
	// Control: with no warm-up and area steps counted, the same healthy shape exceeds a tight limit.
	l := DefaultLimits()
	l.WarmUp, l.AreaStepFDs, l.MaxFDsPerHour = 0, 0, 5
	if r := Analyze(leakhuntShape(time.Hour, 0), l); !fdLeak(r) {
		t.Fatalf("control should fail:\n%s", r.Text())
	}
}

func TestGenuineLeakStillFails(t *testing.T) {
	l := DefaultLimits() // fds limit 5/h
	r := Analyze(leakhuntShape(3*time.Hour, 50), l)
	if !fdLeak(r) {
		t.Fatalf("+50/h leak not caught:\n%s", r.Text())
	}
	if pl := r.FDPlateau[0]; pl.PerHour < 40 || pl.PerHour > 60 {
		t.Fatalf("plateau slope %.1f, want about 50", pl.PerHour)
	}
}

func TestWarmUpDropsEarlyGrowthOfOtherSeries(t *testing.T) {
	ss := leakhuntShape(2*time.Hour, 0)
	for i := range ss { // threads climb 20 -> 80 in the first 8 minutes, then flat
		p := ss[i].Procs["game"]
		p.Threads = 20 + min(i*60/16, 60)
		ss[i].Procs["game"] = p
	}
	if r := Analyze(ss, DefaultLimits()); !r.Pass {
		t.Fatalf("warm-up growth judged:\n%s", r.Text())
	}
	l := DefaultLimits()
	l.WarmUp = 0
	l.MinSlopeSpan = 10 * time.Minute
	if r := Analyze(ss[:60], l); r.Pass {
		t.Fatalf("control without warm-up should fail:\n%s", r.Text())
	}
}

func TestRestartSegmentGetsOwnWarmUp(t *testing.T) {
	ss := leakhuntShape(3*time.Hour, 0)
	// recycle at 1 h: new pid, fds back to 57, then the same warm-up climb again
	for i := range ss {
		el := ss[i].T.Sub(ss[0].T)
		if el >= time.Hour {
			p := ss[i].Procs["game"]
			p.PID, p.Start = 2, 2
			p.FDs = 57 + min(int((el-time.Hour)/(30*time.Second))*5, 70)
			ss[i].Procs["game"] = p
		}
	}
	if r := Analyze(ss, DefaultLimits()); !r.Pass {
		t.Fatalf("second segment warm-up judged:\n%s", r.Text())
	}
}

func TestProfilesSetLeakLimits(t *testing.T) {
	for n, apply := range profiles {
		l := DefaultLimits()
		apply(&l)
		if l.MaxFDsPerHour <= 0 || l.MaxRSSMBPerHour <= 0 || l.MaxGoroutinesPerH <= 0 || l.MaxThreadsPerHour <= 0 {
			t.Errorf("%s: leak limits unset: %+v", n, l)
		}
	}
}

func TestProfileM24(t *testing.T) {
	fs := flag.NewFlagSet("x", flag.ContinueOnError)
	lim, finish := limitFlags(fs)
	if err := fs.Parse([]string{"-profile", "m24", "-max-fds-per-hour", "7"}); err != nil {
		t.Fatal(err)
	}
	if err := finish(); err != nil {
		t.Fatal(err)
	}
	if lim.MaxDeathsPerHour != 0 || !lim.Death.Enabled || lim.MaxNoProgress != 10*time.Minute || lim.MaxLadderPerHour != 2 ||
		lim.MaxSentinelPerHr != 2 || lim.MinXPPerHour != 1000 || lim.MaxRSSMBPerHour != 40 ||
		lim.MaxThreadsPerHour != 5 || lim.MaxGoroutinesPerH != 5 {
		t.Fatalf("%+v", lim)
	}
	if lim.MaxFDsPerHour != 7 {
		t.Fatalf("explicit flag lost: %v", lim.MaxFDsPerHour)
	}
	// untouched limits keep the defaults (wineserver has its own)
	if lim.WineMaxFDsPerHour != 40 || lim.WineMaxGrowth24h != 2000 {
		t.Fatalf("%+v", lim)
	}
}

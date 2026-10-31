package main

import (
	"strings"
	"testing"
	"time"
)

// wineShape: wineserver stepping +12 fds every 40 min (the measured step shape, docs/leakhunt.md WINELEAK) and a
// helper with the same series. slopePerHour adds continuous growth to the named process.
func wineShape(d time.Duration, wineSlope, helperSlope float64) []Sample {
	t0 := time.Date(2026, 10, 29, 8, 0, 0, 0, time.UTC)
	var out []Sample
	for i := 0; time.Duration(i)*time.Minute <= d; i++ {
		el := time.Duration(i) * time.Minute
		steps := float64(el / (40 * time.Minute) * 12)
		out = append(out, Sample{T: t0.Add(el), HealthOK: true, HTTP: 200, Progress: int64(i), Events: map[string]int{},
			Procs: map[string]ProcSample{
				"wineserver": {PID: 2, Start: 1, State: "S", RSSKB: 20000, FDs: 1750 + int(steps+wineSlope*el.Hours()), Threads: 5},
				"helper":     {PID: 3, Start: 1, State: "S", RSSKB: 20000, FDs: 40 + int(helperSlope*el.Hours()), Threads: 5},
			}})
	}
	return out
}

func TestWineserverStepsPass(t *testing.T) {
	r := Analyze(wineShape(5*time.Hour, 0, 0), DefaultLimits())
	if fdLeak(r) || !r.Pass {
		t.Fatalf("steps flagged: %+v", r.Findings)
	}
}

func TestWineserverInsufficientDataNotFail(t *testing.T) {
	// 80 fds/h for 1 h 30 after a 10 min warm-up: under the 2 h span, so not judged
	r := Analyze(wineShape(100*time.Minute, 100, 0), DefaultLimits())
	if fdLeak(r) {
		t.Fatalf("judged too early: %+v", r.Findings)
	}
	if !strings.Contains(r.Text(), "insufficient data") {
		t.Fatal("no insufficient-data note")
	}
}

func TestWineserverRealGrowthFails(t *testing.T) {
	r := Analyze(wineShape(4*time.Hour, 100, 0), DefaultLimits()) // 100/h > 40/h and 2400 per 24 h
	if !fdLeak(r) {
		t.Fatal("real growth not flagged")
	}
}

func TestWineserverProjectionLimit(t *testing.T) {
	l := DefaultLimits()
	l.WineMaxFDsPerHour = 0 // only the 24 h projection: 30/h = 720 passes, 100/h = 2400 fails
	if fdLeak(Analyze(wineShape(4*time.Hour, 30, 0), l)) {
		t.Fatal("30/h flagged")
	}
	if !fdLeak(Analyze(wineShape(4*time.Hour, 100, 0), l)) {
		t.Fatal("100/h not flagged")
	}
}

func TestHelperFDsStayStrict(t *testing.T) {
	r := Analyze(wineShape(4*time.Hour, 0, 10), DefaultLimits()) // 10/h > 5/h
	var hit bool
	for _, f := range r.Findings {
		hit = hit || (f.Code == "leak" && strings.Contains(f.Msg, "proc:helper:fds"))
	}
	if !hit {
		t.Fatalf("helper leak missed: %+v", r.Findings)
	}
}

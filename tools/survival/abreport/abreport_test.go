package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// synth builds a bot log: n minute lines, xp gained xpPerMin each, one chicken at minute chickAt (0 = none), deaths at the end,
// a constant life loss per minute in area 2 (and area 3 from half time).
func synth(n int, xpPerMin uint64, chickAt, deaths, lost int) string {
	var b strings.Builder
	t0 := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	for i := 0; i <= n; i++ {
		d := 0
		if i == n {
			d = deaths
		}
		area := 2
		if i > n/2 {
			area = 3
		}
		fmt.Fprintf(&b, "time=%s msg=minute area=%d clvl=%d xp=%d kills=%d deaths=%d potions=%d gold=%d restarts=0 stashGold=0 life=100 maxLife=100 lifeLost=%d hitsTaken=1 hpPolls=60\n",
			t0.Add(time.Duration(i)*time.Minute).Format(time.RFC3339), area, 11+i/60, 1000+uint64(i)*xpPerMin, i*3, d, i/10, 500+i, lost)
		if chickAt > 0 && i == chickAt {
			fmt.Fprintf(&b, "time=%s msg=\"safety decision\" action=chicken life=10\n", t0.Format(time.RFC3339))
			fmt.Fprintf(&b, "time=%s msg=\"merc: died\"\n", t0.Format(time.RFC3339))
		}
	}
	return b.String()
}

func writeArm(t *testing.T, dir, arm string, seed int, log string) {
	t.Helper()
	p := filepath.Join(dir, arm)
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p, fmt.Sprintf("%d.log", seed)), []byte(log), 0o644); err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, dir string, o Options) Summary {
	t.Helper()
	b, _, err := Load(dir, "legacy")
	if err != nil {
		t.Fatal(err)
	}
	m, _, err := Load(dir, "margin")
	if err != nil {
		t.Fatal(err)
	}
	return Compare(b, m, o)
}

var gate = Options{Horizon: 120, XPFloor: 1.0, MinSeeds: 3}

func TestMetricsOfOneRun(t *testing.T) {
	lines := strings.Split(synth(120, 10, 40, 1, 4), "\n")
	dir := t.TempDir()
	writeArm(t, dir, "legacy", 1, strings.Join(lines, "\n"))
	runs, _, err := Load(dir, "legacy")
	if err != nil || len(runs) != 1 {
		t.Fatalf("%v %d", err, len(runs))
	}
	r := runs[0]
	if r.Minutes != 120 || r.Deaths != 1 || r.Chickens != 1 || r.MercDeaths != 1 {
		t.Fatalf("%+v", r)
	}
	for _, c := range Checkpoints {
		if want := float64(c * 10); r.XPAt[c] != want {
			t.Errorf("xp at %d = %v, want %v", c, r.XPAt[c], want)
		}
	}
	if r.Potions != 12 || r.Gold != 120 || r.ClvlStart != 11 || r.ClvlEnd != 13 {
		t.Fatalf("potions/gold/clvl %+v", r)
	}
	if a2, a3 := r.Intake[2], r.Intake[3]; a2 == nil || a3 == nil || a2.Minutes != 60 || a3.Minutes != 60 || a2.PerMin() != 4 {
		t.Fatalf("intake %+v %+v", a2, a3)
	}
}

func TestGateVerdicts(t *testing.T) {
	cases := []struct {
		name                  string
		baseXP, testXP        uint64
		baseDeaths, testDeath int
		chickBase, chickTest  int
		seeds                 int
		want                  string
	}{
		{"pass", 10, 11, 1, 0, 40, 0, 3, "PASS"},
		{"equal xp passes", 10, 10, 0, 0, 0, 0, 3, "PASS"},
		{"more deaths fail", 10, 12, 0, 1, 0, 0, 3, "FAIL"},
		{"more chickens fail", 10, 12, 0, 0, 0, 40, 3, "FAIL"},
		{"clearly less xp fails", 10, 7, 0, 0, 0, 0, 3, "FAIL"},
		{"too few seeds", 10, 11, 0, 0, 0, 0, 2, "INCONCLUSIVE"},
		{"eight seeds", 10, 11, 0, 0, 0, 0, 8, "PASS"},
	}
	for _, c := range cases {
		dir := t.TempDir()
		for s := 1; s <= c.seeds; s++ {
			// a small per-seed wobble so the paired SD is not zero
			wob := uint64(s % 2)
			writeArm(t, dir, "legacy", s, synth(120, c.baseXP*10+wob, c.chickBase, c.baseDeaths, 3))
			writeArm(t, dir, "margin", s, synth(120, c.testXP*10+wob, c.chickTest, c.testDeath, 2))
		}
		got := run(t, dir, gate)
		if got.Verdict != c.want {
			t.Errorf("%s: verdict %s (%v), want %s\n%s", c.name, got.Verdict, got.Reasons, c.want, got.Format())
		}
	}
}

func TestXPFloorStrictVersusTolerant(t *testing.T) {
	dir := t.TempDir()
	for s := 1; s <= 3; s++ {
		writeArm(t, dir, "legacy", s, synth(120, 1000+uint64(s)*20, 0, 0, 3))
		writeArm(t, dir, "margin", s, synth(120, 980+uint64(s)*40, 0, 0, 2)) // within noise, slightly below on seed 1
	}
	strict := run(t, dir, gate)
	tol := run(t, dir, Options{Horizon: 120, XPFloor: 0.9, MinSeeds: 3})
	if strict.Verdict == "FAIL" {
		t.Fatalf("a within-noise shortfall must not FAIL: %v", strict.Reasons)
	}
	if tol.Verdict != "PASS" {
		t.Fatalf("tolerant floor: %s %v", tol.Verdict, tol.Reasons)
	}
}

func TestShortOrMissingSeedDroppedFromBoth(t *testing.T) {
	dir := t.TempDir()
	for s := 1; s <= 3; s++ {
		writeArm(t, dir, "legacy", s, synth(120, 10, 0, 0, 3))
		writeArm(t, dir, "margin", s, synth(120, 10, 0, 0, 3))
	}
	writeArm(t, dir, "legacy", 4, synth(120, 10, 0, 0, 3))
	writeArm(t, dir, "margin", 4, synth(50, 10, 0, 0, 3)) // crashed
	writeArm(t, dir, "margin", 5, synth(120, 10, 0, 0, 3))
	got := run(t, dir, gate)
	if len(got.Pairs) != 3 || len(got.Excluded) != 2 {
		t.Fatalf("pairs %d excluded %v", len(got.Pairs), got.Excluded)
	}
}

func TestOldLogWithoutIntake(t *testing.T) {
	old := strings.ReplaceAll(synth(120, 10, 0, 0, 3), "lifeLost=3 hitsTaken=1 hpPolls=60", "dmgTaken=0 hpMinPct=90")
	dir := t.TempDir()
	for s := 1; s <= 3; s++ {
		writeArm(t, dir, "legacy", s, old)
		writeArm(t, dir, "margin", s, old)
	}
	got := run(t, dir, gate)
	if !strings.Contains(got.Format(), "no minute line carries lifeLost") {
		t.Fatalf("%s", got.Format())
	}
}

func TestPairedStat(t *testing.T) {
	s := PairedStat([]float64{1, 2, 3})
	if s.Mean != 2 || s.SD != 1 || !s.IntervalOK || s.Lo > 0 || s.Hi < 4 {
		t.Fatalf("%+v", s)
	}
	if one := PairedStat([]float64{5}); one.IntervalOK || one.Mean != 5 {
		t.Fatalf("%+v", one)
	}
}

func TestDeathSlack(t *testing.T) {
	dir := t.TempDir()
	for s := 1; s <= 3; s++ {
		d := 0
		if s == 1 {
			d = 1
		}
		writeArm(t, dir, "legacy", s, synth(120, 10, 0, 0, 3))
		writeArm(t, dir, "margin", s, synth(120, 10, 0, d, 3))
	}
	if got := run(t, dir, gate); got.Verdict != "FAIL" {
		t.Fatalf("strict: %s", got.Verdict)
	}
	o := gate
	o.DeathSlack = 1
	if got := run(t, dir, o); got.Verdict != "PASS" {
		t.Fatalf("slack 1: %s %v", got.Verdict, got.Reasons)
	}
}

// A 2 h run has 119 intervals: the horizon XP is the last minute line, not the 90 minute checkpoint.
func TestHorizonUsesLastMinute(t *testing.T) {
	dir := t.TempDir()
	writeArm(t, dir, "legacy", 1, synth(119, 10, 0, 0, 1))
	writeArm(t, dir, "margin", 1, synth(119, 20, 0, 0, 1))
	b, _, _ := Load(dir, "legacy")
	m, _, _ := Load(dir, "margin")
	r := b[0]
	if r.Minutes != 119 || r.XPAt[30] != 300 || r.XPAt[60] != 600 || r.XPAt[90] != 900 || r.XPEnd != 1190 {
		t.Fatalf("%+v", r)
	}
	if _, ok := r.XPAt[120]; ok {
		t.Fatal("no minute 120 expected")
	}
	if got := xpAtHorizon(r, 120); got != 1190 {
		t.Fatalf("horizon xp %v, want the last minute 1190", got)
	}
	s := Compare(b, m, gate)
	if s.XPRatio != 2 || len(s.Pairs) != 1 {
		t.Fatalf("ratio %v pairs %d", s.XPRatio, len(s.Pairs))
	}
	if out := s.Format(); !strings.Contains(out, "xp@119=1190") {
		t.Fatalf("label missing:\n%s", out)
	}
}

// Runs that end a few minutes apart are compared at the same minute.
func TestPairComparedAtOneMinute(t *testing.T) {
	dir := t.TempDir()
	writeArm(t, dir, "legacy", 1, synth(119, 10, 0, 0, 1))
	writeArm(t, dir, "margin", 1, synth(116, 10, 0, 0, 1))
	b, _, _ := Load(dir, "legacy")
	m, _, _ := Load(dir, "margin")
	s := Compare(b, m, gate)
	if len(s.Pairs) != 1 || s.XPRatio != 1 || s.XPDiff.Mean != 0 {
		t.Fatalf("equal xp per minute must compare equal at minute 116: ratio %v diff %v", s.XPRatio, s.XPDiff.Mean)
	}
}

// A pair whose hero failed the prep self-check or was banked is no survival evidence (featuresoak pass 1: every point banked).
func TestBadHeroDropsThePair(t *testing.T) {
	dir := t.TempDir()
	ok := "hero: run=sa-x profile=played class=paladin build=paladin-normal30-v2 clvl=12 attack=Zeal selfcheck=ok tags=CHEATED,played\n"
	for s := 1; s <= 3; s++ {
		writeArm(t, dir, "legacy", s, ok+synth(120, 10, 0, 0, 3))
		writeArm(t, dir, "margin", s, ok+synth(120, 10, 0, 0, 3))
	}
	writeArm(t, dir, "legacy", 4, strings.Replace(ok, "selfcheck=ok", "selfcheck=skillpts,attack", 1)+synth(120, 10, 0, 0, 3))
	writeArm(t, dir, "margin", 4, ok+synth(120, 10, 0, 0, 3))
	writeArm(t, dir, "legacy", 5, strings.Replace(ok, "profile=played", "profile=banked", 1)+synth(120, 10, 0, 0, 3))
	writeArm(t, dir, "margin", 5, synth(120, 10, 0, 0, 3)) // a log from before the hero library: no line, no judgement
	got := run(t, dir, gate)
	if len(got.Pairs) != 3 || len(got.Excluded) != 2 {
		t.Fatalf("pairs %d excluded %v", len(got.Pairs), got.Excluded)
	}
	if !strings.Contains(got.Excluded[0], "selfcheck=skillpts,attack") || !strings.Contains(got.Excluded[1], "banked") {
		t.Errorf("excluded %v", got.Excluded)
	}
}

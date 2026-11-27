package scenario

import (
	"strings"
	"testing"
	"time"
)

const nightlyFile = "../../tools/provingground/nightly.json"

func fakeScs() ([]Scenario, TierFile) {
	mk := func(id string, seeds ...int64) Scenario {
		return Scenario{ID: id, Pre: Pre{Start: StartTown}, Matrix: Matrix{Classes: []string{"Paladin", "Sorceress"}, Seeds: seeds}}
	}
	scs := []Scenario{mk("b.t3", 1, 2), mk("a.t3", 1, 2), mk("t2.one", 1), mk("t2.two", 1), mk("t2.three", 1), mk("pend.t3", 1), mk("quar.t3", 1)}
	scs[5].Status = "pending"
	cheated := mk("t4.cheated", 1)
	cheated.Pre = Pre{Start: StartField, Area: 2, Warp: &Cell{X: 1, Y: 1}}
	scs = append(scs, cheated, mk("t4.played", 1))
	tf := TierFile{Tiers: map[string]TierEntry{}}
	for _, s := range scs {
		t := TierT3
		switch {
		case strings.HasPrefix(s.ID, "t2."):
			t = TierT2
		case strings.HasPrefix(s.ID, "t4."):
			t = TierT4
		}
		tf.Tiers[s.ID] = TierEntry{Tier: t, Source: "map"}
	}
	return scs, tf
}

func TestNightlyFileValid(t *testing.T) {
	if _, err := LoadNightly(nightlyFile); err != nil {
		t.Fatal(err)
	}
}

func TestNightlyValidateRefuses(t *testing.T) {
	base, err := LoadNightly(nightlyFile)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*NightlyPlan){
		"window":      func(p *NightlyPlan) { p.Window.End = "08:00" },
		"zone":        func(p *NightlyPlan) { p.Window.Zone = "UTC" },
		"too many":    func(p *NightlyPlan) { p.MaxJobs = 4 },
		"jobs>max":    func(p *NightlyPlan) { p.MaxJobs = 2 },
		"nice":        func(p *NightlyPlan) { p.Nice = 0 },
		"hidden":      func(p *NightlyPlan) { p.HiddenOnly = false },
		"sandbox":     func(p *NightlyPlan) { p.SandboxUntouched = false },
		"smoke block": func(p *NightlyPlan) { p.Smoke.ReportOnly = false },
		"smoke days":  func(p *NightlyPlan) { p.Smoke.CleanDays = 3 },
		"job past":    func(p *NightlyPlan) { p.Jobs = append([]NightlyJob(nil), p.Jobs...); p.Jobs[0].MaxWallMin = 481 },
		"parallel":    func(p *NightlyPlan) { p.Jobs = append([]NightlyJob(nil), p.Jobs...); p.Jobs[0].Parallel = 5 },
		"no cap": func(p *NightlyPlan) {
			p.Jobs = append([]NightlyJob(nil), p.Jobs...)
			p.Jobs[0].Stages = append([]NightlyStage(nil), p.Jobs[0].Stages...)
			p.Jobs[0].Stages[0].CapRuns = 0
		},
	}
	for name, mut := range cases {
		p := base
		mut(&p)
		if err := p.Validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestNightlyInWindow(t *testing.T) {
	p, _ := LoadNightly(nightlyFile)
	for at, want := range map[string]bool{
		"2026-11-28T03:59:00Z": false, "2026-11-28T04:00:00Z": true, // 23:00 EST
		"2026-11-28T11:59:00Z": true, "2026-11-28T12:00:00Z": false, // 07:00 EST
		"2026-11-28T18:00:00Z": false,
	} {
		ts, _ := time.Parse(time.RFC3339, at)
		if got := p.InWindow(ts); got != want {
			t.Errorf("%s: %v want %v", at, got, want)
		}
	}
}

func TestNightlyExpand(t *testing.T) {
	scs, tf := fakeScs()
	p, _ := LoadNightly(nightlyFile)
	sat := time.Date(2026, 11, 28, 5, 0, 0, 0, time.UTC)
	sun := sat.AddDate(0, 0, 1)
	count := func(runs []PlannedRun, stage string) (n int) {
		for _, r := range runs {
			if strings.HasSuffix(r.Stage, stage) {
				n++
			}
		}
		return
	}
	runs, _ := p.Expand(scs, tf, map[string]bool{"quar.t3": true}, sat)
	// T3: a.t3, b.t3 x 2 classes x 1 seed; pending and quarantined are left out.
	if n := count(runs, "tier-t3"); n != 4 {
		t.Fatalf("saturday T3 runs %d, want 4: %+v", n, runs)
	}
	if n := count(runs, "tier-t2"); n != 3 { // first class, first seed, three scenarios
		t.Fatalf("T2 runs %d", n)
	}
	for _, r := range runs {
		if strings.HasPrefix(r.Scenario, "pend.") || strings.HasPrefix(r.Scenario, "quar.") {
			t.Fatalf("planned %s", r.Scenario)
		}
	}
	runs, _ = p.Expand(scs, tf, nil, sun)
	if n := count(runs, "tier-t3"); n != 2*2*2+2 { // a.t3 and b.t3 get the second seed on Sunday; quar.t3 one seed x 2 classes
		t.Fatalf("sunday T3 runs %d", n)
	}
	if n := count(runs, "pairs-t2"); n != 6 {
		t.Fatalf("pair runs %d", n)
	}
	for _, r := range runs {
		if strings.HasSuffix(r.Stage, "e2e-t4") && r.Scenario != "t4.played" {
			t.Fatalf("e2e picked %s, want the scenario with no CHEATED setup", r.Scenario)
		}
	}
}

func TestNightlyExpandCapRotates(t *testing.T) {
	scs, tf := fakeScs()
	p, _ := LoadNightly(nightlyFile)
	p.Jobs = []NightlyJob{{Name: "j", MaxWallMin: 60, Parallel: 1, Stages: []NightlyStage{{Kind: StageTier, Tier: TierT3, Classes: "all", CapRuns: 2}}}}
	d1 := time.Date(2026, 11, 2, 5, 0, 0, 0, time.UTC)
	a, notes := p.Expand(scs, tf, nil, d1)
	b, _ := p.Expand(scs, tf, nil, d1.AddDate(0, 0, 1))
	if len(a) != 2 || len(b) != 2 || len(notes) != 1 || !strings.Contains(notes[0], "cut to the cap 2") {
		t.Fatalf("a=%v b=%v notes=%v", a, b, notes)
	}
	if a[0] == b[0] {
		t.Fatalf("the cut tail did not rotate: %v %v", a, b)
	}
}

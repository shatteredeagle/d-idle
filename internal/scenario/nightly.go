package scenario

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// NightlyVersion is the format of tools/provingground/nightly.json: the nightly run set as data (ADR 0027 items 2 and 4,
// design 2026-11-21 sections 3.2, 4.4 and 4.6). The scheduler (private ops repo) only executes what Expand returns.
const NightlyVersion = "didle.pg-nightly/1"

// Stage kinds of a nightly job.
const (
	StageTier  = "tier"  // every active scenario of a tier, times classes and seeds
	StagePairs = "pairs" // determinism pairs: the same scenario, class and seed twice
	StageE2E   = "e2e"   // a T4 item, one per night, rotating
)

// NightlyPlan is the whole file. Window is the standing night allowance of the second worker host; the smoke stage runs
// after each push on the builder and is not bound by the window.
type NightlyPlan struct {
	Version string `json:"version"`
	Window  struct {
		Start string `json:"start"` // HH:MM
		End   string `json:"end"`   // HH:MM, the next morning when before start
		Zone  string `json:"zone"`  // EST only: a fixed UTC-5 offset, as the ADR words it
	} `json:"window"`
	MaxJobs          int  `json:"max_jobs"` // "at most 3 runs" read as at most 3 scheduler jobs a night (U: ADR 0027 item 4 does not define a run)
	Nice             int  `json:"nice"`
	HiddenOnly       bool `json:"hidden_only"`
	SandboxUntouched bool `json:"sandbox_untouched"`
	Smoke            struct {
		Tier       string `json:"tier"`
		ReportOnly bool   `json:"report_only"`
		CleanDays  int    `json:"clean_days_before_decision"` // 7 clean days, then the user decides on blocking (Q2)
		MaxWallMin int    `json:"max_wall_min"`
		Lane       string `json:"lane"`
	} `json:"smoke"`
	Jobs []NightlyJob `json:"jobs"`
}

// NightlyJob is one scheduler job (one labrun call chain); MaxWallMin is its hard limit.
type NightlyJob struct {
	Name       string         `json:"name"`
	MaxWallMin int            `json:"max_wall_min"`
	Parallel   int            `json:"parallel"` // lab slots at once, at most 4
	Stages     []NightlyStage `json:"stages"`
}

// NightlyStage selects runs. Classes is "all" (the scenario's matrix) or "first". ExtraSeedDay is a weekday name on
// which a second seed joins (T3: the second seed weekly). Count and Pick apply to pairs and e2e.
type NightlyStage struct {
	Kind         string `json:"kind"`
	Tier         string `json:"tier"`
	Classes      string `json:"classes,omitempty"`
	ExtraSeedDay string `json:"extra_seed_day,omitempty"`
	Count        int    `json:"count,omitempty"`
	PreferEarned bool   `json:"prefer_earned,omitempty"` // e2e: only scenarios whose setup is not CHEATED
	CapRuns      int    `json:"cap_runs"`
}

// PlannedRun is one scenario, class and seed to run; Pair is 1 or 2 for the two halves of a determinism pair.
type PlannedRun struct {
	Job      string `json:"job"`
	Stage    string `json:"stage"`
	Scenario string `json:"scenario"`
	Class    string `json:"class"`
	Seed     int64  `json:"seed"`
	Pair     int    `json:"pair,omitempty"`
}

// LoadNightly reads and validates the file; unknown fields are errors.
func LoadNightly(path string) (NightlyPlan, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return NightlyPlan{}, err
	}
	var p NightlyPlan
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return NightlyPlan{}, fmt.Errorf("%s: %w", path, err)
	}
	if err := p.Validate(); err != nil {
		return NightlyPlan{}, fmt.Errorf("%s: %w", path, err)
	}
	return p, nil
}

func parseClock(s string) (int, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, fmt.Errorf("clock %q (want HH:MM)", s)
	}
	return t.Hour()*60 + t.Minute(), nil
}

// WindowMinutes is the length of the night window.
func (p NightlyPlan) WindowMinutes() int {
	a, _ := parseClock(p.Window.Start)
	b, _ := parseClock(p.Window.End)
	if b <= a {
		b += 24 * 60
	}
	return b - a
}

// InWindow reports whether t is inside the standing allowance (fixed UTC-5, wrapping midnight).
func (p NightlyPlan) InWindow(t time.Time) bool {
	a, _ := parseClock(p.Window.Start)
	b, _ := parseClock(p.Window.End)
	l := t.UTC().Add(-5 * time.Hour)
	m := l.Hour()*60 + l.Minute()
	if a < b {
		return m >= a && m < b
	}
	return m >= a || m < b
}

// Validate keeps the plan inside ADR 0027: the window, at most 3 jobs, niced, hidden instances only, the sandbox left
// alone, a smoke stage that is report-only, every job inside the window.
func (p NightlyPlan) Validate() error {
	if p.Version != NightlyVersion {
		return fmt.Errorf("version %q, want %q", p.Version, NightlyVersion)
	}
	if _, err := parseClock(p.Window.Start); err != nil {
		return fmt.Errorf("window.start: %w", err)
	}
	if _, err := parseClock(p.Window.End); err != nil {
		return fmt.Errorf("window.end: %w", err)
	}
	if p.Window.Start != "23:00" || p.Window.End != "07:00" || p.Window.Zone != "EST" {
		return fmt.Errorf("window %s-%s %s: the standing allowance is 23:00-07:00 EST (ADR 0027 item 4; a change needs the user)", p.Window.Start, p.Window.End, p.Window.Zone)
	}
	if p.MaxJobs < 1 || p.MaxJobs > 3 || len(p.Jobs) > p.MaxJobs {
		return fmt.Errorf("max_jobs %d with %d jobs: at most 3 runs a night", p.MaxJobs, len(p.Jobs))
	}
	if p.Nice < 10 || p.Nice > 19 {
		return fmt.Errorf("nice %d: niced means 10 to 19", p.Nice)
	}
	if !p.HiddenOnly || !p.SandboxUntouched {
		return fmt.Errorf("hidden_only and sandbox_untouched must both be true (ADR 0027 item 4)")
	}
	if p.Smoke.Tier != TierT2 || !p.Smoke.ReportOnly || p.Smoke.CleanDays != 7 || p.Smoke.MaxWallMin <= 0 || p.Smoke.Lane == "" {
		return fmt.Errorf("smoke: want tier T2, report_only true, clean_days_before_decision 7, max_wall_min and lane set (Q2)")
	}
	names := map[string]bool{}
	for _, j := range p.Jobs {
		if j.Name == "" || names[j.Name] {
			return fmt.Errorf("job name %q missing or duplicate", j.Name)
		}
		names[j.Name] = true
		if j.MaxWallMin <= 0 || j.MaxWallMin > p.WindowMinutes() {
			return fmt.Errorf("job %s: max_wall_min %d outside 1..%d (the window)", j.Name, j.MaxWallMin, p.WindowMinutes())
		}
		if j.Parallel < 1 || j.Parallel > 4 {
			return fmt.Errorf("job %s: parallel %d (1 to 4 lab slots)", j.Name, j.Parallel)
		}
		if len(j.Stages) == 0 {
			return fmt.Errorf("job %s: no stages", j.Name)
		}
		for i, st := range j.Stages {
			if err := st.validate(); err != nil {
				return fmt.Errorf("job %s stage %d: %w", j.Name, i, err)
			}
		}
	}
	return nil
}

func (st NightlyStage) validate() error {
	switch st.Tier {
	case TierT2, TierT3, TierT4:
	default:
		return fmt.Errorf("tier %q", st.Tier)
	}
	if st.CapRuns <= 0 {
		return fmt.Errorf("cap_runs must be positive (every stage has a cap)")
	}
	switch st.Kind {
	case StageTier:
		if st.Classes != "all" && st.Classes != "first" {
			return fmt.Errorf("classes %q (all or first)", st.Classes)
		}
		if st.ExtraSeedDay != "" {
			if _, ok := weekdayByName(st.ExtraSeedDay); !ok {
				return fmt.Errorf("extra_seed_day %q", st.ExtraSeedDay)
			}
		}
	case StagePairs, StageE2E:
		if st.Count <= 0 {
			return fmt.Errorf("%s needs count", st.Kind)
		}
		if st.Kind == StageE2E && st.Tier != TierT4 {
			return fmt.Errorf("e2e stage runs T4 items")
		}
	default:
		return fmt.Errorf("kind %q", st.Kind)
	}
	return nil
}

func weekdayByName(s string) (time.Weekday, bool) {
	for d := time.Sunday; d <= time.Saturday; d++ {
		if strings.EqualFold(d.String(), s) {
			return d, true
		}
	}
	return 0, false
}

// Expand turns the plan into the runs of one night. day only picks the weekday and the rotation; quarantined scenarios
// are left out of the set (a quarantined scenario is released by a person, then runs again).
// Pending scenarios are never planned. A stage that has more runs than its cap is cut at the cap in id order, and the
// cut is reported in the second result so nobody reads a short night as a full one.
func (p NightlyPlan) Expand(scs []Scenario, tiers TierFile, quarantined map[string]bool, day time.Time) ([]PlannedRun, []string) {
	sorted := append([]Scenario(nil), scs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	var out []PlannedRun
	var notes []string
	rot := day.YearDay()
	for _, j := range p.Jobs {
		for _, st := range j.Stages {
			name := j.Name + "/" + st.Kind + "-" + strings.ToLower(st.Tier)
			var cand []Scenario
			for _, s := range sorted {
				if s.Pending() || quarantined[s.ID] || tiers.Tiers[s.ID].Tier != st.Tier {
					continue
				}
				cand = append(cand, s)
			}
			var runs []PlannedRun
			switch st.Kind {
			case StageTier:
				for _, s := range cand {
					classes := s.Matrix.Classes
					if st.Classes == "first" {
						classes = classes[:1]
					}
					seeds := s.Matrix.Seeds[:1]
					if wd, ok := weekdayByName(st.ExtraSeedDay); ok && day.Weekday() == wd && len(s.Matrix.Seeds) > 1 {
						seeds = s.Matrix.Seeds[:2]
					}
					for _, c := range classes {
						for _, sd := range seeds {
							runs = append(runs, PlannedRun{Job: j.Name, Stage: name, Scenario: s.ID, Class: c, Seed: sd})
						}
					}
				}
			case StagePairs, StageE2E:
				if st.PreferEarned {
					cand = preferEarned(cand)
				}
				for i := 0; i < st.Count && i < len(cand); i++ {
					s := cand[(rot+i)%len(cand)]
					r := PlannedRun{Job: j.Name, Stage: name, Scenario: s.ID, Class: s.Matrix.Classes[0], Seed: s.Matrix.Seeds[0]}
					if st.Kind == StagePairs {
						a, b := r, r
						a.Pair, b.Pair = 1, 2
						runs = append(runs, a, b)
					} else {
						runs = append(runs, r)
					}
				}
			}
			if len(runs) > st.CapRuns {
				// rotate by day so the same tail is not the one that is always cut
				at := rot * st.CapRuns % len(runs)
				notes = append(notes, fmt.Sprintf("%s: %d runs cut to the cap %d (window starts at run %d)", name, len(runs), st.CapRuns, at))
				runs = append(append([]PlannedRun(nil), runs[at:]...), runs[:at]...)[:st.CapRuns]
			}
			if len(runs) == 0 {
				notes = append(notes, name+": no scenario selected")
			}
			out = append(out, runs...)
		}
	}
	return out, notes
}

// preferEarned keeps only the scenarios whose first class and seed need no CHEATED setup (all of them when none qualifies,
// so the night still runs one T4 item).
func preferEarned(in []Scenario) []Scenario {
	var good []Scenario
	for _, s := range in {
		su, err := BuildSetup(s, s.Matrix.Classes[0], s.Matrix.Seeds[0])
		if err == nil && SetupTag(su.Tags) != SetupCheated && SetupTag(su.Tags) != SetupImpossible {
			good = append(good, s)
		}
	}
	if len(good) == 0 {
		return in
	}
	return good
}

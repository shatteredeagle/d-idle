package d2dsoakstat

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
)

// Rule is the pass rule of one metric. A gated metric passes when |d2d mean - baseline mean| <= max(Rel*|baseline|, Abs).
// Abs is the tolerance for metrics whose baseline is near 0 (deaths): a relative gap of a zero baseline is meaningless.
type Rule struct {
	Gate bool    `json:"gate"`
	Rel  float64 `json:"rel,omitempty"`
	Abs  float64 `json:"abs,omitempty"`
	Why  string  `json:"why,omitempty"`
}

// BaselineRun is one original-game soak: its extracted metrics and where they come from.
type BaselineRun struct {
	Run
	Source string `json:"source"` // file or doc the numbers come from, with a V/S/U tag
	Note   string `json:"note,omitempty"`
}

// Baselines is the file tools/d2dsoak/baselines.json.
type Baselines struct {
	Description string          `json:"description"`
	Rules       map[string]Rule `json:"rules"`
	Runs        []BaselineRun   `json:"runs"`
	// MinSeeds, MinHours are the sample sizes below which no PASS/FAIL is given (see docs/d2d-soak.md section 3).
	MinSeeds        int     `json:"minSeeds"`
	MinHoursPerSeed float64 `json:"minHoursPerSeed"`
	MinTotalHours   float64 `json:"minTotalHours"`
}

// ReadBaselines parses the JSON file.
func ReadBaselines(r io.Reader) (Baselines, error) {
	var b Baselines
	err := json.NewDecoder(r).Decode(&b)
	if err == nil && len(b.Runs) == 0 {
		err = fmt.Errorf("baselines: no runs")
	}
	return b, err
}

// DefaultRules is the rule set the committed baselines.json carries (kept here so tests and the baseline command agree).
func DefaultRules() map[string]Rule {
	return map[string]Rule{
		"xp_per_hour":           {Gate: true, Rel: 0.20, Why: "the plan's fidelity gate (plan-v2 11, d2d-backend 7)"},
		"kills_per_hour":        {Gate: true, Rel: 0.20},
		"deaths_per_hour":       {Gate: true, Rel: 0.20, Abs: 0.5, Why: "baseline is 0 to 0.5 deaths/h; one extra death per 2 h is noise"},
		"deaths_per_1000_kills": {Gate: true, Rel: 0.20, Abs: 1.0, Why: "baseline about 0 to 1"},
		"potions_per_hour":      {Rel: 0.20, Why: "info: only one clean baseline run (a18), vendor prices differ on d2d"},
		"potions_per_100_kills": {Rel: 0.25, Why: "info under v1; gated under v2"},
		"min_life_fraction":     {Rel: 0.25, Why: "info under v1; gated under v2"},
		"damage_per_kill":       {Rel: 0.25, Why: "info under v1; gated under v2"},
		"gold_per_hour":         {Rel: 0.20, Why: "info: net balance, stash banking and vendor prices differ"},
		"recovery_share":        {Abs: 0.05, Why: "info: share of time"},
		"farming_share":         {Abs: 0.10, Why: "info"},
		"town_share":            {Abs: 0.10, Why: "info"},
		"repeat_sites":          {Abs: 1, Why: "info"},
		"clvl_gain":             {Rel: 0.20, Why: "info: depends on the start level"},
		"clvl_1h":               {Abs: 1, Why: "info: only comparable with the same start clvl"},
		"clvl_2h":               {Abs: 1, Why: "info: only comparable with the same start clvl"},
	}
}

// Status of one row.
const (
	Pass         = "PASS"
	Fail         = "FAIL"
	Inconclusive = "INCONCLUSIVE"
	Info         = "info"
	// NoData is the overall verdict when no seed was usable: nothing was judged, so it is never a pass.
	NoData = "NO-DATA"
)

// Exit codes of the verdict (docs/d2d-soak.md "Exit codes"): only PASS is 0, so a script cannot read a non-pass as a pass.
const (
	ExitPass         = 0
	ExitInconclusive = 2
	ExitFail         = 3
)

// ExitCode maps an overall verdict to its process exit code; FAIL, NO-DATA and anything unknown are 3 (fail closed).
func ExitCode(overall string) int {
	switch overall {
	case Pass:
		return ExitPass
	case Inconclusive:
		return ExitInconclusive
	}
	return ExitFail
}

// Row is one line of the comparison.
type Row struct {
	Metric   string  `json:"metric"`
	D2D      Stat    `json:"d2d"`
	Base     Stat    `json:"baseline"`
	GapPct   float64 `json:"gapPct"` // (d2d - base) / |base| * 100; NaN when the baseline is 0
	Tol      float64 `json:"tol"`    // absolute tolerance in the metric's unit
	Status   string  `json:"status"`
	Note     string  `json:"note,omitempty"`
	NeedSeed int     `json:"needSeeds,omitempty"`
}

// Verdict is the whole comparison.
type Verdict struct {
	Overall  string   `json:"overall"`
	Rows     []Row    `json:"rows"`
	Warnings []string `json:"warnings,omitempty"`
	// Seeds is the number of seeds that count; Excluded lists the others as "name: reason" (zero exposure, zero kills).
	Seeds    int      `json:"seeds"`
	Excluded []string `json:"excludedSeeds,omitempty"`
	// Inv is the invariants report (counters per seed, groups, coverage, INVARIANTS verdict): separate from Overall and
	// without effect on the exit code (fidelity-gate "Verdict exit codes").
	Inv *InvReport `json:"invariants,omitempty"`
}

// Compare judges the d2d runs against the baselines.
//
// Per gated metric: let g = |d2d mean - base mean|, tol = max(rel*|base|, abs), hw = 95 % half width of the d2d mean.
// PASS needs g <= tol and hw <= tol; FAIL needs g - hw > tol (even the interval edge nearest the baseline is outside the
// tolerance); everything else, and any n below MinSeeds, is INCONCLUSIVE with the number of seeds that would settle it.
// The baseline is treated as exact (its own spread is shown as min..max): three original runs are all there is.
func Compare(d2d []Run, b Baselines) Verdict {
	valid, bad := splitValid(d2d)
	v := compare(valid, b)
	return applySample(v, valid, bad, b)
}

// splitValid separates the usable runs from those with a Problem.
func splitValid(rs []Run) (valid []Run, bad []string) {
	for _, r := range rs {
		if p := r.Problem(); p != "" {
			bad = append(bad, fmt.Sprintf("%s: %s", r.Name, p))
		} else {
			valid = append(valid, r)
		}
	}
	return
}

// applySample adds the exclusion warnings and makes the verdict INCONCLUSIVE when runs were excluded or the sample
// is below the minimums of fidelity-gate "Minimum sample" (seeds, hours per seed, total hours); a FAIL stays a FAIL.
func applySample(v Verdict, valid []Run, bad []string, b Baselines) Verdict {
	var why []string
	v.Excluded = append(v.Excluded, bad...)
	v.Seeds = len(valid)
	if len(bad) > 0 {
		why = append(why, "excluded runs")
	}
	var hours float64
	short := 0
	for _, r := range valid {
		hours += r.Hours
		if b.MinHoursPerSeed > 0 && r.Hours < b.MinHoursPerSeed {
			short++
		}
	}
	if short > 0 {
		why = append(why, "seeds under the minimum hours")
	}
	if len(valid) < max(b.MinSeeds, 2) {
		why = append(why, "too few seeds")
	}
	if b.MinTotalHours > 0 && hours < b.MinTotalHours {
		why = append(why, "too few total hours")
	}
	if len(valid) == 0 {
		v.Overall = NoData
		v.Warnings = append(v.Warnings, "NO-DATA: no usable seed, nothing was judged")
		return v
	}
	if len(why) > 0 && v.Overall == Pass {
		v.Overall = Inconclusive
		v.Warnings = append(v.Warnings, "INCONCLUSIVE because of: "+strings.Join(why, ", "))
	}
	return v
}

func compare(d2d []Run, b Baselines) Verdict {
	v := Verdict{Overall: Pass}
	base := Aggregate(runsOf(b))
	got := Aggregate(d2d)
	minSeeds := max(b.MinSeeds, 2)
	var hours float64
	short := 0
	for _, r := range d2d {
		hours += r.Hours
		if b.MinHoursPerSeed > 0 && r.Hours < b.MinHoursPerSeed {
			short++
		}
	}
	if short > 0 {
		v.Warnings = append(v.Warnings, fmt.Sprintf("%d of %d seeds ran under the %.1f h minimum: XP/h of a short run is dominated by the opening (travel, first shop)", short, len(d2d), b.MinHoursPerSeed))
	}
	if len(d2d) < minSeeds {
		v.Warnings = append(v.Warnings, fmt.Sprintf("%d seed(s), minimum %d: no PASS/FAIL can be given, the gaps are indicative only", len(d2d), minSeeds))
	}
	if b.MinTotalHours > 0 && hours < b.MinTotalHours {
		v.Warnings = append(v.Warnings, fmt.Sprintf("%.1f virtual hours in total, minimum %.0f for the death metrics: at about 0.2 deaths/h fewer hours hold under 4 deaths", hours, b.MinTotalHours))
	}
	if len(b.Runs) < 3 {
		v.Warnings = append(v.Warnings, fmt.Sprintf("only %d baseline run(s): the baseline itself is uncertain", len(b.Runs)))
	}
	anyFail, anyInc := false, false
	for _, k := range MetricKeys {
		g, ok := got[k]
		bs, okb := base[k]
		if !ok || !okb {
			continue
		}
		rule := b.Rules[k]
		row := Row{Metric: k, D2D: g, Base: bs, GapPct: math.NaN(), Status: Info}
		if bs.Mean != 0 {
			row.GapPct = (g.Mean - bs.Mean) / math.Abs(bs.Mean) * 100
		}
		row.Tol = math.Max(rule.Rel*math.Abs(bs.Mean), rule.Abs)
		gap := math.Abs(g.Mean - bs.Mean)
		if rule.Gate {
			switch {
			case g.N < minSeeds:
				row.Status, row.Note = Inconclusive, "too few seeds"
			case gap <= row.Tol && g.CIHalf95 <= row.Tol:
				row.Status = Pass
			case gap-g.CIHalf95 > row.Tol:
				row.Status = Fail
			default:
				row.Status = Inconclusive
				row.Note = fmt.Sprintf("95%% interval +-%.3g straddles the tolerance %.3g", g.CIHalf95, row.Tol)
			}
			if row.Status == Inconclusive && g.SD > 0 {
				row.NeedSeed = NeededSeeds(g.SD, row.Tol)
			}
			anyFail = anyFail || row.Status == Fail
			anyInc = anyInc || row.Status == Inconclusive
		} else if gap > row.Tol && row.Tol > 0 {
			row.Note = "outside the info tolerance"
		}
		v.Rows = append(v.Rows, row)
	}
	switch {
	case anyFail:
		v.Overall = Fail
	case anyInc || len(v.Rows) == 0:
		v.Overall = Inconclusive
	}
	return v
}

func runsOf(b Baselines) []Run {
	rs := make([]Run, len(b.Runs))
	for i, r := range b.Runs {
		rs[i] = r.Run
	}
	return rs
}

// Text renders the verdict as a table.
func (v Verdict) Text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%-22s %12s %9s %6s | %12s %-21s | %8s %9s  %s\n", "metric", "d2d mean", "sd", "n", "baseline", "(min .. max)", "gap", "tol", "verdict")
	for _, r := range v.Rows {
		gap := "n/a"
		if !math.IsNaN(r.GapPct) {
			gap = fmt.Sprintf("%+.1f%%", r.GapPct)
		}
		rng := fmt.Sprintf("(%s .. %s)", num(r.Base.Min), num(r.Base.Max))
		fmt.Fprintf(&b, "%-22s %12s %9s %6d | %12s %-21s | %8s %9s  %s", r.Metric, num(r.D2D.Mean), num(r.D2D.SD), r.D2D.N, num(r.Base.Mean), rng, gap, num(r.Tol), r.Status)
		if r.NeedSeed > 0 {
			fmt.Fprintf(&b, " (about %d seeds would settle it at this spread)", r.NeedSeed)
		}
		if r.Note != "" {
			fmt.Fprintf(&b, " [%s]", r.Note)
		}
		b.WriteByte('\n')
	}
	for _, w := range v.Warnings {
		fmt.Fprintf(&b, "WARNING: %s\n", w)
	}
	fmt.Fprintf(&b, "seeds counted: %d\n", v.Seeds)
	if len(v.Excluded) > 0 {
		fmt.Fprintf(&b, "excluded seeds (%d, no exposure hours, no kills, not counted toward the minimum seeds):\n", len(v.Excluded))
		for _, e := range v.Excluded {
			fmt.Fprintf(&b, "  %s\n", e)
		}
	}
	if v.Inv != nil {
		b.WriteString(v.Inv.Text())
	}
	fmt.Fprintf(&b, "OVERALL: %s (exit %d)\n", v.Overall, ExitCode(v.Overall))
	return b.String()
}

func num(x float64) string {
	switch {
	case math.IsInf(x, 0) || math.IsNaN(x):
		return "-"
	case math.Abs(x) >= 100:
		return fmt.Sprintf("%.0f", x)
	}
	return fmt.Sprintf("%.3g", x)
}

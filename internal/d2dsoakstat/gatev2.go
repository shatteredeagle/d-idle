package d2dsoakstat

import (
	"fmt"
	"math"
)

// Gate v2 constants (openspec/changes/gatedelta/specs/fidelity-gate/spec.md).
const (
	V2DeathCapPerHour = 0.15 // absolute cap on the d2d death rate
	V2BlockHours      = 6.0  // one death is allowed per full block of exposure (at least one block is needed)
	V2Alpha           = 0.05 // one-sided level of the Poisson tests
	V2ProxyRel        = 0.25
	V2MinKills        = 200 // fewer kills than this say nothing about deaths per kill (U: floor, about 0.4 h at 500 kills/h)
)

// poissonSF is P(X >= k) for X ~ Poisson(mu).
func poissonSF(k int, mu float64) float64 {
	if k <= 0 {
		return 1
	}
	term, cdf := math.Exp(-mu), 0.0
	for i := 0; i < k; i++ {
		cdf += term
		term *= mu / float64(i+1)
	}
	return math.Max(0, 1-cdf)
}

// poissonLower is the one-sided exact lower confidence bound (level 1-alpha) of the mean of a Poisson count k.
func poissonLower(k int, alpha float64) float64 {
	if k <= 0 {
		return 0
	}
	lo, hi := 0.0, float64(k)+10*math.Sqrt(float64(k))+10
	for range 80 {
		mid := (lo + hi) / 2
		if poissonSF(k, mid) > alpha {
			hi = mid
		} else {
			lo = mid
		}
	}
	return lo
}

// baselineDeathRate pools deaths over exposure of the baseline runs; a zero count uses 0.5 deaths (continuity
// correction) so that the null rate is not exactly 0.
func baselineDeathRate(b Baselines) (perHour, perKill float64) {
	var d, h, k float64
	for _, r := range b.Runs {
		d += float64(r.Deaths)
		h += r.Hours
		k += float64(r.Kills)
	}
	if d == 0 {
		d = 0.5
	}
	if h > 0 {
		perHour = d / h
	}
	if k > 0 {
		perKill = d / k
	}
	return
}

// DeathsV2 judges the pooled death count of the d2d runs. FAIL: more than max(1, floor(H/6)) deaths, or a Poisson
// test against the baseline rate rejects at 5 %, or the exact lower bound of the d2d rate exceeds the 0.15/h cap.
// PASS needs at least one 6 h block of exposure and none of those; less exposure is INCONCLUSIVE.
func DeathsV2(d2d []Run, b Baselines) Row {
	d2d, _ = splitValid(d2d)
	var d int
	var h float64
	for _, r := range d2d {
		d += r.Deaths
		h += r.Hours
	}
	row := Row{Metric: "deaths_per_hour", GapPct: math.NaN(), Tol: V2DeathCapPerHour, Status: Pass}
	row.D2D = Stat{N: len(d2d), Mean: safeDiv(float64(d), h)}
	if bs := Aggregate(runsOf(b))["deaths_per_hour"]; bs.N > 0 {
		row.Base = bs
	}
	rate0, _ := baselineDeathRate(b)
	if math.IsNaN(h) || math.IsInf(h, 0) || rate0 <= 0 {
		row.Status, row.Note = Inconclusive, "no usable exposure or baseline rate"
		return row
	}
	allowed := max(1, int(h/V2BlockHours))
	p := poissonSF(d, rate0*h)
	lb := poissonLower(d, V2Alpha)
	switch {
	case h <= 0:
		row.Status, row.Note = Inconclusive, "no exposure"
	case d > allowed:
		row.Status, row.Note = Fail, fmt.Sprintf("%d deaths in %.1f h, budget %d", d, h, allowed)
	case p < V2Alpha:
		row.Status, row.Note = Fail, fmt.Sprintf("Poisson p=%.3f against the baseline %.3g/h", p, rate0)
	case h > 0 && lb/h > V2DeathCapPerHour:
		row.Status, row.Note = Fail, fmt.Sprintf("lower 95%% bound %.3g/h above the cap", lb/h)
	case h < V2BlockHours:
		row.Status, row.Note = Inconclusive, fmt.Sprintf("%.1f h of exposure, need %.0f", h, V2BlockHours)
	default:
		row.Note = fmt.Sprintf("%d deaths in %.1f h (budget %d, Poisson p=%.3f)", d, h, allowed, p)
		if row.D2D.Mean > V2DeathCapPerHour {
			row.Note += "; point rate above the cap, more hours would settle it"
		}
	}
	return row
}

// DeathsPerKillV2 is the Poisson form of deaths per 1000 kills, with kills as the exposure.
func DeathsPerKillV2(d2d []Run, b Baselines) Row {
	d2d, _ = splitValid(d2d)
	var d, k int
	for _, r := range d2d {
		d += r.Deaths
		k += r.Kills
	}
	row := Row{Metric: "deaths_per_1000_kills", GapPct: math.NaN(), Status: Pass}
	row.D2D = Stat{N: len(d2d), Mean: 1000 * safeDiv(float64(d), float64(k))}
	if bs := Aggregate(runsOf(b))["deaths_per_1000_kills"]; bs.N > 0 {
		row.Base = bs
	}
	_, perKill := baselineDeathRate(b)
	row.Tol = 1000 * perKill
	if k < V2MinKills {
		row.Status, row.Note = Inconclusive, fmt.Sprintf("%d kills, need %d", k, V2MinKills)
		return row
	}
	if perKill <= 0 {
		row.Status, row.Note = Inconclusive, "baseline has no kill exposure"
		return row
	}
	if p := poissonSF(d, perKill*float64(k)); p < V2Alpha {
		row.Status, row.Note = Fail, fmt.Sprintf("Poisson p=%.3f against the baseline %.3g/1000 kills", p, 1000*perKill)
	}
	return row
}

func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

// RulesV2 turns the v1 rules into the v2 ones: XP/h and kills/h stay at 20 %, the death metrics leave the
// relative form (they get the Poisson rows), the three proxies are gated at 25 %.
func RulesV2(r map[string]Rule) map[string]Rule {
	out := make(map[string]Rule, len(r))
	for k, v := range r {
		out[k] = v
	}
	for _, k := range []string{"deaths_per_hour", "deaths_per_1000_kills"} {
		out[k] = Rule{Why: "v2: Poisson bound row"}
	}
	for _, k := range []string{"potions_per_100_kills", "min_life_fraction", "damage_per_kill"} {
		out[k] = Rule{Gate: true, Rel: V2ProxyRel, Why: "v2 proxy"}
	}
	return out
}

// CompareV2 is Compare with the v2 gate (--gate v2). Proxies that the d2d runs or the baselines do not carry give
// an INCONCLUSIVE row with a note, so the overall verdict is never PASS on data that does not exist yet.
func CompareV2(d2d []Run, b Baselines) Verdict {
	b2 := b
	b2.Rules = RulesV2(b.Rules)
	if len(b2.Rules) == 0 {
		b2.Rules = RulesV2(DefaultRules())
	}
	valid, bad := splitValid(d2d)
	v := compare(valid, b2)
	d2d = valid
	have := map[string]int{}
	for i, r := range v.Rows {
		have[r.Metric] = i
	}
	for _, r := range []Row{DeathsV2(d2d, b), DeathsPerKillV2(d2d, b)} {
		if i, ok := have[r.Metric]; ok {
			v.Rows[i] = r
		} else {
			v.Rows = append(v.Rows, r)
		}
	}
	for _, k := range []string{"potions_per_100_kills", "min_life_fraction", "damage_per_kill"} {
		if _, ok := have[k]; !ok {
			v.Rows = append(v.Rows, Row{Metric: k, GapPct: math.NaN(), Status: Inconclusive, Note: "not in the d2d runs or the baselines (U: minute line lacks it)"})
		}
	}
	v.Overall = Pass
	for _, r := range v.Rows {
		switch r.Status {
		case Fail:
			v.Overall = Fail
		case Inconclusive:
			if v.Overall != Fail {
				v.Overall = Inconclusive
			}
		}
	}
	return applySample(v, valid, bad, b)
}

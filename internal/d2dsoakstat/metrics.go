package d2dsoakstat

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Metric keys, in report order. The first group is what d2soak prints; the rest are time-in-state and level checkpoints.
var MetricKeys = []string{
	"xp_per_hour", "kills_per_hour", "deaths_per_hour", "deaths_per_1000_kills", "potions_per_hour", "gold_per_hour",
	"potions_per_100_kills", "min_life_fraction", "damage_per_kill",
	"recovery_share", "farming_share", "town_share", "repeat_sites", "clvl_gain", "clvl_1h", "clvl_2h",
}

// Unit text for the report.
var MetricUnit = map[string]string{
	"xp_per_hour": "XP/h", "kills_per_hour": "kills/h", "deaths_per_hour": "deaths/h", "deaths_per_1000_kills": "deaths/1000 kills",
	"potions_per_hour": "potions/h", "potions_per_100_kills": "potions/100 kills", "min_life_fraction": "life fraction", "damage_per_kill": "life/kill", "gold_per_hour": "gold/h (net)", "recovery_share": "share of time", "farming_share": "share of time",
	"town_share": "share of time", "repeat_sites": "count", "clvl_gain": "levels", "clvl_1h": "clvl", "clvl_2h": "clvl",
}

// Run is the metric set of one run (one seed on d2d, or one original-game soak).
type Run struct {
	Name    string             `json:"name"`
	Hours   float64            `json:"hours"`
	Minutes int                `json:"minutes"`
	Deaths  int                `json:"deaths"`
	Kills   int                `json:"kills"`
	XP      int64              `json:"xp"`
	Metrics map[string]float64 `json:"metrics"`
	// States is the minute-count share per bot activity group (state field); empty in old logs.
	States map[string]float64 `json:"states,omitempty"`
	// Invalid is why the run must not count as exposure (crashed or short seed, unreadable lines); the verdict
	// excludes it and cannot be PASS. Dropped is the number of minute lines that failed to parse.
	Invalid string `json:"invalid,omitempty"`
	Dropped int    `json:"dropped,omitempty"`
	// Seed is the run's seed (0 = unknown: SeedOf then reads it from the name). Inv is the invariant counter block
	// (nil when the run was loaded without its bot log); Cover is what the run's setup says about class and difficulty.
	Seed  int         `json:"seed,omitempty"`
	Inv   *Invariants `json:"invariants,omitempty"`
	Cover *Cover      `json:"cover,omitempty"`
}

// Cover is the setup axes of a d2d run that the log does not carry.
type Cover struct {
	Class      string `json:"class"`
	Difficulty string `json:"difficulty"`
}

// MaxDropFrac is the share of unparseable minute lines above which a run is invalid.
const MaxDropFrac = 0.02

// Problem returns why the run cannot be used as sample, or "".
func (r Run) Problem() string {
	switch {
	case r.Invalid != "":
		return r.Invalid
	case math.IsNaN(r.Hours) || math.IsInf(r.Hours, 0) || r.Hours <= 0:
		return fmt.Sprintf("hours %v", r.Hours)
	case r.Deaths < 0 || r.Kills < 0:
		return "negative counters"
	}
	return ""
}

// NoteDropped records n unparseable minute lines and invalidates the run when they are more than MaxDropFrac.
func (r *Run) NoteDropped(n int) {
	r.Dropped = n
	if n > 0 && r.Invalid == "" && float64(n) > MaxDropFrac*float64(n+r.Minutes) {
		r.Invalid = fmt.Sprintf("%d of %d minute lines unparseable", n, n+r.Minutes)
	}
}

// maxJump caps the per-minute counter increment that repeatSites expands (a garbage counter must not allocate).
const maxJump = 1000

// Options of Extract.
type Options struct {
	// Hours fixes the run length (d2d: the virtual clock, read from d2d). 0 derives it from the minute timestamps
	// (original game: real time). When fixed, the lines are taken as evenly spaced over it.
	Hours float64
}

// Extract computes the Run metrics from minute lines. Definitions (same as docs/d2d-soak.md, V against the d2soak
// report numbers within a few percent, see the tests):
//   - xp/h: net character XP (last - first) over the hours, so unsaved progress lost to a restart counts against it;
//   - kills, deaths, potions: sum of positive increments of the running counters (they reset when the bot restarts);
//   - gold/h: net gold change (spending on shop trips counts, so it is a balance, not an income);
//   - recovery share: minutes whose state is "recovery" or whose runner state mentions recover;
//   - repeat sites: deaths in the same area within 30 min of an earlier one, or within 3 min of the previous death (the
//     d2soak death rule, approximated at minute resolution; the area is the one of the minute before the death).
func Extract(name string, ms []Minute, o Options) Run {
	r := Run{Name: name, Minutes: len(ms), Metrics: map[string]float64{}}
	if len(ms) < 2 {
		r.Invalid = fmt.Sprintf("%d minute line(s), need 2", len(ms))
		return r
	}
	first, last := ms[0], ms[len(ms)-1]
	elapsed := make([]float64, len(ms)) // hours since the first line
	switch {
	case o.Hours > 0:
		for i := range ms {
			elapsed[i] = o.Hours * float64(i) / float64(len(ms)-1)
		}
		r.Hours = o.Hours
	default:
		for i, m := range ms {
			elapsed[i] = m.At.Sub(first.At).Hours()
		}
		r.Hours = last.At.Sub(first.At).Hours()
	}
	if r.Hours <= 0 || math.IsNaN(r.Hours) || math.IsInf(r.Hours, 0) {
		r.Invalid = fmt.Sprintf("hours %v", r.Hours)
		r.Hours = 0
		return r
	}
	inc := func(f func(Minute) int) int {
		n := 0
		for i := 1; i < len(ms); i++ {
			if d := f(ms[i]) - f(ms[i-1]); d > 0 {
				n += d
			}
		}
		return n
	}
	r.Kills = inc(func(m Minute) int { return m.Kills })
	r.Deaths = inc(func(m Minute) int { return m.Deaths })
	potions := inc(func(m Minute) int { return m.PotionsUsed })
	r.XP = int64(last.XP) - int64(first.XP)
	h := r.Hours
	r.Metrics["xp_per_hour"] = float64(r.XP) / h
	r.Metrics["kills_per_hour"] = float64(r.Kills) / h
	r.Metrics["deaths_per_hour"] = float64(r.Deaths) / h
	if r.Kills > 0 {
		r.Metrics["deaths_per_1000_kills"] = 1000 * float64(r.Deaths) / float64(r.Kills)
	}
	r.Metrics["potions_per_hour"] = float64(potions) / h
	if r.Kills > 0 {
		r.Metrics["potions_per_100_kills"] = 100 * float64(potions) / float64(r.Kills)
	}
	proxyHP(&r, ms)
	r.Metrics["gold_per_hour"] = float64(last.Gold-first.Gold) / h
	r.Metrics["clvl_gain"] = float64(last.Clvl - first.Clvl)
	for _, cp := range []struct {
		key string
		at  float64
	}{{"clvl_1h", 1}, {"clvl_2h", 2}} {
		if h >= cp.at {
			c := first.Clvl
			for i, m := range ms {
				if elapsed[i] <= cp.at {
					c = m.Clvl
				}
			}
			r.Metrics[cp.key] = float64(c)
		}
	}

	groups, recov := map[string]int{}, 0
	for _, m := range ms {
		if m.State != "" {
			groups[m.State]++
		}
		if m.State == "recovery" || strings.Contains(m.RState, "recover") {
			recov++
		}
	}
	if len(groups) > 0 {
		r.States = map[string]float64{}
		for k, n := range groups {
			r.States[k] = float64(n) / float64(len(ms))
		}
		r.Metrics["recovery_share"] = float64(recov) / float64(len(ms))
		r.Metrics["farming_share"] = r.States["farming"]
		r.Metrics["town_share"] = r.States["town chores"]
	}
	r.Metrics["repeat_sites"] = float64(repeatSites(ms, elapsed))
	return r
}

type death struct {
	area int
	at   float64
}

func repeatSites(ms []Minute, elapsed []float64) int {
	var ds []death
	for i := 1; i < len(ms); i++ {
		for k := ms[i-1].Deaths; k < ms[i].Deaths && k < ms[i-1].Deaths+maxJump; k++ {
			ds = append(ds, death{ms[i-1].Area, elapsed[i]})
		}
	}
	n := 0
	for i, d := range ds {
		rep := false
		for _, p := range ds[:i] {
			if d.area != 0 && p.area == d.area && d.at-p.at <= 0.5 {
				rep = true
			}
		}
		if i > 0 && d.at-ds[i-1].at <= 3.0/60 {
			rep = true
		}
		if rep {
			n++
		}
	}
	return n
}

// Stat is mean and spread of one metric over runs.
type Stat struct {
	N        int     `json:"n"`
	Mean     float64 `json:"mean"`
	SD       float64 `json:"sd"` // sample standard deviation, 0 for n < 2
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	CIHalf95 float64 `json:"ci95"` // half width of the 95 % interval of the mean (Student t); +Inf for n < 2
}

// Aggregate summarises every metric over runs (runs missing a metric are skipped for it).
func Aggregate(runs []Run) map[string]Stat {
	out := map[string]Stat{}
	for _, k := range MetricKeys {
		var v []float64
		for _, r := range runs {
			if x, ok := r.Metrics[k]; ok {
				v = append(v, x)
			}
		}
		if len(v) > 0 {
			out[k] = stat(v)
		}
	}
	return out
}

func stat(v []float64) Stat {
	s := Stat{N: len(v), Min: v[0], Max: v[0], CIHalf95: math.Inf(1)}
	sum := 0.0
	for _, x := range v {
		sum += x
		s.Min, s.Max = math.Min(s.Min, x), math.Max(s.Max, x)
	}
	s.Mean = sum / float64(len(v))
	if len(v) > 1 {
		ss := 0.0
		for _, x := range v {
			ss += (x - s.Mean) * (x - s.Mean)
		}
		s.SD = math.Sqrt(ss / float64(len(v)-1))
		s.CIHalf95 = tCrit(len(v)-1) * s.SD / math.Sqrt(float64(len(v)))
	}
	return s
}

// tCrit is the two-sided 95 % Student t critical value for df degrees of freedom.
func tCrit(df int) float64 {
	tab := []float64{0, 12.706, 4.303, 3.182, 2.776, 2.571, 2.447, 2.365, 2.306, 2.262, 2.228}
	switch {
	case df < len(tab):
		return tab[df]
	case df <= 15:
		return 2.131
	case df <= 20:
		return 2.086
	case df <= 30:
		return 2.042
	}
	return 1.96
}

// NeededSeeds is the number of runs after which the 95 % interval of the mean is no wider than tol, given a per-run
// standard deviation sd (normal approximation, so a lower bound for small n).
func NeededSeeds(sd, tol float64) int {
	if tol <= 0 {
		return 0
	}
	n := int(math.Ceil(math.Pow(1.96*sd/tol, 2)))
	return max(n, 2)
}

// SortedStates returns the state names of a run, biggest share first (for printing).
func SortedStates(r Run) []string {
	var ks []string
	for k := range r.States {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool {
		if r.States[ks[i]] != r.States[ks[j]] {
			return r.States[ks[i]] > r.States[ks[j]]
		}
		return ks[i] < ks[j]
	})
	return ks
}

// proxyHP fills the life proxies of docs/d2d-fidelity.md (gate v2) when the log carries them (U, see Minute.HasHP):
//   - damage_per_kill: increments of the dmgTaken counter over the kills of the run;
//   - min_life_fraction: mean over the minutes that saw at least one kill of that minute's lowest life percent / 100
//     (a minute is the finest "fight" the log resolves; a per-fight value needs a bot-side event, still U).
func proxyHP(r *Run, ms []Minute) {
	if !ms[0].HasHP {
		return
	}
	dmg, sum, n := 0, 0.0, 0
	for i := 1; i < len(ms); i++ {
		if d := ms[i].DmgTaken - ms[i-1].DmgTaken; d > 0 {
			dmg += d
		}
		if ms[i].HasHP && ms[i].Kills > ms[i-1].Kills {
			sum += float64(ms[i].HPMinPct) / 100
			n++
		}
	}
	if r.Kills > 0 {
		r.Metrics["damage_per_kill"] = float64(dmg) / float64(r.Kills)
	}
	if n > 0 {
		r.Metrics["min_life_fraction"] = sum / float64(n)
	}
}

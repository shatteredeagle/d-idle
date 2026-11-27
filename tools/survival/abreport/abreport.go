// Package main is the report generator of the survival A/B (docs/survival/ab-harness.md): it reads the bot logs of two arms
// (DIR/<arm>/<seed>.log), pairs them by seed and prints the metrics of the S-D gate (deaths, chickens, XP at 30/60/90/120
// virtual minutes, intake per zone, potions, gold) and the verdict. Stdlib only.
package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"d2lod-bot/internal/d2dsoakstat"
)

// Checkpoints are the virtual minutes at which XP is compared.
var Checkpoints = []int{30, 60, 90, 120}

// Run is the record of one log (one arm, one seed). Minutes are indexed by the count of minute lines, because the log clock
// is wall time under lockstep (docs/ab-design.md).
type Run struct {
	Arm        string
	Seed       int
	Minutes    int // elapsed virtual minutes (minute lines minus one)
	Deaths     int // from the deaths counter of the last minute line minus the first
	Chickens   int
	MercDeaths int
	Potions    int
	Gold       int
	ClvlStart  int
	ClvlEnd    int
	XPAt       map[int]float64 // checkpoint minute -> xp gained since the first minute line (absent: run shorter)
	XPEnd      float64         // xp gained at the last minute line (minute Minutes)
	XPLine     []float64       // xp gained at every minute line, indexed by minute
	Intake     map[int]*Intake // area id -> life lost and minutes (only minutes that carry the intake keys)
	HasIntake  bool
	// Hero is the prep's hero line ("hero: run=... profile=... selfcheck=..."), "" for a log made before the hero library.
	Hero string
}

// HeroProblem says why the run's hero is no survival evidence ("" when it is, or when the log has no hero line): a
// failed self-check, or a banked hero (every point unspent, featuresoak pass 1).
func (r Run) HeroProblem() string {
	if r.Hero == "" {
		return ""
	}
	for _, f := range strings.Fields(r.Hero) {
		switch {
		case strings.HasPrefix(f, "selfcheck=") && f != "selfcheck=ok":
			return "hero " + f
		case f == "profile=banked":
			return "hero has every point banked"
		}
	}
	return ""
}

// Intake accumulates the observed life loss of one zone.
type Intake struct {
	Life    int
	Minutes int
}

// PerMin is life lost per minute spent.
func (i Intake) PerMin() float64 {
	if i.Minutes == 0 {
		return 0
	}
	return float64(i.Life) / float64(i.Minutes)
}

// Analyse builds a Run from the lines of one bot log. ok is false with fewer than 2 minute lines.
func Analyse(arm string, seed int, lines []string, mins []d2dsoakstat.Minute) (Run, bool) {
	r := Run{Arm: arm, Seed: seed, XPAt: map[int]float64{}, Intake: map[int]*Intake{}}
	if len(mins) < 2 {
		return r, false
	}
	f, l := mins[0], mins[len(mins)-1]
	r.Minutes = len(mins) - 1
	r.Deaths = l.Deaths - f.Deaths
	r.Potions = l.PotionsUsed - f.PotionsUsed
	r.Gold = l.Gold + l.StashGold - f.Gold - f.StashGold
	r.ClvlStart, r.ClvlEnd = f.Clvl, l.Clvl
	r.XPEnd = float64(l.XP) - float64(f.XP)
	for _, m := range mins {
		r.XPLine = append(r.XPLine, float64(m.XP)-float64(f.XP))
	}
	for _, c := range Checkpoints {
		if c <= r.Minutes {
			r.XPAt[c] = float64(mins[c].XP) - float64(f.XP)
		}
	}
	for _, m := range mins[1:] {
		if !m.HasIntake {
			continue
		}
		r.HasIntake = true
		in := r.Intake[m.Area]
		if in == nil {
			in = &Intake{}
			r.Intake[m.Area] = in
		}
		in.Life += m.LifeLost
		in.Minutes++
	}
	for _, ln := range lines {
		switch {
		case r.Hero == "" && strings.HasPrefix(ln, "hero: "):
			r.Hero = strings.TrimSpace(ln)
		case isChicken(ln):
			r.Chickens++
		case strings.Contains(ln, "merc: died"):
			r.MercDeaths++
		}
	}
	return r, true
}

// isChicken matches the `safety decision` line of a carried-out chicken (the same test as tools/levelsim/shadow.go).
func isChicken(ln string) bool {
	if !strings.Contains(ln, "safety decision") {
		return false
	}
	return strings.Contains(ln, "action=chicken") || strings.Contains(ln, `"action":"chicken"`)
}

// Pair is one seed with both arms complete.
type Pair struct {
	Seed       int
	Base, Test Run
}

// Summary is the whole comparison.
type Summary struct {
	BaseArm, TestArm string
	Horizon          int
	Pairs            []Pair
	Excluded         []string // "seed N: reason"
	Verdict          string
	Reasons          []string
	XPDiff           Stat // paired XP at the horizon, test - base
	XPRatio          float64
}

// Stat is a paired mean with its 95 percent Student-t interval.
type Stat struct {
	N          int
	Mean, SD   float64
	Lo, Hi     float64
	IntervalOK bool // false below 2 pairs
}

var tCrit = [...]float64{0, 0, 12.706, 4.303, 3.182, 2.776, 2.571, 2.447, 2.365, 2.306, 2.262, 2.228, 2.201, 2.179, 2.160, 2.145}

// PairedStat is the paired mean of d with a two-sided 95 percent interval (t table up to 15 degrees, then 1.96).
func PairedStat(d []float64) Stat {
	s := Stat{N: len(d)}
	if len(d) == 0 {
		return s
	}
	for _, v := range d {
		s.Mean += v
	}
	s.Mean /= float64(len(d))
	if len(d) < 2 {
		return s
	}
	var ss float64
	for _, v := range d {
		ss += (v - s.Mean) * (v - s.Mean)
	}
	s.SD = math.Sqrt(ss / float64(len(d)-1))
	t := 1.96
	if len(d) < len(tCrit) {
		t = tCrit[len(d)]
	}
	h := t * s.SD / math.Sqrt(float64(len(d)))
	s.Lo, s.Hi, s.IntervalOK = s.Mean-h, s.Mean+h, true
	return s
}

// Options of Compare.
type Options struct {
	Horizon int     // virtual minutes each run must cover (95 percent of it)
	XPFloor float64 // the test arm's XP at the horizon must reach this share of the base arm's (1.0 = at least equal)
	// DeathSlack and ChickenSlack are the pooled extra deaths or chickens the test arm may have (0 = the gate as worded: no more).
	// Two equal engines fail the strict reading about one time in three at 6 virtual hours per arm (Poisson), so a slack is the
	// user's decision (docs/survival/ab-harness.md).
	DeathSlack, ChickenSlack int
	MinSeeds                 int // pairs below this are INCONCLUSIVE when nothing failed (the gate says 3; docs/ab-design.md says 8 for a claim)
}

// Compare pairs the runs by seed and applies the S-D gate: the margin arm has no more deaths and no more chickens (pooled over
// seeds) and at least XPFloor of the base XP at the horizon. A seed missing or short in either arm is dropped from both. The
// verdict is FAIL as soon as deaths or chickens are higher or the XP interval lies wholly below the floor, PASS only when
// nothing failed and the pairs reach MinSeeds, otherwise INCONCLUSIVE.
func Compare(base, test []Run, o Options) Summary {
	s := Summary{BaseArm: "base", TestArm: "test", Horizon: o.Horizon}
	if len(base) > 0 {
		s.BaseArm = base[0].Arm
	}
	if len(test) > 0 {
		s.TestArm = test[0].Arm
	}
	bs, ts := map[int]Run{}, map[int]Run{}
	for _, r := range base {
		bs[r.Seed] = r
	}
	for _, r := range test {
		ts[r.Seed] = r
	}
	seeds := map[int]bool{}
	for k := range bs {
		seeds[k] = true
	}
	for k := range ts {
		seeds[k] = true
	}
	var order []int
	for k := range seeds {
		order = append(order, k)
	}
	sort.Ints(order)
	need := int(math.Ceil(0.95 * float64(o.Horizon)))
	for _, k := range order {
		b, okb := bs[k]
		t, okt := ts[k]
		switch {
		case !okb || !okt:
			s.Excluded = append(s.Excluded, fmt.Sprintf("seed %d: missing in one arm", k))
		case b.Minutes < need || t.Minutes < need:
			s.Excluded = append(s.Excluded, fmt.Sprintf("seed %d: short (%d and %d of %d min)", k, b.Minutes, t.Minutes, o.Horizon))
		case b.HeroProblem() != "" || t.HeroProblem() != "":
			s.Excluded = append(s.Excluded, fmt.Sprintf("seed %d: %s", k, strings.TrimSpace(b.HeroProblem()+" "+t.HeroProblem())))
		default:
			s.Pairs = append(s.Pairs, Pair{Seed: k, Base: b, Test: t})
		}
	}
	var diffs []float64
	var bxp, txp float64
	var bd, td, bc, tc int
	for _, p := range s.Pairs {
		bd, td = bd+p.Base.Deaths, td+p.Test.Deaths
		bc, tc = bc+p.Base.Chickens, tc+p.Test.Chickens
		x0, x1 := xpAtHorizon(p.Base, o.Horizon), xpAtHorizon(p.Test, o.Horizon)
		if hm := PairMinute(p.Base, p.Test, o.Horizon); hm < o.Horizon && hm < len(p.Base.XPLine) && hm < len(p.Test.XPLine) {
			x0, x1 = p.Base.XPLine[hm], p.Test.XPLine[hm] // one minute for both arms: the runs end a few minutes apart
		}
		bxp, txp = bxp+x0, txp+x1
		diffs = append(diffs, x1-x0)
	}
	s.XPDiff = PairedStat(diffs)
	if bxp > 0 {
		s.XPRatio = txp / bxp
	}
	if len(s.Pairs) == 0 {
		s.Verdict = "INCONCLUSIVE"
		s.Reasons = append(s.Reasons, "no complete pair")
		return s
	}
	fail := false
	if td > bd+o.DeathSlack {
		fail = true
		s.Reasons = append(s.Reasons, fmt.Sprintf("deaths %d > %d (+%d slack)", td, bd, o.DeathSlack))
	}
	if tc > bc+o.ChickenSlack {
		fail = true
		s.Reasons = append(s.Reasons, fmt.Sprintf("chickens %d > %d (+%d slack)", tc, bc, o.ChickenSlack))
	}
	floorDiff := (o.XPFloor - 1) * bxp / float64(len(s.Pairs)) // per-seed XP shortfall the floor allows
	switch {
	case s.XPDiff.IntervalOK && s.XPDiff.Hi < floorDiff:
		fail = true
		s.Reasons = append(s.Reasons, fmt.Sprintf("XP at %d min is below the floor with the 95 percent interval (ratio %.3f)", o.Horizon, s.XPRatio))
	case s.XPRatio < o.XPFloor:
		s.Reasons = append(s.Reasons, fmt.Sprintf("XP ratio %.3f under the floor %.2f but the interval spans it", s.XPRatio, o.XPFloor))
	}
	switch {
	case fail:
		s.Verdict = "FAIL"
	case len(s.Pairs) < o.MinSeeds:
		s.Verdict = "INCONCLUSIVE"
		s.Reasons = append(s.Reasons, fmt.Sprintf("%d pairs under the minimum %d", len(s.Pairs), o.MinSeeds))
	case s.XPRatio < o.XPFloor:
		s.Verdict = "INCONCLUSIVE"
	default:
		s.Verdict = "PASS"
	}
	return s
}

// xpAtHorizon is the XP at minute h. A 2 h run has at most 119 intervals, so a run that covers 95 percent of the horizon
// (the pairing rule) but lacks minute h reports its last minute line instead; xpHorizonMinute says which minute that was.
func xpAtHorizon(r Run, h int) float64 {
	if v, ok := r.XPAt[h]; ok {
		return v
	}
	if r.Minutes >= int(math.Ceil(0.95*float64(h))) {
		return r.XPEnd
	}
	best, bm := 0.0, -1
	for m, v := range r.XPAt {
		if m > bm && m <= h {
			best, bm = v, m
		}
	}
	return best
}

// PairMinute is the minute both arms of a pair reach: the horizon, or the shorter run's last minute line.
func PairMinute(a, b Run, h int) int { return min(h, a.Minutes, b.Minutes) }

// xpHorizonMinute is the minute whose XP xpAtHorizon returned.
func xpHorizonMinute(r Run, h int) int {
	if _, ok := r.XPAt[h]; ok || r.Minutes < int(math.Ceil(0.95*float64(h))) {
		return min(h, r.Minutes)
	}
	return r.Minutes
}

// Format is the text report.
func (s Summary) Format() string {
	var b strings.Builder
	fmt.Fprintf(&b, "survival A/B: base=%s test=%s horizon=%d min pairs=%d\n", s.BaseArm, s.TestArm, s.Horizon, len(s.Pairs))
	for _, e := range s.Excluded {
		fmt.Fprintf(&b, "excluded %s\n", e)
	}
	fmt.Fprintf(&b, "%-5s %-8s %6s %6s %6s %6s %6s %6s %6s %7s %7s\n", "seed", "arm", "clvl", "death", "chick", "mdeath", "potion", "gold", "xp30", "xp60", "xp90")
	for _, p := range s.Pairs {
		for _, r := range []Run{p.Base, p.Test} {
			fmt.Fprintf(&b, "%-5d %-8s %2d-%-3d %6d %6d %6d %6d %6d %6.0f %7.0f %7.0f  xp@%d=%.0f\n", r.Seed, r.Arm, r.ClvlStart, r.ClvlEnd, r.Deaths, r.Chickens,
				r.MercDeaths, r.Potions, r.Gold, r.XPAt[30], r.XPAt[60], r.XPAt[90], xpHorizonMinute(r, s.Horizon), xpAtHorizon(r, s.Horizon))
		}
	}
	fmt.Fprintf(&b, "XP at the horizon (minute %d, or the shorter run's last minute line of a pair), test-base per seed: mean %.0f sd %.0f", s.Horizon, s.XPDiff.Mean, s.XPDiff.SD)
	if s.XPDiff.IntervalOK {
		fmt.Fprintf(&b, " 95%% [%.0f, %.0f]", s.XPDiff.Lo, s.XPDiff.Hi)
	}
	fmt.Fprintf(&b, " ratio %.3f\n", s.XPRatio)
	b.WriteString(s.intakeTable())
	fmt.Fprintf(&b, "verdict: %s", s.Verdict)
	if len(s.Reasons) > 0 {
		fmt.Fprintf(&b, " (%s)", strings.Join(s.Reasons, "; "))
	}
	b.WriteString("\n")
	return b.String()
}

// intakeTable is life lost per minute per area and arm, pooled over the pairs (areas with at least one minute in either arm).
func (s Summary) intakeTable() string {
	pool := func(pick func(Pair) Run) map[int]*Intake {
		m := map[int]*Intake{}
		for _, p := range s.Pairs {
			for a, in := range pick(p).Intake {
				t := m[a]
				if t == nil {
					t = &Intake{}
					m[a] = t
				}
				t.Life += in.Life
				t.Minutes += in.Minutes
			}
		}
		return m
	}
	bm, tm := pool(func(p Pair) Run { return p.Base }), pool(func(p Pair) Run { return p.Test })
	if len(bm) == 0 && len(tm) == 0 {
		return "intake per zone: no minute line carries lifeLost (old bot)\n"
	}
	seen := map[int]bool{}
	var areas []int
	for a := range bm {
		seen[a] = true
	}
	for a := range tm {
		seen[a] = true
	}
	for a := range seen {
		areas = append(areas, a)
	}
	sort.Ints(areas)
	var b strings.Builder
	fmt.Fprintf(&b, "intake per zone (life lost per minute, minutes in brackets; a lower bound, docs/survival/ledger.md)\n%-6s %-16s %-16s\n", "area", s.BaseArm, s.TestArm)
	cell := func(m map[int]*Intake, a int) string {
		if in, ok := m[a]; ok {
			return fmt.Sprintf("%.1f (%d)", in.PerMin(), in.Minutes)
		}
		return "-"
	}
	for _, a := range areas {
		fmt.Fprintf(&b, "%-6d %-16s %-16s\n", a, cell(bm, a), cell(tm, a))
	}
	return b.String()
}

// Load reads DIR/<arm>/<seed>.log for one arm; files that do not parse as <seed>.log are skipped with a note.
func Load(dir, arm string) ([]Run, []string, error) {
	files, err := filepath.Glob(filepath.Join(dir, arm, "*.log"))
	if err != nil {
		return nil, nil, err
	}
	sort.Strings(files)
	var runs []Run
	var notes []string
	for _, f := range files {
		seed, err := strconv.Atoi(strings.TrimSuffix(filepath.Base(f), ".log"))
		if err != nil {
			notes = append(notes, "skipped "+f+": name is not <seed>.log")
			continue
		}
		raw, err := os.ReadFile(f)
		if err != nil {
			return nil, nil, err
		}
		mins, err := d2dsoakstat.ParseMinutes(strings.NewReader(string(raw)))
		if err != nil {
			return nil, nil, err
		}
		r, ok := Analyse(arm, seed, strings.Split(string(raw), "\n"), mins)
		if !ok {
			notes = append(notes, fmt.Sprintf("seed %d (%s): fewer than 2 minute lines", seed, arm))
			continue
		}
		runs = append(runs, r)
	}
	return runs, notes, nil
}

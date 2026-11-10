package readiness

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
)

// ResistBasis says how the hero's resist numbers were read. Whether d2.Player.Resist already holds the difficulty penalty
// is OPEN (TASK-216, TODO in internal/combat/hard.go), so the caller states it; the evaluator never guesses.
type ResistBasis int

const (
	// ResistRaw: the numbers are the gear and skill total before the difficulty penalty (the penalty is applied here).
	ResistRaw ResistBasis = iota
	// ResistWithPenalty: the numbers are what the character sheet shows, penalty included.
	ResistWithPenalty
)

// DefaultResistBasis is what today's code assumes (combat.PlayerResists.Effective subtracts the penalty). U until TASK-216.
const DefaultResistBasis = ResistRaw

// Resists are the four elemental resists the checks read.
type Resists struct{ Fire, Cold, Lightning, Poison int }

// Hero is the read-only snapshot the checks use. Zero means "not measured" for Defence, DamagePerSec and Repair so a
// missing number never turns into a verdict: the check is skipped and says so.
type Hero struct {
	Class      string
	Level      int
	Difficulty string // normal, nightmare, hell
	Life       int
	MaxLife    int
	Defence    int
	Resist     Resists
	Basis      ResistBasis
	// DamagePerSec is the hero's sustained damage output; 0 = unknown.
	DamagePerSec float64
	Potions      int
	// RepairPct is the lowest durability of the worn gear in percent; negative = unknown.
	RepairPct int
}

// States of one check and the light of the report.
const (
	Pass = "pass"
	Warn = "warn"
	Fail = "fail"
	Skip = "skip" // the input was not measured; not counted in the roll-up

	Green = "green"
	Amber = "amber"
	Red   = "red"
)

// Result is one check line.
type Result struct {
	ID     string `json:"id"`
	Threat string `json:"threat"`
	State  string `json:"state"`
	Reason string `json:"reason"`
}

// Report is the answer for one hero and one area.
type Report struct {
	Schema     string   `json:"schema"`
	Class      string   `json:"class"`
	Difficulty string   `json:"difficulty"`
	Area       string   `json:"area"`
	MLvl       int      `json:"mlvl"`
	Light      string   `json:"light"`
	Checks     []Result `json:"checks"`
	Reasons    []string `json:"reasons"` // the warn and fail lines, fails first
}

// ReportSchema tags the /api/readiness body.
const ReportSchema = "readiness.report/1"

type params map[string]float64

type checkRow struct {
	ID     string                       `json:"id"`
	Kind   string                       `json:"kind"`
	Threat string                       `json:"threat"`
	Diff   map[string]params            `json:"diff"`
	Class  map[string]map[string]params `json:"class,omitempty"`
}

type classRow struct {
	Damage []string `json:"damage"`
}

// CheckTable is checks.json.
type CheckTable struct {
	Schema  string              `json:"schema"`
	Doc     string              `json:"doc,omitempty"`
	Classes map[string]classRow `json:"classes"`
	Checks  []checkRow          `json:"checks"`
}

//go:embed checks.json
var shippedChecks []byte

// ShippedChecks parses the embedded table.
func ShippedChecks() (*CheckTable, error) { return ParseChecks(shippedChecks) }

// kinds maps a check kind to the parameters it reads; every row must carry them for all three difficulties.
var kinds = map[string][]string{
	"level_gap":      {"warnGap", "failGap"},
	"resist_floor":   {"warnBelow", "failBelow", "cap", "poison"},
	"life_vs_hit":    {"warnHits", "failHits", "eliteDamagePct"},
	"hit_chance":     {"warnPct", "failPct"},
	"time_to_kill":   {"warnSeconds", "failSeconds", "eliteLifeMult"},
	"immunity_cover": {"eliteWarn"},
	"kit":            {"warnPotions", "failPotions"},
	"durability":     {"warnPct", "failPct"},
}

// ParseChecks parses and validates a table: unknown fields, kinds, classes, difficulties or missing parameters are
// errors that name the row.
func ParseChecks(b []byte) (*CheckTable, error) {
	var t CheckTable
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return nil, fmt.Errorf("readiness checks: %w", err)
	}
	if t.Schema != "readiness.checks/1" {
		return nil, fmt.Errorf("readiness checks: schema %q", t.Schema)
	}
	if len(t.Classes) == 0 || len(t.Checks) == 0 {
		return nil, fmt.Errorf("readiness checks: no classes or no checks")
	}
	for c, cr := range t.Classes {
		if len(cr.Damage) == 0 {
			return nil, fmt.Errorf("readiness checks: class %q has no damage types", c)
		}
		for _, e := range cr.Damage {
			if !slices.Contains(elementNames[:], e) {
				return nil, fmt.Errorf("readiness checks: class %q: unknown damage type %q", c, e)
			}
		}
	}
	ids := map[string]bool{}
	for _, r := range t.Checks {
		need, ok := kinds[r.Kind]
		if !ok {
			return nil, fmt.Errorf("readiness checks: row %q: unknown kind %q", r.ID, r.Kind)
		}
		if r.ID == "" || ids[r.ID] {
			return nil, fmt.Errorf("readiness checks: row id %q is empty or repeated", r.ID)
		}
		ids[r.ID] = true
		for _, d := range diffNames {
			p, ok := r.Diff[d]
			if !ok {
				return nil, fmt.Errorf("readiness checks: row %q has no parameters for %s", r.ID, d)
			}
			for _, k := range need {
				if _, ok := p[k]; !ok {
					return nil, fmt.Errorf("readiness checks: row %q %s: missing %q", r.ID, d, k)
				}
			}
		}
		for c, per := range r.Class {
			if _, ok := t.Classes[c]; !ok {
				return nil, fmt.Errorf("readiness checks: row %q overrides unknown class %q", r.ID, c)
			}
			for d, p := range per {
				if !slices.Contains(diffNames[:], d) {
					return nil, fmt.Errorf("readiness checks: row %q class %q: unknown difficulty %q", r.ID, c, d)
				}
				for k := range p {
					if _, ok := r.Diff[d][k]; !ok {
						return nil, fmt.Errorf("readiness checks: row %q class %q %s: parameter %q is not a parameter of the row", r.ID, c, d, k)
					}
				}
			}
		}
	}
	return &t, nil
}

// Evaluator joins the check table with the area table.
type Evaluator struct {
	checks *CheckTable
	areas  *AreaTable
}

// New checks the pair.
func New(c *CheckTable, a *AreaTable) (*Evaluator, error) {
	if c == nil || a == nil {
		return nil, fmt.Errorf("readiness: the evaluator needs a check table and an area table")
	}
	return &Evaluator{checks: c, areas: a}, nil
}

// Areas is the area table the evaluator reads.
func (e *Evaluator) Areas() *AreaTable { return e.areas }

func diffIndex(name string) (int, error) {
	i := slices.Index(diffNames[:], strings.ToLower(strings.TrimSpace(name)))
	if i < 0 {
		return 0, fmt.Errorf("readiness: unknown difficulty %q", name)
	}
	return i, nil
}

// Evaluate runs every check for hero against area (id or name). An unknown class, difficulty or area is an error naming it.
func (e *Evaluator) Evaluate(h Hero, area string) (Report, error) {
	class := strings.ToLower(strings.TrimSpace(h.Class))
	cr, ok := e.checks.Classes[class]
	if !ok {
		return Report{}, fmt.Errorf("readiness: no check data for class %q", h.Class)
	}
	d, err := diffIndex(h.Difficulty)
	if err != nil {
		return Report{}, err
	}
	a, err := e.areas.Find(area)
	if err != nil {
		return Report{}, err
	}
	ad := a.Diff[d]
	if ad.Spawns == 0 {
		return Report{}, fmt.Errorf("readiness: area %q has no spawns in %s", a.Name, diffNames[d])
	}
	rep := Report{Schema: ReportSchema, Class: class, Difficulty: diffNames[d], Area: a.Name, MLvl: ad.MLvl, Reasons: []string{}}
	for _, row := range e.checks.Checks {
		p := merged(row, class, diffNames[d])
		st, why := runCheck(row.Kind, p, h, d, cr, ad)
		rep.Checks = append(rep.Checks, Result{ID: row.ID, Threat: row.Threat, State: st, Reason: why})
	}
	rep.Light = Green
	for _, c := range rep.Checks {
		switch c.State {
		case Fail:
			rep.Light = Red
		case Warn:
			if rep.Light == Green {
				rep.Light = Amber
			}
		}
	}
	for _, st := range []string{Fail, Warn} {
		for _, c := range rep.Checks {
			if c.State == st {
				rep.Reasons = append(rep.Reasons, c.Reason)
			}
		}
	}
	return rep, nil
}

func merged(r checkRow, class, diff string) params {
	p := params{}
	for k, v := range r.Diff[diff] {
		p[k] = v
	}
	for k, v := range r.Class[class][diff] {
		p[k] = v
	}
	return p
}

// grade turns a value into a state: lowIsBad compares below the lines, otherwise above.
func grade(v, warn, fail float64, lowIsBad bool) string {
	if lowIsBad {
		switch {
		case v < fail:
			return Fail
		case v < warn:
			return Warn
		}
		return Pass
	}
	switch {
	case v > fail:
		return Fail
	case v > warn:
		return Warn
	}
	return Pass
}

func runCheck(kind string, p params, h Hero, d int, cr classRow, ad AreaDiff) (string, string) {
	switch kind {
	case "level_gap":
		gap := ad.MLvl - h.Level
		st := grade(float64(gap), p["warnGap"], p["failGap"], false)
		return st, fmt.Sprintf("level %d against area level %d (gap %d, warn above %.0f, fail above %.0f)", h.Level, ad.MLvl, gap, p["warnGap"], p["failGap"])
	case "resist_floor":
		return resistCheck(p, h, d)
	case "life_vs_hit":
		if ad.TopHit <= 0 || h.MaxLife <= 0 {
			return Skip, "no monster damage in the table or no life read"
		}
		hit := float64(ad.TopHit) * (1 + p["eliteDamagePct"]/100)
		n := float64(h.MaxLife) / hit
		st := grade(n, p["warnHits"], p["failHits"], true)
		return st, fmt.Sprintf("%d life takes %.1f elite hits of %.0f (warn below %.0f, fail below %.0f)", h.MaxLife, n, hit, p["warnHits"], p["failHits"])
	case "hit_chance":
		if h.Defence <= 0 || ad.TopToHit <= 0 {
			return Skip, "defence or monster attack rating not known"
		}
		pct := hitChance(float64(ad.TopToHit), float64(h.Defence), float64(ad.MLvl), float64(h.Level))
		st := grade(pct, p["warnPct"], p["failPct"], false)
		return st, fmt.Sprintf("the strongest attackers hit %.0f%% of the time with %d defence (warn above %.0f%%, fail above %.0f%%)", pct, h.Defence, p["warnPct"], p["failPct"])
	case "time_to_kill":
		if ad.TopLife <= 0 || h.DamagePerSec <= 0 {
			return Skip, "damage output or monster life not known"
		}
		life := float64(ad.TopLife) * p["eliteLifeMult"]
		s := life / h.DamagePerSec
		st := grade(s, p["warnSeconds"], p["failSeconds"], false)
		return st, fmt.Sprintf("an elite with %.0f life takes %.1f s at %.0f damage per second (warn above %.0f s, fail above %.0f s)", life, s, h.DamagePerSec, p["warnSeconds"], p["failSeconds"])
	case "immunity_cover":
		return immunityCheck(p, cr, ad)
	case "kit":
		st := grade(float64(h.Potions), p["warnPotions"], p["failPotions"], true)
		return st, fmt.Sprintf("%d healing potion%s (warn below %.0f, fail below %.0f)", h.Potions, plural(h.Potions), p["warnPotions"], p["failPotions"])
	case "durability":
		if h.RepairPct < 0 {
			return Skip, "gear durability not read"
		}
		st := grade(float64(h.RepairPct), p["warnPct"], p["failPct"], true)
		return st, fmt.Sprintf("worn gear at %d%% durability (warn below %.0f%%, fail below %.0f%%)", h.RepairPct, p["warnPct"], p["failPct"])
	}
	return Skip, "unknown check kind " + kind // ParseChecks rejects these
}

// worstResist is the lowest effective resist among the counted elements. It applies the difficulty penalty only when
// the numbers are raw (ResistRaw); raw numbers cap at cap before the penalty, and the game's floor is -100.
func worstResist(p params, h Hero, d int) (int, string) {
	pen := [3]int{0, -40, -100}[d]
	if h.Basis == ResistWithPenalty {
		pen = 0
	}
	cap := int(p["cap"])
	els := []struct {
		name string
		v    int
	}{{"fire", h.Resist.Fire}, {"cold", h.Resist.Cold}, {"lightning", h.Resist.Lightning}}
	if p["poison"] > 0 {
		els = append(els, struct {
			name string
			v    int
		}{"poison", h.Resist.Poison})
	}
	worst, wname := math.MaxInt, ""
	for _, e := range els {
		v := e.v
		if h.Basis == ResistRaw {
			v = min(v, cap)
		}
		v = max(v+pen, -100)
		if v < worst {
			worst, wname = v, e.name
		}
	}
	return worst, wname
}

func resistCheck(p params, h Hero, d int) (string, string) {
	worst, wname := worstResist(p, h, d)
	st := grade(float64(worst), p["warnBelow"], p["failBelow"], true)
	basis := "raw numbers plus the penalty"
	if h.Basis == ResistWithPenalty {
		basis = "numbers as shown"
	}
	return st, fmt.Sprintf("%s resist %d%% is the lowest (%s; warn below %.0f, fail below %.0f)", wname, worst, basis, p["warnBelow"], p["failBelow"])
}

// ResistGap is how many resist points the hero lacks to reach the warn line of the resist row for its class and
// difficulty (0 = none). It feeds the risk meter's ResistGap (internal/watch/unlockrisk.go). area only selects the
// difficulty's data today; the row has no per-area floor yet.
func (e *Evaluator) ResistGap(h Hero) (int, error) {
	class := strings.ToLower(strings.TrimSpace(h.Class))
	if _, ok := e.checks.Classes[class]; !ok {
		return 0, fmt.Errorf("readiness: no check data for class %q", h.Class)
	}
	d, err := diffIndex(h.Difficulty)
	if err != nil {
		return 0, err
	}
	for _, row := range e.checks.Checks {
		if row.Kind == "resist_floor" {
			p := merged(row, class, diffNames[d])
			worst, _ := worstResist(p, h, d)
			return max(0, int(p["warnBelow"])-worst), nil
		}
	}
	return 0, nil
}

// hitChance is the usual D2 formula (S): 200*AR/(AR+DEF) * mlvl/(mlvl+clvl), clamped to 5..95 percent.
func hitChance(ar, def, mlvl, clvl float64) float64 {
	c := 200 * ar / (ar + def) * mlvl / (mlvl + clvl)
	return math.Min(95, math.Max(5, c))
}

func immunityCheck(p params, cr classRow, ad AreaDiff) (string, string) {
	var open []string
	for _, e := range cr.Damage {
		if !slices.Contains(ad.Immune, e) {
			open = append(open, e)
		}
	}
	switch {
	case len(ad.Immune) == 0 && len(ad.EliteImmune) == 0:
		return Pass, "no immune monsters in the area table"
	case len(open) == 0:
		return Fail, fmt.Sprintf("every damage type of the class (%s) meets an immune monster here", strings.Join(cr.Damage, ", "))
	}
	hit := []string{}
	for _, e := range cr.Damage {
		if slices.Contains(ad.Immune, e) {
			hit = append(hit, e)
		}
	}
	if len(hit) > 0 {
		sort.Strings(hit)
		return Warn, fmt.Sprintf("some monsters here are immune to %s; the class falls back to %s", strings.Join(hit, ", "), strings.Join(open, ", "))
	}
	if p["eliteWarn"] > 0 && len(open) <= 2 {
		all := true
		for _, e := range open {
			all = all && slices.Contains(ad.EliteImmune, e)
		}
		if all {
			return Warn, fmt.Sprintf("elite packs here can become immune to %s, the only damage left to the class", strings.Join(open, ", "))
		}
	}
	return Pass, "the class has damage types left against every immunity here"
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

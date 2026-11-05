package decide

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

//go:embed catalogue.json
var shippedCatalogue []byte

// CatalogueVersion is the newest catalogue format and content version this build reads. Version 2 added the held,
// clvl, retreats, tripgold and packused fields and the answers "fight" and "leave" of t0.retreat.drink. Version 3 added
// the point-specific bands drinklife, lifeengage and mercgold (Field.From), the champbonus field and dropped packdanger
// from t0.retreat.drink, so that defaults equal the legacy knobs inside every band. Version 4 replaced levelgap by merc in
// the key of t0.area.toohard (the tower floor depends on a living merc, no floor reads the gap).
const CatalogueVersion = 4

// Risk classes (spec 5.1).
const (
	Safe   = "safe"
	Normal = "normal"
	Risky  = "risky"
)

// NotSure is the built-in extra answer of every point.
const NotSure = "notsure"

type Answer struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	// Noop marks "do nothing" answers; the default of an irreversible point must be one.
	Noop bool `json:"noop,omitempty"`
}

// Match maps a field to its accepted values, alternatives separated by "|". An empty Match matches everything.
type Match map[string]string

type RiskRow struct {
	Match Match  `json:"match"`
	Class string `json:"class"`
}

type DefaultRow struct {
	When   Match  `json:"when"`
	Answer string `json:"answer"`
}

type Point struct {
	ID           string       `json:"id"`
	Tier         int          `json:"tier"`
	Question     string       `json:"question"`
	Answers      []Answer     `json:"answers"`
	ContextKeys  []string     `json:"contextKeys"`
	RiskTable    []RiskRow    `json:"riskTable"`
	Default      []DefaultRow `json:"default"`
	Irreversible bool         `json:"irreversible,omitempty"`
	// Urgency: stage 2 swaps the "life" key field for "urgency" (spec 2.3).
	Urgency bool `json:"urgency,omitempty"`
}

// Field kinds: enum (closed list), band (numeric edges, shared by all builds) and classband (edges come from the
// class profile, so build-specific numbers live in data).
type Field struct {
	Kind   string    `json:"kind"`
	Values []string  `json:"values,omitempty"`
	Edges  []float64 `json:"edges,omitempty"`
	Labels []string  `json:"labels,omitempty"`
	// From names the observation a band reads when it is not its own name: drinklife bands the "life" number with
	// finer edges than the shared life field, so a call site reports one life value. Optional: a missing enum
	// observation falls back to Absent (a field added after call sites were written).
	From   string `json:"from,omitempty"`
	Absent string `json:"absent,omitempty"`
}

// isLife reports a key field that bands the life percent (the field "life" or a finer band reading it): stage 2
// replaces it by urgency.
func (c *Catalogue) isLife(name string) bool {
	return name == "life" || c.Fields[name].From == "life"
}

type Class struct {
	Bands map[string][]float64 `json:"bands"`
}

type Config struct {
	NByRisk  map[string]int `json:"nByRisk"`
	FadeSec  int            `json:"fadeSec"`
	MaxSkips int            `json:"maxSkips"`
}

// UrgencyTable is stage-2 data (spec 2.3). The store does not evaluate it; combat does, through UrgencyProvider.
type UrgencyTable struct {
	Stage int                `json:"stage"`
	Doc   string             `json:"doc"`
	Edges map[string]float64 `json:"edges"`
	Rows  []struct {
		Band string   `json:"band"`
		Any  []string `json:"any"`
	} `json:"rows"`
}

// Migration rewrites the keys of one point from the previous catalogue version (data, not code): fields are
// dropped, renamed, get a value set and have values mapped, in that order. Applied once when a profile file written
// by an older version is opened; a rule keeps its answer and moves to the new key.
type Migration struct {
	Point  string                       `json:"point"`
	Note   string                       `json:"note"`
	Drop   []string                     `json:"drop,omitempty"`
	Rename map[string]string            `json:"rename,omitempty"`
	Values map[string]map[string]string `json:"values,omitempty"`
	Add    map[string]string            `json:"add,omitempty"`
}

type Catalogue struct {
	Version    int              `json:"version"`
	Migrations []Migration      `json:"migrations,omitempty"`
	Config     Config           `json:"config"`
	Fields     map[string]Field `json:"fields"`
	Classes    map[string]Class `json:"classes"`
	Urgency    UrgencyTable     `json:"urgency"`
	Points     []Point          `json:"points"`

	idx map[string]*Point
}

// Shipped loads the embedded T0 to T2 catalogue.
func Shipped() (*Catalogue, error) { return ParseCatalogue(shippedCatalogue) }

// ParseCatalogue decodes and validates a catalogue document.
func ParseCatalogue(data []byte) (*Catalogue, error) {
	var c Catalogue
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("decide: catalogue: %w", err)
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// Point returns the point or an error naming the unknown id.
func (c *Catalogue) Point(id string) (*Point, error) {
	if p, ok := c.idx[id]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("decide: unknown decision point %q", id)
}

func (p *Point) answer(id string) (Answer, bool) {
	for _, a := range p.Answers {
		if a.ID == id {
			return a, true
		}
	}
	return Answer{}, false
}

func badToken(s string) bool { return s == "" || strings.ContainsAny(s, "|,=") }

func (c *Catalogue) validate() error {
	if c.Version < 1 || c.Version > CatalogueVersion {
		return fmt.Errorf("decide: catalogue version %d not supported", c.Version)
	}
	for _, cl := range []string{Safe, Normal, Risky} {
		if c.Config.NByRisk[cl] < 1 {
			return fmt.Errorf("decide: nByRisk[%s] must be at least 1", cl)
		}
	}
	if c.Config.FadeSec < 1 || c.Config.MaxSkips < 0 {
		return fmt.Errorf("decide: bad fadeSec or maxSkips")
	}
	for name, f := range c.Fields {
		var labels []string
		switch f.Kind {
		case "enum":
			labels = f.Values
		case "band", "classband":
			labels = f.Labels
			if f.Kind == "band" && len(f.Edges)+1 != len(f.Labels) {
				return fmt.Errorf("decide: field %q: %d edges need %d labels", name, len(f.Edges), len(f.Edges)+1)
			}
			if !sort.Float64sAreSorted(f.Edges) {
				return fmt.Errorf("decide: field %q: edges not ascending", name)
			}
		default:
			return fmt.Errorf("decide: field %q: unknown kind %q", name, f.Kind)
		}
		if len(labels) == 0 {
			return fmt.Errorf("decide: field %q has no values", name)
		}
		if f.From != "" && f.Kind != "band" {
			return fmt.Errorf("decide: field %q: from needs a band", name)
		}
		if f.Absent != "" && (f.Kind != "enum" || !contains(f.Values, f.Absent)) {
			return fmt.Errorf("decide: field %q: absent %q is not a value", name, f.Absent)
		}
		for _, l := range labels {
			if badToken(l) {
				return fmt.Errorf("decide: field %q: bad value %q", name, l)
			}
		}
	}
	for _, name := range []string{"act", "diff", "hc"} {
		if f, ok := c.Fields[name]; !ok || f.Kind != "enum" {
			return fmt.Errorf("decide: field %q must be an enum", name)
		}
	}
	for cn, cl := range c.Classes {
		for fn, edges := range cl.Bands {
			f, ok := c.Fields[fn]
			if !ok || f.Kind != "classband" || len(edges)+1 != len(f.Labels) || !sort.Float64sAreSorted(edges) {
				return fmt.Errorf("decide: class %q: bad band %q", cn, fn)
			}
		}
	}
	c.idx = map[string]*Point{}
	for i := range c.Points {
		p := &c.Points[i]
		if badToken(p.ID) || strings.Contains(p.ID, " ") {
			return fmt.Errorf("decide: bad point id %q", p.ID)
		}
		if _, dup := c.idx[p.ID]; dup {
			return fmt.Errorf("decide: duplicate point %q", p.ID)
		}
		c.idx[p.ID] = p
		if err := c.validatePoint(p); err != nil {
			return err
		}
	}
	for _, m := range c.Migrations {
		if _, ok := c.idx[m.Point]; !ok {
			return fmt.Errorf("decide: migration names unknown point %q", m.Point)
		}
		check := func(field, val string) error {
			f, ok := c.Fields[field]
			labels := f.Values
			if f.Kind != "enum" {
				labels = f.Labels
			}
			if !ok || !contains(labels, val) {
				return fmt.Errorf("decide: migration of %q: field %q has no value %q", m.Point, field, val)
			}
			return nil
		}
		for f, v := range m.Add {
			if err := check(f, v); err != nil {
				return err
			}
		}
		for f, vals := range m.Values {
			for _, to := range vals {
				if err := check(f, to); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// MigrateKey rewrites a context key written by an older catalogue version to the current shape. A key of a point
// without a migration, or a key already in the new shape, comes back unchanged.
func (c *Catalogue) MigrateKey(key string) string {
	ctx, err := ParseKey(key)
	if err != nil {
		return key
	}
	changed := false
	for _, m := range c.Migrations {
		if m.Point != ctx.Point {
			continue
		}
		for _, f := range m.Drop {
			if _, ok := ctx.Fields[f]; ok {
				delete(ctx.Fields, f)
				changed = true
			}
		}
		for from, to := range m.Rename {
			if v, ok := ctx.Fields[from]; ok {
				delete(ctx.Fields, from)
				ctx.Fields[to] = v
				changed = true
			}
		}
		for f, vals := range m.Values {
			if v, ok := ctx.Fields[f]; ok && vals[v] != "" {
				ctx.Fields[f] = vals[v]
				changed = true
			}
		}
		for f, v := range m.Add {
			if _, ok := ctx.Fields[f]; !ok {
				ctx.Fields[f] = v
				changed = true
			}
		}
	}
	if !changed {
		return key
	}
	return makeKey(ctx.Point, ctx.Fields)
}

func (c *Catalogue) validateMatch(p *Point, m Match) error {
	for f, v := range m {
		def, ok := c.Fields[f]
		if !ok {
			return fmt.Errorf("decide: point %q: match names unknown field %q", p.ID, f)
		}
		labels := def.Values
		if def.Kind != "enum" {
			labels = def.Labels
		}
		for _, alt := range strings.Split(v, "|") {
			if !contains(labels, alt) {
				return fmt.Errorf("decide: point %q: field %q has no value %q", p.ID, f, alt)
			}
		}
	}
	return nil
}

func (c *Catalogue) validatePoint(p *Point) error {
	if len(p.Answers) < 2 || len(p.Answers) > 5 {
		return fmt.Errorf("decide: point %q: needs 2 to 5 answers", p.ID)
	}
	for _, a := range p.Answers {
		if badToken(a.ID) || a.ID == NotSure {
			return fmt.Errorf("decide: point %q: bad answer id %q", p.ID, a.ID)
		}
	}
	for _, k := range p.ContextKeys {
		if _, ok := c.Fields[k]; !ok {
			return fmt.Errorf("decide: point %q: context key names unknown field %q", p.ID, k)
		}
	}
	if p.Urgency {
		hasLife := false
		for _, k := range p.ContextKeys {
			hasLife = hasLife || c.isLife(k)
		}
		if !hasLife {
			return fmt.Errorf("decide: point %q: urgency point needs a life key", p.ID)
		}
		if _, ok := c.Fields["urgency"]; !ok {
			return fmt.Errorf("decide: point %q: no urgency field", p.ID)
		}
	}
	if n := len(p.RiskTable); n == 0 || len(p.RiskTable[n-1].Match) != 0 {
		return fmt.Errorf("decide: point %q: risk table must end with a catch-all row", p.ID)
	}
	for _, r := range p.RiskTable {
		if c.Config.NByRisk[r.Class] == 0 {
			return fmt.Errorf("decide: point %q: unknown risk class %q", p.ID, r.Class)
		}
		if err := c.validateMatch(p, r.Match); err != nil {
			return err
		}
	}
	if n := len(p.Default); n == 0 || len(p.Default[n-1].When) != 0 {
		return fmt.Errorf("decide: point %q: default must end with a catch-all row", p.ID)
	}
	for _, d := range p.Default {
		a, ok := p.answer(d.Answer)
		if !ok {
			return fmt.Errorf("decide: point %q: default answer %q is not an answer", p.ID, d.Answer)
		}
		if p.Irreversible && !a.Noop {
			return fmt.Errorf("decide: point %q is irreversible: default %q must be a do-nothing answer", p.ID, d.Answer)
		}
		if err := c.validateMatch(p, d.When); err != nil {
			return err
		}
	}
	return nil
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// matches: every named field must be present in fields with an accepted value; a field absent from the key makes
// the row not apply.
func (m Match) matches(fields map[string]string) bool {
	for f, v := range m {
		got, ok := fields[f]
		if !ok || !contains(strings.Split(v, "|"), got) {
			return false
		}
	}
	return true
}

// RiskClass is the fixed class of a context (spec 5.1): first matching row of the point's table.
func (c *Catalogue) RiskClass(ctx Context) (string, error) {
	p, err := c.Point(ctx.Point)
	if err != nil {
		return "", err
	}
	for _, r := range p.RiskTable {
		if r.Match.matches(ctx.Fields) {
			return r.Class, nil
		}
	}
	return "", fmt.Errorf("decide: point %q: no risk row for %s", p.ID, ctx.Key)
}

// N is the streak length needed for the context's risk class.
func (c *Catalogue) N(ctx Context) (int, error) {
	cl, err := c.RiskClass(ctx)
	if err != nil {
		return 0, err
	}
	return c.Config.NByRisk[cl], nil
}

// Default is the untrained behaviour: first matching default row. It never waits and never depends on a rule.
func (c *Catalogue) Default(ctx Context) (string, error) {
	p, err := c.Point(ctx.Point)
	if err != nil {
		return "", err
	}
	for _, d := range p.Default {
		if d.When.matches(ctx.Fields) {
			return d.Answer, nil
		}
	}
	return "", fmt.Errorf("decide: point %q: no default row for %s", p.ID, ctx.Key)
}

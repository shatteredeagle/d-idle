// Package pact is the Pact catalogue and its rules (docs/research/pact-system.md, ADR 0019). A Pact is data in
// data/pacts/pacts.json: no Pact id, class or knob is a Go literal. Every reward and heat number is U (a design note,
// tuned in play).
package pact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	pactdata "d2lod-bot/data/pacts"
)

// Lever families, kinds, away safety classes and unlock triggers the loader accepts.
var (
	families = map[string]bool{"bot": true, "native": true, "overlay": true}
	kinds    = map[string]bool{"ranked": true, "fixed": true}
	aways    = map[string]bool{"safe": true, "needs_consent": true, "present_only": true}
	triggers = map[string]bool{"andariel_first_kill": true, "baal_normal": true, "baal_nightmare": true}
)

// Limits are the heat and slot numbers (data, not constants).
type Limits struct {
	StartCap      int `json:"start_cap"`
	MaxCap        int `json:"max_cap"`
	MasteryStep   int `json:"mastery_step"`
	SlotsAtUnlock int `json:"slots_at_unlock"`
	NoSafetyRank  int `json:"no_safety_rank"` // two Pacts tagged no_safety at this rank or more are incompatible
}

// Params is what a rank changes. Row and Option name a policy sheet option; Knobs are tuner overrides; Values are the
// Pact-specific numbers the bot side reads (death limit, clvl floor, players ...).
type Params struct {
	Row    string             `json:"row,omitempty"`
	Option string             `json:"option,omitempty"`
	Knobs  map[string]float64 `json:"knobs,omitempty"`
	Values map[string]float64 `json:"values,omitempty"`
}

// Reward is paid on the meta layer only (tally, charm points, a title); never vanilla XP or loot.
type Reward struct {
	TallyPct    int    `json:"tally_pct"`
	CharmPoints int    `json:"charm_points,omitempty"`
	Title       string `json:"title,omitempty"`
	Cosmetic    string `json:"cosmetic,omitempty"`
}

// Rank is one defined rank of a Pact. A Pact need not define every rank 1..5; only defined ranks can be chosen.
type Rank struct {
	Rank       int    `json:"rank"`
	Heat       int    `json:"heat"`
	Params     Params `json:"params"`
	Reward     Reward `json:"reward"`
	EffectLine string `json:"effect_line"`
}

// Incompat names the Pact (ID) this one cannot be active with. MinRank and OtherMinRank limit it to high ranks of this
// Pact and of the other; 0 = any rank.
type Incompat struct {
	ID           string `json:"id"`
	MinRank      int    `json:"min_rank,omitempty"`
	OtherMinRank int    `json:"other_min_rank,omitempty"`
	Reason       string `json:"reason"`
}

// Pact is one catalogue entry.
type Pact struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	LeverFamily      string     `json:"lever_family"`
	Kind             string     `json:"kind"`
	UnlockTrigger    string     `json:"unlock_trigger"`
	ClassScope       []string   `json:"class_scope"` // ["all"] or class names
	AwaySafety       string     `json:"away_safety"`
	Tags             []string   `json:"tags"`
	IncompatibleWith []Incompat `json:"incompatible_with"`
	Ranks            []Rank     `json:"ranks"`
	LearnText        string     `json:"learn_text"`
}

// Rank returns the defined rank or an error naming the Pact and the rank.
func (p *Pact) Rank(r int) (*Rank, error) {
	for i := range p.Ranks {
		if p.Ranks[i].Rank == r {
			return &p.Ranks[i], nil
		}
	}
	return nil, fmt.Errorf("pact: %q has no rank %d", p.ID, r)
}

func (p *Pact) hasTag(t string) bool {
	for _, x := range p.Tags {
		if x == t {
			return true
		}
	}
	return false
}

// AllowsClass reports the class scope; the class name is lower case.
func (p *Pact) AllowsClass(class string) bool {
	for _, c := range p.ClassScope {
		if c == "all" || c == class {
			return true
		}
	}
	return false
}

// Catalogue is the loaded and validated file.
type Catalogue struct {
	Version int    `json:"version"`
	Source  string `json:"source"`
	Limits  Limits `json:"limits"`
	Pacts   []Pact `json:"pacts"`
	idx     map[string]*Pact
}

// Refs lets the loader check names against the things a Pact points at, without importing them (the callers hold the
// policy sheet table, the knob tables and the class table). A nil function skips that check.
type Refs struct {
	Class  func(name string) bool
	Option func(row, option string) error
	Knob   func(key string) (min, max float64, ok bool)
}

// Shipped loads the embedded catalogue.
func Shipped(refs Refs) (*Catalogue, error) { return Parse(pactdata.JSON, refs) }

// Parse decodes and validates. Every failure names the Pact (and rank) and the cause: unknown field, family, kind,
// trigger, away class, class, policy row or option, knob, duplicate id, a rank without heat, an incompatibility naming
// a missing Pact.
func Parse(data []byte, refs Refs) (*Catalogue, error) {
	var c Catalogue
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("pact: catalogue: %w", err)
	}
	if c.Version != 1 {
		return nil, fmt.Errorf("pact: catalogue version %d not supported", c.Version)
	}
	l := c.Limits
	if l.StartCap <= 0 || l.MaxCap < l.StartCap || l.MasteryStep <= 0 || l.SlotsAtUnlock <= 0 || l.NoSafetyRank <= 0 {
		return nil, fmt.Errorf("pact: catalogue: bad limits %+v", l)
	}
	c.idx = map[string]*Pact{}
	for i := range c.Pacts {
		p := &c.Pacts[i]
		if p.ID == "" || strings.ContainsAny(p.ID, " \t") {
			return nil, fmt.Errorf("pact: entry %d: bad id %q", i+1, p.ID)
		}
		if c.idx[p.ID] != nil {
			return nil, fmt.Errorf("pact: duplicate id %q", p.ID)
		}
		c.idx[p.ID] = p
	}
	for i := range c.Pacts {
		if err := c.Pacts[i].validate(&c, refs); err != nil {
			return nil, err
		}
	}
	return &c, nil
}

func (p *Pact) validate(c *Catalogue, refs Refs) error {
	if p.Name == "" || p.LearnText == "" {
		return fmt.Errorf("pact %q: name and learn_text are required", p.ID)
	}
	if !families[p.LeverFamily] {
		return fmt.Errorf("pact %q: unknown lever_family %q", p.ID, p.LeverFamily)
	}
	if !kinds[p.Kind] {
		return fmt.Errorf("pact %q: unknown kind %q", p.ID, p.Kind)
	}
	if !aways[p.AwaySafety] {
		return fmt.Errorf("pact %q: unknown away_safety %q", p.ID, p.AwaySafety)
	}
	if !triggers[p.UnlockTrigger] && !validCounted(p.UnlockTrigger) {
		return fmt.Errorf("pact %q: unknown unlock_trigger %q", p.ID, p.UnlockTrigger)
	}
	if len(p.ClassScope) == 0 {
		return fmt.Errorf("pact %q: class_scope is empty (use [\"all\"])", p.ID)
	}
	for _, cl := range p.ClassScope {
		if cl != "all" && refs.Class != nil && !refs.Class(cl) {
			return fmt.Errorf("pact %q: unknown class %q in class_scope", p.ID, cl)
		}
	}
	if len(p.Ranks) == 0 || p.Kind == "fixed" && len(p.Ranks) != 1 {
		return fmt.Errorf("pact %q: kind %q with %d ranks", p.ID, p.Kind, len(p.Ranks))
	}
	prev := 0
	for _, r := range p.Ranks {
		if r.Rank < 1 || r.Rank > 5 || r.Rank <= prev {
			return fmt.Errorf("pact %q: rank %d is out of order or outside 1..5", p.ID, r.Rank)
		}
		prev = r.Rank
		if r.Heat <= 0 {
			return fmt.Errorf("pact %q rank %d: no heat (every rank costs heat)", p.ID, r.Rank)
		}
		if r.EffectLine == "" {
			return fmt.Errorf("pact %q rank %d: no effect_line", p.ID, r.Rank)
		}
		if r.Reward.TallyPct < 0 {
			return fmt.Errorf("pact %q rank %d: negative reward", p.ID, r.Rank)
		}
		if err := r.Params.check(p.ID, r.Rank, refs); err != nil {
			return err
		}
	}
	for _, in := range p.IncompatibleWith {
		if c.idx[in.ID] == nil {
			return fmt.Errorf("pact %q: incompatible_with names missing pact %q", p.ID, in.ID)
		}
		if in.ID == p.ID || in.Reason == "" {
			return fmt.Errorf("pact %q: incompatibility with %q needs another pact and a reason", p.ID, in.ID)
		}
	}
	return nil
}

func validCounted(t string) bool {
	for _, pre := range []string{"hall_count:", "tally:"} {
		if n, ok := strings.CutPrefix(t, pre); ok {
			var v int
			_, err := fmt.Sscanf(n, "%d", &v)
			return err == nil && v > 0
		}
	}
	return false
}

func (pa Params) check(id string, rank int, refs Refs) error {
	if (pa.Row == "") != (pa.Option == "") {
		return fmt.Errorf("pact %q rank %d: row and option go together", id, rank)
	}
	if pa.Row != "" && refs.Option != nil {
		if err := refs.Option(pa.Row, pa.Option); err != nil {
			return fmt.Errorf("pact %q rank %d: %w", id, rank, err)
		}
	}
	if refs.Knob == nil {
		return nil
	}
	keys := make([]string, 0, len(pa.Knobs))
	for k := range pa.Knobs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		lo, hi, ok := refs.Knob(k)
		if !ok {
			return fmt.Errorf("pact %q rank %d: %q is not a knob", id, rank, k)
		}
		if v := pa.Knobs[k]; v < lo || v > hi {
			return fmt.Errorf("pact %q rank %d: knob %s = %v outside %v..%v", id, rank, k, v, lo, hi)
		}
	}
	return nil
}

// Get returns the Pact or an error naming the id.
func (c *Catalogue) Get(id string) (*Pact, error) {
	if p, ok := c.idx[id]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("pact: unknown pact %q", id)
}

// HeatCap is the cap after this many masteries (a Pact cleared at its top rank), never above the maximum.
func (c *Catalogue) HeatCap(masteries int) int {
	return min(c.Limits.StartCap+max(masteries, 0)*c.Limits.MasteryStep, c.Limits.MaxCap)
}

// Pick is one chosen Pact at one rank.
type Pick struct {
	ID   string
	Rank int
}

// Heat is the summed heat of a set, or an error naming the first pick that does not exist.
func (c *Catalogue) Heat(set []Pick) (int, error) {
	sum := 0
	for _, s := range set {
		p, err := c.Get(s.ID)
		if err != nil {
			return 0, err
		}
		r, err := p.Rank(s.Rank)
		if err != nil {
			return 0, err
		}
		sum += r.Heat
	}
	return sum, nil
}

// Check applies the set rules in one pass and returns the first broken one, naming the Pacts: a Pact twice, an unknown
// Pact or rank, a class out of scope, more picks than slots, a heat sum over the cap, a declared incompatibility
// (either side may declare it), and two no_safety Pacts both at Limits.NoSafetyRank or more. It refuses nothing below
// the cap for being a poor choice (the same "your call" stance as the policy sheet).
func (c *Catalogue) Check(set []Pick, class string, slots, masteries int) error {
	seen := map[string]int{}
	for _, s := range set {
		p, err := c.Get(s.ID)
		if err != nil {
			return err
		}
		if _, err := p.Rank(s.Rank); err != nil {
			return err
		}
		if _, dup := seen[s.ID]; dup {
			return fmt.Errorf("pact: %q chosen twice", s.ID)
		}
		seen[s.ID] = s.Rank
		if !p.AllowsClass(class) {
			return fmt.Errorf("pact: %q is not for class %q", s.ID, class)
		}
	}
	if len(set) > slots {
		return fmt.Errorf("pact: %d pacts chosen, %d slots", len(set), slots)
	}
	heat, err := c.Heat(set)
	if err != nil {
		return err
	}
	if cp := c.HeatCap(masteries); heat > cp {
		return fmt.Errorf("pact: heat %d is over the cap %d", heat, cp)
	}
	for _, s := range set {
		p := c.idx[s.ID]
		for _, in := range p.IncompatibleWith {
			or, on := seen[in.ID]
			if on && s.Rank >= in.MinRank && or >= in.OtherMinRank {
				return fmt.Errorf("pact: %q and %q cannot be active together: %s", p.Name, c.idx[in.ID].Name, in.Reason)
			}
		}
	}
	for i, a := range set {
		for _, b := range set[i+1:] {
			pa, pb := c.idx[a.ID], c.idx[b.ID]
			if pa.hasTag("no_safety") && pb.hasTag("no_safety") && a.Rank >= c.Limits.NoSafetyRank && b.Rank >= c.Limits.NoSafetyRank {
				return fmt.Errorf("pact: %q and %q both remove a safety net from rank %d", pa.Name, pb.Name, c.Limits.NoSafetyRank)
			}
		}
	}
	return nil
}

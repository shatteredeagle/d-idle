package decide

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// The chores policy sheet (TASK-128, ADR 0018 item 4, docs/policy-sheet.md). A row is a chore the player settles once
// and may change at any time; each option of a row is a list of ladder rules (point, partial context, answer) and an
// optional set of knob values. The choice is one log line of the decide store (OpPolicy), so it persists and replays
// with the answers and revoking nothing else. A policy answers before a learned rule and before any card: a chore
// point it covers stops opening thoughts (Decision.Source "policy").

//go:embed policy_chores.json
var shippedPolicy []byte

// PolicyRule is one rule of an option; the first matching rule of a row decides. An omitted When field matches anything.
type PolicyRule struct {
	Point  string `json:"point"`
	When   Match  `json:"when,omitempty"`
	Answer string `json:"answer"`
}

// Applies says how much of an option the bot acts on today.
const (
	AppliesFull   = ""       // the rules and knobs change behaviour at their call sites
	AppliesPart   = "part"   // some of it is wired; Note says what is not
	AppliesRecord = "record" // stored and shown, but no bot path reads it yet (a gap, named in Note)
)

// PolicyOption is one allowed value of a row. Suboptimal options are allowed on purpose (user, 2026-11-07): the sheet
// flags them and says the risk, it never blocks them.
type PolicyOption struct {
	ID         string             `json:"id"`
	Label      string             `json:"label"`
	Suboptimal bool               `json:"suboptimal,omitempty"`
	Risk       string             `json:"risk,omitempty"` // one line, required with suboptimal
	Rules      []PolicyRule       `json:"rules,omitempty"`
	Knobs      map[string]float64 `json:"knobs,omitempty"`
	// ClassKnobs are knob values per catalogue class, added to Knobs (a class value wins). When present every
	// catalogue class must have an entry: a missing class is an error naming it, never a fallback to another class.
	ClassKnobs map[string]map[string]float64 `json:"classKnobs,omitempty"`
	// Says is the option in plain words for the sheet and the before/after card.
	Says    string `json:"says,omitempty"`
	Applies string `json:"applies,omitempty"`
	Note    string `json:"note,omitempty"`
	// Unlock names the unlock ladder row (data/unlocks/ladder.json) that opens the option; empty: always available.
	// The sheet refuses a locked option and shows what it costs.
	Unlock string `json:"unlock,omitempty"`
	// Hidden options are set by a Pact only; the sheet does not list them (unless one is the current option).
	Hidden bool `json:"hidden,omitempty"`
}

// KnobsFor is the knob values the option sets for a class: Knobs with the class's ClassKnobs on top. An option with
// class knobs and a class without an entry is an error naming the class.
func (o *PolicyOption) KnobsFor(class string) (map[string]float64, error) {
	out := make(map[string]float64, len(o.Knobs))
	for k, v := range o.Knobs {
		out[k] = v
	}
	if len(o.ClassKnobs) == 0 {
		return out, nil
	}
	ck, ok := o.ClassKnobs[class]
	if !ok {
		return nil, fmt.Errorf("decide: policy option %q has no knob values for class %q", o.ID, class)
	}
	for k, v := range ck {
		out[k] = v
	}
	return out, nil
}

// EffectCounter names one counter of the effect line of a row (the watch package computes it from the event stream).
type EffectCounter struct {
	Counter   string `json:"counter"`
	Label     string `json:"label"`
	Heuristic bool   `json:"heuristic,omitempty"` // the causal link to the setting is a guess
}

// PolicyRow is one chore of the sheet.
type PolicyRow struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Help  string `json:"help"`
	// Classes limits the row to these catalogue classes; empty: every class.
	Classes []string        `json:"classes,omitempty"`
	Default string          `json:"default"`
	Options []PolicyOption  `json:"options"`
	Effect  []EffectCounter `json:"effect,omitempty"`
}

// Option returns the option or an error naming the row and the id.
func (r *PolicyRow) Option(id string) (*PolicyOption, error) {
	for i := range r.Options {
		if r.Options[i].ID == id {
			return &r.Options[i], nil
		}
	}
	return nil, fmt.Errorf("decide: policy row %q has no option %q", r.ID, id)
}

// PolicyTable is the loaded sheet definition.
type PolicyTable struct {
	Version int         `json:"version"`
	Rows    []PolicyRow `json:"rows"`
	idx     map[string]*PolicyRow
}

// ShippedPolicy loads the embedded table and checks it against the catalogue.
func ShippedPolicy(cat *Catalogue) (*PolicyTable, error) { return ParsePolicy(shippedPolicy, cat) }

// ParsePolicy decodes a table and validates every row; an unknown point, field, value, answer, class or default is an
// error naming it.
func ParsePolicy(data []byte, cat *Catalogue) (*PolicyTable, error) {
	var t PolicyTable
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("decide: policy table: %w", err)
	}
	if t.Version != 1 {
		return nil, fmt.Errorf("decide: policy table version %d not supported", t.Version)
	}
	t.idx = map[string]*PolicyRow{}
	for i := range t.Rows {
		r := &t.Rows[i]
		if r.ID == "" || badToken(r.ID) || t.idx[r.ID] != nil {
			return nil, fmt.Errorf("decide: policy row %d: bad or duplicate id %q", i+1, r.ID)
		}
		t.idx[r.ID] = r
		if err := r.validate(cat); err != nil {
			return nil, err
		}
	}
	// Two rows deciding the same point would shadow each other (rows answer in id order).
	owner := map[string]string{}
	for _, r := range t.Rows {
		for _, o := range r.Options {
			for _, ru := range o.Rules {
				if prev, ok := owner[ru.Point]; ok && prev != r.ID {
					return nil, fmt.Errorf("decide: policy rows %q and %q both decide point %q", prev, r.ID, ru.Point)
				}
				owner[ru.Point] = r.ID
			}
		}
	}
	return &t, nil
}

func (r *PolicyRow) validate(cat *Catalogue) error {
	if r.Label == "" || r.Help == "" {
		return fmt.Errorf("decide: policy row %q: label and help are required", r.ID)
	}
	for _, c := range r.Classes {
		if _, ok := cat.Classes[c]; !ok {
			return fmt.Errorf("decide: policy row %q: unknown class %q", r.ID, c)
		}
	}
	def, err := r.Option(r.Default)
	if err != nil {
		return fmt.Errorf("decide: policy row %q: default: %w", r.ID, err)
	}
	if len(def.Rules) > 0 || len(def.Knobs) > 0 || len(def.ClassKnobs) > 0 || def.Suboptimal {
		return fmt.Errorf("decide: policy row %q: the default option must change nothing (it is today's behaviour)", r.ID)
	}
	seen := map[string]bool{}
	for _, o := range r.Options {
		if o.ID == "" || badToken(o.ID) || seen[o.ID] || o.Label == "" {
			return fmt.Errorf("decide: policy row %q: bad, duplicate or unlabelled option %q", r.ID, o.ID)
		}
		seen[o.ID] = true
		if o.Suboptimal && o.Risk == "" {
			return fmt.Errorf("decide: policy row %q option %q: a suboptimal option needs a risk line", r.ID, o.ID)
		}
		if o.Applies != AppliesFull && o.Applies != AppliesPart && o.Applies != AppliesRecord {
			return fmt.Errorf("decide: policy row %q option %q: applies %q", r.ID, o.ID, o.Applies)
		}
		if o.Applies != AppliesFull && o.Note == "" {
			return fmt.Errorf("decide: policy row %q option %q: a partial or record option needs a note", r.ID, o.ID)
		}
		if err := cat.checkPolicyRules(o.Rules); err != nil {
			return fmt.Errorf("decide: policy row %q option %q: %w", r.ID, o.ID, err)
		}
		if len(o.ClassKnobs) > 0 {
			for c := range o.ClassKnobs {
				if _, ok := cat.Classes[c]; !ok {
					return fmt.Errorf("decide: policy row %q option %q: classKnobs name unknown class %q", r.ID, o.ID, c)
				}
			}
			for c := range cat.Classes {
				if _, ok := o.ClassKnobs[c]; !ok {
					return fmt.Errorf("decide: policy row %q option %q: classKnobs has no entry for class %q", r.ID, o.ID, c)
				}
			}
		}
	}
	return nil
}

func (c *Catalogue) checkPolicyRules(rules []PolicyRule) error {
	for i, ru := range rules {
		p, err := c.Point(ru.Point)
		if err != nil {
			return fmt.Errorf("rule %d: %w", i+1, err)
		}
		if _, ok := p.answer(ru.Answer); !ok {
			return fmt.Errorf("rule %d: %q is not an answer of %s", i+1, ru.Answer, p.ID)
		}
		if err := c.validateMatch(p, ru.When); err != nil {
			return fmt.Errorf("rule %d: %w", i+1, err)
		}
	}
	return nil
}

// Row returns the row or an error naming the unknown id.
func (t *PolicyTable) Row(id string) (*PolicyRow, error) {
	if r, ok := t.idx[id]; ok {
		return r, nil
	}
	return nil, fmt.Errorf("decide: unknown policy row %q", id)
}

// RowsFor lists the rows of a class in table order. A class the catalogue does not know is an error naming it (no
// silent fallback to the default class).
func (t *PolicyTable) RowsFor(cat *Catalogue, class string) ([]*PolicyRow, error) {
	if _, ok := cat.Classes[class]; !ok {
		return nil, fmt.Errorf("decide: policy sheet: unknown class %q", class)
	}
	var out []*PolicyRow
	for i := range t.Rows {
		r := &t.Rows[i]
		if len(r.Classes) == 0 || contains(r.Classes, class) {
			out = append(out, r)
		}
	}
	return out, nil
}

// ---- the store side --------------------------------------------------------------------------------------------------

// PolicyState is the current choice of one row.
type PolicyState struct {
	Option string       `json:"option"`
	Rules  []PolicyRule `json:"rules,omitempty"`
	At     time.Duration
}

// PolicyView is PolicyState with its row id.
type PolicyView struct {
	Row    string
	Option string
	Rules  []PolicyRule
	At     time.Duration
}

func (s *Store) applyPolicy(e Entry) error {
	if e.Key == "" || e.Answer == "" {
		return fmt.Errorf("decide: policy entry needs a row and an option")
	}
	if err := s.cat.checkPolicyRules(e.Rules); err != nil {
		return err
	}
	s.policy[e.Key] = &PolicyState{Option: e.Answer, Rules: e.Rules, At: e.T}
	return nil
}

// SetPolicy records that sheet row `row` now has option `option` with these rules (none: the row follows the
// legacy behaviour). It is one log line, so the choice survives a reload and replays like an answer. Rules naming an
// unknown point, field, value or answer are refused and nothing is logged.
func (s *Store) SetPolicy(row, option string, rules []PolicyRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if row == "" || option == "" {
		return fmt.Errorf("decide: SetPolicy needs a row and an option")
	}
	if err := s.cat.checkPolicyRules(rules); err != nil {
		return err
	}
	_, err := s.commit(Entry{Op: OpPolicy, Key: row, Answer: option, Rules: rules})
	return err
}

// Policies lists the current choice of every row that was ever set, sorted by row.
func (s *Store) Policies() []PolicyView {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []PolicyView
	for row, p := range s.policy {
		out = append(out, PolicyView{Row: row, Option: p.Option, Rules: p.Rules, At: p.At})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Row < out[j].Row })
	return out
}

// PolicyOption is the current option id of a row, "" when the row was never set.
func (s *Store) PolicyOption(row string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p := s.policy[row]; p != nil {
		return p.Option
	}
	return ""
}

// PolicyLog lists the policy changes of the log in order (row, option, time on the store's clock).
func (s *Store) PolicyLog() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Entry
	for _, e := range s.log {
		if e.Op == OpPolicy {
			out = append(out, e)
		}
	}
	return out
}

// policyAnswer: rows in id order, rules in file order, first match wins. Caller holds mu.
func (s *Store) policyAnswer(ctx Context) (string, bool) {
	if len(s.policy) == 0 {
		return "", false
	}
	rows := make([]string, 0, len(s.policy))
	for r := range s.policy {
		rows = append(rows, r)
	}
	sort.Strings(rows)
	for _, r := range rows {
		for _, ru := range s.policy[r].Rules {
			if ru.Point == ctx.Point && ru.When.matches(ctx.Fields) {
				return ru.Answer, true
			}
		}
	}
	return "", false
}

// PlainRule is a policy rule in words for the Rules panel: "Repair gear now? [gold: lt100] -> skip repair".
func (s *Store) PlainRule(ru PolicyRule) string {
	p, err := s.cat.Point(ru.Point)
	if err != nil {
		return ru.Point + " -> " + ru.Answer
	}
	label := ru.Answer
	if a, ok := p.answer(ru.Answer); ok {
		label = a.Label
	}
	var when []string
	for f, v := range ru.When {
		when = append(when, f+" "+v)
	}
	sort.Strings(when)
	w := "always"
	if len(when) > 0 {
		w = strings.Join(when, ", ")
	}
	return fmt.Sprintf("%s (%s): %s", p.Question, w, label)
}

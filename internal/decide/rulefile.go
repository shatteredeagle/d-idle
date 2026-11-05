package decide

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
)

//go:embed default_rules.json
var shippedRules []byte

// FileRule is one row of a rule file: a partial context (omitted fields match anything) and an answer. The first
// matching row of a point wins. Knobs names the api.Tuning knobs the row was derived from (docs/knobs.md, spec 5.3).
type FileRule struct {
	Point  string `json:"point"`
	When   Match  `json:"when"`
	Answer string `json:"answer"`
	Knobs  string `json:"knobs,omitempty"`
	// Vs documents how the row compares with the catalogue's untrained default: "match" (every context the row
	// covers gets the same answer) or "differs" (at least one does not). The rule-file test enforces both.
	Vs string `json:"vs,omitempty"`
}

// Gap is something the rule file still cannot express, in plain words.
type Gap struct {
	Point string `json:"point"`
	Gap   string `json:"gap"`
}

type RuleFile struct {
	Version int        `json:"version"`
	Rules   []FileRule `json:"rules"`
	Gaps    []Gap      `json:"gaps"`
}

// ShippedRuleFile is the T0 to T2 file that encodes today's knob behaviour.
func ShippedRuleFile() (*RuleFile, error) { return ParseRuleFile(shippedRules) }

func ParseRuleFile(data []byte) (*RuleFile, error) {
	var f RuleFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("decide: rule file: %w", err)
	}
	if f.Version != 1 {
		return nil, fmt.Errorf("decide: rule file version %d not supported", f.Version)
	}
	return &f, nil
}

// Validate checks every row against the catalogue; an unknown point, field, value or answer is an error naming it.
func (f *RuleFile) Validate(cat *Catalogue) error {
	for i, r := range f.Rules {
		p, err := cat.Point(r.Point)
		if err != nil {
			return fmt.Errorf("rule file row %d: %w", i+1, err)
		}
		if _, ok := p.answer(r.Answer); !ok {
			return fmt.Errorf("rule file row %d: %q is not an answer of %s", i+1, r.Answer, p.ID)
		}
		if err := cat.validateMatch(p, r.When); err != nil {
			return fmt.Errorf("rule file row %d: %w", i+1, err)
		}
	}
	for _, g := range f.Gaps {
		if _, err := cat.Point(g.Point); err != nil {
			return fmt.Errorf("rule file gap: %w", err)
		}
	}
	return nil
}

// Lookup returns the file's answer for a context, if a row covers it.
func (f *RuleFile) Lookup(ctx Context) (string, bool) {
	for _, r := range f.Rules {
		if r.Point == ctx.Point && r.When.matches(ctx.Fields) {
			return r.Answer, true
		}
	}
	return "", false
}

// Contexts enumerates every context of a point (all values of its key fields, act 1 and normal difficulty, both
// hardcore flags). It is the test universe for the rule-file check.
func (c *Catalogue) Contexts(pointID string) ([]Context, error) {
	p, err := c.Point(pointID)
	if err != nil {
		return nil, err
	}
	names := append([]string{"hc"}, p.ContextKeys...)
	sort.Strings(names[1:])
	fixed := map[string]string{"act": "1", "diff": "normal"}
	out := []map[string]string{fixed}
	for _, n := range names {
		f := c.Fields[n]
		vals := f.Values
		if f.Kind != "enum" {
			vals = f.Labels
		}
		var next []map[string]string
		for _, m := range out {
			for _, v := range vals {
				c := map[string]string{n: v}
				for k, x := range m {
					c[k] = x
				}
				next = append(next, c)
			}
		}
		out = next
	}
	ctxs := make([]Context, len(out))
	for i, m := range out {
		ctxs[i] = Context{Point: pointID, Fields: m, Key: makeKey(pointID, m)}
	}
	return ctxs, nil
}

// PointCheck is the result of comparing the rule file with the catalogue defaults for one point.
type PointCheck struct {
	Total, Covered int
	Mismatch       int // covered contexts where the file and the default disagree
	RowMismatch    map[int]bool
	RowHasMatch    map[int]bool
}

// Check compares the file with the untrained defaults on every context of every point.
func (f *RuleFile) Check(cat *Catalogue) (map[string]*PointCheck, error) {
	out := map[string]*PointCheck{}
	for _, p := range cat.Points {
		pc := &PointCheck{RowMismatch: map[int]bool{}, RowHasMatch: map[int]bool{}}
		out[p.ID] = pc
		ctxs, err := cat.Contexts(p.ID)
		if err != nil {
			return nil, err
		}
		for _, ctx := range ctxs {
			pc.Total++
			row := -1
			for i, r := range f.Rules {
				if r.Point == p.ID && r.When.matches(ctx.Fields) {
					row = i
					break
				}
			}
			if row < 0 {
				continue
			}
			pc.Covered++
			def, err := cat.Default(ctx)
			if err != nil {
				return nil, err
			}
			if def != f.Rules[row].Answer {
				pc.Mismatch++
				pc.RowMismatch[row] = true
			} else {
				pc.RowHasMatch[row] = true
			}
		}
	}
	return out, nil
}

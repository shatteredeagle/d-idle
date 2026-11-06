package fidprobe

import (
	"fmt"
	"math"
	"sort"
)

// Result classes.
const (
	Equal      = "EQUAL"
	KnownDev   = "KNOWN"
	Mismatch   = "MISMATCH"
	Unreadable = "UNREADABLE"
)

// Cell is one compared field.
type Cell struct {
	Key    string `json:"key"`
	Class  string `json:"class"`
	A      string `json:"a"`
	B      string `json:"b"`
	Detail string `json:"detail,omitempty"` // set members that differ, or the relative error
	Row    string `json:"row,omitempty"`    // matrix row for KNOWN
	Reason string `json:"reason,omitempty"` // known reason or the unreadable why
}

// Report is the outcome of one scenario.
type Report struct {
	Scenario    string         `json:"scenario"`
	SourceA     string         `json:"source_a"`
	SourceB     string         `json:"source_b"`
	Cells       []Cell         `json:"cells"`
	Counts      map[string]int `json:"counts"`
	EmptyGroups []string       `json:"unreadable_groups,omitempty"` // listed groups with no field on either side
}

// Diff compares two sheets over the fields whose group is in groups (all groups when empty).
// A field present on one side only, or voided on either side, is UNREADABLE, never EQUAL.
func Diff(a, b *Sheet, groups []string, rules *Rules) *Report {
	want := map[string]bool{}
	for _, g := range groups {
		want[g] = true
	}
	keys := map[string]bool{}
	for k := range a.Fields {
		keys[k] = true
	}
	for k := range b.Fields {
		keys[k] = true
	}
	var order []string
	for k := range keys {
		if len(want) == 0 || want[Group(k)] {
			order = append(order, k)
		}
	}
	sort.Strings(order)
	rep := &Report{SourceA: a.Source, SourceB: b.Source, Counts: map[string]int{}}
	seen := map[string]bool{}
	for _, k := range order {
		seen[Group(k)] = true
		c := compare(k, a.Fields[k], b.Fields[k], a.Fields, b.Fields, rules)
		rep.Cells = append(rep.Cells, c)
		rep.Counts[c.Class]++
	}
	for _, g := range groups {
		if !seen[g] {
			rep.EmptyGroups = append(rep.EmptyGroups, g)
			rep.Counts[Unreadable]++
		}
	}
	return rep
}

func compare(k string, fa, fb Field, ma, mb map[string]Field, rules *Rules) Cell {
	c := Cell{Key: k, A: fa.String(), B: fb.String()}
	_, inA := ma[k]
	_, inB := mb[k]
	if !fa.Readable() || !fb.Readable() {
		c.Class = Unreadable
		switch {
		case !inA:
			c.Reason = "absent on A"
		case !inB:
			c.Reason = "absent on B"
		case fa.Why != "":
			c.Reason = "A: " + fa.Why
		case fb.Why != "":
			c.Reason = "B: " + fb.Why
		}
		return c
	}
	if fa.Kind != fb.Kind {
		c.Class, c.Detail = Mismatch, fmt.Sprintf("kind %s vs %s", fa.Kind, fb.Kind)
		return classify(c, rules)
	}
	switch fa.Kind {
	case KindSet:
		onlyA, onlyB := setDiff(fa.Set, fb.Set)
		if len(onlyA)+len(onlyB) == 0 {
			c.Class = Equal
			return c
		}
		c.Class = Mismatch
		c.Detail = fmt.Sprintf("only A %v; only B %v", onlyA, onlyB)
	default:
		x, y := num(fa), num(fb)
		r := rules.ruleFor(k)
		if r.Rule == "rel" {
			e := relErr(x, y)
			if e <= r.Tol {
				c.Class = Equal
				if e > 0 {
					c.Detail = fmt.Sprintf("rel %.3f%% within %.1f%%", e*100, r.Tol*100)
				}
				return c
			}
			c.Class, c.Detail = Mismatch, fmt.Sprintf("rel %.3f%% over %.1f%%", e*100, r.Tol*100)
		} else if x == y {
			c.Class = Equal
			return c
		} else {
			c.Class, c.Detail = Mismatch, fmt.Sprintf("delta %g", y-x)
		}
	}
	return classify(c, rules)
}

func classify(c Cell, rules *Rules) Cell {
	if k, ok := rules.knownFor(c.Key); ok && c.Class == Mismatch {
		c.Class, c.Row, c.Reason = KnownDev, k.Row, k.Reason
	}
	return c
}

func num(f Field) float64 {
	if f.Kind == KindFloat {
		return f.F
	}
	return float64(f.I)
}

func relErr(a, b float64) float64 {
	m := math.Max(math.Abs(a), math.Abs(b))
	if m == 0 {
		return 0
	}
	return math.Abs(a-b) / m
}

func setDiff(a, b []string) (onlyA, onlyB []string) {
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		switch {
		case j >= len(b) || (i < len(a) && a[i] < b[j]):
			onlyA = append(onlyA, a[i])
			i++
		case i >= len(a) || b[j] < a[i]:
			onlyB = append(onlyB, b[j])
			j++
		default:
			i++
			j++
		}
	}
	return
}

// Pass is true when no field is a MISMATCH (exit code 0).
func (r *Report) Pass() bool { return r.Counts[Mismatch] == 0 }

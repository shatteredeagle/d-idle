package decide

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// The policy preview (card policydryrun, docs/policy-preview.md): replay the RECORDED recent events against the knob
// values a policy option would set and report deltas, only where the recorded events carry what the delta needs. Pure
// and read-only: it writes nothing, changes no knob and never touches the store. A row with no recorded evidence says
// so (Supported false, Reason) and shows no number: a counterfactual that the log cannot support is not guessed.

//go:embed preview_rows.json
var shippedPreviewRows []byte

// PreviewSchema is the version of PreviewResult.
const PreviewSchema = "decide.policy.preview/1"

// Preview event kinds (the same words as the watch away log).
const (
	PvFight  = "close" // one fight; LifePct is the lowest life percent of that fight (the fight telemetry's minlife)
	PvPotion = "potion"
	PvDrop   = "drop"
	PvDeath  = "death"
)

// PreviewEvent is one recorded event, reduced to what a preview reads. T is seconds on the log's time base.
type PreviewEvent struct {
	T       float64
	Kind    string
	LifePct int
	Item    string
	Quality string
	Type    string
}

// previewSpec is one row of preview_rows.json. Kind "lows" reads fight life lows against a drink and a leave knob;
// "pickups" reads recorded pickups against a strictness knob; "none" explains why the row has no preview yet.
type previewSpec struct {
	Kind       string              `json:"kind"`
	DrinkKnob  string              `json:"drinkKnob,omitempty"`
	LeaveKnob  string              `json:"leaveKnob,omitempty"`
	Knob       string              `json:"knob,omitempty"`
	Hide       map[string][]string `json:"hide,omitempty"`       // knob level -> qualities the level no longer picks up
	AlwaysKept []string            `json:"alwaysKept,omitempty"` // item types no level hides
	Reason     string              `json:"reason,omitempty"`
}

type previewFile struct {
	Version   int                    `json:"version"`
	MinEvents int                    `json:"minEvents"`
	Rows      map[string]previewSpec `json:"rows"`
}

var (
	ErrPreviewRow   = errors.New("decide: no preview spec for policy row")
	ErrPreviewKnob  = errors.New("decide: preview needs a knob value that is neither set by the option nor in the baseline")
	ErrPreviewInput = errors.New("decide: bad preview input")
)

// PreviewSpecs is the parsed preview table.
type PreviewSpecs struct{ f previewFile }

// ParsePreviewSpecs reads and validates a preview table; a kind or knob key it does not understand is an error.
func ParsePreviewSpecs(b []byte) (*PreviewSpecs, error) {
	var f previewFile
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("decide: preview table: %w", err)
	}
	if f.Version != 1 || f.MinEvents < 1 {
		return nil, errors.New("decide: preview table: version must be 1 and minEvents at least 1")
	}
	for id, s := range f.Rows {
		switch s.Kind {
		case "lows":
			if s.DrinkKnob == "" || s.LeaveKnob == "" {
				return nil, fmt.Errorf("decide: preview row %q: lows needs drinkKnob and leaveKnob", id)
			}
		case "pickups":
			if s.Knob == "" || len(s.Hide) == 0 {
				return nil, fmt.Errorf("decide: preview row %q: pickups needs knob and hide", id)
			}
		case "none":
			if s.Reason == "" {
				return nil, fmt.Errorf("decide: preview row %q: none needs a reason", id)
			}
		default:
			return nil, fmt.Errorf("decide: preview row %q: unknown kind %q", id, s.Kind)
		}
	}
	return &PreviewSpecs{f}, nil
}

// ShippedPreviewSpecs is the embedded table.
func ShippedPreviewSpecs() (*PreviewSpecs, error) { return ParsePreviewSpecs(shippedPreviewRows) }

// Has reports whether the table has an entry for the row.
func (p *PreviewSpecs) Has(row string) bool { _, ok := p.f.Rows[row]; return ok }

// PreviewInput is everything Preview reads. Baseline gives the knob value that applies when an option sets none (the
// profile default); it is called only for keys the spec names.
type PreviewInput struct {
	Row      *PolicyRow
	Class    string
	Current  string // option id now chosen
	New      string // option id to preview
	Baseline func(key string) (float64, bool)
	Events   []PreviewEvent
	Now      float64
	Window   float64 // seconds to look back
}

// PreviewWindow says what the numbers are based on.
type PreviewWindow struct {
	Seconds float64 `json:"seconds"`
	Events  int     `json:"events"` // events of any kind inside the window
}

// PreviewLine is one reported delta. Delta is nil when the line is a statement without a number.
type PreviewLine struct {
	Key        string `json:"key"`
	Text       string `json:"text"`
	Delta      *int   `json:"delta,omitempty"`
	Based      int    `json:"based"` // events this line counted from
	Basis      string `json:"basis"` // 'based on N fights in the last M minutes'
	Confidence string `json:"confidence"`
	Bound      string `json:"bound,omitempty"` // 'at least', 'at most' or empty (exact count of the recorded events)
}

// FilteredItem is something the new setting would have hidden: listed, never dropped silently (hide but log).
type FilteredItem struct {
	Item    string  `json:"item"`
	Quality string  `json:"quality,omitempty"`
	T       float64 `json:"t"`
	Why     string  `json:"why"`
}

// PreviewResult is the preview of one row change.
type PreviewResult struct {
	Schema        string         `json:"schema"`
	Row           string         `json:"row"`
	From          string         `json:"from"`
	To            string         `json:"to"`
	Supported     bool           `json:"supported"`
	Enough        bool           `json:"enough"`
	Reason        string         `json:"reason,omitempty"` // why there is no preview, or why the numbers are missing
	Window        PreviewWindow  `json:"window"`
	Lines         []PreviewLine  `json:"lines"`
	Filtered      []FilteredItem `json:"filtered"`
	FilteredTotal int            `json:"filteredTotal"`
	Notes         []string       `json:"notes,omitempty"`
}

// MaxFiltered bounds the Filtered list; FilteredTotal always carries the full count.
const MaxFiltered = 50

func minutes(sec float64) string {
	m := int(sec/60 + 0.5)
	if m < 1 {
		m = 1
	}
	if m == 1 {
		return "1 minute"
	}
	return fmt.Sprintf("%d minutes", m)
}

// confidence grades by the number of events a line counted from. A bound (at least, at most) never reaches 'high'.
func confidence(n int, bound bool) string {
	switch {
	case n < 10:
		return "low"
	case n < 30 || bound:
		return "medium"
	}
	return "high"
}

func optionKnob(o *PolicyOption, class, key string, base func(string) (float64, bool)) (float64, error) {
	kv, err := o.KnobsFor(class)
	if err != nil {
		return 0, err
	}
	if v, ok := kv[key]; ok {
		return v, nil
	}
	if base != nil {
		if v, ok := base(key); ok {
			return v, nil
		}
	}
	return 0, fmt.Errorf("%w: knob %q, option %q, class %q", ErrPreviewKnob, key, o.ID, class)
}

func pct(v float64) string { return fmt.Sprintf("%g%%", v) }

// Preview computes the preview. An unknown row or option is an error naming it. A row without evidence returns a
// result with Supported false and a Reason, not an error.
func (sp *PreviewSpecs) Preview(in PreviewInput) (PreviewResult, error) {
	if in.Row == nil || in.Window <= 0 {
		return PreviewResult{}, fmt.Errorf("%w: a row and a positive window are required", ErrPreviewInput)
	}
	spec, ok := sp.f.Rows[in.Row.ID]
	if !ok {
		return PreviewResult{}, fmt.Errorf("%w %q", ErrPreviewRow, in.Row.ID)
	}
	cur, err := in.Row.Option(in.Current)
	if err != nil {
		return PreviewResult{}, err
	}
	nw, err := in.Row.Option(in.New)
	if err != nil {
		return PreviewResult{}, err
	}
	win := in.Window
	if in.Now > 0 && in.Now < win {
		win = in.Now // the log cannot reach back before the run started
	}
	res := PreviewResult{Schema: PreviewSchema, Row: in.Row.ID, From: cur.ID, To: nw.ID, Lines: []PreviewLine{}, Filtered: []FilteredItem{},
		Window: PreviewWindow{Seconds: win}}
	var evs []PreviewEvent
	for _, e := range in.Events {
		if e.T >= in.Now-win && e.T <= in.Now {
			evs = append(evs, e)
		}
	}
	res.Window.Events = len(evs)
	if spec.Kind == "none" {
		res.Reason = "no preview for this row yet: " + spec.Reason
		return res, nil
	}
	res.Supported = true
	if cur.ID == nw.ID {
		res.Reason = "this is the option already chosen"
		res.Enough = true
		return res, nil
	}
	switch spec.Kind {
	case "lows":
		err = sp.previewLows(&res, spec, cur, nw, in, evs, win)
	case "pickups":
		err = sp.previewPickups(&res, spec, cur, nw, in, evs, win)
	}
	return res, err
}

func (sp *PreviewSpecs) notEnough(res *PreviewResult, n int, what string, win float64) {
	res.Reason = fmt.Sprintf("not enough events: %d %s in the last %s (need %d)", n, what, minutes(win), sp.f.MinEvents)
}

// previewLows: fight life lows against the drink and leave lines. The recorded low of a fight was reached under the
// CURRENT lines, so a fight whose low sits between the old and a higher new line would have crossed it; lowering a line
// can only be answered for the fights that were stopped by it, and what followed is unknown.
func (sp *PreviewSpecs) previewLows(res *PreviewResult, spec previewSpec, cur, nw *PolicyOption, in PreviewInput, evs []PreviewEvent, win float64) error {
	var lows []int
	potions := 0
	for _, e := range evs {
		switch {
		case e.Kind == PvFight && e.LifePct > 0:
			lows = append(lows, e.LifePct)
		case e.Kind == PvPotion:
			potions++
		}
	}
	n := len(lows)
	if n < sp.f.MinEvents {
		sp.notEnough(res, n, "fights with a recorded life low", win)
		return nil
	}
	res.Enough = true
	basis := fmt.Sprintf("based on %d fights in the last %s", n, minutes(win))
	between := func(lo, hi float64) int { // lows with lo < low <= hi
		c := 0
		for _, l := range lows {
			if float64(l) > lo && float64(l) <= hi {
				c++
			}
		}
		return c
	}
	for _, l := range []struct {
		knob, key, what string
	}{{spec.LeaveKnob, "leave", "retreat"}, {spec.DrinkKnob, "drink", "drink"}} {
		a, err := optionKnob(cur, in.Class, l.knob, in.Baseline)
		if err != nil {
			return err
		}
		b, err := optionKnob(nw, in.Class, l.knob, in.Baseline)
		if err != nil {
			return err
		}
		pl := PreviewLine{Key: l.key, Based: n, Basis: basis}
		d := 0
		switch {
		case a == b:
			pl.Text = fmt.Sprintf("The %s line stays at %s.", l.what, pct(a))
			pl.Confidence = confidence(n, false)
			res.Lines = append(res.Lines, pl)
			continue
		case b > a:
			d = between(a, b)
			pl.Delta, pl.Bound = &d, "at least"
			if l.key == "leave" {
				pl.Text = fmt.Sprintf("The hero would have retreated at %s instead of %s: %d of the %d fights dipped into that band and would have ended in a retreat.", pct(b), pct(a), d, n)
			} else {
				pl.Text = fmt.Sprintf("The hero would have drunk at %s instead of %s: at least %d more fights with a potion (potions drunk in the window: %d; one fight can use several).", pct(b), pct(a), d, potions)
			}
		default:
			d = -between(b, a)
			pl.Delta, pl.Bound = &d, "at most"
			if l.key == "leave" {
				pl.Text = fmt.Sprintf("The hero would have retreated at %s instead of %s: up to %d of the %d fights that fell to that line would have gone on (what followed is not recorded).", pct(b), pct(a), -d, n)
			} else {
				pl.Text = fmt.Sprintf("The hero would have drunk at %s instead of %s: up to %d fewer fights with a potion (potions drunk in the window: %d).", pct(b), pct(a), -d, potions)
			}
		}
		pl.Confidence = confidence(n, true)
		res.Lines = append(res.Lines, pl)
	}
	res.Notes = append(res.Notes, "Deaths, kills and questions asked are not previewed: the log cannot say what would have happened after a different decision.")
	return nil
}

func hasStr(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// previewPickups: recorded pickups against a pickup-strictness level. A stricter level lists the pickups it would have
// left on the ground (an upper bound: the pickit may still keep a wanted item of that quality). A looser level would pick
// up items the log never saw, so no number is given.
func (sp *PreviewSpecs) previewPickups(res *PreviewResult, spec previewSpec, cur, nw *PolicyOption, in PreviewInput, evs []PreviewEvent, win float64) error {
	var drops []PreviewEvent
	for _, e := range evs {
		if e.Kind == PvDrop && e.Quality != "" {
			drops = append(drops, e)
		}
	}
	n := len(drops)
	if n < sp.f.MinEvents {
		sp.notEnough(res, n, "recorded item pickups", win)
		return nil
	}
	res.Enough = true
	a, err := optionKnob(cur, in.Class, spec.Knob, in.Baseline)
	if err != nil {
		return err
	}
	b, err := optionKnob(nw, in.Class, spec.Knob, in.Baseline)
	if err != nil {
		return err
	}
	basis := fmt.Sprintf("based on %d pickups in the last %s", n, minutes(win))
	hidden := func(level float64, e PreviewEvent) bool {
		if hasStr(spec.AlwaysKept, e.Type) {
			return false
		}
		return hasStr(spec.Hide[fmt.Sprint(int(level))], e.Quality)
	}
	pl := PreviewLine{Key: "pickups", Based: n, Basis: basis}
	switch {
	case a == b:
		pl.Text = fmt.Sprintf("Pickup strictness stays at level %g.", a)
		pl.Confidence = confidence(n, false)
	case b > a:
		var out []FilteredItem
		for _, e := range drops {
			if hidden(b, e) && !hidden(a, e) {
				out = append(out, FilteredItem{Item: e.Item, Quality: e.Quality, T: e.T,
					Why: fmt.Sprintf("%s quality: strictness %g no longer picks it up (the pickit may still have kept it)", e.Quality, b)})
			}
		}
		d := -len(out)
		pl.Delta, pl.Bound = &d, "at most"
		pl.Text = fmt.Sprintf("Up to %d of the %d recorded pickups would have stayed on the ground. Nothing is dropped silently: they are listed below.", len(out), n)
		pl.Confidence = confidence(n, true)
		res.FilteredTotal = len(out)
		sort.SliceStable(out, func(i, j int) bool { return out[i].T > out[j].T })
		if len(out) > MaxFiltered {
			out = out[:MaxFiltered]
		}
		res.Filtered = out
	default:
		pl.Text = "A looser setting would pick up items the log never saw (items left on the ground are not recorded), so there is no number for it."
		pl.Confidence = "none"
	}
	res.Lines = append(res.Lines, pl)
	res.Notes = append(res.Notes, "What a merchant would have paid for these items is not previewed: no sell or gold event is recorded.")
	return nil
}

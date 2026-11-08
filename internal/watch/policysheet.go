package watch

// The chores policy sheet (TASK-128, ADR 0018 item 4, docs/policy-sheet.md): one page that lists the chore decisions with
// today's setting, the default and every allowed value (poor ones too, flagged with their risk). A change is one log line
// of the decide store (Store.SetPolicy) plus, for knob options, an edit through the tuner; nothing else is stored. Each
// row carries an effect line computed from the away events since the change, and the return screen gets a section that
// sets the 'after' window against an equally long 'before' window.

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"mime"
	"net/http"
	"sort"
	"strings"
	"sync"

	"d2lod-bot/internal/data"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/plandata"
)

const policySchema = "watch.policy/1"

// PolicyRoutes are the GET routes of PolicyHandler (viewer); PolicyPostRoute is the POST route (answer listener).
var PolicyRoutes = []string{"/watch/policy", "/api/policy", "/watch/policy.js"}

const PolicyPostRoute = "/api/policy"

var (
	ErrPolicyNoClass = errors.New("watch: policy sheet: the hero's class is not known yet")
	ErrPolicyNoKnobs = errors.New("watch: policy sheet: this option sets tuning knobs and no tuner is connected")
	ErrPolicyRow     = errors.New("watch: unknown policy row")
	ErrPolicyOption  = errors.New("watch: unknown policy option")
	ErrPolicyScope   = errors.New("watch: policy row is not for this class")
)

// KnobSetter edits tuning knobs by key; a nil value resets the knob to the profile value (bot.Tuner.Edit, api.TuneEdit).
// It must be all or nothing.
type KnobSetter func(over map[string]*float64) error

// PolicyChange is one change of a row in this run, on the Hub's time base. It is memory only: a restart has no events
// to compare, so it has no effect lines either (the current option itself persists in the decide store).
type PolicyChange struct {
	Row  string  `json:"row"`
	From string  `json:"from"`
	To   string  `json:"to"`
	T    float64 `json:"t"`
}

// PolicySheet connects the table to the store, the tuner and the event stream.
type PolicySheet struct {
	store  *decide.Store
	cat    *decide.Catalogue
	table  *decide.PolicyTable
	class  func() string // lower-case catalogue class of the hero, "" until known
	now    func() float64
	events *AwayLog
	knobs  KnobSetter

	// starts gives the option ids a class starts on (plandata.Chores); nil reads the embedded table.
	starts func(class string) (map[string]string, error)

	mu      sync.Mutex
	changes []PolicyChange
	started map[string]bool // classes whose starting options were applied
	// reapplyClass: Reapply ran before the class was known and left the class-specific knobs for EnsureStart.
	reapplyClass bool
}

// NewPolicySheet checks that every effect counter of the table exists; an unknown one is an error naming it.
func NewPolicySheet(store *decide.Store, cat *decide.Catalogue, table *decide.PolicyTable, class func() string, now func() float64, events *AwayLog, knobs KnobSetter) (*PolicySheet, error) {
	if store == nil || cat == nil || table == nil || class == nil || now == nil {
		return nil, errors.New("watch: policy sheet needs a store, a catalogue, a table, a class and a clock")
	}
	for _, r := range table.Rows {
		for _, c := range r.Effect {
			if _, ok := policyCounters[c.Counter]; !ok {
				return nil, fmt.Errorf("watch: policy row %q: unknown effect counter %q", r.ID, c.Counter)
			}
		}
	}
	return &PolicySheet{store: store, cat: cat, table: table, class: class, now: now, events: events, knobs: knobs}, nil
}

func (p *PolicySheet) rows() (string, []*decide.PolicyRow, error) {
	c, rows, err := p.classRows()
	if err != nil {
		return c, rows, err
	}
	return c, rows, p.applyStart(c, rows)
}

func (p *PolicySheet) classRows() (string, []*decide.PolicyRow, error) {
	c := strings.ToLower(p.class())
	if c == "" {
		return "", nil, ErrPolicyNoClass
	}
	rows, err := p.table.RowsFor(p.cat, c)
	return c, rows, err
}

// applyStart puts the class's starting options (internal/plandata chores-defaults.json, card classchores) on rows the
// player never set, once per class. It is a start, not a change: no PolicyChange is recorded, and a row the player
// already set (the store holds an option) is left alone. A row or option id the table does not have is an error
// naming it; a start equal to the row's default does nothing.
func (p *PolicySheet) applyStart(class string, rows []*decide.PolicyRow) error {
	p.mu.Lock()
	if p.started[class] {
		p.mu.Unlock()
		return nil
	}
	p.mu.Unlock()
	get := p.starts
	if get == nil {
		get = func(c string) (map[string]string, error) {
			ch, err := plandata.Chores(c)
			return ch.Start, err
		}
	}
	start, err := get(class)
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(start))
	for id := range start {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		var row *decide.PolicyRow
		for _, r := range rows {
			if r.ID == id {
				row = r
			}
		}
		if row == nil {
			return fmt.Errorf("watch: chore start of class %q names row %q, which the sheet does not have for it", class, id)
		}
		opt, err := row.Option(start[id])
		if err != nil {
			return fmt.Errorf("watch: chore start of class %q: %w", class, err)
		}
		if opt.ID == row.Default || p.store.PolicyOption(row.ID) != "" {
			continue
		}
		if _, _, err := p.set(row, opt, false); err != nil {
			return fmt.Errorf("watch: chore start %s/%s of class %q: %w", row.ID, opt.ID, class, err)
		}
	}
	p.mu.Lock()
	if p.started == nil {
		p.started = map[string]bool{}
	}
	p.started[class] = true
	p.mu.Unlock()
	return nil
}

func (p *PolicySheet) current(r *decide.PolicyRow) string {
	if o := p.store.PolicyOption(r.ID); o != "" {
		return o
	}
	return r.Default
}

// Set changes a row. It returns the change and whether anything changed (the same option again is not a change). Knobs
// first (all or nothing), then the log line; a failed log line puts the knobs back.
func (p *PolicySheet) Set(rowID, optID string) (PolicyChange, bool, error) {
	_, rows, err := p.rows()
	if err != nil {
		return PolicyChange{}, false, err
	}
	row, err := p.table.Row(rowID)
	if err != nil {
		return PolicyChange{}, false, fmt.Errorf("%w %q", ErrPolicyRow, safeText(rowID, 40))
	}
	in := false
	for _, r := range rows {
		in = in || r == row
	}
	if !in {
		return PolicyChange{}, false, fmt.Errorf("%w: %q, class %q", ErrPolicyScope, row.ID, strings.ToLower(p.class()))
	}
	opt, err := row.Option(optID)
	if err != nil {
		return PolicyChange{}, false, fmt.Errorf("%w %q of row %q", ErrPolicyOption, safeText(optID, 40), row.ID)
	}
	return p.set(row, opt, true)
}

// set applies an option; record false (a class start) leaves out the change record the return screen shows.
func (p *PolicySheet) set(row *decide.PolicyRow, opt *decide.PolicyOption, record bool) (PolicyChange, bool, error) {
	from := p.current(row)
	if from == opt.ID {
		return PolicyChange{Row: row.ID, From: from, To: opt.ID}, false, nil
	}
	prev, _ := row.Option(from)
	over, err := knobEdit(prev, opt, strings.ToLower(p.class()))
	if err != nil {
		return PolicyChange{}, false, fmt.Errorf("watch: policy %s/%s: %w", row.ID, opt.ID, err)
	}
	if len(over) > 0 {
		if p.knobs == nil {
			return PolicyChange{}, false, fmt.Errorf("%w (row %q, option %q)", ErrPolicyNoKnobs, row.ID, opt.ID)
		}
		if err := p.knobs(over); err != nil {
			return PolicyChange{}, false, fmt.Errorf("watch: policy %s/%s: knobs: %w", row.ID, opt.ID, err)
		}
	}
	if err := p.store.SetPolicy(row.ID, opt.ID, opt.Rules); err != nil {
		if len(over) > 0 {
			back, _ := knobEdit(opt, prev, strings.ToLower(p.class()))
			if e := p.knobs(back); e != nil {
				err = fmt.Errorf("%w (and the knobs could not be put back: %v)", err, e)
			}
		}
		return PolicyChange{}, false, err
	}
	ch := PolicyChange{Row: row.ID, From: from, To: opt.ID, T: p.now()}
	if record {
		p.mu.Lock()
		p.changes = append(p.changes, ch)
		p.mu.Unlock()
	}
	return ch, true, nil
}

// knobEdit is the edit that takes the knobs from option a to option b: b's values, and a reset for what only a set.
func knobEdit(a, b *decide.PolicyOption, class string) (map[string]*float64, error) {
	over := map[string]*float64{}
	if a != nil {
		ak, err := a.KnobsFor(class)
		if err != nil {
			return nil, err
		}
		for k := range ak {
			over[k] = nil
		}
	}
	bk, err := b.KnobsFor(class)
	if err != nil {
		return nil, err
	}
	for k, v := range bk {
		v := v
		over[k] = &v
	}
	return over, nil
}

// Reapply sets the knobs of every stored option again (the store is the truth; the tuner keeps its own copy and may
// have lost it). Rules need nothing: the store already answers with them. Call it once after the store is open.
func (p *PolicySheet) Reapply() error {
	if p.knobs == nil {
		return nil
	}
	class := strings.ToLower(p.class())
	over := map[string]*float64{}
	for _, pv := range p.store.Policies() {
		row, err := p.table.Row(pv.Row)
		if err != nil {
			return err // a stored row the table no longer has: say so, do not guess
		}
		opt, err := row.Option(pv.Option)
		if err != nil {
			return err
		}
		if class == "" && len(opt.ClassKnobs) > 0 {
			// The class is not known yet (the first snapshot is still to come): the class-specific knobs follow in
			// EnsureStart, which runs when the class shows up.
			p.mu.Lock()
			p.reapplyClass = true
			p.mu.Unlock()
			for k, v := range opt.Knobs {
				v := v
				over[k] = &v
			}
			continue
		}
		kv, err := opt.KnobsFor(class)
		if err != nil {
			return err
		}
		for k, v := range kv {
			v := v
			over[k] = &v
		}
	}
	if len(over) == 0 {
		return nil
	}
	return p.knobs(over)
}

// EnsureStart settles what needs the hero's class: the class's starting options and the class-specific knobs of the
// stored options. Call it when the class becomes known. Before that it does nothing (ErrPolicyNoClass is not an error
// here).
func (p *PolicySheet) EnsureStart() error {
	_, _, err := p.rows()
	if errors.Is(err, ErrPolicyNoClass) {
		return nil
	}
	if err != nil {
		return err
	}
	p.mu.Lock()
	again := p.reapplyClass
	p.reapplyClass = false
	p.mu.Unlock()
	if again {
		return p.Reapply()
	}
	return nil
}

// SetStarts replaces the source of the class's starting options (nil: the embedded table). The bot sets one that gives
// nothing when a named run (D2BOT_RUN, a gate or soak) is on, so a measured run keeps today's knobs.
func (p *PolicySheet) SetStarts(f func(class string) (map[string]string, error)) { p.starts = f }

// Changes lists this run's changes, oldest first.
func (p *PolicySheet) Changes() []PolicyChange {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]PolicyChange(nil), p.changes...)
}

func (p *PolicySheet) lastChange(row string) *PolicyChange {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i := len(p.changes) - 1; i >= 0; i-- {
		if p.changes[i].Row == row {
			c := p.changes[i]
			return &c
		}
	}
	return nil
}

// ---- effect counters -------------------------------------------------------------------------------------------------

// counterFn counts one thing in the events of [from, to]. ok false: no event carries it (the line says 'not reported').
// all is the whole log, for counters that look back before the window.
type counterFn func(all []AwayEvent, from, to float64) (n int, ok bool)

const dryLookback = 120.0 // seconds before a death in which a potion counts as 'drunk just before' (U: my choice)

func inWin(e AwayEvent, from, to float64) bool { return e.T >= from && e.T <= to }

func countWhere(all []AwayEvent, from, to float64, pred func(AwayEvent) bool) (int, bool) {
	n := 0
	for _, e := range all {
		if inWin(e, from, to) && pred(e) {
			n++
		}
	}
	return n, true
}

func potionIs(sub string) func(AwayEvent) bool {
	return func(e AwayEvent) bool {
		return e.Kind == AwayPotion && strings.Contains(strings.ToLower(e.Potion), sub)
	}
}

func kindIs(k string) func(AwayEvent) bool { return func(e AwayEvent) bool { return e.Kind == k } }

// policyCounters is the small mapping from a counter name (used by policy_chores.json) to the way it is read from the
// event stream. Potion kinds are told by the item name (S: 'Mana', 'Healing', 'Rejuvenation' in the potion's name; U for
// a modded name). The counters named U in the table are guesses at a cause and say so on the page.
var policyCounters = map[string]counterFn{
	"potions":        func(a []AwayEvent, f, t float64) (int, bool) { return countWhere(a, f, t, kindIs(AwayPotion)) },
	"potions_mana":   func(a []AwayEvent, f, t float64) (int, bool) { return countWhere(a, f, t, potionIs("mana")) },
	"potions_health": func(a []AwayEvent, f, t float64) (int, bool) { return countWhere(a, f, t, potionIs("heal")) },
	"potions_rejuv":  func(a []AwayEvent, f, t float64) (int, bool) { return countWhere(a, f, t, potionIs("rejuv")) },
	"deaths":         func(a []AwayEvent, f, t float64) (int, bool) { return countWhere(a, f, t, kindIs(AwayDeath)) },
	"kills":          func(a []AwayEvent, f, t float64) (int, bool) { return countWhere(a, f, t, kindIs(AwayKill)) },
	"closecalls":     func(a []AwayEvent, f, t float64) (int, bool) { return countWhere(a, f, t, kindIs(AwayClose)) },
	"trips": func(a []AwayEvent, f, t float64) (int, bool) {
		return countWhere(a, f, t, func(e AwayEvent) bool { return e.Kind == AwayArea && data.IsTown(e.AreaID) })
	},
	"whites": func(a []AwayEvent, f, t float64) (int, bool) {
		return countWhere(a, f, t, func(e AwayEvent) bool { q, _ := rankOf(e); return e.Kind == AwayDrop && q == "normal" })
	},
	"drops": func(a []AwayEvent, f, t float64) (int, bool) {
		return countWhere(a, f, t, func(e AwayEvent) bool {
			q, r := rankOf(e)
			return e.Kind == AwayDrop && q != "" && r >= dropRank["rare"]
		})
	},
	"deaths_dry": func(all []AwayEvent, from, to float64) (int, bool) {
		n := 0
		for _, d := range all {
			if d.Kind != AwayDeath || !inWin(d, from, to) {
				continue
			}
			dry := true
			for _, e := range all {
				if e.Kind == AwayPotion && e.T <= d.T && d.T-e.T <= dryLookback {
					dry = false
					break
				}
			}
			if dry {
				n++
			}
		}
		return n, true
	},
	"gold_in": func(a []AwayEvent, f, t float64) (int, bool) {
		n, seen := 0, false
		for _, e := range a {
			if e.Kind == AwayGold {
				seen = true
				if inWin(e, f, t) {
					n += int(e.Gold)
				}
			}
		}
		return n, seen
	},
	// Nothing emits a repair, a purchase or a durability reading; the counter exists so the line says so.
	"repairs": func([]AwayEvent, float64, float64) (int, bool) { return 0, false },
}

// CounterValue is one counter of an effect line.
type CounterValue struct {
	Counter   string `json:"counter"`
	Label     string `json:"label"`
	N         int    `json:"n"`
	Reported  bool   `json:"reported"`
	Heuristic bool   `json:"heuristic,omitempty"`
}

func (p *PolicySheet) count(row *decide.PolicyRow, from, to float64) []CounterValue {
	all := p.events.Events()
	out := make([]CounterValue, 0, len(row.Effect))
	for _, c := range row.Effect {
		n, ok := policyCounters[c.Counter](all, from, to)
		out = append(out, CounterValue{Counter: c.Counter, Label: c.Label, N: n, Reported: ok, Heuristic: c.Heuristic})
	}
	return out
}

func counterText(cs []CounterValue) string {
	parts := make([]string, len(cs))
	for i, c := range cs {
		v := fmt.Sprint(c.N)
		if !c.Reported {
			v = "not reported"
		}
		parts[i] = c.Label + ": " + v
		if c.Heuristic {
			parts[i] += " (U)"
		}
	}
	return strings.Join(parts, ", ")
}

// window returns the equally long after and before windows of a change, as seconds from the change, or ok false with the
// reason. The after window runs from the change; the before window ends at it; both are the shorter of the time since
// the change and the history before it.
func (p *PolicySheet) window(ch PolicyChange) (win float64, why string) {
	now := p.now()
	since := now - ch.T
	if since <= 0 {
		return 0, "just changed"
	}
	evs := p.events.Events()
	if len(evs) == 0 {
		return 0, "no events recorded yet"
	}
	first := evs[0].T
	for _, e := range evs {
		first = min(first, e.T)
	}
	hist := ch.T - first
	if hist < 60 {
		return 0, "no history before the change to compare"
	}
	return min(since, hist), ""
}

// ---- view ------------------------------------------------------------------------------------------------------------

type PolicyOptionView struct {
	ID         string   `json:"id"`
	Label      string   `json:"label"`
	Current    bool     `json:"current"`
	Default    bool     `json:"default,omitempty"`
	Suboptimal bool     `json:"suboptimal,omitempty"`
	Risk       string   `json:"risk,omitempty"`
	Says       string   `json:"says,omitempty"`    // the option in plain words
	Applies    string   `json:"applies,omitempty"` // part, record or empty (fully wired)
	Note       string   `json:"note,omitempty"`
	Rules      []string `json:"rules,omitempty"`
	Knobs      []string `json:"knobs,omitempty"`
}

type PolicyEffectView struct {
	Since    float64        `json:"since"`
	Option   string         `json:"option"`
	Text     string         `json:"text"`
	Counters []CounterValue `json:"counters"`
}

type PolicyRowView struct {
	ID      string             `json:"id"`
	Label   string             `json:"label"`
	Help    string             `json:"help"`
	Current string             `json:"current"`
	Default string             `json:"default"`
	Changed bool               `json:"changed"` // not the default
	Options []PolicyOptionView `json:"options"`
	Effect  *PolicyEffectView  `json:"effect,omitempty"`
	// EffectNote says why there is no effect line (never changed this run, set before this run).
	EffectNote string `json:"effectNote,omitempty"`
}

type PolicyView struct {
	Schema string          `json:"schema"`
	Class  string          `json:"class"`
	Now    float64         `json:"now"`
	Rows   []PolicyRowView `json:"rows"`
	Legend string          `json:"legend"`
}

const policyLegend = "(U) marks a counter whose link to the setting is a guess, not a measurement. 'Not reported' means no event carries it yet."

// View is the sheet as data (GET /api/policy and the input of the HTML).
func (p *PolicySheet) View() (PolicyView, error) {
	class, rows, err := p.rows()
	if err != nil {
		return PolicyView{}, err
	}
	now := p.now()
	v := PolicyView{Schema: policySchema, Class: class, Now: now, Rows: []PolicyRowView{}, Legend: policyLegend}
	for _, r := range rows {
		cur := p.current(r)
		rv := PolicyRowView{ID: r.ID, Label: r.Label, Help: r.Help, Current: cur, Default: r.Default, Changed: cur != r.Default, Options: []PolicyOptionView{}}
		for _, o := range r.Options {
			ov := PolicyOptionView{ID: o.ID, Label: o.Label, Current: o.ID == cur, Default: o.ID == r.Default, Suboptimal: o.Suboptimal,
				Risk: o.Risk, Applies: o.Applies, Note: o.Note}
			for _, ru := range o.Rules {
				ov.Rules = append(ov.Rules, p.store.PlainRule(ru))
			}
			kv, kerr := o.KnobsFor(class)
			if kerr != nil {
				return PolicyView{}, kerr
			}
			ov.Says = o.Says
			ks := make([]string, 0, len(kv))
			for k, val := range kv {
				ks = append(ks, fmt.Sprintf("%s = %v", k, val))
			}
			sort.Strings(ks)
			ov.Knobs = ks
			rv.Options = append(rv.Options, ov)
		}
		switch ch := p.lastChange(r.ID); {
		case ch != nil:
			opt, _ := r.Option(ch.To)
			ev := &PolicyEffectView{Since: ch.T, Option: ch.To, Counters: p.count(r, ch.T, now)}
			ev.Text = fmt.Sprintf("Since you changed this to \"%s\" (%s ago): %s.", opt.Label, fmtDur(now-ch.T), counterText(ev.Counters))
			rv.Effect = ev
		case rv.Changed:
			rv.EffectNote = "Set before this run: there are no events to compare yet."
		default:
			rv.EffectNote = "Not changed. This row follows today's behaviour."
		}
		v.Rows = append(v.Rows, rv)
	}
	return v, nil
}

// ---- the return screen section ---------------------------------------------------------------------------------------

// PolicySectionID is the id the section registers under.
const PolicySectionID = "policy"

const policyCardWindow = 24 * 3600.0 // changes older than this stop being shown as cards

// maxPolicyCards bounds the section.
const maxPolicyCards = 5

// Section is the return screen's policy section: one before/after card per recent change, newest first, with a 'change it
// back' button while the row still has the option the change set. Hidden when there is nothing to show.
func (p *PolicySheet) Section() ReturnSection {
	return SectionFunc(func(*ReturnData) *SectionView {
		_, rows, err := p.rows()
		if err != nil {
			return nil
		}
		byID := map[string]*decide.PolicyRow{}
		for _, r := range rows {
			byID[r.ID] = r
		}
		now := p.now()
		chs := p.Changes()
		sv := &SectionView{ID: PolicySectionID, Title: "Policy changes", After: true}
		for i := len(chs) - 1; i >= 0 && len(sv.Lines) < maxPolicyCards; i-- {
			ch := chs[i]
			row := byID[ch.Row]
			if row == nil || now-ch.T > policyCardWindow {
				continue
			}
			to, _ := row.Option(ch.To)
			from, _ := row.Option(ch.From)
			head := fmt.Sprintf("%s: now \"%s\" (was \"%s\").", row.Label, to.Label, from.Label)
			if to.Says != "" {
				head += " " + to.Says
			}
			win, why := p.window(ch)
			if why != "" {
				sv.Lines = append(sv.Lines, head+" No comparison yet: "+why+".")
			} else {
				after := p.count(row, ch.T, ch.T+win)
				before := p.count(row, ch.T-win, ch.T)
				sv.Lines = append(sv.Lines, fmt.Sprintf("%s After, %s: %s. Before, %s: %s.", head, fmtDur(win), counterText(after), fmtDur(win), counterText(before)))
			}
			if p.current(row) == ch.To {
				sv.Buttons = append(sv.Buttons, SectionButton{Label: "Change it back: " + row.Label, Row: row.ID, Option: ch.From})
			}
		}
		if len(sv.Lines) == 0 {
			return nil
		}
		sv.Lines = append(sv.Lines, policyLegend)
		return sv
	})
}

// ---- HTTP ------------------------------------------------------------------------------------------------------------

const policyCSP = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self' http: https:; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

// PolicyHandler serves the page, its JSON and its script, read-only (mount the three PolicyRoutes on the viewer).
func (p *PolicySheet) PolicyHandler() http.Handler {
	m := http.NewServeMux()
	hdr := func(w http.ResponseWriter, ctype string) {
		w.Header().Set("Content-Type", ctype)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", policyCSP)
	}
	m.HandleFunc("/watch/policy", func(w http.ResponseWriter, r *http.Request) {
		v, err := p.View()
		if err != nil {
			hdr(w, "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			io.WriteString(w, err.Error())
			return
		}
		hdr(w, "text/html; charset=utf-8")
		w.Write(RenderPolicy(v))
	})
	m.HandleFunc("/api/policy", func(w http.ResponseWriter, r *http.Request) {
		v, err := p.View()
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, policyReply{Code: "class", Error: err.Error()})
			return
		}
		writeJSON(w, 200, v)
	})
	m.HandleFunc("/watch/policy.js", func(w http.ResponseWriter, r *http.Request) {
		hdr(w, "text/javascript; charset=utf-8")
		io.WriteString(w, policyJS)
	})
	return readOnly(m)
}

// decodeSmall reads one small JSON object with known fields and nothing after it.
func decodeSmall(r *http.Request, v any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, maxBody+1))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil || dec.More() {
		return errors.New("bad body")
	}
	return nil
}

type policyReply struct {
	OK      bool     `json:"ok"`
	Changed bool     `json:"changed,omitempty"`
	Code    string   `json:"code,omitempty"`
	Error   string   `json:"error,omitempty"`
	Allowed []string `json:"allowed,omitempty"`
}

type policyBody struct {
	Row    string `json:"row"`
	Option string `json:"option"`
}

// policyGate runs the checks every POST route of the sheet shares (origin list, token, JSON only, small body with known
// fields) and returns the decoded body. When ok is false the reply is already written.
func policyGate(o AnswerOptions, w http.ResponseWriter, r *http.Request, dst any) bool {
	origin := r.Header.Get("Origin")
	allowed := origin != "" && containsStr(o.Origins, origin)
	if allowed {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Vary", "Origin")
	}
	fail := func(code int, c, msg string) {
		writeJSON(w, code, policyReply{Code: c, Error: msg})
	}
	switch {
	case r.Method == http.MethodOptions && allowed:
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusNoContent)
		return false
	case r.Method != http.MethodPost:
		w.Header().Set("Allow", "POST, OPTIONS")
		fail(http.StatusMethodNotAllowed, "method", "POST only")
		return false
	case origin != "" && !allowed:
		fail(http.StatusForbidden, "origin", "origin not allowed")
		return false
	case !o.tokenOK(r):
		fail(http.StatusUnauthorized, "token", "unauthorized")
		return false
	}
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		fail(http.StatusUnsupportedMediaType, "content-type", "send application/json")
		return false
	}
	if err := decodeSmall(r, dst); err != nil {
		fail(http.StatusBadRequest, "body", "body must be one small JSON object with known fields")
		return false
	}
	return true
}

// PolicyAnswerHandler is the write side: POST /api/policy {row, option}. Same gate as the other answer routes (origin
// list, token, JSON only, small body).
func (p *PolicySheet) PolicyAnswerHandler(o AnswerOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b policyBody
		if !policyGate(o, w, r, &b) {
			return
		}
		fail := func(code int, c, msg string, opts ...string) {
			writeJSON(w, code, policyReply{Code: c, Error: msg, Allowed: opts})
		}
		_, changed, err := p.Set(b.Row, b.Option)
		switch {
		case err == nil:
			writeJSON(w, 200, policyReply{OK: true, Changed: changed})
		case errors.Is(err, ErrPolicyRow):
			fail(http.StatusNotFound, "row", err.Error())
		case errors.Is(err, ErrPolicyOption):
			var opts []string
			if row, e := p.table.Row(b.Row); e == nil {
				for _, op := range row.Options {
					opts = append(opts, op.ID)
				}
			}
			fail(http.StatusBadRequest, "option", err.Error(), opts...)
		case errors.Is(err, ErrPolicyScope):
			fail(http.StatusBadRequest, "scope", err.Error())
		case errors.Is(err, ErrPolicyNoClass):
			fail(http.StatusServiceUnavailable, "class", err.Error())
		case errors.Is(err, ErrPolicyNoKnobs):
			fail(http.StatusNotImplemented, "knobs", err.Error())
		default:
			fail(http.StatusInternalServerError, "internal", "could not apply the change")
		}
	})
}

// ---- page ------------------------------------------------------------------------------------------------------------

var policyTmpl = template.Must(template.New("policy").Parse(`<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><title>d.idle - policy sheet</title>
<meta name="viewport" content="width=device-width,initial-scale=1"><meta name="color-scheme" content="light dark">
<link rel="icon" href="data:,"><link rel="stylesheet" href="/watch/app.css"><link rel="stylesheet" href="/watch/return.css">
</head><body>
<header class="top"><div class="brand"><span class="brand-mark" aria-hidden="true"></span><h1>d.idle</h1></div>
<a class="btn" href="/watch/return">While you were away</a> <a class="btn primary" href="/watch?live=1">Live screen</a></header>
<main class="ret" id="policy">
<section class="card" aria-labelledby="p-head"><h2 id="p-head">Policy sheet ({{.Class}})</h2>
<p>How the hero handles its chores. Set it once, change it any time. Poor choices are allowed: try them and read what they cost.</p>
<p class="hint">{{.Legend}}</p></section>
{{range .Rows}}<section class="card" data-row="{{.ID}}" aria-labelledby="row-{{.ID}}"><h2 id="row-{{.ID}}">{{.Label}}</h2><p>{{.Help}}</p>
<div class="opts">{{$row := .ID}}{{range .Options}}<div class="opt{{if .Current}} cur{{end}}">
<button type="button" class="btn small{{if .Current}} primary{{end}}" data-act="policy" data-row="{{$row}}" data-option="{{.ID}}" {{if .Current}}aria-pressed="true"{{else}}aria-pressed="false"{{end}}>{{.Label}}</button>
{{if .Says}}<span class="hint">{{.Says}}</span>{{end}}{{if .Default}}<span class="badge">default</span>{{end}}{{if .Suboptimal}}<span class="badge">your call</span> <span class="hint">{{.Risk}}</span>{{end}}
{{if .Applies}}<span class="badge">{{if eq .Applies "record"}}not wired yet{{else}}partly wired{{end}}</span> <span class="hint">{{.Note}}</span>{{end}}
</div>{{end}}</div>
<div class="preview" data-preview="{{.ID}}" hidden></div>
{{if .Effect}}<p class="effect">{{.Effect.Text}}</p>{{else}}<p class="hint">{{.EffectNote}}</p>{{end}}
</section>{{end}}
</main>
<p id="msg" class="hint" role="status"></p>
<script src="/watch/policy.js"></script>
</body></html>`))

// RenderPolicy is the sheet as server-rendered HTML.
func RenderPolicy(v PolicyView) []byte {
	var b strings.Builder
	if err := policyTmpl.Execute(&b, v); err != nil {
		return []byte("render error")
	}
	return []byte(b.String())
}

// policyJS posts a change to the answer route (another port, like the return screen). All text via textContent.
const policyJS = `(function(){'use strict';
var msg=document.getElementById('msg');
function say(t){msg.textContent=t||'';}
function base(){return fetch('/api/config',{cache:'no-store'}).then(function(r){return r.json();}).then(function(c){
 if(c.answerBase)return c.answerBase;return location.protocol+'//'+location.hostname+':'+c.answerPort;});}
function post(path,body){return base().then(function(u){return fetch(u+path,{method:'POST',credentials:'include',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});}).then(function(r){return r.json();});}
function el(tag,cls,text){var n=document.createElement(tag);if(cls)n.className=cls;if(text!==undefined)n.textContent=text;return n;}
function box(row){return document.querySelector('[data-preview="'+row+'"]');}
function close(row){var b=box(row);if(b){b.hidden=true;b.textContent='';}}
function save(row,option){post('/api/policy',{row:row,option:option}).then(function(j){
 say(j.ok?(j.changed?'Changed':'Already set'):(j.error||'failed'));if(j.ok&&j.changed)setTimeout(function(){location.reload();},500);})
 .catch(function(){say('the answer route is not reachable');});}
function render(row,option,label,j){var b=box(row);b.textContent='';b.hidden=false;
 b.appendChild(el('h3','','Preview: '+label));
 if(!j||j.error){b.appendChild(el('p','hint',(j&&j.error)||'The preview is not available.'));}
 else if(!j.supported){b.appendChild(el('p','hint',j.reason));}
 else{
  if(j.reason)b.appendChild(el('p','hint',j.reason));
  (j.lines||[]).forEach(function(l){var p=el('p','effect',l.text);
   p.appendChild(el('span','hint',' ('+l.basis+'; confidence '+l.confidence+')'));b.appendChild(p);});
  if(j.filteredTotal>0){b.appendChild(el('p','hint','Filtered out ('+j.filteredTotal+'), shown here and not lost:'));
   var ul=el('ul','');(j.filtered||[]).forEach(function(f){ul.appendChild(el('li','',f.item+' ('+f.quality+'): '+f.why));});
   if(j.filteredTotal>(j.filtered||[]).length)ul.appendChild(el('li','','and '+(j.filteredTotal-j.filtered.length)+' more'));b.appendChild(ul);}
  (j.notes||[]).forEach(function(n){b.appendChild(el('p','hint',n));});
 }
 var sv=el('button','btn small primary','Save');sv.type='button';sv.onclick=function(){save(row,option);};
 var cn=el('button','btn small','Cancel');cn.type='button';cn.onclick=function(){close(row);say('');};
 b.appendChild(sv);b.appendChild(cn);}
document.addEventListener('click',function(e){var b=e.target.closest('button[data-act="policy"]');if(!b||b.disabled)return;
 var row=b.dataset.row,option=b.dataset.option;if(b.getAttribute('aria-pressed')==='true')return;
 say('Computing the preview...');
 post('/api/policy/preview',{row:row,option:option}).then(function(j){say('');render(row,option,b.textContent,j);})
 .catch(function(){say('');render(row,option,b.textContent,{error:'The preview route is not reachable; you can still save.'});});});
})();`

// Now is the sheet's clock in seconds (the same one the counters use).
func (p *PolicySheet) Now() float64 { return p.now() }

// Counters is the counter text of a row over [from, to] seconds, for the Pact cards. ok is false for a row the table
// lacks or one with no counters.
func (p *PolicySheet) Counters(rowID string, from, to float64) (text string, ok bool) {
	row, err := p.table.Row(rowID)
	if err != nil || len(row.Effect) == 0 {
		return "", false
	}
	return counterText(p.count(row, from, to)), true
}

// CurrentOption is the option now chosen for a row (the row default when never set); "" for a row the table lacks.
func (p *PolicySheet) CurrentOption(rowID string) string {
	row, err := p.table.Row(rowID)
	if err != nil {
		return ""
	}
	return p.current(row)
}

// DefaultOption is the default option id of a row, "" for a row the table lacks.
func (p *PolicySheet) DefaultOption(rowID string) string {
	row, err := p.table.Row(rowID)
	if err != nil {
		return ""
	}
	return row.Default
}

// SetByPact is Set for a Pact (TASK-167): the same knobs, rules and log line, but no recorded change, so the return
// screen offers no "change it back" card for something a Pact the player chose is holding in place.
func (p *PolicySheet) SetByPact(rowID, optID string) error {
	_, rows, err := p.rows()
	if err != nil {
		return err
	}
	row, err := p.table.Row(rowID)
	if err != nil {
		return fmt.Errorf("%w %q", ErrPolicyRow, safeText(rowID, 40))
	}
	for _, r := range rows {
		if r == row {
			opt, err := row.Option(optID)
			if err != nil {
				return fmt.Errorf("%w %q of row %q", ErrPolicyOption, safeText(optID, 40), row.ID)
			}
			_, _, err = p.set(row, opt, false)
			return err
		}
	}
	return fmt.Errorf("%w: %q, class %q", ErrPolicyScope, row.ID, strings.ToLower(p.class()))
}

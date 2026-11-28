// Package arbiter is a pure, deterministic intent arbiter (docs/arbiter.md).
// Sources propose at most one Intent per tick; Decide picks one with strict
// priority classes, leases, a travel budget, exponential back-off and town
// batching, and explains every choice. No goroutines, no real clock, no I/O:
// time and the world come in through View. Sources must not read each other's
// state: they see only the world facts in View and the previous Decision.
package arbiter

import (
	"fmt"
	"sort"
	"strings"
)

// Class is the strict priority order, low to high.
type Class int

const (
	Filler Class = iota
	Objective
	Upkeep
	Player
	Reflex
)

func (c Class) String() string {
	return [...]string{"filler", "objective", "upkeep", "player", "reflex"}[c]
}

const (
	Town = "town"
	Any  = "*" // the intent can run where the hero is
)

type Intent struct {
	Source    string
	Kind      string
	Class     Class
	Urgency   float64  // 0..1, only compared inside a class
	NeedsArea string   // Town, a field area id, or Any
	EstCost   float64  // virtual seconds, a tie-break and part of the trace
	Reason    string   // why the source wants this, for the watch UI
	Resumable bool     // if preempted by a higher class, continue it afterwards
	Steps     []string // what the executor should do, in order
	// TTL makes the intent TEMPORARY: posted with Post or Refused, it is proposed for TTL virtual seconds, or until the
	// condition in Until holds (outcome-rule syntax over View.Facts, "potions>=potions_min"), whichever is first.
	TTL   float64
	Until string
	// Cause names which of the source's reasons this is (a short stable word); a failure backs off (source, kind, cause)
	// only, so a declined "plan would buy" does not silence the same source's other reasons.
	Cause string
	// NoBackoff: a failure never backs this intent off (a running town trip, emergency chores).
	NoBackoff bool
}

func (i Intent) Key() string { return i.Source + "/" + i.Kind }

func (i Intent) backoffKey() string { return i.Key() + "#" + i.Cause }

// backsOff says a failure of the intent may silence it: never a player order, a reflex or a NoBackoff intent.
func (i Intent) backsOff() bool { return i.Class < Player && !i.NoBackoff }

// View is everything the arbiter and its sources may look at. Progress is a
// monotonic counter the world bumps on real progress: a kill, XP, a quest fact
// changing, gold or belt gained.
type View struct {
	Now      float64 // virtual seconds
	Area     string
	Progress uint64
	Facts    map[string]int
	// ObjectiveArea is where the current objective must be done ("" when unknown or anywhere).
	ObjectiveArea string
	// Prev is the arbiter's own previous decision (its trace), the only thing a
	// source may learn about the other sources. Set by Decide, nil on the first tick.
	Prev *Decision
}

type Source interface {
	Name() string
	Propose(View) *Intent
}

// LostControl is optional: asked when the intent a source proposed was NOT chosen. by is what won; starved is true once
// the source has been outranked continuously for Config.StarveSec (called once per episode, plain overrides are called
// once per change of the winner). The source may drop or re-shape its proposal; it must not call back into the arbiter.
type LostControl interface {
	Lost(in, by Intent, why string, starved bool)
}

// TownVisitor is optional: asked when an upkeep intent takes the hero to town,
// so chores happen in the same visit.
type TownVisitor interface {
	WhileInTown(View) *Intent
}

// FuncSource adapts closures to Source and TownVisitor.
type FuncSource struct {
	N    string
	F    func(View) *Intent
	Town func(View) *Intent
}

func (s FuncSource) Name() string           { return s.N }
func (s FuncSource) Propose(v View) *Intent { return s.F(v) }
func (s FuncSource) WhileInTown(v View) *Intent {
	if s.Town == nil {
		return nil
	}
	return s.Town(v)
}

type Config struct {
	MinDwell     float64 // lease length: only a higher class preempts inside it
	TravelWindow float64 // an area change is not undone inside this without progress
	BackoffBase  float64 // first back-off after a failure
	BackoffMax   float64
	StarveSec    float64 // a proposal outranked this long continuously is starved (0 = 600)
}

func DefaultConfig() Config {
	return Config{MinDwell: 120, TravelWindow: 600, BackoffBase: 30, BackoffMax: 1800}
}

type Rejected struct {
	Intent Intent
	Why    string
}

type Decision struct {
	At       float64
	Area     string
	Chosen   *Intent
	Held     bool // chosen only because its lease holds
	Resumed  bool // chosen again after a higher class preempted it
	Batch    []Intent
	Rejected []Rejected
	Starved  []Intent // proposals outranked for Config.StarveSec, first seen this decision (the invariant arbiter-starved)
}

type lease struct {
	in    Intent
	since float64
}

type change struct {
	from, to string
	at       float64
	progress uint64
}

type backoff struct {
	n     int
	until float64
}

type Arbiter struct {
	cfg       Config
	sources   []Source
	cur       *lease
	lastArea  string
	last      *change
	fails     map[string]*backoff
	suspended map[string]Intent
	prev      *Decision
	temps     []temp
	outranked map[string]*outrank
}

type temp struct {
	in    Intent
	until float64
}

type outrank struct {
	since   float64
	by      string
	starved bool
}

func New(cfg Config, sources ...Source) *Arbiter {
	return &Arbiter{cfg: cfg, sources: sources, fails: map[string]*backoff{}, suspended: map[string]Intent{}, outranked: map[string]*outrank{}}
}

// Report tells the arbiter how an intent ended. A failure backs the intent's
// (source, kind, cause) off exponentially and drops its lease; success clears
// it. Player orders, reflexes and NoBackoff intents are never backed off.
func (a *Arbiter) Report(in Intent, now float64, ok bool) {
	k, bk := in.Key(), in.backoffKey()
	if ok {
		delete(a.fails, bk)
		if in.Class == Upkeep && in.NeedsArea == Town {
			// A finished town trip did its work (repair, potions, gold spent) though the objective's progress did not
			// move: going back to the field is not undoing it (live 2026-11-28, L5: the quest step was refused and held).
			a.last = nil
		}
		for i, t := range a.temps { // a temporary intent that was done is not proposed again
			if t.in.Key() == k {
				a.temps = append(a.temps[:i], a.temps[i+1:]...)
				break
			}
		}
		return
	}
	if !in.backsOff() {
		return
	}
	b := a.fails[bk]
	if b == nil {
		b = &backoff{}
		a.fails[bk] = b
	}
	d := a.cfg.BackoffBase
	for i := 0; i < b.n && d < a.cfg.BackoffMax; i++ {
		d *= 2
	}
	if d > a.cfg.BackoffMax {
		d = a.cfg.BackoffMax
	}
	b.n++
	b.until = now + d
	if a.cur != nil && a.cur.in.Key() == k {
		a.cur = nil
	}
}

func (a *Arbiter) Decide(v View) Decision {
	d := Decision{At: v.Now, Area: v.Area}
	v.Prev = a.prev
	defer func() { cp := d; a.prev = &cp }()
	if a.lastArea != "" && v.Area != a.lastArea {
		a.last = &change{from: a.lastArea, to: v.Area, at: v.Now, progress: v.Progress}
	}
	a.lastArea = v.Area

	var cands []Intent
	props := make([]*Intent, 0, len(a.sources)+len(a.temps))
	for _, s := range a.sources {
		in := s.Propose(v)
		if in != nil {
			in.Source = s.Name()
		}
		props = append(props, in)
	}
	props = append(props, a.liveTemps(v)...)
	for _, in := range props {
		if in == nil {
			continue
		}
		if why := a.refuse(*in, v); why != "" {
			d.Rejected = append(d.Rejected, Rejected{*in, why})
			continue
		}
		cands = append(cands, *in)
	}
	curKey := ""
	if a.cur != nil {
		curKey = a.cur.in.Key()
	}
	sort.SliceStable(cands, func(i, j int) bool {
		x, y := cands[i], cands[j]
		if x.Class != y.Class {
			return x.Class > y.Class
		}
		if x.Urgency != y.Urgency {
			return x.Urgency > y.Urgency
		}
		if (x.Key() == curKey) != (y.Key() == curKey) {
			return x.Key() == curKey // the running intent wins a tie
		}
		if x.EstCost != y.EstCost {
			return x.EstCost < y.EstCost
		}
		return x.Key() < y.Key()
	})

	if len(cands) == 0 {
		a.cur = nil
		a.lostControl(nil, cands, &d, v)
		return d
	}
	pick := 0
	if a.cur != nil && v.Now-a.cur.since < a.cfg.MinDwell {
		for i, c := range cands {
			if c.Key() == curKey && c.Class >= cands[0].Class {
				pick, d.Held = i, i != 0 // only a higher class breaks the lease
			}
		}
	}
	chosen := cands[pick]
	for i, c := range cands {
		if i != pick {
			d.Rejected = append(d.Rejected, Rejected{c, a.why(c, chosen, d.Held, v)})
		}
	}
	if a.cur != nil && chosen.Key() != curKey && chosen.Class > a.cur.in.Class && a.cur.in.Resumable {
		for _, c := range cands { // still wanted, only outranked
			if c.Key() == curKey {
				a.suspended[curKey] = c
			}
		}
	}
	if _, ok := a.suspended[chosen.Key()]; ok && chosen.Key() != curKey {
		d.Resumed = true
		delete(a.suspended, chosen.Key())
	}
	if a.cur == nil || a.cur.in.Key() != chosen.Key() {
		a.cur = &lease{in: chosen, since: v.Now}
	} else {
		a.cur.in = chosen
	}
	d.Chosen = &chosen
	a.lostControl(&chosen, cands, &d, v)

	if chosen.Class == Upkeep && chosen.NeedsArea == Town {
		for _, s := range a.sources {
			tv, ok := s.(TownVisitor)
			if !ok || s.Name() == chosen.Source {
				continue
			}
			if in := tv.WhileInTown(v); in != nil {
				in.Source = s.Name()
				d.Batch = append(d.Batch, *in)
			}
		}
	}
	return d
}

// refuse implements back-off and the travel budget. Reflexes and player
// orders are never refused here.
func (a *Arbiter) refuse(in Intent, v View) string {
	if in.Class >= Player {
		return ""
	}
	if b := a.fails[in.backoffKey()]; in.backsOff() && b != nil && v.Now < b.until {
		return fmt.Sprintf("backed off %.0fs more after %d failure(s)", b.until-v.Now, b.n)
	}
	if in.NeedsArea == Any || in.NeedsArea == v.Area || a.last == nil {
		return ""
	}
	l := a.last
	if in.NeedsArea == l.from && v.Now-l.at < a.cfg.TravelWindow && v.Progress == l.progress {
		return fmt.Sprintf("travel budget: going to %s undoes %s->%s %.0fs ago and nothing progressed since (wants it for: %s)",
			in.NeedsArea, l.from, l.to, v.Now-l.at, in.Reason)
	}
	return ""
}

func (a *Arbiter) why(c, chosen Intent, held bool, v View) string {
	switch {
	case held:
		return fmt.Sprintf("lease: %s holds %.0fs more (class %s does not outrank it)", chosen.Key(), a.cfg.MinDwell-(v.Now-a.cur.since), c.Class)
	case c.Class != chosen.Class:
		return fmt.Sprintf("class %s below %s", c.Class, chosen.Class)
	case c.Urgency != chosen.Urgency:
		return fmt.Sprintf("urgency %.2f below %.2f", c.Urgency, chosen.Urgency)
	}
	return "ranked after " + chosen.Key() + " (tie, cost or name)"
}

// String is the one trace line per decision.
func (d Decision) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "t=%05.0f area=%s ", d.At, d.Area)
	if d.Chosen == nil {
		b.WriteString("chose nothing")
	} else {
		c := d.Chosen
		fmt.Fprintf(&b, "chose %s %s u=%.2f need=%s cost=%.0f %q", c.Class, c.Key(), c.Urgency, c.NeedsArea, c.EstCost, c.Reason)
		if len(c.Steps) > 0 {
			fmt.Fprintf(&b, " steps=%v", c.Steps)
		}
		if d.Held {
			b.WriteString(" [lease]")
		}
		if d.Resumed {
			b.WriteString(" [resumed]")
		}
	}
	for _, in := range d.Batch {
		fmt.Fprintf(&b, " +town:%s(%s)", in.Key(), in.Reason)
	}
	for _, r := range d.Rejected {
		fmt.Fprintf(&b, " | rej %s %s: %s", r.Intent.Class, r.Intent.Key(), r.Why)
	}
	return b.String()
}

// Compact is the short form of the choice for the divergence log (reasons follow it there).
func (d Decision) Compact() string {
	if d.Chosen == nil {
		return "nothing"
	}
	s := fmt.Sprintf("%s (%s)", d.Chosen.Key(), d.Chosen.Reason)
	if d.Held {
		s += " [lease]"
	}
	return s
}

// Post adds a TEMPORARY intent (in.TTL > 0 or in.Until set): the arbiter proposes it next to the sources' own until it
// expires, is chosen and reported done, or its Until condition holds. A second post with the same key replaces the first.
func (a *Arbiter) Post(in Intent, now float64) {
	if in.TTL <= 0 && in.Until == "" {
		in.TTL = a.cfg.MinDwell
	}
	t := temp{in: in, until: now + in.TTL}
	if in.TTL <= 0 {
		t.until = now + 1800
	}
	for i := range a.temps {
		if a.temps[i].in.Key() == in.Key() {
			a.temps[i] = t
			return
		}
	}
	a.temps = append(a.temps, t)
}

// RefusalIntents maps a refusal need (internal/bot/outcome.go needTable codes) to the temporary intent it becomes. Data:
// a new need is a row. An empty Source means the refusal only pauses the step (no intent). TTL is virtual seconds.
var RefusalIntents = map[string]Intent{
	"needs_potions":   {Source: SrcResupply, Kind: "refill", Class: Upkeep, Urgency: 0.6, NeedsArea: Town, EstCost: 60, TTL: 300, Until: "potions>=potions_min", Reason: "refused: needs potions"},
	"needs_gold_farm": {Source: SrcRecovery, Kind: "farm_gold", Class: Objective, Urgency: 0.8, NeedsArea: Any, EstCost: 600, TTL: 900, Reason: "refused: needs gold"},
	"too_weak":        {Source: SrcLevel, Kind: "level_up", Class: Filler, Urgency: 0.4, NeedsArea: Any, EstCost: 600, TTL: 900, Reason: "refused: too weak, level first"},
}

// Refused turns a typed refusal (need, wake) into a temporary intent when RefusalIntents has a row for it, and reports
// whether one was posted. The wake text only goes into the reason: the arbiter does not parse it.
func (a *Arbiter) Refused(need, wake string, now float64) bool {
	in, ok := RefusalIntents[need]
	if !ok || in.Source == "" {
		return false
	}
	if wake != "" {
		in.Reason += " (wakes on: " + wake + ")"
	}
	a.Post(in, now)
	return true
}

func (a *Arbiter) liveTemps(v View) []*Intent {
	var out []*Intent
	kept := a.temps[:0]
	for _, t := range a.temps {
		if v.Now >= t.until || t.in.Until != "" && holds(t.in.Until, v.Facts) {
			continue
		}
		kept = append(kept, t)
		in := t.in
		out = append(out, &in)
	}
	a.temps = kept
	return out
}

// lostControl tells every source whose proposal was outranked who won, and flags starvation.
func (a *Arbiter) lostControl(chosen *Intent, cands []Intent, d *Decision, v View) {
	limit := a.cfg.StarveSec
	if limit <= 0 {
		limit = 600
	}
	seen := map[string]bool{}
	win := ""
	if chosen != nil {
		win = chosen.Key()
		delete(a.outranked, win)
	}
	for _, c := range cands {
		k := c.Key()
		if k == win {
			continue
		}
		seen[k] = true
		o := a.outranked[k]
		if o == nil {
			o = &outrank{since: v.Now}
			a.outranked[k] = o
		}
		starve := !o.starved && v.Now-o.since >= limit
		if win == o.by && !starve {
			continue
		}
		o.by = win
		if starve {
			o.starved = true
			d.Starved = append(d.Starved, c)
		}
		if chosen == nil {
			continue
		}
		if lc, ok := a.sourceOf(c.Source).(LostControl); ok {
			lc.Lost(c, *chosen, a.reasonFor(d, k), starve)
		}
	}
	for k := range a.outranked {
		if !seen[k] {
			delete(a.outranked, k)
		}
	}
}

func (a *Arbiter) reasonFor(d *Decision, key string) string {
	for _, r := range d.Rejected {
		if r.Intent.Key() == key {
			return r.Why
		}
	}
	return ""
}

func (a *Arbiter) sourceOf(name string) Source {
	for _, s := range a.sources {
		if s.Name() == name {
			return s
		}
	}
	return nil
}

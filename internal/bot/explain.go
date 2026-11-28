package bot

import (
	"sort"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/arbiter"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/questboard"
)

// The decision ring (Fable logic review 2026-11-25, P7): the last explainRing decisions the chain made (chosen hook
// family, reason code, params), the refusals standing now with what they need, and the arbiter's last shadow choice.
// Read-only and class-agnostic: it records what the bot already says through setWhy and its refusals, it changes no
// behaviour. Served by /api/explain and the operator `status --why` (docs/operator.md).

const (
	explainRing    = 200
	explainRefMax  = 12
	explainRefLife = 10 * time.Minute
)

// ExplainDecision is one change of what the hero is doing: the first tick of an activity, with how long it held.
type ExplainDecision struct {
	T      time.Time         `json:"t"`
	Tick   int               `json:"tick"`
	Family string            `json:"family,omitempty"`
	Code   string            `json:"code"`
	Params map[string]string `json:"params,omitempty"`
	Area   string            `json:"area,omitempty"`
	Ticks  int               `json:"ticks"` // ticks this decision held (updated while it stands)
}

// ExplainRefusal is a refusal standing now: what was refused, what it needs, when last seen.
type ExplainRefusal struct {
	What  string    `json:"what"`
	Need  string    `json:"need"`
	Code  string    `json:"code,omitempty"` // typed refusals: the need code (needTable)
	Wake  string    `json:"wake,omitempty"` // typed refusals: the fact that ends it
	Seen  time.Time `json:"seen"`
	Count int       `json:"count"`
}

// ExplainView is the answer of /api/explain.
type ExplainView struct {
	Decisions []ExplainDecision `json:"decisions"` // oldest first, at most 200
	Refusals  []ExplainRefusal  `json:"refusals"`
	Arbiter   string            `json:"arbiter,omitempty"` // the shadow's last choice, when the shadow runs
	// ArbiterDecision is the same choice as a record: intent, family, reason and what it displaced.
	ArbiterDecision *arbiter.Explained `json:"arbiterDecision,omitempty"`
	// Admission is the admission function's line (admission.go): mode, agreement with the old judges, last refusal with its row, need and wake.
	Admission AdmissionExplain `json:"admission"`
	Journey   string           `json:"journey,omitempty"` // why the quest journey is blocked now
	// Difficulty is the readiness line for the next difficulty (bot.diffReadyV2 1): ready and waiting for the owner's go, or what is missing.
	Difficulty string `json:"difficulty,omitempty"`
	// DifficultyOffer is the same as fields: target, ready, whether the owner's go is pending, what is missing.
	DifficultyOffer *DifficultyOffer `json:"difficultyOffer,omitempty"`
	// Invariants are the monitor counters with a since time; the Hard ones are what the flip watcher reverts on.
	Invariants []InvCounter `json:"invariants"`
	// Knobs are the live values of every knob (set by the wiring, not by the runner); tools/flip --verify reads the flipped one.
	Knobs map[string]float64 `json:"knobs,omitempty"`
	// Notices are standing operator warnings (a repaired character file, low disk space); set by the wiring.
	Notices []string `json:"notices,omitempty"`
}

type explainState struct {
	mu   sync.Mutex
	tick int
	ring []ExplainDecision
	refs map[string]*ExplainRefusal
}

// noteStanding records a standing refusal (what the step was, what it needs). Same key again only bumps the counter.
func (r *Runner) noteStanding(what, need string) { r.noteStandingRef(what, need, "", "") }

// noteRefused is the typed entry of the ring: the refusal handler (handleRefused, holdRefused) records the need code and
// the wake fact next to the words, so `status --why` shows what the hero waits for and what ends it.
func (r *Runner) noteRefused(what string, ref *Refused) {
	r.noteStandingRef(what, ref.Why, ref.Need, ref.Wake)
}

func (r *Runner) noteStandingRef(what, need, code, wake string) {
	x := &r.explain
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.refs == nil {
		x.refs = map[string]*ExplainRefusal{}
	}
	now := r.e.Clock.Now()
	if e := x.refs[what]; e != nil && e.Need == shorten(need, 200) && e.Code == code {
		e.Seen, e.Count = now, e.Count+1
		return
	}
	if len(x.refs) >= 4*explainRefMax {
		x.pruneLocked(now)
	}
	x.refs[what] = &ExplainRefusal{What: what, Need: shorten(need, 200), Code: code, Wake: wake, Seen: now, Count: 1}
}

func (x *explainState) pruneLocked(now time.Time) {
	for k, e := range x.refs {
		if now.Sub(e.Seen) > explainRefLife {
			delete(x.refs, k)
		}
	}
}

func shorten(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > n {
		s = s[:n]
	}
	return s
}

// explainTick records the activity the chain acted for. A change of family, code or area opens a new ring entry; the
// same activity only counts ticks, so the ring holds decisions and not seconds.
func (r *Runner) explainTick(snap *d2.Snapshot, acted bool) {
	if !acted {
		return
	}
	r.mu.Lock()
	w := r.why
	r.mu.Unlock()
	if w.Code == "" {
		return
	}
	area := ""
	if snap != nil && snap.Player != nil {
		area = "town"
		if !snap.Player.InTown() {
			area = r.areaName(snap.Player.Area)
		}
	}
	fam := chainFamily(w.Code)
	x := &r.explain
	x.mu.Lock()
	defer x.mu.Unlock()
	x.tick++
	if n := len(x.ring); n > 0 {
		last := &x.ring[n-1]
		if last.Code == w.Code && last.Family == fam && last.Area == area && sameParams(last.Params, w.Params) {
			last.Ticks++
			return
		}
	}
	if len(x.ring) >= explainRing {
		x.ring = append(x.ring[:0], x.ring[1:]...)
	}
	p := make(map[string]string, len(w.Params))
	for k, v := range w.Params {
		p[k] = shorten(v, 120)
	}
	x.ring = append(x.ring, ExplainDecision{T: r.e.Clock.Now(), Tick: x.tick, Family: fam, Code: w.Code, Params: p, Area: area, Ticks: 1})
}

func sameParams(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range b {
		if shorten(v, 120) != a[k] {
			return false
		}
	}
	return true
}

// Explain is the read-only answer for the watch API and the operator socket.
func (r *Runner) Explain() ExplainView {
	x := &r.explain
	now := r.e.Clock.Now()
	x.mu.Lock()
	v := ExplainView{Decisions: append([]ExplainDecision{}, x.ring...), Refusals: []ExplainRefusal{}}
	for _, e := range x.refs {
		if now.Sub(e.Seen) <= explainRefLife {
			v.Refusals = append(v.Refusals, *e)
		}
	}
	x.mu.Unlock()
	sort.Slice(v.Refusals, func(i, j int) bool { return v.Refusals[i].Seen.After(v.Refusals[j].Seen) })
	if len(v.Refusals) > explainRefMax {
		v.Refusals = v.Refusals[:explainRefMax]
	}
	v.Invariants = r.inv.Counters()
	v.Admission = r.admissionExplain()
	r.mu.Lock()
	v.Journey = r.journeyWhy
	r.mu.Unlock()
	if o := r.DifficultyOffer(); o.Target != "" {
		v.DifficultyOffer = &o
	}
	v.Difficulty = r.DifficultyReadiness()
	if d, ok := r.Arbiter(); ok && d.Chosen != nil {
		v.Arbiter = d.Chosen.Key() + " (" + d.Chosen.Reason + ")"
		if e, ok := d.Explained(); ok {
			v.ArbiterDecision = &e
		}
	}
	return v
}

// QuestRefusal is the freshest standing typed refusal as the quest board's "Why not?" facts (questboard.DeskConfig.Refusal):
// the need code, the fact that ends it and the guard words. False when nothing is standing.
func (r *Runner) QuestRefusal() (questboard.RefusalFacts, bool) {
	x := &r.explain
	now := r.e.Clock.Now()
	x.mu.Lock()
	defer x.mu.Unlock()
	var best *ExplainRefusal
	for _, e := range x.refs {
		if e.Code != "" && now.Sub(e.Seen) <= explainRefLife && (best == nil || e.Seen.After(best.Seen)) {
			best = e
		}
	}
	if best == nil {
		return questboard.RefusalFacts{}, false
	}
	return questboard.RefusalFacts{Need: best.Code, Wake: best.Wake, Admission: best.Need, Seen: best.Seen}, true
}

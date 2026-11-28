package questboard

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// The run machine of the quest board (design docs/design/fable-review-2026-11-21 section 3): facts are derived,
// intent is stored, completion is latched. This file is pure: no I/O, no clock, no game. It is not wired in until
// the knob quests.boardSM (card bsm4).

// RunStatus is the stored status of a run (3.2).
type RunStatus string

const (
	RunQueued    RunStatus = "queued"     // live, waits for an earlier run of the same chain
	RunRunning   RunStatus = "running"    // the head run, the hero works on it
	RunHeld      RunStatus = "held"       // the head cannot proceed now for a reason the bot resolves or waits out
	RunAwaiting  RunStatus = "awaiting"   // field work done, waits for the player's tap
	RunHandingIn RunStatus = "handing_in" // the dialogue runs
	RunParked    RunStatus = "parked"     // the bot gave up; the player acts
	RunFinished  RunStatus = "finished"   // closed with a latch
	RunStopped   RunStatus = "stopped"    // closed by the player or a cascade
)

// RunStatuses lists the eight, in table order.
var RunStatuses = []RunStatus{RunQueued, RunRunning, RunHeld, RunAwaiting, RunHandingIn, RunParked, RunFinished, RunStopped}

// Live reports a run that has not been closed.
func (s RunStatus) Live() bool { return s != RunFinished && s != RunStopped && s != "" }

// Head reports the statuses of the one head run per hero and difficulty (D3).
func (s RunStatus) Head() bool {
	return s == RunRunning || s == RunHeld || s == RunAwaiting || s == RunHandingIn
}

// Wire is the projection to the nine wire states of the page; a stopped run shows its offer status, so "".
func (s RunStatus) Wire() State {
	switch s {
	case RunQueued:
		return Accepted
	case RunRunning, RunHeld:
		return InProgress
	case RunAwaiting, RunHandingIn:
		return Awaiting
	case RunParked:
		return Blocked
	case RunFinished:
		return Done
	}
	return ""
}

// Run is the player's intent for one chunk in one difficulty.
type Run struct {
	Quest    string    `json:"quest"`
	Chunk    string    `json:"chunk"`
	Diff     string    `json:"difficulty"`
	Status   RunStatus `json:"status"`
	Code     string    `json:"code,omitempty"` // the reason code of a live run that is not running
	By       string    `json:"by,omitempty"`   // player, chain or auto
	Mode     string    `json:"mode,omitempty"` // the mode that was active when it was accepted
	At       time.Time `json:"at"`
	Failures int       `json:"failures,omitempty"` // kept with a park
	Deaths   int       `json:"deaths,omitempty"`
	// ParkedAt is when the run was parked and AutoResumes how many times the pause policy (resume.go) lifted its parks in a
	// row; a player's Try again clears the count. Additive in plan file v1: old files read both as zero.
	ParkedAt    time.Time `json:"parked_at,omitempty"`
	AutoResumes int       `json:"auto_resumes,omitempty"`
	// Release is the bot release that stored a gave_up park; RetriesAfterRelease counts the one automatic retry a new
	// release gets (additive in plan file v1: old files read both as zero).
	Release             string `json:"release,omitempty"`
	RetriesAfterRelease int    `json:"retries_after_release,omitempty"`
	// Hold, HoldWhy and HoldMin keep what a hold_timeout park waited for: the hold's code, the producer's words (the
	// readiness reason, "level 24 needed for the Barracks (clvl 20)") and the minutes it waited (additive: old files read
	// them empty).
	Hold    string `json:"hold,omitempty"`
	HoldWhy string `json:"hold_why,omitempty"`
	HoldMin int    `json:"hold_min,omitempty"`
	// RefusalCode and RefusalWhy keep the guard's code and text of a refused_repeatedly park (quests.refusalPark); additive.
	RefusalCode string `json:"refusal_code,omitempty"`
	RefusalWhy  string `json:"refusal_why,omitempty"`
}

// Latch marks a chunk as seen done (3.6). Tier only rises.
type Latch struct {
	Quest  string    `json:"quest"`
	Chunk  string    `json:"chunk"`
	Diff   string    `json:"difficulty"`
	Tier   Tier      `json:"tier"`
	At     time.Time `json:"at"`
	Source string    `json:"source,omitempty"`
}

// SM is the stored lifecycle: runs and latches (plan file v2, card bsm3).
type SM struct {
	Runs    []Run   `json:"runs,omitempty"`
	Latches []Latch `json:"latches,omitempty"`
}

// Event is a cause of a transition (3.3). The table of RunTransition is total over RunStatus x Event.
type Event string

const (
	EvPromote     Event = "promote"      // first live run in data order, nothing parked ahead
	EvRefuse      Event = "refuse"       // the step refused for a reason the bot resolves (code: a hold)
	EvClear       Event = "clear"        // the hold's fact cleared
	EvHoldTimeout Event = "hold_timeout" // the hold outlived its budget
	EvFail        Event = "fail"         // a step failed under the cap
	EvCap         Event = "cap"          // failures or deaths at the cap (code: gave_up or died_too_often)
	EvObjective   Event = "objective"    // field work done, the chunk ends in a hand-in
	EvChunkDone   Event = "chunk_done"   // the chunk is done and ends with an item or by itself
	EvHandInTap   Event = "handin_tap"
	EvHandInDone  Event = "handin_done"
	EvHandInFail  Event = "handin_fail"
	EvRetry       Event = "retry"
	EvStop        Event = "stop"
	EvExactDone   Event = "exact_done"  // exact evidence says done
	EvGameFailed  Event = "game_failed" // the game failed the quest
	EvRestart     Event = "restart"
)

// Events lists every event, in table order.
var Events = []Event{EvPromote, EvRefuse, EvClear, EvHoldTimeout, EvFail, EvCap, EvObjective, EvChunkDone, EvHandInTap, EvHandInDone, EvHandInFail, EvRetry, EvStop, EvExactDone, EvGameFailed, EvRestart}

// RunTransition is the answer of Step. Noop means the cell is not in the table and the caller answers "already".
type RunTransition struct {
	To   RunStatus
	Code string // the reason code of To ("" for running and the closed statuses)
	Noop bool
}

// ErrReason is returned when the code an event needs is missing or is not of the right kind.
var ErrReason = errors.New("questboard: reason code missing or of the wrong kind")

// Step is the transition table. code is the reason the caller supplies for the events that need one (refuse, cap,
// hold_timeout and a handin_fail use their own fixed or given code). It is total: every (status, event) pair
// returns a RunTransition, never a panic, and a live target always carries a code of the closed list (I5, I8).
func Step(from RunStatus, ev Event, code string) (RunTransition, error) {
	noop := RunTransition{To: from, Code: codeOf(from), Noop: true} // Code is "" for held and parked: the run keeps its own
	if !from.Live() {
		return noop, nil // finished and stopped never move
	}
	to := func(s RunStatus, c string) (RunTransition, error) { return RunTransition{To: s, Code: c}, nil }
	need := func(k ReasonKind, s RunStatus) (RunTransition, error) {
		r, ok := ReasonOf(code)
		if !ok || r.Kind != k {
			return noop, fmt.Errorf("%w: %q for %s", ErrReason, code, s)
		}
		return to(s, code)
	}
	switch ev {
	case EvPromote:
		if from == RunQueued {
			return to(RunRunning, "")
		}
	case EvRefuse:
		if from == RunRunning || from == RunHeld {
			return need(KindHold, RunHeld)
		}
	case EvClear:
		if from == RunHeld {
			return to(RunRunning, "")
		}
	case EvHoldTimeout:
		if from == RunHeld {
			return to(RunParked, "hold_timeout")
		}
	case EvFail:
		if from == RunRunning {
			return to(RunRunning, "")
		}
	case EvCap:
		if from == RunRunning || from == RunHeld {
			return need(KindNeeds, RunParked)
		}
	case EvObjective:
		if from == RunRunning || from == RunHeld {
			return to(RunAwaiting, "handin_wanted")
		}
	case EvChunkDone:
		if from == RunRunning || from == RunHeld {
			return to(RunFinished, "")
		}
	case EvHandInTap:
		if from == RunAwaiting {
			return to(RunHandingIn, "confirming")
		}
	case EvHandInDone:
		if from == RunHandingIn {
			return to(RunFinished, "")
		}
	case EvHandInFail:
		if from == RunHandingIn {
			return to(RunParked, "handin_failed")
		}
	case EvRetry:
		if from == RunParked {
			return to(RunQueued, "waits_for") // Promote makes it the head again when it is first
		}
	case EvStop:
		return to(RunStopped, "")
	case EvExactDone:
		return to(RunFinished, "")
	case EvGameFailed:
		return to(RunParked, "game_failed")
	case EvRestart:
		switch from {
		case RunRunning, RunHeld:
			return to(RunQueued, "waits_for")
		case RunHandingIn:
			return to(RunAwaiting, "handin_wanted")
		}
	}
	return noop, nil
}

// codeOf is the code a live status carries when it holds no code of its own (the no-op answer).
func codeOf(s RunStatus) string {
	switch s {
	case RunQueued:
		return "waits_for"
	case RunAwaiting:
		return "handin_wanted"
	case RunHandingIn:
		return "confirming"
	}
	return ""
}

// ErrBusy is Start in another quest while one is live (D5); Quest names the running one.
type ErrBusy struct{ Quest string }

func (e ErrBusy) Error() string { return "questboard: " + e.Quest + " is running" }

// ErrDone is Start of a chunk that is already done.
var ErrDone = errors.New("questboard: the chunk is already done")

// ErrNotStartable names the part of the chain that cannot start and why.
type ErrNotStartable struct {
	Part Ref
	Err  error
}

func (e ErrNotStartable) Error() string {
	return fmt.Sprintf("questboard: %s/%s cannot start: %v", e.Part.Quest, e.Part.Chunk, e.Err)
}
func (e ErrNotStartable) Unwrap() error { return e.Err }

// order is the data order of the chunks of one board: the head is the first live run in it (D3, I3).
func order(b *Board) map[Ref]int {
	m := map[Ref]int{}
	n := 0
	for _, q := range b.Quests() {
		for _, c := range q.Chunks {
			m[Ref{q.ID, c.ID}] = n
			n++
		}
	}
	return m
}

func (m *SM) find(quest, chunk, diff string) *Run {
	for i := range m.Runs {
		r := &m.Runs[i]
		if r.Quest == quest && r.Chunk == chunk && r.Diff == diff && r.Status.Live() {
			return r
		}
	}
	return nil
}

func (m *SM) latch(quest, chunk, diff string) *Latch {
	for i := range m.Latches {
		l := &m.Latches[i]
		if l.Quest == quest && l.Chunk == chunk && l.Diff == diff {
			return l
		}
	}
	return nil
}

// Live returns the live runs of a difficulty in data order.
func (m *SM) Live(b *Board, diff string) []Run {
	ord := order(b)
	var out []Run
	for _, r := range m.Runs {
		if r.Diff == diff && r.Status.Live() {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return ord[Ref{out[i].Quest, out[i].Chunk}] < ord[Ref{out[j].Quest, out[j].Chunk}]
	})
	return out
}

// Latched reports a latch for the chunk.
func (m *SM) Latched(quest, chunk, diff string) bool { return m.latch(quest, chunk, diff) != nil }

// ancestors returns the transitive `after` chunks of chunk in data order (the chain D1 accepts).
func ancestors(b *Board, quest, chunk string) ([]string, error) {
	q, err := b.Quest(quest)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var walk func(string) error
	walk = func(id string) error {
		c, err := q.ChunkByID(id)
		if err != nil {
			return err
		}
		for _, a := range c.After {
			if !seen[a] {
				seen[a] = true
				if err := walk(a); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(chunk); err != nil {
		return nil, err
	}
	var out []string
	for _, c := range q.Chunks {
		if seen[c.ID] {
			out = append(out, c.ID)
		}
	}
	return out, nil
}

// dependants returns the live runs of the same quest that wait for chunk, directly or not (D2).
func (m *SM) dependants(b *Board, quest, chunk, diff string) []string {
	var out []string
	for _, r := range m.Runs {
		if r.Quest != quest || r.Diff != diff || !r.Status.Live() || r.Chunk == chunk {
			continue
		}
		anc, err := ancestors(b, quest, r.Chunk)
		if err != nil {
			continue
		}
		for _, a := range anc {
			if a == chunk {
				out = append(out, r.Chunk)
				break
			}
		}
	}
	return out
}

// Start creates the run of chunk and the runs of its unmet earlier parts in one step (D1, D5). startable is asked for
// every part that would be created; a refusal creates nothing. already is true when a live run of chunk exists. The
// returned changed are the chunks that got a run, in data order.
func (m *SM) Start(b *Board, diff, quest, chunk, by string, at time.Time, startable func(Ref) error) (changed []Ref, already bool, err error) {
	if m.find(quest, chunk, diff) != nil {
		return nil, true, nil
	}
	if m.latch(quest, chunk, diff) != nil {
		return nil, false, ErrDone
	}
	for _, r := range m.Live(b, diff) {
		if r.Quest != quest {
			return nil, false, ErrBusy{r.Quest}
		}
	}
	anc, err := ancestors(b, quest, chunk)
	if err != nil {
		return nil, false, err
	}
	var add []Run
	for _, id := range append(anc, chunk) {
		if m.find(quest, id, diff) != nil || m.latch(quest, id, diff) != nil {
			continue
		}
		part := Ref{quest, id}
		if startable != nil {
			if e := startable(part); e != nil {
				return nil, false, ErrNotStartable{part, e}
			}
		}
		by2 := "chain"
		if id == chunk {
			by2 = by
		}
		add = append(add, Run{Quest: quest, Chunk: id, Diff: diff, Status: RunQueued, Code: "waits_for", By: by2, At: at})
		changed = append(changed, part)
	}
	m.Runs = append(m.Runs, add...)
	m.Promote(b, diff)
	return changed, false, nil
}

// Stop closes the live run of chunk and every live run that waits for it (D2). already is true when there was none.
func (m *SM) Stop(b *Board, diff, quest, chunk string) (changed []Ref, already bool) {
	ids := append(m.dependants(b, quest, chunk, diff), chunk)
	for _, id := range ids {
		if r := m.find(quest, id, diff); r != nil {
			r.Status, r.Code = RunStopped, ""
			changed = append(changed, Ref{quest, id})
		}
	}
	m.Promote(b, diff)
	return changed, len(changed) == 0
}

// Retry takes a parked run back to the queue and clears its counters; with again it also accepts the regressed
// earlier part of a prereq_regressed park (R4). already is true when the run is not parked.
func (m *SM) Retry(b *Board, diff, quest, chunk string, again bool, at time.Time) (changed []Ref, already bool, err error) {
	r := m.find(quest, chunk, diff)
	if r == nil {
		return nil, false, fmt.Errorf("questboard: %s/%s has no run", quest, chunk)
	}
	if r.Status != RunParked {
		return nil, true, nil
	}
	if r.Code == "game_failed" {
		return nil, false, fmt.Errorf("questboard: %s/%s failed in this game", quest, chunk)
	}
	if again && r.Code == "prereq_regressed" {
		anc, _ := ancestors(b, quest, chunk)
		for _, id := range anc {
			if m.find(quest, id, diff) == nil && m.latch(quest, id, diff) == nil {
				m.Runs = append(m.Runs, Run{Quest: quest, Chunk: id, Diff: diff, Status: RunQueued, Code: "waits_for", By: "chain", At: at})
				changed = append(changed, Ref{quest, id})
			}
		}
	}
	tr, _ := Step(r.Status, EvRetry, "")
	r.Status, r.Code, r.Failures, r.Deaths = tr.To, tr.Code, 0, 0
	r.ParkedAt, r.AutoResumes = time.Time{}, 0
	changed = append(changed, Ref{quest, chunk})
	m.Promote(b, diff)
	return changed, false, nil
}

// Promote selects the head (D3): the first live run in data order. A running, held or awaiting run that is no longer
// first (an earlier part was started behind it) goes back to the queue; a run whose dialogue is in flight is never
// swapped out. When the first run is queued and nothing ahead of it is parked, it becomes running; a parked head stops
// everything behind it. It reports the runs whose status it changed.
func (m *SM) Promote(b *Board, diff string) (changed []Ref) {
	live := m.Live(b, diff)
	for _, r := range live {
		if r.Status == RunHandingIn {
			return nil
		}
	}
	for _, r := range live[min(1, len(live)):] {
		if r.Status == RunRunning || r.Status == RunHeld || r.Status == RunAwaiting {
			rr := m.find(r.Quest, r.Chunk, diff)
			rr.Status, rr.Code = RunQueued, "waits_for"
			changed = append(changed, Ref{r.Quest, r.Chunk})
		}
	}
	if len(live) > 0 && live[0].Status == RunQueued {
		r := m.find(live[0].Quest, live[0].Chunk, diff)
		tr, _ := Step(r.Status, EvPromote, "")
		r.Status, r.Code = tr.To, tr.Code
		changed = append(changed, Ref{r.Quest, r.Chunk})
	}
	return changed
}

// Apply runs one event on the live run of chunk and then re-selects the head. A no-op reports already.
func (m *SM) Apply(b *Board, diff, quest, chunk string, ev Event, code string) (already bool, err error) {
	r := m.find(quest, chunk, diff)
	if r == nil {
		return true, nil
	}
	tr, err := Step(r.Status, ev, code)
	if err != nil {
		return false, err
	}
	if tr.Noop {
		return true, nil
	}
	r.Status, r.Code = tr.To, tr.Code
	if tr.To == RunFinished || tr.To == RunStopped {
		r.Code = ""
	}
	m.Promote(b, diff)
	return false, nil
}

// NoteDone latches a chunk seen done (R1, R2): the first read writes the latch, a higher tier raises it, a lower or
// equal one changes nothing. It returns true when the latch changed, and closes the live run of the chunk (I4).
func (m *SM) NoteDone(b *Board, diff, quest, chunk string, tier Tier, at time.Time, source string) bool {
	changed := false
	if l := m.latch(quest, chunk, diff); l == nil {
		m.Latches = append(m.Latches, Latch{Quest: quest, Chunk: chunk, Diff: diff, Tier: tier, At: at, Source: source})
		changed = true
	} else if tier > l.Tier {
		l.Tier, l.At, l.Source = tier, at, source
		changed = true
	}
	if r := m.find(quest, chunk, diff); r != nil {
		r.Status, r.Code = RunFinished, ""
		changed = true
	}
	m.Promote(b, diff)
	return changed
}

// Regress is R3 and R4: exact evidence (T1 or T2) says "not done" for a latched chunk. The latch is cleared (the
// caller logs a regressed event); when live runs wait for the chunk the head is parked with prereq_regressed.
// It reports the parked run, if any. A non-exact tier never clears a latch (R1).
func (m *SM) Regress(b *Board, diff, quest, chunk string, tier Tier) (parked *Ref, cleared bool) {
	if !tier.Exact() {
		return nil, false
	}
	for i := range m.Latches {
		l := m.Latches[i]
		if l.Quest == quest && l.Chunk == chunk && l.Diff == diff {
			m.Latches = append(m.Latches[:i], m.Latches[i+1:]...)
			cleared = true
			break
		}
	}
	if !cleared {
		return nil, false
	}
	deps := m.dependants(b, quest, chunk, diff)
	if len(deps) == 0 {
		return nil, true
	}
	live := m.Live(b, diff)
	h := m.find(live[0].Quest, live[0].Chunk, diff)
	h.Status, h.Code = RunParked, "prereq_regressed"
	return &Ref{h.Quest, h.Chunk}, true
}

// closureExempt: a run parked because its earlier part regressed (R4) or could not start (D4) is the visible answer to
// a missing part, not a breach of closure; the player decides (Run it again, Stop).
func closureExempt(r Run) bool {
	return r.Status == RunParked && (r.Code == "prereq_regressed" || r.Code == "prereq_missing")
}

// Violation is one broken invariant.
type Violation struct{ Inv, Text string }

// Check returns the broken invariants I1 to I5 of the stored set for one difficulty (empty: sound).
func (m *SM) Check(b *Board, diff string) []Violation {
	var out []Violation
	add := func(inv, f string, a ...any) { out = append(out, Violation{inv, fmt.Sprintf(f, a...)}) }
	live := m.Live(b, diff)
	heads, quest := 0, ""
	parkedAhead := false
	for i, r := range live {
		if r.Status.Head() {
			heads++
			if i > 0 {
				add("I3", "%s/%s is a head but %s/%s comes first in data order", r.Quest, r.Chunk, live[0].Quest, live[0].Chunk)
			}
			if parkedAhead {
				add("I3", "%s/%s runs behind a parked run", r.Quest, r.Chunk)
			}
		}
		if r.Status == RunParked {
			parkedAhead = true
		}
		if quest != "" && r.Quest != quest {
			add("D5", "live runs of two quests: %s and %s", quest, r.Quest)
		}
		quest = r.Quest
		if m.latch(r.Quest, r.Chunk, diff) != nil {
			add("I4", "%s/%s is live and latched", r.Quest, r.Chunk)
		}
		if r.Status != RunRunning {
			if info, ok := ReasonOf(r.Code); !ok || info.Kind == KindNobody && r.Status != RunParked {
				add("I5", "%s/%s is %s with reason %q", r.Quest, r.Chunk, r.Status, r.Code)
			}
		}
		anc, _ := ancestors(b, r.Quest, r.Chunk)
		for _, a := range anc {
			if closureExempt(r) {
				break
			}
			if m.find(r.Quest, a, diff) == nil && m.latch(r.Quest, a, diff) == nil {
				add("I2", "%s/%s lacks its earlier part %s", r.Quest, r.Chunk, a)
			}
		}
	}
	if heads > 1 {
		add("I1", "%d head runs", heads)
	}
	return out
}

// Repair is the load step of D4: it adds the missing earlier parts of every live run when they are startable, else it
// parks the run with prereq_missing. It returns what it added and what it parked; the caller logs both.
func (m *SM) Repair(b *Board, diff string, at time.Time, startable func(Ref) error) (added, parked []Ref) {
	for _, r := range m.Live(b, diff) {
		if closureExempt(r) {
			continue
		}
		anc, _ := ancestors(b, r.Quest, r.Chunk)
		for _, a := range anc {
			if m.find(r.Quest, a, diff) != nil || m.latch(r.Quest, a, diff) != nil {
				continue
			}
			if startable != nil && startable(Ref{r.Quest, a}) != nil {
				if rr := m.find(r.Quest, r.Chunk, diff); rr.Status != RunParked {
					rr.Status, rr.Code = RunParked, "prereq_missing"
					parked = append(parked, Ref{r.Quest, r.Chunk})
				}
				continue
			}
			m.Runs = append(m.Runs, Run{Quest: r.Quest, Chunk: a, Diff: diff, Status: RunQueued, Code: "waits_for", By: "chain", At: at})
			added = append(added, Ref{r.Quest, a})
		}
	}
	m.Promote(b, diff)
	return added, parked
}

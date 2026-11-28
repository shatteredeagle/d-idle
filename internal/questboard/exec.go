package questboard

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// The executor hand-off (slice S3). An accepted plan record starts the existing chunk handler (the task layer's
// quest steps) through Actions; the bot does the chunk live, walks to the NPC and STOPS (Awaiting); the player's
// HandIn tap performs the dialogue and the board verifies the reward by read-back. The Exec holds no handler
// progress that outlives the process: every decision is re-derived from Env (T1/T2/T3 evidence, inventory
// checks) and the plan records, so a restart at any step resumes by itself (design 1.5, 2).

// ErrOff is returned by every action while the knob quests.board is 0.
var ErrOff = errors.New("questboard: quests.board is off")

// StepOutcome is what one slice of the chunk handler reports.
type StepOutcome int

const (
	StepRan     StepOutcome = iota // progress: call again
	StepWaiting                    // nothing to do this tick (recovering, loading); not a failure
	StepFailed                     // a retryable failure; Why says what
	StepHeld                       // a guard refused for a reason that is no failure (StepResult.Code, a hold of appendix B.1); nothing is counted
	StepDied                       // the hero died; the bot recovers, then the chunk resumes from the evidence
)

// StepResult is the outcome of Actions.Step.
type StepResult struct {
	Outcome StepOutcome
	Why     string
	Code    string // StepHeld: the hold's reason code; StepFailed: the code of a coded refusal (quests.refusalPark)
}

// Env is the read-only evidence of one tick. Everything is re-read each time; none of it is stored here.
type Env interface {
	Diff() int                        // difficulty index of the running game
	Facts() Facts                     // quest reader (T1 save, T2 memory, T3 proxy)
	Chunk(q Quest, c Chunk) ChunkFact // handler checks and inventory read-back; Running and Blocked are set by Exec
	Ready(q Quest, c Chunk) Word      // the readiness word (S2; LegacyReadiness until then)
}

// Actions is the seam to the task layer (the bot adapter): HandIn is the hand-in dialogue only (it never
// walks the chunk), Stop cancels a running slice cleanly.
type Actions interface {
	HandIn(ctx context.Context, q Quest, c Chunk) error
	Stop(q Quest, c Chunk)
}

// Stepper is one bounded slice of the chunk handler that never performs the hand-in. The bot runs its slices
// inside the runner's guard and reports them with Report; Tick takes a Stepper for drivers without a guard.
type Stepper func(ctx context.Context, q Quest, c Chunk) StepResult

// Options are the executor limits. Zero values take the defaults.
type Options struct {
	Enabled          bool   // the knob quests.board
	SM               bool   // the knob quests.boardSM: runs and latches in the plan file drive the board (sm.go, exec_sm.go)
	MaxFailures      int    // consecutive failed slices before the chunk parks (default 3)
	MaxDeaths        int    // deaths on one chunk before it parks (default 2)
	MaxHandIn        int    // dialogue attempts per HandIn call before it parks (default 3)
	Release          string // the bot release id; a gave_up park from another release is retried once (exec_sm.go)
	Mode             string // recorded on the plan record (train, idle, farm); default train
	Log              func(msg string, kv ...any)
	Now              func() time.Time // the clock of the auto-start controller (default time.Now); tests pass a fake
	Auto             AutoOptions      // the auto-start limits (auto.go); the zero value never starts anything by itself
	Resume           ResumeOptions    // the pause policy (resume.go); the zero value resumes nothing
	RefusalParkAfter int              // quests.refusalPark: identical coded refusals that park a chunk as refused_repeatedly (refusal.go); 0: off
	Ledger           func() Ledger    // the account wallet (ledger.go); nil, or a nil result, makes the Ether hook inert
}

func (o Options) norm() Options {
	if o.MaxFailures <= 0 {
		o.MaxFailures = 3
	}
	if o.MaxDeaths <= 0 {
		o.MaxDeaths = 2
	}
	if o.MaxHandIn <= 0 {
		o.MaxHandIn = 3
	}
	if o.Mode == "" {
		o.Mode = "train"
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.Log == nil {
		o.Log = func(string, ...any) {}
	}
	return o
}

// runState is the in-memory bookkeeping of one chunk: counters and the park reason. With the run machine the failure
// and death counters are stored on the live run at every count (smCount) and read back after a restart (smSeed), so a
// restart between two deaths keeps counting; on the legacy board a restart clears them.
type runState struct {
	started  bool
	failures int
	deaths   int
	lastWhy  string
	blocked  string
	hold     string    // the code of the hold the head run is under (quests.boardSM), "" when none
	holdWhy  string    // the guard's words
	holdFrom time.Time // when the hold began; the budget of its code runs from here
	pathWhy  string    // the last counted navigation refusal and when it was counted (samePathWindow)
	pathAt   time.Time
	refCode  string // the code of the refusal streak (quests.refusalPark), refN identical ones in a row, refWhy the last text
	refN     int
	refWhy   string
}

// Exec is the board's executor and the API-free interface the S4 API calls: Offers, Accept, HandIn, State
// (plus Cancel, Reset, Pause, Unblock).
type Exec struct {
	b   *Board
	st  *Store
	env Env
	act Actions

	mu      sync.Mutex
	relDone bool // smReleaseRetry has run
	o       Options
	rs      map[string]*runState
	paused  bool
	handing bool // a HandIn dialogue is in flight: Next answers Busy
	// lastChanged is the runs the last machine verb touched, read by the same caller under e.mu (AcceptRuns and kin).
	lastChanged []Ref
	ax          autoState
	// try is the chunk key of the owner's one-shot "try anyway" start (docs/questboard.md, ADR 0030 amendment 2026-11-24);
	// tryNow is true only while AcceptTry accepts it, so CanStart skips the readiness word for that one chunk. Memory only:
	// a restart ends it, and the gates then hold the chunk again.
	try    string
	tryNow bool
}

// NewExec builds the executor of one hero. env and act are the bot adapter (or the Fake in tests).
func NewExec(b *Board, st *Store, env Env, act Actions, o Options) *Exec {
	o = o.norm()
	e := &Exec{b: b, st: st, env: env, act: act, o: o, rs: map[string]*runState{}}
	e.ax.born = o.Now()
	return e
}

// Enabled reports the knob.
func (e *Exec) Enabled() bool { e.mu.Lock(); defer e.mu.Unlock(); return e.o.Enabled }

// SetEnabled follows the knob at run time. Switching off stops the running slice and drops the in-memory state; the
// plan records and the stored parks stay (a knob switch is no Try again: only the player lifts a park).
func (e *Exec) SetEnabled(on bool) {
	e.mu.Lock()
	was := e.o.Enabled
	e.o.Enabled = on
	e.mu.Unlock()
	if was && !on {
		e.reset(false)
	}
}

// SetSM follows the knob quests.boardSM at run time, like the guards that read it every tick: a knob edited from the
// tuning page must not leave the executor on the other board (fable review 2 finding 5). Stored runs and parks stay.
func (e *Exec) SetSM(on bool) { e.mu.Lock(); e.o.SM = on; e.mu.Unlock() }

// SM reports whether the executor runs on the run machine now.
func (e *Exec) SM() bool { e.mu.Lock(); defer e.mu.Unlock(); return e.o.SM }

// SetMode records the between-chunks mode on new plan records (modes are slice S6).
func (e *Exec) SetMode(m string) { e.mu.Lock(); e.o.Mode = m; e.mu.Unlock() }

func (e *Exec) state(k string) *runState {
	r := e.rs[k]
	if r == nil {
		r = &runState{}
		e.rs[k] = r
	}
	return r
}

// seeded is state for a chunk about to be counted or parked: a state made here starts from the stored counters of the
// live run, so a park written after a restart does not store zeros over them (review 10 B3). Callers hold e.mu.
func (e *Exec) seeded(quest, chunk string) *runState {
	k := ChunkKey(quest, chunk)
	_, known := e.rs[k]
	rs := e.state(k)
	if !known {
		e.smSeed(quest, chunk, rs)
	}
	return rs
}

// derive builds the views for the current difficulty. Callers hold e.mu.
func (e *Exec) derive() ([]QuestView, Plan, int, error) {
	plan, _, err := e.st.Load()
	if err != nil {
		return nil, Plan{}, 0, err
	}
	diff := e.env.Diff()
	var sm *smOverlay
	if !e.boardLegacy() {
		if sm, err = e.smLoad(diff); err != nil {
			return nil, Plan{}, 0, err
		}
		if plan, _, err = e.st.Load(); err != nil { // the load may have imported or repaired
			return nil, Plan{}, 0, err
		}
	}
	in := Input{Diff: diff, Facts: e.env.Facts(), Plan: plan,
		Chunks: map[string]ChunkFact{}, Ready: map[string]Word{}}
	for _, q := range e.b.Quests() {
		if !q.Exists(diff) {
			continue
		}
		for _, c := range q.Chunks {
			k := ChunkKey(q.ID, c.ID)
			cf := e.env.Chunk(q, c)
			// After a restart nothing says "running" but the evidence: a step that is already satisfied.
			for _, ok := range cf.Steps {
				cf.Running = cf.Running || ok
			}
			if rs := e.rs[k]; rs != nil {
				cf.Running = cf.Running || rs.started
				if rs.blocked != "" {
					cf.Blocked = rs.blocked
				}
			}
			if sm != nil {
				cf = sm.apply(q, c, cf, e.rs[k])
			} else if run := plan.storedPark(q.ID, c.ID, Difficulties[diff]); run != nil && cf.Blocked == "" {
				// A park the machine stored outlives the knob going back to 0 (read only): a revert never restarts
				// the chunk that killed the hero. Try again lifts it (UnblockRuns).
				cf.Blocked = smParkText(q, *run)
			}
			in.Chunks[k] = cf
			in.Ready[k] = e.env.Ready(q, c)
		}
	}
	if sm != nil && sm.dirty() {
		if err := e.smFlush(sm, diff); err != nil {
			return nil, Plan{}, 0, err
		}
		return e.derive() // the latches or the regression just written are now part of the stored state
	}
	vs, err := Derive(e.b, in)
	if err == nil && sm != nil && e.smAutoFinish(vs, diff) {
		return e.derive()
	}
	if err == nil && !plan.based(diff) {
		// First sight of this hero in this difficulty: whatever is done now never pays (no backfill).
		if berr := e.st.RecordBaseline(diff, doneKeys(vs, diff)); berr != nil {
			e.o.Log("quest baseline not saved", "err", berr)
			return vs, plan, diff, nil
		}
		return e.derive()
	}
	return vs, plan, diff, err
}

func chunkView(vs []QuestView, quest, chunk string) (QuestView, ChunkView, bool) {
	for _, v := range vs {
		if v.ID != quest {
			continue
		}
		for _, c := range v.Chunks {
			if c.ID == chunk {
				return v, c, true
			}
		}
	}
	return QuestView{}, ChunkView{}, false
}

// Ref names one chunk.
type Ref struct{ Quest, Chunk string }

func (r Ref) String() string { return r.Quest + "/" + r.Chunk }

// open lists the accepted chunks of this difficulty that are neither done nor skipped (normally 0 or 1).
func open(vs []QuestView, plan Plan, diff int) []Ref {
	var out []Ref
	for _, r := range plan.Accepted {
		if r.Diff != Difficulties[diff] {
			continue
		}
		if _, c, ok := chunkView(vs, r.Quest, r.Chunk); ok && c.State != Done && c.State != Skipped {
			out = append(out, Ref{r.Quest, r.Chunk})
		}
	}
	// Chain order (the questspec data order), not the order of the taps: the first runs, the rest wait for it.
	pos := map[Ref]int{}
	for _, v := range vs {
		for _, c := range v.Chunks {
			pos[Ref{v.ID, c.ID}] = len(pos)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return pos[out[i]] < pos[out[j]] })
	return out
}

// open is the package-level open for the Exec: with the run machine only a record with a live run counts, so a
// finished run (kept in Accepted as the proof that the board ran it) never reopens when its evidence goes back.
func (e *Exec) open(vs []QuestView, plan Plan, diff int) []Ref {
	out := open(vs, plan, diff)
	if e.boardLegacy() {
		return out
	}
	d := Difficulties[diff]
	live := out[:0]
	for _, r := range out {
		for _, run := range plan.Runs {
			if run.Quest == r.Quest && run.Chunk == r.Chunk && run.Diff == d && run.Status.Live() {
				live = append(live, r)
				break
			}
		}
	}
	return live
}

// WaitsFor is the visible reason of a queued chunk: it waits for the one that runs.
func WaitsFor(running Ref) string { return "waits for " + running.String() }

// Offer is one chunk that raises an offer now (state Ready).
type Offer struct {
	Quest      string
	QuestTitle string
	Chunk      string
	Title      string
	Act        int
	Risk       string
	Readiness  Word
	Minutes    [2]int
	Ends       string
	Startable  bool // false while another chunk is accepted (one at a time)
}

// Offers lists the Ready chunks of the current difficulty in data order.
func (e *Exec) Offers() ([]Offer, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.o.Enabled {
		return nil, ErrOff
	}
	vs, plan, diff, err := e.derive()
	if err != nil {
		return nil, err
	}
	busy := len(e.open(vs, plan, diff)) > 0
	var out []Offer
	for _, v := range vs {
		q, err := e.b.Quest(v.ID)
		if err != nil {
			return nil, err
		}
		for _, cv := range v.Chunks {
			if cv.State != Ready {
				continue
			}
			c, err := e.b.Chunk(v.ID, cv.ID)
			if err != nil {
				return nil, err
			}
			risk := c.Risk
			if risk == "" {
				risk = q.Risk
			}
			out = append(out, Offer{Quest: v.ID, QuestTitle: v.Title, Chunk: cv.ID, Title: c.Title, Act: q.Act, Risk: risk,
				Readiness: cv.Reason, Minutes: c.Minutes, Ends: c.Ends, Startable: !busy})
		}
	}
	return out, nil
}

// Accept records the player's Start tap for the CURRENT chunk (a tap never covers the next part). It refuses
// anything that is not Ready and a second concurrent chunk, naming the one that runs.
func (e *Exec) Accept(quest, chunk string) error {
	_, err := e.AcceptRuns(quest, chunk)
	return err
}

// AcceptRuns is Accept that also names the chunks that got a run (the reply's `changed`); nil with the machine off
// except for the chunk itself.
func (e *Exec) AcceptRuns(quest, chunk string) ([]Ref, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.o.Enabled {
		return nil, ErrOff
	}
	vs, plan, diff, err := e.derive()
	if err != nil {
		return nil, err
	}
	e.lastChanged = nil
	if err := e.acceptLocked(vs, plan, diff, quest, chunk, nil); err != nil {
		return nil, err
	}
	if e.boardLegacy() {
		return []Ref{{quest, chunk}}, nil
	}
	return e.lastChanged, nil
}

// acceptLocked records an accept; ev (the auto-start controller) makes it an auto record with its log line. Callers
// hold e.mu and pass the views they just derived.
func (e *Exec) acceptLocked(vs []QuestView, plan Plan, diff int, quest, chunk string, ev *AutoEvent) error {
	if !e.boardLegacy() {
		return e.smAccept(vs, diff, quest, chunk, ev)
	}
	if _, cur, ok := chunkView(vs, quest, chunk); ok && (cur.State == Accepted || cur.State == InProgress || cur.State == Awaiting) {
		return nil // the same tap again is a no-op
	}
	if ev == nil {
		if _, cur, ok := chunkView(vs, quest, chunk); ok && afterOnly(cur) {
			// A chunk that only waits for earlier steps of its quest: accepting it accepts those steps too.
			chain, err := e.prereqChain(vs, quest, chunk)
			if err != nil {
				return err
			}
			return e.acceptChainLocked(vs, diff, quest, append(chain, chunk), "")
		}
	}
	if o := e.open(vs, plan, diff); len(o) > 0 {
		return fmt.Errorf("questboard: %s is already accepted; finish or stop it first (one chunk at a time)", o[0])
	}
	if err := e.canStart(vs, quest, chunk); err != nil {
		return err
	}
	finished := func(r Record) bool {
		_, c, ok := chunkView(vs, r.Quest, r.Chunk)
		return ok && (c.State == Done || c.State == Skipped)
	}
	var err error
	if ev != nil {
		err = e.st.AcceptAuto(quest, chunk, diff, e.o.Mode, finished, *ev)
	} else {
		err = e.st.AcceptIf(quest, chunk, diff, e.o.Mode, finished)
	}
	if err != nil {
		return err
	}
	delete(e.rs, ChunkKey(quest, chunk)) // a fresh start: no old counters or park
	e.o.Log("quest board: accepted "+ChunkKey(quest, chunk), "quest", quest, "chunk", chunk)
	return nil
}

// acceptChainLocked records the chunks (prerequisites first, the asked chunk last) in one save. repair is the
// log line of a board-load repair ("" for a tap).
func (e *Exec) acceptChainLocked(vs []QuestView, diff int, quest string, chunks []string, repair string) error {
	finished := func(r Record) bool {
		_, c, ok := chunkView(vs, r.Quest, r.Chunk)
		return ok && (c.State == Done || c.State == Skipped)
	}
	if err := e.st.AcceptChain(quest, chunks, diff, e.o.Mode, finished); err != nil {
		return err
	}
	for _, c := range chunks {
		delete(e.rs, ChunkKey(quest, c))
	}
	last, pre := chunks[len(chunks)-1], chunks[:len(chunks)-1]
	if repair != "" {
		e.o.Log(repair, "quest", quest, "chunk", last)
	} else {
		e.o.Log("quest board: accepted "+ChunkKey(quest, last)+" (after: "+strings.Join(pre, ", ")+")", "quest", quest, "chunk", last)
	}
	return nil
}

// afterOnly says the chunk is locked only because earlier chunks of its own quest are not done.
func afterOnly(c ChunkView) bool { return c.State == Locked && strings.HasPrefix(c.Why, "after ") }

// prereqChain lists the unfinished earlier chunks the chunk waits for (its After chunks, transitively), in
// questspec order. Each must be startable or already accepted; otherwise the error says which one to start first.
func (e *Exec) prereqChain(vs []QuestView, quest, chunk string) ([]string, error) {
	q, err := e.b.Quest(quest)
	if err != nil {
		return nil, err
	}
	need := map[string]bool{}
	var walk func(id string)
	walk = func(id string) {
		c, err := e.b.Chunk(quest, id)
		if err != nil {
			return
		}
		for _, a := range c.After {
			if !need[a] {
				need[a] = true
				walk(a)
			}
		}
	}
	walk(chunk)
	var out []string
	for _, c := range q.Chunks {
		if !need[c.ID] || c.ID == chunk {
			continue
		}
		_, cv, ok := chunkView(vs, quest, c.ID)
		if !ok || cv.State == Done {
			continue
		}
		switch {
		case cv.State == Ready || cv.State == Accepted || cv.State == InProgress || cv.State == Awaiting || afterOnly(cv):
			out = append(out, c.ID)
		default:
			return nil, fmt.Errorf("questboard: start %q first: it is %s%s", c.Title, cv.State, because(cv.Why))
		}
	}
	return out, nil
}

// repairLocked accepts the missing prerequisite steps of the first accepted chunk (a plan file that holds a
// chunk whose predecessor was never accepted, or was stopped). It reports whether it wrote records.
func (e *Exec) repairLocked(vs []QuestView, plan Plan, diff int, first Ref) bool {
	_, c, ok := chunkView(vs, first.Quest, first.Chunk)
	if !ok || !afterOnly(c) {
		return false
	}
	chain, err := e.prereqChain(vs, first.Quest, first.Chunk)
	if err != nil || len(chain) == 0 {
		return false
	}
	line := "quest board: " + first.Chunk + " waits for " + strings.Join(chain, ", ") + " (auto-started)"
	return e.acceptChainLocked(vs, diff, first.Quest, append(chain, first.Chunk), line) == nil
}

// Dependants lists the accepted chunks of the quest that wait, directly or not, for the chunk (what a Stop of it
// withdraws too).
func (e *Exec) Dependants(quest, chunk string) []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.dependantsLocked(quest, chunk)
}

func (e *Exec) dependantsLocked(quest, chunk string) []string {
	p, _, err := e.st.Load()
	q, qerr := e.b.Quest(quest)
	if err != nil || qerr != nil {
		return nil
	}
	diff := e.env.Diff()
	waits := func(id string) bool { // id waits for chunk, transitively
		seen := map[string]bool{}
		var walk func(string) bool
		walk = func(x string) bool {
			c, err := e.b.Chunk(quest, x)
			if err != nil {
				return false
			}
			for _, a := range c.After {
				if a == chunk || (!seen[a] && walk(a)) {
					return true
				}
				seen[a] = true
			}
			return false
		}
		return walk(id)
	}
	var out []string
	for _, c := range q.Chunks {
		if c.ID != chunk && p.accepted(diff, quest, c.ID) && waits(c.ID) {
			out = append(out, c.ID)
		}
	}
	return out
}

// HasRecord reports whether the plan file holds an accepted record of the chunk in this difficulty, whatever state
// the evidence derives for it (a queued chunk whose predecessor is not done derives Locked).
func (e *Exec) HasRecord(quest, chunk string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	p, _, err := e.st.Load()
	return err == nil && p.accepted(e.env.Diff(), quest, chunk)
}

// Cancel withdraws an accepted chunk (the player's Stop): the record goes, the running slice is stopped, the
// chunk is Ready again. Cancelling something that is not accepted is a no-op.
func (e *Exec) Cancel(quest, chunk string) error {
	_, err := e.CancelRuns(quest, chunk)
	return err
}

// CancelRuns is Cancel that also names the runs it closed: the chunk and every run that waited for it (D2).
func (e *Exec) CancelRuns(quest, chunk string) ([]Ref, error) {
	q, err := e.b.Quest(quest)
	if err != nil {
		return nil, err
	}
	c, err := e.b.Chunk(quest, chunk)
	if err != nil {
		return nil, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.o.Enabled {
		return nil, ErrOff
	}
	if !e.boardLegacy() {
		e.lastChanged = nil
		err := e.smCancel(q, c)
		return e.lastChanged, err
	}
	deps := e.dependantsLocked(quest, chunk)
	if err := e.st.Cancel(quest, chunk, e.env.Diff()); err != nil {
		return nil, err
	}
	out := []Ref{{quest, chunk}}
	for _, d := range deps { // a step that waited for it would only park, or start it again
		if err := e.st.Cancel(quest, d, e.env.Diff()); err != nil {
			return nil, err
		}
		delete(e.rs, ChunkKey(quest, d))
		out = append(out, Ref{quest, d})
	}
	delete(e.rs, ChunkKey(quest, chunk))
	e.act.Stop(q, c)
	return out, nil
}

// Reset cancels the running slice cleanly and forgets every counter and park. The plan records stay accepted:
// the chunk resumes from the evidence at the next tick (the player's Reset).
func (e *Exec) Reset() { e.reset(true) }

// reset is Reset; without parks it keeps the stored ones (the knob quests.board going off).
func (e *Exec) reset(parks bool) {
	e.NoteHold()
	e.mu.Lock()
	vs, plan, diff, err := e.derive()
	var act []Ref
	if err == nil {
		act = e.open(vs, plan, diff)
	}
	e.rs = map[string]*runState{}
	if parks && err == nil && (!e.boardLegacy() || plan.anyStoredPark(Difficulties[diff])) {
		// The player's Reset is the one explicit way a stored park is cleared. The legacy board honours stored parks too
		// (Next), so its Reset clears them as well; it writes the runs only when there is one (card fix1b, review B2).
		e.smRetryAll()
	}
	e.mu.Unlock()
	for _, r := range act {
		if q, qerr := e.b.Quest(r.Quest); qerr == nil {
			if c, cerr := e.b.Chunk(r.Quest, r.Chunk); cerr == nil {
				e.act.Stop(q, c)
			}
		}
	}
}

// Pause holds the running chunk (Next answers Paused); Resume continues. A chunk that is awaiting hand-in
// stays awaiting.
func (e *Exec) Pause(on bool) { e.mu.Lock(); e.paused = on; e.mu.Unlock() }

// Unblock clears the park of a chunk (the player's "Try again").
func (e *Exec) Unblock(quest, chunk string) { e.UnblockRuns(quest, chunk) }

// UnblockRuns is Unblock that also names the runs it changed.
func (e *Exec) UnblockRuns(quest, chunk string) []Ref {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.rs, ChunkKey(quest, chunk))
	e.lastChanged = nil
	if !e.boardLegacy() {
		e.smRetry(quest, chunk)
		return e.lastChanged
	}
	if plan, _, err := e.st.Load(); err == nil && plan.storedPark(quest, chunk, Difficulties[e.env.Diff()]) != nil {
		e.smRetry(quest, chunk) // the one write the legacy board makes to the runs: the player lifts a park the machine stored
	}
	return []Ref{{quest, chunk}}
}

// Status is what the board shows now.
type Status struct {
	Enabled bool
	Paused  bool
	Diff    int
	Views   []QuestView
	Active  *Active // the accepted chunk that is not done, nil when none
	// Extra lists the further accepted chunks of the plan file in chain order; they wait for Active.
	Extra []Ref
}

// Active is the accepted chunk that runs or waits for the tap.
type Active struct {
	Quest, Chunk string
	State        State
	Step         string
	Why          string // the Needs-you reason of a blocked chunk
	StepIdx      int    // index of the first open step in the chunk's list, -1 when none is open (waiting at the NPC)
	Failures     int
	Deaths       int
	Hold         string // the reason code of the hold the chunk is under (quests.boardSM), "" when none
	HoldWhy      string
}

// State derives the board now.
func (e *Exec) State() (Status, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.o.Enabled {
		return Status{}, ErrOff
	}
	vs, plan, diff, err := e.derive()
	if err != nil {
		return Status{}, err
	}
	st := Status{Enabled: true, Paused: e.paused, Diff: diff, Views: vs}
	o := e.open(vs, plan, diff)
	if len(o) == 0 {
		return st, nil
	}
	r := o[0]
	_, c, _ := chunkView(vs, r.Quest, r.Chunk)
	a := &Active{Quest: r.Quest, Chunk: r.Chunk, State: c.State, Step: c.StepName, Why: c.Why, StepIdx: c.Step}
	if rs := e.rs[ChunkKey(r.Quest, r.Chunk)]; rs != nil {
		a.Failures, a.Deaths = rs.failures, rs.deaths
		a.Hold, a.HoldWhy = rs.hold, rs.holdWhy
	}
	st.Active, st.Extra = a, o[1:]
	return st, nil
}

// Kind is what the bot does this tick.
type Kind string

const (
	DecIdle    Kind = "idle"    // no accepted chunk: the planner trains (S6 modes)
	DecRun     Kind = "run"     // do one slice of the chunk handler
	DecAwait   Kind = "await"   // objective met: stand at the NPC and wait for the player's tap
	DecBlocked Kind = "blocked" // parked with a visible reason
	DecPaused  Kind = "paused"
	DecBusy    Kind = "busy" // a hand-in dialogue is running
)

// Decision is the answer of Next.
type Decision struct {
	Kind         Kind
	Quest, Chunk string
	Why          string
}

// Next says what to do now. It reads evidence and the plan records only; calling it twice changes nothing.
func (e *Exec) Next() (Decision, error) {
	d, err := e.next()
	e.tryCheck(d, err)
	return d, err
}

func (e *Exec) next() (Decision, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.o.Enabled {
		return Decision{}, ErrOff
	}
	if e.handing {
		return Decision{Kind: DecBusy}, nil
	}
	vs, plan, diff, err := e.derive()
	if err != nil {
		return Decision{}, err
	}
	o := e.open(vs, plan, diff)
	if len(o) == 0 {
		return Decision{Kind: DecIdle}, nil
	}
	if e.boardLegacy() && e.repairLocked(vs, plan, diff, o[0]) { // with the machine the repair is the load step (store_sm.go)
		if vs, plan, diff, err = e.derive(); err != nil {
			return Decision{}, err
		}
		o = e.open(vs, plan, diff)
	}
	r := o[0]
	_, c, _ := chunkView(vs, r.Quest, r.Chunk)
	d := Decision{Quest: r.Quest, Chunk: r.Chunk, Why: c.Why}
	switch c.State {
	case Awaiting:
		d.Kind = DecAwait
	case Accepted, InProgress:
		d.Kind = DecRun
		if e.paused {
			d.Kind = DecPaused
		}
	default: // Blocked, or evidence that took the offer away (a prerequisite regressed)
		d.Kind = DecBlocked
		if d.Why == "" {
			d.Why = "chunk is " + string(c.State)
		}
	}
	return d, nil
}

// Report feeds the outcome of a slice back: counters, parks, deaths.
func (e *Exec) Report(d Decision, res StepResult) {
	e.mu.Lock()
	defer e.mu.Unlock()
	k := ChunkKey(d.Quest, d.Chunk)
	rs := e.seeded(d.Quest, d.Chunk)
	f0, d0 := rs.failures, rs.deaths
	defer func() {
		if rs.blocked == "" && (rs.failures != f0 || rs.deaths != d0) {
			e.smCount(d.Quest, d.Chunk, rs) // a park stores its own counters (smPark)
		}
	}()
	if res.Outcome != StepHeld {
		rs.hold, rs.holdWhy, rs.holdFrom = "", "", time.Time{} // anything but a hold lifts it
	}
	switch res.Outcome {
	case StepHeld:
		e.holdLocked(d, rs, res)
	case StepRan:
		rs.started, rs.failures = true, 0
		rs.refCode, rs.refN = "", 0
	case StepFailed:
		rs.started = true
		rs.lastWhy = res.Why
		if sameTryKind(res.Why) {
			// The same refusal of the world (the text names the exit, the cause and the cell, or the NPC) inside samePathWindow
			// is the same try: three retries from one cell in 27 s parked the Countess (live 2026-11-28, L3); three "no exit"
			// in 0.9 s parked the Cain scroll chunk from a side level (lab 2026-11-28, TASK-1120) and six "npc not found" in
			// 0.2 s the Duriel chunk (TASK-1126). The task layer prefixes its third report, hence the suffix match.
			now := e.o.Now()
			if rs.pathWhy != "" && strings.HasSuffix(res.Why, rs.pathWhy) && now.Sub(rs.pathAt) < samePathWindow {
				break
			}
			rs.pathWhy, rs.pathAt = res.Why, now
		}
		rs.failures++
		if e.refusalCount(rs, res) {
			rs.blocked = fmt.Sprintf("refused %d times in a row (%s): %s", rs.refN, rs.refCode, nonEmpty(res.Why, "the step was refused"))
			e.o.Log("quest board: parked "+k, "chunk", k, "why", rs.blocked)
			e.smPark(d.Quest, d.Chunk, "refused_repeatedly", rs)
		} else if rs.failures >= e.o.MaxFailures {
			rs.blocked = fmt.Sprintf("gave up after %d tries: %s", rs.failures, nonEmpty(res.Why, "the step kept failing"))
			e.o.Log("quest board: parked "+k, "chunk", k, "why", rs.blocked)
			e.smPark(d.Quest, d.Chunk, parkCode(res.Why), rs)
		}
	case StepDied:
		rs.refCode, rs.refN = "", 0
		if e.try == k {
			e.try = "" // a death ends the try: the gates hold the chunk again (the death is counted like any other)
			e.o.Log("override: try anyway ended: the hero died", "chunk", k)
		}
		rs.started = true
		rs.deaths++
		rs.lastWhy = nonEmpty(res.Why, "died")
		if rs.deaths >= e.o.MaxDeaths {
			rs.blocked = fmt.Sprintf("died %d times on this part: %s", rs.deaths, rs.lastWhy)
			e.o.Log("quest board: parked "+k, "chunk", k, "why", rs.blocked)
			e.smPark(d.Quest, d.Chunk, "died_too_often", rs)
		}
	}
}

// samePathWindow: an identical navigation refusal inside it is not a new try (U: judgment; the live retries were 13 s apart).
const samePathWindow = time.Minute

// parkCode is the code of a failure park: "no_path" when the step failed with a navigation refusal that names its cause
// (nav ErrNoPath text "no path: ... (cause=...)": the exit and the cause stay in the why), else "gave_up" (card cellar3gate).
func parkCode(why string) string {
	if strings.Contains(why, "nav: no path") && strings.Contains(why, "cause=") {
		return "no_path"
	}
	return "gave_up"
}

// sameTryMarks are the failure texts that say the world refused, not the hero: a navigation refusal that names its cause,
// a level with no known exit to the next one (nav.ErrNoExit) and an NPC that is not in the reachable part of the level
// (the step executor's Approach). Repeating the step at once cannot change any of them.
var sameTryMarks = []string{"nav: no exit to that level", "npc_out_of_reach"}

func sameTryKind(why string) bool {
	if parkCode(why) == "no_path" {
		return true
	}
	for _, m := range sameTryMarks {
		if strings.Contains(why, m) {
			return true
		}
	}
	return false
}

func nonEmpty(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

// Tick is Next plus one handler slice: what a driver without its own guard calls. The bot calls Next and
// Report itself because its slice runs inside the runner's guard.
func (e *Exec) Tick(ctx context.Context, step Stepper) (Decision, error) {
	d, err := e.Next()
	if err != nil || d.Kind != DecRun {
		return d, err
	}
	q, err := e.b.Quest(d.Quest)
	if err != nil {
		return d, err
	}
	c, err := e.b.Chunk(d.Quest, d.Chunk)
	if err != nil {
		return d, err
	}
	res := step(ctx, q, c)
	if ctx.Err() != nil {
		e.act.Stop(q, c)
		return d, nil
	}
	e.Report(d, res)
	return d, nil
}

// HandInOutcome is how a hand-in ended.
type HandInOutcome string

const (
	HandInDone       HandInOutcome = "done"       // T1/T2 confirm the quest word (or the chunk's exact check)
	HandInConfirming HandInOutcome = "confirming" // the reward is seen (T3) but no exact evidence yet; pays nothing until it lands
	HandInBlocked    HandInOutcome = "blocked"    // every attempt failed: parked with Why
)

// HandInResult is the reply of HandIn. Awards are the meta keys the evidence earned now (the S10 ledger
// refuses duplicates; empty for T3 and for a quest the board never ran).
type HandInResult struct {
	Outcome  HandInOutcome
	Already  bool // the chunk was already done: no dialogue ran
	Attempts int
	Why      string
	Awards   []Award
	Awarded  []Pay // what the ledger credited now (empty on a repeat, for T3 and without a wallet)
}

// HandIn is the player's tap: it needs the chunk Awaiting, performs the dialogue and verifies by read-back.
// An evidence conflict is not an error: while the quest word still reads "not done" the attempt is retried
// (at most Options.MaxHandIn per call), then the chunk parks with a reason. A repeated tap after success
// returns Already.
func (e *Exec) HandIn(ctx context.Context, quest, chunk string) (HandInResult, error) {
	q, err := e.b.Quest(quest)
	if err != nil {
		return HandInResult{}, err
	}
	c, err := e.b.Chunk(quest, chunk)
	if err != nil {
		return HandInResult{}, err
	}
	e.mu.Lock()
	if !e.o.Enabled {
		e.mu.Unlock()
		return HandInResult{}, ErrOff
	}
	if e.handing {
		e.mu.Unlock()
		return HandInResult{}, errors.New("questboard: a hand-in is already running")
	}
	vs, _, diff, err := e.derive()
	if err != nil {
		e.mu.Unlock()
		return HandInResult{}, err
	}
	qv, cv, _ := chunkView(vs, quest, chunk)
	switch {
	case cv.State == Done:
		e.mu.Unlock()
		aw := chunkAwards(qv, quest, chunk)
		return HandInResult{Outcome: confirmOutcome(cv), Already: true, Awards: aw, Awarded: e.settle(aw, diff)}, nil
	case cv.State != Awaiting:
		e.mu.Unlock()
		return HandInResult{}, fmt.Errorf("questboard: %s/%s is %s, not awaiting hand-in%s", quest, chunk, cv.State, because(cv.Why))
	}
	e.handing = true
	max := e.o.MaxHandIn
	e.mu.Unlock()
	defer func() { e.mu.Lock(); e.handing = false; e.mu.Unlock() }()

	res := HandInResult{}
	var last string
	for res.Attempts < max {
		if ctx.Err() != nil {
			return res, ctx.Err()
		}
		res.Attempts++
		last = "the reward did not register after the dialogue"
		if err := e.act.HandIn(ctx, q, c); err != nil {
			last = err.Error()
			e.o.Log("quest board: hand-in failed "+ChunkKey(quest, chunk), "quest", quest, "chunk", chunk, "attempt", res.Attempts, "err", err)
		}
		// The evidence is read after EVERY attempt: a dialogue that paid the reward and then reported an error is done,
		// and the next attempts would only find an NPC with nothing to say ("failed 3 times", review 2 finding 8).
		e.mu.Lock()
		vs, _, _, derr := e.derive()
		e.mu.Unlock()
		if derr != nil {
			return res, derr
		}
		qv, cv, _ := chunkView(vs, quest, chunk)
		if cv.State == Done {
			res.Outcome, res.Awards = confirmOutcome(cv), chunkAwards(qv, quest, chunk)
			res.Awarded = e.settle(res.Awards, diff)
			e.mu.Lock()
			delete(e.rs, ChunkKey(quest, chunk))
			e.mu.Unlock()
			e.o.Log("quest board: handed in "+ChunkKey(quest, chunk), "quest", quest, "chunk", chunk, "outcome", res.Outcome, "tier", cv.Tier.String())
			return res, nil
		}
	}
	res.Outcome = HandInBlocked
	res.Why = fmt.Sprintf("hand-in failed %d times: %s", res.Attempts, last)
	e.mu.Lock()
	rs := e.seeded(quest, chunk)
	rs.blocked = res.Why
	e.smPark(quest, chunk, "handin_failed", rs)
	e.mu.Unlock()
	e.o.Log("quest board: parked "+ChunkKey(quest, chunk), "chunk", ChunkKey(quest, chunk), "why", res.Why)
	return res, nil
}

func confirmOutcome(cv ChunkView) HandInOutcome {
	if cv.Confirmed {
		return HandInDone
	}
	return HandInConfirming
}

func chunkAwards(v QuestView, quest, chunk string) []Award {
	var out []Award
	for _, a := range v.Awards {
		if strings.HasPrefix(a.Key, "quest/"+quest+"/"+chunk+"/") || strings.HasPrefix(a.Key, "quest/"+quest+"/-/") {
			out = append(out, a)
		}
	}
	return out
}

// LegacyReadiness is the transitional stand-in for the readiness model (slice S2): the legacy clvl floor per
// class family, safe at the floor, risky within 2 levels, else too hard (design 1.1 "legacyFloor"). A quest
// without a floor for the family is safe. An unknown class is an error naming it.
func (b *Board) LegacyReadiness(class string, clvl int) (map[string]Word, error) {
	out := map[string]Word{}
	for _, q := range b.quests {
		floor, ok, err := b.Floor(q, class)
		if err != nil {
			return nil, err
		}
		r := ReadySafe
		if ok {
			switch {
			case clvl >= floor:
			case clvl >= floor-2:
				r = ReadyRisky
			default:
				r = ReadyTooHard
			}
		}
		for _, c := range q.Chunks {
			out[ChunkKey(q.ID, c.ID)] = r
		}
	}
	return out, nil
}

package task

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Outcome is what one Tick did.
type Outcome int

const (
	OutDone        Outcome = iota // the planner has nothing left
	OutRan                        // a step ran and Need is now false (or assumed)
	OutRetry                      // a step failed and will be tried again
	OutSkipped                    // a step was given up on (optional) or assumed done without proof
	OutBlocked                    // a group is blocked: needs_attention, planner moves on or the goal pauses
	OutDeath                      // died during the step; Runner.Recovering takes over, next tick resumes
	OutInterrupted                // parent context cancelled (guard); not counted as a failure
	OutInserted                   // prerequisite steps spliced in front
	OutWait                       // nothing runnable (waiting on Requires of skipped nodes)
)

func (o Outcome) String() string {
	return [...]string{"done", "ran", "retry", "skipped", "blocked", "death", "interrupted", "inserted", "wait"}[o]
}

// Result describes one tick for logs, events and the dashboard.
type Result struct {
	Outcome   Outcome
	StepID    string
	Group     string
	Kind      Kind
	Evidence  Evidence
	Attention string // set with OutBlocked: text for goal.Note / needs_attention
	Stuck     bool   // with OutBlocked: nothing else is runnable, the goal should pause
	Err       error
}

// Policy is the retry/escape policy (tasklayer.md sections 3 and 6). Counters live in memory only; a restart forgets
// them, which is safe because everything else is re-derived.
type Policy struct {
	MaxFails    int // failures of one step before escape (default 3)
	MaxTimeouts int // timeout retries before escape (default 1)
	MaxDeaths   int // deaths in one step before the group is blocked (default 3)
	MaxInsert   int // splice depth (default 2)
}

func (p Policy) norm() Policy {
	if p.MaxFails <= 0 {
		p.MaxFails = 3
	}
	if p.MaxTimeouts <= 0 {
		p.MaxTimeouts = 1
	}
	if p.MaxDeaths <= 0 {
		p.MaxDeaths = 3
	}
	if p.MaxInsert <= 0 {
		p.MaxInsert = 2
	}
	return p
}

// Hooks lets the bot react around a step. OnStart is where the Runner applies Constraints (NoTownPortal ...).
type Hooks struct {
	OnStart func(s Step)
	OnEnd   func(s Step, r Result)
}

// Executor runs one decision per Tick. It holds only failure counters, never a cursor.
type Executor struct {
	Env     Env
	Act     Actions
	Policy  Policy
	Hooks   Hooks
	Now     func() time.Time
	Journal []Result // optional ring for tests/dashboard; appended when non-nil via Record

	fails    map[string]int
	timeouts map[string]int
	deaths   map[string]int
	assumed  map[string]bool // steps whose Do succeeded but Need stays true on EvProxy/EvNone only
	skipped  map[string]bool
	blocked  map[string]string // group -> why
	finished map[string]bool
	inserted []Node
	insDepth int
	Record   bool

	// strikes survive ForgetFails (a new game): a chicken exit is a new game, so per-game counters can never reach
	// MaxDeaths. They expire after strikeWindow or when the character gains a level.
	strikes map[string]*strikeRec
	level   int // clvl of the last world read
	gold    int // gold of the last world read

	// kitBlocks remembers the gold and clvl a group was blocked at for a supply shortage ("not ready: 0 mana potions"):
	// that block is released when the character has earned kitRetryGold more or gained a level, instead of lasting until
	// the next game (EARLYGAME: a poor caster otherwise ends with every quest blocked and a self-pause).
	kitBlocks map[string]kitBlock
}

// kitRetryGold is the gold a supply-blocked quest waits for before it is tried again (U: about two potions).
const kitRetryGold = 60

type kitBlock struct{ gold, level int }

// strikeWindow is how long deaths, chickens and too-weak aborts of one quest count against it.
const strikeWindow = 90 * time.Minute

type strikeRec struct {
	at     []time.Time
	level  int // clvl of the latest strike
	parked bool
	why    string
}

// NewExecutor returns an executor with default policy.
func NewExecutor(env Env, act Actions) *Executor {
	return &Executor{Env: env, Act: act, fails: map[string]int{}, timeouts: map[string]int{}, deaths: map[string]int{},
		assumed: map[string]bool{}, skipped: map[string]bool{}, blocked: map[string]string{}, finished: map[string]bool{},
		strikes: map[string]*strikeRec{}}
}

// BlockedAll lists blocked groups and why (for needs_attention).
func (e *Executor) BlockedAll() map[string]string { return e.blocked }

// Finished implements Session: the group ran out of needed steps (completed, or only assumed).
func (e *Executor) Finished(group string) bool { return e.finished[group] }

// Blocked implements Session.
func (e *Executor) Blocked(group string) (string, bool) { why, ok := e.blocked[group]; return why, ok }

// Peek returns the step the next tick would run, for progress and phase text.
func (e *Executor) Peek(p Plan, w World) (Step, Evidence, bool) {
	n, ev, ok, _ := e.pick(p.Nodes, w)
	return n.Step, ev, ok
}

// Assumed reports whether a step is only assumed done (no proof); progress then reports Verified=false.
func (e *Executor) Assumed(id string) bool { return e.assumed[id] }

// Skipped reports whether a step was given up on.
func (e *Executor) Skipped(id string) bool { return e.skipped[id] }

// Reset forgets counters (new goal, new game, user "retry" on a needs_attention goal).
func (e *Executor) Reset() {
	e.fails, e.timeouts, e.deaths = map[string]int{}, map[string]int{}, map[string]int{}
	e.assumed, e.skipped, e.blocked, e.finished, e.inserted, e.insDepth = map[string]bool{}, map[string]bool{}, map[string]string{}, map[string]bool{}, nil, 0
	e.strikes = map[string]*strikeRec{}
	e.kitBlocks = nil
}

// ForgetFails drops the failure bookkeeping (fails, timeouts, deaths, skips, blocks, spliced steps) but keeps what was
// proven or assumed done. The bot calls it on a new game and after a game restart: a poverty spell or a bad area early
// in a long run must not block a group for the rest of the goal.
func (e *Executor) ForgetFails() {
	e.fails, e.timeouts, e.deaths = map[string]int{}, map[string]int{}, map[string]int{}
	e.skipped, e.blocked, e.inserted, e.insDepth = map[string]bool{}, map[string]string{}, nil, 0
	e.kitBlocks = nil
	e.expireStrikes()
	for g, r := range e.strikes { // a parked quest stays parked until its strikes expire
		if r.parked {
			e.blocked[g] = r.why
		}
	}
}

func (e *Executor) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

// expireStrikes drops strikes older than strikeWindow or from a lower clvl than the character has now.
func (e *Executor) expireStrikes() {
	now := e.now()
	for g, r := range e.strikes {
		keep := r.at[:0]
		for _, t := range r.at {
			if now.Sub(t) < strikeWindow {
				keep = append(keep, t)
			}
		}
		r.at = keep
		if len(keep) == 0 || (e.level > r.level && r.level > 0) {
			delete(e.strikes, g)
			if r.parked {
				delete(e.blocked, g)
			}
		}
	}
}

// strike records a death, chicken or too-weak abort of group g across games and reports whether the group is now
// parked (MaxDeaths strikes within strikeWindow at this clvl).
func (e *Executor) strike(g, why string) bool {
	if g == "" {
		return false
	}
	if e.strikes == nil {
		e.strikes = map[string]*strikeRec{}
	}
	e.expireStrikes()
	r := e.strikes[g]
	if r == nil {
		r = &strikeRec{}
		e.strikes[g] = r
	}
	r.at, r.level = append(r.at, e.now()), e.level
	if len(r.at) >= e.Policy.norm().MaxDeaths {
		r.parked, r.why = true, fmt.Sprintf("%s; %d failed attempts in %v, parked until it expires or the character levels up", why, len(r.at), strikeWindow)
		return true
	}
	return false
}

// satisfied: a node no longer blocks its dependents.
func (e *Executor) satisfied(s Step, w World) bool {
	if e.skipped[s.ID] || e.assumed[s.ID] {
		return true
	}
	need, _ := s.Need(w)
	return !need
}

// pick chooses the first node that is needed and whose Requires are satisfied.
func (e *Executor) pick(nodes []Node, w World) (Node, Evidence, bool, bool) {
	byID := map[string]Step{}
	for _, n := range nodes {
		byID[n.Step.ID] = n.Step
	}
	pending := false
	for _, n := range nodes {
		if e.satisfied(n.Step, w) {
			continue
		}
		pending = true
		ok := true
		for _, r := range n.Requires {
			if rs, found := byID[r]; found && !e.satisfied(rs, w) {
				ok = false
				break
			}
		}
		if ok {
			_, ev := n.Step.Need(w)
			return n, ev, true, true
		}
	}
	return Node{}, EvNone, false, pending
}

// Tick makes one decision and runs at most one step.
func (e *Executor) Tick(ctx context.Context, p Planner) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{Outcome: OutInterrupted, Err: err}, nil
	}
	w, err := e.Env.World(ctx)
	if err != nil {
		return Result{Outcome: OutInterrupted, Err: err}, err
	}
	e.level, e.gold = w.Level, w.Gold
	e.expireStrikes()
	e.releaseKitBlocks(w)
	plan := p.Plan(w, e)
	if plan.Done {
		return Result{Outcome: OutDone}, nil
	}
	if plan.Stuck {
		return Result{Outcome: OutBlocked, Stuck: true, Attention: "needs_attention: " + plan.Why}, nil
	}
	live := e.inserted[:0]
	for _, n := range e.inserted { // spliced prerequisites fall away once satisfied
		if !e.satisfied(n.Step, w) {
			live = append(live, n)
		}
	}
	e.inserted = live
	if len(live) == 0 {
		e.insDepth = 0
	}
	var (
		n           Node
		ev          Evidence
		ok, pending bool
		r           Result
	)
	// A Ready fault with prerequisites splices them in front and the first one runs in this same tick, so a restart
	// between ticks cannot lose the splice.
	for try := 0; ; try++ {
		nodes := append(append([]Node(nil), e.inserted...), plan.Nodes...)
		n, ev, ok, pending = e.pick(nodes, w)
		if !ok {
			break
		}
		r = Result{StepID: n.Step.ID, Group: plan.Group, Evidence: ev}
		if n.Step.Ready != nil {
			if f := n.Step.Ready(w); f != nil {
				r = e.handleFault(plan, n.Step, f, r)
				if r.Outcome == OutInserted && try < e.Policy.norm().MaxInsert {
					continue
				}
				return r, nil
			}
		}
		return e.run(ctx, w, plan, n.Step, r), nil
	}
	if !ok {
		if pending {
			return Result{Outcome: OutWait, Group: plan.Group}, nil
		}
		// nothing needed: the world moved on, or only skipped/assumed nodes are left, which finishes the group for
		// this session so the planner advances.
		if plan.Group != "" {
			e.finished[plan.Group] = true
		}
		return Result{Outcome: OutRan, Group: plan.Group}, nil
	}
	return Result{}, nil
}

func (e *Executor) run(ctx context.Context, w World, plan Plan, s Step, r Result) Result {
	runCtx := ctx
	if s.Budget > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, s.Budget)
		defer cancel()
	}
	if e.Hooks.OnStart != nil {
		e.Hooks.OnStart(s)
	}
	err := s.Do(runCtx, e.Act)
	// A deadline on the caller's ctx (the Runner's step budget) is an overrun and counts as a timeout; a plain cancel
	// (the Runner's guard) is not the step's fault.
	if err != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		err = Wrap(KindTimeout, err)
	} else if err != nil && ctx.Err() != nil {
		r.Outcome, r.Err = OutInterrupted, ctx.Err()
		return e.end(s, r)
	}
	if err != nil {
		var f *Fault
		if s.Fail != nil {
			f = s.Fail(w, err)
		}
		if f == nil {
			f = &Fault{Kind: Classify(err), Why: err.Error()}
		}
		r.Err = err
		return e.end(s, e.handleRunFault(plan, s, f, r))
	}
	// Do returned nil: confirm with a fresh read; Need must be false, or we count it.
	w2, werr := e.Env.World(ctx)
	if werr != nil {
		r.Outcome, r.Err = OutInterrupted, werr
		return e.end(s, r)
	}
	need, ev := s.Need(w2)
	if !need {
		e.fails[s.ID], e.timeouts[s.ID] = 0, 0
		r.Outcome = OutRan
		return e.end(s, r)
	}
	if s.Repeat { // a bounded slice of a longer job (leveling): partial progress is normal
		r.Outcome = OutRan
		return e.end(s, r)
	}
	if ev <= EvProxy { // no reader: trust a clean Do for this session only
		e.assumed[s.ID] = true
		r.Outcome, r.Evidence = OutSkipped, ev
		return e.end(s, r)
	}
	why := fmt.Sprintf("%s ran but is still needed (evidence %d)", s.ID, ev)
	if s.Optional {
		// A clean Do that changed nothing the readers can see will not change it the second time either (LIVE G1 attempt
		// 14: the Charsi greeting ran twice, 25 s, before it was skipped).
		r.Err = errors.New(why)
		return e.end(s, e.escape(plan, s, r, why))
	}
	return e.end(s, e.failure(plan, s, r, why))
}

func (e *Executor) end(s Step, r Result) Result {
	if e.Hooks.OnEnd != nil {
		e.Hooks.OnEnd(s, r)
	}
	if e.Record {
		e.Journal = append(e.Journal, r)
	}
	return r
}

// failure counts one failure of s and escapes after MaxFails.
func (e *Executor) failure(plan Plan, s Step, r Result, why string) Result {
	pol := e.Policy.norm()
	e.fails[s.ID]++
	if e.fails[s.ID] < pol.MaxFails {
		r.Outcome = OutRetry
		r.Err = errors.New(why)
		return r
	}
	return e.escape(plan, s, r, why)
}

// escape gives up on s: optional steps are skipped, others block their group (the planner moves on).
func (e *Executor) escape(plan Plan, s Step, r Result, why string) Result {
	e.dropInserted(s.ID)
	if s.Optional {
		e.skipped[s.ID] = true
		r.Outcome = OutSkipped
		return r
	}
	return e.block(plan, s, r, why)
}

// dropInserted forgets a spliced prerequisite that was given up on. Inserted nodes run before the planner's own, so a
// blocked one that stayed would be re-picked every tick and starve every other step (LIVE: chores that cannot buy
// potions blocked about once a second and the level step never ran).
func (e *Executor) dropInserted(id string) {
	keep := e.inserted[:0]
	for _, n := range e.inserted {
		if n.Step.ID != id {
			keep = append(keep, n)
		}
	}
	e.inserted = keep
}

// kitFault is the Why of a supply shortage from gateReady (a level shortage says "clvl" and is never released here).
func kitFault(why string) bool {
	return strings.HasPrefix(why, "not ready: ") && !strings.Contains(why, "clvl")
}

func (e *Executor) releaseKitBlocks(w World) {
	for g, kb := range e.kitBlocks {
		if w.Gold < kb.gold+kitRetryGold && w.Level <= kb.level {
			continue
		}
		delete(e.kitBlocks, g)
		delete(e.blocked, g)
		for _, m := range []map[string]int{e.fails, e.timeouts, e.deaths} {
			for id := range m {
				if strings.HasPrefix(id, "quest."+g+"/") {
					delete(m, id)
				}
			}
		}
	}
}

func (e *Executor) block(plan Plan, s Step, r Result, why string) Result {
	e.dropInserted(s.ID)
	if plan.Group != "" {
		e.blocked[plan.Group] = s.ID + ": " + why
		if kitFault(why) {
			if e.kitBlocks == nil {
				e.kitBlocks = map[string]kitBlock{}
			}
			e.kitBlocks[plan.Group] = kitBlock{gold: e.gold, level: e.level}
		}
	}
	r.Outcome = OutBlocked
	r.Attention = fmt.Sprintf("needs_attention: %s (%s) %s", s.Title, s.ID, why)
	return r
}

func (e *Executor) handleRunFault(plan Plan, s Step, f *Fault, r Result) Result {
	r.Kind = f.Kind
	pol := e.Policy.norm()
	switch f.Kind {
	case KindDeath:
		e.deaths[s.ID]++
		if e.deaths[s.ID] >= pol.MaxDeaths {
			return e.block(plan, s, r, fmt.Sprintf("died %d times; level up or change gear", e.deaths[s.ID]))
		}
		if e.strike(plan.Group, "died or left the game in "+s.Title) {
			return e.block(plan, s, r, e.strikes[plan.Group].why)
		}
		r.Outcome = OutDeath
		return r
	case KindFatal:
		r.Outcome = OutInterrupted
		return r
	case KindTimeout:
		e.timeouts[s.ID]++
		if e.timeouts[s.ID] <= pol.MaxTimeouts {
			r.Outcome = OutRetry
			return r
		}
		return e.escape(plan, s, r, "timeout: "+f.Why)
	case KindPrereq, KindResource, KindDanger:
		if f.Kind == KindDanger && e.strike(plan.Group, "too weak for "+s.Title) {
			return e.block(plan, s, r, e.strikes[plan.Group].why)
		}
		return e.handleFault(plan, s, f, r)
	}
	// NotFound and UI: count; the step's own Do rescans on the retry.
	return e.failure(plan, s, r, f.Why)
}

// givenUp reports whether a prerequisite already failed MaxFails times: splicing it in again would only repeat the
// failure, so the faulting step takes the fail path instead.
func (e *Executor) givenUp(ins []Step) bool {
	pol := e.Policy.norm()
	for _, s := range ins {
		if e.fails[s.ID] >= pol.MaxFails || e.timeouts[s.ID] > pol.MaxTimeouts || e.deaths[s.ID] >= pol.MaxDeaths {
			return true
		}
	}
	return false
}

// handleFault deals with a Ready fault (or a run fault of a precondition kind).
func (e *Executor) handleFault(plan Plan, s Step, f *Fault, r Result) Result {
	r.Kind = f.Kind
	pol := e.Policy.norm()
	if len(f.Insert) > 0 && e.insDepth < pol.MaxInsert && !e.givenUp(f.Insert) {
		e.insDepth++
		front := make([]Node, 0, len(f.Insert)+len(e.inserted))
		for _, is := range f.Insert {
			front = append(front, Node{Step: is})
		}
		e.inserted = append(front, e.inserted...) // prerequisites of a prerequisite run first
		r.Outcome = OutInserted
		return r
	}
	if f.Kind == KindDanger || f.Kind == KindPrereq || f.Kind == KindResource {
		e.fails[s.ID]++
		if e.fails[s.ID] >= pol.MaxFails {
			return e.escape(plan, s, r, f.Why)
		}
		r.Outcome = OutRetry
		r.Err = errors.New(f.Why)
		return r
	}
	return e.failure(plan, s, r, f.Why)
}

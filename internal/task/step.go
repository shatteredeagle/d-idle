package task

import (
	"context"
	"time"
)

// Constraints are checked by the Runner when it starts the step (tasklayer.md section 3). The executor reports them
// through Hooks.OnStart.
type Constraints struct {
	NoTownPortal   bool // suppress Fighter.Safety TownPortal and the "town trip due" guard cause (Ancients)
	NoLeaveArea    bool // guard aborts instead of walking home
	NeedsTeleport  bool
	ExclusiveInput bool
}

// Step is one unit of work. Need and Ready are pure functions of the World; Do is idempotent and ends with a state
// change Need can see (a Checkpoint). Done is simply !Need.
type Step struct {
	ID    string // stable: "quest.den_of_evil/clear"
	Title string // dashboard text
	Phase string // "travel", "fight", "turn-in", "chore", "level"; becomes goals.Progress.Phase

	// Need reports whether the step still has to run and how sure that is. Never skip a turn-in on EvProxy alone.
	Need func(w World) (bool, Evidence)
	// Ready checks preconditions; nil = go. May return Insert steps.
	Ready func(w World) *Fault
	// Do performs the work with the capabilities in a.
	Do func(ctx context.Context, a Actions) error
	// Fail may rewrite a Do error into a Fault (optional); default is Classify(err).
	Fail func(w World, err error) *Fault

	Budget      time.Duration // hard cap per run; over = KindTimeout
	Est         time.Duration // expected duration, used for ETA
	Constraints Constraints
	Repeat      bool // Do is a bounded slice; Need staying true afterwards is progress, not a failure
	Optional    bool // may be skipped after repeated NotFound/Timeout without blocking its group
}

// Done reports !Need.
func (s Step) Done(w World) bool { n, _ := s.Need(w); return !n }

// Node places a Step in a group's graph. Requires lists step IDs (within the plan) that must not be needed any
// more (done or skipped) before this node can run. With no Requires a node follows list order.
type Node struct {
	Step     Step
	Requires []string
}

// Plan is the work for the current decision: data built by pure functions, no goroutines.
type Plan struct {
	ID    string // "quest.den_of_evil", "level.8"
	Group string // quest id the nodes belong to ("" for level plans)
	Why   string // "auto: need clvl 22 for Andariel"
	Nodes []Node
	Done  bool // nothing left in the whole goal
	Stuck bool // work remains but every remaining group is blocked (Why says which)
}

// Session is the executor's per-session memory the planner routes around: groups it finished without proof of
// completion (no quest reader) and groups it gave up on.
type Session interface {
	Finished(group string) bool
	Blocked(group string) (why string, blocked bool)
}

// Planner builds the plan for the world.
type Planner interface {
	Plan(w World, s Session) Plan
}

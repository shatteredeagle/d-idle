package questspec

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// The generic step executor (card stepexec1, review fable-logic-review-2026-11-25 P9, docs/design/stepexec-p9.md).
// A step of the five verbs interact, pickup, talk, use and transmute is: APPROACH the target, ACT, WAIT for the declared fact
// (the step's post clauses, read from the snapshot), retry within a budget. The engine is pure: it knows no game, only
// a Backend. Its result is the typed step outcome of internal/bot/outcome.go (same five kinds, mapped by the bot
// adapter; this package cannot import bot).

// Verbs the executor runs.
var ExecVerbs = set("interact", "pickup", "talk", "use", "transmute")

// Exec is the executor row of a step (data, ADR 0021): how to run it. Absent: the step is not run by the executor.
type Exec struct {
	Verb string `json:"verb"` // interact | pickup | talk | use | transmute
	// On is what the verb acts on; empty: the step's own target. The Cain scroll interacts with the tree (On) to make
	// the item (Target) appear.
	On *Target `json:"on,omitempty"`
	// Then is a second verb run on the step's target on every wait poll (pickup after the click: the scroll lies on the
	// ground, the post says it is in the pack).
	Then string `json:"then,omitempty"`
	// Pick is the dialogue option a talk step picks ("bark", "reward"; the bot's dialog table names the lines). Empty: the
	// NPC's default line.
	Pick string `json:"pick,omitempty"`
	// Inputs are the base codes a transmute step puts in the Horadric Cube (the step's target is the product).
	Inputs       []string `json:"inputs,omitempty"`
	Tolerance    int      `json:"tolerance,omitempty"`  // approach: subtiles from the target that count as arrived
	WaitPolls    int      `json:"wait_polls"`           // looks at the post after an act
	WaitMS       int      `json:"wait_ms"`              // between two looks
	BackoffMS    int      `json:"backoff_ms,omitempty"` // pause before attempt 2; doubles per attempt, capped at BackoffCapMS
	BackoffCapMS int      `json:"backoff_cap_ms,omitempty"`
	// Wake names the facts that end a refusal (shown in Refused.Wake when the backend gives none).
	Wake []string `json:"wake,omitempty"`
}

func (e *Exec) validate(s Step, bad func(string, ...any)) {
	if !ExecVerbs[e.Verb] {
		bad("step %s: exec verb %q (want interact, pickup, talk, use or transmute)", s.ID, e.Verb)
	}
	if e.Then != "" && !ExecVerbs[e.Then] {
		bad("step %s: exec then %q", s.ID, e.Then)
	}
	if e.WaitPolls < 1 || e.WaitMS < 0 || e.Tolerance < 0 || e.BackoffMS < 0 || e.BackoffCapMS < 0 {
		bad("step %s: exec bounds (wait_polls >= 1, the rest >= 0)", s.ID)
	}
	if len(s.Post) == 0 {
		bad("step %s: exec without a post: the executor waits for a declared fact", s.ID)
	}
	if s.Tries < 1 {
		bad("step %s: exec without tries: the retry budget is part of the row", s.ID)
	}
	if e.On != nil && e.On.Kind != "object" && e.On.Kind != "npc" && e.On.Kind != "item" {
		bad("step %s: exec on kind %q", s.ID, e.On.Kind)
	}
	on := s.Target
	if e.On != nil {
		on = *e.On
	}
	if e.Verb == "talk" && on.Kind != "npc" || e.Verb == "use" && on.Kind != "item" {
		bad("step %s: exec verb %s acts on a %s, not %q", s.ID, e.Verb, map[string]string{"talk": "npc", "use": "item"}[e.Verb], on.Kind)
	}
	if e.Pick != "" && e.Verb != "talk" {
		bad("step %s: exec pick %q needs the verb talk", s.ID, e.Pick)
	}
	if e.Verb == "transmute" {
		if on.Kind != "item" || len(e.Inputs) < 2 {
			bad("step %s: exec transmute makes an item (target kind item) from at least two inputs", s.ID)
		}
		for _, c := range e.Inputs {
			if !itemCode.MatchString(c) {
				bad("step %s: exec input %q is not an item code", s.ID, c)
			}
		}
	} else if len(e.Inputs) > 0 {
		bad("step %s: exec inputs need the verb transmute", s.ID)
	}
}

// ActArgs are the row's arguments of one act: the dialogue option of a talk, the Cube inputs of a transmute.
type ActArgs struct {
	Pick   string
	Inputs []string
}

func (e *Exec) args() ActArgs { return ActArgs{Pick: e.Pick, Inputs: e.Inputs} }

// Backend is what the executor needs of the game. Every fact comes from the snapshot, never from the executor's own
// bookkeeping: a restarted bot reads the same answer.
type Backend interface {
	// Approach walks until the target is within tol subtiles. ErrProgress: moved, not there yet.
	Approach(ctx context.Context, t Target, tol int) error
	// Act performs the verb on the target (click, pick up, say a line, use an item, transmute in the Cube).
	Act(ctx context.Context, verb string, t Target, args ActArgs) error
	// Holds reads one post clause from the current snapshot.
	Holds(c Clause) bool
	Sleep(d time.Duration)
}

// Backend errors the engine understands; anything else is a failed attempt (retried within the budget).
var (
	ErrProgress = errors.New("progress: moved, not arrived")
	// ErrNothing: the act ran and nothing is there yet (no item on the ground). Not a fault: the wait goes on.
	ErrNothing = errors.New("nothing there yet")
)

// NeedError is a refusal: the step may not run now. Need is the closed code (bot needTable), Wake the fact that ends it.
type NeedError struct{ Need, Wake, Why string }

func (e *NeedError) Error() string { return fmt.Sprintf("refused (%s): %s", e.Need, e.Why) }

// FatalError ends the step at once with this fault (no retry): the input is gone, the target does not exist.
type FatalError struct{ Err error }

func (e *FatalError) Error() string { return e.Err.Error() }
func (e *FatalError) Unwrap() error { return e.Err }

type Kind uint8

const (
	Done Kind = iota
	Progress
	Refused
	Failed
	Interrupted
)

func (k Kind) String() string {
	return [...]string{"done", "progress", "refused", "failed", "interrupted"}[k]
}

// Outcome is the typed result of one run of a step.
type Outcome struct {
	Kind     Kind
	Need     string // Refused
	Wake     string // Refused
	Why      string // Refused: the words of the guard
	Err      error  // Failed: the fault; Interrupted: the cause
	Attempts int    // acts made
}

// RunStep runs one step with the executor row. The post is checked BEFORE the first act (a step already satisfied is
// Done and touches nothing: resumption after a restart), after every act and on every wait poll.
func RunStep(ctx context.Context, s Step, be Backend) Outcome {
	e := s.Exec
	if e == nil {
		return Outcome{Kind: Failed, Err: fmt.Errorf("step %s has no exec row", s.ID)}
	}
	posts := s.PostClauses()
	holds := func() bool {
		for _, c := range posts {
			if !be.Holds(c) {
				return false
			}
		}
		return true
	}
	on := s.Target
	if e.On != nil {
		on = *e.On
	}
	out := Outcome{}
	if holds() {
		return out
	}
	var last error
	for try := 0; try < s.Tries; try++ {
		if try > 0 {
			be.Sleep(backoff(e, try))
		}
		if err := ctx.Err(); err != nil {
			return Outcome{Kind: Interrupted, Err: err, Attempts: out.Attempts}
		}
		if o, stop := classify(be.Approach(ctx, on, e.Tolerance), out.Attempts, e); stop {
			return o
		} else if o.Err != nil {
			last = o.Err
			continue
		}
		out.Attempts++
		if o, stop := classify(be.Act(ctx, e.Verb, on, e.args()), out.Attempts, e); stop {
			return o
		} else if o.Err != nil {
			last = o.Err
			continue
		}
		for poll := 0; poll < e.WaitPolls; poll++ {
			if holds() {
				return out
			}
			if e.Then != "" {
				if o, stop := classify(be.Act(ctx, e.Then, s.Target, ActArgs{Pick: e.Pick}), out.Attempts, e); stop {
					return o
				}
				if holds() {
					return out
				}
			}
			if err := ctx.Err(); err != nil {
				return Outcome{Kind: Interrupted, Err: err, Attempts: out.Attempts}
			}
			be.Sleep(time.Duration(e.WaitMS) * time.Millisecond)
		}
		last = fmt.Errorf("%s done, but post %v not observed after %d looks", e.Verb, s.Post, e.WaitPolls)
	}
	return Outcome{Kind: Failed, Attempts: out.Attempts, Err: fmt.Errorf("step %s: %d tries used: %w", s.ID, s.Tries, last)}
}

// classify turns a backend error into an outcome. stop true: the run ends with o. stop false and o.Err set: a failed
// attempt, try again. stop false and no error: carry on.
func classify(err error, attempts int, e *Exec) (o Outcome, stop bool) {
	var need *NeedError
	var fatal *FatalError
	switch {
	case err == nil, errors.Is(err, ErrNothing):
		return Outcome{}, false
	case errors.Is(err, ErrProgress):
		return Outcome{Kind: Progress, Attempts: attempts}, true
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return Outcome{Kind: Interrupted, Err: err, Attempts: attempts}, true
	case errors.As(err, &need):
		wake := need.Wake
		if wake == "" && len(e.Wake) > 0 {
			wake = e.Wake[0]
		}
		return Outcome{Kind: Refused, Need: need.Need, Wake: wake, Why: need.Why, Attempts: attempts}, true
	case errors.As(err, &fatal):
		return Outcome{Kind: Failed, Err: fatal.Err, Attempts: attempts}, true
	}
	return Outcome{Err: err}, false
}

func backoff(e *Exec, try int) time.Duration {
	d := time.Duration(e.BackoffMS) * time.Millisecond << (try - 1)
	if c := time.Duration(e.BackoffCapMS) * time.Millisecond; c > 0 && d > c {
		d = c
	}
	return d
}

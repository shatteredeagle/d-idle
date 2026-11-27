package bot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/task"
)

// The typed step outcome (docs/outcomes.md, knob bot.typedOutcome, review fable-logic-review-2026-11-25 P2). A step ends as
// Done, Progress, Refused{need, wake}, Failed{fault} or Interrupted{cause}. A refusal is data: it names what is missing
// and the fact that ends it, and ONE handler (handleRefused, called from do) deals with it: it never counts as a failure.
// Stage 1 converts one path, the quest board step (boardSlice); every other path still returns plain errors.

type OutcomeKind uint8

const (
	OutDoneKind        OutcomeKind = iota // the step finished
	OutProgressKind                       // the step ran and more is left
	OutRefusedKind                        // the step may not run now: a need names why, a wake fact says when to look again
	OutFailedKind                         // a real fault (game, input, navigation)
	OutInterruptedKind                    // cancelled from outside (guard, player, new goal); not a failure
)

func (k OutcomeKind) String() string {
	return [...]string{"done", "progress", "refused", "failed", "interrupted"}[k]
}

// StepOutcome is the typed result of one step.
type StepOutcome struct {
	Kind OutcomeKind
	Need string // Refused: the closed code of what is missing (needTable)
	Wake string // Refused: the fact that ends the refusal (needTable)
	Why  string // Refused: the words of the guard, for the log
	Err  error  // Failed: the fault; Interrupted: the cause
}

// needRow is one refusal need: the fact that wakes it, and the pause between two looks while that fact is unchanged.
type needRow struct {
	Wake     string
	PauseSec int
}

// needTable is data: a new refusal reason is a row. The codes are those of the quest board (task.Fault.Code); the row
// "not_ready" is the fallback for a guard that gave no code.
var needTable = map[string]needRow{
	"needs_potions":   {"potions or portal scroll in the pack", 5},
	"needs_gold_farm": {"gold in purse or stash rises", 15},
	"too_weak":        {"character level rises", 15},
	"route_closed":    {"kit, level or gold change", 10},
	"area_banned":     {"the ban ends or the level rises", 15},
	"not_ready":       {"any of level, kit, gold or area changes", 5},
	"parked":          {"the park ends or the level rises", 5},
}

// refusalOf is the typed refusal for a guard's words (need from the words), for holds that are not step errors.
func refusalOf(why string) *Refused {
	need := needOf("", why)
	return &Refused{StepOutcome{Kind: OutRefusedKind, Need: need, Wake: needTable[need].Wake, Why: why}}
}

// needOf maps a refused result to a need code: the board's own code when it has one, else the words of the guard.
func needOf(code, why string) string {
	if _, ok := needTable[code]; ok {
		return code
	}
	w := strings.ToLower(why)
	switch {
	case strings.Contains(w, "early road"), strings.Contains(w, "route guard"), strings.Contains(w, "tower guard"):
		return "route_closed"
	case strings.Contains(w, "banned"):
		return "area_banned"
	case strings.Contains(w, "gold"):
		return "needs_gold_farm"
	case strings.Contains(w, "potion"), strings.Contains(w, "kit"):
		return "needs_potions"
	case strings.Contains(w, "clvl"), strings.Contains(w, "level"), strings.Contains(w, "too weak"):
		return "too_weak"
	}
	return "not_ready"
}

// refusedResult reports a step result that is a refusal: the guard said "not ready" or "danger" and nothing is wrong with
// the game or the input. Same test as planningFault without the not-found kind and without the executor's own stuck pause.
func refusedResult(res task.Result) bool {
	if res.Stuck || res.Err == nil || res.Outcome != task.OutRetry && res.Outcome != task.OutBlocked {
		return false
	}
	return res.Kind == task.KindDanger || strings.Contains(res.Err.Error(), "not ready")
}

// classify turns an executor result into the typed outcome.
func classify(res task.Result) StepOutcome {
	switch res.Outcome {
	case task.OutDone:
		return StepOutcome{Kind: OutDoneKind}
	case task.OutInterrupted, task.OutDeath:
		return StepOutcome{Kind: OutInterruptedKind, Err: res.Err}
	case task.OutRetry, task.OutBlocked:
		if refusedResult(res) {
			need := needOf(res.Code, res.Err.Error())
			return StepOutcome{Kind: OutRefusedKind, Need: need, Wake: needTable[need].Wake, Why: res.Err.Error()}
		}
		if res.Err != nil {
			return StepOutcome{Kind: OutFailedKind, Err: res.Err}
		}
	}
	return StepOutcome{Kind: OutProgressKind}
}

// Refused is a refusal travelling as an error through the legacy func(ctx) error actions; do unwraps it.
type Refused struct{ StepOutcome }

func (e *Refused) Error() string { return fmt.Sprintf("refused (%s): %s", e.Need, e.Why) }
func (e *Refused) Unwrap() error {
	if e.Err != nil {
		return e.Err // a gate refusal keeps its sentinel (errAreaGated) for errors.Is
	}
	return errors.New(e.Why)
}

// refuse turns a legacy refusal error into the typed one when the knob is on (docs/outcomes.md); the legacy error stays
// reachable through Unwrap, so errors.Is(err, errAreaGated) and the wording tests keep working. Off: the error itself.
func (r *Runner) refuse(legacy error) error {
	if legacy == nil || r.e.Tune.Bot.TypedOutcome == 0 {
		return legacy
	}
	var ref *Refused
	if errors.As(legacy, &ref) {
		return legacy
	}
	need := needOf("", legacy.Error())
	return &Refused{StepOutcome{Kind: OutRefusedKind, Need: need, Wake: needTable[need].Wake, Why: legacy.Error(), Err: legacy}}
}

// asError is the adapter for code that still returns errors: nil for Done and Progress, *Refused for a refusal.
func (o StepOutcome) asError() error {
	switch o.Kind {
	case OutRefusedKind:
		return &Refused{o}
	case OutFailedKind, OutInterruptedKind:
		return o.Err
	}
	return nil
}

// refusalState is what the handler remembers between two refusals (guarded by the step goroutine, like the loop state).
type refusalState struct {
	key   string    // need and why of the last logged refusal
	facts string    // wakeFacts when it was logged
	since time.Time // when this refusal began
	n     int       // refusals with this key and these facts
}

// wakeFacts is the fingerprint of the facts a refusal can wait on: level, belt and pack potions, portal scrolls, gold
// in 50-gold steps, area. A change is a wake: the next look is not delayed.
func (r *Runner) wakeFacts(s *d2.Snapshot) string {
	if s == nil || s.Player == nil {
		return ""
	}
	p := s.Player
	k := r.kitOf(s)
	return fmt.Sprintf("%d|%d|%d|%d|%d|%d", p.Level, k.BeltHP+k.PackHP, k.BeltMP, k.TownPortals, (p.Gold+p.StashGold)/50, p.Area)
}

// handleRefused is the ONE place a refusal is dealt with: no failure count, one INFO line per change of need or facts,
// and a pause that grows while nothing it waits on has changed (the next tick re-plans around the refusal).
func (r *Runner) handleRefused(ctx context.Context, ref *Refused) {
	l := &r.loop
	l.failures = 0
	now := r.e.Clock.Now()
	facts := r.wakeFacts(r.latest())
	key := ref.Need + "|" + ref.Why
	st := &l.refusal
	if key != st.key || facts != st.facts {
		if key != st.key {
			st.since = now
		}
		st.key, st.facts, st.n = key, facts, 0
		r.e.Log.Info("step refused", "need", ref.Need, "wake", ref.Wake, "why", ref.Why)
	}
	st.n++
	r.noteRefused("step:"+ref.Need, ref)
	r.inv.noteRefusal(key)
	r.arbRefused(ref.Need, ref.Wake)
	row := needTable[ref.Need]
	pause := time.Duration(row.PauseSec) * time.Second
	if st.n == 1 { // the first look after a change is cheap: the planner may already have another step
		pause = time.Second
	}
	r.pause(ctx, pause)
}

// holdRefused records a refusal that is a decision and not a step error (the kit gate holding a trip back, a town chore
// that bought nothing): one INFO line per change of need, words or wake facts, the standing refusal for explain, no pause
// and no failure count. Knob bot.typedOutcome 0: nothing (the legacy lines stay as they are).
func (r *Runner) holdRefused(ref *Refused) {
	if r.e.Tune.Bot.TypedOutcome == 0 {
		return
	}
	l := &r.loop
	facts := r.wakeFacts(r.latest())
	key := ref.Need + "|" + ref.Why
	if l.holds == nil {
		l.holds = map[string]string{}
	}
	r.noteRefused("hold:"+ref.Need, ref)
	if f, ok := l.holds[key]; ok && f == facts {
		return
	}
	if len(l.holds) > 32 {
		clear(l.holds)
	}
	l.holds[key] = facts
	r.e.Log.Info("step refused", "need", ref.Need, "wake", ref.Wake, "why", ref.Why, "hold", true)
}

// holdNeed builds the refusal of a hold from its need code.
func holdNeed(need, why string) *Refused {
	return &Refused{StepOutcome{Kind: OutRefusedKind, Need: need, Wake: needTable[need].Wake, Why: why}}
}

// choreHolds turns what a finished town visit refused (a chore it skipped, slots the purse could buy but the hero cannot
// wear yet) into typed holds: the need says what to wait for, the wake what ends it.
func (r *Runner) choreHolds(rep ChoreReport) {
	for _, sk := range rep.Skipped {
		r.holdRefused(refusalOf(sk))
	}
	if len(rep.Blocked) > 0 {
		r.holdRefused(holdNeed("too_weak", "gear the hero cannot wear yet: "+strings.Join(rep.Blocked, ", ")))
	}
}

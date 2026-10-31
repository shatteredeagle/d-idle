package bot

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"d2lod-bot/internal/goals"
)

// A bug in one action (a nil pointer in the town code, V: Soaky soak 2026-10-29, internal/town/trip.go:114) must not
// end the process and leave the game at the death screen. The step loop recovers, logs the stack, tells the dashboard
// and carries on through the failsafe ladder; too many panics in a window pause the bot itself (it resumes on its own).
const (
	PanicBudget  = 5         // panics tolerated per PanicWindow before the bot pauses itself
	PanicWindow  = time.Hour // sliding window of the budget
	panicBackoff = 10 * time.Second
)

// ErrPanicStorm is what Run returns when panics persist and pausing does not help (no goal to pause, already paused,
// or a second budget overrun within PanicWindow): the process should end non-zero so the supervisor restarts it with
// clean in-memory state.
var ErrPanicStorm = errors.New("bot: persistent panics")

// safeStep is step with a recover: a panic is handled by onPanic and the tick is spent.
func (r *Runner) safeStep(ctx context.Context) (acted bool) {
	defer func() {
		if v := recover(); v != nil {
			r.onPanic(ctx, v, debug.Stack())
			acted = false
		}
	}()
	return r.step(ctx)
}

// notePanic records a recovered panic (any goroutine): log with stack, error event, budget. It reports whether the
// budget is now exceeded.
func (r *Runner) notePanic(v any, stack []byte) (over bool) {
	r.e.Log.Error("panic recovered", "panic", fmt.Sprint(v), "stack", string(stack))
	now := r.e.Clock.Now()
	r.mu.Lock()
	keep := r.panicTimes[:0]
	for _, t := range r.panicTimes {
		if now.Sub(t) < PanicWindow {
			keep = append(keep, t)
		}
	}
	r.panicTimes = append(keep, now)
	r.panics++
	over = len(r.panicTimes) > PanicBudget
	r.mu.Unlock()
	r.emit(goals.Event{Type: goals.EvError, Message: fmt.Sprintf("Failsafe: internal error recovered (panic: %v)", v)})
	return over
}

// onPanic runs on the runner goroutine after a recovered step panic: reset what the dead action left behind, count the
// panic, then either pause (over budget) or take the ordinary failure path, which climbs the failsafe ladder.
func (r *Runner) onPanic(ctx context.Context, v any, stack []byte) {
	defer func() { // the handler must not take the loop down either
		if v2 := recover(); v2 != nil {
			r.e.Log.Error("panic in the panic handler", "panic", fmt.Sprint(v2), "stack", string(debug.Stack()))
			r.pause(ctx, panicBackoff)
		}
	}()
	over := r.notePanic(v, stack)
	r.mu.Lock()
	r.cancelAct = nil
	r.mu.Unlock()
	r.armWatchdog(false)
	r.loop.inLadder = false
	if over {
		r.mu.Lock()
		now := r.e.Clock.Now()
		escalate := r.goal == nil || r.selfStop || (!r.lastOverrun.IsZero() && now.Sub(r.lastOverrun) < PanicWindow)
		r.lastOverrun = now
		if escalate {
			r.crash = fmt.Errorf("%w: %d panics within %v (last: %v)", ErrPanicStorm, len(r.panicTimes), PanicWindow, v)
		}
		r.mu.Unlock()
		if escalate {
			return
		}
		r.selfPause(fmt.Sprintf("Failsafe: %d internal errors within %v (last: %v). Paused itself, retrying in a few minutes.", PanicBudget+1, PanicWindow, v))
		r.pause(ctx, panicBackoff) // a persistent bug must not spin at tick rate while paused
		return
	}
	r.fail(ctx, r.control(), "internal error", fmt.Errorf("panic: %v", v))
}

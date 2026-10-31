package bot

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/event"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/task"
)

// Run is the runner's goroutine: it blocks until ctx ends or a fatal error stops the bot (then it returns that error,
// after the Error event with Fatal set was emitted). Call Stop before cancelling ctx to finish in town.
func (r *Runner) Run(ctx context.Context) error {
	r.e.Log.Info("bot runner started", "profile", r.cfg.Profile)
	r.lastStep.Store(time.Now().UnixNano())
	for {
		r.mu.Lock()
		fatal := r.fatal
		if fatal == nil {
			fatal = r.crash
		}
		r.mu.Unlock()
		if fatal != nil {
			return fatal
		}
		if ctx.Err() != nil {
			return nil
		}
		if !r.safeStep(ctx) {
			r.pause(ctx, time.Duration(r.e.Tune.Bot.TickMs)*time.Millisecond)
		}
		r.lastStep.Store(time.Now().UnixNano())
	}
}

// pause sleeps d in slices so a cancelled ctx or a control change is noticed quickly.
func (r *Runner) pause(ctx context.Context, d time.Duration) {
	const slice = 100 * time.Millisecond
	for d > 0 && ctx.Err() == nil {
		s := min(d, slice)
		r.e.Clock.Sleep(s)
		d -= s
	}
}

// step runs one decision and at most one action. It reports whether an action ran (false: the caller sleeps a tick).
func (r *Runner) step(ctx context.Context) bool {
	r.applySettings()
	snap := r.poll()
	c := r.control()
	if g := r.gameEvent(); g != nil {
		r.gameLost(ctx, c, g.Event)
		return true
	}
	if c.gen != r.loop.gen {
		if c.goal != nil && r.betweenGoals(ctx, snap) {
			return true // the game was restarted: read everything again before the goal starts
		}
		r.resetForGoal(c)
	}
	if why := r.stuckCheck(r.e.Clock.Now()); why != "" && c.goal != nil {
		r.escalate(ctx, c, why, errStuck)
		return true
	}
	if snap == nil {
		return r.notInGame(ctx, c)
	}
	p := snap.Player
	r.obMu.Lock()
	pending := r.ob.deathPending
	r.ob.deathPending = false
	r.obMu.Unlock()
	if pending { // also when the respawn was faster than a tick
		r.openRecovery()
	}
	if isDead(p) {
		return r.deathWait(c)
	}
	if r.campaignStall(ctx, snap, c.goal != nil && !c.paused) {
		return true
	}
	if r.loop.rec != nil {
		return r.finishRecovery(ctx, snap, c)
	}
	if c.goal == nil {
		return r.noGoal(ctx, snap, c)
	}
	if r.errorWait(snap) {
		return false
	}
	if r.guardedExit(ctx, snap, c) {
		return true
	}
	if c.paused && r.levelerResume(snap) {
		c.paused = false
	}
	if c.paused && r.autoResumed() {
		c.paused = false
	}
	if c.paused {
		// Never fall through to the goal while paused: with a monster in reach the old code ran the goal (state
		// clearing) until the guard cancelled it, and the status flapped paused/clearing every tick (soak 2026-10-29).
		r.setState(StatePaused, "Paused")
		if r.safeToPause(snap) {
			r.hazard(ctx, snap)
		} else {
			r.pausedFight(ctx, snap)
		}
		return false
	}
	return r.runGoal(ctx, snap, c)
}

// betweenGoals asks the recycler (Deps.Recycle) whether the game should be restarted now. Only in town with nothing hostile
// near and no panel open; never mid-goal. It reports whether the game was restarted (or half-restarted).
func (r *Runner) betweenGoals(ctx context.Context, snap *d2.Snapshot) bool {
	if r.d.Recycle == nil || snap == nil || snap.Player == nil || !snap.Player.InTown() || isDead(snap.Player) ||
		r.hostileNear(snap, r.e.Tune.Bot.ClearRadius) > 0 || snap.UI.BlockingPanelOpen() {
		return false
	}
	if r.d.RecycleDue != nil {
		if due, _ := r.d.RecycleDue(); !due {
			return false
		}
	}
	rctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	r.setState(StateRecovering, "Restarting the game (recycle)")
	did, why, err := r.d.Recycle(rctx)
	if !did && err == nil {
		return false
	}
	r.obMu.Lock()
	r.ob.gone, r.ob.errSince = nil, time.Time{} // the old process exiting is the plan, not a crash
	r.obMu.Unlock()
	state := "recycle"
	if err == nil {
		r.noteRecycle()
	}
	if err != nil {
		state = "recycle_failed"
		r.e.Log.Error("game recycle failed", "reason", why, "err", err)
	} else {
		r.e.Log.Info("game recycled", "reason", why)
	}
	if r.bus != nil {
		msg := why
		if err != nil {
			msg += ": " + err.Error()
		}
		r.bus.Publish(event.Event{V: event.Version, Kind: event.KindLifecycle, Source: event.SrcLaunch,
			Data: event.Lifecycle{State: state, Reason: why, Message: msg}})
	}
	return true
}

func (r *Runner) resetForGoal(c control) {
	l := &r.loop
	l.gen = c.gen
	l.failures, l.stuckStage = 0, 0
	l.trip = nil
	l.stepArea = 0
	l.needBuff = true
	l.waitNote = false
	l.pendingErr = ""
	l.stopTries = 0
	l.healFails, l.reentries, l.lowLifeUntil = 0, nil, time.Time{}
}

// ---- settings and failures ------------------------------------------------------------------------

// settingKnobs maps user settings to knob keys; a key that no domain defines is skipped.
var settingKnobs = []struct {
	key string
	get func(goals.Settings) float64
}{
	{"combat.chickenLifePct", func(s goals.Settings) float64 { return float64(s.ChickenLifePct) }},
	{"combat.potionLifePct", func(s goals.Settings) float64 { return float64(s.PotionLifePct) }},
}

// applySettings runs between actions, when no behavior is reading the Tuning.
func (r *Runner) applySettings() {
	r.mu.Lock()
	if !r.setDirty {
		r.mu.Unlock()
		return
	}
	s := r.settings
	r.setDirty = false
	r.mu.Unlock()
	knobs := map[string]env.Knob{}
	for _, k := range env.Knobs() {
		knobs[k.Key] = k
	}
	for _, sk := range settingKnobs {
		if k, ok := knobs[sk.key]; ok {
			if err := SetKnob(r.e.Tune, k, sk.get(s)); err != nil {
				r.e.Log.Warn("setting not applied", "knob", sk.key, "err", err)
			}
		}
	}
}

func (r *Runner) fatalf(format string, a ...any) {
	err := fmt.Errorf(format, a...)
	r.mu.Lock()
	if r.fatal != nil {
		r.mu.Unlock()
		return
	}
	r.fatal = err
	r.setLocked(StateError, "Stopped: "+err.Error())
	r.message = err.Error()
	r.mu.Unlock()
	r.e.Log.Error("fatal", "err", err)
	r.emit(goals.Event{Type: goals.EvError, Message: err.Error(), Fatal: true})
}

// setError enters the Error state (non-fatal) for retry; it emits one Error event per distinct message.
func (r *Runner) setError(msg string, retry time.Duration) {
	now := r.e.Clock.Now()
	r.obMu.Lock()
	r.loop.errGames = r.ob.newGames
	r.obMu.Unlock()
	r.mu.Lock()
	r.errorUntil = now.Add(retry)
	r.message = msg
	r.setLocked(StateError, msg)
	r.mu.Unlock()
	if r.loop.pendingErr != msg {
		r.loop.pendingErr = msg
		r.e.Log.Error("bot error", "msg", msg, "retry", retry)
		r.emit(goals.Event{Type: goals.EvError, Message: msg})
	}
}

// errorWait holds the loop in the Error state until the retry time, a new goal or a fresh game.
func (r *Runner) errorWait(snap *d2.Snapshot) bool {
	r.mu.Lock()
	st, until := r.st, r.errorUntil
	r.mu.Unlock()
	if st != StateError {
		return false
	}
	r.obMu.Lock()
	games := r.ob.newGames
	r.obMu.Unlock()
	if r.e.Clock.Now().Before(until) && games == r.loop.errGames {
		r.hazard(context.Background(), snap)
		return true
	}
	r.mu.Lock()
	r.errorUntil, r.message = time.Time{}, ""
	r.setLocked(StateIdle, "Retrying")
	r.mu.Unlock()
	r.loop.pendingErr = ""
	r.loop.failures, r.loop.stuckStage = 0, 0
	return false
}

// pausedFight defends the character while the bot is paused with a monster in reach: safety first, then a bounded
// fight, with the displayed state left at paused.
func (r *Runner) pausedFight(ctx context.Context, snap *d2.Snapshot) {
	if a := r.d.Fighter.Safety(snap); a != SafetyNone {
		r.carryOut(ctx, a)
		return
	}
	fctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if _, err := r.d.Fighter.Clear(fctx, ClearOpts{Radius: r.e.Tune.Bot.ClearRadius}); err != nil && fctx.Err() == nil {
		r.e.Log.Warn("paused: fight failed", "err", err)
	}
}

// hazard lets the fighter drink or leave while the bot holds still (paused, error wait).
func (r *Runner) hazard(ctx context.Context, snap *d2.Snapshot) {
	if snap == nil || snap.Player == nil || snap.Player.InTown() {
		return
	}
	if a := r.d.Fighter.Safety(snap); a != SafetyNone {
		r.carryOut(ctx, a)
	}
}

func (r *Runner) carryOut(ctx context.Context, a SafetyAction) error {
	sctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	err := r.d.Fighter.Act(sctx, a)
	if err != nil {
		r.e.Log.Warn("safety action failed", "action", a, "err", err)
	}
	return err
}

// ---- actions ---------------------------------------------------------------------------------------

func watchedState(st State) bool {
	return st == StateTraveling || st == StateClearing || st == StateResupplying
}

// runAction runs f under a context that the guard, Pause, Stop and Start can cancel with a cause. It returns f's error
// and the cancel cause (nil when nothing cancelled the action).
func (r *Runner) runAction(parent context.Context, gen uint64, st State, activity string, timeout time.Duration, f func(context.Context) error) (error, error) {
	actx, cancel := context.WithCancelCause(parent)
	defer cancel(nil)
	r.mu.Lock()
	if r.gen != gen {
		r.mu.Unlock()
		return nil, errNewGoal
	}
	r.cancelAct = cancel
	if r.st != st {
		r.e.Log.Info("state", "from", r.st, "to", st, "activity", activity)
	}
	r.setLocked(st, activity)
	r.mu.Unlock()
	r.armWatchdog(watchedState(st))
	if st == StateTraveling {
		r.setWalking(true)
		defer r.setWalking(false)
	}
	tctx := actx
	if timeout > 0 {
		var tc context.CancelFunc
		tctx, tc = context.WithTimeoutCause(actx, timeout, errStepTimeout)
		defer tc()
	}
	done, gdone := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(gdone)
		defer func() {
			if v := recover(); v != nil { // fail safe: without the guard nothing ends the action on death or a lost game
				r.notePanic(v, debug.Stack())
				cancel(errGuardPanic)
			}
		}()
		r.guard(tctx, cancel, done, st)
	}()
	err := f(tctx)
	close(done)
	<-gdone
	r.mu.Lock()
	r.cancelAct = nil
	r.mu.Unlock()
	r.armWatchdog(false)
	return err, context.Cause(tctx)
}

// do runs an action and applies the shared reaction to how it ended. It reports success.
func (r *Runner) do(ctx context.Context, c control, st State, activity string, timeout time.Duration, what string, f func(context.Context) error) bool {
	err, cause := r.runAction(ctx, c.gen, st, activity, timeout, f)
	l := &r.loop
	if err == nil {
		l.failures = 0
		return true
	}
	if ctx.Err() != nil {
		return false
	}
	var gone gameGone
	var si safetyInterrupt
	switch {
	case errors.As(cause, &gone):
		r.gameLost(ctx, c, gone.Event)
	case errors.As(cause, &si):
		r.handleSafety(ctx, si.Action)
	case errors.Is(err, ErrPortalOpened):
		area := 0
		if s := r.latest(); s != nil {
			area = s.Player.Area
		}
		l.trip = &trip{plan: r.fullChores("emergency town portal"), returnTo: fieldArea0(area), emergency: true}
		r.setLastTrip(r.e.Clock.Now())
	case errors.Is(err, ErrLeftGame):
		// the menus are handled by the next step
	case errors.Is(cause, errNewGoal), errors.Is(cause, errStop), errors.Is(cause, errPause), errors.Is(cause, errTripDue), errors.Is(cause, errPoints), errors.Is(cause, errReplan), errors.Is(cause, errDied), errors.Is(cause, errLeftGame):
		// the next step re-reads control and the snapshot
	case errors.Is(cause, errStuck):
		if !l.inLadder {
			r.escalate(ctx, c, what, errStuck)
		}
	case errors.Is(err, errAreaGated):
		// A deliberate refusal, not a fault: no failure count, no ladder, no restart. The planner skips the target.
		l.failures = 0
		r.gatedWait(ctx, err)
	default:
		if s := r.latest(); s != nil && isDead(s.Player) {
			break // a walk that stopped because the character died (nav.ErrDead): the next step recovers
		}
		if errors.Is(cause, errStepTimeout) {
			err = fmt.Errorf("%s timed out: %w", what, err)
		}
		r.fail(ctx, c, what, err)
	}
	return false
}

// gatedWait is the answer to a gate refusal: show why, and wait a moment so the loop does not spin.
func (r *Runner) gatedWait(ctx context.Context, err error) {
	msg := err.Error()
	if strings.Contains(msg, noAreaMsg) {
		msg = "Waiting: no eligible farm area, retrying shortly"
	}
	if w := r.banWait(); w != "" {
		msg = w
		if r.idleBreak(false) {
			return
		}
	}
	r.e.Log.Info("area gate refused the walk", "err", err)
	r.skipTarget(whyGate) // the planner must not insist on a target the gate refuses (the gated safe area is exempt)
	r.setState(StateInTown, msg)
	r.pause(ctx, 5*time.Second)
}

func fieldArea0(a int) int { return a }

func (r *Runner) fail(ctx context.Context, c control, what string, err error) {
	l := &r.loop
	l.failures++
	r.e.Log.Warn("action failed", "what", what, "err", err, "consecutive", l.failures)
	if l.failures >= r.cfg.MaxStepFailures {
		r.escalate(ctx, c, what, err)
		return
	}
	r.pause(ctx, min(time.Duration(l.failures)*500*time.Millisecond, 5*time.Second))
}

// gameLost handles a game process that vanished, crashed or hung without the bot restarting it (for example the
// game exiting by itself after a chicken Save and Exit, V: soak livebot/20261029-035908-d2bot 04:23:14). In managed
// mode (Deps.Restart set) it restarts and logs in again, at most MaxFailsafeRestarts times without real progress in
// between; unmanaged, or when the restart fails or the budget is spent, it is fatal as before.
func (r *Runner) gameLost(ctx context.Context, c control, ev launch.Event) {
	l := &r.loop
	r.obMu.Lock()
	prog := r.ob.real
	r.obMu.Unlock()
	if prog-l.goneProgress >= realResetMin {
		l.goneProgress, l.goneRestarts = prog, 0
	}
	if r.d.Restart == nil || l.goneRestarts >= r.cfg.MaxFailsafeRestarts {
		r.fatalf("the game is gone (%s)", ev)
		return
	}
	if r.restartCapHit() {
		r.fatalf("the game is gone (%s) and %d unplanned restarts happened within %v", ev, maxRestartsPerHour, time.Hour)
		return
	}
	l.goneRestarts++
	r.logXPAtRisk("game lost: " + ev.String())
	r.e.Log.Warn("game lost, restarting it", "event", ev.String(), "attempt", l.goneRestarts)
	l.trip = nil
	r.setState(StateRecovering, "Game lost: restarting it")
	var err error
	for try := 0; try < 2; try++ {
		if try > 0 {
			r.e.Log.Warn("restart failed, retrying once", "err", err)
			r.pause(ctx, 20*time.Second)
			r.Beat() // two attempts can outlast the d2bot watchdog (S2); each finished attempt is observable progress
		}
		r.expectGameExit(true)
		rctx, cancel := context.WithTimeout(ctx, r.restartTimeout())
		err = r.d.Restart(rctx)
		cancel()
		r.Beat()
		r.expectGameExit(false)
		if err == nil || ctx.Err() != nil {
			break
		}
	}
	if err != nil && ctx.Err() == nil {
		r.fatalf("the game is gone (%s) and the restart failed twice: %v", ev, err)
		return
	}
	if err == nil {
		r.noteReentry()
	}
}

// Restart budgets: real progress must be sustained (realResetMin XP/kill/level events) to refill them, and at most
// maxRestartsPerHour unplanned restarts happen per hour whatever the progress, so a crash loop cannot spin all night.
const (
	realResetMin       = 3
	maxRestartsPerHour = 8
)

// restartCapHit records an unplanned restart and reports whether the hourly cap is exceeded.
func (r *Runner) restartCapHit() bool {
	l := &r.loop
	now := r.e.Clock.Now()
	keep := l.restartTimes[:0]
	for _, t := range l.restartTimes {
		if now.Sub(t) < time.Hour {
			keep = append(keep, t)
		}
	}
	l.restartTimes = append(keep, now)
	return len(l.restartTimes) > maxRestartsPerHour
}

// restartTimeout covers Leave, Shutdown, helper start, Launch and Login of a recycle (about 200 s, more on a loaded box).
func (r *Runner) restartTimeout() time.Duration { return 3 * sec(r.cfg.TownTimeoutSec) }

// escalate is the error ladder (prior-art 07-safety section 7): close panels, then reset in town, then give up
// visibly (Error state with a retry timer), never loop silently.
func (r *Runner) escalate(ctx context.Context, c control, what string, err error) {
	l := &r.loop
	if s := r.latest(); s != nil && s.Player != nil && s.Player.InTown() && r.State() == StateInTown && r.idleBreak(true) {
		// the stall is a "Waiting: ... banned" in town, not a fault: farm the least bad area instead of climbing the ladder
		r.rearm()
		return
	}
	was := l.inLadder
	l.inLadder = true
	defer func() { l.inLadder = was }()
	r.obMu.Lock()
	prog := r.ob.real
	r.obMu.Unlock()
	if prog-l.ladderProgress >= realResetMin {
		l.stuckStage, l.restarts = 0, 0
		l.ladderProgress = prog // real progress since the last rung: the ladder starts over
	}
	l.failures = 0
	l.trip = nil
	// Steps that cannot apply (no cycler, no restart hook, already in town) are skipped without waiting.
	for {
		l.stuckStage++
		r.hmu.Lock()
		r.ladder = l.stuckStage
		r.hmu.Unlock()
		r.e.Log.Warn("failsafe ladder", "step", l.stuckStage, "what", what, "err", err)
		snap := r.latest()
		dead := snap == nil || isDead(snap.Player)
		inTown := snap != nil && snap.Player.InTown()
		applied := false
		switch l.stuckStage {
		case 1: // skip the target, close panels / Esc
			applied = true
			r.skipTarget(whyNavStuck)
			if dead {
				for i := 0; i < 3; i++ {
					r.e.In.Key("esc")
					r.e.Clock.Sleep(300 * time.Millisecond)
				}
			} else if cerr := r.e.In.CloseAll(); cerr != nil {
				r.e.Log.Warn("close panels failed", "err", cerr)
			}
		case 2: // reset to town
			if !dead && !inTown && r.d.Town != nil {
				applied = true
				stage := l.stuckStage
				r.do(ctx, c, StateResupplying, "Failsafe: resetting in town", sec(r.cfg.TownTimeoutSec), "reset", r.d.Town.ToTown)
				l.stuckStage = stage
			}
		case 3: // leave the game and re-enter through the menus
			if r.d.Cycler != nil {
				applied = true
				stage := l.stuckStage
				r.do(ctx, c, StateRecovering, "Failsafe: leaving and re-entering the game", sec(r.cfg.TownTimeoutSec), "new game", func(ctx context.Context) error {
					return r.newGameRun(func(ctx context.Context) error { return r.d.Cycler.NewGame(ctx, c.spec.difficulty) })(ctx)
				})
				l.stuckStage = stage
			}
		case 4: // restart the game process and log in again (managed mode); the goal carries on afterwards
			if r.d.Restart != nil && l.restarts < r.cfg.MaxFailsafeRestarts {
				applied = true
				l.restarts++
				stage := l.stuckStage
				// The old process exiting is the plan: without this the guard saw the exit and ended the run with
				// "the game is gone" (V: soaks livebot/20261029-032705-d2bot 03:41:29 and -021327, docs/stallfatal.md).
				if r.restartCapHit() {
					l.stuckStage = 4 // the next rung is the visible pause
					break
				}
				r.logXPAtRisk("failsafe rung 4")
				r.expectGameExit(true)
				ok := r.do(ctx, c, StateRecovering, "Failsafe: restarting the game", r.restartTimeout(), "restart", r.oogRun(r.d.Restart))
				r.expectGameExit(false)
				l.stuckStage = stage
				if ok && l.restarts < r.cfg.MaxFailsafeRestarts {
					l.stuckStage = 0 // a fresh game: the ladder starts over, bounded by MaxFailsafeRestarts
					l.trip = nil
				}
			}
		default: // pause with a visible error
			l.stuckStage = 0
			r.selfPause(fmt.Sprintf("Failsafe: %s (%v). Esc, town, re-entry and restart did not help; paused itself, retrying in %v. Look at the game, or Resume now.", what, err, failsafeRetry))
			r.autoResumeIn(failsafeRetry)
			r.rearm()
			r.obMu.Lock()
			l.ladderProgress = r.ob.real
			r.obMu.Unlock()
			return
		}
		if applied {
			r.rearm()
			r.obMu.Lock()
			l.ladderProgress = r.ob.real
			r.obMu.Unlock()
			return
		}
	}
}

// skipTarget is the cheapest rung: a leveling or campaign goal that stands still on its target area avoids that area for a
// while and re-plans (the planner then picks another one). Farm goals name their area, so nothing is skipped for them.
func (r *Runner) skipTarget(why string) {
	c := r.control()
	if c.spec.kind != goals.ReachLevel && c.spec.camp == nil {
		return
	}
	l := &r.loop
	r.obMu.Lock()
	area := r.ob.targetArea
	r.obMu.Unlock()
	if area == 0 {
		area = l.stepArea
	}
	if area == 0 || (r.d.Plan.IsTown(area)) {
		return
	}
	if s := r.latest(); s != nil && r.gearVerdict(s).Gated && area == r.safeFarmArea(s) {
		// A gated character has nowhere else to go: avoiding the safe farm area left it with no eligible area at all and
		// the planning block climbed the ladder to game restarts (LIVE G1 attempt 4).
		l.stepArea = 0
		return
	}
	r.skipArea(area, r.navSkip(), why)
	l.stepArea = 0
	l.futile = false
	delete(l.cleared, area)
	r.e.Log.Warn("failsafe: skipped the stalled target area", "area", area, "name", r.areaName(area), "reason", why)
}

// selfPause pauses the runner for a failsafe and says why (Status paused, non-fatal Error event). Resume continues.
func (r *Runner) selfPause(msg string) {
	r.mu.Lock()
	if r.goal == nil || r.selfStop {
		r.mu.Unlock()
		return
	}
	r.paused, r.selfStop, r.message = true, true, msg
	r.autoResumeAt = r.e.Clock.Now().Add(r.selfPauseDelay())
	r.mu.Unlock()
	r.e.Log.Error("failsafe: paused", "msg", msg)
	r.emit(goals.Event{Type: goals.EvError, Message: msg})
}

func (r *Runner) isSelfPaused() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.selfStop
}

// selfPauseDelay is how long a self pause lasts before the bot retries by itself: every self pause on the unattended
// path ends on its own, with a backoff (5, 10, 20, 40, 60 minutes) while pauses keep coming within two hours. Caller
// holds r.mu.
func (r *Runner) selfPauseDelay() time.Duration {
	now := r.e.Clock.Now()
	if now.Sub(r.lastSelfPause) > 2*time.Hour {
		r.pauseStreak = 0
	}
	r.lastSelfPause = now
	r.pauseStreak++
	return min(5*time.Minute<<min(r.pauseStreak-1, 4), time.Hour)
}

// failsafeRetry is how long the end of the failsafe ladder pauses before the bot tries again by itself: an unattended
// soak must not idle forever on a human (live 2026-10-29, Soaky).
const failsafeRetry = 5 * time.Minute

// autoResumeIn makes the current self pause end by itself after d (no effect when not self paused).
func (r *Runner) autoResumeIn(d time.Duration) {
	r.mu.Lock()
	if r.selfStop {
		r.autoResumeAt = r.e.Clock.Now().Add(d)
	}
	r.mu.Unlock()
}

// autoResumed ends a timed self pause once its time is up.
func (r *Runner) autoResumed() bool {
	r.mu.Lock()
	if !r.selfStop || r.autoResumeAt.IsZero() || r.e.Clock.Now().Before(r.autoResumeAt) {
		r.mu.Unlock()
		return false
	}
	r.autoResumeAt = time.Time{}
	r.selfStop, r.paused, r.message = false, false, ""
	r.mu.Unlock()
	r.e.Log.Info("self pause ended by timeout; trying again")
	return true
}

// LevelerPause is leveler.Options.Pause: the leveler hit something it must not guess about (a skill read-back that does not
// match the plan, no build for the class). The bot pauses itself with the reason visible in Status and as an error event, and
// resumes by itself once no stat or skill points are pending any more (the plan is satisfied, for example after the
// operator spent them or the character was changed); Resume by hand works as always.
func (r *Runner) LevelerPause(reason string) {
	r.mu.Lock()
	had := r.selfStop
	r.mu.Unlock()
	r.selfPause("Leveler: " + reason)
	r.mu.Lock()
	if !had && r.selfStop {
		r.levelerPaused = true
	}
	r.mu.Unlock()
}

// levelerResume ends a leveler pause once nothing is left for the leveler to do.
func (r *Runner) levelerResume(snap *d2.Snapshot) bool {
	r.mu.Lock()
	on := r.levelerPaused && r.selfStop
	if !r.selfStop {
		r.levelerPaused = false
	}
	r.mu.Unlock()
	if !on || pointsPending(snap) {
		return false
	}
	r.mu.Lock()
	r.levelerPaused, r.selfStop, r.paused, r.message = false, false, false, ""
	r.mu.Unlock()
	r.e.Log.Info("leveler pause ended: no points pending")
	return true
}

// HealthReport is the bot's self report for GET /api/v1/bot/health (the out-of-process sentinel reads it).
type HealthReport struct {
	State               State   `json:"state"`
	Status              string  `json:"status"` // goals.RunnerState
	Activity            string  `json:"activity"`
	SinceSeconds        float64 `json:"sinceSeconds"`        // time in the current state
	LastProgressSeconds float64 `json:"lastProgressSeconds"` // since the last progress (xp, position, area, gold, kills, items)
	ProgressCounter     uint64  `json:"progressCounter"`     // monotonic: a stalled value means a stalled bot
	StuckReason         string  `json:"stuckReason"`
	LastError           string  `json:"lastError"`
	LadderStep          int     `json:"ladderStep"` // failsafe ladder position, 0 = none
	Paused              bool    `json:"paused"`
	SelfPaused          bool    `json:"selfPaused"`
	Panics              int     `json:"panics"` // recovered panics since start
	Goal                string  `json:"goal,omitempty"`
	Fatal               string  `json:"fatal,omitempty"`
}

// Health returns the self report. Safe from any goroutine.
func (r *Runner) Health() HealthReport {
	now := r.e.Clock.Now()
	r.mu.Lock()
	h := HealthReport{State: r.st, Activity: r.activity, SinceSeconds: now.Sub(r.stateSince).Seconds(), Paused: r.paused, SelfPaused: r.selfStop, Panics: r.panics}
	if r.goal != nil {
		h.Goal = r.goal.ID
	}
	if r.fatal != nil {
		h.Fatal = r.fatal.Error()
	}
	r.mu.Unlock()
	h.Status = string(r.Status().State)
	r.obMu.Lock()
	h.LastProgressSeconds = now.Sub(r.ob.lastProgress).Seconds()
	h.ProgressCounter, h.StuckReason = r.ob.progress, r.ob.stuckReason
	r.obMu.Unlock()
	r.hmu.Lock()
	h.LastError, h.LadderStep = r.lastError, r.ladder
	r.hmu.Unlock()
	return h
}

// latest returns the newest in-game snapshot, or nil.
func (r *Runner) latest() *d2.Snapshot {
	r.obMu.Lock()
	defer r.obMu.Unlock()
	if s := r.ob.lastSnap; s != nil && s.State == d2.StateInGame && s.Player != nil {
		return s
	}
	return nil
}

// handleSafety carries out the fighter's decision after the guard interrupted an action.
func (r *Runner) handleSafety(ctx context.Context, a SafetyAction) {
	var area int
	if s := r.latest(); s != nil {
		area = s.Player.Area
	}
	if err := r.carryOut(ctx, a); err != nil {
		r.loop.failures++
		return
	}
	switch a {
	case SafetyTownPortal:
		r.loop.trip = &trip{plan: r.fullChores("emergency town portal"), returnTo: area, stage: 0, emergency: true}
	case SafetyCrowdAbandon:
		// the walk is abandoned: keep the planner out of that area for a while instead of walking back into the crowd
		r.skipArea(area, r.navSkip()*4, whyRoute)
		r.loop.stepArea = 0
		r.e.Log.Warn("crowd: walk abandoned, retreated and opened a portal", "area", area, "name", r.areaName(area))
		r.loop.trip = &trip{plan: r.fullChores("crowd abandon"), returnTo: 0, stage: 0, emergency: true}
	}
}

func (r *Runner) fullChores(reason string) ChorePlan {
	return ChorePlan{Reason: reason, Heal: true, Potions: true, Repair: true, Identify: true, Sell: true, Stash: true, GoldFloor: r.e.Tune.Bot.GoldFloor,
		Antidotes: r.antidotesWanted()}
}

// antidotesWanted is the antidote stock a town visit buys: the Andariel kit once the character is within two levels of
// her floor in Normal and has not killed her (the gate itself re-checks the stock, task.BossGate).
func (r *Runner) antidotesWanted() int {
	s := r.latest()
	if s == nil || s.Player == nil || s.Player.Difficulty != d2.Normal || s.Player.Level < r.e.Tune.Boss.AndarielMinLevel-2 || r.killedBoss(task.BossAndy) {
		return 0
	}
	return 2
}

// ---- guard -------------------------------------------------------------------------------------

// guard watches while an action runs, every Tuning.Bot.WatchMs: it keeps the observer fed (kills, drops, level ups
// are emitted live), and cancels the action when the character died, the game is gone, a pause is due, the fighter
// wants an emergency action, or the watchdog sees no progress. It never acts itself, so input stays single-threaded.
func (r *Runner) guard(ctx context.Context, cancel context.CancelCauseFunc, done <-chan struct{}, st State) {
	every := time.Duration(r.e.Tune.Bot.WatchMs) * time.Millisecond
	// A lockstep clock (d2d) runs the guard on the acting goroutine, after the step that crossed WatchMs of virtual
	// time: a second goroutine sleeping on that clock would step the game itself and interleave with the action by
	// scheduling, so two same-seed runs diverge at once (docs/d2d-determinism.md). Game.exe keeps the loop below.
	if w, ok := r.e.Clock.(interface {
		Watch(time.Duration, func()) func()
	}); ok {
		var fired atomic.Bool
		if stop := w.Watch(every, func() {
			if fired.Load() || ctx.Err() != nil {
				return
			}
			if r.guardOnce(ctx, cancel, st) {
				fired.Store(true)
			}
		}); stop != nil {
			defer stop()
			select {
			case <-done:
			case <-ctx.Done():
			}
			return
		}
	}
	for {
		r.e.Clock.Sleep(every)
		select {
		case <-done:
			return
		case <-ctx.Done():
			return
		default:
		}
		if r.guardOnce(ctx, cancel, st) {
			return
		}
	}
}

// guardOnce is one look of the guard; true when it cancelled the action (the guard is then finished).
func (r *Runner) guardOnce(ctx context.Context, cancel context.CancelCauseFunc, st State) bool {
	{
		snap := r.poll()
		now := r.e.Clock.Now()
		if g := r.gameEvent(); g != nil {
			cancel(*g)
			return true
		}
		r.obMu.Lock()
		menu, since, oog := r.ob.state == d2.StateMenu, r.ob.menuSince, r.ob.oog > 0
		r.obMu.Unlock()
		if menu && !since.IsZero() && !oog && now.Sub(since) >= sec(r.cfg.MenuGraceSec) {
			cancel(errLeftGame)
			return true
		}
		if snap == nil {
			return false
		}
		if isDead(snap.Player) {
			cancel(errDied)
			return true
		}
		if c := r.control(); c.paused && (r.safeToPause(snap) || r.isSelfPaused()) {
			cancel(errPause)
			return true
		}
		if !snap.Player.InTown() {
			a := r.d.Fighter.Safety(snap)
			if (a == SafetyTownPortal || a == SafetyCrowdAbandon) && r.noTP.Load() {
				a = SafetyNone // Ancients: a portal would reset the fight (Constraints.NoTownPortal)
			}
			if a != SafetyNone && (st != StateClearing || a == SafetyTownPortal || a == SafetyChicken || a == SafetyCrowdAbandon) && r.safetyDue(a, now) {
				if rt := r.e.Tune.Bot.ReactionMs; rt > 0 {
					r.e.Clock.Sleep(time.Duration(rt) * time.Millisecond)
				}
				cancel(safetyInterrupt{a})
				return true
			}
		}
		if why := r.replanCause(snap, st); why != "" {
			r.e.Log.Info("action cancelled: re-plan", "why", why, "area", snap.Player.Area, "level", snap.Player.Level)
			cancel(errReplan)
			return true
		}
		if (st == StateClearing || st == StateTraveling) && r.pointsDue(snap, now) {
			cancel(errPoints) // a calm moment with points to spend: the loop spends them, then carries on
			return true
		}
		if (st == StateClearing || st == StateTraveling) && !snap.Player.InTown() && r.tripDue(snap) {
			cancel(errTripDue)
			return true
		}
		if r.stuckCheck(now) != "" {
			cancel(errStuck)
			return true
		}
	}
	return false
}

// replanCause says why the guard should cancel the running action to re-plan ("" = no reason). A boss run owns the step
// until it ends or its own deadline: the area and level checks are off for it (Soaky soak 2026-10-29, V: 181 boss runs
// of about 1 s, cancelled by the stale leveling target area every guard tick).
func (r *Runner) replanCause(snap *d2.Snapshot, st State) string {
	if r.bossRun.Load() {
		return ""
	}
	if st == StateClearing {
		r.obMu.Lock()
		ta := r.ob.targetArea
		r.obMu.Unlock()
		if ta != 0 && snap.Player.Area != ta && !snap.Player.InTown() {
			r.noteCrossing(ta)
			return fmt.Sprintf("exploring crossed an area border (target area %d)", ta) // walk back
		}
	}
	if c := r.control(); c.spec.kind == goals.ReachLevel && st == StateClearing {
		r.obMu.Lock()
		stale := r.ob.stepLevel != 0 && r.ob.stepLevel != snap.Player.Level
		r.obMu.Unlock()
		if stale {
			return "character level changed"
		}
	}
	return ""
}

// safetyDue rate-limits repeats of the same drink so a potion cooldown does not turn into an interrupt storm.
func (r *Runner) safetyDue(a SafetyAction, now time.Time) bool {
	const gap = 800 * time.Millisecond
	r.obMu.Lock()
	defer r.obMu.Unlock()
	if a == r.ob.lastSafety && now.Sub(r.ob.lastSafetyAt) < gap && a != SafetyTownPortal && a != SafetyChicken {
		return false
	}
	r.ob.lastSafety, r.ob.lastSafetyAt = a, now
	return true
}

// safeToPause: in town, or nothing hostile within the clear radius.
func (r *Runner) safeToPause(s *d2.Snapshot) bool {
	return s.Player.InTown() || r.hostileNear(s, r.e.Tune.Bot.ClearRadius) == 0
}

// ---- not in game, no goal, death ------------------------------------------------------------------

func (r *Runner) notInGame(ctx context.Context, c control) bool {
	r.obMu.Lock()
	menu, since := r.ob.state == d2.StateMenu, r.ob.menuSince
	r.obMu.Unlock()
	if !menu || since.IsZero() || r.e.Clock.Now().Sub(since) < sec(r.cfg.MenuGraceSec) || c.goal == nil {
		return false
	}
	if r.d.Cycler == nil {
		r.setError("The game is at the menus and no out-of-game flow is wired; enter a game by hand", 5*time.Second)
		return false
	}
	ok := r.do(ctx, c, StateRecovering, "Re-entering the game", sec(r.cfg.TownTimeoutSec), "enter game", r.oogRun(func(ctx context.Context) error {
		return r.d.Cycler.EnterGame(ctx, c.spec.difficulty)
	}))
	if ok {
		r.noteReentry()
	}
	return ok
}

func (r *Runner) noGoal(ctx context.Context, snap *d2.Snapshot, c control) bool {
	if !c.stopping {
		if r.State() != StateIdle {
			r.setState(StateIdle, "Idle")
		}
		if r.loop.lastRecycleCheck.IsZero() || r.e.Clock.Now().Sub(r.loop.lastRecycleCheck) > time.Minute {
			r.loop.lastRecycleCheck = r.e.Clock.Now()
			return r.betweenGoals(ctx, snap)
		}
		return false
	}
	l := &r.loop
	if !snap.Player.InTown() && l.stopTries < 2 {
		l.stopTries++
		return r.do(ctx, c, StateResupplying, "Returning to town", sec(r.cfg.StopTimeoutSec), "stop: go to town", r.d.Town.ToTown)
	}
	r.mu.Lock()
	if r.gen == c.gen && r.goal == nil {
		r.stopping = false
		r.setLocked(StateIdle, "Stopped")
	}
	r.mu.Unlock()
	return false
}

func equipCount(s *d2.Snapshot) int {
	if s == nil || s.Inventory == nil {
		return 0
	}
	return len(s.Inventory.Container(d2.ContEquipment))
}

// retrieveCorpse is the default retrieval (VERIFIED LIVE, docs/combat.md "Death"): walking within combat.CorpseTouch
// of the body returns the gear, and the dropped gold lies as a ground item next to it; walking onto it takes it back.
// No clicks are needed. A missing gold pile is not an error (nothing was carried, or it is already picked up).
func (r *Runner) retrieveCorpse(ctx context.Context, area int, at input.Point) error {
	if r.d.Corpse != nil {
		return r.d.Corpse.Retrieve(ctx, area, at)
	}
	if err := r.gotoDeathArea(ctx, area); err != nil {
		return err
	}
	if err := r.d.Mover.GoTo(ctx, at); err != nil {
		return fmt.Errorf("walk to corpse: %w", err)
	}
	s := r.poll()
	if s == nil {
		return nil
	}
	if pile, ok := combat.GoldPile(s.Ground, at, r.isGoldClass); ok {
		if err := r.d.Mover.GoTo(ctx, pile); err != nil {
			return fmt.Errorf("walk to gold: %w", err)
		}
	}
	return nil
}

func (r *Runner) isGoldClass(class int) bool {
	if r.d.Catalog == nil {
		return false
	}
	m, ok := r.d.Catalog.Item(class)
	return ok && m.GoalType == "gold"
}

// gotoDeathArea brings the character from town (or wherever it is) into area.
func (r *Runner) gotoDeathArea(ctx context.Context, area int) error {
	// Neither the portal nor the waypoint walk may end in a guarded area the character is not allowed in (G1 attempt 17:
	// the waypoint-and-walk corpse run went to Tower Cellar 5).
	if s := r.latest(); s != nil {
		if why := r.areaGuardWhy(s, area); why != "" {
			return fmt.Errorf("%w: %s (%d): route guard (%s)", errAreaGated, r.areaName(area), area, why)
		}
	}
	if r.preferWaypointRun(area) {
		r.e.Log.Info("corpse: waypoint and walk instead of the town portal", "area", area)
		var terr error
		for try := 0; try < 2; try++ {
			if terr = r.travelTo(ctx, area); terr == nil || ctx.Err() != nil {
				break
			}
		}
		if terr != nil {
			return fmt.Errorf("corpse run: %w", terr)
		}
		return nil
	}
	if err := r.d.Town.BackToField(ctx, area); errors.Is(err, ErrPortalWrongLevel) {
		// The town portal leads to where the last scroll was read, not to the corpse, and BackToField only knows
		// single exits: walk the rest like the outbound trip did (waypoints, known exits, exploring), twice, since
		// travelTo marks a dead edge and re-routes (LIVE G1 attempt 12: "no exit to that level: level 26 from 3").
		// The landing level is not the corpse area: the walk must not cross a banned or guarded area (the 02:30 run landed
		// in Monastery Gate and walked straight back into the crowd that had just killed it).
		if s := r.latest(); s != nil && r.areaAllowed(s, area) != "" {
			why := r.areaAllowed(s, area)
			return fmt.Errorf("%w: %s (%d): %s", errAreaGated, r.areaName(area), area, why)
		}
		r.e.Log.Info("back to field failed, travelling to the corpse area", "area", area, "err", err)
		var terr error
		for try := 0; try < 2; try++ {
			if terr = r.travelTo(ctx, area); terr == nil || ctx.Err() != nil {
				break
			}
		}
		if terr != nil {
			return fmt.Errorf("back to field: %w; travel: %v", err, terr)
		}
	} else if err != nil {
		return fmt.Errorf("back to field: %w", err)
	}
	return nil
}

// setWalking tells a fighter that supports it whether a travel or quest walk is running (the crowd-abandon rule).
func (r *Runner) setWalking(on bool) {
	if w, ok := r.d.Fighter.(interface{ SetWalking(bool) }); ok {
		w.SetWalking(on)
	}
}

// preferWaypointRun: from town, the town portal leads to the area where the last scroll was read (V live, G1 attempt
// 12: "no way back to level 26 from 3"), so when that is not the corpse area and an activated waypoint shortens the
// way, waypoint plus walk beats portal plus a long walk. The route guard still vets every area (travelTo).
func (r *Runner) preferWaypointRun(area int) bool {
	s := r.latest()
	if s == nil || s.Player == nil || !s.Player.InTown() || !r.e.Tune.Nav.UseWaypoints || !r.d.Plan.Waypoints[s.Player.Area] {
		return false
	}
	r.obMu.Lock()
	from := r.ob.portalFrom
	r.obMu.Unlock()
	if from == area {
		return false
	}
	hops, err := r.d.Mover.Route(s.Player.Area, area)
	if err != nil || len(hops) == 0 {
		return false
	}
	path := append([]int{s.Player.Area}, hops...)
	active, known := r.activeWaypoints(s)
	for i := len(path) - 1; i >= max(2, int(r.e.Tune.Nav.WpMinDistance)); i-- {
		if w := path[i]; r.d.Plan.Waypoints[w] && r.wpTripWorthTrying(s, w, active, known) {
			return true
		}
	}
	return false
}

// Border-loop breaker: crossings of an area border that the walk did not want, counted over a minute. d2d puts the
// character inside a cave mouth's trigger area on every return, so a mouth the walk cannot leave sideways turned into a
// shuttle between two levels at 10 Hz for hours (D2DLOOP r24, V live: Cold Plains and Cave Level 1, 30 % of an 8 h run).
// After borderLoopCrossings in borderLoopWindow the target area is skipped for borderLoopSkip so the planner picks another.
const (
	borderLoopCrossings = 10
	borderLoopWindow    = time.Minute
	borderLoopSkip      = 3 * time.Minute
)

func (r *Runner) noteCrossing(target int) {
	now := r.e.Clock.Now()
	r.obMu.Lock()
	keep := r.crossLog[:0]
	for _, t := range r.crossLog {
		if now.Sub(t) < borderLoopWindow {
			keep = append(keep, t)
		}
	}
	r.crossLog = append(keep, now)
	loop := len(r.crossLog) >= borderLoopCrossings
	if loop {
		r.crossLog = nil
		r.skipAreaLocked(target, borderLoopSkip, "border-loop")
	}
	r.obMu.Unlock()
}

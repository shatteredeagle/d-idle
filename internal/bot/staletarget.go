package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/task"
)

// Stale target, border-loop bans and blocked journey steps (docs/staletarget.md, V from the hero14 log 2026-11-14).

// whyBorderLoop is the skip reason of the border-loop and bounce breakers. It is a fault of the bot's own walking, not a danger.
const whyBorderLoop = "border-loop"

// staleOrCrossing decides what an area that is not the clearing target means. A character that arrived from a town
// (portal, waypoint or the walk out) did not cross a border: the target it carries is a leftover of an earlier run, so
// it is dropped and nothing is cancelled. Anything else is a real crossing: one stamp per arrival counts for the
// border-loop breaker, and the guard re-plans.
func (r *Runner) staleOrCrossing(area, ta int) string {
	r.obMu.Lock()
	jump := r.ob.fromTown
	fresh := r.ob.crossArea != area
	if jump {
		r.ob.targetArea = 0
	} else if fresh {
		r.ob.crossArea = area
	}
	r.obMu.Unlock()
	if jump {
		r.e.Log.Info("stale target area dropped: arrived from town, not by crossing a border", "area", area, "target", ta, "name", r.areaName(ta))
		return ""
	}
	if fresh {
		r.noteCrossing(ta)
	}
	return fmt.Sprintf("exploring crossed an area border (target area %d)", ta) // walk back
}

// banReason is the text for a refused area: the real ban reason, not always the death streak.
func (r *Runner) banReason(area int) string {
	r.obMu.Lock()
	why := r.avoidWhy[area]
	r.obMu.Unlock()
	switch why {
	case "", whyDeath:
		return "banned after repeated deaths"
	}
	return "skipped (" + why + ", not a death)"
}

// dangerBan says a ban reason is a danger signal (deaths) and not a fault of the bot's own walking.
func dangerBan(why string) bool { return why == "" || why == whyDeath || why == whyRepeat }

// jumpedHops is how many leading hops of a walk from cur the waypoint trip of travelTo skips (0 when none): a hop
// before the last waypoint area on the route that is far enough away to be worth the panel trip.
func (r *Runner) jumpedHops(cur int, hops []int) int {
	if !r.d.Plan.Waypoints[cur] || !r.e.Tune.Nav.UseWaypoints {
		return 0
	}
	minPath := max(2, int(r.e.Tune.Nav.WpMinDistance))
	for i := len(hops) - 1; i >= 0; i-- {
		if i+1 >= minPath && r.d.Plan.Waypoints[hops[i]] {
			return i
		}
	}
	return 0
}

// questWins is true while a quest step walks: the loop breaker's ban on the step's own area must not stop the step.
func (r *Runner) questWins(h int, why string) bool {
	if why != whyBorderLoop || !r.questTravel.Load() {
		return false
	}
	r.e.Log.Info("quest step wins over the border-loop ban", "area", h, "name", r.areaName(h))
	return true
}

// Blocked journey: a quest step the guards refuse must not park the hero in town. After questBlockedFarmMin the hero
// farms the best leveling area for its level by waypoint (the same pick as the level steps), and retries the step
// after a growing wait.
type blockedJourney struct {
	since   time.Time // first refusal of the current block
	why     string
	retryAt time.Time // while in the future the hero farms
	n       int       // retries that blocked again (backoff exponent)
	key     string    // last refusal text and how often it repeated (spin guard)
	repeats int
	backoff time.Duration
}

const (
	refuseSpinN   = 5                // identical refusals in a row before the spin guard slows the loop
	refuseSpinMax = 60 * time.Second // longest spin-guard wait
	farmRetryMax  = 6                // cap of the retry exponent (10 min x 64 would be a day)
)

func (r *Runner) blockedFarmAfter() time.Duration {
	return time.Duration(max(r.e.Tune.Bot.QuestBlockedFarmMin, 1)) * time.Minute
}

// spinGuard slows the refusal path: after refuseSpinN identical refusals in a row the wait doubles (1 s up to a
// minute) and one line is logged per doubling. A different refusal or a step that runs resets it.
func (r *Runner) spinGuard(ctx context.Context, key string) {
	b := &r.loop.task.bj
	if b.key != key {
		b.key, b.repeats, b.backoff = key, 0, 0
	}
	b.repeats++
	if b.repeats < refuseSpinN {
		return
	}
	if b.backoff == 0 {
		b.backoff = time.Second
	} else {
		b.backoff = min(b.backoff*2, refuseSpinMax)
	}
	r.e.Log.Warn("journey step refused again and again: backing off", "refusals", b.repeats, "waitSec", int(b.backoff/time.Second), "why", key)
	r.pause(ctx, b.backoff)
}

// journeyBlocked is called by taskStep when the plan is stuck or the step was refused. It returns handled = true when
// it took over this tick (farming or waiting).
func (r *Runner) journeyBlocked(ctx context.Context, snap *d2.Snapshot, c control, ex *task.Executor, why string) bool {
	b := &r.loop.task.bj
	now := r.e.Clock.Now()
	if b.since.IsZero() {
		b.since, b.why = now, why
		r.e.Log.Warn("journey blocked", "reason", why, "farmAfterMin", int(r.blockedFarmAfter()/time.Minute))
	}
	b.why = why
	r.noteJourneyBlocked(why)
	r.setWhy(WhyJourneyBlocked, "why", why)
	if now.Sub(b.since) < r.blockedFarmAfter() {
		r.spinGuard(ctx, why)
		r.setState(StateInTown, "Journey blocked: "+why)
		r.pause(ctx, time.Second)
		return true
	}
	if b.retryAt.IsZero() {
		b.retryAt = now.Add(r.blockedFarmAfter() << min(b.n, farmRetryMax))
		r.e.Log.Warn("journey blocked: farming the level plan, retrying the step later", "reason", why, "retryMin", int(b.retryAt.Sub(now)/time.Minute))
	}
	if !now.Before(b.retryAt) {
		for g := range ex.BlockedAll() {
			ex.Unpark(g)
		}
		b.retryAt = time.Time{}
		b.n++
		r.e.Log.Info("journey blocked: retrying the quest step", "retries", b.n)
		return false
	}
	lvl := snap.Player.Level + 1
	r.setState(StateClearing, fmt.Sprintf("Journey blocked (%s): farming for level %d", why, lvl))
	ok := r.do(ctx, c, StateClearing, fmt.Sprintf("Journey blocked: farming for level %d", lvl), 40*time.Minute, "journey farm", func(ctx context.Context) error {
		return taskActs{r, c}.LevelTo(ctx, lvl)
	})
	if !ok {
		r.pause(ctx, 5*time.Second)
	}
	return true
}

// journeyOK forgets the block once a quest step ran.
func (r *Runner) journeyOK() {
	r.loop.task.bj = blockedJourney{}
	r.noteJourneyBlocked("")
}

func (r *Runner) noteJourneyBlocked(why string) {
	r.mu.Lock()
	r.journeyWhy = why
	r.mu.Unlock()
}

func isBlockedPlan(p task.Plan) bool { return p.Stuck && strings.HasPrefix(p.Why, "blocked: ") }

// noteRefusal starts the blocked-journey clock on a danger refusal of a quest step and slows an identical repeat.
func (r *Runner) noteRefusal(ctx context.Context, res task.Result) {
	b := &r.loop.task.bj
	key := res.StepID + ": " + res.Err.Error()
	if b.since.IsZero() {
		b.since, b.why = r.e.Clock.Now(), res.Err.Error()
		r.e.Log.Warn("journey blocked", "step", res.StepID, "reason", b.why, "farmAfterMin", int(r.blockedFarmAfter()/time.Minute))
		r.noteJourneyBlocked(b.why)
	}
	r.spinGuard(ctx, key)
}

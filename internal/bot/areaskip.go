package bot

import (
	"fmt"
	"sort"
	"time"
)

// Why an area is out of play. Only a death streak is a danger signal; the others come from the bot's own faults
// (nav stuck, unreachable, a gate refusal) and must never keep it standing in town.
const (
	whyNavStuck    = "nav-stuck"
	whyUnreachable = "unreachable"
	whyGate        = "gate"
	whyDeath       = "death-streak"
	whyRoute       = "route"
	// whyRepeat (deathrule.go) is a repeat-site ban; idleBreak treats it as short-waited, like a fault, so the bot never idles in town for it
)

// idleWatch is how often a wait must be seen for it to count as one continuous wait.
const idleWatch = 30 * time.Second

// skipArea keeps the planner out of area for d and logs the producer. It takes obMu itself.
func (r *Runner) skipArea(area int, d time.Duration, why string) {
	r.obMu.Lock()
	r.skipAreaLocked(area, d, why)
	r.obMu.Unlock()
}

// skipAreaLocked is skipArea for a caller that holds obMu.
func (r *Runner) skipAreaLocked(area int, d time.Duration, why string) {
	r.avoid[area] = r.e.Clock.Now().Add(d)
	r.avoidWhy[area] = why
	r.noteDecision(fmt.Sprintf("skip area %d %s %ds reason=%s", area, r.areaName(area), int(d/time.Second), why))
	r.e.Log.Warn("area skipped", "area", area, "name", r.areaName(area), "seconds", int(d/time.Second), "reason", why)
}

// navSkip is how long a nav fault (stuck, no exit) keeps the planner out of an area: a second is enough to try another
// way or another area, a quarter of an hour left the bot standing in town (LIVE G1 attempt 14).
func (r *Runner) navSkip() time.Duration {
	return time.Duration(max(r.e.Tune.Bot.NavSkipSec, 10)) * time.Second
}

// areaIdleMax is the longest the bot waits in town for bans that are not a death streak.
func (r *Runner) areaIdleMax() time.Duration {
	return time.Duration(max(r.e.Tune.Bot.AreaIdleSec, 10)) * time.Second
}

// idleBreak is called every time the bot is about to wait in town because areas are banned. Once the wait has lasted
// areaIdleMax (a death streak alone: starterBanMax, at least that) it lifts the ban of the least bad area so the planner
// farms it with normal combat. force skips the time check (the stall detector). It reports whether a ban was lifted.
func (r *Runner) idleBreak(force bool) bool {
	l := &r.loop
	now := r.e.Clock.Now()
	type cand struct {
		area   int
		left   time.Duration
		death  bool
		starts bool
	}
	var cs []cand
	r.obMu.Lock()
	for a, until := range r.avoid {
		if until.After(now) {
			why := r.avoidWhy[a]
			cs = append(cs, cand{a, until.Sub(now), why == whyDeath || why == "", false})
		}
	}
	r.obMu.Unlock()
	if len(cs) == 0 {
		l.waitSince = time.Time{}
		return false
	}
	if l.waitSince.IsZero() || now.Sub(l.waitLast) > idleWatch {
		l.waitSince = now
	}
	l.waitLast = now
	anyOther, limit := false, r.areaIdleMax()
	for i := range cs {
		cs[i].starts = r.isStarter(cs[i].area)
		anyOther = anyOther || !cs[i].death
	}
	if !anyOther {
		limit = max(limit, r.starterBanMax())
	}
	if !force && now.Sub(l.waitSince) < limit {
		return false
	}
	// least bad: faults before death bans; among death bans the safe starter field first; then the shortest ban left
	sort.Slice(cs, func(i, j int) bool {
		a, b := cs[i], cs[j]
		switch {
		case a.death != b.death:
			return !a.death
		case a.death && a.starts != b.starts:
			return a.starts
		case a.left != b.left:
			return a.left < b.left
		}
		return a.area < b.area
	})
	best := cs[0]
	r.obMu.Lock()
	why := r.avoidWhy[best.area]
	delete(r.avoid, best.area)
	r.obMu.Unlock()
	l.stepArea = 0
	l.waitSince = now.Add(-limit + 10*time.Second) // if the planner still finds nothing, the next area follows soon
	r.e.Log.Warn(fmt.Sprintf("area wait: all areas banned/skipped, farming %s anyway (remaining %ds)", r.areaName(best.area), int(best.left/time.Second)),
		"area", best.area, "reason", why, "forced", force)
	return true
}

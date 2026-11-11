package bot

import (
	"context"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/repair"
)

// playerActionGap is the shortest time between two attempts of the player's queued item actions after one was
// postponed (a fight started under the click path).
const playerActionGap = 10 * time.Second

// safeMoment is the OPEN OPPORTUNITY predicate shared by the player's item actions and the errand trip: the player's
// commands rank directly behind safety (docs/inventory-actions.md, priority ladder), so only safety and state hold them:
// no town trip is running (it owns the hero), the pause after a postponed attempt is over, and nothing hostile is in
// reach (repairSafeMoment, the repair trip's own rule). The hero walking is NOT a reason to wait: a roam or travel walk
// is interrupted by the guard (playerDue). It returns the clock reading it used and, when it says no, the rule that held.
func (r *Runner) safeMoment(snap *d2.Snapshot) (time.Time, bool, string) {
	l := &r.loop
	now := r.e.Clock.Now()
	switch {
	case l.trip != nil:
		return now, false, "a town trip is running"
	case !l.playerActAt.IsZero() && now.Sub(l.playerActAt) < playerActionGap:
		return now, false, "pause after a postponed attempt"
	}
	act, why := repairSafeMoment(safeIn{Fight: r.hostileNear(snap, r.e.Tune.Bot.ClearRadius), Close: r.hostileWithin(snap, r.e.Tune.Bot.ClearRadius)}, repair.Trip{})
	return now, act == safeGo, why
}

// playerCancelGap is the least time between two guard interrupts for the player's commands, so a step that declines
// (the click path found a monster) cannot turn into an interrupt storm.
const playerCancelGap = 3 * time.Second

// playerDue is the guard's check (loop.go guardOnce): a player command is ready and the moment is safe, so the walk in
// progress (roam, explore, travel) is cancelled and the loop runs the command next. It reads loop state the main
// goroutine only writes between actions, like pointsDue. Safety actions are handled before it in the guard.
func (r *Runner) playerDue(s *d2.Snapshot, now time.Time) bool {
	if s == nil || s.Player == nil || r.loop.trip != nil {
		return false
	}
	if last := r.playerCancelNs.Load(); last != 0 && now.UnixNano()-last < int64(playerCancelGap) {
		return false
	}
	due := r.playerPending(s, now) || r.repairDue(s, now)
	if due {
		r.playerCancelNs.Store(now.UnixNano())
	}
	return due
}

func (r *Runner) playerPending(s *d2.Snapshot, now time.Time) bool {
	pg := r.d.PlayerGear
	if pg == nil {
		return false
	}
	l := &r.loop
	if !l.playerActAt.IsZero() && now.Sub(l.playerActAt) < playerActionGap {
		return false
	}
	want := pg.Pending() > 0
	if k, ok := pg.(playerKinds); ok && !want && r.d.Town != nil && !s.Player.InTown() && k.ErrandsWaiting() > 0 && !now.Before(l.errandSkipUntil) {
		want = true // an errand: the trip home starts at this safe moment
	}
	if !want {
		return false
	}
	return r.hostileNear(s, r.e.Tune.Bot.ClearRadius) == 0 && r.hostileWithin(s, r.e.Tune.Bot.ClearRadius) == 0
}

// repairDue: an open "repair now" request whose trip may start (repairTripStep's own gates) interrupts the walk too.
// A "farm until repair" request keeps playing until its figure is reached.
func (r *Runner) repairDue(s *d2.Snapshot, now time.Time) bool {
	rc := r.rep
	if rc == nil || s.Player.InTown() || r.noTP.Load() || r.bossRun.Load() {
		return false
	}
	q, open := rc.Active()
	if !open || q.Kind != repair.KindRepair || (q.Phase != repair.PhaseQueued && q.Phase != repair.PhaseWaiting) {
		return false
	}
	l := &r.loop
	if (!l.repairRetryAt.IsZero() && now.Before(l.repairRetryAt)) || rc.TripWait(now) > 0 {
		return false
	}
	return r.hostileNear(s, r.e.Tune.Bot.ClearRadius) == 0 && r.hostileWithin(s, rc.TripCfg().PortalClearTiles) == 0
}

// noteDelayed logs which rule held a queued player command (at most every 10 s) so a slow command is explained.
func (r *Runner) noteDelayed(now time.Time, why string) {
	if why == "" || now.Sub(r.loop.playerWhyAt) < 10*time.Second {
		return
	}
	r.loop.playerWhyAt = now
	r.e.Log.Info("player action waits for a safe moment", "rule", why)
}

// playerKinds is the optional part of PlayerGear the Why line and the errand trip read.
type playerKinds interface {
	NextKind() string    // kind of the request that runs first ("identify", "equip", ...)
	ErrandsWaiting() int // queued town errands (buy scrolls, Cain)
}

func nextKind(pg PlayerGear) string {
	if k, ok := pg.(playerKinds); ok {
		if n := k.NextKind(); n != "" {
			return n
		}
	}
	return "item"
}

// errandBackoff pauses the errand trips after one failed to reach town.
const errandBackoff = 5 * time.Minute

// errandTripStep takes the hero home at the next safe moment when the player queued a town errand (buy identify scrolls,
// Cain) and the hero is in the field. It comes right after the player's own actions in runGoal, so it outranks the routine
// chores. Once in town the same playerActionStep runs the errand. The walk home is Town.ToTown, as the claim trip's.
func (r *Runner) errandTripStep(ctx context.Context, snap *d2.Snapshot, c control) bool {
	k, ok := r.d.PlayerGear.(playerKinds)
	if !ok || r.d.Town == nil || snap == nil || snap.Player == nil || snap.Player.InTown() || k.ErrandsWaiting() == 0 {
		return false
	}
	l := &r.loop
	if r.e.Clock.Now().Before(l.errandSkipUntil) {
		return false
	}
	if _, ok, _ := r.safeMoment(snap); !ok {
		return false
	}
	r.setWhy(WhyPlayerItems, "what", "errand")
	if r.do(ctx, c, StateResupplying, "Going to town for the errand you asked for", sec(r.cfg.TownTimeoutSec), "errand trip", r.d.Town.ToTown) {
		return true
	}
	l.errandSkipUntil = r.e.Clock.Now().Add(errandBackoff)
	r.e.Log.Warn("errand trip failed; no new errand trip for a while", "minutes", int(errandBackoff/time.Minute))
	return true
}

// playerActionStep performs the item actions the player queued on the watch page (docs/inventory-actions.md): equip a
// backpack item, trash a backpack item. The OPEN OPPORTUNITY is the same predicate as the points step: in town, or in
// the field while no hostile is within the clear radius (hostileNear; no fight is on), no town trip is running (the
// trip owns the hero); a walk is interrupted, not waited for (playerDue). The click path checks again
// (leveler.busy) and postpones the request when a monster is close. It never undoes anything: a worn item stays pinned,
// a trashed one stays dropped. It reports whether it acted.
func (r *Runner) playerActionStep(ctx context.Context, snap *d2.Snapshot, c control) bool {
	pg := r.d.PlayerGear
	if pg == nil || snap == nil || snap.Player == nil || pg.Pending() == 0 {
		return false
	}
	now, ok, why := r.safeMoment(snap)
	if !ok {
		r.noteDelayed(now, why)
		return false
	}
	l := &r.loop
	l.playerActAt = now
	r.setWhy(WhyPlayerItems, "what", nextKind(pg))
	r.do(ctx, c, StateInTown, "Item actions from the watch page", sec(r.cfg.TownTimeoutSec), "player item actions", func(ctx context.Context) error {
		for _, line := range pg.RunPending(ctx) {
			r.e.Log.Info("player " + line)
		}
		return ctx.Err()
	})
	return true
}

// QuestView is the quest reader's view for a difficulty (memory first, then the save); ok is false without a reader or
// when nothing is known. The watch page's "Identify all (Cain)" command reads it to see whether the Search for Cain is done.
func (r *Runner) QuestView(diff int) (QuestView, bool) {
	if r.d.Task == nil || r.d.Task.Quests == nil {
		return QuestView{}, false
	}
	return r.d.Task.Quests.Read(diff)
}

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

// Player modes while the hero walks or runs (U: the numbers are the game's player mode table, 2 walk, 3 run, 6 town
// walk; not yet seen in a live capture of this bot). Any other mode, including an unfilled 0, counts as standing.
func heroMoving(p *d2.Player) bool { return p.Mode == 2 || p.Mode == 3 || p.Mode == 6 }

// safeMoment is the OPEN OPPORTUNITY predicate shared by the player's item actions and the errand trip (and for the repair
// trip to reuse): no town trip is running (it owns the hero), the pause after a postponed attempt is over, the hero is not
// walking or running and no hostile is within the clear radius. It returns the clock reading it used.
func (r *Runner) safeMoment(snap *d2.Snapshot) (time.Time, bool) {
	l := &r.loop
	now := r.e.Clock.Now()
	switch {
	case l.trip != nil:
		return now, false
	case !l.playerActAt.IsZero() && now.Sub(l.playerActAt) < playerActionGap:
		return now, false
	case heroMoving(snap.Player):
		return now, false // stand still first: the click path needs a quiet hero
	}
	// the one safe-moment rule (repairSafeMoment, repairtrip.go): nothing hostile in reach, the fighter's ignore list not applied
	act, _ := repairSafeMoment(safeIn{Fight: r.hostileNear(snap, r.e.Tune.Bot.ClearRadius), Close: r.hostileWithin(snap, r.e.Tune.Bot.ClearRadius)}, repair.Trip{})
	return now, act == safeGo
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
	if _, ok := r.safeMoment(snap); !ok {
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
// trip owns the hero) and the hero is not walking or running. The click path checks again
// (leveler.busy) and postpones the request when a monster is close. It never undoes anything: a worn item stays pinned,
// a trashed one stays dropped. It reports whether it acted.
func (r *Runner) playerActionStep(ctx context.Context, snap *d2.Snapshot, c control) bool {
	pg := r.d.PlayerGear
	if pg == nil || snap == nil || snap.Player == nil || pg.Pending() == 0 {
		return false
	}
	now, ok := r.safeMoment(snap)
	if !ok {
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

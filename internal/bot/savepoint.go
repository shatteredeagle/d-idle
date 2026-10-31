package bot

import (
	"context"
	"time"

	"d2lod-bot/internal/d2"
)

// Save points. The game writes the .d2s only on Save and Exit, never on area changes (V: SAVECHECK on i5, docs/
// soak-analysis.md 6c), so a crash or kill -9 loses all XP, items, gold and quest progress since the character
// entered the game. Every game entry (first start, re-entry, recycle, farm NewGame, fresh-game leveling) is
// the last save to the observer (ob.inGameAt, which also resets savedExp and so the xpRisk/XPLOSS figures); a long
// game in town gets a planned Save and Exit and re-entry of the same character.
const (
	savePointRetry      = 5 * time.Minute // wait after a failed save point
	savePointAfterDeath = 2 * time.Minute
	savePointMinLife    = 50 // percent
)

// savePointDue reports whether a save point may run now, and why not when it may not. Pure given the observer
// times, so the guards are unit-tested without a loop.
func (r *Runner) savePointDue(s *d2.Snapshot, c control, now time.Time) (bool, string) {
	every := time.Duration(r.e.Tune.Bot.SavePointMin) * time.Minute
	if every <= 0 {
		return false, "off"
	}
	l := &r.loop
	if r.d.Cycler == nil {
		return false, "no cycler"
	}
	if s == nil || s.Player == nil || !s.Player.InTown() || isDead(s.Player) {
		return false, "not in town"
	}
	if l.trip != nil {
		return false, "trip in progress"
	}
	if s.UI.BlockingPanelOpen() {
		return false, "panel open"
	}
	if s.Player.LifePct() < savePointMinLife {
		return false, "low life"
	}
	if r.hostileNear(s, r.e.Tune.Bot.ClearRadius) > 0 {
		return false, "hostile near"
	}
	if now.Before(l.saveNext) {
		return false, "backoff"
	}
	if c.spec.camp != nil && len(l.task.pseudo) > 0 {
		return false, "mid quest step" // staff placed, book read ...: that memory is per game
	}
	r.obMu.Lock()
	entered, lastDeath, rec := r.ob.inGameAt, r.ob.lastDeathAt, r.loop.rec
	r.obMu.Unlock()
	if rec != nil {
		return false, "recovering from a death"
	}
	if !lastDeath.IsZero() && now.Sub(lastDeath) < savePointAfterDeath {
		return false, "recent death"
	}
	if entered.IsZero() || now.Sub(entered) < every {
		return false, "too soon"
	}
	return true, ""
}

// savePointStep leaves and re-enters the game when a save point is due. It reports true when it acted. It is
// out-of-game work like a recycle: failures only back off, they never feed the failsafe ladder or the re-entry
// pause (noteReentry), and the minute line does not count the new game as a restart.
func (r *Runner) savePointStep(ctx context.Context, s *d2.Snapshot, c control) bool {
	now := r.e.Clock.Now()
	if ok, _ := r.savePointDue(s, c, now); !ok {
		return false
	}
	l := &r.loop
	l.saveNext = now.Add(savePointRetry)
	r.logXPAtRisk("save point")
	r.e.Log.Info("save point: saving and re-entering the game", "level", s.Player.Level)
	r.obMu.Lock()
	r.ob.plannedGames++
	r.obMu.Unlock()
	diff := difficultyName(s.Player.Difficulty)
	err, _ := r.runAction(ctx, c.gen, StateRecovering, "Save point: saving the game", sec(r.cfg.TownTimeoutSec), r.oogRun(func(ctx context.Context) error {
		return r.d.Cycler.NewGame(ctx, diff)
	}))
	if err != nil {
		if ctx.Err() == nil {
			r.e.Log.Warn("save point failed; trying again later", "err", err, "retry", savePointRetry)
		}
		r.obMu.Lock()
		r.ob.plannedGames-- // no new game came of it
		r.obMu.Unlock()
		return true
	}
	r.syncTaskGame()
	return true
}

package bot

import (
	"context"
	"strconv"
	"time"

	"d2lod-bot/internal/d2"
)

// The kill/XP drought watchdog (Gate 2 g2d, docs/research/g2d-nokill-diagnosis.md section 5.4). The campaign stall
// ladder counts area entries, purchases and quest steps as progress, so it cannot see ten minutes without a kill; this
// one looks at kills and XP only, which is what the NOKILL criterion measures.

// droughtMinField is how long the character must have stood in a field area before a drought counts: the minutes of
// a town trip and its walk back are chores, not a field without monsters.
const droughtMinField = 30 * time.Second

// droughtStage returns 0 (nothing to do) or the number of the drought action now due (1 leave the area, 2 and up a
// fresh game). It changes no state besides arming the clock.
func (r *Runner) droughtStage(s *d2.Snapshot, now time.Time) int {
	m := time.Duration(r.e.Tune.Bot.DroughtMin) * time.Minute
	if m <= 0 || s == nil || s.Player == nil {
		return 0
	}
	r.obMu.Lock()
	if o := &r.ob; o.killXPAt.After(o.droughtAt) {
		o.droughtStage = 0 // a kill or XP gain since the last action ends the series
	}
	r.obMu.Unlock()
	p := s.Player
	if p.InTown() || isDead(p) || (p.MaxLife > 0 && p.Life*100 < 60*p.MaxLife) {
		return 0
	}
	r.mu.Lock()
	active := r.goal != nil && !r.paused
	r.mu.Unlock()
	r.obMu.Lock()
	defer r.obMu.Unlock()
	o := &r.ob
	if !active || o.killXPAt.IsZero() {
		o.killXPAt = now // the window starts when the bot is active
		return 0
	}
	anchor := o.killXPAt
	if o.droughtAt.After(anchor) {
		anchor = o.droughtAt
	}
	if o.fieldSince.IsZero() || now.Sub(o.fieldSince) < droughtMinField || now.Sub(anchor) < m {
		return 0
	}
	return o.droughtStage + 1
}

// droughtCheck acts on a due drought. Stage 1 keeps the planner out of the current area for a while and re-plans
// (the next route step); a goal that names its area cannot skip it, and a second drought in a row starts a fresh game.
// It reports whether it acted (the caller then skips the rest of the step).
func (r *Runner) droughtCheck(ctx context.Context, s *d2.Snapshot, c control) bool {
	now := r.e.Clock.Now()
	stage := r.droughtStage(s, now)
	if stage == 0 {
		return false
	}
	area := s.Player.Area
	r.obMu.Lock()
	r.ob.droughtAt, r.ob.droughtStage = now, stage
	since := now.Sub(r.ob.killXPAt)
	r.obMu.Unlock()
	r.noteNokill("drought" + strconv.Itoa(stage))
	r.e.Log.Warn("kill drought", "stage", stage, "area", area, "name", r.areaName(area), "minutes", int(since/time.Minute))
	if stage == 1 && r.skipTargetFor(whyDrought, 4*r.navSkip()) {
		return true
	}
	if r.d.Cycler == nil {
		return false
	}
	err := r.newGameRun(func(ctx context.Context) error {
		return r.d.Cycler.NewGame(ctx, difficultyName(s.Player.Difficulty))
	})(ctx)
	if err != nil {
		r.e.Log.Warn("kill drought: fresh game failed", "err", err)
		return false
	}
	r.forgetGame()
	return true
}

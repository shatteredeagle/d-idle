package bot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
)

// ErrCampaignStalled is what Run returns when the campaign made no progress through the whole stall ladder; d2bot maps
// it to exit status 5 so the supervisor restarts the process (NOWALL: soaks sat 60 and 167 wall minutes in a town
// wedge printing the same line, the clock advanced so the loop watchdog never fired).
var ErrCampaignStalled = errors.New("campaign stalled, exiting for a restart")

// campaignStall is the stall ladder, run every step while a goal is active. Campaign progress is a kill or xp gain, a
// non-town area entered, a purchase or a quest step (observer.campaignAt), all measured on the virtual clock.
// Ladder: bot.stallWarnMin without progress logs one WARN and starts a fresh game (forgetGame); bot.stallExitMin more
// without progress ends Run with ErrCampaignStalled. Any progress drops back to stage 0. It reports whether it acted
// (the caller then skips the rest of the step). Shared by both backends; the wedge it answers was seen on d2d only.
func (r *Runner) campaignStall(ctx context.Context, s *d2.Snapshot, active bool) bool {
	warn := time.Duration(r.e.Tune.Bot.StallWarnMin) * time.Minute
	now := r.e.Clock.Now()
	r.obMu.Lock()
	last := r.ob.campaignAt
	if last.IsZero() || !active || warn <= 0 || s == nil {
		// no goal, paused, not in game, off: nothing counts, the window starts when the bot is active again
		r.ob.campaignAt = now
		r.loop.stallStage = 0
		r.obMu.Unlock()
		return false
	}
	r.obMu.Unlock()
	l := &r.loop
	if l.stallStage > 0 && last.After(l.stallAt) {
		l.stallStage = 0
	}
	switch {
	case l.stallStage == 0 && now.Sub(last) >= warn:
		l.stallStage, l.stallAt = 1, now
		r.e.Log.Warn(fmt.Sprintf("campaign stalled: no progress for %d virtual min", int(now.Sub(last)/time.Minute)),
			"lastProgress", last.Format(time.RFC3339), "state", string(r.stateNow()), "step", r.activityNow(), "area", s.Player.Area)
		if r.d.Cycler == nil {
			return false
		}
		err := r.newGameRun(func(ctx context.Context) error {
			return r.d.Cycler.NewGame(ctx, difficultyName(s.Player.Difficulty))
		})(ctx)
		if err != nil {
			r.e.Log.Warn("campaign stalled: fresh game failed", "err", err)
		} else {
			r.forgetGame()
		}
		return true
	case l.stallStage == 1 && now.Sub(l.stallAt) >= time.Duration(max(r.e.Tune.Bot.StallExitMin, 1))*time.Minute:
		r.e.Log.Error("campaign stalled, exiting for a restart", "sinceProgress", now.Sub(last).Round(time.Minute).String(), "lastProgress", last.Format(time.RFC3339))
		r.mu.Lock()
		r.crash = fmt.Errorf("%w: no progress for %d virtual min", ErrCampaignStalled, int(now.Sub(last)/time.Minute))
		r.mu.Unlock()
		return true
	}
	return false
}

func (r *Runner) stateNow() State {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.st
}

func (r *Runner) activityNow() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.activity
}

package bot

import (
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/questboard"
)

// The knobs of the opt-in auto-start (slice S8, docs/questboard.md "Auto-start and away"): the limits live in
// internal/env QuestsTuning, the rule in internal/questboard/auto.go. Nothing here runs with quests.board = 0.

func (r *Runner) questAutoOptions() questboard.AutoOptions {
	q := r.e.Tune.Quests
	return questboard.AutoOptions{DelaySec: q.AutoDelaySec, AwayMin: q.AwayMin, HoldMin: q.AutoHoldMin, GapMin: q.AutoGapMin, DayCap: q.AutoDayCap}
}

// noteQuestHold tells the board the player paused, resumed or reset: no auto-start for quests.autoHoldMin minutes.
func (r *Runner) noteQuestHold() {
	if rt := r.qbRT(); rt != nil {
		rt.exec.NoteHold()
	}
}

// QuestLevel is the loaded hero's level (0 before a snapshot), for "levels to go" on an offer.
func (r *Runner) QuestLevel() int {
	if s := r.latest(); s != nil && s.Player != nil {
		return s.Player.Level
	}
	return 0
}

// questResumeOptions is the pause policy of the board (quests.resumeAfterDeaths, table in data/questboard/resume.json).
// A table that does not parse leaves the policy off: a parked chunk then waits for the player, as before.
func (r *Runner) questResumeOptions() questboard.ResumeOptions {
	if r.e.Tune.Quests.ResumeAfterDeaths == 0 {
		return questboard.ResumeOptions{}
	}
	p, err := questboard.LoadResume()
	if err != nil {
		r.boardNote("resume", "quest board: pause policy off: "+err.Error())
		return questboard.ResumeOptions{}
	}
	return questboard.ResumeOptions{Enabled: true, Policy: p}
}

// questRefusalParkAfter is the number of identical coded refusals that park a chunk (quests.refusalPark, table in
// data/questboard/refusal.json). A table that does not parse leaves the rule off.
func (r *Runner) questRefusalParkAfter() int {
	if r.e.Tune.Quests.RefusalPark == 0 {
		return 0
	}
	p, err := questboard.LoadRefusal()
	if err != nil {
		r.boardNote("refusal", "quest board: refusal park off: "+err.Error())
		return 0
	}
	return p.ParkAfter
}

// questResumeInput is what the pause policy needs from the hero now.
func (r *Runner) questResumeInput(snap *d2.Snapshot) questboard.ResumeInput {
	r.obMu.Lock()
	died := r.ob.lastDeathAt
	r.obMu.Unlock()
	return questboard.ResumeInput{
		Paused:    r.PlayerPauseView().State != PauseRunning,
		Calm:      r.safeToPause(snap),
		Recovery:  r.recoveryActive(),
		LifePct:   snap.Player.Life * 100 / max(snap.Player.MaxLife, 1),
		LastDeath: died,
	}
}

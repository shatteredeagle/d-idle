package bot

import (
	"context"
	"fmt"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/questboard"
)

// The between-chunks modes of the quest board (slice S6, docs/questboard.md "Modes"; design 3.4). With quests.board = 1
// and no accepted chunk the hero is in the mode the player picked:
//
//	train  the leveling slice (best zone for the level, the zone model when bot.zoneModel is 1) with the next quest
//	       as the pinned target: it names the goal and feeds the readiness meter; it never starts the quest.
//	idle   parked in town on purpose: the town chores of the normal loop, no field work, no watchdog stall.
//	farm   the leveling slice with the farm-gold zone choice (one rung below what the level is ready for, so never
//	       above the readiness line); the farm loop keeps its own rules there.
//
// An accepted chunk (Run, Await, Busy) comes first and never reads the mode; after the hand-in the hero returns to it.
// A mode change is adopted at the next tick boundary where it is safe (in town, or nothing hostile in the clear radius),
// so a fight or a chunk slice is never cut for it. Player orders and the Farm gold order sit above boardStep in goal.go
// and so override the mode. Knob 0: none of this runs.

const (
	modeTrain = "train"
	modeIdle  = "idle"
	modeFarm  = "farm"
)

// boardMode is the mode the hero acts on now. It follows the stored choice only at a safe moment.
func (r *Runner) boardMode(snap *d2.Snapshot) string {
	rt := r.qbRT()
	if rt == nil {
		return modeTrain
	}
	want := rt.exec.Mode()
	if !questboard.ValidMode(want) {
		want = questboard.DefaultMode
	}
	rt.mu.Lock()
	cur := rt.applied
	rt.mu.Unlock()
	if cur == want {
		return cur
	}
	if cur != "" && !r.safeToPause(snap) {
		return cur // a fight is on: the change waits
	}
	rt.mu.Lock()
	rt.applied = want
	rt.mu.Unlock()
	if cur != "" {
		r.e.Log.Info("quest board: mode changed", "from", cur, "to", want)
	}
	return want
}

// boardBetween is the tick of a hero with no chunk to run: the mode decides.
func (r *Runner) boardBetween(ctx context.Context, snap *d2.Snapshot, c control) bool {
	switch r.boardMode(snap) {
	case modeIdle:
		return r.boardIdle(ctx, snap, c)
	case modeFarm:
		return r.boardFarm(ctx, snap, c)
	}
	return r.boardTrain(ctx, snap, c)
}

// boardIdle parks the hero in town: home first (when it stands in the field), then the chores the town loop does
// anyway. The watchdog is told it is on purpose (the same idea as Pause); nothing goes into the field.
func (r *Runner) boardIdle(ctx context.Context, snap *d2.Snapshot, c control) bool {
	r.parkedBoard.Store(true)
	ok := r.rest(ctx, snap, c)
	if snap.Player.InTown() {
		r.setState(StateInTown, "Quest board: idle in town, waiting for you")
	}
	return ok
}

// boardFarm runs the leveling slice with the farm zone choice (target0 reads boardFarming).
func (r *Runner) boardFarm(ctx context.Context, snap *d2.Snapshot, c control) bool {
	rt := r.qbRT()
	if rt == nil {
		return r.boardTrain(ctx, snap, c)
	}
	rt.mu.Lock()
	rt.farming = true
	rt.mu.Unlock()
	defer func() { rt.mu.Lock(); rt.farming = false; rt.mu.Unlock() }()
	r.boardNote("mode", "quest board: farming loot and gold, not training toward the next quest")
	return r.levelSlice(ctx, snap, c, "Quest board: farming loot and gold", "boardfarm")
}

// boardFarming is asked by the area choice: true only inside a farm slice, so a quest step that levels is not bent.
func (r *Runner) boardFarming() bool {
	rt := r.qbRT()
	if rt == nil {
		return false
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.farming
}

// boardTarget is the pinned training target and the line the log and the meter say about it.
func (r *Runner) boardTarget(snap *d2.Snapshot) (questboard.Target, bool) {
	rt := r.qbRT()
	if rt == nil {
		return questboard.Target{}, false
	}
	st, err := rt.exec.State()
	if err != nil {
		return questboard.Target{}, false
	}
	return questboard.NextTarget(rt.board, st.Views, rt.classNow())
}

func (rt *boardRT) classNow() string {
	_, class, _ := rt.world()
	return class
}

// QuestMeter is DeskConfig.Meter: the mode block of GET /api/quests (words, no margins).
func (r *Runner) QuestMeter(mode string, st questboard.Status) *questboard.Meter {
	rt := r.qbRT()
	s := r.latest()
	if rt == nil || s == nil || s.Player == nil {
		return nil
	}
	_, class, _ := rt.world()
	rt.mu.Lock()
	applied := rt.applied
	rt.mu.Unlock()
	running := ""
	if a := st.Active; a != nil {
		if q, err := rt.board.Quest(a.Quest); err == nil {
			running = q.Title
		}
	}
	t, has := questboard.NextTarget(rt.board, st.Views, class)
	m := questboard.BuildMeter(mode, applied, t, has, s.Player.Level, running)
	return &m
}

// noteTrain logs the pinned target once per distinct line.
func (r *Runner) noteTrain(snap *d2.Snapshot) {
	t, has := r.boardTarget(snap)
	if !has {
		return
	}
	m := questboard.BuildMeter(modeTrain, "", t, true, snap.Player.Level, "")
	r.boardNote("train", fmt.Sprintf("quest board: %s", m.Line))
}

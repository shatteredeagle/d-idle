package bot

import (
	"bytes"
	"log/slog"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/questboard"
	"d2lod-bot/internal/task"
)

// Den reward after a town trip (drill 2026-11-22 section 12): a portal trip right after the clear loses the reward game-side.
// The first hold covered potion restocks only; a full pack, a rebuy or a merc errand left by portal all the same. Every
// non-emergency trip waits (bounded) while the live reader lists a quest cleared with its reward pending.
func TestPendingRewardHoldsEveryNonEmergencyTrip(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		q := &fakeQuests{}
		g := newRig(t)
		g.r.d.Task = &TaskDeps{Quests: q, QuestsLive: true}
		g.w.set(func(w *world) {
			w.area, w.gold0, w.level = 8, 0, 8
			w.items = nil
			for i := 0; i < 6; i++ { // 36 of 40 cells: the pack-full trip
				w.items = append(w.items, d2.Unit{ID: uint32(800 + i), Type: d2.TypeItem, Class: 3, Item: &d2.Item{Where: d2.ContInventory}})
			}
		})
		snap := func() *d2.Snapshot {
			s, _ := g.w.snapshot()
			s.Player.Class = class
			return s
		}
		plan, emerg, ok := g.r.needChores(snap())
		if !ok || emerg || restockReason(plan.Reason) {
			t.Fatalf("%v: want a plain pack-full trip without a pending reward, got ok=%v emergency=%v %q", class, ok, emerg, plan.Reason)
		}
		q.set(task.QDen, task.QuestState{Stage: task.StageCleared, Ev: task.EvMemory})
		if _, _, ok := g.r.needChores(snap()); ok {
			t.Fatalf("%v: a pack-full trip by portal must wait for the pending Den reward", class)
		}
		if g.r.tripDue(snap()) {
			t.Fatalf("%v: the guard must not cancel the clear for a trip that is held", class)
		}
		g.clock.Sleep(rewardHoldMax + time.Second)
		if _, _, ok := g.r.needChores(snap()); !ok {
			t.Fatalf("%v: the hold is bounded", class)
		}
		q.set(task.QDen, task.QuestState{Stage: task.StageDone, Ev: task.EvMemory})
		if _, _, ok := g.r.needChores(snap()); !ok || g.r.loop.rewardSince.IsZero() == false {
			t.Fatalf("%v: reward taken: trip due again and the hold reset (since %v)", class, g.r.loop.rewardSince)
		}
	}
}

// L4 (live review 2026-11-28): the reward word of a quest whose chunk the board will not run (an owner-only chunk, or
// nothing accepted) holds no trip, and a real hold writes its line once, not three times a second.
func TestRewardNobodyClaimsHoldsNoTripAndTheHoldLogsOnce(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		for _, board := range []bool{true, false} { // true: a board with nothing accepted; false: no board built yet (the hold stands)
			g := newRig(t)
			var buf bytes.Buffer
			g.e.Log = slog.New(slog.NewTextHandler(&buf, nil))
			g.r.e.Log = g.e.Log
			q := &fakeQuests{}
			g.r.d.Task = &TaskDeps{Quests: q, QuestsLive: true}
			if board {
				b, err := questboard.Load()
				if err != nil {
					t.Fatal(err)
				}
				st, err := questboard.Open(b, t.TempDir(), "Testy")
				if err != nil {
					t.Fatal(err)
				}
				g.r.NewQuestBoard(b, st)
			}
			g.w.set(func(w *world) {
				w.area, w.gold0, w.level = 8, 0, 18
				w.items = nil
				for i := 0; i < 6; i++ {
					w.items = append(w.items, d2.Unit{ID: uint32(800 + i), Type: d2.TypeItem, Class: 3, Item: &d2.Item{Where: d2.ContInventory}})
				}
			})
			q.set(task.QCain, task.QuestState{Stage: task.StageCleared, Ev: task.EvMemory})
			due := 0
			for i := 0; i < 180; i++ { // 60 fake seconds, three asks a second
				s, _ := g.w.snapshot()
				s.Player.Class = class
				if _, _, ok := g.r.needChores(s); ok {
					due++
				}
				g.clock.Sleep(time.Second / 3)
			}
			if board && due != 180 {
				t.Errorf("%v: a reward the board will not walk home held the trip (%d of 180 asks due)", class, due)
			}
			if !board && due != 0 {
				t.Errorf("%v: without a built board the bounded hold stands (%d due)", class, due)
			}
			if n := count(&buf, `"trip held: a quest reward is pending, the board walks home first"`); n > 1 {
				t.Errorf("%v board=%v: %d trip-held lines in 60 s, want at most one", class, board, n)
			}
		}
	}
}

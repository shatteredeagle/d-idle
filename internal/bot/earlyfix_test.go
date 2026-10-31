package bot

import (
	"context"
	"errors"
	"testing"
	"time"

	"d2lod-bot/internal/task"
)

func denRig(t *testing.T, q *fakeQuests) *rig {
	g := newRig(t)
	g.r.d.Quester = fakeQuester{w: g.w}
	g.r.d.Task = &TaskDeps{Quests: q, QuestsLive: true}
	g.mover.exploreFn = func(ctx context.Context, done func() bool) error {
		if g.r.Status().Character.AreaID == areaDenOfEvil {
			return nil // exploration ran out
		}
		return blockExplore(ctx, done)
	}
	g.w.set(func(w *world) { w.level = 2 })
	return g
}

// Without the quest word the exhausted Den sweep must not start a claim (LIVE: claim failed, the planner went back).
func TestDenNotClaimedWhileQuestWordOpen(t *testing.T) {
	g := denRig(t, &fakeQuests{})
	g.run()
	defer g.finish()
	g.r.Start(reach(10))
	// Wait on the call log, not on Status(): the sweep is exhausted at once and the bot leaves the Den again within
	// microseconds, so under CPU load a poll of the current area misses the visit (timed out twice in -p 4 runs).
	g.waitFor("den visited", func() bool { return g.w.count("exit 8") >= 1 })
	time.Sleep(300 * time.Millisecond)
	if n := g.w.count("claim"); n != 0 {
		t.Fatalf("claimed %d times with the objective open: %s", n, g.w.log())
	}
	if len(g.r.loop.questPending) != 0 {
		t.Fatalf("pending %v", g.r.loop.questPending)
	}
}

func TestDenClaimedWhenQuestWordCleared(t *testing.T) {
	q := &fakeQuests{}
	q.set(task.QDen, task.QuestState{Stage: task.StageCleared, Ev: task.EvMemory})
	g := denRig(t, q)
	g.run()
	defer g.finish()
	g.r.Start(reach(10))
	g.waitFor("claim", func() bool { return g.w.count("claim den_of_evil") == 1 })
}

// A claim trip that cannot get home is not retried at once.
func TestClaimTripBacksOffAfterFailure(t *testing.T) {
	g := newRig(t)
	g.r.d.Quester = fakeQuester{w: g.w}
	g.town.toTownErr = errors.New("town: no town portal available")
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.level = 2 })
	g.r.loop.questPending = map[string]bool{"den_of_evil": true}
	g.run()
	defer g.finish()
	g.r.Start(reach(10))
	g.waitFor("first claim trip", func() bool { return g.w.count("totown") >= 1 })
	time.Sleep(300 * time.Millisecond)
	if n := g.w.count("totown"); n != 1 {
		t.Fatalf("claim trip retried %d times: %s", n, g.w.log())
	}
}

// The area choice does not flip back and forth at the same level (Blood Moor / Cold Plains at the border).
func TestAreaChoiceHasHysteresis(t *testing.T) {
	g := newRig(t)
	g.w.set(func(w *world) { w.level = 3 })
	s, _ := g.w.snapshot()
	if a := g.r.holdTarget(s, 2); a != 2 {
		t.Fatal(a)
	}
	g.clock.Sleep(30 * time.Second)
	if a := g.r.holdTarget(s, 3); a != 2 {
		t.Fatalf("flipped after 30 s: %d", a)
	}
	g.clock.Sleep(holdMin)
	if a := g.r.holdTarget(s, 3); a != 3 {
		t.Fatalf("never switched: %d", a)
	}
	// a level change releases the hold at once
	g.w.set(func(w *world) { w.level = 4 })
	s, _ = g.w.snapshot()
	if a := g.r.holdTarget(s, 2); a != 2 {
		t.Fatalf("level change ignored: %d", a)
	}
}

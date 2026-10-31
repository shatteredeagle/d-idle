package bot

import (
	"context"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/task"
)

// Recorded numbers: G1 attempt 4, Soaky clvl 8, 0 gold, no weapon or body armor, no potions (docs/g1-log.md).
func soakyRig(t *testing.T, area int) *rig {
	g := campaignRig(t, &fakeQuests{}, 8)
	g.w.set(func(w *world) { w.area, w.gold0, w.items = area, 0, nil })
	g.w.belt()
	g.r.poll()
	return g
}

func levelGoalControl() control {
	return control{goal: &goals.Goal{}, spec: goalSpec{kind: goals.ReachLevel, level: 14}}
}

// The gated character's target is the safe area even where the planner's xp floor would refuse it (clvl 8, Blood Moor).
func TestGatedTargetIsTheSafeAreaDespiteTheFloor(t *testing.T) {
	g := soakyRig(t, 1)
	s := g.r.latest()
	a, ok := g.r.target0(s, levelGoalControl())
	if !ok || a != g.r.safeFarmArea(s) || a == 0 {
		t.Fatalf("target %d ok %v, want the safe area %d", a, ok, g.r.safeFarmArea(s))
	}
}

// The failsafe's skip rung must not take the only open area away from a gated character (the cause of 'no eligible
// reachable leveling area' live).
func TestSkipTargetKeepsTheGatedSafeArea(t *testing.T) {
	g := soakyRig(t, 2)
	g.r.ob.targetArea = 2
	g.r.skipTarget(whyNavStuck)
	if g.r.banned(2) {
		t.Fatal("the safe area was avoided")
	}
}

// A cleared safe area gives no target (fresh game) instead of a sweep that spins without progress.
func TestGatedClearedSafeAreaStartsFreshGame(t *testing.T) {
	g := soakyRig(t, 2)
	g.r.d.Cycler = fakeCycler{g.w}
	g.r.syncGames() // soakyRig polled: the observer already counts this game
	g.r.loop.cleared = map[int]bool{2: true}
	// in the field: walk home first, then the fresh game
	g.r.d.Town = nil
	if _, ok := g.r.target0(g.r.latest(), levelGoalControl()); ok {
		t.Fatal("cleared safe area must not be a target")
	}
	g.w.set(func(w *world) { w.area = 1 })
	g.r.poll()
	if err := (taskActs{r: g.r, c: control{goal: &goals.Goal{}}}).LevelTo(context.Background(), 14); err != nil {
		t.Fatal(err)
	}
	if g.w.count("newgame normal") != 1 {
		t.Fatalf("calls: %v", g.w.log())
	}
}

// A planning block (no eligible area, no way to start a fresh game) is waited out in town and never restarts the game.
func TestPlanningBlockNeverClimbsTheLadder(t *testing.T) {
	g := soakyRig(t, 1)
	g.r.loop.task.ex = nil
	res := task.Result{Outcome: task.OutBlocked, StepID: "level.14", Kind: task.KindNotFound,
		Err: task.Errorf(task.KindNotFound, "no eligible reachable leveling area for level 8")}
	if !planningFault(res) {
		t.Fatal("not_found is a planning fault")
	}
	if planningFault(task.Result{Outcome: task.OutRetry, Kind: task.KindUI, Err: task.Errorf(task.KindUI, "click failed")}) {
		t.Fatal("a UI fault is not a planning fault")
	}
	g.r.Start(goals.Goal{ID: "x", Type: goals.Idle})
	start := g.clock.Now()
	ctx := context.Background()
	for i := 0; i < planBlockMax; i++ {
		if err := g.r.planningBlock(ctx, res); err != nil {
			t.Fatal(err)
		}
	}
	if g.clock.Now().Sub(start) < 2*planBlockWait {
		t.Fatalf("no waiting in town: %v", g.clock.Now().Sub(start))
	}
	if g.r.loop.stuckStage != 0 || strings.Contains(g.w.log(), "restart") {
		t.Fatalf("ladder %d, calls %s", g.r.loop.stuckStage, g.w.log())
	}
	if !g.r.isSelfPaused() {
		t.Fatalf("after %d blocks it must pause for attention", planBlockMax)
	}
	_ = time.Second
}

// Healthy characters plan as before: a geared clvl 8 gets the planner's choice, not necessarily the first field.
func TestHealthyCharacterIsNotGated(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 8)
	g.r.poll()
	if v := g.r.gearVerdict(g.r.latest()); v.Gated {
		t.Fatalf("healthy gated: %+v", v)
	}
}

// Rebuy: a naked character in town with the gold gets the rebuy chores; at 0 gold it gets none but still sells loot.
func TestGatedRebuyAndSell(t *testing.T) {
	g := soakyRig(t, 1)
	g.w.set(func(w *world) { w.diff = 1 }) // a caster in Normal never counts armor (early-gear policy)
	if _, ok := g.r.rebuyDue(g.r.latest()); ok {
		t.Fatal("rebuy at 0 gold")
	}
	g.w.set(func(w *world) { w.gold0 = gearRebuyGold })
	g.r.poll()
	plan, ok := g.r.rebuyDue(g.r.latest())
	if !ok || !plan.Poverty || !plan.Sell {
		t.Fatalf("rebuy plan %+v ok %v", plan, ok)
	}
}

// The rebuy bar moves only when a rebuy trip starts: a call that the trip cooldown swallows must not lift the bar above
// the gold for good (G1 attempt 6: gold 3779, never rebuilt, the character stayed gated for two hours).
func TestRebuyBarIgnoresSwallowedCalls(t *testing.T) {
	g := soakyRig(t, 1)
	g.w.set(func(w *world) { w.diff = 1 })
	g.w.set(func(w *world) { w.gold0 = 400 })
	g.r.poll()
	for i := 0; i < 3; i++ { // evaluated again and again, no trip started
		if _, ok := g.r.rebuyDue(g.r.latest()); !ok {
			t.Fatalf("call %d: rebuy not due at 400 gold", i)
		}
	}
	if g.r.loop.rebuyBar != 0 {
		t.Fatalf("bar %d raised without a trip", g.r.loop.rebuyBar)
	}
	plan, _ := g.r.rebuyDue(g.r.latest())
	g.r.noteRebuyTrip(plan, 400)
	if g.r.loop.rebuyBar != 600 {
		t.Fatalf("bar %d after the trip, want 600", g.r.loop.rebuyBar)
	}
	if _, ok := g.r.rebuyDue(g.r.latest()); ok {
		t.Fatal("asked again at the same gold")
	}
}

// A refused quest step is logged once with its reason, not on every new game.
func TestStepRefusalLoggedOnce(t *testing.T) {
	g := soakyRig(t, 1)
	res := task.Result{Outcome: task.OutRetry, StepID: "tools.talk-charsi", Kind: task.KindDanger, Err: task.Errorf(task.KindDanger, "not ready: no weapon")}
	for i := 0; i < 3; i++ {
		g.r.noteStepRefused(res)
	}
	if len(g.r.loop.task.refused) != 1 {
		t.Fatalf("refused %v", g.r.loop.task.refused)
	}
	g.r.noteStepRefused(task.Result{Outcome: task.OutRan, StepID: "x"})
	if len(g.r.loop.task.refused) != 1 {
		t.Fatal("a step that ran was logged as refused")
	}
}

// After the fresh game starts the cleared set belongs to the old game: the next LevelTo slice must not start another
// fresh game before one clear pass has declared an area cleared again (live livelock: hundreds of restarts a minute
// at clvl 3, the task path never called syncGames).
func TestFreshGameDoesNotRetriggerBeforeANewClear(t *testing.T) {
	g := soakyRig(t, 2)
	g.r.d.Cycler = fakeCycler{g.w}
	g.r.syncGames() // soakyRig polled: the observer already counts this game
	g.r.loop.cleared = map[int]bool{2: true}
	g.w.set(func(w *world) { w.area = 1 })
	g.r.poll()
	acts := taskActs{r: g.r, c: control{goal: &goals.Goal{}}}
	if err := acts.LevelTo(context.Background(), 14); err != nil {
		t.Fatal(err)
	}
	g.r.obMu.Lock()
	g.r.ob.newGames++ // the observer saw the menus and the new game
	g.r.obMu.Unlock()
	g.r.poll()
	for i := 0; i < 3; i++ {
		acts.LevelTo(context.Background(), 14) // result irrelevant: it must not start a game
	}
	if n := g.w.count("newgame normal"); n != 1 {
		t.Fatalf("%d fresh games, calls: %v", n, g.w.log())
	}
}

// Same, but the observer never saw the menus (d2d restarts faster than a poll): the fresh game itself must forget the
// cleared set and the other per-game loop state (soakfull: hundreds of fresh games a minute at clvl 3).
func TestFreshGameForgetsClearedWithoutObserver(t *testing.T) {
	g := soakyRig(t, 2)
	g.r.d.Cycler = fakeCycler{g.w}
	g.r.syncGames()
	g.r.loop.cleared = map[int]bool{2: true}
	g.r.loop.sweepKills = map[int]int{2: 5}
	g.r.loop.stepArea = 7
	g.w.set(func(w *world) { w.area = 1 })
	g.r.poll()
	acts := taskActs{r: g.r, c: control{goal: &goals.Goal{}}}
	if _, err := acts.r.freshGameToLevel(context.Background(), g.r.latest()); err != nil {
		t.Fatal(err)
	}
	if len(g.r.loop.cleared) != 0 || g.r.loop.sweepKills != nil || g.r.loop.stepArea != 0 || !g.r.loop.needBuff {
		t.Fatalf("per-game state survived the fresh game: cleared %v sweepKills %v stepArea %d needBuff %v", g.r.loop.cleared, g.r.loop.sweepKills, g.r.loop.stepArea, g.r.loop.needBuff)
	}
	for i := 0; i < 4; i++ {
		acts.LevelTo(context.Background(), 14)
	}
	if n := g.w.count("newgame normal"); n != 1 {
		t.Fatalf("%d fresh games, calls: %v", n, g.w.log())
	}
}

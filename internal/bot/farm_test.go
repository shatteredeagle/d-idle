package bot

import (
	"errors"
	"sync"
	"time"

	"context"
	"d2lod-bot/internal/d2"
	"testing"

	"d2lod-bot/internal/boss"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/task"
)

func runsGoal(name string, n int) goals.Goal {
	return goals.Goal{Type: goals.Runs, Params: goals.Params{Area: name, Count: n}}
}

func TestRunsGoalRefusedWithoutFarm(t *testing.T) {
	g := newRig(t)
	if err := g.r.Start(runsGoal("Pindleskin", 1)); err == nil {
		t.Fatal("Start succeeded without Task.Farm")
	}
	g.r.d.Task = &TaskDeps{}
	if err := g.r.Start(runsGoal("Pindleskin", 1)); err == nil {
		t.Fatal("Start succeeded with nil Farm")
	}
}

func TestRunsGoalEmitsOneBossKillPerCompletedRun(t *testing.T) {
	g := newRig(t)
	var call FarmCall
	g.r.d.Task = &TaskDeps{Farm: func(ctx context.Context, c FarmCall) (boss.FarmReport, error) {
		if call.Ops == nil {
			call = c
		}
		c.OnRun("Pindleskin", true)
		c.OnRun("Pindleskin", false) // a failed run is not a kill
		c.OnRun("Pindleskin", true)
		return boss.FarmReport{Runs: 3, Kills: 2, Stop: "runs"}, nil
	}}
	g.run()
	defer g.finish()
	if err := g.r.Start(runsGoal("Pindleskin", 2)); err != nil {
		t.Fatal(err)
	}
	g.waitFor("two boss kills", func() bool { return len(g.events(goals.EvKill)) >= 2 })
	for _, ev := range g.events(goals.EvKill) {
		if !ev.Boss || ev.Monster != "Pindleskin" {
			t.Errorf("event %+v", ev)
		}
	}
	if call.Run != "Pindleskin" || call.Runs != 2 || call.Ops == nil {
		t.Errorf("call %+v", call)
	}
}

func TestFarmOpsUnlocksFromQuests(t *testing.T) {
	g := newRig(t)
	o := farmOps{r: g.r}
	if len(o.unlocks(2)) != 0 {
		t.Fatal("no Task: nothing unlocked")
	}
	q := &fakeQuests{}
	g.r.d.Task = &TaskDeps{Quests: q}
	if o.unlocks(2)["anya_portal"] {
		t.Fatal("Prison of Ice not done")
	}
	q.set(task.QPrison, task.QuestState{Stage: task.StageDone})
	if !o.unlocks(2)["anya_portal"] {
		t.Fatal("Prison of Ice done must unlock anya_portal")
	}
}

func TestPollQuestsEmitsOnStageChangeOnly(t *testing.T) {
	g := newRig(t)
	q := &fakeQuests{}
	g.r.d.Task = &TaskDeps{Quests: q}
	p := &d2.Player{Difficulty: d2.Nightmare}
	t0 := time.Unix(1000, 0)
	drain := func() []goals.Event {
		var out []goals.Event
		for {
			select {
			case ev := <-g.r.ch:
				out = append(out, ev)
			default:
				return out
			}
		}
	}
	q.set(task.QDen, task.QuestState{Stage: task.StageStarted})
	g.r.pollQuests(p, t0)
	if evs := drain(); len(evs) != 1 || evs[0].Quest != task.QDen || evs[0].Done || evs[0].Diff != 1 {
		t.Fatalf("first poll: %+v", evs)
	}
	q.set(task.QPrison, task.QuestState{Stage: task.StageDone})
	g.r.pollQuests(p, t0.Add(time.Second)) // rate limited
	if evs := drain(); len(evs) != 0 {
		t.Fatalf("not rate limited: %+v", evs)
	}
	g.r.pollQuests(p, t0.Add(questPollEvery))
	evs := drain()
	if len(evs) != 1 || evs[0].Quest != task.QPrison || !evs[0].Done || evs[0].Unlock != "anya_portal" {
		t.Fatalf("second poll: %+v", evs)
	}
	g.r.pollQuests(p, t0.Add(3*questPollEvery))
	if evs := drain(); len(evs) != 0 {
		t.Fatalf("unchanged quests re-emitted: %+v", evs)
	}
}

func TestLockedNeed(t *testing.T) {
	if got := lockedNeed(errors.New("boss: run not unlocked: Nihlathak: needs anya_portal")); got != "anya_portal" {
		t.Fatalf("got %q", got)
	}
	if got := lockedNeed(errors.New("boss: run not unlocked: Mephisto: needs clvl 24, have 20")); got != "" {
		t.Fatalf("clvl lock gave %q", got)
	}
}

func TestFarmRemaining(t *testing.T) {
	p := goals.Params{Count: 5, UntilMins: 10}
	if r, d := farmRemaining(p, 2, 120); r != 3 || d != 8*time.Minute {
		t.Errorf("got %d %v", r, d)
	}
	if r, d := farmRemaining(p, 9, 9999); r != 1 || d != time.Second {
		t.Errorf("exhausted must stay positive, got %d %v", r, d)
	}
	if r, d := farmRemaining(goals.Params{}, 4, 50); r != 0 || d != 0 {
		t.Errorf("unlimited must stay 0, got %d %v", r, d)
	}
}

func TestFarmResumeKeepsCountAndFirstHere(t *testing.T) {
	g := newRig(t)
	var mu sync.Mutex
	var calls []FarmCall
	g.r.d.Task = &TaskDeps{Farm: func(ctx context.Context, c FarmCall) (boss.FarmReport, error) {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, c)
		if len(calls) == 1 {
			c.OnRun("Pindleskin", true)
			c.OnRun("Pindleskin", true)
		}
		return boss.FarmReport{Stop: "interrupted"}, nil
	}}
	g.run()
	defer g.finish()
	if err := g.r.Start(runsGoal("Pindleskin", 5)); err != nil {
		t.Fatal(err)
	}
	g.waitFor("second farm call", func() bool { mu.Lock(); defer mu.Unlock(); return len(calls) >= 2 })
	mu.Lock()
	defer mu.Unlock()
	if calls[0].Runs != 5 || !calls[0].FirstHere {
		t.Errorf("first %+v", calls[0])
	}
	if calls[1].Runs != 3 || calls[1].FirstHere {
		t.Errorf("resume must ask for 3 runs and a new game: %+v", calls[1])
	}
}

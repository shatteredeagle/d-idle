package task_test

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/task"
	"d2lod-bot/internal/task/sim"
)

// H1: a world in another difficulty must not make a campaign look done.
func TestReportWrongDifficultyCountsNothing(t *testing.T) {
	var done []string
	for _, e := range task.Template("nightmare_all", false).Entries {
		done = append(done, e.Quest)
	}
	c := task.Template("nightmare_all", false)
	p := c.Report(nmWorld("normal", 80, done...), nil)
	if p.Current != 0 || p.Target == 0 {
		t.Fatalf("wrong-difficulty report: %d/%d", p.Current, p.Target)
	}
	if p := c.Report(nmWorld("nightmare", 80, done...), nil); p.Current != p.Target {
		t.Fatalf("right difficulty: %d/%d", p.Current, p.Target)
	}
}

// H3: chickens (deaths) of one quest are counted across new games.
func TestStrikesSurviveForgetFails(t *testing.T) {
	g := sim.New(true)
	now := time.Unix(1000, 0)
	ex := task.NewExecutor(g, g)
	ex.Now = func() time.Time { return now }
	p := fixedPlanner{task.Plan{Group: "boss", Nodes: []task.Node{{Step: task.Step{
		ID: "kill", Budget: time.Second,
		Need: func(w task.World) (bool, task.Evidence) { return true, task.EvMemory },
		Do:   func(ctx context.Context, a task.Actions) error { return task.Errorf(task.KindDeath, "chicken") },
	}}}}}
	var last task.Result
	for i := 0; i < 3; i++ {
		last, _ = ex.Tick(context.Background(), p)
		ex.ForgetFails() // a new game each time
		now = now.Add(5 * time.Minute)
	}
	if last.Outcome != task.OutBlocked {
		t.Fatalf("third chicken did not park the quest: %+v", last)
	}
	if _, b := ex.Blocked("boss"); !b {
		t.Fatal("parked quest forgotten by ForgetFails")
	}
	now = now.Add(2 * time.Hour)
	ex.ForgetFails()
	if _, b := ex.Blocked("boss"); b {
		t.Fatal("park did not expire")
	}
}

func TestStrikesClearOnLevelUp(t *testing.T) {
	g := sim.New(true)
	ex := task.NewExecutor(g, g)
	p := fixedPlanner{task.Plan{Group: "boss", Nodes: []task.Node{{Step: task.Step{
		ID: "kill", Budget: time.Second,
		Need: func(w task.World) (bool, task.Evidence) { return true, task.EvMemory },
		Do:   func(ctx context.Context, a task.Actions) error { return task.Errorf(task.KindDeath, "chicken") },
	}}}}}
	for i := 0; i < 3; i++ {
		ex.Tick(context.Background(), p)
		ex.ForgetFails()
	}
	if _, b := ex.Blocked("boss"); !b {
		t.Fatal("not parked")
	}
	g.W.Level++
	ex.Tick(context.Background(), fixedPlanner{}) // reads the new level
	ex.ForgetFails()
	if _, b := ex.Blocked("boss"); b {
		t.Fatal("level up did not unpark")
	}
}

// M7: a deadline on the caller's ctx (the bot's 40 min step guard) is a timeout and counts.
func TestCallerDeadlineCountsAsTimeout(t *testing.T) {
	g := sim.New(true)
	ex := task.NewExecutor(g, g)
	p := fixedPlanner{task.Plan{Group: "x", Nodes: []task.Node{{Step: task.Step{
		ID:   "slow",
		Need: func(w task.World) (bool, task.Evidence) { return true, task.EvMemory },
		Do:   func(ctx context.Context, a task.Actions) error { <-ctx.Done(); return ctx.Err() },
	}}}}}
	for i := 0; i < 2; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		r, _ := ex.Tick(ctx, p)
		cancel()
		if i == 0 && (r.Outcome != task.OutRetry || r.Kind != task.KindTimeout) {
			t.Fatalf("first: %+v", r)
		}
		if i == 1 && r.Outcome != task.OutBlocked {
			t.Fatalf("second overrun not escaped: %+v", r)
		}
	}
}

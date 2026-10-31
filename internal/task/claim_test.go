package task_test

import (
	"context"
	"testing"

	"d2lod-bot/internal/task"
	"d2lod-bot/internal/task/sim"
)

func claimWorld(stage task.Stage, mut func(*task.World)) task.World {
	w := task.World{Act: 1, Level: 12, Difficulty: "normal", Quests: map[string]task.QuestState{
		task.QDen: {Stage: task.StageDone}, task.QBurial: {Stage: stage}}}
	if mut != nil {
		mut(&w)
	}
	return w
}

func TestClaimNeeded(t *testing.T) {
	cases := []struct {
		name string
		w    task.World
		want bool
	}{
		{"blood raven dead, reward open", claimWorld(task.StageCleared, nil), true},
		{"blood raven done and rewarded", claimWorld(task.StageDone, nil), true},
		{"blood raven alive", claimWorld(task.StageStarted, nil), false},
		{"merc employed", claimWorld(task.StageDone, func(w *task.World) { w.HasMerc = true }), false},
		{"merc reader says alive", claimWorld(task.StageDone, func(w *task.World) { w.MercKnown, w.MercAlive = true, true }), false},
		{"merc reader says dead", claimWorld(task.StageDone, func(w *task.World) { w.MercKnown = true }), true},
		{"act 2", claimWorld(task.StageDone, func(w *task.World) { w.Act = 2 }), false},
		{"nightmare", claimWorld(task.StageDone, func(w *task.World) { w.Difficulty = "nightmare" }), false},
	}
	for _, c := range cases {
		if got := task.ClaimNeeded(c.w); got != c.want {
			t.Errorf("%s: ClaimNeeded = %v, want %v", c.name, got, c.want)
		}
	}
	if task.KashyaHireAllowed(claimWorld(task.StageStarted, nil)) || !task.KashyaHireAllowed(claimWorld(task.StageCleared, nil)) {
		t.Error("a paid Kashya hire is allowed only once Blood Raven is done")
	}
}

// The quest word says Blood Raven is done: the claim step is planned, behind the Burial entry's own reward talk, and
// only for a character without a merc. The plain template is untouched.
func TestClaimPlannedAfterBloodRaven(t *testing.T) {
	c := task.NormalAct1(task.Act1Options{SkipTools: true, SkipTower: true}).WithRogueClaim()
	if p := c.Plan(claimWorld(task.StageCleared, nil), nil); p.Group != task.QBurial {
		t.Errorf("cleared: plan %q, want the reward talk first", p.ID)
	}
	if p := c.Plan(claimWorld(task.StageDone, nil), nil); p.Group != task.QBurialMerc {
		t.Errorf("done: plan %q group %q, want the claim", p.ID, p.Group)
	}
	if p := c.Plan(claimWorld(task.StageDone, func(w *task.World) { w.HasMerc = true }), nil); p.Group == task.QBurialMerc {
		t.Errorf("merc employed: claim planned")
	}
	plain := task.NormalAct1(task.Act1Options{SkipTools: true, SkipTower: true})
	if p := plain.Plan(claimWorld(task.StageDone, nil), nil); p.Group == task.QBurialMerc {
		t.Errorf("plain template plans the claim")
	}
	if n := len(c.WithRogueClaim().Entries); n != len(plain.Entries)+1 {
		t.Errorf("WithRogueClaim twice: %d entries, want %d", n, len(plain.Entries)+1)
	}
	if (*task.Campaign)(nil).WithRogueClaim() != nil {
		t.Error("nil campaign")
	}
}

type claimGame struct {
	*sim.Game
	err   error
	calls int
}

func (g *claimGame) ClaimFreeMerc(context.Context) error { g.calls++; return g.err }

// A whole campaign with the claim step: a backend is called once; with none, or an unsupported one, the step is skipped
// and the campaign still ends.
func TestClaimStepSkipsSafely(t *testing.T) {
	run := func(act func(*sim.Game) task.Actions) {
		t.Helper()
		g := sim.New(true)
		c := task.NormalAct1(task.Act1Options{}).WithRogueClaim()
		ex := task.NewExecutor(g, act(g))
		for i := 0; i < 300; i++ {
			r, err := ex.Tick(context.Background(), c)
			if err != nil {
				t.Fatal(err)
			}
			if r.Outcome == task.OutDone {
				return
			}
			if r.Stuck {
				t.Fatalf("blocked: %+v log %v", r, tail(g.Log))
			}
		}
		t.Fatalf("no end: %v", tail(g.Log))
	}
	run(func(g *sim.Game) task.Actions { return g }) // no capability
	var ok *claimGame
	run(func(g *sim.Game) task.Actions { ok = &claimGame{Game: g}; return ok })
	if ok.calls != 1 {
		t.Errorf("claimer called %d times, want 1", ok.calls)
	}
	run(func(g *sim.Game) task.Actions { return &claimGame{Game: g, err: task.ErrClaimUnsupported} })
}

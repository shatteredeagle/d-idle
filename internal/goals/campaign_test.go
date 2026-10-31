package goals

import (
	"testing"
	"time"
)

type progRunner struct {
	*fakeRunner
	p Progress
}

func (r *progRunner) Progress() (Progress, bool) { return r.p, true }

func TestCampaignGoalValidatesAndDefaults(t *testing.T) {
	m, _, _ := newMgr(t, "")
	g := mustAdd(t, m, Goal{Type: Campaign})
	if g.Params.Template != CampaignNormalAct1 || g.Title != "Campaign: Normal Act 1" || g.Progress.Target != 6 {
		t.Fatalf("%+v", g)
	}
	if _, err := m.Add(Goal{Type: Campaign, Params: Params{Template: "hell_act5"}}, -1); err == nil {
		t.Fatal("unknown template accepted")
	}
	if g := mustAdd(t, m, Goal{Type: Campaign, Params: Params{Cain: true}}); g.Progress.Target != 7 {
		t.Fatalf("cain target %d", g.Progress.Target)
	}
}

// The manager merges the runner's report and ends the goal once every quest is done.
func TestCampaignProgressMergesAndCompletes(t *testing.T) {
	f := newFake()
	pr := &progRunner{fakeRunner: f}
	c := &clock{t: time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)}
	m, err := New(Options{Runner: pr, Clock: c.now})
	if err != nil {
		t.Fatal(err)
	}
	g := mustAdd(t, m, Goal{Type: Campaign})
	m.Start()
	pr.p = Progress{Current: 2, Target: 6, Percent: 33.3, Phase: "fight", Steps: []Step{{ID: "quest.den_of_evil", State: StateDone}}, Source: "mixed"}
	c.t = c.t.Add(time.Second)
	m.Tick(c.t)
	got, _ := m.Goal(g.ID)
	if got.Progress.Current != 2 || got.Progress.Phase != "fight" || len(got.Progress.Steps) != 1 || got.State != StateActive {
		t.Fatalf("merge: %+v", got)
	}
	pr.p.Current = 6
	c.t = c.t.Add(time.Second)
	m.Tick(c.t)
	if got, _ = m.Goal(g.ID); got.State != StateDone {
		t.Fatalf("not done: %+v", got)
	}
}

// LIVE (i4): a level-up event set Current to the character level and ended a 6-quest campaign at level 6.
func TestLevelUpDoesNotCompleteACampaign(t *testing.T) {
	m, _, _ := newMgr(t, "")
	g := mustAdd(t, m, Goal{Type: Campaign})
	m.Start()
	m.Handle(Event{Type: EvLevelUp, Level: 6})
	got, _ := m.Goal(g.ID)
	if got.State != StateActive || got.Progress.Current != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestNightmareTemplateSetsDifficulty(t *testing.T) {
	m, _, _ := newMgr(t, "")
	g := mustAdd(t, m, Goal{Type: Campaign, Params: Params{Template: CampaignHellAll}})
	if g.Params.Difficulty != "hell" || g.Title != "Campaign: Hell" || g.Progress.Target != CampaignQuests(CampaignNormalAll, false) {
		t.Fatalf("%+v", g)
	}
	if _, err := m.Add(Goal{Type: Campaign, Params: Params{Template: CampaignNightmareAll, Difficulty: "hell"}}, -1); err == nil {
		t.Fatal("conflicting difficulty accepted")
	}
}

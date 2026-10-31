package goals

import "testing"

func TestParseRuns(t *testing.T) {
	g, err := ParseRuns("runs:Eldritch_Shenk:5")
	if err != nil || g.Type != Runs || g.Params.Area != "Eldritch Shenk" || g.Params.Count != 5 {
		t.Fatalf("%+v %v", g, err)
	}
	if g, _ := ParseRuns("runs:Baal"); g.Params.Count != 1 {
		t.Fatalf("default count: %+v", g)
	}
	for _, bad := range []string{"runs:", "runs::3", "runs:Baal:0", "runs:Baal:x", "farm:Baal"} {
		if _, err := ParseRuns(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestRunsGoalAddsAndProgresses(t *testing.T) {
	m, _, _ := newMgr(t, "")
	g, _ := ParseRuns("runs:auto:2")
	added, err := m.Add(g, -1)
	if err != nil {
		t.Fatal(err)
	}
	if added.Title != "Run auto x2" {
		t.Fatalf("title %q", added.Title)
	}
	if added.Params.Count != 2 || added.target() != 2 {
		t.Fatalf("target %+v", added)
	}
	added.Progress.BossKills = 2
	if !added.refreshProgress() {
		t.Fatal("two runs must complete the goal")
	}
}

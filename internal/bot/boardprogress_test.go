package bot

import (
	"testing"
	"time"

	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/questboard"
	"d2lod-bot/internal/task"
)

// The campaign goal under the quest board read 0 of N for the life of a hero whose quests the board had finished
// (sandbox logs 2026-11-27, problem 7): with the board on the task executor never ticked, so nothing refreshed its report.
func TestCampaignProgressCountsQuestsUnderTheBoard(t *testing.T) {
	q := &fakeQuests{}
	q.set(task.QDen, task.QuestState{Stage: task.StageDone, Ev: task.EvSave})
	g := campaignRig(t, q, 3)
	g.mover.exploreFn = nil
	g.r.e.Tune.Quests.Board = 1
	b, err := questboard.Load()
	if err != nil {
		t.Fatal(err)
	}
	st, err := questboard.Open(b, t.TempDir(), "Testy")
	if err != nil {
		t.Fatal(err)
	}
	g.r.NewQuestBoard(b, st)
	g.run()
	defer g.finish()
	if err := g.r.Start(campaign()); err != nil {
		t.Fatal(err)
	}
	g.waitFor("progress under the board", func() bool { p, ok := g.r.Progress(); return ok && p.Current >= 1 })
	p, _ := g.r.Progress()
	if p.Steps[0].State != goals.StateDone || p.Percent <= 0 {
		t.Fatalf("progress %+v", p)
	}
	time.Sleep(10 * time.Millisecond)
}

// A quest the board shows done counts although the world's quest word is silent (a latch, a chunk the player did by hand).
func TestBoardProgressCountsBoardDoneQuests(t *testing.T) {
	for _, tmpl := range []string{goals.CampaignNormalAct1, goals.CampaignNormalAct2} {
		camp := task.Template(tmpl, false)
		w := task.World{Act: 1, InTown: true, Level: 30, Killed: map[string]bool{}, Cleared: map[task.Area]bool{}, WPs: map[task.Area]bool{}}
		rep := camp.Report(w, nil)
		if rep.Current != 0 || len(rep.Steps) == 0 {
			t.Fatalf("%s: empty world already counts: %+v", tmpl, rep)
		}
		first, _ := camp.Entries[0], 0
		got := boardProgress(rep, map[string]bool{first.Quest: true})
		if got.Current != 1 || got.Steps[0].State != goals.StateDone || got.Percent != float64(100)/float64(got.Target) {
			t.Fatalf("%s: %+v", tmpl, got)
		}
		if rep.Steps[0].State == goals.StateDone {
			t.Fatalf("%s: the input report was changed", tmpl)
		}
		if again := boardProgress(got, map[string]bool{first.Quest: true}); again.Current != 1 {
			t.Fatalf("%s: counted twice: %+v", tmpl, again)
		}
	}
}

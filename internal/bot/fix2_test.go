package bot

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/questboard"
)

// smBoardRig is boardRig on the run machine with extra knobs set before the board is built.
func smBoardRig(t *testing.T, class d2.Class, mutate func(*env.Tuning)) (*rig, *questboard.Exec, *questboard.Store) {
	t.Helper()
	g := campaignRig(t, &fakeQuests{}, 3)
	g.mover.exploreFn = func(ctx context.Context, done func() bool) error { return nil }
	g.w.set(func(w *world) { w.class = class })
	g.r.e.Tune.Quests.Board, g.r.e.Tune.Quests.BoardSM = 1, 1
	if mutate != nil {
		mutate(g.r.e.Tune)
	}
	b, err := questboard.Load()
	if err != nil {
		t.Fatal(err)
	}
	st, err := questboard.Open(b, t.TempDir(), "Testy")
	if err != nil {
		t.Fatal(err)
	}
	return g, g.r.NewQuestBoard(b, st), st
}

// Fable review 2 finding 3 (card fix2): the pause policy lifts a died_too_often park under every goal kind, as the page
// says ("Resumes by itself in about N min"). Before: ResumeTick ran only inside boardStep, which a level, farm, idle or
// runs goal never reaches while the head run is parked.
func TestQuestResumeLiftsParkUnderEveryGoalKind(t *testing.T) {
	kinds := []goals.Goal{
		farm("Blood Moor"),
		{ID: "g2", Type: goals.ReachLevel, Params: goals.Params{Level: 30}},
		{ID: "g3", Type: goals.Idle},
	}
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		for _, goal := range kinds {
			t.Run(class.String()+"/"+string(goal.Type), func(t *testing.T) {
				g, x, st := smBoardRig(t, class, func(tu *env.Tuning) { tu.Quests.ResumeAfterDeaths = 1 })
				parked := make(chan struct{})
				g.mover.exploreFn = func(ctx context.Context, done func() bool) error { // the goal's own work holds until the park is stored
					select {
					case <-parked:
					case <-ctx.Done():
					case <-time.After(30 * time.Second):
					}
					return nil
				}
				g.run()
				defer g.finish()
				if err := g.r.Start(goal); err != nil {
					t.Fatal(err)
				}
				g.waitFor("the goal runs", func() bool { return g.w.snaps.Load() > 3 })
				if err := x.Accept("den_of_evil", "clear_den"); err != nil {
					t.Fatal(err)
				}
				d := questboard.Decision{Quest: "den_of_evil", Chunk: "clear_den"}
				x.Report(d, questboard.StepResult{Outcome: questboard.StepDied, Why: "Corpsefire"})
				x.Report(d, questboard.StepResult{Outcome: questboard.StepDied, Why: "Corpsefire"})
				if !boardParked(t, st) {
					t.Fatal("not parked")
				}
				g.clock.Sleep(46 * time.Minute) // the quiet stretch of data/questboard/resume.json
				close(parked)
				g.waitFor("the pause policy lifts the park", func() bool {
					plan, _, err := st.Load()
					if err != nil {
						return false
					}
					for _, ev := range plan.AutoLog {
						if ev.Kind == questboard.EvAutoResume {
							return true
						}
					}
					return false
				})
			})
		}
	}
}

func boardParked(t *testing.T, st *questboard.Store) bool {
	t.Helper()
	plan, _, err := st.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range plan.Runs {
		if r.Status == questboard.RunParked {
			return true
		}
	}
	return false
}

// Fable review 2 finding 5 (card fix2): quests.boardSM edited at run time reaches the executor, which was built with the
// knob's value at start; the guards read it live, so the two must not disagree.
func TestBoardExecFollowsBoardSMKnobLive(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		t.Run(class.String(), func(t *testing.T) {
			g, x := boardRig(t, true)
			g.w.set(func(w *world) { w.class = class })
			if x.SM() {
				t.Fatal("built with quests.boardSM 0 but on the machine")
			}
			g.r.e.Tune.Quests.BoardSM = 1
			g.r.boardRTFor()
			if !x.SM() {
				t.Fatal("quests.boardSM 1 at run time: the executor stayed on the legacy board")
			}
			g.r.e.Tune.Quests.BoardSM = 0
			g.r.boardRTFor()
			if x.SM() {
				t.Fatal("quests.boardSM 0 at run time: the executor stayed on the machine")
			}
		})
	}
}

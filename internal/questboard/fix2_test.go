package questboard

import (
	"strings"
	"testing"
)

// Fable review 2 finding 1 (card fix2): what leads to a park is stored at every count, so a restart between two deaths
// (self-update, crash, a flip window) keeps counting. Before: the counters lived in runState only and the hero died on
// the chunk once per process lifetime, for ever.
func TestSMCountersSurviveRestart(t *testing.T) {
	for _, class := range smClasses {
		t.Run(class+"/deaths", func(t *testing.T) {
			r := newRigSM(t, class, 6, true)
			if err := r.x.Accept(den, clr); err != nil {
				t.Fatal(err)
			}
			died := StepResult{StepDied, "killed by Corpsefire", ""}
			r.f.Script = []StepResult{died}
			r.tick(1)
			r.restart()
			if d := r.next(); d.Kind != DecRun {
				t.Fatalf("one death must not park: %+v", d)
			}
			r.f.Script = []StepResult{died}
			r.tick(1)
			if d := r.next(); d.Kind != DecBlocked || !strings.Contains(d.Why, "died 2 times") {
				t.Fatalf("second death after a restart must park: %+v", d)
			}
			r.restart()
			if d := r.next(); d.Kind != DecBlocked {
				t.Fatalf("park lost: %+v", d)
			}
			r.x.Unblock(den, clr) // Try again starts the count afresh
			r.f.Script = []StepResult{died}
			r.tick(1)
			if d := r.next(); d.Kind != DecRun {
				t.Fatalf("Try again must clear the stored counters: %+v", d)
			}
		})
		t.Run(class+"/failures", func(t *testing.T) {
			r := newRigSM(t, class, 6, true)
			if err := r.x.Accept(den, clr); err != nil {
				t.Fatal(err)
			}
			fail := StepResult{StepFailed, "no path to the Den", ""}
			r.f.Script = []StepResult{fail, fail}
			r.tick(2)
			r.restart()
			r.f.Script = []StepResult{fail}
			r.tick(1)
			if d := r.next(); d.Kind != DecBlocked {
				t.Fatalf("third failure in a row after a restart must park: %+v", d)
			}
		})
		t.Run(class+"/a good slice clears the stored failures", func(t *testing.T) {
			r := newRigSM(t, class, 6, true)
			if err := r.x.Accept(den, clr); err != nil {
				t.Fatal(err)
			}
			fail := StepResult{StepFailed, "no path to the Den", ""}
			r.f.Script = []StepResult{fail, fail, {StepRan, "", ""}}
			r.tick(3)
			r.restart()
			r.f.Script = []StepResult{fail, fail}
			r.tick(2)
			if d := r.next(); d.Kind != DecRun {
				t.Fatalf("two failures after a good slice must not park: %+v", d)
			}
		})
	}
}

// Finding 1, scenario B: a park the machine stored is honoured by the legacy board (quests.boardSM back at 0 after a
// revert), so the safety revert never restarts the chunk that killed the hero. Try again lifts it there too.
func TestLegacyBoardHonoursStoredPark(t *testing.T) {
	for _, class := range smClasses {
		t.Run(class, func(t *testing.T) {
			r := newRigSM(t, class, 6, true)
			if err := r.x.Accept(den, clr); err != nil {
				t.Fatal(err)
			}
			died := StepResult{StepDied, "killed by Corpsefire", ""}
			r.f.Script = []StepResult{died, died}
			r.tick(2)
			if d := r.next(); d.Kind != DecBlocked {
				t.Fatalf("not parked: %+v", d)
			}
			r.old = true
			r.restart()
			if !r.x.boardLegacy() {
				t.Fatal("rig is not on the legacy board")
			}
			d := r.next()
			if d.Kind != DecBlocked || !strings.Contains(d.Why, "2") {
				t.Fatalf("legacy board restarted a parked chunk: %+v", d)
			}
			if n := r.f.Count("step "); n != 2 {
				t.Fatalf("steps = %d, want the 2 that died", n)
			}
			r.x.Unblock(den, clr)
			if d := r.next(); d.Kind != DecRun {
				t.Fatalf("Try again on the legacy board must lift the stored park: %+v", d)
			}
			r.old = false
			r.restart()
			if d := r.next(); d.Kind != DecRun {
				t.Fatalf("the park came back with the machine: %+v", d)
			}
		})
	}
}

// Finding 7: switching quests.board off drops the in-memory state only; a stored park stays until the player lifts it.
func TestBoardOffKeepsStoredParks(t *testing.T) {
	for _, class := range smClasses {
		t.Run(class, func(t *testing.T) {
			r := parkRig(t, class, true)
			r.x.SetEnabled(false)
			r.x.SetEnabled(true)
			if d := r.next(); d.Kind != DecBlocked {
				t.Fatalf("the knob switch lifted the park: %+v", d)
			}
			r.x.Reset() // the player's Reset still clears it
			if d := r.next(); d.Kind != DecRun {
				t.Fatalf("Reset must clear the park: %+v", d)
			}
		})
	}
}

// Finding 5: the executor follows quests.boardSM at run time (SetSM), like the guards that read the knob live.
func TestSetSMFollowsTheKnob(t *testing.T) {
	for _, class := range smClasses {
		t.Run(class, func(t *testing.T) {
			r := &rig{t: t, b: mustBoard(t), dir: t.TempDir(), hero: "Testy", old: true}
			r.f = NewFake(r.b, class, 6)
			r.restart()
			r.x.SetSM(true)
			if !r.x.SM() {
				t.Fatal("SetSM(true) did not switch the executor")
			}
			if err := r.x.Accept(den, clr); err != nil {
				t.Fatal(err)
			}
			fail := StepResult{StepFailed, "no path to the Den", ""}
			r.f.Script = []StepResult{fail, fail, fail}
			r.tick(3)
			r.old, r.sm = false, true
			r.restart()
			if d := r.next(); d.Kind != DecBlocked {
				t.Fatalf("the park of a machine switched on at run time was not stored: %+v", d)
			}
			r.x.SetSM(false)
			if r.x.SM() || !r.x.boardLegacy() {
				t.Fatal("SetSM(false) did not switch back")
			}
		})
	}
}

// Finding 6: the park codes that were gave_up before cellar3gate and refusalpark keep the one retry a new release gives.
func TestSMReleaseRetriesNoPathAndRefusedOnce(t *testing.T) {
	for _, class := range smClasses {
		for _, code := range []string{"no_path", "refused_repeatedly"} {
			t.Run(class+"/"+code, func(t *testing.T) {
				r := releaseParkRig(t, class)
				if err := r.x.st.UpdateSM(nil, func(m *SM) error {
					m.find(den, clr, "normal").Code = code
					return nil
				}); err != nil {
					t.Fatal(err)
				}
				r.restartRelease("A")
				if d := r.next(); d.Kind != DecBlocked {
					t.Fatalf("same release must stay parked: %+v", d)
				}
				r.restartRelease("B")
				if d := r.next(); d.Kind != DecRun {
					t.Fatalf("release B must retry the %s park once: %+v", code, d)
				}
				r.restartRelease("B")
				m, _ := r.x.st.LoadSM(nil)
				if run := m.find(den, clr, "normal"); run == nil || run.RetriesAfterRelease != 1 {
					t.Fatalf("history: %+v", run)
				}
			})
		}
	}
}

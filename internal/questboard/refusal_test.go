package questboard

import (
	"strings"
	"testing"
	"time"
)

// The refusal park (card refusalpark, knob quests.refusalPark). Every test runs for two classes.

func refusalRig(t *testing.T, class string, n int) *rig {
	t.Helper()
	r := newRigSM(t, class, 6, true)
	r.x.SetRefusal(n)
	if err := r.x.Accept(den, clr); err != nil {
		t.Fatal(err)
	}
	return r
}

func refusal(code string) StepResult {
	return StepResult{StepFailed, "refused: " + code + " text", code}
}

func TestRefusalPolicyFile(t *testing.T) {
	p, err := LoadRefusal()
	if err != nil || p.ParkAfter != 3 {
		t.Fatalf("embedded policy: %+v %v", p, err)
	}
	for _, bad := range []string{`{"schema":"x","parkAfter":3}`, `{"schema":"quests.refusal/1","parkAfter":1}`, `not json`} {
		if _, err := ParseRefusal([]byte(bad)); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
}

func TestRefusalIdenticalCodedRefusalsPark(t *testing.T) {
	for _, class := range smClasses {
		t.Run(class, func(t *testing.T) {
			r := refusalRig(t, class, 3)
			c := refusal("area_ban")
			r.f.Script = []StepResult{c, c}
			r.tick(2)
			if d := r.next(); d.Kind != DecRun {
				t.Fatalf("parked after two: %+v", d)
			}
			r.f.Script = []StepResult{c}
			r.tick(1)
			if d := r.next(); d.Kind != DecBlocked {
				t.Fatalf("not parked after three: %+v", d)
			}
			run := r.x.runOf(t, den, clr)
			if run.Code != "refused_repeatedly" || run.RefusalCode != "area_ban" || !strings.Contains(run.RefusalWhy, "text") {
				t.Fatalf("stored park: %+v", run)
			}
			v := r.desk(nil).viewFresh()
			if v.Blocked == nil || v.Blocked.Reason == nil || v.Blocked.Reason.Code != "refused_repeatedly" ||
				!strings.Contains(v.Blocked.Reason.Text, "area_ban") || !strings.Contains(v.Blocked.Reason.Text, "text") {
				t.Fatalf("reason: %+v", v.Blocked)
			}
			if l := r.desk(nil).ParkLines(); len(l) != 1 || !strings.Contains(l[0], "area_ban") {
				t.Fatalf("return screen lines: %q", l)
			}
		})
	}
}

func TestRefusalMixedCodesDoNotPark(t *testing.T) {
	for _, class := range smClasses {
		t.Run(class, func(t *testing.T) {
			r := refusalRig(t, class, 3)
			r.f.Script = []StepResult{refusal("a"), refusal("b"), refusal("a")}
			r.tick(3)
			// the plain failure counter still ends it, as gave_up
			if run := r.x.runOf(t, den, clr); run.Code != "gave_up" {
				t.Fatalf("want gave_up for a mix, got %+v", run)
			}
		})
	}
}

func TestRefusalUncodedAndKnobOffKeepToday(t *testing.T) {
	for _, class := range smClasses {
		for _, n := range []int{0, 3} {
			t.Run(class, func(t *testing.T) {
				r := refusalRig(t, class, n)
				c := refusal("area_ban")
				if n == 3 {
					c.Code = "" // uncoded
				}
				r.f.Script = []StepResult{c, c, c}
				r.tick(3)
				if run := r.x.runOf(t, den, clr); run.Code != "gave_up" || run.RefusalCode != "" {
					t.Fatalf("n=%d: want gave_up, got %+v", n, run)
				}
			})
		}
	}
}

// Card cellar3gate: a step that fails three times with a navigation refusal that names its cause parks as no_path (the exit
// and the cause stay in the text); the same count of plain failures stays gave_up.
func TestNoPathParkNamesTheExitAndCause(t *testing.T) {
	for _, class := range smClasses {
		t.Run(class, func(t *testing.T) {
			r := refusalRig(t, class, 0)
			why := "exit to Tower Cellar Level 4: nav: no path: gate to level 24 is on the map but not reachable from {45 20} (cause=cut_off, no way out found)"
			c := StepResult{Outcome: StepFailed, Why: why}
			now := time.Unix(1_800_000_000, 0)
			r.x.o.Now = func() time.Time { return now }
			r.f.Script = []StepResult{c, c, c, c, c, c}
			// L3 (live review 2026-11-28): three identical refusals from one cell inside a minute are one try, not a park
			r.tick(3)
			if v := r.chunk(den, clr); v.State == Blocked {
				t.Fatalf("three retries from one cell in the same minute parked the chunk: %+v", v)
			}
			for i := 0; i < 2; i++ { // the same refusal a minute later is a new try
				now = now.Add(samePathWindow + time.Second)
				r.tick(1)
			}
			if run := r.x.runOf(t, den, clr); run.Code != "no_path" {
				t.Fatalf("stored park: %+v", run)
			}
			v := r.desk(nil).viewFresh()
			if v.Blocked == nil || v.Blocked.Reason == nil || v.Blocked.Reason.Code != "no_path" ||
				!strings.Contains(v.Blocked.Reason.Text, "Tower Cellar Level 4") || !strings.Contains(v.Blocked.Reason.Text, "cause=cut_off") {
				t.Fatalf("reason: %+v", v.Blocked.Reason)
			}
		})
	}
}

// TASK-1120 (lab 2026-11-28, board.sm.restart-mid-chain): a chunk started while the hero stood in a side level with no
// known exit failed its travel step three times in 0.9 s ("nav: no exit to that level", the third report prefixed by the
// task layer) and parked gave_up. TASK-1126: the same with an NPC that is not in the walkable part of the level. The world
// refused, not the hero: inside samePathWindow it is one try; a minute apart each counts. Must fail without sameTryKind.
func TestWorldRefusalsInOneSecondAreOneTry(t *testing.T) {
	for _, base := range []string{
		"exit to Black Marsh: nav: no exit to that level: level 6 from 20",
		"step talk_tyrael: 2 tries used: npc_out_of_reach: npc class 251 is not in the part of area 73 the hero can walk",
	} {
		for _, class := range smClasses {
			t.Run(class+"/"+base[:12], func(t *testing.T) {
				r := refusalRig(t, class, 0)
				c := StepResult{Outcome: StepFailed, Why: base}
				third := StepResult{Outcome: StepFailed, Why: "needs_attention: quest.x/step: " + base}
				now := time.Unix(1_800_000_000, 0)
				r.x.o.Now = func() time.Time { return now }
				r.f.Script = []StepResult{c, c, third, c, c}
				r.tick(3)
				if v := r.chunk(den, clr); v.State == Blocked {
					t.Fatalf("three identical refusals at one instant parked the chunk: %+v", v)
				}
				for i := 0; i < 2; i++ {
					now = now.Add(samePathWindow + time.Second)
					r.tick(1)
				}
				if run := r.x.runOf(t, den, clr); run.Code != "gave_up" {
					t.Fatalf("three tries a minute apart did not park: %+v", run)
				}
			})
		}
	}
}

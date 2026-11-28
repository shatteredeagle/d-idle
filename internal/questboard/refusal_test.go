package questboard

import (
	"strings"
	"testing"
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
				!strings.Contains(v.Blocked.Reason.Text, "area_ban") || !strings.Contains(v.Blocked.Reason.Text, "text") || strings.Contains(v.Blocked.Reason.Text, "{") {
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

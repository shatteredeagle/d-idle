package drill

import (
	"context"
	"testing"
)

// RT.leave after the wiring: the hook exits and rejoins inside Act, so the lab reports LeftGame at tick 0. The PASS stays
// numeric: effect inside the bound, life floor above 0.
func TestRetreatLeaveRejoinedPasses(t *testing.T) {
	st := &retreatStub{info: RetreatInfo{Adjacent: 1, Decision: "chicken", Acted: true, LifeFloor: 30, TicksToEffect: 0, LeftGame: true}}
	var got Outcome
	for _, d := range RT(st) {
		if d.ID == "RT.leave" {
			got = d.Do(context.Background(), nil)
		}
	}
	if got.Verdict != Pass {
		t.Fatalf("%v: %s", got.Verdict, got.Reason)
	}
	// the unwired rig still reports Blocked, not a pass
	st.info = RetreatInfo{Err: "leave hook not wired"}
	for _, d := range RT(st) {
		if d.ID == "RT.leave" {
			if o := d.Do(context.Background(), nil); o.Verdict != Blocked {
				t.Fatalf("unwired: %v", o.Verdict)
			}
		}
	}
	// a hook that ran but a game that never left stays a FAIL
	st.info = RetreatInfo{Adjacent: 1, Decision: "chicken", Acted: true, LifeFloor: 30, TicksToEffect: 0}
	for _, d := range RT(st) {
		if d.ID == "RT.leave" {
			if o := d.Do(context.Background(), nil); o.Verdict != Fail {
				t.Fatalf("not left: %v", o.Verdict)
			}
		}
	}
}

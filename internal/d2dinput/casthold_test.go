package d2dinput

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dctl/fake"
	"d2lod-bot/internal/input"
)

func holdRig(t *testing.T, status []string, refuse string) (*Controller, *fake.Server) {
	fix := fake.Sample()
	fix.Hold, fix.HoldErr = status, refuse
	return rig(t, fix)
}

func TestCastHoldLinesAndEnd(t *testing.T) {
	at := input.Point{X: 65, Y: 147} // cell 13.1, 29.5
	cases := []struct {
		name string
		left bool
		unit uint32
		d    time.Duration
		want string
	}{
		{"right hand at a unit", false, 5, 600 * time.Millisecond, "cmd skillhold 7 13.1 29.5 5 15 0"},
		{"left hand at the point", true, 0, 610 * time.Millisecond, "cmd skillhold 7 13.1 29.5 -1 16 1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, srv := holdRig(t, []string{"held=4 casts=1 frames=15 state=running", "held=15 casts=1 frames=15 state=done"}, "")
			if err := c.CastHold(7, tc.left, at, tc.unit, tc.d); err != nil {
				t.Fatal(err)
			}
			want := []string{tc.want, "cmd skillhold status", "cmd skillhold status"}
			if got := srv.Lines(); !reflect.DeepEqual(got, want) {
				t.Fatalf("lines\n got %q\nwant %q", got, want)
			}
			if r := c.LastHold(); r.State != "done" || r.Held != 15 || r.Casts != 1 {
				t.Fatalf("last hold %+v", r)
			}
		})
	}
}

func TestCastHoldRefusedStartIsAnError(t *testing.T) {
	c, srv := holdRig(t, []string{"state=done"}, "a hold is running")
	err := c.CastHold(7, false, input.Point{X: 65, Y: 147}, 0, time.Second)
	var ce *d2dctl.CmdError
	if !errors.As(err, &ce) {
		t.Fatalf("err %v, want the refusal", err)
	}
	if n := len(srv.Lines()); n != 1 {
		t.Fatalf("%d lines after a refused start, want 1 (no status poll)", n)
	}
}

func TestCastHoldUnknownVerbIsAnError(t *testing.T) {
	c, _ := holdRig(t, nil, "")
	if err := c.CastHold(7, false, input.Point{X: 65, Y: 147}, 0, time.Second); err == nil {
		t.Fatal("a build without the verb must refuse, so the caller's legacy cast acts")
	}
}

// Once the fork accepted the hold the cast is sent: a hold that never reports its end is bounded and is no error.
func TestCastHoldStuckRunningIsBoundedAndNoError(t *testing.T) {
	c, srv := holdRig(t, []string{"held=0 casts=0 frames=5 state=running"}, "")
	if err := c.CastHold(7, false, input.Point{X: 65, Y: 147}, 0, 200*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	polls := strings.Count(strings.Join(srv.Lines(), "\n"), "status")
	// 5 frames (200 ms) plus the 500 ms margin at one poll per 100 ms.
	if polls < 2 || polls > 9 {
		t.Fatalf("%d status polls, want a bounded handful", polls)
	}
	if r := c.LastHold(); r.State != "running" {
		t.Fatalf("last hold %+v, want state running", r)
	}
}

func TestHoldFrames(t *testing.T) {
	for d, want := range map[time.Duration]int{0: 1, time.Millisecond: 1, 40 * time.Millisecond: 1, 41 * time.Millisecond: 2, 600 * time.Millisecond: 15, time.Hour: 2500} {
		if got := HoldFrames(d); got != want {
			t.Errorf("HoldFrames(%v) = %d, want %d", d, got, want)
		}
	}
}

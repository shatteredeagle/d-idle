package d2dinput

import (
	"reflect"
	"testing"

	"d2lod-bot/internal/d2dctl/fake"
	"d2lod-bot/internal/input"
)

func TestRunPolicyHysteresis(t *testing.T) {
	p := DefaultRunPolicy()
	cases := []struct {
		cur     int
		running bool
		want    bool
	}{
		{100, false, true}, {25, false, true}, {24, false, false}, // resume at On
		{100, true, true}, {9, true, true}, {7, true, false}, {0, true, false}, // stop below Off
		{15, false, false}, {15, true, true}, // between the thresholds the state holds
	}
	for _, c := range cases {
		if got := p.want(c.cur, 100, c.running); got != c.want {
			t.Errorf("want(%d/100, running=%v) = %v, want %v", c.cur, c.running, got, c.want)
		}
	}
	if !p.want(0, 0, true) {
		t.Error("an unknown max must leave the state alone")
	}
}

func TestMoveTogglesRunOnce(t *testing.T) {
	c, srv := rig(t, fake.Sample())
	c.Run = DefaultRunPolicy()
	w := input.Point{X: 175, Y: 70}
	for i := 0; i < 2; i++ {
		if err := c.Move(w); err != nil {
			t.Fatal(err)
		}
	}
	var keys, moves int
	for _, l := range srv.Lines() {
		switch l {
		case "key r":
			keys++
		case "cmd move 35.1 14.1":
			moves++
		}
	}
	// Full stamina, belief "walking": one toggle on the first move, none on the second.
	if keys != 1 || moves != 2 {
		t.Fatalf("keys=%d moves=%d, lines %q", keys, moves, srv.Lines())
	}
}

func TestMoveWithoutPolicySendsNoKey(t *testing.T) {
	c, srv := rig(t, fake.Sample())
	if err := c.Move(input.Point{X: 175, Y: 70}); err != nil {
		t.Fatal(err)
	}
	want := []string{"debug level", "click 400 300", "cmd move 35.1 14.1"}
	if got := srv.Lines(); !reflect.DeepEqual(got, want) {
		t.Fatalf("lines %q, want %q", got, want)
	}
}

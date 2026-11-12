package d2dinput

import (
	"errors"
	"reflect"
	"testing"
	"time"

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
	want := []string{"debug level", "click 494 340", "cmd move 35.1 14.1"}
	if got := srv.Lines(); !reflect.DeepEqual(got, want) {
		t.Fatalf("lines %q, want %q", got, want)
	}
}

func countLines(srv *fake.Server, want string) int {
	n := 0
	for _, l := range srv.Lines() {
		if l == want {
			n++
		}
	}
	return n
}

// B1: a fresh game starts walking, so ResetRun must make the next move press run exactly once more, and the
// moves after that none.
func TestResetRunPressesRunAgainOnce(t *testing.T) {
	c, srv := rig(t, fake.Sample())
	c.Run = DefaultRunPolicy()
	w := input.Point{X: 175, Y: 70}
	move := func(n int) {
		for i := 0; i < n; i++ {
			if err := c.Move(w); err != nil {
				t.Fatal(err)
			}
		}
	}
	move(3)
	if k := countLines(srv, "key r"); k != 1 {
		t.Fatalf("before the reset: %d run keys, want 1", k)
	}
	c.ResetRun() // a new game
	if c.RunInfo().Believed {
		t.Fatal("belief survived the reset")
	}
	move(3)
	if k := countLines(srv, "key r"); k != 2 {
		t.Fatalf("after the reset: %d run keys, want 2", k)
	}
	if i := c.RunInfo(); !i.Believed || i.Presses != 2 || i.Resets != 1 || i.Max == 0 {
		t.Fatalf("RunInfo %+v", i)
	}
}

func TestRunLogKV(t *testing.T) {
	c, _ := rig(t, fake.Sample())
	c.Run = DefaultRunPolicy()
	if err := c.Move(input.Point{X: 175, Y: 70}); err != nil {
		t.Fatal(err)
	}
	kv := c.LogKV()
	got := map[string]any{}
	for i := 0; i+1 < len(kv); i += 2 {
		got[kv[i].(string)] = kv[i+1]
	}
	if got["runBelieved"] != true || got["runKeys"] != 1 || got["stam"] != 15360 || got["stamMax"] != 15360 {
		t.Fatalf("log kv %v", got)
	}
}

// B3: the 400 ms stall wait is virtual time on the controller's clock (never a wall sleep). A player that never moves
// costs three stall windows (kick after the first, give up at the third), each 400 ms plus at most one poll.
func TestMoveToStallWaitIsClockTime(t *testing.T) {
	c, _ := rig(t, fake.Sample())
	clk := c.Clock.(*fclock)
	start := clk.t
	if err := c.MoveTo(input.Point{X: 253, Y: 68}, input.MoveOpts{Timeout: time.Minute}); !errors.Is(err, input.ErrMoveFailed) {
		t.Fatalf("%v", err)
	}
	if got := clk.t.Sub(start); got < 3*400*time.Millisecond || got > 3*(400*time.Millisecond+c.poll()) {
		t.Fatalf("stall cost %v of clock time", got)
	}
}

// WALKSMOOTH: the belief says running, but stamina never drains during a long walk outside town: the character is
// walking (the toggle started on). One extra key press fixes it, never more than driftMaxHeals, never in town.
func TestRunBeliefHealsWhenStaminaNeverDrains(t *testing.T) {
	for _, tc := range []struct {
		name  string
		level string
		keys  int
	}{
		{"field", "2 30.70 13.70 57 41 864 912", 1 + driftMaxHeals}, // first press + the corrections
		{"town", "1 30.70 13.70 57 41 864 912", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fix := fake.Sample()
			fix.Level = tc.level
			c, srv := rig(t, fix)
			c.Run = DefaultRunPolicy()
			clk := c.Clock.(*fclock)
			for i := 0; i < 80; i++ { // 80 moves, 300 ms apart: 24 s of continuous walking at full stamina
				if err := c.Move(input.Point{X: 175 + i, Y: 70}); err != nil {
					t.Fatal(err)
				}
				clk.Sleep(300 * time.Millisecond)
			}
			if k := countLines(srv, "key r"); k != tc.keys {
				t.Fatalf("%d run keys, want %d", k, tc.keys)
			}
		})
	}
}

// A walk with gaps (fights between moves) at full stamina is not evidence of anything.
func TestRunBeliefHealIgnoresSeparateMoves(t *testing.T) {
	fix := fake.Sample()
	fix.Level = "2 30.70 13.70 57 41 864 912"
	c, srv := rig(t, fix)
	c.Run = DefaultRunPolicy()
	clk := c.Clock.(*fclock)
	for i := 0; i < 40; i++ {
		if err := c.Move(input.Point{X: 175 + i, Y: 70}); err != nil {
			t.Fatal(err)
		}
		clk.Sleep(3 * time.Second)
	}
	if k := countLines(srv, "key r"); k != 1 {
		t.Fatalf("%d run keys, want 1", k)
	}
}

// Exact run state (fid-stamina.md Q2/Q4): a fall is running, a walking-rate rise is walking, nothing proves nothing.
func TestClassifyStep(t *testing.T) {
	const max = 36352 // walk +71 per tick, stand +142
	cases := []struct {
		name            string
		prev, cur, tick int
		want            stepKind
	}{
		{"run light armor, 100 ticks", 30000, 26000, 100, stepRun},
		{"run heavy armor", 30000, 22000, 100, stepRun},
		{"assassin run", 30000, 27000, 100, stepRun},
		{"walk regen", 10000, 10000 + 71*25, 25, stepWalk},
		{"stand regen is not a walk", 10000, 10000 + 142*25, 25, stepUnknown},
		{"at the cap", max, max, 25, stepUnknown},
		{"stuck at zero", 0, 0, 25, stepUnknown},
		{"no time", 100, 50, 0, stepUnknown},
	}
	for _, c := range cases {
		if got := classifyStep(c.prev, c.cur, max, c.tick); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}

func TestStamTrackObserve(t *testing.T) {
	var s stamTrack
	t0 := time.Unix(1000, 0)
	if k := s.observe(t0, 30000, 36352, false); k != stepUnknown {
		t.Fatalf("first sample: %v", k)
	}
	if k := s.observe(t0.Add(time.Second), 26000, 36352, false); k != stepRun {
		t.Fatalf("drain: %v", k)
	}
	if k := s.observe(t0.Add(3*time.Second), 26000+71*50, 36352, false); k != stepUnknown {
		t.Fatalf("a gap over driftChain must not classify: %v", k)
	}
	if k := s.observe(t0.Add(4*time.Second), 26000+71*50+71*25, 36352, false); k != stepWalk {
		t.Fatalf("walk regen: %v", k)
	}
	if k := s.observe(t0.Add(5*time.Second), 1, 36352, true); k != stepUnknown {
		t.Fatalf("town: %v", k)
	}
}

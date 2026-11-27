package input

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

type castRec struct {
	got  []string
	fail error
}

func (r *castRec) CastUnit(skill int, left bool, at Point, unit uint32) error {
	r.got = append(r.got, fmt.Sprintf("unit %d %v %d", skill, left, unit))
	return r.fail
}

func (r *castRec) CastPoint(skill int, left bool, at Point) error {
	r.got = append(r.got, fmt.Sprintf("point %d %v %d,%d", skill, left, at.X, at.Y))
	return r.fail
}

type holdRec struct{ castRec }

func (r *holdRec) CastHold(skill int, left bool, at Point, unit uint32, d time.Duration) error {
	r.got = append(r.got, fmt.Sprintf("hold %d %v %s", skill, left, d))
	return r.fail
}

func TestCastCapAndSend(t *testing.T) {
	cases := []struct {
		c    Cast
		cap  string
		want string
	}{
		{Cast{Skill: 7, At: Point{X: 3, Y: 4}}, CapCastPoint, "point 7 false 3,4"},
		{Cast{Skill: 7, Left: true, Unit: 9}, CapCastUnit, "unit 7 true 9"},
		{Cast{Skill: 7, Left: true, Unit: 9, Hold: time.Second}, CapCastHold, "hold 7 true 1s"},
	}
	for _, tc := range cases {
		r := &holdRec{}
		if tc.c.Cap() != tc.cap {
			t.Fatalf("%+v cap %s", tc.c, tc.c.Cap())
		}
		if err := tc.c.Send(r); err != nil || len(r.got) != 1 || r.got[0] != tc.want {
			t.Fatalf("%+v: %v %q", tc.c, err, r.got)
		}
	}
	if err := (Cast{Hold: time.Second}).Send(&castRec{}); !errors.Is(err, ErrNoCastVerb) {
		t.Fatalf("hold without a hold verb: %v", err)
	}
	if err := (Cast{}).Send(struct{}{}); !errors.Is(err, ErrNoCastVerb) {
		t.Fatalf("no verbs: %v", err)
	}
}

func TestRunCastNeverActsTwice(t *testing.T) {
	on := func() bool { return true }
	caps := &BackendCaps{Verbs: []string{CapCastPoint, CapCastUnit}}
	c := Cast{Skill: 5, At: Point{X: 1, Y: 2}}
	legacy := 0
	leg := func() error { legacy++; return nil }

	// Knob off: legacy only, counted as the legacy path.
	r := &castRec{}
	off := &Planner{Caps: caps}
	if path, err := off.RunCast(c, r, leg, PathKey, nil); err != nil || path != PathKey || legacy != 1 || len(r.got) != 0 {
		t.Fatalf("off: %v %v legacy=%d verbs=%q", path, err, legacy, r.got)
	}
	// On, but the build does not list the cap: legacy.
	legacy = 0
	if path, _ := (&Planner{Enabled: on}).RunCast(c, r, leg, PathKey, nil); path != PathKey || legacy != 1 || len(r.got) != 0 {
		t.Fatalf("no cap: %v legacy=%d", path, legacy)
	}
	// On with the cap: the verb alone.
	legacy = 0
	p := &Planner{Enabled: on, Caps: caps}
	if path, err := p.RunCast(c, r, leg, PathKey, nil); err != nil || path != PathVerb || legacy != 0 || len(r.got) != 1 {
		t.Fatalf("verb: %v %v legacy=%d verbs=%q", path, err, legacy, r.got)
	}
	// The verb was sent but the expectation failed: no second cast by the legacy path.
	if _, err := p.RunCast(c, r, leg, PathKey, func() (bool, error) { return false, nil }); err == nil || legacy != 0 {
		t.Fatalf("sent verb without effect must not fall back: %v legacy=%d", err, legacy)
	}
	// The verb was not sent (error reply): the legacy path acts once.
	r.fail = errors.New("err unknown verb")
	if path, err := p.RunCast(c, r, leg, PathKey, nil); err != nil || path != PathKey || legacy != 1 {
		t.Fatalf("unsent verb falls back: %v %v legacy=%d", path, err, legacy)
	}
	if v, k, _, _ := off.Counts(); v+k != 0 {
		t.Fatal("knob off counts nothing")
	}
	if v, k, _, f := p.Counts(); v != 1 || k != 1 || f != 1 {
		t.Fatalf("counts %d %d %d", v, k, f)
	}
}

func TestPlannerOnceStopsAfterSent(t *testing.T) {
	keys := 0
	p := &Planner{}
	_, err := p.Run(Action{Name: "x", Once: true,
		Key:    func() error { keys++; return nil },
		Click:  func() error { t.Fatal("click after a sent key"); return nil },
		Verify: func() (bool, error) { return false, nil }})
	if !errors.Is(err, ErrNoPath) || keys != 1 {
		t.Fatalf("%v keys=%d", err, keys)
	}
	if path, err := p.Run(Action{Name: "y", Click: func() error { return nil }}); err != nil || path != PathClick {
		t.Fatalf("nil Verify accepts: %v %v", path, err)
	}
}

package bot

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/ladder"
)

func TestEnableLadderFollowsTheKnob(t *testing.T) {
	g := newRig(t)
	ladder.Install(nil)
	if l, err := g.r.EnableLadder("", "t0", nil); l != nil || err != nil || ladder.Active() != nil {
		t.Fatalf("knob off must install nothing: %v %v", l, err)
	}
	g.e.Tune.Ladder.Enabled = 1
	l, err := g.r.EnableLadder("", "t0", nil)
	if err != nil || l == nil || ladder.Active() != l {
		t.Fatalf("knob on: %v %v", l, err)
	}
	ladder.Install(nil)
}

// teachLadder installs a ladder whose scripted player answers the point with answer until a rule is learned. It
// returns the clock to advance and the ladder to pump.
func teachLadder(t *testing.T, point, answer string) (*ladder.Ladder, *decide.ManualClock) {
	t.Helper()
	src, err := ladder.ParseScript([]byte(`{"rows":[{"point":"` + point + `","answer":"` + answer + `","confirm":"yes","delaySec":1}]}`))
	if err != nil {
		t.Fatal(err)
	}
	clk := &decide.ManualClock{}
	l, err := ladder.Open("", "bot", clk, nil, src, nil)
	if err != nil {
		t.Fatal(err)
	}
	ladder.Install(l)
	t.Cleanup(func() { ladder.Install(nil) })
	return l, clk
}

// With the ladder on and nothing learned, the trip policy and the field gate return the legacy verdicts unchanged; a
// learned answer then stops the trip or the departure (docs/t0-slice.md, points t1.town.trigger and t1.return).
func TestLadderTripAndReturnPoints(t *testing.T) {
	type scen struct {
		name       string
		retreat    int
		minPct     int
		gold, area int
		belt       []uint32
	}
	for _, sc := range []scen{{"thin belt far", 1, 50, 5000, 6, nil}, {"poor", 1, 50, 0, 1, nil}, {"gate off", 0, 0, 300, 1, []uint32{101, 102}}} {
		g, _, s := gateRig(t, sc.retreat, sc.minPct, sc.gold)
		ladder.Install(nil)
		wantTrip, wantField := g.r.tripVerdict(s), g.r.fieldVerdict(s)
		l, err := ladder.Open("", "bot", &decide.ManualClock{}, nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		ladder.Install(l)
		if got := g.r.tripVerdict(s); got != wantTrip {
			t.Errorf("%s: trip %+v, legacy %+v", sc.name, got, wantTrip)
		}
		if got := g.r.fieldVerdict(s); got != wantField {
			t.Errorf("%s: field %+v, legacy %+v", sc.name, got, wantField)
		}
		if l.Stats().Faults != 0 {
			t.Errorf("%s: faults %+v", sc.name, l.Stats())
		}
		ladder.Install(nil)
	}

	// a learned "stay in town" turns a Go into a restock round
	g, _, s := gateRig(t, 0, 0, 300)
	l, clk := teachLadder(t, ladder.PointReturn, "no")
	if v := g.r.fieldVerdict(s); !v.Go {
		t.Fatalf("untaught gate must go: %+v", v)
	}
	for i := 0; i < 8; i++ {
		g.r.fieldVerdict(s)
		clk.Advance(5 * time.Second)
		l.Pump()
	}
	if v := g.r.fieldVerdict(s); v.Go || !v.Restock {
		t.Fatalf("learned stay: %+v stats %+v", v, l.Stats())
	}

	// a learned "keep going" stops a trip the policy wants, and never invents one
	g, _, s = gateRig(t, 1, 0, 5000)
	g.w.set(func(w *world) { w.area = 6 })
	s, _ = g.w.snapshot()
	ladder.Install(nil)
	legacy := g.r.tripVerdict(s)
	if !legacy.Go {
		t.Skipf("scenario no longer trips in the legacy policy: %+v", legacy)
	}
	l, clk = teachLadder(t, ladder.PointTrigger, "continue")
	for i := 0; i < 8; i++ {
		g.r.tripVerdict(s)
		clk.Advance(5 * time.Second)
		l.Pump()
	}
	if v := g.r.tripVerdict(s); v.Go {
		t.Fatalf("learned continue: %+v stats %+v", v, l.Stats())
	}
}

// EnableLadderWith builds the store first and hands it to wire; the knob off never calls wire; the knob off makes no state dir.
func TestEnableLadderWithWiresAfterTheStore(t *testing.T) {
	g := newRig(t)
	ladder.Install(nil)
	t.Cleanup(func() { ladder.Install(nil) })
	dir := filepath.Join(t.TempDir(), "ladder")
	wired := 0
	wire := func(st *decide.Store, cat *decide.Catalogue, clk decide.Clock) (ladder.Sink, ladder.AnswerSource, error) {
		wired++
		if st == nil || cat == nil || clk == nil {
			t.Fatal("wire got nil parts")
		}
		return &ladder.RecordSink{}, nil, nil
	}
	if l, err := g.r.EnableLadderWith(dir, "human", wire); l != nil || err != nil || wired != 0 {
		t.Fatalf("knob off: %v %v %d", l, err, wired)
	}
	if _, err := os.Stat(dir); err == nil {
		t.Fatal("the knob off made the state dir")
	}
	g.e.Tune.Ladder.Enabled = 1
	l, err := g.r.EnableLadderWith(dir, "human", wire)
	if err != nil || l == nil || wired != 1 || ladder.Active() != l {
		t.Fatalf("knob on: %v %v %d", l, err, wired)
	}
	ladder.Install(nil)
	bad := func(*decide.Store, *decide.Catalogue, decide.Clock) (ladder.Sink, ladder.AnswerSource, error) {
		return nil, nil, errors.New("no screen")
	}
	if l, err := g.r.EnableLadderWith("", "human", bad); l != nil || err == nil || ladder.Active() != nil {
		t.Fatalf("a failing wire must install nothing: %v %v", l, err)
	}
}

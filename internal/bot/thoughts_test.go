package bot

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
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

// setSnap makes snap the runner's last in-game snapshot (the rig does not run the loop).
func setSnap(r *Runner, s *d2.Snapshot) {
	s.State = d2.StateInGame
	r.obMu.Lock()
	r.ob.lastSnap = s
	r.obMu.Unlock()
}

// Clearing is not a fight: a question is held only while a living hostile is within the fight radius, and is
// delivered again once it is gone (T0 live drill 2026-11-05, defect 1).
func TestLadderQuestionsReachThePlayerInTheField(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Barbarian} {
		for _, radius := range []int{22, 12} {
			g, _, s := gateRig(t, 1, 0, 5000)
			g.e.Tune.Combat.FightRadius = radius
			g.e.Tune.Ladder.Enabled = 1
			g.w.set(func(w *world) { w.area = 6 })
			s, _ = g.w.snapshot()
			ladder.Install(nil)
			if !g.r.tripVerdict(s).Go {
				t.Skip("scenario no longer trips in the legacy policy")
			}
			src, err := ladder.ParseScript([]byte(`{"rows":[{"point":"` + ladder.PointTrigger + `","answer":"continue","confirm":"yes","delaySec":1}]}`))
			if err != nil {
				t.Fatal(err)
			}
			l, err := g.r.EnableLadderWith("", "t0", func(*decide.Store, *decide.Catalogue, decide.Clock) (ladder.Sink, ladder.AnswerSource, error) {
				return &ladder.RecordSink{}, src, nil
			})
			if err != nil || l == nil {
				t.Fatal(err)
			}
			g.r.mu.Lock()
			g.r.st = StateClearing
			g.r.mu.Unlock()
			p := &d2.Player{Class: class, Life: 100, MaxLife: 100, Area: 6, X: 100, Y: 100}
			mon := func(dist int) *d2.Snapshot {
				return &d2.Snapshot{Player: p, Monsters: []d2.Unit{{ID: 9, Type: d2.TypeMonster, Class: 10, Mode: d2.MonModeStand, X: 100 + dist, Y: 100}}}
			}
			pump := func(rounds int) {
				for i := 0; i < rounds; i++ {
					g.r.tripVerdict(s)
					g.clock.Sleep(5 * time.Second)
					l.Pump()
				}
			}
			name := class.String()
			setSnap(g.r, mon(radius-1))
			if !g.r.engaged() {
				t.Fatalf("%s r%d: hostile in range must be engaged", name, radius)
			}
			pump(4)
			if st := l.Stats(); st.Answered != 0 {
				t.Fatalf("%s r%d: answered %d while engaged", name, radius, st.Answered)
			}
			setSnap(g.r, mon(radius+8))
			if g.r.engaged() {
				t.Fatalf("%s r%d: hostile out of range is not a fight", name, radius)
			}
			pump(4)
			if st := l.Stats(); st.Answered == 0 {
				t.Fatalf("%s r%d: a clearing state with nothing in range must deliver the question: %+v", name, radius, st)
			}
			ladder.Install(nil)
		}
	}
}

// Pump runs from the bot loop (Runner.step), not only inside Decide: a question opened by one decision point is
// delivered on a later tick, held while a hostile is within the fight radius, answered by the scripted player, and a
// reply after the fade is a late answer that changes nothing (t0drill Result 2, defect 5).
func TestLadderPumpRunsFromTheLoop(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Barbarian} {
		for _, delaySec := range []int{1, 100} {
			g, _, s := gateRig(t, 1, 0, 5000)
			g.e.Tune.Ladder.Enabled = 1
			g.w.set(func(w *world) { w.area = 6 })
			s, _ = g.w.snapshot()
			ladder.Install(nil)
			if !g.r.tripVerdict(s).Go {
				t.Skip("scenario no longer trips in the legacy policy")
			}
			src, err := ladder.ParseScript([]byte(fmt.Sprintf(`{"rows":[{"point":%q,"answer":"continue","confirm":"yes","delaySec":%d}]}`, ladder.PointTrigger, delaySec)))
			if err != nil {
				t.Fatal(err)
			}
			var st *decide.Store
			l, err := g.r.EnableLadderWith("", "t0", func(s *decide.Store, _ *decide.Catalogue, _ decide.Clock) (ladder.Sink, ladder.AnswerSource, error) {
				st = s
				return &ladder.RecordSink{}, src, nil
			})
			if err != nil || l == nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { ladder.Install(nil) })
			late := 0
			st.Subscribe(func(e decide.Event) {
				if e.Kind == decide.EvLateAnswer {
					late++
				}
			})
			name := fmt.Sprintf("%s delay %d", class, delaySec)
			legacy := g.r.tripVerdict(s).Go // tick N: opens the thought
			p := &d2.Player{Class: class, Life: 100, MaxLife: 100, Area: 6, X: 100, Y: 100}
			setSnap(g.r, &d2.Snapshot{Player: p, Monsters: []d2.Unit{{ID: 9, Type: d2.TypeMonster, Class: 10, Mode: d2.MonModeStand, X: 105, Y: 100}}})
			g.clock.Sleep(2 * time.Second)
			g.r.pumpLadder()
			if l.Stats().Answered != 0 || late != 0 {
				t.Fatalf("%s: answered while engaged: %+v", name, l.Stats())
			}
			setSnap(g.r, &d2.Snapshot{Player: p})
			g.clock.Sleep(2 * time.Second) // tick N+1, nothing hostile near
			g.r.pumpLadder()
			if delaySec == 1 {
				if l.Stats().Answered != 1 {
					t.Fatalf("%s: the question was not delivered and answered by tick N+1: %+v", name, l.Stats())
				}
				continue
			}
			g.clock.Sleep(time.Duration(delaySec) * time.Second)
			g.r.pumpLadder()
			if l.Stats().Answered != 0 || late != 1 {
				t.Fatalf("%s: a reply after the fade must be a late answer: %+v late %d", name, l.Stats(), late)
			}
			if got := g.r.tripVerdict(s).Go; got != legacy {
				t.Fatalf("%s: a late answer changed the action: %v -> %v", name, legacy, got)
			}
		}
	}
}

// The loop itself calls it: one step with a ladder installed answers a due question.
func TestStepPumpsTheLadder(t *testing.T) {
	g, _, s := gateRig(t, 1, 0, 5000)
	g.e.Tune.Ladder.Enabled = 1
	ladder.Install(nil)
	src, err := ladder.ParseScript([]byte(`{"rows":[{"point":"` + ladder.PointTrigger + `","answer":"continue","confirm":"yes","delaySec":1}]}`))
	if err != nil {
		t.Fatal(err)
	}
	l, err := g.r.EnableLadderWith("", "t0", func(*decide.Store, *decide.Catalogue, decide.Clock) (ladder.Sink, ladder.AnswerSource, error) {
		return &ladder.RecordSink{}, src, nil
	})
	if err != nil || l == nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ladder.Install(nil) })
	g.w.set(func(w *world) { w.area = 6 })
	s, _ = g.w.snapshot()
	g.r.tripVerdict(s)
	g.clock.Sleep(3 * time.Second)
	g.r.step(context.Background())
	if l.Stats().Answered != 1 {
		t.Fatalf("step did not pump the ladder: %+v", l.Stats())
	}
}

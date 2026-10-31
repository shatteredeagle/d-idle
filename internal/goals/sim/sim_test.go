package sim_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/goals/sim"
)

func drain(r *sim.Runner) []goals.Event {
	var out []goals.Event
	for {
		select {
		case e := <-r.Events():
			out = append(out, e)
		default:
			return out
		}
	}
}

func play(seed int64, g goals.Goal, secs int) ([]goals.Event, goals.Status) {
	r := sim.New(sim.Options{Seed: seed})
	r.Start(g)
	for i := 0; i < secs; i++ {
		r.Advance(time.Second)
	}
	return drain(r), r.Status()
}

func TestDeterministic(t *testing.T) {
	g := goals.Goal{ID: "g1", Type: goals.ReachLevel, Params: goals.Params{Level: 50}}
	a, sa := play(7, g, 600)
	b, sb := play(7, g, 600)
	if len(a) == 0 || !reflect.DeepEqual(a, b) || !reflect.DeepEqual(sa, sb) {
		t.Fatalf("same seed must give the same run (%d vs %d events)", len(a), len(b))
	}
	c, _ := play(8, g, 600)
	if reflect.DeepEqual(a, c) {
		t.Fatal("different seeds should differ")
	}
}

func TestPlayProducesEverythingAndRespectsPause(t *testing.T) {
	r := sim.New(sim.Options{Seed: 3})
	r.Start(goals.Goal{ID: "g1", Type: goals.FarmArea, Params: goals.Params{Area: "Cold Plains"}})
	for i := 0; i < 4000; i++ {
		r.Advance(time.Second)
	}
	counts := map[goals.EventType]int{}
	for _, e := range drain(r) {
		counts[e.Type]++
	}
	for _, ty := range []goals.EventType{goals.EvKill, goals.EvDrop, goals.EvLevelUp, goals.EvAreaChange, goals.EvPotion, goals.EvDeath} {
		if counts[ty] == 0 {
			t.Errorf("no %s events: %v", ty, counts)
		}
	}
	st := r.Status()
	if st.State != goals.RunnerRunning || st.Character.Level < 3 || st.Character.MaxLife == 0 || st.Character.Life > st.Character.MaxLife {
		t.Fatalf("%+v", st)
	}
	r.Pause()
	r.Advance(time.Minute)
	if n := len(drain(r)); n != 0 {
		t.Fatalf("%d events while paused", n)
	}
	r.Resume()
	r.Advance(30 * time.Second)
	if len(drain(r)) == 0 {
		t.Fatal("nothing after resume")
	}
	r.Stop()
	if s := r.Status(); s.State != goals.RunnerIdle || s.Character.Area != "Rogue Encampment" || s.GoalID != "" {
		t.Fatalf("%+v", s)
	}
}

func TestStartFailure(t *testing.T) {
	r := sim.New(sim.Options{FailStart: func(goals.Goal) error { return fmt.Errorf("nope") }})
	if err := r.Start(goals.Goal{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestEventChannelNeverBlocks(t *testing.T) {
	r := sim.New(sim.Options{Seed: 1})
	r.Start(goals.Goal{Type: goals.FarmArea, Params: goals.Params{Area: "x"}})
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100000; i++ {
			r.Advance(time.Second)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("simulator blocked on an undrained channel")
	}
}

// Manager + simulator end to end: a queue of different goals completes by itself.
func TestManagerWithSim(t *testing.T) {
	r := sim.New(sim.Options{Seed: 11})
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	m, err := goals.New(goals.Options{Runner: r, Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	add := func(g goals.Goal) {
		if _, err := m.Add(g, -1); err != nil {
			t.Fatal(err)
		}
	}
	add(goals.Goal{Type: goals.ReachLevel, Params: goals.Params{Level: 6}})
	add(goals.Goal{Type: goals.FarmArea, Params: goals.Params{Area: "Cold Plains", UntilKills: 40}})
	add(goals.Goal{Type: goals.KillBoss, Params: goals.Params{Boss: "Mephisto", Count: 2}})
	add(goals.Goal{Type: goals.CollectItem, Params: goals.Params{Count: 2, Item: goals.ItemFilter{Type: "rune"}}})
	add(goals.Goal{Type: goals.Idle, Params: goals.Params{Minutes: 1}})
	m.Start()
	for i := 0; i < 20000 && m.Snapshot().ActiveGoalID != ""; i++ {
		now = now.Add(time.Second)
		r.Advance(time.Second)
		for len(r.Events()) > 0 {
			m.Handle(<-r.Events())
		}
		m.Tick(now)
	}
	for _, g := range m.Goals() {
		if g.State != goals.StateDone {
			t.Errorf("%s is %s (%+v)", g.Title, g.State, g.Progress)
		}
	}
	if s := m.StatsSummary(); s.Totals.Kills == 0 || s.Totals.XP == 0 {
		t.Fatalf("%+v", s.Totals)
	}
}

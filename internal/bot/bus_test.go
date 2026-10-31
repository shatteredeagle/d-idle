package bot

import (
	"testing"
	"time"

	"d2lod-bot/internal/env"
	"d2lod-bot/internal/event"
	"d2lod-bot/internal/goals"
)

// With Deps.Bus every emitted event lands on the bus and still reaches the manager's channel through the bridge;
// manager-kind events on the bus do not loop back.
func TestBusPublishAndBridge(t *testing.T) {
	bus := event.NewBus("t1", nil)
	sub := bus.Subscribe("test", 16)
	g := newRig(t, func(_ *Config, d *Deps, _ *env.Tuning) { d.Bus = bus })
	defer g.r.Close()
	g.r.emit(goals.Event{Type: goals.EvKill, Monster: "Fallen"})
	bus.Publish(event.FromGoals(goals.Event{Type: goals.EvGoalAdded, GoalID: "g1"}, event.SrcManager))
	e := <-sub.C()
	if e.Kind != event.KindKill || e.Source != event.SrcObserver || e.Seq == 0 || e.Instance != "t1" {
		t.Fatalf("bus event %+v", e)
	}
	select {
	case ev := <-g.r.Events():
		if ev.Type != goals.EvKill || ev.Monster != "Fallen" {
			t.Fatalf("bridged %+v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no bridged event")
	}
	select {
	case ev := <-g.r.Events():
		t.Fatalf("manager event looped back: %+v", ev)
	case <-time.After(100 * time.Millisecond):
	}
}

package main

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/event"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/goals/sim"
)

// The manager's own events reach the bus; runner events (already published by the bot) are not duplicated.
func TestBusBridgeForwardsManagerEventsOnly(t *testing.T) {
	m, err := goals.New(goals.Options{Runner: sim.New(sim.Options{Seed: 1})})
	if err != nil {
		t.Fatal(err)
	}
	bus := event.NewBus("t", nil)
	sub := bus.Subscribe("test", 16)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go busBridge(ctx, m, bus)
	time.Sleep(50 * time.Millisecond)
	if _, err := m.Add(goals.Goal{Type: goals.ReachLevel, Params: goals.Params{Level: 5}}, -1); err != nil {
		t.Fatal(err)
	}
	m.Handle(goals.Event{Type: goals.EvKill, Monster: "x"})
	select {
	case e := <-sub.C():
		if e.Kind != event.KindGoalAdded || e.Source != event.SrcManager {
			t.Fatalf("got %+v", e)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no goal_added on the bus")
	}
	select {
	case e := <-sub.C():
		t.Fatalf("runner event duplicated on the bus: %+v", e)
	case <-time.After(150 * time.Millisecond):
	}
}

package town_test

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town"
)

// The scroll is read but no portal object ever appears: ToTown must fail in bounded (fake) time, not spin.
func TestCoverToTownPortalNeverAppears(t *testing.T) {
	r := newRig(t)
	w := r.W
	w.Player.Area = 2
	w.Add("tsc", d2.ContInventory, 0, 0)
	w.Objs = nil // sim spawns the portal on read; pretend it does not
	w.PortalLevel = 99
	done := make(chan error, 1)
	go func() { done <- r.T.ToTown(context.Background()) }()
	select {
	case err := <-done:
		if err == nil && w.Player.Area == 2 {
			t.Fatal("ToTown claims success while still in the field")
		}
	case <-time.After(20 * time.Second):
		t.Fatal("ToTown does not give up")
	}
}

func TestCoverToTownCancelledContext(t *testing.T) {
	r := newRig(t)
	w := r.W
	w.Player.Area = 2
	w.Add("tsc", d2.ContInventory, 0, 0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan error, 1)
	go func() { done <- r.T.ToTown(ctx) }()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("ToTown ignores a cancelled context")
	}
}

func TestCoverBackToFieldOutsideTownIsAnError(t *testing.T) {
	r := newRig(t)
	r.W.Player.Area = 3
	if err := r.T.BackToField(context.Background(), 2); err == nil {
		t.Fatal("expected an error from the field")
	}
}

func TestCoverChoresCancelledContext(t *testing.T) {
	r := newRig(t)
	r.W.Player.Life = 12
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan struct{})
	go func() { _, _ = r.T.Chores(ctx, town.ChorePlan{Heal: true}); close(done) }()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("Chores ignores a cancelled context")
	}
}

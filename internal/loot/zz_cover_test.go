package loot_test

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/loot"
)

// An item that can never be hovered must cost a bounded number of tries and end up in Skipped, not loop.
func TestCoverUnpickableItemIsBounded(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	w.NoHover = 1000
	w.AddGround("r05", 2, 6022, 4930)
	done := make(chan struct{})
	var rep loot.PickReport
	var err error
	go func() {
		rep, err = l.PickUp(context.Background(), loot.PickOpts{})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("PickUp does not give up on an unhoverable item")
	}
	if len(rep.Picked) != 0 {
		t.Fatalf("picked %+v", rep.Picked)
	}
	if err == nil && len(rep.Skipped) == 0 {
		t.Fatalf("failure is silent: %+v", rep)
	}
}

func TestCoverCancelledContextStopsPickUp(t *testing.T) {
	l, r := newLooter(t)
	r.W.AddGround("r05", 2, 6022, 4930)
	r.W.AddGround("r06", 2, 6024, 4930)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan struct{})
	go func() {
		_, _ = l.PickUp(ctx, loot.PickOpts{})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("PickUp ignores a cancelled context")
	}
}

// Another unit takes the item between scan and click: not an error, nothing picked.
func TestCoverItemStolenFromGround(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	w.AddGround("r05", 2, 6022, 4930)
	calls := 0
	rep, err := l.PickUp(context.Background(), loot.PickOpts{Abort: func() bool {
		calls++
		if calls == 1 {
			w.Ground = nil
		}
		return false
	}})
	if err != nil && len(rep.Picked) != 0 {
		t.Fatalf("%v %+v", err, rep)
	}
	if len(rep.Picked) != 0 {
		t.Fatalf("picked a stolen item: %+v", rep.Picked)
	}
}

package main

import (
	"context"
	"testing"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/town/sim"
)

// recoveryLooter mirrors the unexported interface the runner asserts on (internal/bot/broke.go).
type recoveryLooter interface {
	SetRecovering(on bool)
	RecoveryTotals() (picked, gold, full int)
}

// The recovery loot pass through the real adapter (primitives-review F2): the runner reaches the Looter only as a
// bot.Looter, so it must find the recovery methods on the adapter. A broke-ish character (gold between the poor line and
// the kit line) takes small piles and a sellable sword only while recovering, and the totals line is non-zero.
func TestRecoveryPickupThroughTheAdapter(t *testing.T) {
	r := sim.NewRig(t)
	l := loot.New(r.Env, r.Mv)
	l.UseSource(r.W.Snapshot, r.W.UnitStats)
	var lt bot.Looter = looterAdapter{l}
	rl, ok := lt.(recoveryLooter)
	if !ok {
		t.Fatal("looterAdapter does not expose SetRecovering/RecoveryTotals: recovery pickup can never switch on")
	}
	w := r.W
	for _, e := range []struct {
		code string
		body int
	}{{"dgr", d2.BodyRHand}, {"qui", d2.BodyTorso}} {
		w.Add(e.code, d2.ContEquipment, 0, 0).Item.Body = e.body
	}
	w.Player.Gold = 400 // above loot.poorGoldBelow, under the recovery kit line
	pile := w.AddGround("gld", 2, 6021, 4931)
	w.Stats[pile.Addr] = d2.Stats{d2.StatGold: 5}
	w.AddGround("ssd", 2, 6030, 4931)

	rep, err := lt.PickUp(context.Background(), bot.PickOpts{})
	if err != nil || rep.Picked != 0 {
		t.Fatalf("not recovering: picked %d err %v", rep.Picked, err)
	}
	rl.SetRecovering(true)
	rep, err = lt.PickUp(context.Background(), bot.PickOpts{})
	if err != nil || rep.Picked != 2 {
		t.Fatalf("recovering: picked %d (want pile and sword) err %v", rep.Picked, err)
	}
	if p, g, _ := rl.RecoveryTotals(); p != 2 || g <= 0 {
		t.Errorf("totals picked %d gold %d: the recovery loot line would print +0", p, g)
	}
	// A second pass finds nothing left: no loop, no extra time.
	rep, _ = lt.PickUp(context.Background(), bot.PickOpts{})
	if rep.Picked != 0 {
		t.Errorf("second pass picked %d", rep.Picked)
	}
	rl.SetRecovering(false)
}

// The loot memory and the gold priority reach the runner only through the adapter (the walk-back of TASK-320 never ran
// in production because the adapter hid these methods): the runner's optional interfaces must be found on it.
func TestAdapterExposesLootMemoryAndGold(t *testing.T) {
	var lt bot.Looter = looterAdapter{}
	if _, ok := lt.(interface {
		ObserveGround(*d2.Snapshot)
		ResetMemory()
		WalkBack(context.Context, func() bool, func(n, gold int, first string)) (int, error)
	}); !ok {
		t.Fatal("looterAdapter hides the loot memory (ObserveGround/ResetMemory/WalkBack)")
	}
	if _, ok := lt.(interface {
		GoldNear(*d2.Snapshot, int) bool
		GoldLedger() (seen, picked, gold, abandoned int)
	}); !ok {
		t.Fatal("looterAdapter hides the gold priority (GoldNear/GoldLedger)")
	}
}

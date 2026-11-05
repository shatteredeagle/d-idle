package town

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/ladder"
)

// With no ladder installed the repair and scroll points are the legacy tests (byte-for-byte path); with one installed
// the wired call sites ask it. Parity of the answers is proven in internal/ladder.
func TestLadderCallSites(t *testing.T) {
	tune := env.DefaultTuning()
	tn := &Town{e: &env.Env{Tune: &tune}}
	snap := &d2.Snapshot{Player: &d2.Player{Class: d2.Paladin, Gold: 300}}

	ladder.Install(nil)
	if tn.repairDue(snap, 29) != RepairDue(29, tune.Town.RepairBelowPct) || !tn.repairDue(snap, 29) || tn.repairDue(snap, 30) {
		t.Fatal("legacy repair test changed")
	}
	if !tpBuyAllowed(snap.Player) {
		t.Fatal("legacy path must always allow the purchase test to decide")
	}

	l, err := ladder.Open("", "town", &decide.ManualClock{}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	ladder.Install(l)
	defer ladder.Install(nil)
	if !tn.repairDue(snap, 29) || tn.repairDue(snap, 30) {
		t.Fatal("ladder repair differs from the legacy test at the default knob")
	}
	// nothing learned: the legacy answers stand, whatever the catalogue default would say
	for _, gold := range []int{0, 99, 100, 5000} {
		snap.Player.Gold = gold
		if !tpBuyAllowed(snap.Player) {
			t.Fatalf("gold %d: tpBuyAllowed is asked after BuyKit allowed, nothing learned must allow", gold)
		}
		for _, floor := range []int{0, 100, 300, 9000} {
			if got := tn.restockFree(snap, floor); got != (gold >= floor) {
				t.Fatalf("gold %d floor %d: restockFree %v, legacy %v", gold, floor, got, gold >= floor)
			}
		}
	}
	ladder.Install(nil)
	for _, gold := range []int{0, 299, 300} {
		snap.Player.Gold = gold
		if got := tn.restockFree(snap, 300); got != (gold >= 300) {
			t.Fatalf("legacy path: gold %d restockFree %v", gold, got)
		}
	}
	ladder.Install(l)
	snap.Player.Gold = 300
	// an unknown class is a fault, and the call site keeps the legacy answer
	snap.Player.Class = d2.Class(99)
	if !tn.repairDue(snap, 29) || l.Stats().Faults != 1 {
		t.Fatal("fault must fall back to the legacy test")
	}
}

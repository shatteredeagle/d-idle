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
	snap.Player.Gold = 99
	if tpBuyAllowed(snap.Player) {
		t.Fatal("ladder should say walk at 99 gold")
	}
	snap.Player.Gold = 100
	if !tpBuyAllowed(snap.Player) {
		t.Fatal("ladder should say buy at 100 gold")
	}
	// an unknown class is a fault, and the call site keeps the legacy answer
	snap.Player.Class = d2.Class(99)
	if !tn.repairDue(snap, 29) || l.Stats().Faults != 1 {
		t.Fatal("fault must fall back to the legacy test")
	}
}

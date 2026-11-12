package main

import (
	"testing"

	"d2lod-bot/internal/unlock"
)

// The gear sets unlock (docs/gearsets.md): the shipped ladder row `gearsets` opens at character level 15, a ladder
// without the row keeps the feature off, no ladder and the dev image leave it open.
func TestSetsGateFollowsTheShippedLadderRow(t *testing.T) {
	lad, err := unlock.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	eng, err := unlock.Open("", "t", lad, nil)
	if err != nil {
		t.Fatal(err)
	}
	none := func(string) string { return "" }
	gate := setsGate(eng, none)
	if open, why := gate(); open || why != "Reach level 15." {
		t.Fatalf("fresh hero: open=%v why=%q", open, why)
	}
	eng.Observe(unlock.Event{Type: unlock.EvLevel, N: 14})
	if open, _ := gate(); open {
		t.Fatal("level 14 must not open it")
	}
	eng.Observe(unlock.Event{Type: unlock.EvLevel, N: 15})
	if open, why := gate(); !open || why != "" {
		t.Fatalf("level 15: open=%v why=%q", open, why)
	}
	// the dev image and a run with no ladder at all
	fresh, _ := unlock.Open("", "t2", lad, nil)
	if open, _ := setsGate(fresh, func(k string) string {
		if k == "DIDLE_DEV" {
			return "1"
		}
		return ""
	})(); !open {
		t.Error("DIDLE_DEV=1 leaves the feature open")
	}
	if open, _ := setsGate(nil, none)(); !open {
		t.Error("no ladder: nothing locks")
	}
	// a ladder without the row: off, with a reason
	empty, err := unlock.Parse([]byte(`{"schema":"unlock.ladder/1","entries":[]}`))
	if err == nil {
		eng2, _ := unlock.Open("", "t3", empty, nil)
		if open, why := setsGate(eng2, none)(); open || why == "" {
			t.Errorf("row missing: open=%v why=%q", open, why)
		}
	}
}

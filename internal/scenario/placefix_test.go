package scenario

import (
	"slices"
	"testing"
)

// A drill that places the Horadric Staff must not start with the Staff's quest record done: d2d reads record 10 bit 0
// as "placed" (S SevenTombsQuest::join, FUN_0059d4f0) and its orifice then drops the C->S 0x44 placement without a
// reply, so the bot's `cmd place` got an empty answer and the Staff stayed in the pack (V lab drill 2026-11-27).
func TestStaffDrillsKeepThePlacementOpen(t *testing.T) {
	scs, err := LoadDir("../../tools/provingground/scenarios")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, s := range scs {
		if s.Step == nil || !slices.Contains(s.Step.Nodes, "place_staff") {
			continue
		}
		n++
		for _, q := range s.Pre.Quests {
			if q.Record != 10 {
				continue
			}
			if q.Word&0x0001 != 0 {
				t.Errorf("%s: record 10 = %#04x sets bit 0 (the Staff already placed); the orifice will refuse it", s.ID, q.Word)
			}
			// Bits 4 to 6 do not survive the join without the parts (HoradricStaffQuest::join): write what the game keeps.
			if q.Word&0x0070 != 0 || q.Word&0x0800 == 0 {
				t.Errorf("%s: record 10 = %#04x; want bit 11 (the Staff made) and bits 4 to 6 clear (0x0c08 is what the d2d keeps)", s.ID, q.Word)
			}
		}
	}
	if n == 0 {
		t.Fatal("no scenario runs place_staff")
	}
}

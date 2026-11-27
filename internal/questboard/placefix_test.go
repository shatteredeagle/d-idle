package questboard

import (
	"strings"
	"testing"
)

// The Seven Tombs opens on the Horadric Staff made (the Staff's objective, here a proxy: hst in the pack), not on the
// Staff's done bit: only the placement in the orifice sets that bit (S d2d SevenTombsQuest::join: record 10 bit 0 reads
// as "placed", and the orifice then drops C->S 0x44), so asking for done locked the Duriel chunk for good in real play.
func TestSevenTombsOpensOnTheStaffMade(t *testing.T) {
	b := mustBoard(t)
	staff := Fact{Known: true, Started: true, Tier: TierMemory} // 0x0c78 after Cain's talk: bit 0 clear (V lab drill 2026-11-27)
	f := facts{0: {"summoner": done(TierSave), "horadric_staff": staff}}
	in := Input{Facts: f, Ready: map[string]Word{ChunkKey("seven_tombs", "duriel"): ReadySafe}}

	if c := chunk(view(t, b, in)["seven_tombs"], "duriel"); c.State != Locked || !strings.Contains(c.Why, "Horadric Staff") {
		t.Fatalf("no Staff made: duriel = %s %q, want locked until The Horadric Staff", c.State, c.Why)
	}
	f[0]["horadric_staff"] = Fact{Known: true, Started: true, Cleared: true, Tier: TierProxy}
	if c := chunk(view(t, b, in)["seven_tombs"], "duriel"); c.State == Locked {
		t.Fatalf("Staff in the pack, word 0x0c78: duriel = %s %q, want open", c.State, c.Why)
	}
	// The Summoner still gates it on exact evidence.
	f[0]["summoner"] = done(TierProxy)
	if c := chunk(view(t, b, in)["seven_tombs"], "duriel"); c.State != Locked || !strings.Contains(c.Why, "Summoner") {
		t.Errorf("Summoner on a proxy: duriel = %s %q, want locked", c.State, c.Why)
	}
}

func TestRequiresClearedUnknownQuest(t *testing.T) {
	b := mustBoard(t)
	q, err := b.Quest("seven_tombs")
	if err != nil {
		t.Fatal(err)
	}
	q.RequiresCleared = []string{"ghost"}
	if err := b.validateQuest(q); err == nil || !strings.Contains(err.Error(), `"ghost"`) {
		t.Errorf("unknown requiresCleared = %v", err)
	}
}

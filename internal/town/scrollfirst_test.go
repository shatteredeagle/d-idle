package town_test

import (
	"testing"

	"d2lod-bot/internal/d2"
)

// g2g S1: the restock the bot runs before a gear shop. A funded hero with one scroll (reserve 2) and a full belt gets
// the second scroll and nothing else; with the reserve held it buys nothing. Sorceress and Paladin.
func TestRestockBeforeGearBuysTheMissingScroll(t *testing.T) {
	for _, cls := range []d2.Class{d2.Sorceress, d2.Paladin} {
		r := audit(t, 400, 15, true)
		r.W.Player.Class = cls
		r.W.Add("tsc", d2.ContInventory, 9, 3)
		rep := run(t, r)
		if rep.Bought["Scroll of Town Portal"] != 1 || rep.GoldAfter != 300 {
			t.Fatalf("class %v: bought %v gold %d, want one scroll for 100", cls, rep.Bought, rep.GoldAfter)
		}
		snap, _ := r.W.Snapshot()
		if n := r.T.TPCount(snap); n != 2 {
			t.Fatalf("class %v: %d scrolls after the visit", cls, n)
		}
	}
}

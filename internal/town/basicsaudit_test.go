package town_test

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/gear/geartest"
	"d2lod-bot/internal/town"
	"d2lod-bot/internal/town/sim"
)

// BASICSAUDIT (docs/basics-audit.md): the SME's town scenarios of BACKLOG section 0, run through Town.Chores on the
// sim rig (a Sorceress, real item tables, sim vendor prices: a sale pays Cost/4). Where the current behaviour matches
// the SME the test says so; where it differs the test PINS the current behaviour and names the gap, so a fix flips the
// test on purpose. Gold values are the SME's; the rig, not the original game, supplies prices (UNVERIFIED on d2d).

// fullBelt fills the 4-slot belt to the Sorceress plan (two healing columns, two mana columns).
func fullBelt(w *sim.World) {
	w.Add("hp1", d2.ContBelt, 0, 0)
	w.Add("hp1", d2.ContBelt, 1, 0)
	w.Add("mp1", d2.ContBelt, 2, 0)
	w.Add("mp1", d2.ContBelt, 3, 0)
}

func townPlan() town.ChorePlan {
	return town.ChorePlan{Potions: true, Scrolls: true, Sell: true, Identify: true, Stash: true, Gold: true, Repair: true}
}

func audit(t *testing.T, gold, clvl int, belt bool) *rig {
	t.Helper()
	geartest.Need(t)
	r := newRig(t)
	dressed(r.W)
	r.W.Player.Gold, r.W.Player.Level = gold, clvl
	if belt {
		fullBelt(r.W)
	}
	return r
}

func run(t *testing.T, r *rig) town.ChoreReport {
	t.Helper()
	rep, err := r.T.Chores(context.Background(), townPlan())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("bought %v spent %d sold %d gold %d -> %d %+v", rep.Bought, rep.Spent, rep.Sold, rep.GoldBefore, rep.GoldAfter, rep.Steps)
	return rep
}

func addMagic(r *rig, code string, x, y int) {
	r.W.Add(code, d2.ContInventory, x, y)
	r.W.Items[len(r.W.Items)-1].Item.Quality = 4
}

// S1 agrees with the SME: gold 50, empty belt: one healing potion, no scroll, no tome.
func TestAuditS1PoorBuysOnePotion(t *testing.T) {
	r := audit(t, 50, 1, false)
	rep := run(t, r)
	if rep.Bought["Minor Healing Potion"] != 1 || len(rep.Bought) != 1 || rep.GoldAfter != 20 {
		t.Fatalf("bought %v gold %d", rep.Bought, rep.GoldAfter)
	}
}

// S2 differs (gap G1): with a scroll and a full belt the SME expects no shop visit and 120 gold kept. IDScrollTarget
// is 2 with no tome, so needScrolls opens the shop and buys an identify scroll.
func TestAuditS2ScrollHeldBeltFullBuysIDScroll(t *testing.T) {
	r := audit(t, 120, 1, true)
	r.W.Add("tsc", d2.ContInventory, 9, 3)
	rep := run(t, r)
	if rep.Bought["Scroll of Town Portal"] != 0 || rep.Bought["Scroll of Identify"] != 1 || rep.GoldAfter != 40 {
		t.Fatalf("pinned current behaviour changed: bought %v gold %d (SME: nothing bought, 120 kept)", rep.Bought, rep.GoldAfter)
	}
}

// S3 differs (gap G1): gold 300, no scroll, clvl 9: the portal scroll is bought (as the SME wants) and so is an identify
// scroll, leaving 120 where the SME wants at least 150.
func TestAuditS3OneScrollButAlsoAnIDScroll(t *testing.T) {
	r := audit(t, 300, 9, true)
	rep := run(t, r)
	if rep.Bought["Scroll of Town Portal"] != 1 || rep.Bought["Scroll of Identify"] != 1 || rep.GoldAfter != 120 {
		t.Fatalf("pinned current behaviour changed: bought %v gold %d (SME: 1 scroll, keep 150)", rep.Bought, rep.GoldAfter)
	}
}

// S5 partly agrees: one visit sells all seven items and the portal scroll count is unchanged; an identify scroll is
// bought with the proceeds (gap G1).
func TestAuditS5OneVisitSellsAllScrollCountUnchanged(t *testing.T) {
	r := audit(t, 30, 1, true)
	r.W.Add("tsc", d2.ContInventory, 9, 3)
	for i := 0; i < 7; i++ {
		addMagic(r, "ssd", i, 0)
	}
	rep := run(t, r)
	if rep.Sold != 7 || countCode(r, "tsc") != 1 || rep.Bought["Scroll of Town Portal"] != 0 {
		t.Fatalf("sold %d tsc %d bought %v", rep.Sold, countCode(r, "tsc"), rep.Bought)
	}
}

// S6 (rune and gem part) agrees: neither is sold; both go to the stash. The magic charm part depends on its stats
// (pickit rule "charms"): an identified magic charm without life, resist, FHR or skills is sold (code reading only).
func TestAuditS6RuneAndGemAreKept(t *testing.T) {
	r := audit(t, 30, 1, true)
	r.W.Add("r07", d2.ContInventory, 3, 0)
	r.W.Add("gcv", d2.ContInventory, 4, 0)
	rep := run(t, r)
	if rep.Sold != 0 || count(r.W, d2.ContStash) != 2 {
		t.Fatalf("sold %d stash %d", rep.Sold, count(r.W, d2.ContStash))
	}
}

// S7 differs (gap G2): an unidentified magic ring is identified (an identify scroll is bought) and then sold; the SME
// wants it sold unidentified with no scroll bought.
func TestAuditS7UnidentifiedRingIsIdentifiedFirst(t *testing.T) {
	r := audit(t, 200, 1, true)
	addMagic(r, "rin", 0, 0)
	r.W.Items[len(r.W.Items)-1].Item.Flags = 0
	rep := run(t, r)
	if rep.Identified != 1 || rep.Bought["Scroll of Identify"] == 0 {
		t.Fatalf("pinned current behaviour changed: identified %d bought %v", rep.Identified, rep.Bought)
	}
}

// S8 and S9 agree: durability 28 percent with 200 gold is repaired in the same visit, 90 percent is left alone.
func TestAuditS8S9RepairThreshold(t *testing.T) {
	for _, c := range []struct {
		pct      int32
		repaired bool
	}{{28, true}, {90, false}} {
		r := audit(t, 200, 1, true)
		r.W.RepairBtn = town.RepairAll // the sim's default button is elsewhere
		for _, u := range r.W.Items {
			if u.Item.Where == d2.ContEquipment {
				r.W.Stats[u.Addr] = d2.Stats{72: c.pct, 73: 100}
			}
		}
		rep := run(t, r)
		if rep.Repaired != c.repaired {
			t.Errorf("durability %d%%: repaired %v, want %v", c.pct, rep.Repaired, c.repaired)
		}
	}
}

// S11 agrees on the keep line (300 + 50 x clvl = 550 at clvl 5) but the shop spends the surplus first: the stash step
// banks only what is left (gap G3, a policy question: a tome and five scrolls come before the bank).
func TestAuditS11KeepLineAtClvl5(t *testing.T) {
	r := audit(t, 1500, 5, true)
	rep := run(t, r)
	if rep.GoldAfter != 550 || r.W.Player.StashGold != 40 {
		t.Fatalf("gold %d bank %d bought %v", rep.GoldAfter, r.W.Player.StashGold, rep.Bought)
	}
}

// S12 agrees: a spare healing potion in the pack goes to a free belt slot before leaving town, nothing is bought.
func TestAuditS12SpareGoesToTheBelt(t *testing.T) {
	r := audit(t, 50, 1, false)
	r.W.Add("hp1", d2.ContBelt, 0, 0)
	r.W.Add("mp1", d2.ContBelt, 2, 0)
	r.W.Add("hp1", d2.ContInventory, 0, 0)
	rep := run(t, r)
	if count(r.W, d2.ContBelt) != 3 || countCode(r, "hp1") != 0 || rep.Bought["Minor Healing Potion"] != 0 {
		t.Fatalf("belt %d pack hp %d bought %v", count(r.W, d2.ContBelt), countCode(r, "hp1"), rep.Bought)
	}
}

// S10 differs (gap G4): a repair that does not confirm is retried on every visit (no latch inside the chore; only the
// trip-level NeedsTrip memory is quiet). The SME wants one log line and no retry until gold or durability changes.
func TestAuditS10FailedRepairIsRetriedEveryVisit(t *testing.T) {
	r := audit(t, 200, 1, true) // the sim button is not at town.RepairAll: the click changes nothing, as a failed repair
	for _, u := range r.W.Items {
		if u.Item.Where == d2.ContEquipment {
			r.W.Stats[u.Addr] = d2.Stats{72: 28, 73: 100}
		}
	}
	for visit := 1; visit <= 2; visit++ {
		rep := run(t, r)
		tried := false
		for _, st := range rep.Steps {
			if st.Name == "repair" && st.Ran && st.Err != "" {
				tried = true
			}
		}
		if !tried {
			t.Fatalf("visit %d: repair was not attempted again: %+v", visit, rep.Steps)
		}
	}
}

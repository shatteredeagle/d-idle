package town_test

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town"
)

// BASICSFIX tests. Policy numbers are the d2-sme's EXP rule of thumb; prices come from the sim rig.

func plan2() town.ChorePlan {
	return town.ChorePlan{Potions: true, Scrolls: true, Identify: true, Sell: true, Gold: true}
}

// A rare ring in the pack is worth a scroll: it is bought and the ring identified.
func TestIDScrollBoughtForUnidentifiedRare(t *testing.T) {
	r := audit(t, 400, 5, true)
	r.W.Add("tsc", d2.ContInventory, 9, 3)
	r.W.Add("rin", d2.ContInventory, 0, 0)
	r.W.Items[len(r.W.Items)-1].Item.Quality = d2.QualityRare
	r.W.Items[len(r.W.Items)-1].Item.Flags = 0
	rep := run(t, r)
	if rep.Bought["Scroll of Identify"] == 0 || rep.Identified != 1 {
		t.Fatalf("bought %v identified %d", rep.Bought, rep.Identified)
	}
}

// idScrollNeed 0 is the old rule: identify scrolls are topped up with nothing to identify.
func TestIDScrollNeedOffRestoresOldTopUp(t *testing.T) {
	r := audit(t, 120, 1, true)
	r.Env.Tune.Town.IDScrollNeed = 0
	r.W.Add("tsc", d2.ContInventory, 9, 3)
	rep := run(t, r)
	if rep.Bought["Scroll of Identify"] != 1 {
		t.Fatalf("bought %v", rep.Bought)
	}
}

// Potions first: gold 120 at clvl 1 buys potions only, no scroll (the scroll needs clvl 4).
func TestPotionsFirstNoScrollBeforeKitLevel(t *testing.T) {
	r := audit(t, 120, 1, false)
	rep, err := r.T.Chores(context.Background(), plan2())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Bought["Scroll of Town Portal"] != 0 || len(rep.Bought) == 0 {
		t.Fatalf("bought %v", rep.Bought)
	}
}

// From the kit level, the kit potions come first, then the scroll when gold for one is left.
func TestPotionsFirstThenScrollAtKitLevel(t *testing.T) {
	r := audit(t, 700, 8, false) // keep line 700; kit potions about 450 in the sim, then 100 for the scroll
	rep, err := r.T.Chores(context.Background(), plan2())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Bought["Scroll of Town Portal"] < 1 || len(rep.Bought) < 2 { // goldfloor: the mana cap leaves gold for a second scroll
		t.Fatalf("bought %v gold %d", rep.Bought, rep.GoldAfter)
	}
}

// At the kit level but with too little gold left after the kit potions the scroll is skipped.
func TestPotionsFirstSkipsScrollWhenPoor(t *testing.T) {
	r := audit(t, 130, 4, false)
	rep, err := r.T.Chores(context.Background(), plan2())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Bought["Scroll of Town Portal"] != 0 {
		t.Fatalf("bought %v", rep.Bought)
	}
}

// bankFirst never banks below the minimal kit even with a tiny keep line.
func TestBankFirstKeepsTheKit(t *testing.T) {
	r := audit(t, 1000, 1, true)
	r.Env.Tune.Town.StashGoldBase, r.Env.Tune.Town.StashGoldPerLvl = 0, 0
	r.W.Add("tsc", d2.ContInventory, 9, 3)
	r.Env.Tune.Town.TPScrollTarget = 1
	r.Env.Tune.Town.IDScrollNeed = 1
	_ = run(t, r)
	if r.W.Player.Gold < 190 {
		t.Fatalf("gold %d: the kit line (one scroll and three potions) was banked", r.W.Player.Gold)
	}
}

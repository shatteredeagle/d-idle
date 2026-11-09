package town_test

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/gear/geartest"
	"d2lod-bot/internal/town"
)

type fakeMarks map[uint32]string

func (m fakeMarks) MarkOf(u d2.Unit) string { return m[u.ID] }

// A keep mark protects an item the town would sell; a sell mark sells one the pickit keeps. The marks come from the
// watch screen's inventory page (docs/inventory-ui.md).
func TestPlayerMarksDecideTheSale(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	dressed(r.W)
	r.W.Player.Gold = 0
	fillPack(r, "rin", 4, true) // a full pack of unidentified rings is sold (TestFullPackOfUnidentifiedJewelryIsSold)
	var kept d2.Unit
	for _, u := range r.W.Items {
		if u.Item.Where == d2.ContInventory {
			kept = u
			break
		}
	}
	r.T.SetItemMarks(fakeMarks{kept.ID: "keep"})
	if _, err := r.T.Chores(context.Background(), town.ChorePlan{Identify: true, Sell: true}); err != nil {
		t.Fatal(err)
	}
	if n := countCode(r, "rin"); n != 1 {
		t.Errorf("%d rings left, want only the one marked keep", n)
	}
}

func TestPlayerSellMarkSellsWhatThePickitKeeps(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	dressed(r.W)
	a := r.W.Add("hp1", d2.ContInventory, 0, 0)
	r.W.Add("hp1", d2.ContInventory, 1, 0)
	r.T.SetItemMarks(fakeMarks{a.ID: "sell"})
	if _, err := r.T.Chores(context.Background(), town.ChorePlan{Sell: true}); err != nil {
		t.Fatal(err)
	}
	if n := countCode(r, "hp1"); n != 1 {
		t.Errorf("%d potions left, want 1 (the marked one sold)", n)
	}
}

// A pinned slot is not filled by found gear; once unpinned the same call wears it.
func TestPinnedSlotIsLeftAlone(t *testing.T) {
	r := earlyRig(t, d2.Sorceress, 9)
	r.W.Add("qui", d2.ContInventory, 0, 0)
	pinned := true
	r.T.SetPinned(func(body int) bool { return pinned && body == d2.BodyTorso })
	if done, err := r.T.EquipFound(context.Background()); err != nil || len(done) != 0 {
		t.Fatalf("pinned slot was filled: %v %v", done, err)
	}
	pinned = false
	if done, err := r.T.EquipFound(context.Background()); err != nil || len(done) != 1 {
		t.Fatalf("unpinned slot: %v %v", done, err)
	}
}

func TestNeverSellReasonIsClassIndependent(t *testing.T) {
	if town.NeverSellReason(nil, d2.Unit{}) == "" {
		t.Error("an unknown item must be refused")
	}
}

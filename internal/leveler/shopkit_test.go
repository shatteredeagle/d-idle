package leveler

import (
	"d2lod-bot/internal/gear/geartest"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/resource"
)

// The Javelin of the soak (stack priced 240 at the vendor, base cost 5, max stack 60 on the real tables).
func TestEstimatePriceStackable(t *testing.T) {
	b := &data.ItemBase{Cost: 5, Stackable: true, MaxStack: 60}
	got := estimatePrice(b, &d2.Item{Quality: d2.QualityNormal})
	if got < 240 {
		t.Fatalf("estimate %d for a 240 gold javelin stack", got)
	}
	if got := estimatePrice(&data.ItemBase{Cost: 168}, &d2.Item{Quality: d2.QualityNormal}); got != 168 {
		t.Fatalf("plain item estimate %d", got)
	}
}

func TestOffClassWeaponIsNeverBought(t *testing.T) {
	staff := cand(1, "Short Staff", 4, true, d2.BodyRHand)
	staff.Price = 168
	spear := cand(2, "Spear", 3.6, true, d2.BodyRHand)
	spear.Price = 281
	spear.OffClass = true
	got := PlanBuys(map[int]Cand{}, nil, []Cand{spear}, 2399)
	if len(got) != 0 {
		t.Fatalf("bought %v with only an off-class weapon on offer", got)
	}
	got = PlanBuys(map[int]Cand{}, nil, []Cand{spear, staff}, 2399)
	if len(got) != 1 || got[0].ID() != 1 {
		t.Fatalf("buys %v", got)
	}
}

// Budget 2399 with an empty belt and no portal: the kit reserve comes off the budget before any gear.
func TestKitReserveComesFirst(t *testing.T) {
	tu := resource.Tuning{KitTPGold: 100, KitPotionGold: 30, HPFirst: 3, MPFirst: 2}
	r := resource.KitShortfall(false, 0, 0, tu)
	if r != 100+5*30 {
		t.Fatalf("shortfall %d", r)
	}
	if got := resource.GearBudget(2399, 2399, r); got != 2399-r {
		t.Fatalf("budget %d", got)
	}
	if got := resource.GearBudget(2399, 300, r); got != 50 {
		t.Fatalf("poor budget %d", got)
	}
	if got := resource.GearBudget(2399, 200, r); got != 0 {
		t.Fatalf("broke budget %d", got)
	}
	if resource.KitShortfall(true, 3, 2, tu) != 0 {
		t.Fatal("complete kit has no shortfall")
	}
}

func TestEquipFailureMemoryPerItemAndSlot(t *testing.T) {
	lvl := 6
	l := &Leveler{o: Options{Snapshot: func() (*d2.Snapshot, error) { return &d2.Snapshot{Player: &d2.Player{Level: lvl}}, nil }}}
	l.markBadSwap(Swap{Item: cand(7, "amulet", 5, true, d2.BodyNeck), Slot: d2.BodyNeck})
	if !l.isBadSlot(7, d2.BodyNeck, 6) {
		t.Fatal("not remembered")
	}
	if l.isBadSlot(7, d2.BodyNeck, 7) {
		t.Fatal("level up must clear")
	}
	if l.isBadSlot(8, d2.BodyNeck, 6) {
		t.Fatal("another item is fresh")
	}
	if !l.badEverywhere(7, []int{d2.BodyNeck}, 6) {
		t.Fatal("single-slot item should be bad everywhere")
	}
	l.markBadSwap(Swap{Item: cand(9, "ring", 5, true), Slot: d2.BodyRRing})
	if l.badEverywhere(9, []int{d2.BodyRRing, d2.BodyLRing}, 6) {
		t.Fatal("ring failed one slot only")
	}
	l.markBadSwap(Swap{Slot: -1, Item: cand(10, "x", 1, true)})
	if !l.badEverywhere(10, []int{d2.BodyHead}, 6) {
		t.Fatal("failed purchase is bad everywhere")
	}
}

func TestRealJavelinAndSpearForSorceress(t *testing.T) {
	geartest.Need(t)
	db := realDB(t)
	p := &d2.Player{Class: d2.Sorceress, Level: 6, Strength: 20, Dexterity: 25}
	for code, want := range map[string]struct{ usable, off bool }{"jav": {false, false}, "spr": {true, true}, "sst": {true, false}} {
		for i, b := range db.Items {
			if b != nil && b.Code == code {
				c := Candidate(db, p, d2.Unit{Class: i, Item: &d2.Item{Quality: d2.QualityNormal}}, nil)
				if c.Usable != want.usable || c.OffClass != want.off {
					t.Errorf("%s: usable %v off %v why %q", code, c.Usable, c.OffClass, c.Why)
				}
			}
		}
	}
}

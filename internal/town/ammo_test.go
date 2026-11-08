package town_test

import (
	"context"
	"d2lod-bot/internal/gear/geartest"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/gear"
	"d2lod-bot/internal/town"
)

func codesOf(ps []gear.Purchase) (o []string) {
	for _, p := range ps {
		o = append(o, p.Code)
	}
	return
}

func TestAmmoPlan(t *testing.T) {
	geartest.Need(t)
	jav := gear.Equipped{Weapon: "jav", WeaponQty: 40, Offhand: "buc"}
	// bow path, nothing ranged: start list (bow, arrows)
	if got := codesOf(town.AmmoPlan("Amazon", "bow", jav, nil)); len(got) != 2 || got[0] != "sbw" || got[1] != "aqv" {
		t.Fatalf("start: %v", got)
	}
	// the bow is already in the pack: only arrows
	if got := codesOf(town.AmmoPlan("Amazon", "bow", jav, map[string]bool{"sbw": true})); len(got) != 1 || got[0] != "aqv" {
		t.Fatalf("bow in pack: %v", got)
	}
	// javelin path with a healthy stack, and other classes: nothing
	if got := town.AmmoPlan("Amazon", "javelin", jav, nil); len(got) != 0 {
		t.Fatalf("javelin ok: %v", got)
	}
	if got := town.AmmoPlan("Necromancer", "", gear.Equipped{Weapon: "wnd"}, nil); len(got) != 0 {
		t.Fatalf("necro: %v", got)
	}
	// KITSTALL: a Sorceress wearing a bow buys nothing
	if got := town.AmmoPlan("Sorceress", "", gear.Equipped{Weapon: "sbw"}, nil); len(got) != 0 {
		t.Fatalf("sorceress with a bow: %v", got)
	}
	// worn bow, low arrows: restock
	low := gear.Equipped{Weapon: "sbw", Offhand: "aqv", OffhandQty: 10}
	if got := codesOf(town.AmmoPlan("Amazon", "bow", low, nil)); len(got) != 1 || got[0] != "aqv" {
		t.Fatalf("low: %v", got)
	}
	// worn bow, healthy quiver
	if got := town.AmmoPlan("Amazon", "bow", gear.Equipped{Weapon: "sbw", Offhand: "aqv", OffhandQty: 300}, nil); len(got) != 0 {
		t.Fatalf("healthy: %v", got)
	}
}

// Needs /tmp/d2mpq (skipped without): the Amazon with a Javelin and Buckler buys a bow and arrows at Charsi and wears them.
func TestAmmoChoreStartFlow(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	w := r.W
	w.Player.Class = d2.Amazon
	w.Stock(r.Charsi, "sbw", "aqv", "cqv")
	w.Add("jav", d2.ContEquipment, 0, 0)
	w.Add("buc", d2.ContEquipment, 0, 0)
	r.T.SetAmmoPath("bow")
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Ammo: true})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Bought["Short Bow"] != 1 {
		t.Fatalf("bought %v steps %+v", rep.Bought, rep.Steps)
	}
	if w.UI.AnyPanelOpen() || w.Cursor != nil {
		t.Errorf("panels %v cursor %v", w.UI, w.Cursor)
	}
}

// BUILDV2: the bow-path Amazon on the starting Javelin makes a trip at 150 gold for a bow, and stops once one is owned.
func TestBowTripAtGold(t *testing.T) {
	geartest.Need(t)
	r := earlyRig(t, d2.Amazon, 3)
	w := r.W
	for i := 0; i < 4; i++ {
		w.Add("hp1", d2.ContBelt, i, 0)
	}
	w.Add("mp1", d2.ContBelt, 3, 0)
	w.Add("tsc", d2.ContInventory, 9, 3)
	w.Add("jav", d2.ContEquipment, 0, 0)
	w.Add("buc", d2.ContEquipment, 0, 0)
	r.T.SetAmmoPath("bow")
	w.Player.Gold = 149
	s, _ := w.Snapshot()
	if ok, why := r.T.NeedsTrip(s); ok {
		t.Fatalf("trip at 149 gold: %q", why)
	}
	w.Player.Gold = 150
	s, _ = w.Snapshot()
	if ok, why := r.T.NeedsTrip(s); !ok || why != "bow-path Amazon without a bow and gold for one" {
		t.Fatalf("no bow trip at 150 gold: %v %q", ok, why)
	}
	w.Add("sbw", d2.ContInventory, 0, 0)
	s, _ = w.Snapshot()
	if ok, why := r.T.NeedsTrip(s); ok && why == "bow-path Amazon without a bow and gold for one" {
		t.Fatalf("bow trip with a bow in the pack")
	}
}

// equipfail (VERIFIED LIVE): the Amazon's Short Bow was refused while the Javelin AND the Buckler were worn (one worn
// item may come off per equip click), bought, sold and bought again. The bow must end up worn with the quiver.
func TestAmmoWearsTwoHandedBowOverJavelinAndBuckler(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	w := r.W
	w.OneOffPerClick = true
	for _, b := range w.DB.Items {
		if b.Code == "sbw" && !b.TwoHanded {
			t.Fatal("Short Bow is not two-handed in this table: the rule under test does not apply")
		}
	}
	w.Player.Class = d2.Amazon
	w.Stock(r.Charsi, "sbw", "aqv", "cqv")
	w.Add("jav", d2.ContEquipment, 0, 0).Item.Body = d2.BodyRHand
	w.Add("buc", d2.ContEquipment, 0, 0).Item.Body = d2.BodyLHand
	r.T.SetAmmoPath("bow")
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Ammo: true})
	if err != nil {
		t.Fatal(err)
	}
	codes := map[int]string{}
	for _, u := range w.Items {
		if u.Item.Where == d2.ContEquipment {
			codes[u.Item.Body] = w.DB.Items[u.Class].Code
		}
	}
	if codes[d2.BodyRHand] != "sbw" || w.Cursor != nil {
		t.Fatalf("worn %v cursor %v steps %+v", codes, w.Cursor, rep.Steps)
	}
}

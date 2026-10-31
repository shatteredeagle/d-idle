package town_test

import (
	"context"
	"d2lod-bot/internal/gear/geartest"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
)

// earlyRig is the rig with the early-gear policy on (the shared rig pins it off) and a character of class cls at clvl.
func earlyRig(t *testing.T, cls d2.Class, clvl int) *rig {
	r := newRig(t)
	r.Env.Tune.Resource.EarlyGearMax = 15
	r.W.Player.Class, r.W.Player.Level = cls, clvl
	r.W.Player.Gold = 600
	return r
}

// A naked Sorceress at clvl 11 buys no armor and no weapon: the rebuy is potions only.
func TestEarlyRebuyBuysNoArmorForSorceress(t *testing.T) {
	r := earlyRig(t, d2.Sorceress, 11)
	rep, err := r.T.Rebuy(context.Background(), 600)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := r.W.Snapshot()
	if len(rep.Equipped) != 0 || len(rep.Missing) != 0 || s.Inventory.Equipped(d2.BodyTorso) != nil || s.Inventory.Equipped(d2.BodyRHand) != nil {
		t.Fatalf("early Sorceress rebuy bought gear: %+v", rep)
	}
	// the same character past the early limit rebuys armor again (the knob keeps the old path reachable)
	r2 := earlyRig(t, d2.Sorceress, 16)
	rep2, _ := r2.T.Rebuy(context.Background(), 600)
	if len(rep2.Equipped) == 0 {
		t.Fatalf("clvl 16 rebuy does not buy gear: %+v", rep2)
	}
}

// An Amazon without a weapon buys a bow, not the cheapest melee weapon, and no armor.
func TestEarlyRebuyAmazonBuysBow(t *testing.T) {
	r := earlyRig(t, d2.Amazon, 4)
	r.W.Stock(r.Charsi, "sbw")
	rep, err := r.T.Rebuy(context.Background(), 600)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := r.W.Snapshot()
	w := s.Inventory.Equipped(d2.BodyRHand)
	if w == nil || w.Class != codeClass(r, "sbw") {
		t.Fatalf("Amazon weapon %+v rep %+v", w, rep)
	}
	if s.Inventory.Equipped(d2.BodyTorso) != nil {
		t.Errorf("armor bought: %+v", rep)
	}
}

// Found armor is worn only when better; an empty slot takes anything; the worse piece stays in the pack.
func TestEquipFoundOnlyWhenBetter(t *testing.T) {
	r := earlyRig(t, d2.Sorceress, 9)
	w := r.W
	w.Add("qui", d2.ContInventory, 0, 0) // quilted armor, defense 8-11
	ctx := context.Background()
	if done, err := r.T.EquipFound(ctx); err != nil || len(done) != 1 {
		t.Fatalf("empty slot: %v %v", done, err)
	}
	s, _ := w.Snapshot()
	if s.Inventory.Equipped(d2.BodyTorso) == nil {
		t.Fatal("quilted armor not worn")
	}
	w.Add("qui", d2.ContInventory, 2, 0) // same base: not better
	if done, _ := r.T.EquipFound(ctx); len(done) != 0 {
		t.Fatalf("equal armor was worn: %v", done)
	}
	w.Add("stu", d2.ContInventory, 4, 0) // studded leather, defense 32
	done, err := r.T.EquipFound(ctx)
	if err != nil || len(done) != 1 {
		t.Fatalf("better armor: %v %v", done, err)
	}
	s, _ = w.Snapshot()
	if got := s.Inventory.Equipped(d2.BodyTorso); got == nil || got.Class != codeClass(r, "stu") {
		t.Fatalf("worn %+v", got)
	}
	if w.Cursor != nil {
		t.Error("the swapped-out armor is still on the cursor")
	}
	// strength too low for what lies in the pack: not worn
	w.Player.Strength = 5
	w.Add("plt", d2.ContInventory, 6, 0) // plate mail, needs far more strength
	if done, _ := r.T.EquipFound(ctx); len(done) != 0 {
		t.Fatalf("unwearable armor worn: %v", done)
	}
}

// A trip is never made for armor; a missing needed weapon in the pack is a reason.
func TestFoundGearTripOnlyForNeededWeapon(t *testing.T) {
	geartest.Need(t)
	r := earlyRig(t, d2.Sorceress, 9)
	w := r.W
	for i := 0; i < 3; i++ {
		w.Add("hp1", d2.ContBelt, i, 0)
	}
	w.Add("mp1", d2.ContBelt, 3, 0)
	w.Add("tsc", d2.ContInventory, 9, 3)
	w.Add("qui", d2.ContInventory, 0, 0)
	w.Add("hax", d2.ContInventory, 2, 0)
	s, _ := w.Snapshot()
	if ok, why := r.T.NeedsTrip(s); ok {
		t.Fatalf("a Sorceress trip for found gear: %q", why)
	}
	w.Player.Class = d2.Barbarian
	s, _ = w.Snapshot()
	if ok, why := r.T.NeedsTrip(s); !ok || why != "found weapon to wear" {
		t.Fatalf("Barbarian with a hand axe in the pack and no weapon: %v %q", ok, why)
	}
}

func codeClass(r *rig, code string) int {
	for i, b := range r.W.DB.Items {
		if b.Code == code {
			return i
		}
	}
	return -1
}

// Live 2026-10-30 (soak i4/i5): the game ignored the first slot click and left the item on the cursor; the equip
// retries at other points of the box (the right-click retry is switched off, 1.14d equips by click only), and says why when nothing works.
func TestEquipSurvivesIgnoredSlotClicks(t *testing.T) {
	for i, tc := range []struct {
		name           string
		ignore         int
		rightClick, ok bool
	}{
		{"first click ignored", 1, false, true},
		{"all clicks ignored, right click is off (EQUIPREQ)", 99, true, false},
		{"nothing works", 99, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := earlyRig(t, d2.Barbarian, 20+i) // own level per case: refusals are remembered by unit id and stats
			w := r.W
			w.SlotIgnore, w.RightClickEquip = tc.ignore, tc.rightClick
			w.Add("hax", d2.ContInventory, 2, 0)
			done, err := r.T.EquipFound(context.Background())
			s, _ := w.Snapshot()
			worn := s.Inventory.Equipped(d2.BodyRHand) != nil
			if worn != tc.ok || (err == nil) != tc.ok {
				t.Fatalf("worn %v err %v done %v", worn, err, done)
			}
			if w.Cursor != nil {
				t.Errorf("item left on the cursor: %+v", w.Cursor)
			}
			if !tc.ok && !strings.Contains(err.Error(), "still on the cursor") {
				t.Errorf("no reason in %v", err)
			}
		})
	}
}

package town_test

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2"
)

// gear.decisions = player: across a town visit (found-gear equip, the poverty rebuy, the chores with a belt and a worn
// slot to fill) the town equips and buys no gear and makes no click on an equipment slot (docs/gear-policy.md).
func TestPlayerPolicyTownVisitMakesNoGearDecision(t *testing.T) {
	for _, cls := range []d2.Class{d2.Paladin, d2.Sorceress} {
		r := earlyRig(t, cls, 30)
		r.Env.Tune.Resource.GearDecisions = 0
		r.W.Player.Gold = 3000
		r.W.Add("hax", d2.ContInventory, 4, 0)
		r.W.Add("cap", d2.ContInventory, 0, 0)
		before := len(r.W.Items)
		if done, err := r.T.EquipFound(context.Background()); err != nil || len(done) != 0 {
			t.Fatalf("%v: EquipFound %v %v", cls, done, err)
		}
		rep, err := r.T.Rebuy(context.Background(), 3000)
		if err != nil {
			t.Fatal(err)
		}
		if len(rep.Equipped) != 0 {
			t.Fatalf("%v: rebuy equipped %v", cls, rep.Equipped)
		}
		s, _ := r.W.Snapshot()
		for _, b := range []int{d2.BodyRHand, d2.BodyLHand, d2.BodyTorso, d2.BodyHead, d2.BodyBelt, d2.BodyFeet, d2.BodyGloves} {
			if s.Inventory.Equipped(b) != nil {
				t.Fatalf("%v: slot %d filled", cls, b)
			}
		}
		// the pack grew only by consumables, never by a vendor weapon or armor
		for _, u := range r.W.Items[before:] {
			if u.Item.Where == d2.ContInventory {
				if b := r.W.DB.Items[u.Class]; b != nil && (b.Code == "hax" || b.Code == "ssd" || b.Code == "cap" || b.Code == "lbt" || b.Code == "buc") {
					t.Fatalf("%v: bought gear %s", cls, b.Code)
				}
			}
		}
	}
}

// The same hero under the bot policy still wears the weapon (the legacy path is reachable).
func TestPlayerPolicyOffKeepsLegacyEquip(t *testing.T) {
	r := earlyRig(t, d2.Paladin, 30)
	r.Env.Tune.Resource.GearDecisions = 1
	r.W.Add("hax", d2.ContInventory, 4, 0)
	if done, err := r.T.EquipFound(context.Background()); err != nil || len(done) == 0 {
		t.Fatalf("legacy equip: %v %v", done, err)
	}
}

package leveler_test

import (
	"context"
	"errors"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/gear/geartest"
	"d2lod-bot/internal/leveler"
)

// The player's equip goes through the normal click path, and the pinned slot survives a better item in the pack. The
// same test runs for two classes: nothing here is class specific (docs/inventory-ui.md).
func TestPlayerEquipPinsTheSlot(t *testing.T) {
	geartest.Need(t)
	for _, cls := range []d2.Class{d2.Sorceress, d2.Paladin} {
		for _, pinned := range []bool{false, true} {
			r, _, us := equipRig(t, "amu")
			r.W.Player.Class = cls
			pin := func(body int) bool { return pinned && body == d2.BodyNeck }
			lv := leveler.New(r.Env, leveler.Options{Snapshot: r.W.Snapshot, Pinned: pin})
			s, _ := r.W.Snapshot()
			if _, err := lv.CheckPlayerEquip(s, us[0].ID, d2.BodyHead); !errors.Is(err, leveler.ErrNoSlot) {
				t.Fatalf("%v: amulet into the head slot: %v", cls, err)
			}
			if err := lv.PlayerEquip(context.Background(), us[0].ID, d2.BodyNeck); err != nil {
				t.Fatalf("%v: %v", cls, err)
			}
			if b, ok := worn(r.W, us[0].ID); !ok || b != d2.BodyNeck {
				t.Fatalf("%v: not worn", cls)
			}
			better := r.W.Add("amu", d2.ContInventory, 5, 0)
			better.Item.Quality = d2.QualityUnique
			better.Item.Flags |= d2.FlagIdentified
			r.W.Items[len(r.W.Items)-1] = better
			lv.Equip(context.Background())
			_, betterWorn := worn(r.W, better.ID)
			if pinned && betterWorn {
				t.Errorf("%v: the bot swapped a pinned slot", cls)
			}
			if !pinned && !betterWorn {
				t.Errorf("%v: control: the better amulet should replace the old one without a pin", cls)
			}
		}
	}
}


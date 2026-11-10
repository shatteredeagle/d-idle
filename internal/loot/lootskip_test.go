package loot_test

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
)

// lootskip: live log of the Paladin soak (clvl 14, sash, 1 healing in column 0, 2 mana in column 3) shows ground
// healing potions "keep [C] potion already in the pack" although the belt has room.
func TestPaladinSashHealingOnGroundWanted(t *testing.T) {
	l, w := potionRig(t)
	w.Player.Class = d2.Paladin
	w.Player.Level = 14
	w.Add("lbl", d2.ContEquipment, 0, 0)
	for i := range w.Items {
		if w.Items[i].Item.Where == d2.ContEquipment {
			w.Items[i].Item.Body = d2.BodyBelt
		}
	}
	w.Add("hp1", d2.ContBelt, 0, 0)
	w.Add("mp1", d2.ContBelt, 3, 0)
	w.Add("mp1", d2.ContBelt, 3, 1)
	for _, c := range []string{"hp1", "hp2", "mp1"} {
		if !wanted(l, w.NewItem(c, 2)) {
			t.Errorf("%s with room in the belt was skipped", c)
		}
	}
	for _, c := range []string{"vps", "yps"} { // antidote, thawing: never worth a box (SME)
		if wanted(l, w.NewItem(c, 2)) {
			t.Errorf("%s wanted", c)
		}
	}
}

// SME advice (docs/lootskip.md): a small pack buffer beyond a full belt; the default is 2 per kind.
func TestDefaultPackBuffer(t *testing.T) {
	if got := env.DefaultLootTuning().PotionBuffer; got != 2 {
		t.Fatalf("PotionBuffer default = %d, want 2", got)
	}
	for _, cls := range []d2.Class{d2.Paladin, d2.Sorceress} {
		l, w := potionRig(t) // buffer 1 in the rig
		w.Player.Class = cls
		w.Player.Level = 14
		w.Add("lbl", d2.ContEquipment, 0, 0)
		for i := range w.Items {
			if w.Items[i].Item.Where == d2.ContEquipment {
				w.Items[i].Item.Body = d2.BodyBelt
			}
		}
		for b := 0; b < 8; b++ {
			w.Add("hp1", d2.ContBelt, b%4, b/4)
		}
		if !wanted(l, w.NewItem("hp2", 2)) {
			t.Errorf("%v: first spare not wanted", cls)
		}
		w.Add("hp1", d2.ContInventory, 0, 0)
		if wanted(l, w.NewItem("hp2", 2)) {
			t.Errorf("%v: spare wanted beyond the buffer", cls)
		}
	}
}

package loot_test

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/loot"
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
	for _, c := range []string{"hp2", "hp3"} {
		if !wanted(l, w.NewItem(c, 2)) {
			t.Errorf("%s with room in the belt was skipped", c)
		}
	}
	// BELTFIX layout rule: the Paladin plans one mana column and it is full (two boxes), so the game would open a second
	// mana column; a mana potion is left on the ground rather than take a column healing may still need.
	if wanted(l, w.NewItem("mp2", 2)) {
		t.Error("mp2 wanted although the planned mana column is full")
	}
	if wanted(l, w.NewItem("hp1", 2)) {
		t.Error("hp1 wanted below the clvl 14 tier floor")
	}
	for _, c := range []string{"vps", "yps"} { // antidote, thawing: never worth a box (SME)
		if wanted(l, w.NewItem(c, 2)) {
			t.Errorf("%s wanted", c)
		}
	}
}

// SME advice (docs/lootskip.md): a small pack buffer beyond a full belt; the default is 4 healing and 2 mana (docs/levelplan.md).
func TestDefaultPackBuffer(t *testing.T) {
	if got := env.DefaultLootTuning().PotionBuffer; got != 4 {
		t.Fatalf("PotionBuffer default = %d, want 4 (SME: 4 healing in the pack)", got)
	}
	if got := env.DefaultLootTuning().PotionBufferMana; got != 2 {
		t.Fatalf("PotionBufferMana default = %d, want 2", got)
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

// Tier floor data (potionfloor.json): same table for Paladin and Sorceress, Normal 13-25 wants tier 2 and up.
func TestPotionTierFloorTable(t *testing.T) {
	cases := []struct {
		lvl  int
		diff d2.Difficulty
		want int
	}{
		{1, d2.Normal, 1}, {12, d2.Normal, 1}, {13, d2.Normal, 2}, {25, d2.Normal, 2}, {26, d2.Normal, 3},
		{20, d2.Nightmare, 3}, {40, d2.Nightmare, 4}, {5, d2.Hell, 4},
	}
	for _, cls := range []d2.Class{d2.Paladin, d2.Sorceress} {
		for _, c := range cases {
			got, err := loot.PotionTierFloor(cls, c.lvl, c.diff)
			if err != nil || got != c.want {
				t.Errorf("%v clvl %d %v: floor %d err %v, want %d", cls, c.lvl, c.diff, got, err, c.want)
			}
		}
	}
}

func TestPotionBelowFloorLeftWhateverTheBelt(t *testing.T) {
	for _, cls := range []d2.Class{d2.Paladin, d2.Sorceress} {
		l, w := potionRig(t)
		w.Player.Class = cls
		w.Player.Level = 30
		w.Add("lbl", d2.ContEquipment, 0, 0)
		for i := range w.Items {
			if w.Items[i].Item.Where == d2.ContEquipment {
				w.Items[i].Item.Body = d2.BodyBelt
			}
		}
		if wanted(l, w.NewItem("hp2", 2)) || wanted(l, w.NewItem("mp1", 2)) {
			t.Errorf("%v: low tier wanted at clvl 30", cls)
		}
		if !wanted(l, w.NewItem("hp3", 2)) {
			t.Errorf("%v: tier 3 left at clvl 30", cls)
		}
	}
}

// User rule: scrolls only until the tome is full; with no tome the shop target of loose scrolls.
func TestScrollsOnlyUntilTomeFull(t *testing.T) {
	for _, cls := range []d2.Class{d2.Paladin, d2.Sorceress} {
		for _, c := range []struct{ scroll, tome string }{{"tsc", "tbk"}, {"isc", "ibk"}} {
			l, r := newLooter(t)
			w := r.W
			w.Player.Class = cls
			w.Player.Level = 14
			if l.EvaluateUnit(w.NewItem(c.scroll, 2)).Rule != "town scrolls" {
				t.Errorf("%v %s: loose scroll refused with no tome and none carried", cls, c.scroll)
			}
			tb := w.Add(c.tome, d2.ContInventory, 0, 0)
			for _, q := range []struct {
				qty  int32
				want bool
			}{{19, true}, {loot.TomeCap, false}} {
				w.Stats[tb.Addr] = d2.Stats{70: q.qty}
				if got := l.EvaluateUnit(w.NewItem(c.scroll, 2)).Rule == "town scrolls"; got != q.want {
					t.Errorf("%v %s tome qty %d: wanted %v, want %v", cls, c.scroll, q.qty, got, q.want)
				}
			}
			delete(w.Stats, tb.Addr) // empty tome: no stat list at all
			if l.EvaluateUnit(w.NewItem(c.scroll, 2)).Rule != "town scrolls" {
				t.Errorf("%v %s: empty tome refused a scroll", cls, c.scroll)
			}
		}
	}
}

// The healing buffer is its own knob: a pack that holds 2 spare potions is full for mana (2) and not for healing (4).
func TestPackBufferHealingAndManaApart(t *testing.T) {
	for _, cls := range []d2.Class{d2.Paladin, d2.Barbarian, d2.Sorceress} {
		l, w := potionRig(t)
		w.Player.Class = cls
		w.Player.Level = 14
		l.Env().Tune.Loot.PotionBuffer = 4
		l.Env().Tune.Loot.PotionBufferMana = 2
		w.Add("lbl", d2.ContEquipment, 0, 0)
		for i := range w.Items {
			if w.Items[i].Item.Where == d2.ContEquipment {
				w.Items[i].Item.Body = d2.BodyBelt
			}
		}
		for b := 0; b < 8; b++ {
			w.Add("hp1", d2.ContBelt, b%4, b/4)
		}
		w.Add("hp1", d2.ContInventory, 0, 0)
		w.Add("hp1", d2.ContInventory, 1, 0)
		w.Add("mp1", d2.ContInventory, 2, 0)
		w.Add("mp1", d2.ContInventory, 3, 0)
		if !wanted(l, w.NewItem("hp2", 2)) {
			t.Errorf("%v: healing not wanted with 2 of 4 in the pack", cls)
		}
		if wanted(l, w.NewItem("mp2", 2)) {
			t.Errorf("%v: mana wanted with 2 of 2 in the pack", cls)
		}
	}
}

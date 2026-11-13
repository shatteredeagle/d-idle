package loot_test

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/town/sim"
)

// fixpickup scenarios: the potion want follows where game.exe puts the potion (belt first), see docs/fixpickup.md.

func potionRig(t *testing.T) (*loot.Looter, *sim.World) {
	l, r := newLooter(t)
	w := r.W
	w.Player.Class = d2.Barbarian // no mana column in Normal: columns are hp, hp, hp, rv (clvl 40)
	w.Player.Level = 40
	w.Player.Difficulty = d2.Normal
	r.Env.Tune.Loot.PotionBuffer = 1
	r.Env.Tune.Loot.PotionBufferMana = 1
	return l, w
}

func wanted(l *loot.Looter, u d2.Unit) bool { return l.EvaluateUnit(u).Rule == "healing and mana" }

func addBelt(w *sim.World) {
	bu := w.Add("mbl", d2.ContEquipment, 0, 0) // 12-slot belt
	for i := range w.Items {
		if w.Items[i].ID == bu.ID {
			w.Items[i].Item.Body = d2.BodyBelt
		}
	}
}

// Log signature "potion already in the pack" with room in a matching belt column: wanted.
func TestPotionWantedWhenMatchingColumnHasRoom(t *testing.T) {
	l, w := potionRig(t)
	addBelt(w)
	w.Add("hp1", d2.ContBelt, 0, 0)
	w.Add("hp1", d2.ContInventory, 0, 0)
	if !wanted(l, w.NewItem("hp3", 2)) {
		t.Fatal("potion with a matching belt column and a pack potion was skipped")
	}
}

// Belt full for the kind (4-slot belt, hp in every column) and one in the pack: the pack buffer rule skips it.
func TestPotionSkippedByBufferWhenBeltFull(t *testing.T) {
	l, w := potionRig(t)
	for i := 0; i < 4; i++ {
		w.Add("hp1", d2.ContBelt, i, 0)
	}
	w.Add("hp1", d2.ContInventory, 0, 0)
	if wanted(l, w.NewItem("hp3", 2)) {
		t.Fatal("belt full and pack buffer met: potion wanted")
	}
}

// The matching column is full but another column is empty: the potion takes the empty column.
func TestPotionWantedForEmptyColumn(t *testing.T) {
	l, w := potionRig(t)
	w.Add("hp1", d2.ContBelt, 0, 0) // 4-slot belt: column 0 is full, columns 1..3 empty
	w.Add("hp1", d2.ContInventory, 0, 0)
	if !wanted(l, w.NewItem("hp3", 2)) {
		t.Fatal("an empty column is free but the potion was skipped")
	}
}

// Every column is taken and the hp column is full of another kind of box: pack rule.
func TestPotionPackRuleWhenNoColumnFits(t *testing.T) {
	l, w := potionRig(t)
	w.Add("hp1", d2.ContBelt, 0, 0)
	w.Add("rvs", d2.ContBelt, 1, 0)
	w.Add("rvs", d2.ContBelt, 2, 0)
	w.Add("rvs", d2.ContBelt, 3, 0)
	w.Add("hp1", d2.ContInventory, 0, 0)
	if wanted(l, w.NewItem("hp3", 2)) {
		t.Fatal("no column fits but the potion was wanted over the buffer")
	}
}

// DRILL D1: the unit id changes on pickup, the potion sits on the belt: it counts as picked up.
func TestArrivalByClassWhenIDChanges(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	g := w.AddGround("hp1", 2, 6022, 4930)
	s0, _ := w.Snapshot()
	held0 := loot.HeldOf(l, s0.Inventory, g.Class)
	w.Ground = nil
	nb := w.Add("hp1", d2.ContBelt, 0, 0) // a new unit id on the belt
	if nb.ID == g.ID {
		t.Fatal("test needs a different id")
	}
	s, _ := w.Snapshot()
	if !loot.ArrivedFor(l, s, g.ID, g.Class, held0) {
		t.Fatal("potion on the belt under a new unit id not counted as arrived")
	}
	if loot.ArrivedFor(l, s, g.ID, g.Class, held0+1) {
		t.Fatal("no gain in held count but counted as arrived")
	}
}

// FIXBELT2: a Sorceress sash (hp, hp, mp, mp columns) holding healing in columns 0 to 2: a ground healing potion would open
// the last empty column, which the plan keeps for mana, so it is left; a mana potion is wanted.
func TestHealingNotWantedWhenItWouldTakeTheLastManaColumn(t *testing.T) {
	l, w := potionRig(t)
	w.Player.Class = d2.Sorceress
	w.Player.Level = 5
	w.Add("lbl", d2.ContEquipment, 0, 0)
	for i := range w.Items {
		if w.Items[i].Item.Where == d2.ContEquipment {
			w.Items[i].Item.Body = d2.BodyBelt
		}
	}
	for _, b := range []int{0, 4, 1, 5, 2, 6} {
		w.Add("hp1", d2.ContBelt, b, 0)
	}
	w.Add("hp1", d2.ContInventory, 0, 0)
	if wanted(l, w.NewItem("hp1", 2)) {
		t.Fatal("healing potion wanted although it would take the last empty column, kept for mana")
	}
	if !wanted(l, w.NewItem("mp1", 2)) {
		t.Fatal("mana potion with the empty column free was skipped")
	}
}

// BELTFIX (docs/beltfix.md): the sandbox state, mana in three of four columns and no healing. A ground mana potion that the
// game would put in the last empty column is left (it would take the column healing needs); a healing potion is wanted.
// Paladin (three healing columns) and Sorceress (two).
func TestBeltLayoutKeepsTheLastColumnForHealing(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		l, w := potionRig(t)
		w.Player.Class = class
		w.Player.Level = 5
		for c := 0; c < 3; c++ {
			w.Add("mp1", d2.ContBelt, c, 0)
		}
		w.Add("mp1", d2.ContInventory, 0, 0)
		w.Add("mp1", d2.ContInventory, 1, 0)
		if wanted(l, w.NewItem("mp1", 2)) {
			t.Errorf("%v: a mana potion was wanted although it would take the last column", class)
		}
		if !wanted(l, w.NewItem("hp1", 2)) {
			t.Errorf("%v: a healing potion was not wanted with no healing column", class)
		}
	}
}

package plandata

import (
	"strings"
	"testing"
)

var sevenClasses = []string{"amazon", "assassin", "barbarian", "druid", "necromancer", "paladin", "sorceress"}

func TestChoresAllSevenClasses(t *testing.T) {
	for _, c := range sevenClasses {
		if _, err := Chores(c); err != nil {
			t.Errorf("%s: %v", c, err)
		}
	}
	data, _ := FS.ReadFile("chores-defaults.json")
	tab, _ := ParseChores(data)
	if len(tab.Classes) != len(sevenClasses) {
		t.Errorf("%d classes, want %d", len(tab.Classes), len(sevenClasses))
	}
}

// Melee with no mana potions against a mana class, and the ammo trigger only for the Amazon bow.
func TestChoresTwoClasses(t *testing.T) {
	bar, _ := Chores("barbarian")
	sor, _ := Chores("sorceress")
	if bar.Potions.Mana != 0 || sor.Potions.Mana == 0 {
		t.Errorf("mana: barbarian %d, sorceress %d", bar.Potions.Mana, sor.Potions.Mana)
	}
	if bar.Repair.Cadence != "every_trip" || sor.Repair.Cadence != "rare" {
		t.Errorf("repair: %q / %q", bar.Repair.Cadence, sor.Repair.Cadence)
	}
	for _, c := range sevenClasses {
		r, _ := Chores(c)
		if (r.Ammo != nil) != (c == "amazon") {
			t.Errorf("%s: ammo row %v", c, r.Ammo)
		}
	}
	if _, err := Chores("monk"); err == nil || !strings.Contains(err.Error(), `"monk"`) {
		t.Errorf("unknown class error: %v", err)
	}
}

func TestChoresErrorsNameTheCause(t *testing.T) {
	for _, c := range []struct{ name, json, want string }{
		{"unknown field", `{"version":1,"classes":{"a":{"bogus":1}}}`, "bogus"},
		{"bad cadence", `{"version":1,"classes":{"a":{"potions":{"health":1,"mana":0},"repair":{"cadence":"daily"}}}}`, `"daily"`},
		{"empty mix", `{"version":1,"classes":{"a":{"potions":{"health":0,"mana":0},"repair":{"cadence":"rare"}}}}`, "potion mix"},
		{"version", `{"version":2,"classes":{}}`, "version 2"},
	} {
		if _, err := ParseChores([]byte(c.json)); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
		}
	}
}

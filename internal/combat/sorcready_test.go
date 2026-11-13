package combat

import (
	"testing"

	"d2lod-bot/internal/d2"
)

// Only profiles that name a skill in "avoid" lose it; every early (clvl 1..12) band of the Sorceress has a spell
// the fresh character can cast, and the Paladin (a second class) is untouched.
func TestAvoidListIsPerProfile(t *testing.T) {
	all := func(string) bool { return true }
	for _, p := range Profiles() {
		if p.Class == "sorceress" && (p.Build == "lightning" || p.Build == "cold") {
			if len(p.Avoid) != 2 {
				t.Errorf("lightning avoid %v", p.Avoid)
			}
			continue
		}
		if len(p.Avoid) != 0 {
			t.Errorf("%s/%s avoids %v", p.Class, p.Build, p.Avoid)
		}
	}
	sor := ProfileFor(d2.Sorceress, BuildLightning)
	for lvl := 1; lvl <= 12; lvl++ {
		if o := sor.AttackNames(lvl, func(n string) bool { return n == "Charged Bolt" }); len(o) == 0 || o[0] != "Charged Bolt" {
			t.Errorf("clvl %d: %v", lvl, o)
		}
	}
	if o := ProfileFor(d2.Paladin, BuildBlizzard).AttackNames(30, all); o[0] != "Blessed Hammer" {
		t.Errorf("paladin %v", o)
	}
}

// sorcready2: the cold profile leads with Ice Blast once it is learned, never ranks the broken lightning skills, and
// falls back to Charged Bolt / Ice Bolt before it; the Paladin profile is unchanged.
func TestColdProfileBands(t *testing.T) {
	cold := ProfileFor(d2.Sorceress, Build("cold"))
	learned := func(names ...string) func(string) bool {
		return func(n string) bool {
			for _, x := range names {
				if x == n {
					return true
				}
			}
			return false
		}
	}
	for lvl := 13; lvl <= 30; lvl++ {
		o := cold.AttackNames(lvl, learned("Charged Bolt", "Ice Bolt", "Ice Blast", "Lightning", "Chain Lightning"))
		if len(o) == 0 || o[0] != "Ice Blast" {
			t.Errorf("clvl %d: %v", lvl, o)
		}
		for _, n := range o {
			if n == "Lightning" || n == "Chain Lightning" {
				t.Errorf("clvl %d ranks %s", lvl, n)
			}
		}
	}
	if o := cold.AttackNames(8, learned("Charged Bolt")); len(o) == 0 || o[0] != "Charged Bolt" {
		t.Errorf("before Ice Blast: %v", o)
	}
	if o := ProfileFor(d2.Paladin, BuildBlizzard).AttackNames(15, learned("Zeal")); o[0] != "Zeal" {
		t.Errorf("paladin %v", o)
	}
}

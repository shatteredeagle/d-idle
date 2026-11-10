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
		if p.Class == "sorceress" && p.Build == "lightning" {
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

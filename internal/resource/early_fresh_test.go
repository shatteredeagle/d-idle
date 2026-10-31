package resource

import (
	"testing"

	"d2lod-bot/internal/d2"
)

// EARLYGAME: a fresh character of every class (starter weapon, no body armor, 4 minor healing potions, no gold: U, a new
// character's gold is not read anywhere) must never be gated at clvl 1..14 in Normal Act 1, and a class that needs a
// weapon is gated at once without one (docs/earlygame.md).
func TestFreshCharacterIsNeverGatedEarly(t *testing.T) {
	classes := []d2.Class{d2.Sorceress, d2.Necromancer, d2.Paladin, d2.Amazon, d2.Barbarian, d2.Druid, d2.Assassin}
	caster := map[d2.Class]bool{d2.Sorceress: true, d2.Necromancer: true, d2.Druid: true}
	tn := DefaultGearTune()
	for _, c := range classes {
		for lvl := 1; lvl <= 14; lvl++ {
			in := GearInput{Level: lvl, Gold: 0, Potions: 4, Weapon: true, Caster: caster[c], NeedWeapon: NeedsWeapon(c), Act: 1, MaxLife: 40 + 6*lvl}
			if v := GearGate(in, 150, tn); v.Gated || v.Risk != RiskLow {
				t.Errorf("%s clvl %d fresh kit: %+v", c, lvl, v)
			}
			// the potions are gone and the pockets empty: from clvl 5 this is the starvation case (safe farm area only)
			in.Potions = 0
			v := GearGate(in, 150, tn)
			if lvl < GearGateLevel && v.Gated {
				t.Errorf("%s clvl %d without potions is gated below the poverty level: %+v", c, lvl, v)
			}
			if lvl >= GearGateLevel && !v.Gated {
				t.Errorf("%s clvl %d no potions and no gold must be gated: %+v", c, lvl, v)
			}
			// no weapon: only the classes in the needed-weapon table are gated
			in = GearInput{Level: lvl, Gold: 0, Potions: 4, Weapon: false, Caster: caster[c], NeedWeapon: NeedsWeapon(c), Act: 1, MaxLife: 40 + 6*lvl}
			if v := GearGate(in, 150, tn); v.Gated != NeedsWeapon(c) {
				t.Errorf("%s clvl %d no weapon: gated=%v want %v (%+v)", c, lvl, v.Gated, NeedsWeapon(c), v)
			}
		}
	}
}

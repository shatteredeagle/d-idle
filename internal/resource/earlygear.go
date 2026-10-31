package resource

import "d2lod-bot/internal/d2"

// DefaultEarlyGearMax is the character level below which armor does not matter (docs/sme/early-gear-policy.md, SRC:
// d2-sme veteran and the user, both experienced D2 players): wear what drops, never trip or gate for it.
const DefaultEarlyGearMax = 15

// neededWeapon is the per-class "needed weapon" table for clvl 1-15: the classes whose skills run on a weapon. A missing
// weapon of these is a rebuy reason and a high risk; Sorceress, Necromancer and Druid need nothing (a wand or shield
// is worn if one drops). Amazon needs a bow, crossbow or javelins, the melee classes any usable melee weapon (claws or
// katar for the Assassin).
var neededWeapon = map[d2.Class]bool{
	d2.Amazon: true, d2.Paladin: true, d2.Barbarian: true, d2.Assassin: true,
}

// NeedsWeapon says whether the class cannot play the early game without a weapon.
func NeedsWeapon(c d2.Class) bool { return neededWeapon[c] }

// Early reports whether level is below the early-gear limit max (max <= 0 disables the policy).
func Early(level, max int) bool { return level < max }

package leveler

import "strings"

// Prefs is the per-class weapon preference table the gear score reads (replaces the old hard-coded Sorceress
// "+8 for staff, wand, orb"). Everything is in the score's defence-equivalents (score.go).
type Prefs struct {
	// DmgWeight multiplies a weapon's average base damage (0.4 for a caster whose weapon barely matters).
	DmgWeight float64
	// TypeBonus is a flat bonus by item type code (any ancestor type matches): casters like staves, wands and orbs.
	TypeBonus map[string]float64
	// TwoHandMult multiplies the score of a two-handed weapon (1 = no preference).
	TwoHandMult float64
	// WantsShield: the build fights with a shield, so a two-handed weapon is never planned while a shield is worn or
	// available (ChooseGear), and rebuy never buys one (town/rebuy.go).
	WantsShield bool
}

// prefsByClass keys are lower-case class names (d2.Class.String()). Sorceress values equal the old constants.
// Tags: Sorceress V (behaviour kept identical, score_test.go). Paladin U: one-hand weapon plus shield is the Paladin
// plan of docs/classes/paladin.md (Holy Shield needs a shield, Zeal and Hammer work with any one-hander); the weights
// are a judgement, not measured.
var prefsByClass = map[string]Prefs{
	"sorceress": {DmgWeight: wWeaponDmg, TypeBonus: map[string]float64{"staf": wCasterBase, "wand": wCasterBase, "orb": wCasterBase}, TwoHandMult: 1},
	"paladin":   {DmgWeight: 0.8, TwoHandMult: 0.5, WantsShield: true},
}

// PrefsFor returns the preference table of a class name; an unknown class gets the Sorceress table (the previous
// behaviour for every class).
func PrefsFor(class string) Prefs {
	if p, ok := prefsByClass[strings.ToLower(class)]; ok {
		return p
	}
	return prefsByClass["sorceress"]
}

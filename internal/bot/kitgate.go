package bot

import (
	"fmt"

	"d2lod-bot/internal/d2"
)

// KITGATE (docs/kitgate.md): a recovering character that lacks the kit (no healing potion in the belt, or no town
// portal scroll) farms only the open fields next to town and keeps its walks out of the dungeon areas. The kit is the
// one resource.KitShortfall prices: a portal scroll plus the healing potions of the belt plan.

// kitSafeTier is the farm list of a recovering character without the kit: Blood Moor and Cold Plains, the areas whose way
// home is a short walk (SME rule of thumb EXP; narrower than safeTier, which still holds the Den and the caves).
var kitSafeTier = map[int]bool{2: true, 3: true}

// dungeonMouths are the act 1 areas a kit-less recovery must not enter or route through: the Den of Evil, Cave Levels
// 1 and 2, Burial Grounds (Blood Raven), the Crypt and the Mausoleum.
var dungeonMouths = map[int]bool{areaDenOfEvil: true, 9: true, 13: true, 17: true, 18: true, 19: true}

// kitMissing reports a belt without a healing (or rejuvenation) potion or a pack without a town portal scroll. An
// unknown belt (no catalog or inventory yet) is not missing: the old behaviour stands until the kit can be read.
func (r *Runner) kitMissing(s *d2.Snapshot) bool {
	_, _, known := r.beltAndInv(s)
	if !known {
		return false
	}
	in := r.resInput(s)
	return in.Belt.HP+in.Belt.RV < 1 || !in.HasTP
}

// kitGateActive is the gate: knob on, recovery running, kit missing, Normal act 1 (where the tables above apply).
func (r *Runner) kitGateActive(s *d2.Snapshot) bool {
	return r.e.Tune.Resource.RecoveryKitGate != 0 && r.recoveryActive() && s != nil && s.Player != nil &&
		difficultyName(s.Player.Difficulty) == "normal" && s.Player.Act == 0 && r.kitMissing(s)
}

// kitAvoider is the part of a Mover that keeps walks out of levels (nav.Navigator.SetAvoidLevels).
type kitAvoider interface{ SetAvoidLevels(map[int]bool) }

// syncKitGate hands the dungeon list to the navigator while the gate is active and clears it after; it logs the edges.
func (r *Runner) syncKitGate(s *d2.Snapshot) {
	b := &r.loop.broke
	on := r.kitGateActive(s)
	if on == b.kitGate {
		return
	}
	b.kitGate = on
	if a, ok := r.d.Mover.(kitAvoider); ok {
		if on {
			a.SetAvoidLevels(dungeonMouths)
		} else {
			a.SetAvoidLevels(nil)
		}
	}
	r.e.Log.Info(fmt.Sprintf("recovery: kit gate %s", map[bool]string{true: "on (no potion or no portal): Blood Moor and Cold Plains only", false: "off"}[on]), "gold", s.Player.Gold)
}

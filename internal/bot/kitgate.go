package bot

import (
	"fmt"
	"strings"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/resource"
)

// KITGATE (docs/kitgate.md): a recovering character that lacks the kit (no healing potion in the belt, or no town
// portal scroll) farms only the open fields next to town and keeps its walks out of the dungeon areas. The kit is the
// one resource.KitShortfall prices (clvl 1 to 7; from KitGateLevel see kitTrip): a portal scroll plus the healing potions of the belt plan.

// kitSafeTier is the farm list of a recovering character without the kit: Blood Moor and Cold Plains, the areas whose way
// home is a short walk (SME rule of thumb EXP; narrower than safeTier, which still holds the Den and the caves).
var kitSafeTier = map[int]bool{2: true, 3: true}

// dungeonMouths are the act 1 areas a kit-less recovery must not enter or route through: the Den of Evil, Cave Levels
// 1 and 2, Burial Grounds (Blood Raven), the Crypt and the Mausoleum.
var dungeonMouths = map[int]bool{areaDenOfEvil: true, 9: true, 13: true, 17: true, 18: true, 19: true}

// kit is the minimum kit read from the pack in one place (KITGATELEVEL, docs/kitgate.md): a portal source, healing
// potions (belt, pack and rejuvenation) and, for a build that needs mana, mana potions in the belt. The minimums are the
// knobs resource.kitMinHP and resource.kitMinMP; rejuvenation and identify scrolls are not required.
type kit struct {
	TP     bool
	HP, MP int
	minHP  int
	minMP  int // 0 for a build that needs no mana (resource.ManaNeed), so no class is named here
}

// kitRead reads the kit; known is false before the belt and pack can be read.
func (r *Runner) kitRead(s *d2.Snapshot) (k kit, known bool) {
	if _, _, known = r.beltAndInv(s); !known {
		return k, false
	}
	in := r.resInput(s)
	rt := r.e.Tune.Resource
	k = kit{TP: in.HasTP, HP: in.Belt.HP + in.Belt.RV + in.Belt.PackHP + in.Belt.PackRV, MP: in.Belt.MP, minHP: rt.KitMinHP}
	if resource.ManaNeed(in.Class) >= 2 {
		k.minMP = rt.KitMinMP
	}
	return k, true
}

// missing lists what the kit lacks against the minimum, empty when complete.
func (k kit) missing() []string {
	var m []string
	if !k.TP {
		m = append(m, "town portal")
	}
	if k.HP < k.minHP {
		m = append(m, fmt.Sprintf("healing potions %d/%d", k.HP, k.minHP))
	}
	if k.MP < k.minMP {
		m = append(m, fmt.Sprintf("mana potions %d/%d", k.MP, k.minMP))
	}
	return m
}

// kitMissing is the old (clvl 1 to 7) gate condition: no healing (or rejuvenation) potion in the belt or the pack, or a
// pack without a town portal scroll. An unknown belt is not missing: the old behaviour stands until the kit can be read.
func (r *Runner) kitMissing(s *d2.Snapshot) bool {
	k, known := r.kitRead(s)
	// F6: healing potions waiting in the pack count; the belt fill (town chores and the field step) belts them
	return known && (k.HP < 1 || !k.TP)
}

// kitBandOpen reports a character at or above KitGateLevel: the band where an incomplete kit is a town trip, not a hold.
func (r *Runner) kitBandOpen(s *d2.Snapshot) bool {
	lv := r.e.Tune.Resource.KitGateLevel
	return lv > 0 && s != nil && s.Player != nil && s.Player.Level >= lv
}

// kitHolds is the hold condition: below KitGateLevel the old one; from it only a character that can neither flee nor heal
// (no portal source and no healing potion), the one case a trip cannot be made safely.
func (r *Runner) kitHolds(s *d2.Snapshot) bool {
	if !r.kitBandOpen(s) {
		return r.kitMissing(s)
	}
	k, known := r.kitRead(s)
	return known && !k.TP && k.HP < 1
}

// kitGateActive is the gate: knob on, recovery running, the hold condition, Normal act 1 (where the tables above apply).
func (r *Runner) kitGateActive(s *d2.Snapshot) bool {
	return r.e.Tune.Resource.RecoveryKitGate != 0 && r.recoveryActive() && s != nil && s.Player != nil &&
		difficultyName(s.Player.Difficulty) == "normal" && s.Player.Act == 0 && r.kitHolds(s)
}

// kitTrip decides, from KitGateLevel up, whether an incomplete kit makes a town trip now. It returns the reason for the
// chore plan. The trip needs a way home (portal source, or a walkable field) and gold for the cheapest missing item
// (otherwise the sell and poverty logic of the chores bring the gold first and the character goes on). After KitTripCap
// kit trips with the kit still incomplete the bot proceeds anyway and logs kitgate-skip once.
func (r *Runner) kitTrip(s *d2.Snapshot) (string, bool) {
	rt := r.e.Tune.Resource
	if !r.kitBandOpen(s) || s.Player.InTown() || difficultyName(s.Player.Difficulty) != "normal" || s.Player.Act != 0 {
		return "", false
	}
	k, known := r.kitRead(s)
	if !known {
		return "", false
	}
	miss := k.missing()
	r.obMu.Lock()
	defer r.obMu.Unlock()
	if len(miss) == 0 {
		r.ob.kitTrips, r.ob.kitSkipped = 0, false
		return "", false
	}
	if r.ob.kitTrips >= rt.KitTripCap {
		if !r.ob.kitSkipped {
			r.ob.kitSkipped = true
			r.e.Log.Info("kitgate-skip", "trips", r.ob.kitTrips, "missing", strings.Join(miss, ", "), "gold", s.Player.Gold, "level", s.Player.Level)
		}
		return "", false
	}
	// The cheapest thing the visit can buy: a missing portal scroll is bought first only when it is affordable
	// (Town.buyFirstTP), so a character short of healing potions needs one potion's gold to make the walk home worth
	// it. Pricing the scroll kept a 37-gold clvl 12 Sorceress without potions in the field for 12 virtual minutes
	// (Gate 1 20261102, docs/research/beltfix-analysis.md).
	price := rt.KitPotionGold
	if !k.TP && k.HP >= k.minHP {
		price = rt.KitTPGold
	}
	if s.Player.Gold < price || !(k.TP || resource.NearTownArea(s.Player.Area) || k.HP == 0 && resource.ChainHomeArea(s.Player.Area) || r.wpActive(s)) {
		return "", false
	}
	return "kit: " + strings.Join(miss, ", "), true
}

// wpActive reports an activated waypoint in the current area: Town.ToTown without a scroll takes it to the town
// (waypointToTown), so the way home works from fields beyond the walkable ones (Stony Field, Gate 1 20261102 BELT window).
func (r *Runner) wpActive(s *d2.Snapshot) bool {
	if r.d.Task == nil || r.d.Task.Quests == nil {
		return false
	}
	v, ok := r.d.Task.Quests.Read(int(s.Player.Difficulty))
	return ok && v.WPs[s.Player.Area]
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

// potionDiag is the minute line's view of where the potions are (primitives-review "what to log" 4): the pack's healing
// and mana potions and the belt as box:code, so an empty-belt minute splits into "nothing anywhere" and "in the pack or
// behind another kind" without inference.
func (r *Runner) potionDiag(s *d2.Snapshot) (packHP, packMP int, boxes string) {
	if r.d.Catalog == nil || s == nil || s.Inventory == nil {
		return 0, 0, ""
	}
	for i := range s.Inventory.Items {
		u := &s.Inventory.Items[i]
		if u.Item == nil {
			continue
		}
		m, ok := r.d.Catalog.Item(u.Class)
		if !ok || m.Potion == PotionNone {
			continue
		}
		switch u.Item.Where {
		case d2.ContInventory:
			switch m.Potion {
			case PotionHealing:
				packHP++
			case PotionMana:
				packMP++
			}
		case d2.ContBelt:
			if boxes != "" {
				boxes += " "
			}
			boxes += fmt.Sprintf("%d:%s", u.Item.GY*4+u.Item.GX, strings.ReplaceAll(m.Name, " ", "_"))
		}
	}
	return packHP, packMP, boxes
}

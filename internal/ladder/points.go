package ladder

import (
	"fmt"

	"d2lod-bot/internal/decide"
)

// Wired points (docs/t0-slice.md). Each helper takes the answer the legacy code gives at the call site and returns the
// action to take: the legacy one until a person teaches a rule for the context. Every other point stays on its legacy
// path.
const (
	PointRepair  = "t1.repair"
	PointTPLost  = "t1.tp.lost"
	PointReturn  = "t1.return"
	PointTrigger = "t1.town.trigger"
	PointRestock = "t1.restock"

	PointDrink   = "t0.retreat.drink"
	PointTooHard = "t0.area.toohard"
	PointUnique  = "t0.unique.fight"
	PointMerc    = "t2.mercenary"
)

// Kit values of the t1.return key (docs/research/decision-point-sme-review.md).
const (
	KitOK, KitThin, KitNone = "OK", "THIN", "NONE"
)

// KitFor classes the supplies against the key of the T0 points: NONE with no portal and no healing potion, THIN with no
// portal or under two healing potions, else OK. One definition for every site (U, simplified from the SME review, which
// also counts gold).
func KitFor(hasPortal bool, healing int) string {
	switch {
	case !hasPortal && healing <= 0:
		return KitNone
	case !hasPortal || healing < 2:
		return KitThin
	}
	return KitOK
}

// Held values of the t0.retreat.drink key: whether a healing or rejuvenation potion is in the belt.
const (
	HeldNone, HeldHP, HeldRV = "none", "hp", "rv"
)

// HeldFor classes the belt for the t0.retreat.drink key.
func HeldFor(hp, rv int) string {
	switch {
	case hp > 0:
		return HeldHP
	case rv > 0:
		return HeldRV
	}
	return HeldNone
}

// Belt values of the t1.town.trigger and t1.restock keys.
const (
	BeltFull, BeltHalf, BeltLow = "full", "half", "low"
)

// BeltBand classes healing potions held against the plan's capacity for them: full at or above it (or no plan), low
// under half of it, else half. The one definition both belt-keyed points use.
func BeltBand(have, want int) string {
	switch {
	case want <= 0 || have >= want:
		return BeltFull
	case have*2 < want:
		return BeltLow
	}
	return BeltHalf
}

// RepairWanted answers t1.repair: repair now at this lowest worn durability (percent)? legacy is the knob test. The
// other legacy guards (latch, gold for the estimate) stay with the caller.
func (l *Ladder) RepairWanted(h Hero, durabilityPct int, legacy bool) (bool, error) {
	h.Durability = max(durabilityPct, 0)
	return l.yes(PointRepair, h, decide.Obs{Sit: decide.Situation{"durability": fmt.Sprint(h.Durability)}}, legacy, "repair", "skip")
}

// TPBuyWanted answers t1.tp.lost: buy a town portal scroll or walk. legacy is the purchase test (BuyKit); the caller
// still needs the gold for the price.
func (l *Ladder) TPBuyWanted(h Hero, legacy bool) (bool, error) {
	return l.yes(PointTPLost, h, decide.Obs{}, legacy, "buy", "walk")
}

// ReturnWanted answers t1.return: leave town for the field now. legacy is the field gate's Go.
func (l *Ladder) ReturnWanted(h Hero, kit string, legacy bool) (bool, error) {
	// the area class of this key is where the hero goes, not where it stands
	return l.yes(PointReturn, h, decide.Obs{Cat: map[string]string{"kit": kit, "areaclass": "field"}, Sit: decide.Situation{"kit": kit}}, legacy, "yes", "no")
}

// TripWanted answers t1.town.trigger: go to town now. legacy is the resource policy's Go. packUsedPct is the share of
// backpack cells in use.
func (l *Ladder) TripWanted(h Hero, belt string, packUsedPct int, legacy bool) (bool, error) {
	o := decide.Obs{Cat: map[string]string{"belt": belt}, Num: map[string]float64{"packused": float64(packUsedPct)},
		Sit: decide.Situation{"packused": fmt.Sprint(packUsedPct)}}
	return l.yes(PointTrigger, h, o, legacy, "go", "continue")
}

// RestockWanted answers t1.restock: spend this gold on the belt potions. legacy is "the trip budget is not spent on
// gold" (gold at or above the trip gold floor).
func (l *Ladder) RestockWanted(h Hero, belt string, tripGold int, legacy bool) (bool, error) {
	o := decide.Obs{Cat: map[string]string{"belt": belt}, Num: map[string]float64{"tripgold": float64(tripGold)},
		Sit: decide.Situation{"tripgold": fmt.Sprint(tripGold)}}
	return l.yes(PointRestock, h, o, legacy, "yes", "no")
}

// yes asks the point with the legacy answer and reports whether the answer is the positive one.
func (l *Ladder) yes(point string, h Hero, extra decide.Obs, legacy bool, pos, neg string) (bool, error) {
	leg := neg
	if legacy {
		leg = pos
	}
	a, err := l.Decide(point, h, extra, leg)
	if err != nil {
		return false, err
	}
	if a != pos && a != neg {
		return false, fmt.Errorf("ladder: %s: %q is not an answer", point, a)
	}
	return a == pos, nil
}

// DrinkChoice answers t0.retreat.drink: what to do at this life percent with this belt. legacy is one of drink, run,
// portal, leave, fight. champion is the champion-pack early bonus (resource.ChampionEarly). Hardcore comes from h.
func (l *Ladder) DrinkChoice(h Hero, lifePct int, held, kit string, champion bool, legacy string) (string, error) {
	o := decide.Obs{Num: map[string]float64{"life": float64(lifePct)}, Cat: map[string]string{"held": held, "kit": kit},
		Sit: decide.Situation{"life": fmt.Sprint(lifePct), "kit": kit}}
	if champion {
		o.Cat["champbonus"] = "early"
		o.Sit["champion"] = "yes"
	}
	return l.Decide(PointDrink, h, o, legacy)
}

// UniqueFight answers t0.unique.fight: fight or avoid an elite pack at this life percent. legacy is fight or avoid.
// packDanger is the pack rating 0..3 (0 counts as the lowest class).
func (l *Ladder) UniqueFight(h Hero, lifePct, packDanger int, kit, legacy string) (string, error) {
	o := decide.Obs{Num: map[string]float64{"life": float64(lifePct)},
		Cat: map[string]string{"kit": kit, "packdanger": fmt.Sprint(min(max(packDanger, 1), 3))},
		Sit: decide.Situation{"life": fmt.Sprint(lifePct), "kit": kit}}
	return l.Decide(PointUnique, h, o, legacy)
}

// TooHard answers t0.area.toohard for an area the hero is about to enter: go, skip or lower. merc is
// task.MercState (alive, dead or none); legacy is go when the route guard is open, else lower with no kit, else skip.
func (l *Ladder) TooHard(h Hero, clvl int, merc, kit, legacy string) (string, error) {
	o := decide.Obs{Num: map[string]float64{"clvl": float64(clvl)}, Cat: map[string]string{"merc": merc, "kit": kit, "areaclass": "field"},
		Sit: decide.Situation{"merc": merc, "kit": kit, "herolevel": fmt.Sprint(clvl)}}
	return l.Decide(PointTooHard, h, o, legacy)
}

// MercHire answers t2.mercenary: hire or revive now (true) or wait for gold. h.Gold is the gold held; legacy is the
// bot's hire gate (gold at or above bot.mercHireMinGold).
func (l *Ladder) MercHire(h Hero, merc, kit string, legacy bool) (bool, error) {
	return l.yes(PointMerc, h, decide.Obs{Cat: map[string]string{"merc": merc, "kit": kit}, Sit: decide.Situation{"merc": merc}}, legacy, "hire", "wait")
}

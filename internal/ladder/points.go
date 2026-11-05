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
)

// Kit values of the t1.return key (docs/research/decision-point-sme-review.md).
const (
	KitOK, KitThin, KitNone = "OK", "THIN", "NONE"
)

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
	return l.yes(PointRepair, h, decide.Obs{}, legacy, "repair", "skip")
}

// TPBuyWanted answers t1.tp.lost: buy a town portal scroll or walk. legacy is the purchase test (BuyKit); the caller
// still needs the gold for the price.
func (l *Ladder) TPBuyWanted(h Hero, legacy bool) (bool, error) {
	return l.yes(PointTPLost, h, decide.Obs{}, legacy, "buy", "walk")
}

// ReturnWanted answers t1.return: leave town for the field now. legacy is the field gate's Go.
func (l *Ladder) ReturnWanted(h Hero, kit string, legacy bool) (bool, error) {
	// the area class of this key is where the hero goes, not where it stands
	return l.yes(PointReturn, h, decide.Obs{Cat: map[string]string{"kit": kit, "areaclass": "field"}}, legacy, "yes", "no")
}

// TripWanted answers t1.town.trigger: go to town now. legacy is the resource policy's Go. packUsedPct is the share of
// backpack cells in use.
func (l *Ladder) TripWanted(h Hero, belt string, packUsedPct int, legacy bool) (bool, error) {
	o := decide.Obs{Cat: map[string]string{"belt": belt}, Num: map[string]float64{"packused": float64(packUsedPct)}}
	return l.yes(PointTrigger, h, o, legacy, "go", "continue")
}

// RestockWanted answers t1.restock: spend this gold on the belt potions. legacy is "the trip budget is not spent on
// gold" (gold at or above the trip gold floor).
func (l *Ladder) RestockWanted(h Hero, belt string, tripGold int, legacy bool) (bool, error) {
	o := decide.Obs{Cat: map[string]string{"belt": belt}, Num: map[string]float64{"tripgold": float64(tripGold)}}
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

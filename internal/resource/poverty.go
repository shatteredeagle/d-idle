package resource

import "fmt"

// KitCost is the gold of the minimal survival kit: one town portal scroll and HPFirst healing potions.
func KitCost(t Tuning) int { return t.KitTPGold + t.HPFirst*t.KitPotionGold }

// Poverty is the verdict of PovertyCheck.
type Poverty struct {
	On      bool
	Reason  string // why poverty mode is on, or why it ended
	Missing int    // gold still missing for the kit (0 when affordable)
}

// PovertyCheck decides whether the character is too poor to take the next leveling step. Poverty starts when the kit
// is incomplete (no town portal or fewer than HPFirst healing potions) and gold is below KitCost; once on it stays on
// until gold reaches KitCost+PovertyMargin or the kit is complete (hysteresis, so a sale that barely reaches the cost
// does not flap).
func PovertyCheck(in Input, t Tuning, was bool) Poverty {
	cost := KitCost(t)
	incomplete := !in.HasTP || in.Belt.HP+in.Belt.RV < t.HPFirst
	miss := max(cost-in.Gold, 0)
	switch {
	case !incomplete:
		return Poverty{Reason: "kit complete"}
	case was && in.Gold >= cost+t.PovertyMargin:
		return Poverty{Reason: fmt.Sprintf("gold %d covers the kit %d", in.Gold, cost)}
	case !was && in.Gold >= cost:
		return Poverty{Reason: "kit affordable"}
	}
	why := fmt.Sprintf("gold %d below kit cost %d", in.Gold, cost)
	if !in.HasTP {
		why += ", no town portal"
	}
	if in.Belt.HP+in.Belt.RV < t.HPFirst {
		why += fmt.Sprintf(", %d healing potions", in.Belt.HP+in.Belt.RV)
	}
	return Poverty{On: true, Reason: why, Missing: miss}
}

// KitBuy is how to spend gold on the minimal kit: the town portal scroll first (it is the way out of a bad fight and
// the way home to spend the rest), then healing potions, as many as the gold buys up to HPFirst.
type KitBuy struct {
	TP bool
	HP int
}

func BuyKit(in Input, t Tuning) KitBuy {
	gold := in.Gold
	var k KitBuy
	if !in.HasTP && gold >= t.KitTPGold {
		k.TP = true
		gold -= t.KitTPGold
	}
	if t.KitPotionGold > 0 {
		k.HP = min(max(t.HPFirst-in.Belt.HP-in.Belt.RV, 0), gold/t.KitPotionGold)
	}
	return k
}

// PovertyAction is what a poor character does next.
type PovertyAction int

const (
	PovertyNone PovertyAction = iota // not poor: normal play
	SellTrip                         // go to town and sell the loot: it is the only gold source that does not need gold
	PlayLean                         // no sale worth a trip: keep farming without potions, cautiously
)

func (a PovertyAction) String() string { return [...]string{"none", "sell-trip", "play-lean"}[a] }

// PovertyPlan is the verdict of PovertyAdvice.
type PovertyPlan struct {
	Action        PovertyAction
	MaxPackDanger int // PlayLean: do not engage packs above this danger (no potion to save a bad fight)
	Reason        string
}

// PovertyAdvice turns a poverty verdict into a plan. A poor character with at least sellMin sellable items makes a sell
// trip (needs a portal or a short walk), but at most twice in a row without gold coming out of it (IdleTrips); otherwise
// it plays lean: potion-free, trash and champion-free packs only, never a town trip for a potion it cannot pay (the
// flee<->town loop). Heal-only trips are handled by TownTripTrigger.
func PovertyAdvice(in Input, p Poverty, sellMin int) PovertyPlan {
	if !p.On {
		return PovertyPlan{Reason: p.Reason}
	}
	if in.Sellable >= sellMin && in.IdleTrips < 2 && (in.HasTP || in.NearTown || in.InTown) {
		return PovertyPlan{Action: SellTrip, MaxPackDanger: 0, Reason: fmt.Sprintf("poor (%s), %d items to sell", p.Reason, in.Sellable)}
	}
	return PovertyPlan{Action: PlayLean, MaxPackDanger: 0, Reason: fmt.Sprintf("poor (%s), nothing worth a sell trip", p.Reason)}
}

// KitShortfall is the gold the survival kit still needs before any gear is bought: a town portal scroll when the pack
// holds none, plus the healing and mana potions the belt plan (HPFirst, MPFirst) lacks. The gear shop keeps this
// reserve so the kit is bought first (soak WP2: a 1175 gold gear visit left an empty belt five minutes later).
func KitShortfall(hasTP bool, hp, mp int, t Tuning) int {
	g := max(t.HPFirst-hp, 0)*t.KitPotionGold + max(t.MPFirst-mp, 0)*t.KitPotionGold
	if !hasTP {
		g += t.KitTPGold
	}
	return g
}

// GearBudget caps a gear-shop budget so that gold minus the kit shortfall is never spent on gear.
func GearBudget(budget, gold, shortfall int) int {
	return max(min(budget, gold-shortfall), 0)
}

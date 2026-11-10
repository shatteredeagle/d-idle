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
	healing := in.Belt.HP + in.Belt.RV
	incomplete := !in.HasTP || healing < t.HPFirst
	if !was {
		// Entry needs an empty healing belt (or no portal): one potion drunk below HPFirst is the normal run of a
		// fight, not poverty. Exit still needs the full HPFirst, so the two edges are a band, not one line (the 53
		// flips of the 11.7 h sandbox log came from one potion more or less).
		incomplete = !in.HasTP || healing < min(t.HPFirst, 1)
	}
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
	if healing < t.HPFirst {
		why += fmt.Sprintf(", %d healing potions", healing)
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

// Saving is what a poor character is saving for: the first thing of the must-have order it cannot afford yet.
type Saving struct {
	Item string // "a town portal scroll", "healing potions", "the repair"; "" = nothing is missing
	Need int    // gold still missing for Item
}

// SavingFor walks the must-have order (town portal scroll, healing potions up to HPFirst, a requested repair of
// repairNeed gold; 0 = none) and names the first one the gold does not cover. The order is the one BuyKit and the repair
// gate spend in, so the whyline says what the next coin goes to.
func SavingFor(in Input, t Tuning, repairNeed int) Saving {
	gold := in.Gold
	if !in.HasTP {
		if gold < t.KitTPGold {
			return Saving{"a town portal scroll", t.KitTPGold - gold}
		}
		gold -= t.KitTPGold
	}
	if n := max(t.HPFirst-in.Belt.HP-in.Belt.RV, 0); n > 0 && t.KitPotionGold > 0 {
		if c := n * t.KitPotionGold; gold < c {
			return Saving{"healing potions", c - gold}
		}
		gold -= n * t.KitPotionGold
	}
	if repairNeed > gold {
		return Saving{"the repair", repairNeed - gold}
	}
	return Saving{}
}

// ManaBuyRule says how a class with mana need `need` (ManaNeed) may buy mana potions. Data, not a class check: the
// need levels are the class trait table (types.go ManaNeed).
type ManaBuyRule struct {
	Allow       bool // false: never buy mana potions (a melee class that casts nothing)
	MinHealing  int  // healing potions (belt, pack) required before the first mana potion
	KeepKit     bool // the purchase must leave the whole minimal kit affordable (KitCost)
	MaxPerVisit int  // cap per visit (0 = the belt plan decides)
}

// ManaBuy is the rule for a mana need level. The Sorceress group keeps its old behaviour (the belt plan and the
// manaReserve of the shop decide); a class with little mana use buys a mana potion only after the healing floor is met
// and only when a full kit stays affordable (sandbox log: a Paladin with 30 max mana bought 60 gold mana potions at 21
// gold left while its healing belt was short).
func ManaBuy(need int, t Tuning) ManaBuyRule {
	switch {
	case need >= 2:
		return ManaBuyRule{Allow: true}
	case need == 1:
		return ManaBuyRule{Allow: true, MinHealing: t.HPFirst, KeepKit: true, MaxPerVisit: 2}
	}
	return ManaBuyRule{}
}

// ManaBuyOK applies the rule to one purchase of a potion costing price, with haveHealing healing potions carried and
// bought mana potions already bought this visit.
func ManaBuyOK(r ManaBuyRule, t Tuning, gold, price, haveHealing, bought int) bool {
	switch {
	case !r.Allow, haveHealing < r.MinHealing:
		return false
	case r.MaxPerVisit > 0 && bought >= r.MaxPerVisit:
		return false
	case r.KeepKit && gold-price < KitCost(t):
		return false
	}
	return true
}

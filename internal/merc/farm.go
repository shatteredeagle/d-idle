package merc

// KitRefill is the gold of a full kit: a town portal scroll plus the minimum healing and mana potions
// (resource.kitMinHP, resource.kitMinMP at the shop prices KitTPGold, KitPotionGold; mana at the healing price, U).
// A constant, not the shortfall: the reserve must cover the next refill after a fight, not the current belt.
func KitRefill(tpGold, potionGold, minHP, minMP int) int { return tpGold + (minHP+minMP)*potionGold }

// GoldNeed is the gold that pays the next merc action: price (the revive estimate, formula and pre-check only, or the
// hire quote) times the multiple, plus the kit reserve.
func GoldNeed(c Ctx, price int) int { return c.need(price, c.Knobs.ReviveGoldMult) }

// GoldFarm says whether the character should prefer the farm areas (the farm flag of the plan steps): the merc is missing (dead, or none after the claim),
// gold is short of need, and clvl is below the cap (0 = off), so a high level character is never stuck in the fields.
func GoldFarm(mercMissing bool, gold, need, clvl int, k Knobs) bool {
	return k.GoldFarmMaxLevel > 0 && mercMissing && gold < need && clvl < k.GoldFarmMaxLevel
}

// ShouldWaitForMerc: after a town portal or waypoint the merc arrives a moment after the player; the character waits
// for a living merc that is not yet near, up to maxWait, then goes on.
func ShouldWaitForMerc(s MercState, near bool, waited, maxWait int) bool {
	return s.Known && s.Owned && s.Alive && !near && waited < maxWait
}

// LeaveWithMercDown is the crowd rule with the merc down: when the crowd rule fires (the pack that killed the merc is
// still here) and the merc is dead, leave without waiting for it.
func LeaveWithMercDown(s MercState, crowd bool) bool { return crowd && s.Known && s.Owned && !s.Alive }

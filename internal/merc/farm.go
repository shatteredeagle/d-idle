package merc

import "time"

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

// Why a farm arming ended (FarmBudget.Step).
const (
	FarmEndNone   = ""
	FarmEndBudget = "budget"
	FarmEndGold   = "gold"
	FarmEndWant   = "want" // GoldFarm turned false by itself (merc back, level cap)
)

// rearmPct: after the farm ended because gold reached the bar it arms again only below this % of the bar, so a purse
// hovering at the bar (chores spend a few gold) does not flap.
const rearmPct = 90

// FarmBudget limits the gold farm in time: at most GoldFarmBudgetMin virtual minutes per arming, then
// GoldFarmCooldownMin minutes of no farm; it also ends when gold reaches the bar. Zero value is ready. Not
// goroutine safe: the caller holds its own lock.
type FarmBudget struct {
	On        bool
	armedAt   time.Time
	coolUntil time.Time
	needDrop  bool // ended at the bar: wait for gold below rearmPct of it
}

// Step takes the plain GoldFarm answer (want) and returns the farm state after the budget, plus the reason an
// arming ended now (FarmEndNone when nothing ended).
func (b *FarmBudget) Step(now time.Time, want bool, gold, need int, k Knobs) (on bool, ended string) {
	if b.On {
		switch {
		case gold >= need:
			b.On, b.needDrop = false, true
			return false, FarmEndGold
		case !want:
			b.On = false
			return false, FarmEndWant
		case k.GoldFarmBudgetMin > 0 && now.Sub(b.armedAt) >= time.Duration(k.GoldFarmBudgetMin)*time.Minute:
			b.On = false
			b.coolUntil = now.Add(time.Duration(k.GoldFarmCooldownMin) * time.Minute)
			return false, FarmEndBudget
		}
		return true, FarmEndNone
	}
	if !want || gold >= need || now.Before(b.coolUntil) {
		return false, FarmEndNone
	}
	if b.needDrop {
		if gold*100 >= need*rearmPct {
			return false, FarmEndNone
		}
		b.needDrop = false
	}
	b.On, b.armedAt = true, now
	return true, FarmEndNone
}

// ShouldWaitForMerc: after a town portal or waypoint the merc arrives a moment after the player; the character waits
// for a living merc that is not yet near, up to maxWait, then goes on.
func ShouldWaitForMerc(s MercState, near bool, waited, maxWait int) bool {
	return s.Known && s.Owned && s.Alive && !near && waited < maxWait
}

// LeaveWithMercDown is the crowd rule with the merc down: when the crowd rule fires (the pack that killed the merc is
// still here) and the merc is dead, leave without waiting for it.
func LeaveWithMercDown(s MercState, crowd bool) bool { return crowd && s.Known && s.Owned && !s.Alive }

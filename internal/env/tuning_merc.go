package env

// MercTuning holds the mercenary policy knobs read by internal/merc. Policy and switches, not skill: no handicap
// profile moves them. U: every default is a judgment from docs/plan/plan-v2.md section 3 and docs/merc.md.
type MercTuning struct {
	ReviveMaxPerHour     int // revives allowed per hour
	ReviveGoldMult       int // revive only with gold >= this x the price
	StopDeathsPerHour    int // more merc deaths than this per hour stops reviving ...
	StopGoldMult         int // ... until gold >= this x the price
	HireGoldMult         int // hire/replace only with gold after the kit >= this x the quote
	FreeRogueOnly        int // 1: never pay for the Act 1 Rogue (even after Blood Raven is skipped for good)
	ReplaceAtAct2        int // 1: swap the Rogue for a Desert Merc on the first Lut Gholein visit per difficulty
	SwapGainPct          int // merc gear swaps need this % score gain
	ReqStrDex            int // 1: merc gear needs the merc's own Str/Dex (d2decomped finding, UNVERIFIED live); 0: level only
	PotionLifePct        int // healing potion for the merc only below this life %
	KitReservePct        int // revive/hire also needs this % of the full kit refill cost on top of the price (potions first)
	GoldFarmMaxLevel     int // while the merc is missing and gold short, level in the early areas up to this clvl; 0 = off
	KitReservePctFullKit int // the kit reserve in % while the kit is already full (just bought)
	GoldFarmBudgetMin    int // one gold farm arming lasts at most this many virtual minutes; 0 = no budget
	GoldFarmCooldownMin  int // after a budget end the farm may not arm again for this many minutes
	ReviveReserveMax     int // a revive never keeps back more than this many gold for the kit (the hire reserve is not capped)
	Enabled              int // 1: the bot runs the merc policy (revive, guard input, status); 0: Game.exe and d2d behave as before
}

func DefaultMercTuning() MercTuning {
	return MercTuning{ReviveMaxPerHour: 2, ReviveGoldMult: 1, StopDeathsPerHour: 3, StopGoldMult: 5, HireGoldMult: 1,
		FreeRogueOnly: 0, ReplaceAtAct2: 1, SwapGainPct: 10, ReqStrDex: 1, PotionLifePct: 50, KitReservePct: 100, GoldFarmMaxLevel: 17,
		KitReservePctFullKit: 50, GoldFarmBudgetMin: 40, GoldFarmCooldownMin: 60, ReviveReserveMax: 100, Enabled: 1}
}

func mercKnobs() []Knob {
	return []Knob{
		{Key: "merc.reviveMaxPerHour", Label: "Merc revives per hour", Unit: "count", Default: 2, Min: 0, Max: 12, NoHandicap: true,
			Desc: "Most merc revives in one hour; a merc that keeps dying costs more than it saves."},
		{Key: "merc.reviveGoldMult", Label: "Revive gold multiple", Unit: "x", Default: 1, Min: 1, Max: 10, NoHandicap: true,
			Desc: "Revive only with gold of at least this many times the price plus merc.kitReservePct of the full kit refill (potions come first)."},
		{Key: "merc.stopDeathsPerHour", Label: "Merc deaths before stop", Unit: "count", Default: 3, Min: 1, Max: 12, NoHandicap: true,
			Desc: "More merc deaths than this in an hour stop reviving until gold reaches the stop multiple."},
		{Key: "merc.stopGoldMult", Label: "Stop-state gold multiple", Unit: "x", Default: 5, Min: 1, Max: 20, NoHandicap: true,
			Desc: "While reviving is stopped by deaths, revive only with gold of at least this many times the price."},
		{Key: "merc.hireGoldMult", Label: "Hire gold multiple", Unit: "x", Default: 1, Min: 1, Max: 10, NoHandicap: true,
			Desc: "Hire or replace only when gold is at least this many times the quote plus merc.kitReservePct of the full kit refill."},
		{Key: "merc.freeRogueOnly", Label: "Free Rogue only", Unit: "bool", Default: 0, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1 never pays for the Act 1 Rogue, even once Blood Raven is skipped for good; 0 allows that fallback."},
		{Key: "merc.replaceAtAct2", Label: "Replace at Act 2", Unit: "bool", Default: 1, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1 swaps the Rogue for a Desert Mercenary with the class aura at the first Lut Gholein visit per difficulty."},
		{Key: "merc.swapGainPct", Label: "Merc gear swap gain", Unit: "%", Default: 10, Min: 0, Max: 100, NoHandicap: true,
			Desc: "A merc item is swapped only when its score is this much better. Lower swaps more often."},
		{Key: "merc.reqStrDex", Label: "Merc gear Str/Dex rule", Unit: "bool", Default: 1, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1 checks the merc's own Strength and Dexterity against gear (d2decomped finding, UNVERIFIED live); 0 checks only level."},
		{Key: "merc.potionLifePct", Label: "Merc potion life", Unit: "%", Default: 50, Min: 10, Max: 80, NoHandicap: true,
			Desc: "A healing potion goes to the merc only below this life percent in a non-trivial fight."},
		{Key: "merc.kitReservePct", Label: "Merc kit reserve", Unit: "%", Default: 100, Min: 0, Max: 500, NoHandicap: true,
			Desc: "A revive or hire also needs this percent of the full kit refill cost (town portal, resource.kitMinHP healing, resource.kitMinMP mana) in gold on top of the price: potions first, then the merc."},
		{Key: "merc.kitReservePctFullKit", Label: "Merc kit reserve, full kit", Unit: "%", Default: 50, Min: 0, Max: 500, NoHandicap: true,
			Desc: "The kit reserve used instead of merc.kitReservePct while the kit is already full (potions and portal just bought, so the next refill is not due soon)."},
		{Key: "merc.reviveReserveMax", Label: "Merc revive reserve cap", Unit: "gold", Default: 100, Min: 0, Max: 2000, NoHandicap: true,
			Desc: "The kit reserve a revive keeps back (merc.kitReservePct of the full kit refill) is never more than this many gold, so a dead merc is revived once the purse covers the price plus a town portal scroll. g2f seed 17 stayed dead 7 hours with gold 100 to 300 short of price plus the 155 to 310 reserve."},
		{Key: "merc.goldFarmBudgetMin", Label: "Merc gold farm budget", Unit: "min", Default: 40, Min: 0, Max: 600, NoHandicap: true,
			Desc: "One gold farm arming lasts at most this many virtual minutes, and ends early when gold reaches the revive bar; then merc.goldFarmCooldownMin applies. 0 = no time limit."},
		{Key: "merc.goldFarmCooldownMin", Label: "Merc gold farm cooldown", Unit: "min", Default: 60, Min: 0, Max: 600, NoHandicap: true,
			Desc: "Minutes after a budget end before the gold farm may arm again."},
		{Key: "merc.goldFarmMaxLevel", Label: "Merc gold farm up to clvl", Unit: "levels", Default: 17, Min: 0, Max: 40, NoHandicap: true,
			Desc: "While the merc is dead or missing and gold cannot pay for a revive or hire, a Normal Act 1 character below this clvl levels only in Blood Moor, Cold Plains, Stony Field, Den, Caves, Burial Grounds and Crypt. 17 equals boss.andarielMinLevel, the no-merc Andariel floor: at that level farming buys nothing. 0 turns it off."},
		{Key: "merc.enabled", Label: "Merc policy on", Unit: "bool", Default: 1, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1 runs the mercenary policy in the bot (revive, Act 2 hire, merc state for the route guards, status line); 0 leaves the bot exactly as it was. On by default since the 2026-11-02 merc decision (docs/merc-wire.md)."},
	}
}

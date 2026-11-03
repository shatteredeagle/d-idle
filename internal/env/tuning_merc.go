package env

// MercTuning holds the mercenary policy knobs read by internal/merc. Policy and switches, not skill: no handicap
// profile moves them. U: every default is a judgment from docs/plan/plan-v2.md section 3 and docs/merc.md.
type MercTuning struct {
	ReviveMaxPerHour  int // revives allowed per hour
	ReviveGoldMult    int // revive only with gold >= this x the price
	StopDeathsPerHour int // more merc deaths than this per hour stops reviving ...
	StopGoldMult      int // ... until gold >= this x the price
	HireGoldMult      int // hire/replace only with gold after the kit >= this x the quote
	FreeRogueOnly     int // 1: never pay for the Act 1 Rogue (even after Blood Raven is skipped for good)
	ReplaceAtAct2     int // 1: swap the Rogue for a Desert Merc on the first Lut Gholein visit per difficulty
	SwapGainPct       int // merc gear swaps need this % score gain
	ReqStrDex         int // 1: merc gear needs the merc's own Str/Dex (d2decomped finding, UNVERIFIED live); 0: level only
	PotionLifePct     int // healing potion for the merc only below this life %
	KitReservePct     int // revive/hire also needs this % of the full kit refill cost on top of the price (potions first)
	GoldFarmMaxLevel  int // while the merc is missing and gold short, level in the early areas up to this clvl; 0 = off
	Enabled           int // 1: the bot runs the merc policy (revive, guard input, status); 0: Game.exe and d2d behave as before
}

func DefaultMercTuning() MercTuning {
	return MercTuning{ReviveMaxPerHour: 2, ReviveGoldMult: 1, StopDeathsPerHour: 3, StopGoldMult: 5, HireGoldMult: 1,
		FreeRogueOnly: 0, ReplaceAtAct2: 1, SwapGainPct: 10, ReqStrDex: 1, PotionLifePct: 50, KitReservePct: 150, GoldFarmMaxLevel: 16, Enabled: 1}
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
		{Key: "merc.kitReservePct", Label: "Merc kit reserve", Unit: "%", Default: 150, Min: 0, Max: 500, NoHandicap: true,
			Desc: "A revive or hire also needs this percent of the full kit refill cost (town portal, resource.kitMinHP healing, resource.kitMinMP mana) in gold on top of the price: potions first, then the merc."},
		{Key: "merc.goldFarmMaxLevel", Label: "Merc gold farm up to clvl", Unit: "levels", Default: 16, Min: 0, Max: 40, NoHandicap: true,
			Desc: "While the merc is dead or missing and gold cannot pay for a revive or hire, a Normal Act 1 character below this clvl levels only in Blood Moor, Cold Plains, Stony Field, Den, Caves, Burial Grounds and Crypt. 0 turns it off."},
		{Key: "merc.enabled", Label: "Merc policy on", Unit: "bool", Default: 1, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1 runs the mercenary policy in the bot (revive, Act 2 hire, merc state for the route guards, status line); 0 leaves the bot exactly as it was. On by default since the 2026-11-02 merc decision (docs/merc-wire.md)."},
	}
}

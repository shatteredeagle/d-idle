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
	Enabled           int // 1: the bot runs the merc policy (revive, guard input, status); 0: Game.exe and d2d behave as before
}

func DefaultMercTuning() MercTuning {
	return MercTuning{ReviveMaxPerHour: 2, ReviveGoldMult: 2, StopDeathsPerHour: 3, StopGoldMult: 5, HireGoldMult: 3,
		FreeRogueOnly: 0, ReplaceAtAct2: 1, SwapGainPct: 10, ReqStrDex: 1, PotionLifePct: 40, Enabled: 0}
}

func mercKnobs() []Knob {
	return []Knob{
		{Key: "merc.reviveMaxPerHour", Label: "Merc revives per hour", Unit: "count", Default: 2, Min: 0, Max: 12, NoHandicap: true,
			Desc: "Most merc revives in one hour; a merc that keeps dying costs more than it saves."},
		{Key: "merc.reviveGoldMult", Label: "Revive gold multiple", Unit: "x", Default: 2, Min: 1, Max: 10, NoHandicap: true,
			Desc: "Revive only with gold of at least this many times the price (and the town portal and potions kit)."},
		{Key: "merc.stopDeathsPerHour", Label: "Merc deaths before stop", Unit: "count", Default: 3, Min: 1, Max: 12, NoHandicap: true,
			Desc: "More merc deaths than this in an hour stop reviving until gold reaches the stop multiple."},
		{Key: "merc.stopGoldMult", Label: "Stop-state gold multiple", Unit: "x", Default: 5, Min: 1, Max: 20, NoHandicap: true,
			Desc: "While reviving is stopped by deaths, revive only with gold of at least this many times the price."},
		{Key: "merc.hireGoldMult", Label: "Hire gold multiple", Unit: "x", Default: 3, Min: 1, Max: 10, NoHandicap: true,
			Desc: "Hire or replace only when gold after the kit is at least this many times the quote."},
		{Key: "merc.freeRogueOnly", Label: "Free Rogue only", Unit: "bool", Default: 0, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1 never pays for the Act 1 Rogue, even once Blood Raven is skipped for good; 0 allows that fallback."},
		{Key: "merc.replaceAtAct2", Label: "Replace at Act 2", Unit: "bool", Default: 1, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1 swaps the Rogue for a Desert Mercenary with the class aura at the first Lut Gholein visit per difficulty."},
		{Key: "merc.swapGainPct", Label: "Merc gear swap gain", Unit: "%", Default: 10, Min: 0, Max: 100, NoHandicap: true,
			Desc: "A merc item is swapped only when its score is this much better. Lower swaps more often."},
		{Key: "merc.reqStrDex", Label: "Merc gear Str/Dex rule", Unit: "bool", Default: 1, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1 checks the merc's own Strength and Dexterity against gear (d2decomped finding, UNVERIFIED live); 0 checks only level."},
		{Key: "merc.potionLifePct", Label: "Merc potion life", Unit: "%", Default: 40, Min: 10, Max: 80, NoHandicap: true,
			Desc: "A healing potion goes to the merc only below this life percent in a non-trivial fight."},
		{Key: "merc.enabled", Label: "Merc policy on", Unit: "bool", Default: 0, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1 runs the mercenary policy in the bot (revive decision, merc state for the route guards, status line); 0 leaves the bot exactly as it was. Off until the merc reader is verified live (docs/merc-wire.md)."},
	}
}

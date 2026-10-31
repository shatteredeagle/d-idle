package env

// DefaultAndarielMinLevel is the character level at which the campaign lets the bot try Andariel in Normal (the one
// source of truth: internal/task, internal/boss and plandata runs.json read it or are tested against it).
// U: a judgment from docs/sme/pace-tables.md s5.2 (humans kill her at clvl 14 to 18) and docs/plan/code-reality.md
// s3.2 (Normal Act 1 monsters top out near level 12, so XP decays from about clvl 17); the fight itself was never run live.
const DefaultAndarielMinLevel = 17

// DefaultAndarielMinLevelMerc is the same floor with a living merc (ADR 0011, grill 0003: 17 without, 15 with).
const DefaultAndarielMinLevelMerc = 15

// BossTuning holds the boss-gate knobs. Owner: the planner.
type BossTuning struct {
	AndarielMinLevel     int // clvl at which the Normal campaign attempts Andariel without a living merc (22 reproduces the old hold)
	AndarielMinLevelMerc int // the same with a living merc
	AndarielHold         int // 1: the campaign makes no unattended Andariel attempt until the throwaway fight (card m3-andariel) has won once

	// Blood Raven kit gate (plan-v2 10.1; numbers from docs/sme/blood-raven-caster.md, U: judgment, not measured).
	BRCasterLevel int // clvl floor for Sorceress, Druid and Amazon
	BRMeleeLevel  int // clvl floor for the Necromancer (skeletons) and the melee classes
	BRPotions     int // healing potions in belt and pack
	BRManaPotions int // mana potions a caster needs
	BRTPCharges   int // town portal charges (tome) or scrolls
	BRGold        int // gold kept for the refill a retreat trip triggers
	BRMaxRetreats int // retreats inside one attempt; the last one ends the attempt (town portal out)
}

// AndarielLowFloor is the lower of the two Andariel floors (the one a living merc gets; the no-merc floor when the merc
// knob is 0): the earliest level any attempt can start at.
func (b BossTuning) AndarielLowFloor() int {
	if b.AndarielMinLevelMerc > 0 {
		return min(b.AndarielMinLevel, b.AndarielMinLevelMerc)
	}
	return b.AndarielMinLevel
}

func DefaultBossTuning() BossTuning {
	return BossTuning{AndarielMinLevel: DefaultAndarielMinLevel, AndarielMinLevelMerc: DefaultAndarielMinLevelMerc, AndarielHold: 1, BRCasterLevel: 9, BRMeleeLevel: 8, BRPotions: 4, BRManaPotions: 2,
		BRTPCharges: 2, BRGold: 250, BRMaxRetreats: 2}
}

func bossKnobs() []Knob {
	return []Knob{
		{Key: "boss.andarielMinLevel", Label: "Andariel level floor (no merc)", Unit: "levels", Default: DefaultAndarielMinLevel, Min: 12, Max: 22, HigherIsBetter: true, NoHandicap: true,
			Desc: "Character level at which the Normal campaign attempts Andariel without a living merc (a dead or unknown merc counts as none; this replaces the old +2 for a dead merc); the kit gate (potions, town portal, at least 2 antidotes) applies on top. Interim (ADR 0011): 17. 22 is the old hold, 17 equals the old 15 plus 2 for a dead merc."},
		{Key: "boss.andarielMinLevelMerc", Label: "Andariel level floor (living merc)", Unit: "levels", Default: DefaultAndarielMinLevelMerc, Min: 12, Max: 22, HigherIsBetter: true, NoHandicap: true,
			Desc: "The same floor with a living merc. Interim (ADR 0011): 15 (the old floor)."},
		{Key: "boss.andarielHold", Label: "Andariel: no unattended attempt yet", Unit: "bool", Default: 1, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1: the campaign does not enter Andariel unattended (log: boss: held for andariel (unattended hold ...)) until the throwaway fight of card m3-andariel has won once; the level farms instead. Set 0 after that win. Attended runs (bossrun) are not affected. ADR 0011 gate 3."},
		{Key: "boss.brCasterLevel", Label: "Blood Raven floor (Sorc/Druid/Ama)", Unit: "levels", Default: 9, Min: 6, Max: 14, HigherIsBetter: true, NoHandicap: true,
			Desc: "Character level a Sorceress, Druid or Amazon needs before the campaign attempts Blood Raven (the free Rogue). SME: below 8 it is a death trap, 9 with the kit is the first clean try."},
		{Key: "boss.brMeleeLevel", Label: "Blood Raven floor (Necro/melee)", Unit: "levels", Default: 8, Min: 6, Max: 14, HigherIsBetter: true, NoHandicap: true,
			Desc: "Same floor for the Necromancer (skeletons and Clay Golem do physical damage, her 0 percent resist) and the melee classes."},
		{Key: "boss.brPotions", Label: "Blood Raven healing potions", Unit: "count", Default: 4, Min: 2, Max: 8, HigherIsBetter: true, NoHandicap: true,
			Desc: "Healing potions (belt and pack) the kit needs before each Blood Raven attempt."},
		{Key: "boss.brManaPotions", Label: "Blood Raven mana potions (caster)", Unit: "count", Default: 2, Min: 1, Max: 6, HigherIsBetter: true, NoHandicap: true,
			Desc: "Mana potions a caster needs before each Blood Raven attempt (she resists every element by half: a mana race)."},
		{Key: "boss.brTPCharges", Label: "Blood Raven town portal charges", Unit: "count", Default: 2, Min: 1, Max: 5, HigherIsBetter: true, NoHandicap: true,
			Desc: "Town portal tome charges or scrolls needed before each attempt. The bot cannot read charges yet (U), then any portal counts."},
		{Key: "boss.brGold", Label: "Blood Raven gold reserve", Unit: "gold", Default: 250, Min: 0, Max: 1500, HigherIsBetter: true, NoHandicap: true,
			Desc: "Gold needed at each attempt: a retreat trip refills potions and a portal, and an empty purse made G1 attempt 8 drain its gold. Below it the character levels and farms first."},
		{Key: "boss.brMaxRetreats", Label: "Blood Raven retreats per attempt", Unit: "count", Default: 2, Min: 1, Max: 4, HigherIsBetter: false, NoHandicap: true,
			Desc: "Retreats to the entry (drink, regroup) one attempt may use. The last one ends the attempt with a town portal; the next attempt waits for more levels."},
	}
}

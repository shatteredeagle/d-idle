package resource

import "d2lod-bot/internal/env"

// Tuning holds the resource knobs (register as env keys "resource.*", see Knobs).
type Tuning struct {
	// PreEngage
	RestStartManaPct int // rest before a pack when mana is below this % (casters)
	CastsPerMonster  int // casts budgeted per monster of the pack ahead
	RestMaxSec       int // give up resting after this many seconds
	EngageLifePct    int // below this life % do not walk into a pack without fixing it
	MinEngageLifePct int // below this a pack is never engaged (heal first)
	RetreatDanger    int // pack danger at or above which resting happens out of sight (retreat and rest)
	HPReserve        int // healing potions never spent on engaging, only on emergencies

	// AttackChoice
	EconomyManaPct  int // below this mana % prefer the cheapest attack skill
	MeleeMinLifePct int // melee fallback needs at least this life %
	MeleeMaxPack    int // ... and at most this many hostiles
	LowManaWaitMs   int // wait this long when nothing can be done but wait
	WaitMaxSec      int // longest single stretch of waiting for mana inside a fight
	BackoffMaxSteps int // consecutive back-off steps after which the melee fallback ignores pack size and danger 2; 0 = never

	// PotionPolicy
	DrinkLifePct      int
	DrinkRejuvPct     int
	DrinkRejuvBossPct int
	ChickenPct        int
	ChickenHell       int
	DrinkManaPct      int
	PoorGold          int // below this gold the character is poor: life potions first, mana potions only when needed
	ReserveGold       int // never spend below this on potions
	PotionGoldPct     int // share of spendable gold potions may take (not when poor)
	HPFirst           int // healing potions bought before any other kind
	MPFirst           int // mana potions bought second for casters

	// TownTripTrigger
	TripHPMin   int // trip when healing potions are at or below this
	TripMPMin   int // trip (casters) when mana potions are at or below this
	TripMinGold int // below this gold a restock trip is futile
	FutileStep  int // after a trip that could not fix a reason, gold must grow by this much first
	TripHealPct int // free healing in town when life % is below this and the character cannot regenerate

	// AbortFight
	MercDeadPack int // merc dead and at least this many hostiles: leave

	// Flee and poverty
	FleeLifePct   int // no potion, no portal: break away below this life %
	KitTPGold     int
	KitPotionGold int
	PovertyMargin int
	PovertyLevels int

	RetreatBelt      int
	RetreatCountPack int
	ManaPaysCasts    int
	MinFieldBeltPct  int

	GearGateMode, GearRiskMedium, GearRiskHigh, GearDeathsHigh, EarlyGearMax int
	TripBudget                                                               int
	TripWindowMin                                                            int
	TripGoldFloor                                                            int

	RecoveryOn, RecoveryKitGold, RecoveryTargetGold, RecoveryMaxMin, RecoveryCooldownMin, RecoveryPotionFloor, RecoveryManaFloor, RecoveryXPPick, RecoveryMaxAreaAbove int
	RecoveryXPEarly, PovertySellMin                                                                                                                                    int
	BRAttempts, BRLevelGap                                                                                                                                             int
}

func DefaultTuning() Tuning { return FromEnv(env.DefaultTuning()) }

// Knobs lists the knobs; cmd/d2bot registers them with env.RegisterKnobs. Ranges: the window in which the bot still works on a
// Sorceress in Normal act 1 (U, from the simulation harness).
func Knobs() []env.Knob {
	d := DefaultTuning()
	k := func(key, label, unit, desc string, def int, min, max float64, higher bool) env.Knob {
		return env.Knob{Key: "resource." + key, Label: label, Unit: unit, Desc: desc, Default: float64(def), Min: min, Max: max, HigherIsBetter: higher}
	}
	return []env.Knob{
		k("restStartManaPct", "Rest before pack below", "%", "Casters rest in place until mana reaches this percent (or the pack's cast budget) before engaging.", d.RestStartManaPct, 20, 90, true),
		k("castsPerMonster", "Casts per monster", "count", "Casts budgeted per monster of the next pack when deciding how much mana is enough.", d.CastsPerMonster, 1, 6, true),
		k("restMaxSec", "Rest timeout", "s", "Longest rest before a pack; beyond it the policy drinks, falls back to melee or goes on.", d.RestMaxSec, 10, 120, true),
		k("engageLifePct", "Engage only above", "%", "Life percent below which the bot fixes its life before a pack.", d.EngageLifePct, 40, 90, true),
		k("minEngageLifePct", "Never engage below", "%", "Life percent below which a pack is not engaged at all.", d.MinEngageLifePct, 25, 70, true),
		k("retreatDanger", "Retreat-rest danger", "level", "Pack danger at which resting happens out of sight.", d.RetreatDanger, 1, 3, false),
		k("hpReserve", "Healing potion reserve", "count", "Healing potions kept for emergencies, not spent on engaging.", d.HPReserve, 0, 4, true),
		k("economyManaPct", "Economy casting below", "%", "Below this mana percent the cheapest attack skill is used.", d.EconomyManaPct, 10, 70, true),
		k("meleeMinLifePct", "Melee fallback min life", "%", "Life percent needed for the melee fallback when out of mana.", d.MeleeMinLifePct, 30, 80, true),
		k("meleeMaxPack", "Melee fallback max pack", "count", "Largest hostile count the melee fallback accepts.", d.MeleeMaxPack, 1, 6, false),
		k("lowManaWaitMs", "Low-mana wait", "ms", "Wait when out of mana and nothing else is possible.", d.LowManaWaitMs, 300, 2000, false),
		k("waitMaxSec", "Longest wait in a fight", "s", "Longest stretch of waiting for mana inside a fight before retreating.", d.WaitMaxSec, 2, 30, false),
		k("backoffMaxSteps", "Back-off cap", "steps", "Consecutive out-of-mana back-off steps before a weak caster swings its weapon instead of running on (any pack size, danger below 3, life above the melee line). 0 = never.", d.BackoffMaxSteps, 0, 30, false),
		k("recoveryManaFloor", "Recovery mana potions", "count", "While recovering, a caster may also buy mana potions up to this many (belt plus pack), after the healing floor. 0 = none.", d.RecoveryManaFloor, 0, 4, true),
		k("poorGold", "Poor below", "gold", "Below this gold life potions come before mana potions.", d.PoorGold, 50, 500, true),
		k("reserveGold", "Gold reserve", "gold", "Never spend below this on potions.", d.ReserveGold, 0, 200, true),
		k("potionGoldPct", "Potion share of gold", "%", "Share of spendable gold potions may take when not poor.", d.PotionGoldPct, 30, 100, true),
		k("hpFirst", "Healing potions first", "count", "Healing potions bought before any mana potion.", d.HPFirst, 1, 6, true),
		k("mpFirst", "Mana potions second", "count", "Mana potions bought for casters after the first healing potions.", d.MPFirst, 0, 6, true),
		k("tripHPMin", "Trip at healing potions", "count", "Town trip when healing potions are at or below this.", d.TripHPMin, 0, 4, true),
		k("tripMPMin", "Trip at mana potions", "count", "Town trip (casters) when mana potions are at or below this.", d.TripMPMin, 0, 4, true),
		k("tripMinGold", "Restock trip min gold", "gold", "Below this gold a potion trip is futile.", d.TripMinGold, 30, 300, true),
		k("futileStep", "Futile trip gold step", "gold", "After a trip that could not fix a need, gold must grow this much before the next.", d.FutileStep, 30, 500, true),
		k("tripHealPct", "Free heal trip below", "%", "Life percent below which a character that cannot regenerate goes to town for free healing.", d.TripHealPct, 25, 80, true),
		k("mercDeadPack", "Leave with dead merc at", "count", "Hostiles at which a fight is left once the merc is dead.", d.MercDeadPack, 1, 6, false),
		k("fleeLifePct", "Flee below (no potion, no portal)", "%", "Life percent below which a character with no healing potion and no portal breaks away from hostiles instead of trading blows.", d.FleeLifePct, 30, 80, true),
		k("kitTPGold", "Town portal scroll price", "gold", "Price of one town portal scroll, for the minimal-kit cost.", d.KitTPGold, 50, 200, true),
		k("kitPotionGold", "Healing potion price", "gold", "Price of the cheapest healing potion, for the minimal-kit cost.", d.KitPotionGold, 20, 100, true),
		k("povertyMargin", "Poverty exit margin", "gold", "Gold above the minimal kit cost at which poverty mode ends.", d.PovertyMargin, 0, 300, true),
		k("retreatBelt", "Retreat at healing potions", "count", "In a fight with life under the drink line and at most this many healing potions left, leave (portal, or walk home near town) instead of drinking. 0 = off.", d.RetreatBelt, 0, 3, true),
		k("retreatCountPack", "Count pack potions for the retreat", "0/1", "1 = spare healing/rejuvenation potions in the backpack (up to the free belt slots) count towards retreatBelt, so a full pack does not trigger a retreat; 0 = belt only.", d.RetreatCountPack, 0, 1, true),
		k("manaPaysCasts", "Casts that skip a mana potion", "casts", "Below the mana drink line a caster skips the potion while its mana still pays this many casts of the cheapest spell. 0 = drink at the line regardless (veteran play: drink at 30 percent before the skill swap).", d.ManaPaysCasts, 0, 6, false),
		k("minFieldBeltPct", "Min belt to leave town", "%", "Leave town for the field only with at least this percent of the planned healing potions (unless poor, then Blood Moor only). 0 = off.", d.MinFieldBeltPct, 0, 100, true),
		k("gearGateMode", "Gear gate mode", "0/1", "0 = hard gate (no weapon, no body armor or no potions and no gold closes the field), 1 = graded risk model (docs/gear-gate.md).", d.GearGateMode, 0, 1, true),
		k("gearRiskMedium", "Gear risk medium at", "points", "Risk points at which gear risk is medium (logged, rebuy priority).", d.GearRiskMedium, 1, 10, false),
		k("gearRiskHigh", "Gear risk high at", "points", "Risk points at which gear risk is high and only the first field is open.", d.GearRiskHigh, 2, 20, false),
		k("gearDeathsHigh", "Deaths that add gear risk", "deaths", "Deaths in the hour or in one area from which deaths add gear risk. Dying is normal: a few change nothing.", d.GearDeathsHigh, 2, 10, true),
		k("earlyGearMax", "Early gear below clvl", "level", "Below this level armor is ignored (worn only if it drops and is better, never bought, never a trip or gate reason) and a weapon matters only for Amazon, Paladin, Barbarian, Assassin. 0 = off (old rebuy and gate). docs/sme/early-gear-policy.md.", d.EarlyGearMax, 0, 40, true),
		k("tripBudget", "Trip budget", "trips", "Finished town trips allowed per window before a thin belt no longer sends the bot home (it drinks and fights on). 0 = no budget.", d.TripBudget, 0, 10, true),
		k("tripWindowMin", "Trip budget window", "min", "Minutes the trip budget looks back.", d.TripWindowMin, 1, 60, true),
		k("tripGoldFloor", "Trip gold floor", "gold", "Below this gold a thin belt does not trigger a retreat and town buys only one portal scroll. 0 = off.", d.TripGoldFloor, 0, 1000, true),
		k("recoveryOn", "Recovery phase", "0/1", "1 = a broke character pauses its quest step, farms gold and sellable loot in a safe area and returns at the target (docs/recovery.md). 0 = old behaviour.", d.RecoveryOn, 0, 1, true),
		k("recoveryKitGold", "Recovery starts below", "gold", "Recovery starts when gold is below this and the missing kit (portal, potions) costs more than the gold. SME: about 400-600.", d.RecoveryKitGold, 200, 1000, false),
		k("recoveryTargetGold", "Recovery target", "gold", "Recovery ends at this gold once the loot is sold. SME: 800-1000.", d.RecoveryTargetGold, 400, 2000, true),
		k("recoveryMaxMin", "Recovery time cap", "min", "Longest recovery cycle; it ends at the cap even when the target is not reached. SME: 30-60.", d.RecoveryMaxMin, 10, 120, true),
		k("recoveryCooldownMin", "Recovery cooldown", "min", "Minutes after a recovery before the next one may start.", d.RecoveryCooldownMin, 0, 120, true),
		k("recoveryPotionFloor", "Recovery potion floor", "count", "While recovering, town buys healing potions only up to this many in the belt (no mana, no rejuvenation) so the last gold stays for the kit. 0 = normal belt plan.", d.RecoveryPotionFloor, 0, 6, false),
		k("recoveryXPPick", "Recovery area by XP", "0/1", "1 = a recovering Normal act 1 character farms the safe area with the best XP per kill for its level (gold per kill is the same everywhere, d2d); 0 = the plan area of a character povertyLevels lower.", d.RecoveryXPPick, 0, 1, true),
		k("recoveryMaxAreaAbove", "Recovery area level above", "levels", "XP pick: the recovery area's monster level may be at most this many above the character level.", d.RecoveryMaxAreaAbove, 0, 4, false),
		k("recoveryXPEarly", "Recovery area early entry", "levels", "XP pick: a recovering character may enter an area this many levels below its plan enter and kit level (XPGAP: Cold Plains pays 1.5x the XP per kill of Blood Moor at the same kill rate). 0 = the plan levels.", d.RecoveryXPEarly, 0, 3, true),
		k("povertySellMin", "Poverty sell trip items", "items", "A poor character walks to town to sell junk only with this many sellable items (XPGAP: 4-item trips netted +23 gold for 3.5 minutes). 0 = built-in 4 on foot, 6 with a portal scroll.", d.PovertySellMin, 0, 16, true),
		k("brAttempts", "Blood Raven attempts", "count", "Blood Raven attempts before the bot waits for more levels (or skips her). 0 = unlimited. SME: 2-3.", d.BRAttempts, 0, 6, false),
		k("brLevelGap", "Blood Raven retry levels", "levels", "Levels to gain after the last attempt before the Blood Raven budget renews. SME: clvl+2.", d.BRLevelGap, 1, 6, true),
		k("povertyLevels", "Poverty farm levels below", "levels", "Poverty mode farms the plan area for a character this many levels lower.", d.PovertyLevels, 1, 10, true),
	}
}

// FromEnv builds the policy tuning from the shared tuning: the own knobs from Resource, the potion thresholds from
// Combat so they stay single-sourced (docs/resource.md).
func FromEnv(t env.Tuning) Tuning {
	r, c := t.Resource, t.Combat
	return Tuning{
		RestStartManaPct: r.RestStartManaPct, CastsPerMonster: r.CastsPerMonster, RestMaxSec: r.RestMaxSec,
		EngageLifePct: r.EngageLifePct, MinEngageLifePct: r.MinEngageLifePct, RetreatDanger: r.RetreatDanger, HPReserve: r.HPReserve,
		EconomyManaPct: r.EconomyManaPct, MeleeMinLifePct: r.MeleeMinLifePct, MeleeMaxPack: r.MeleeMaxPack,
		LowManaWaitMs: r.LowManaWaitMs, WaitMaxSec: r.WaitMaxSec, BackoffMaxSteps: r.BackoffMaxSteps,
		DrinkLifePct: c.DrinkLifePct, DrinkRejuvPct: c.DrinkRejuvPct, DrinkRejuvBossPct: c.DrinkRejuvBossPct, ChickenPct: c.ChickenPct, ChickenHell: c.ChickenHellPct, DrinkManaPct: c.DrinkManaPct,
		PoorGold: r.PoorGold, ReserveGold: r.ReserveGold, PotionGoldPct: r.PotionGoldPct, HPFirst: r.HPFirst, MPFirst: r.MPFirst,
		TripHPMin: r.TripHPMin, TripMPMin: r.TripMPMin, TripMinGold: r.TripMinGold, FutileStep: r.FutileStep, TripHealPct: r.TripHealPct,
		MercDeadPack: r.MercDeadPack,
		FleeLifePct:  r.FleeLifePct, KitTPGold: r.KitTPGold, KitPotionGold: r.KitPotionGold, PovertyMargin: r.PovertyMargin, PovertyLevels: r.PovertyLevels,
		RetreatBelt: r.RetreatBelt, RetreatCountPack: r.RetreatCountPack, ManaPaysCasts: r.ManaPaysCasts, MinFieldBeltPct: r.MinFieldBeltPct,
		GearGateMode: r.GearGateMode, GearRiskMedium: r.GearRiskMedium, GearRiskHigh: r.GearRiskHigh, GearDeathsHigh: r.GearDeathsHigh, EarlyGearMax: r.EarlyGearMax,
		TripBudget: r.TripBudget, TripWindowMin: r.TripWindowMin, TripGoldFloor: r.TripGoldFloor,
		RecoveryOn: r.RecoveryOn, RecoveryKitGold: r.RecoveryKitGold, RecoveryTargetGold: r.RecoveryTargetGold,
		RecoveryMaxMin: r.RecoveryMaxMin, RecoveryCooldownMin: r.RecoveryCooldownMin, RecoveryPotionFloor: r.RecoveryPotionFloor, RecoveryManaFloor: r.RecoveryManaFloor, RecoveryXPPick: r.RecoveryXPPick, RecoveryMaxAreaAbove: r.RecoveryMaxAreaAbove, RecoveryXPEarly: r.RecoveryXPEarly, PovertySellMin: r.PovertySellMin, BRAttempts: r.BRAttempts, BRLevelGap: r.BRLevelGap,
	}
}

// Gear is the graded gear-gate tuning.
func (t Tuning) Gear() GearTune {
	return GearTune{Mode: t.GearGateMode, MediumAt: t.GearRiskMedium, HighAt: t.GearRiskHigh, DeathsHigh: t.GearDeathsHigh, EarlyMax: t.EarlyGearMax}
}

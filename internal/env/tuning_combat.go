package env

// CombatTuning holds the combat knobs. Owner: the combat agent; add fields, defaults and Knob entries here only.
// Distances are subtiles (5 per tile). Percentages are 0-100. Ranges in combatKnobs are the window in which the
// bot still survives on a Sorceress in Normal Act 1 (tested offline in internal/combat, see docs/combat.md).
type CombatTuning struct {
	// Survival thresholds (docs/plan/data/potions-belt.json).
	DrinkLifePct      int // drink a healing potion below this life %
	DrinkRejuvPct     int // drink a rejuvenation potion below this life %
	DrinkRejuvBossPct int // same against a champion/boss pack once the fight is on (0 = off)
	ChickenPct        int // leave the game at or below this life % (Normal, Nightmare)
	ChickenHellPct    int // same in Hell
	DrinkManaPct      int // drink a mana potion below this mana %
	TownPotionMin     int // go to town when healing potions in the belt drop to this count or fewer (not mid-fight)

	// Crowd rule (docs/combat.md "Crowd rule"): additive to the thresholds above, only with this many hostiles close.
	CrowdN         int // hostiles within 8 subtiles that make a crowd (0 = rule off)
	CrowdPortalPct int // in a crowd: open a town portal at or below this life % (a scroll exists, fewer than 3 melee adjacent)
	CrowdDrinkPct  int // in a crowd: drink a healing potion below this life % (when the portal cannot be cast)
	CrowdAbandonN  int // hostiles within 8 subtiles on a walk: abandon the walk, retreat 20+ subtiles, then portal

	// Potion timing.
	PotionGapMs int // minimum gap between healing/mana potions (the game's own recovery lasts about a second)
	RejuvGapMs  int // minimum gap between rejuvenation potions
	ReactionMs  int // delay between noticing a survival event and acting on it
	PollMs      int // loop tick: how often the fight loop re-reads the game

	// Targeting and aim.
	TargetQuality   int // 100 = always the best target; lower = more often the nearest one, and occasional lost ticks
	RangeMargin     int // stay this many subtiles inside the skill's range before casting
	ClearRadius     int // default Clear radius when the caller gives none
	FightRadius     int // a hostile monster this close counts as a fight (no town trips, tighter potion rules)
	MaxTargetSec    int // give up on one target after this many seconds (docs/plan danger rules: 45)
	NoProgressSec   int // give up on a target whose life did not drop for this many seconds
	ClearMaxSec     int // upper bound of one Clear call when the caller gives no deadline
	CastGapMs       int // minimum gap between two casts (cast animation)
	LowManaWaitMs   int // wait this long when out of mana and no potion is worth drinking
	StaticCasts     int // Static Field casts opening an elite / boss fight
	StaticCastsBR   int // the same for Blood Raven (0: her 50 percent resists make it a waste of mana, U)
	StaticFloorNorm int // do not Static Field targets below this life % (Normal)
	StaticFloorNM   int // Nightmare
	StaticFloorHell int // Hell
	NovaManaFloor   int // Frost Nova on a crowd is cast only with at least this much mana (a 9-mana cast against a 50-60 pool starves the bolts)

	// Positioning.
	KiteEnabled     int // 1 = back away from melee monsters that get too close
	KiteDistance    int // back away when a melee monster is closer than this
	KiteLifePct     int // kite only below this life % (elites trigger it at any life)
	KiteMaxSteps    int // retreat steps per target before standing and fighting
	KiteRetreat     int // how far one retreat step goes
	DodgeRadius     int // step away from hostile missiles closer than this
	MissileDodge    int // 1 = sidestep fire walls, lightning, hydra and chaos missiles (off: U until verified live)
	CurseRetreatPct int // an Amplify/Iron Maiden curse on the player counts as danger and retreats below this life %
	MaxApproaches   int // approach steps per target before it is dropped as unreachable
	ReachSec        int // engaged this long without a single cast at a target: ignore it ("no-shot")
	IgnoreMemorySec int // an ignored target is left alone this long (doubles with each repeat, up to 4x)

	// Buffs.
	BuffRecastSec int // recast armors/shields this long after the last cast (state is not readable yet)

	// Class profiles (docs/combat.md).
	AuraCheckMs int // re-read the active aura this often during a fight (the read is a few memory words)
	LeftFailSec int // after a left skill could not be selected, do not try it again for this long
}

func DefaultCombatTuning() CombatTuning {
	return CombatTuning{
		DrinkLifePct: 55, DrinkRejuvPct: 35, DrinkRejuvBossPct: 50, ChickenPct: 25, ChickenHellPct: 30, DrinkManaPct: 25, TownPotionMin: 2,
		CrowdN: 8, CrowdPortalPct: 70, CrowdDrinkPct: 65, CrowdAbandonN: 15,
		PotionGapMs: 1000, RejuvGapMs: 300, ReactionMs: 120, PollMs: 60,
		TargetQuality: 100, RangeMargin: 3, ClearRadius: 40, FightRadius: 22, MaxTargetSec: 45, NoProgressSec: 12, ClearMaxSec: 180,
		CastGapMs: 450, LowManaWaitMs: 700, StaticCasts: 4, StaticCastsBR: 0, StaticFloorNorm: 25, StaticFloorNM: 33, StaticFloorHell: 50, NovaManaFloor: 25,
		KiteEnabled: 1, KiteDistance: 10, KiteLifePct: 70, KiteMaxSteps: 6, KiteRetreat: 12, DodgeRadius: 4, MissileDodge: 0, CurseRetreatPct: 60, MaxApproaches: 12, ReachSec: 15, IgnoreMemorySec: 90,
		BuffRecastSec: 150, AuraCheckMs: 1500, LeftFailSec: 60,
	}
}

func combatKnobs() []Knob {
	k := func(key, label, unit, desc string, def, min, max float64, higher bool) Knob {
		return Knob{Key: "combat." + key, Label: label, Unit: unit, Desc: desc, Default: def, Min: min, Max: max, HigherIsBetter: higher}
	}
	d := DefaultCombatTuning()
	return []Knob{
		k("drinkLifePct", "Drink healing potion below", "%", "Life percent that triggers a healing potion.", float64(d.DrinkLifePct), 35, 80, true),
		k("drinkRejuvPct", "Drink rejuvenation below", "%", "Life percent that triggers a rejuvenation potion.", float64(d.DrinkRejuvPct), 20, 60, true),
		k("drinkRejuvBossPct", "Drink rejuvenation below (boss)", "%", "Life percent that triggers a rejuvenation potion in a fight against champions, uniques or bosses (danger 1+); 0 = off. SME advice, EXP.", float64(d.DrinkRejuvBossPct), 0, 70, true),
		k("chickenPct", "Leave game at", "%", "Life percent at or below which the bot leaves the game (Normal, Nightmare).", float64(d.ChickenPct), 15, 45, true),
		k("chickenHellPct", "Leave game at (Hell)", "%", "Same as chicken, in Hell.", float64(d.ChickenHellPct), 20, 50, true),
		k("drinkManaPct", "Drink mana potion below", "%", "Mana percent that triggers a mana potion.", float64(d.DrinkManaPct), 5, 60, true),
		k("townPotionMin", "Town trip at potions left", "count", "Healing potions left in the belt at which the bot wants a town trip.", float64(d.TownPotionMin), 0, 4, true),
		k("crowdN", "Crowd size", "count", "Hostiles within 8 subtiles that make a crowd: below it the normal drink and chicken rules apply alone. 0 turns the crowd rule off.", float64(d.CrowdN), 0, 20, false),
		k("crowdPortalPct", "Crowd portal at life", "%", "In a crowd, open a town portal at or below this life percent when a scroll exists and fewer than 3 monsters are adjacent (a 2 s cast fails when surrounded).", float64(d.CrowdPortalPct), 40, 90, true),
		k("crowdDrinkPct", "Crowd drink below life", "%", "In a crowd, drink a healing potion below this life percent (the portal cannot be cast when surrounded); the normal rule drinks at 55.", float64(d.CrowdDrinkPct), 40, 85, true),
		k("crowdAbandonN", "Crowd abandon size", "count", "Hostiles within 8 subtiles on a travel or quest walk that make the bot abandon the walk at once, retreat 20+ subtiles and open a portal. 0 turns it off.", float64(d.CrowdAbandonN), 0, 30, false),
		k("potionGapMs", "Potion gap", "ms", "Minimum time between healing or mana potions.", float64(d.PotionGapMs), 900, 3000, false),
		k("rejuvGapMs", "Rejuvenation gap", "ms", "Minimum time between rejuvenation potions.", float64(d.RejuvGapMs), 250, 1500, false),
		k("reactionMs", "Reaction time", "ms", "Delay before the bot acts on a survival event.", float64(d.ReactionMs), 40, 700, false),
		k("pollMs", "Fight loop tick", "ms", "How often the fight loop looks at the game.", float64(d.PollMs), 30, 400, false),
		k("targetQuality", "Target selection quality", "%", "100 picks the best target; lower picks the nearest more often and loses ticks.", float64(d.TargetQuality), 20, 100, true),
		k("rangeMargin", "Range margin", "subtiles", "Distance kept inside a skill's maximum range.", float64(d.RangeMargin), 0, 8, true),
		k("clearRadius", "Default clear radius", "subtiles", "Radius around the start point cleared when the caller gives none.", float64(d.ClearRadius), 15, 80, true),
		k("fightRadius", "Fight radius", "subtiles", "A hostile monster this close counts as a fight.", float64(d.FightRadius), 10, 40, true),
		k("maxTargetSec", "Time per target", "s", "Give up on one target after this long.", float64(d.MaxTargetSec), 20, 90, false),
		k("noProgressSec", "No-damage timeout", "s", "Give up on a target whose life did not drop for this long.", float64(d.NoProgressSec), 5, 30, false),
		k("clearMaxSec", "Clear time limit", "s", "Upper bound of one Clear call.", float64(d.ClearMaxSec), 30, 600, true),
		k("castGapMs", "Cast gap", "ms", "Minimum time between two casts.", float64(d.CastGapMs), 250, 1200, false),
		k("lowManaWaitMs", "Out-of-mana wait", "ms", "Pause when out of mana.", float64(d.LowManaWaitMs), 300, 3000, false),
		k("staticCasts", "Static Field casts", "count", "Static Field casts that open an elite or boss fight.", float64(d.StaticCasts), 0, 8, true),
		k("staticCastsBR", "Static Field casts (Blood Raven)", "count", "Static Field casts that open the Blood Raven fight; 0 keeps the mana for the attack skill (Charged Bolt for the Sorceress).", float64(d.StaticCastsBR), 0, 8, false),
		k("staticFloorNorm", "Static Field floor (Normal)", "%", "Target life percent below which Static Field is not cast.", float64(d.StaticFloorNorm), 10, 60, false),
		k("staticFloorNM", "Static Field floor (Nightmare)", "%", "Same in Nightmare.", float64(d.StaticFloorNM), 10, 60, false),
		k("staticFloorHell", "Static Field floor (Hell)", "%", "Same in Hell.", float64(d.StaticFloorHell), 10, 70, false),
		k("novaManaFloor", "Frost Nova mana floor", "mana", "Frost Nova on a crowd of 3 or more is not cast with less mana than this, so the mana stays for the bolts.", float64(d.NovaManaFloor), 0, 60, false),
		k("kiteEnabled", "Kite melee monsters", "bool", "1 backs away from melee monsters that get close.", float64(d.KiteEnabled), 0, 1, true),
		k("kiteDistance", "Kite distance", "subtiles", "Back away when a melee monster is closer than this.", float64(d.KiteDistance), 4, 20, true),
		k("kiteLifePct", "Kite below life", "%", "Kite normal monsters only below this life percent.", float64(d.KiteLifePct), 40, 100, true),
		k("kiteMaxSteps", "Kite steps per target", "count", "Retreat steps per target before standing to fight.", float64(d.KiteMaxSteps), 0, 12, true),
		k("kiteRetreat", "Kite step length", "subtiles", "Length of one retreat step.", float64(d.KiteRetreat), 6, 20, true),
		k("dodgeRadius", "Missile dodge radius", "subtiles", "Step away from hostile missiles closer than this.", float64(d.DodgeRadius), 0, 8, true),
		k("missileDodge", "Sidestep dangerous missiles", "bool", "1 sidesteps fire walls, lightning, hydra and chaos missiles (unverified live, off by default).", float64(d.MissileDodge), 0, 1, true),
		k("curseRetreatPct", "Retreat when cursed below", "%", "With Amplify Damage or Iron Maiden on the player, retreat and drink below this life percent.", float64(d.CurseRetreatPct), 30, 90, true),
		k("maxApproaches", "Approach steps per target", "count", "Approach steps per target before it is dropped as unreachable.", float64(d.MaxApproaches), 4, 30, true),
		k("reachSec", "Time to get a shot", "s", "Give up on a target that was not shot at once in this long (unreachable, no line of sight).", float64(d.ReachSec), 6, 40, false),
		k("ignoreMemorySec", "Ignore memory", "s", "How long an ignored target is left alone (the memory doubles with every repeat).", float64(d.IgnoreMemorySec), 20, 300, true),
		k("auraCheckMs", "Aura check", "ms", "How often the fight loop re-reads the active aura and re-applies it.", float64(d.AuraCheckMs), 500, 5000, false),
		k("leftFailSec", "Left skill retry", "s", "After a left skill could not be selected, wait this long before trying it again.", float64(d.LeftFailSec), 10, 300, true),
		k("buffRecastSec", "Buff recast", "s", "Recast armors and shields this long after the last cast.", float64(d.BuffRecastSec), 60, 300, false),
	}
}

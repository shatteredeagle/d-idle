package env

// CombatTuning holds the combat knobs. Owner: the combat agent; add fields, defaults and Knob entries here only.
// Distances are subtiles (5 per tile). Percentages are 0-100. Ranges in combatKnobs are the window in which the
// bot still survives on a Sorceress in Normal Act 1 (tested offline in internal/combat, see docs/combat.md).
type CombatTuning struct {
	// Survival thresholds (docs/plan/data/potions-belt.json).
	DrinkLifePct      int // drink a healing potion below this life %
	DrinkRejuvPct     int // drink a rejuvenation potion below this life %
	DrinkRejuvBossPct int // same against a champion/boss pack once the fight is on (0 = off)
	ChampionEarlyPct  int // extra life percentage points on the drink lines against champions in Normal below clvl 15 (0 = off)
	ChickenPct        int // leave the game at or below this life % (Normal, Nightmare)
	ChickenHellPct    int // same in Hell
	DrinkManaPct      int // drink a mana potion below this mana %
	TownPotionMin     int // go to town when healing potions in the belt drop to this count or fewer (not mid-fight)

	// Crowd rule (docs/combat.md "Crowd rule"): additive to the thresholds above, only with this many hostiles close.
	CrowdN         int // hostiles within 8 subtiles that make a crowd (0 = rule off)
	CrowdPortalPct int // in a crowd: open a town portal at or below this life % (a scroll exists, fewer than 3 melee adjacent)
	CrowdDrinkPct  int // in a crowd: drink a healing potion below this life % (when the portal cannot be cast)
	CrowdAdjacent  int // subtiles: a hostile this close is in melee reach and counts as adjacent (crowd rule, melee cast guard)
	CrowdAbandonN  int // hostiles within 8 subtiles on a walk: abandon the walk, retreat 20+ subtiles, then portal

	// Potion timing.
	PotionGapMs        int // minimum gap between healing/mana potions (a potion heals over 5 to 10 s, see PotionEmergencyPct)
	PotionEmergencyPct int // below this life % a healing potion is drunk even while one is still working (a same-tier drink only extends the effect, docs/potions.md)
	RejuvGapMs         int // minimum gap between rejuvenation potions
	ReactionMs         int // delay between noticing a survival event and acting on it
	PollMs             int // loop tick: how often the fight loop re-reads the game

	// Targeting and aim.
	TargetQuality int // 100 = always the best target; lower = more often the nearest one, and occasional lost ticks
	RangeMargin   int // stay this many subtiles inside the skill's range before casting
	ClearRadius   int // default Clear radius when the caller gives none
	FightRadius   int // a hostile monster this close counts as a fight (no town trips, tighter potion rules)
	MaxTargetSec  int // give up on one target after this many seconds (docs/plan danger rules: 45)
	NoProgressSec int // give up on a target whose life did not drop for this many seconds
	ClearMaxSec   int // upper bound of one Clear call when the caller gives no deadline
	// CastGapMs: minimum gap between two casts. It MUST stay above the cast animation: the early Sorceress casts
	// Fire Bolt / Charged Bolt in 560 ms (V-d2d, docs/d2d-casts-real.md), and at the old 450 the fighter issued 1.6 to
	// 1.75 commands per real cast (the rest dropped mid-cast; 5.4 commands per kill vs 1.1 on Game.exe), at 650 1.02 to
	// 1.04. See docs/fixmissile.md.
	CastGapMs int
	// CastPerSkill 1: the gap after a cast is the larger of CastGapMs and that skill's cast length at CastFCR plus
	// CastMarginMs (internal/combat/castgap.go); 0: CastGapMs alone for every skill (the override).
	CastPerSkill    int
	CastMarginMs    int
	CastFCR         int // faster cast rate the gap assumes (0 is the safe worst case: more FCR only shortens a cast)
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
	KiteLifePct     int // step back from a single melee monster only below this life % and falling (60 = the rule's old constant)
	KiteMaxSteps    int // retreat steps per target before standing and fighting
	KiteRetreat     int // how far one retreat step goes
	DodgeRadius     int // step away from hostile missiles closer than this
	ThreatDodge     int // 1 = dodge only missiles the threat table says can be dodged in DodgeReactMs (card botthreat, U live)
	DodgeReactMs    int // time from poll to the character moving, the threat model's dodge budget (I)
	ShooterKillMin  int // >0: kill first ranged monsters with an elemental-only attack and a peak hit of at least this (card botthreat, U)
	RunnerGuard     int // 1 = no walk-away retreat while a monster that runs after the player is close (V run, monster-ai-trace D1-D5)
	StandOK         int // 1 = a standing Fallen is not dropped as "never in range" (S, mon-wake-trace; off until a live drill)
	MissileDodge    int // 1 = sidestep fire walls, lightning, hydra and chaos missiles (off: U until verified live)
	CurseRetreatPct int // an Amplify/Iron Maiden curse on the player counts as danger and retreats below this life %
	MaxApproaches   int // approach steps per target before it is dropped as unreachable
	ReachSec        int // engaged this long without a single cast at a target: ignore it ("no-shot")
	IgnoreMemorySec int // an ignored target is left alone this long (doubles with each repeat, up to 4x)

	// Buffs.
	BuffRecastSec int // upper bound / fallback: recast armors/shields this long after the last cast (the real duration, when known, is shorter)
	BuffMarginSec int // recast an armor this many seconds before its real duration (Skills.txt auralencalc at the skill's level) ends

	// Class profiles (docs/combat.md).
	AuraCheckMs int // re-read the active aura this often during a fight (the read is a few memory words)
	LeftFailSec int // after a left skill could not be selected, do not try it again for this long

	// Paladin early profile (docs/palprofile.md): Sacrifice costs life, Holy Fire only pays in a pack.
	Sacrifice           int // 1 = a profile may attack with Sacrifice (its band lists it); 0 = never, plain Attack instead
	SacrificeMinLifePct int // Sacrifice only at or above this life %; below it plain Attack until life is back 10 points above
	SacrificeMinPotions int // Sacrifice only with at least this many healing/rejuvenation potions in the belt
	HolyFireMinMonsters int // an aura marked needsCrowd (Holy Fire) is wanted only with this many hostiles within 8 subtiles
}

func DefaultCombatTuning() CombatTuning {
	return CombatTuning{
		DrinkLifePct: 55, DrinkRejuvPct: 35, DrinkRejuvBossPct: 50, ChampionEarlyPct: 10, ChickenPct: 25, ChickenHellPct: 30, DrinkManaPct: 25, TownPotionMin: 2,
		CrowdN: 8, CrowdAdjacent: 8, CrowdPortalPct: 70, CrowdDrinkPct: 65, CrowdAbandonN: 15,
		PotionGapMs: 1000, PotionEmergencyPct: 30, RejuvGapMs: 300, ReactionMs: 120, PollMs: 60,
		TargetQuality: 100, RangeMargin: 3, ClearRadius: 40, FightRadius: 22, MaxTargetSec: 45, NoProgressSec: 12, ClearMaxSec: 180,
		CastGapMs: 650, CastPerSkill: 1, CastMarginMs: 90, CastFCR: 0, LowManaWaitMs: 700, StaticCasts: 4, StaticCastsBR: 0, StaticFloorNorm: 25, StaticFloorNM: 33, StaticFloorHell: 50, NovaManaFloor: 25,
		KiteEnabled: 1, KiteDistance: 10, KiteLifePct: 60, KiteMaxSteps: 6, KiteRetreat: 12, DodgeRadius: 4, MissileDodge: 0, ThreatDodge: 0, DodgeReactMs: 400, ShooterKillMin: 0, RunnerGuard: 1, StandOK: 0, CurseRetreatPct: 60, MaxApproaches: 12, ReachSec: 15, IgnoreMemorySec: 90,
		BuffRecastSec: 150, BuffMarginSec: 6, AuraCheckMs: 1500, LeftFailSec: 60,
		Sacrifice: 1, SacrificeMinLifePct: 70, SacrificeMinPotions: 1, HolyFireMinMonsters: 3,
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
		k("championEarlyPct", "Champion early extra", "pp", "Percentage points added to the healing potion line and the rejuvenation (boss) line while champions (not bosses) are in range, in Normal below character level 15. Champions have about 3x life and +50% damage. 0 = off. SME rule of thumb, EXP.", float64(d.ChampionEarlyPct), 0, 25, true),
		k("chickenPct", "Leave game at", "%", "Life percent at or below which the bot leaves the game (Normal, Nightmare).", float64(d.ChickenPct), 15, 45, true),
		k("chickenHellPct", "Leave game at (Hell)", "%", "Same as chicken, in Hell.", float64(d.ChickenHellPct), 20, 50, true),
		k("drinkManaPct", "Drink mana potion below", "%", "Mana percent that triggers a mana potion.", float64(d.DrinkManaPct), 5, 60, true),
		k("townPotionMin", "Town trip at potions left", "count", "Healing potions left in the belt at which the bot wants a town trip.", float64(d.TownPotionMin), 0, 4, true),
		k("crowdN", "Crowd size", "count", "Hostiles within 8 subtiles that make a crowd: below it the normal drink and chicken rules apply alone. 0 turns the crowd rule off.", float64(d.CrowdN), 0, 20, false),
		k("crowdPortalPct", "Crowd portal at life", "%", "In a crowd, open a town portal at or below this life percent when a scroll exists and fewer than 3 monsters are adjacent (a 2 s cast fails when surrounded).", float64(d.CrowdPortalPct), 40, 90, true),
		k("crowdDrinkPct", "Crowd drink below life", "%", "In a crowd, drink a healing potion below this life percent (the portal cannot be cast when surrounded); the normal rule drinks at 55.", float64(d.CrowdDrinkPct), 40, 85, true),
		k("crowdAdjacent", "Melee reach for adjacent", "subtiles", "A hostile within this distance (centre to centre) is in melee reach and counts as adjacent: three or more make a town portal cast a non-escape (potion, else leave the game) and a crowd 'surrounded'. Default 8 from the live zombie reads (drills11), UNVERIFIED for other monsters.", float64(d.CrowdAdjacent), 3, 12, false),
		k("crowdAbandonN", "Crowd abandon size", "count", "Hostiles within 8 subtiles on a travel or quest walk that make the bot abandon the walk at once, retreat 20+ subtiles and open a portal. 0 turns it off.", float64(d.CrowdAbandonN), 0, 30, false),
		k("potionGapMs", "Potion gap", "ms", "Minimum time between healing or mana potions.", float64(d.PotionGapMs), 900, 3000, false),
		k("potionEmergencyPct", "Potion emergency line", "%", "Below this life percent a healing potion is drunk even while an earlier one is still working. Above it the bot drinks again only for a higher tier: the game adds a same-tier drink to the running effect (longer, not faster; fid-potions.md Q5).", float64(d.PotionEmergencyPct), 10, 50, true),
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
		k("castPerSkill", "Per-skill cast gap", "bool", "1 = the gap after a cast is at least that skill's cast length (Lightning and Chain Lightning take 760 ms in Game.exe, longer than the flat gap; d2d casts them as plain SC and keeps 650); 0 = the flat cast gap for every skill.", float64(d.CastPerSkill), 0, 1, true),
		k("castMarginMs", "Cast gap margin", "ms", "Added to the cast length (in 40 ms ticks, plus one dispatch tick) for the per-skill gap; 90 ms keeps the early Sorceress bolts at the old 650.", float64(d.CastMarginMs), 0, 400, false),
		k("castFCR", "Assumed faster cast rate", "%", "Faster cast rate the per-skill gap assumes; 0 is safe for any gear, a real value shortens the gap of the slow skills.", float64(d.CastFCR), 0, 200, true),
		k("lowManaWaitMs", "Out-of-mana wait", "ms", "Pause when out of mana.", float64(d.LowManaWaitMs), 300, 3000, false),
		k("staticCasts", "Static Field casts", "count", "Static Field casts that open an elite or boss fight.", float64(d.StaticCasts), 0, 8, true),
		k("staticCastsBR", "Static Field casts (Blood Raven)", "count", "Static Field casts that open the Blood Raven fight; 0 keeps the mana for the attack skill (Charged Bolt for the Sorceress).", float64(d.StaticCastsBR), 0, 8, false),
		k("staticFloorNorm", "Static Field floor (Normal)", "%", "Target life percent below which Static Field is not cast.", float64(d.StaticFloorNorm), 10, 60, false),
		k("staticFloorNM", "Static Field floor (Nightmare)", "%", "Same in Nightmare.", float64(d.StaticFloorNM), 10, 60, false),
		k("staticFloorHell", "Static Field floor (Hell)", "%", "Same in Hell.", float64(d.StaticFloorHell), 10, 70, false),
		k("novaManaFloor", "Frost Nova mana floor", "mana", "Frost Nova on a crowd of 3 or more is not cast with less mana than this, so the mana stays for the bolts.", float64(d.NovaManaFloor), 0, 60, false),
		k("kiteEnabled", "Kite melee monsters", "bool", "1 backs away from melee monsters that get close.", float64(d.KiteEnabled), 0, 1, true),
		k("kiteDistance", "Kite distance", "subtiles", "Back away when a melee monster is closer than this.", float64(d.KiteDistance), 4, 20, true),
		k("kiteLifePct", "Kite below life", "%", "Step back from a single melee monster only below this life percent (and falling).", float64(d.KiteLifePct), 40, 100, true),
		k("kiteMaxSteps", "Kite steps per target", "count", "Retreat steps per target before standing to fight.", float64(d.KiteMaxSteps), 0, 12, true),
		k("kiteRetreat", "Kite step length", "subtiles", "Length of one retreat step.", float64(d.KiteRetreat), 6, 20, true),
		k("dodgeRadius", "Missile dodge radius", "subtiles", "Step away from hostile missiles closer than this.", float64(d.DodgeRadius), 0, 8, true),
		k("missileDodge", "Sidestep dangerous missiles", "bool", "1 sidesteps fire walls, lightning, hydra and chaos missiles (unverified live, off by default).", float64(d.MissileDodge), 0, 1, true),
		k("threatDodge", "Threat-model dodge", "bool", "1 dodges only hostile missiles the missile threat table says can be dodged in dodgeReactMs and fights the owners of the fast ones first (card botthreat, off until a live drill).", float64(d.ThreatDodge), 0, 1, true),
		k("dodgeReactMs", "Dodge reaction budget", "ms", "Poll plus click time the threat model assumes before the character moves; a missile reaching us sooner is not dodged (I).", float64(d.DodgeReactMs), 100, 800, true),
		k("shooterKillMin", "Kill shooters first from", "pts", "Ranged monsters with an elemental-only attack and a peak hit of at least this are fought first; 0 = off (derived from monstats, U).", float64(d.ShooterKillMin), 0, 200, true),
		k("runnerGuard", "No retreat from runners", "bool", "1 skips the kite, step-back and curse retreat while a monster that runs at the player (Zombie, Brute, Blood Hawk, Corrupt Rogue) is close; walking away cannot break contact.", float64(d.RunnerGuard), 0, 1, true),
		k("standOK", "Standing Fallen is fine", "bool", "1 does not drop a Fallen that stands still as never in range (S, off until a live drill).", float64(d.StandOK), 0, 1, true),
		k("curseRetreatPct", "Retreat when cursed below", "%", "With Amplify Damage or Iron Maiden on the player, retreat and drink below this life percent.", float64(d.CurseRetreatPct), 30, 90, true),
		k("maxApproaches", "Approach steps per target", "count", "Approach steps per target before it is dropped as unreachable.", float64(d.MaxApproaches), 4, 30, true),
		k("reachSec", "Time to get a shot", "s", "Give up on a target that was not shot at once in this long (unreachable, no line of sight).", float64(d.ReachSec), 6, 40, false),
		k("ignoreMemorySec", "Ignore memory", "s", "How long an ignored target is left alone (the memory doubles with every repeat).", float64(d.IgnoreMemorySec), 20, 300, true),
		k("auraCheckMs", "Aura check", "ms", "How often the fight loop re-reads the active aura and re-applies it.", float64(d.AuraCheckMs), 500, 5000, false),
		k("leftFailSec", "Left skill retry", "s", "After a left skill could not be selected, wait this long before trying it again.", float64(d.LeftFailSec), 10, 300, true),
		k("sacrifice", "Paladin Sacrifice attack", "bool", "1 lets a profile attack with Sacrifice (about x1.9 damage over Attack, costs 8 percent of the damage dealt in life, docs/palprofile.md); 0 keeps plain Attack.", float64(d.Sacrifice), 0, 1, true),
		k("sacrificeMinLifePct", "Sacrifice life floor", "%", "Sacrifice only at or above this life percent; below it plain Attack until life is 10 points above the floor again.", float64(d.SacrificeMinLifePct), 40, 100, true),
		k("sacrificeMinPotions", "Sacrifice potion floor", "count", "Sacrifice only with at least this many healing or rejuvenation potions in the belt (and no dead merc).", float64(d.SacrificeMinPotions), 0, 8, true),
		k("holyFireMinMonsters", "Holy Fire crowd size", "count", "A needsCrowd aura (Holy Fire) is wanted only with at least this many hostiles within 8 subtiles; fewer and Might is used (Might wins below about 3 monsters, docs/palprofile.md).", float64(d.HolyFireMinMonsters), 1, 10, false),
		k("buffRecastSec", "Buff recast", "s", "Upper bound and fallback: recast armors and shields this long after the last cast. An armor whose real duration is known (Frozen Armor 120 s at level 1) is recast earlier, see buffMarginSec.", float64(d.BuffRecastSec), 60, 300, false),
		k("buffMarginSec", "Buff recast margin", "s", "Recast an armor or shield this many seconds before its real duration (Skills.txt auralencalc at the skill's level) runs out.", float64(d.BuffMarginSec), 0, 30, false),
	}
}

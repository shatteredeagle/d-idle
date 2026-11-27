package env

// BotTuning holds the knobs of the bot core (internal/bot): loop cadence, reaction time, when to go to town,
// how deep a level margin the planner keeps and how far a clear sweep reaches. Owner: the bot agent.
//
// Every default is the optimal setting; the documented Min/Max is the range in which the bot still survives and
// makes progress. A handicap profile moves a knob from its default toward the worse end of that range, never past it.
// Times are milliseconds, distances subtiles.
type BotTuning struct {
	TickMs          int // main loop period while no action is running
	WatchMs         int // guard period while an action runs: events, safety, death, watchdog
	ReactionMs      int // delay between noticing a hazard (low life, chicken) and acting on it
	ResupplyPotions int // go to town when healing potions in the belt drop below this
	InvFullPct      int // go to town when this share of inventory cells is occupied
	LevelMargin     int // character levels above a step's enter level required before the planner enters it
	LeaveSlack      int // character levels past a step's leave level the planner stays before moving on
	CorpseRetries   int // corpse retrievals attempted per death (softcore); 0 abandons the corpse; after the last failed try the game is reset so the corpse moves to town
	PortalSettleMs  int // wait in town after chores before returning to the field
	ClearRadius     int // radius of one clear sweep
	ExploredPct     int // percent of a level's roam coverage points that must be visited before a clear run counts the area as explored (100 = all, docs/exploredpct.md)
	// The farm loop (docs/farmloop.md): in a leveling zone at FarmResetPct coverage the hero takes a fresh game and comes back to the same zone when that pays.
	FarmReset            int // 1 = on
	FarmResetPct         int // coverage percent from which a reset is considered
	FarmResetCooldownMin int // least minutes between two farm resets
	ReflexDrink          int // 1: the guard sends a belt drink itself without cancelling the running action (internal/bot/safetylat.go; Fable review P3 c)
	ReflexDeadlineMs     int // ms after a portal, chicken or crowd cancel when the guard carries the action out itself if the cancelled action has not returned (0 = off)
	LootShadow           int // 1: the loot rules engine also judges every item decision and logs where it differs from today's code (docs/lootrules.md); log only, never acts

	// Death policy (PLAN.md "Death is normal"; docs/bot.md "Death").
	CorpseValueMin      int // estimated gold value of the lost gear below which the corpse is abandoned
	SpiralDeaths        int // deaths in one area within SpiralMinutes that trigger the death-spiral breaker
	SpiralMinutes       int // window of the spiral breaker
	DangerBanMin        int // minutes the breaker keeps the character out of an area (a Config.DangerBanSec above 0 overrides)
	NavSkipSec          int // seconds a nav fault (stuck, no exit) keeps the planner out of an area
	StallWarnMin        int // campaign stall ladder (internal/bot/stall.go): virtual minutes without campaign progress before the WARN and a fresh game (0 = ladder off)
	StallExitMin        int // virtual minutes still without progress after the fresh game before the bot exits for a supervisor restart
	AreaIdleSec         int // longest wait in town for area bans that are not a death streak before the least bad area is farmed anyway
	QuestBlockedFarmMin int // minutes a quest step may stay blocked before the hero farms the level plan and retries the step with a growing wait (staletarget)
	ParkAfter           int // identical "planning blocked" results of one quest step before the step is parked (0 = rule off)
	ParkMinutes         int // minutes a parked quest step stays parked while the hero farms the level plan
	StarterBanMin       int // longest ban of the first field of an act once the character stands in town because of it
	RepeatSiteBanMin    int // minutes a repeat-site death keeps the character out of the area (0 = rule off)
	RepeatSiteEscalate  int // 1: the second repeat-site ban in one area within 12 h lasts 4x, the third 12x (cap 6 h)
	RepeatSiteWindowMin int // window in which a second death in one area counts as a repeat site
	RepeatSiteGapSec    int // a death this soon after the previous one (any area) counts as a repeat site
	CorpseMinValue      int // gold-equivalent below which the in-field corpse run is skipped outright
	// Early death (TASK-39): a pack we fled from is not walked back into, and a potion burst becomes a retreat.
	StalePackHostiles   int // hostiles within the fight radius at a chicken, portal or crowd exit that make the area a stale pack (0 = rule off)
	StalePackMinutes    int // minutes the stale pack keeps the area out of the planner and the portal return unless the kit is rebuilt
	StalePackMinHP      int // healing potions (belt and pack) the kit needs to lift the stale-pack hold early
	StalePackMinTier    int // best healing potion tier in the belt the kit needs to lift the hold early (1 minor, 2 light)
	PotionBurstN        int // life drinks within PotionBurstSec that, with hostiles near and a portal, become a town portal (0 = rule off)
	PotionBurstSec      int // window of the burst rule
	PotionBurstHostiles int // hostiles within the fight radius the burst rule needs

	PauseDeathsPerHour int // deaths in the last hour (any area) after which the bot pauses itself
	PauseCapMin        int // minutes a player Pause (frozen game) lasts before it resumes by itself (docs/pause.md); 0 = no cap
	PovertyTier        int // gear tier bought when rebuilding from vendors after an abandoned corpse: 0 cheapest viable
	GoldFloor          int // gold the town visit must not spend, kept for rebuying gear

	// Leveling (docs/leveler.md, docs/bot.md "Leveling"): points, gear, poverty, difficulty.
	PointsRetrySec   int // wait before spending unspent points again after a failed attempt
	GearCheckSec     int // minimum gap between gear passes (equip from the pack, shop) in town
	GearMinGold      int // gold needed before the bot shops for gear
	GearBudgetPct    int // share of the gold above GearReserve a shopping pass may spend
	GearReserve      int // gold kept for potions when shopping for gear
	SellMinItems     int // sellable pack items that justify a town trip for a starving character
	DifficultyBanMin int // minutes the bot stays in its difficulty after a switch to a higher one failed
	// DiffReadyV2 [dark-launch]: 1 replaces the clvl-only "ready for the next difficulty" check by the per-class
	// predicate (internal/diffready) and makes the hop opt-in: the bot shows readiness and waits for the owner's go.
	DiffReadyV2 int

	// SavePointMin: the game writes the .d2s only on Save and Exit (V: docs/soak-analysis.md 6c), so a crash loses all
	// progress since. In town, idle, the bot leaves and re-enters the game after this many minutes. 0 = off.
	SavePointMin int

	// HeapLimitMB: d2bot's heap guard (cmd/d2bot/heapguard.go) writes a heap profile and exits for a supervisor restart
	// when the Go heap in use passes this many MB (G1 attempt 16: a runaway took 28 GB in 30 s and the OOM killer ended
	// the process). The Go GC also gets a soft limit 25 percent above it. 0 = off.
	HeapLimitMB int
	// FailsafeRung3Max: leave/re-enter cycles (failsafe ladder rung 3) per progress window before the ladder goes to the
	// pause rung (0 = unbounded). FailsafePauseMaxSec caps the doubling pause of that rung.
	FailsafeRung3Max int
	// Anti-flail guard (internal/bot/flail.go): FlailMoves area changes within FlailWindowMin minutes without a kill or
	// XP hold the bot in place for FlailHoldMin minutes (doubled per repeat, at most x8). 0 moves = off.
	FlailMoves, FlailWindowMin, FlailHoldMin int
	// Intent arbiter, shadow mode only (internal/arbiter, docs/arbiter.md): ArbiterShadow 1 runs it beside the chain and
	// logs where it would choose differently; it never acts. Dwell is its commitment lease, TravelSec the window in
	// which an area change is not undone without progress, LogGapSec the rate limit of the divergence lines.
	ArbiterShadow, ArbiterDwellSec, ArbiterTravelSec, ArbiterLogGapSec int
	// ArbiterControl [dark-launch]: 1 lets the arbiter ACT for the families in the ArbiterFamilies bit list (bit order:
	// arbiter.Families); the chain's hooks of such a family defer to its decision. 0 (default) changes nothing. Needs
	// ArbiterShadow 1. Enable rule: data/darklaunch.json.
	ArbiterControl, ArbiterFamilies int
	// TickLogSec [dark-launch]: above 0 writes one "tick" line at most every this many seconds with the arbiter's world
	// facts and the chain's family, for offline replay (cmd/d2arbiter, docs/design/ticklog-t4.md). 0 (default) is off.
	TickLogSec int
	// Admission [dark-launch]: 1 lets internal/admit decide the area entry rows (early road, poverty, kit floor, lean) and the
	// quest-walk kit at the town exit; 0 (default) keeps the old judges in control and only logs where admit would differ.
	Admission           int
	FailsafePauseMaxSec int
	// Invariant monitor (internal/bot/invariants.go, docs/invariants.md): 1 checks the invariant table once per tick and
	// logs one rate-limited line per hit; it never acts. 0 is off.
	Invariants int
	// TypedOutcome 1 turns a refusal of the quest board step into a typed Refused outcome handled in one place
	// (internal/bot/outcome.go, docs/outcomes.md); 0 keeps the legacy error path (a failure count per refusal).
	TypedOutcome int

	// NOKILL fixes (docs/research/g2d-nokill-diagnosis.md). Rung3WindowMax rung-3 cycles within Rung3WindowMin virtual
	// minutes end in a field reset whatever progress came between them (0 = off). DroughtMin: no kill and no XP for
	// that many virtual minutes in a field area at healthy life forces a re-plan (0 = off). EmptyFieldSec: a quest-run
	// area with no hostile in sight and no kill for that long is called cleared (non-quest areas keep Config.EmptyAreaSec).
	Rung3WindowMax int
	Rung3WindowMin int
	DroughtMin     int
	EmptyFieldSec  int
	// ClearIdleMin: a leveling slice that killed nothing for that many minutes calls its area cleared, so the planner
	// moves on (docs/hourfix.md: Blood Moor held 6 min past exhaustion). 0 = off; Config.EmptyAreaSec still wins when set.
	ClearIdleMin int

	// g2e fixes (docs/research/g2e-diagnosis.md, addendum "F2-F5 fixes"). NoPortalWaitSec: seconds after the first heal
	// trip that failed for lack of a town portal before the hero leaves the game (240 or more = never, the old wait).
	// DroughtStage2Min: virtual minutes after a stage 1 drought action before stage 2 (fresh game); 0 = DroughtMin.
	// FreshAfterRecoverySec: no unplanned fresh game this long after a lost-game recovery (0 = off).
	NoPortalWaitSec int
	// PortalRetryWaitSec: a town portal cast cancelled before the click by a drink or a retreat (nothing spent) is retried
	// ONCE after this many virtual seconds, as a town trip; 0 = off (the next planning tick decides, the old behaviour).
	PortalRetryWaitSec int
	// Checkpoint portals (docs/design/danger-protocol-2026-11-25.md): a portal cast on purpose at a danger area while
	// healthy, used as the retreat and the way back after a death. CheckpointEnable 0 = the whole feature is off.
	CheckpointEnable, CheckpointMinLifePct, CheckpointRetreatPct, CheckpointGapSec, CheckpointMinScrolls int
	CheckpointSafeRadius, CheckpointEntryDist, CheckpointReach, CheckpointArrivalHostiles                int
	DroughtStage2Min                                                                                     int
	// Slow field (TASK-138): fewer than DroughtSlowKills kills in DroughtSlowMin minutes, after that long in the same
	// field area, below clvl DroughtSlowMaxLevel, starts a fresh game (a depleted early field refills only in a new
	// game; stragglers kept the kill drought quiet). 0 for DroughtSlowMin or DroughtSlowMaxLevel = off.
	DroughtSlowMin        int
	DroughtSlowKills      int
	DroughtSlowMaxLevel   int
	FreshAfterRecoverySec int
	// Dry recovery field (palready): while a recovery farms gold, fewer than RecoveryDryKills kills in RecoveryDryMin
	// minutes in the same field area send the hero to the next eligible area (the area is skipped for
	// RecoveryDrySkipMin minutes; no other eligible area: a fresh game). 0 for RecoveryDryMin = off.
	RecoveryDryMin     int
	RecoveryDryKills   int
	RecoveryDrySkipMin int

	// Town portal cast guard (internal/resource/castguard.go, card portalguard). A hit does not interrupt a Town Portal cast
	// (V run, decomp28 Q5); the risk is the damage taken during the cast and the walk into the portal.
	// Survival slice S-D (docs/survival/design-2026-11-14.md section 10, docs/survival/zonemodel.md).
	ZoneModel int // 0 = plan tables choose the zone (legacy), 1 = the survival controller's power-vs-danger margin decides, the plan tables stay candidates and fallback
	ZoneDial  int // risk dial of the margin model: 0 cautious, 1 balanced, 2 bold

	PortalGuard   int // 0 = old rule (3 adjacent block the cast), 1 = expected damage over the exposure (falls back to the old rule when a damage term is missing)
	PortalCastMs  int // cast time of the scroll or tome read (U until drill R3)
	PortalWalkMs  int // walk into the portal after the cast (U until drill R3)
	PortalHitMs   int // assumed time between two hits of one monster (every swing lands: an upper bound)
	PortalLifePct int // block the cast when the expected damage reaches this % of current life

	// Mercenary chore (docs/merc-farm.md). Hiring clicks a verified dialog; reviving clicks an UNVERIFIED one, so it is off.
	MercRetryMin    int // minutes between merc chore attempts (a failed hire or revive is not retried sooner)
	MercHireMinGold int // gold in Act 2 town below which no hire is attempted
	// Merc revive loop breaker (survival controller, bot.zoneModel 1 only): after MercReviveLoopN revives that each ended
	// in his death within MercReviveLoopMin minutes in the same zone, no revive while that zone is the plan. 0 = off.
	MercReviveLoopN      int
	MercReviveLoopMin    int
	MercReviveUnverified bool // allow town.Revive to click the unverified revive dialog (live probe only)

	// Cube chore (internal/cubejobs): gold-free gem and rune upgrades in town. UNVERIFIED live, so it is off.
	CubeJobs    bool // run the cube chore
	CubeMaxJobs int  // transmutes per town visit
	CubeRuneCap int  // highest rune number the chore produces (10 = Thul; Dol->Hel and above are ladder-flagged, docs/items.md Q1)
	CubeGemCap  int  // highest gem grade it produces, 0 chipped .. 4 perfect
}

func DefaultBotTuning() BotTuning {
	return BotTuning{
		TickMs: 100, WatchMs: 50, ReactionMs: 0, ResupplyPotions: 2, InvFullPct: 85,
		LevelMargin: 0, LeaveSlack: 0, CorpseRetries: 3, PortalSettleMs: 600, ClearRadius: 30, ExploredPct: 100, FarmReset: 0, FarmResetPct: 85, FarmResetCooldownMin: 5, LootShadow: 1, ReflexDrink: 1, ReflexDeadlineMs: 1000,
		CorpseValueMin: 1000, SpiralDeaths: 4, SpiralMinutes: 15, DangerBanMin: 20, StarterBanMin: 3, QuestBlockedFarmMin: 10, ParkAfter: 2, ParkMinutes: 15, RepeatSiteBanMin: 30, RepeatSiteEscalate: 1, RepeatSiteWindowMin: 30, RepeatSiteGapSec: 180, CorpseMinValue: 50, NavSkipSec: 150, AreaIdleSec: 60, StallWarnMin: 30, StallExitMin: 30, PauseDeathsPerHour: 8, PauseCapMin: 60, StalePackHostiles: 4, StalePackMinutes: 10, StalePackMinHP: 5, StalePackMinTier: 2, PotionBurstN: 5, PotionBurstSec: 20, PotionBurstHostiles: 3, PovertyTier: 0, GoldFloor: 2000,
		PointsRetrySec: 30, GearCheckSec: 60, GearMinGold: 150, GearBudgetPct: 70, GearReserve: 100,
		SellMinItems: 4, DifficultyBanMin: 60, SavePointMin: 20, HeapLimitMB: 4096, FailsafeRung3Max: 6, FailsafePauseMaxSec: 3600, FlailMoves: 6, FlailWindowMin: 10, FlailHoldMin: 5,
		Invariants: 1, TypedOutcome: 1, ArbiterShadow: 0, ArbiterDwellSec: 120, ArbiterTravelSec: 600, ArbiterLogGapSec: 60, ArbiterControl: 0, ArbiterFamilies: 3, TickLogSec: 0, Admission: 0,
		Rung3WindowMax: 3, Rung3WindowMin: 20, DroughtMin: 8, EmptyFieldSec: 150, ClearIdleMin: 3, NoPortalWaitSec: 20, PortalRetryWaitSec: 3, CheckpointEnable: 0, CheckpointMinLifePct: 80, CheckpointRetreatPct: 40, CheckpointGapSec: 60, CheckpointMinScrolls: 2, CheckpointSafeRadius: 30, CheckpointEntryDist: 40, CheckpointReach: 60, CheckpointArrivalHostiles: 4, DroughtStage2Min: 2, DroughtSlowMin: 10, DroughtSlowKills: 3, DroughtSlowMaxLevel: 10, FreshAfterRecoverySec: 180, RecoveryDryMin: 5, RecoveryDryKills: 5, RecoveryDrySkipMin: 15,
		ZoneModel: 0, ZoneDial: 1, PortalGuard: 0, PortalCastMs: 3000, PortalWalkMs: 3000, PortalHitMs: 700, PortalLifePct: 50,
		MercRetryMin: 10, MercHireMinGold: 400, MercReviveLoopN: 3, MercReviveLoopMin: 10, CubeMaxJobs: 6, CubeRuneCap: 10, CubeGemCap: 2,
	}
}

func botKnobs() []Knob {
	return []Knob{
		{Key: "bot.zoneModel", Label: "Zone choice model", Unit: "mode", Default: 0, Min: 0, Max: 1, HigherIsBetter: false, NoHandicap: true,
			Desc: "0 (default until the A/B of docs/survival/ab-harness.md passes): the plan tables choose the leveling zone. 1: the survival controller chooses among the plan's zones by power-vs-danger margin and expected value (docs/survival/zonemodel.md); any controller error falls back to the plan tables with a log line. Also lets the margin replace the Andariel level floor when boss margin data exists, may leave a dead merc dead, and may ask for a fresh game when the best zone is cleared."},
		{Key: "bot.zoneDial", Label: "Zone risk dial", Unit: "mode", Default: 1, Min: 0, Max: 2, HigherIsBetter: false, NoHandicap: true,
			Desc: "Risk setting of the margin model (bot.zoneModel 1): 0 cautious, 1 balanced, 2 bold. Moves the enter and stay lines of the margin; read only by the model."},
		{Key: "bot.portalGuard", Label: "Portal cast guard", Unit: "mode", Default: 0, Min: 0, Max: 1, HigherIsBetter: true, NoHandicap: true,
			Desc: "0 (default until drill R3, docs/fixportalmelee.md): three or more adjacent attackers block a town portal cast (the old count rule). 1: block it when the expected damage of the adjacent attackers over the cast and the walk reaches bot.portalLifePct of current life; with any damage term missing the count rule applies. A hit does not interrupt the cast (V run, decomp28), so the risk is damage taken, not a cut."},
		{Key: "bot.portalCastMs", Label: "Portal cast time", Unit: "ms", Default: 3000, Min: 500, Max: 5000,
			Desc: "Time of the town portal scroll or tome read used by the guard (generous upper bound, U, drill R3 measures it). Longer blocks more casts."},
		{Key: "bot.portalWalkMs", Label: "Portal walk time", Unit: "ms", Default: 3000, Min: 0, Max: 5000,
			Desc: "Walk from the cast spot into the portal used by the guard (generous upper bound, U, drill R3 measures it). Longer blocks more casts."},
		{Key: "bot.portalHitMs", Label: "Portal guard hit interval", Unit: "ms", Default: 700, Min: 300, Max: 3000, HigherIsBetter: true,
			Desc: "Assumed time between two hits of one adjacent monster in the guard (its strongest attack lands every time: an upper bound, U). Shorter blocks more casts."},
		{Key: "bot.portalLifePct", Label: "Portal guard life share", Unit: "%", Default: 50, Min: 10, Max: 100, HigherIsBetter: true,
			Desc: "The guard blocks a portal cast when the expected damage over the exposure reaches this share of current life. Lower blocks more casts."},
		{Key: "bot.tickMs", Label: "Decision period", Unit: "ms", Default: 100, Min: 100, Max: 1000,
			Desc: "How often the bot re-plans between actions. Slower means more idle gaps."},
		{Key: "bot.watchMs", Label: "Watch period", Unit: "ms", Default: 50, Min: 50, Max: 300,
			Desc: "How often events, hazards and death are checked while an action runs. Above 300 the bot dies on Hell packs."},
		{Key: "bot.reactionMs", Label: "Reaction time", Unit: "ms", Default: 0, Min: 0, Max: 300,
			Desc: "Extra delay before reacting to low life or a chicken condition."},
		{Key: "bot.resupplyPotions", Label: "Resupply threshold", Unit: "potions", Default: 2, Min: 2, Max: 4,
			Desc: "Healing potions left in the belt that trigger a town trip. Higher means more trips."},
		{Key: "bot.invFullPct", Label: "Inventory full", Unit: "%", Default: 85, Min: 40, Max: 85, HigherIsBetter: true,
			Desc: "Share of occupied inventory cells that triggers a town trip. Lower means more trips."},
		{Key: "bot.levelMargin", Label: "Entry level margin", Unit: "levels", Default: 0, Min: 0, Max: 8,
			Desc: "Levels above a step's enter level the planner requires. Higher is safer but slower."},
		{Key: "bot.leaveSlack", Label: "Overstay", Unit: "levels", Default: 0, Min: 0, Max: 6,
			Desc: "Levels past a step's leave level before moving to the next area. Higher wastes time in easy areas."},
		{Key: "bot.corpseRetries", Label: "Corpse retrievals", Unit: "count", Default: 3, Min: 0, Max: 3, HigherIsBetter: true,
			Desc: "Attempts to recover the corpse in the field after a softcore death, whatever it is worth. After the last failed try the game is reset: the corpse moves to town and is taken there. 0 abandons the gear."},
		{Key: "bot.portalSettleMs", Label: "Town settle time", Unit: "ms", Default: 600, Min: 600, Max: 4500,
			Desc: "Pause in town after chores before going back out."},
		{Key: "bot.clearRadius", Label: "Clear radius", Unit: "subtiles", Default: 30, Min: 12, Max: 45, HigherIsBetter: true,
			Desc: "Reach of one clear sweep: how far away a monster is noticed and fought. Smaller means more walking between sweeps; above 30 is the Aggressive stance (UNVERIFIED survival)."},
		{Key: "bot.exploredPct", Label: "Explored share to move on", Unit: "%", Default: 100, Min: 30, Max: 100, HigherIsBetter: false, NoHandicap: true,
			Desc: "Share of a zone's roam coverage points the hero must have visited before a clear run calls the zone explored and moves on (the planner then takes the next area or a fresh game). 100 visits every point (the old behaviour). Quest and boss runs always explore everything. Needs a roaming (fully known map) backend; others ignore it. docs/exploredpct.md."},
		{Key: "bot.farmReset", Label: "Farm loop", Unit: "bool", Default: 0, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1: in a leveling zone, once its coverage reaches bot.farmResetPct and a fresh game pays (the zone's current XP per minute is below what a reset yields after the 15 s restart and the walk back, measured in this run), the hero goes home, takes a fresh game (monsters respawn) and returns to the same zone. Never during a fight, a quest step, a boss run, a named run, a recovery or within 180 s of one. docs/farmloop.md."},
		{Key: "bot.farmResetPct", Label: "Farm reset at explored share", Unit: "%", Default: 85, Min: 50, Max: 100, NoHandicap: true,
			Desc: "Share of a zone's roam coverage points that must be visited before the farm loop may reset it (bot.farmReset). Keep it at or above bot.exploredPct, which would end the clear first. A zone below this share is never reset."},
		{Key: "bot.farmResetCooldownMin", Label: "Farm reset cooldown", Unit: "min", Default: 5, Min: 1, Max: 60, NoHandicap: true,
			Desc: "Least minutes between two farm resets: a reset that did not pay cannot repeat at once."},
		{Key: "bot.reflexDrink", Label: "Reflex drink", Unit: "bool", Default: 1, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1: when the safety check wants a life or rejuvenation potion from the belt, the guard sends the key itself through the input controller and does not cancel the running action (a drink needs no stop). 0: the old way, the guard cancels the action and the loop drinks after it returns. A drink that needs the pack, a portal, a chicken and a crowd retreat always cancel. Measured by safetyLatAvgMs, safetyLatMaxMs and reflexDrinks on the minute line."},
		{Key: "bot.reflexDeadlineMs", Label: "Reflex deadline", Unit: "ms", Default: 1000, Min: 0, Max: 5000, NoHandicap: true,
			Desc: "A portal, chicken or crowd cancel gives the running action this long to return; after that the guard carries the safety action out itself, so a stuck action cannot hold a flee back. 0 turns the deadline off."},
		{Key: "bot.lootShadow", Label: "Loot rules shadow", Unit: "bool", Default: 1, Min: 0, Max: 1, HigherIsBetter: false, NoHandicap: true,
			Desc: "1: the loot rules engine (docs/lootrules.md) judges every ground pickup, tidy and sell decision next to today's code and logs the differences as `loot shadow: ...` lines (read-only report at /api/lootrules/shadow). Log only: it never changes what the bot does. 0 turns it off."},
		{Key: "bot.corpseValueMin", Label: "Corpse value threshold", Unit: "gold", Default: 1000, Min: 1, Max: 20000,
			Desc: "Only the older-body search reads it now: a body worth less than this does not count as the one holding the gear. The corpse run itself no longer depends on value (bot.corpseRetries tries, then a fresh game moves the corpse to town)."},
		{Key: "bot.spiralDeaths", Label: "Spiral breaker deaths", Unit: "deaths", Default: 4, Min: 3, Max: 8,
			Desc: "Deaths in one area within the spiral window that make the bot avoid that area (one fewer when the area is clearly too high for the level or the character died with no potions and no gold). Dying is normal: the default allows three deaths on the same goal. Higher tolerates more."},
		{Key: "bot.dangerBanMin", Label: "Area ban length", Unit: "min", Default: 20, Min: 5, Max: 120,
			Desc: "How long the spiral breaker keeps the character out of an area. Longer stalls the goal more."},
		{Key: "bot.starterBanMin", Label: "Starter area ban cap", Unit: "min", Default: 3, Min: 1, Max: 30,
			Desc: "A ban of the first field of an act is cut to this once the character stands in town because of it (it is the only way out)."},
		{Key: "bot.repeatSiteBanMin", Label: "Repeat-site ban length", Unit: "min", Default: 30, Min: 0, Max: 120,
			Desc: "A second death in one area within the repeat-site window, or a death within the repeat-site gap of the previous one, bans the area this long and writes a planner dump. 0 turns the rule off. Dying is normal: this changes where the bot farms, not how careful it is."},
		{Key: "bot.repeatSiteEscalate", Label: "Repeat-site ban escalation", Unit: "bool", Default: 1, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1: the second repeat-site ban of the same area within 12 hours lasts 4 times the ban length, the third and later 12 times (capped at 6 hours). It stops a bot from walking back into the same death spot every half hour. 0 keeps every ban at the base length."},
		{Key: "bot.repeatSiteWindowMin", Label: "Repeat-site window", Unit: "min", Default: 30, Min: 5, Max: 120,
			Desc: "Two deaths in one area within this many minutes make it a repeat site."},
		{Key: "bot.repeatSiteGapSec", Label: "Repeat-site gap", Unit: "s", Default: 180, Min: 30, Max: 600,
			Desc: "A death within this many seconds of the previous death (any area) makes the area a repeat site."},
		{Key: "bot.corpseMinValue", Label: "Corpse run minimum value", Unit: "gold", Default: 50, Min: 0, Max: 5000,
			Desc: "No longer read: a corpse with anything on it is chased (user 2026-11-25); one worth 0 is skipped."},
		{Key: "bot.navSkipSec", Label: "Nav-stuck area skip", Unit: "s", Default: 150, Min: 30, Max: 900,
			Desc: "How long a navigation fault (stuck, no exit to the area) keeps the planner out of an area. Short on purpose: it is a bot fault, not danger."},
		{Key: "bot.areaIdleSec", Label: "Longest wait for area bans", Unit: "s", Default: 60, Min: 15, Max: 600,
			Desc: "When every candidate area is skipped or banned, the bot waits this long in town at most, then farms the least bad area anyway (a death streak waits at least the starter ban cap)."},
		{Key: "bot.questBlockedFarmMin", Label: "Blocked quest step: farm after", Unit: "min", Default: 10, Min: 1, Max: 120,
			Desc: "A quest step the guards refuse is waited out this long; then the hero farms the best leveling area of the level plan and retries the step after this wait, doubling each time (cap x64). Shown as journeyBlocked in the status."},
		{Key: "bot.parkAfter", Label: "Park a quest step after blocks", Unit: "blocks", Default: 2, Min: 0, Max: 10,
			Desc: "A quest step that fails with 'not found' this many times (planning blocked, the same step) is parked: the hero farms the level plan instead of waiting in town. 0 turns the rule off."},
		{Key: "bot.parkMinutes", Label: "Parked quest step: retry after", Unit: "min", Default: 15, Min: 1, Max: 240,
			Desc: "How long a parked quest step stays parked before the planner offers it again. The reason shows as journeyBlocked in the status."},
		{Key: "bot.stallWarnMin", Label: "Campaign stall: warn and fresh game", Unit: "min", Default: 30, Min: 0, Max: 240,
			Desc: "Virtual minutes without campaign progress (no kill or xp, no non-town area entered, no purchase, no quest step) before the bot logs 'campaign stalled' and starts a fresh game. 0 turns the stall ladder off. Longest legitimate stretch in the navsoak logs: 19 min."},
		{Key: "bot.stallExitMin", Label: "Campaign stall: exit window", Unit: "min", Default: 30, Min: 5, Max: 240,
			Desc: "Virtual minutes still without campaign progress after the fresh game before the bot exits (status 5) so the supervisor restarts the whole process."},
		{Key: "bot.spiralMinutes", Label: "Spiral breaker window", Unit: "min", Default: 15, Min: 5, Max: 15, HigherIsBetter: true,
			Desc: "Window in which the deaths are counted. Shorter forgets sooner."},
		{Key: "bot.pauseDeathsPerHour", Label: "Pause after deaths per hour", Unit: "deaths", Default: 8, Min: 8, Max: 20,
			Desc: "Deaths in the last hour after which the bot pauses itself and asks the user."},
		{Key: "bot.pauseCapMin", Label: "Player Pause: resume after", Unit: "min", Default: 60, Min: 0, Max: 720,
			Desc: "A Pause the player asked for (the frozen game) ends by itself after this many wall minutes and a Needs you row says how long it was. 0 = no cap."},
		{Key: "bot.stalePackHostiles", Label: "Stale pack size", Unit: "count", Default: 4, Min: 0, Max: 12,
			Desc: "Hostiles near the hero when it left a field by chicken, town portal or crowd exit that make the area a stale pack: no return by portal and no planner pick of that area until bot.stalePackMinutes pass or the kit is rebuilt. 0 turns the rule off. Seed 13 of Gate 2 g2f died 2 s after re-entering a 12-hostile pack at clvl 4."},
		{Key: "bot.stalePackMinutes", Label: "Stale pack hold", Unit: "min", Default: 10, Min: 1, Max: 60,
			Desc: "How long a stale pack keeps its area closed. Monsters of a live game stay where they are (U), so the hold is a guess at the time the kit needs to be rebuilt."},
		{Key: "bot.stalePackMinHP", Label: "Rebuilt kit: healing potions", Unit: "count", Default: 5, Min: 1, Max: 8,
			Desc: "Healing potions (belt and pack) that lift the stale-pack hold early, together with bot.stalePackMinTier."},
		{Key: "bot.stalePackMinTier", Label: "Rebuilt kit: potion tier", Unit: "tier", Default: 2, Min: 1, Max: 3,
			Desc: "Best healing potion tier in the belt that lifts the stale-pack hold early (1 Minor Healing, 2 Light Healing)."},
		{Key: "bot.potionBurstN", Label: "Potion burst: drinks", Unit: "count", Default: 5, Min: 0, Max: 12,
			Desc: "Life drinks within bot.potionBurstSec that turn the next drink into a town portal while bot.potionBurstHostiles or more hostiles are near and a portal exists. 0 turns the rule off. Seed 13 drank 11 potions in about 17 s and then died."},
		{Key: "bot.potionBurstSec", Label: "Potion burst: window", Unit: "s", Default: 20, Min: 5, Max: 60,
			Desc: "Window in which the life drinks are counted."},
		{Key: "bot.potionBurstHostiles", Label: "Potion burst: hostiles", Unit: "count", Default: 3, Min: 1, Max: 8,
			Desc: "Hostiles within the fight radius the potion burst rule needs before it retreats."},
		{Key: "bot.povertyTier", Label: "Poverty gear tier", Unit: "tier", Default: 0, Min: 0, Max: 2,
			Desc: "Gear tier bought after an abandoned corpse: 0 is the cheapest viable weapon and armor, potions first."},
		{Key: "bot.goldFloor", Label: "Gold floor", Unit: "gold", Default: 2000, Min: 0, Max: 2000, HigherIsBetter: true,
			Desc: "Gold town visits must not spend, kept to rebuy gear after a death."},
		{Key: "bot.pointsRetrySec", Label: "Point spending retry", Unit: "s", Default: 30, Min: 30, Max: 300,
			Desc: "Wait before spending stat and skill points again after an attempt that did not finish. Longer leaves points unspent."},
		{Key: "bot.gearCheckSec", Label: "Gear check gap", Unit: "s", Default: 60, Min: 60, Max: 600,
			Desc: "Minimum time between gear passes in town (equip better items, shop). Longer wears worse gear for longer."},
		{Key: "bot.gearMinGold", Label: "Gear shopping gold", Unit: "gold", Default: 150, Min: 150, Max: 1500,
			Desc: "Gold needed before the bot shops for weapons and armor. Higher shops rarely."},
		{Key: "bot.gearBudgetPct", Label: "Gear budget", Unit: "%", Default: 70, Min: 20, Max: 70, HigherIsBetter: true,
			Desc: "Share of the gold above the potion reserve one shopping pass may spend on gear."},
		{Key: "bot.gearReserve", Label: "Potion reserve", Unit: "gold", Default: 100, Min: 0, Max: 100, HigherIsBetter: true,
			Desc: "Gold kept for potions when shopping for gear. Lower risks starving after a shopping spree."},
		{Key: "bot.sellMinItems", Label: "Sell trip items", Unit: "items", Default: 4, Min: 4, Max: 12,
			Desc: "Sellable pack items that justify a town trip for a character without gold. Higher carries loot longer."},
		{Key: "bot.difficultyBanMin", Label: "Difficulty retry ban", Unit: "min", Default: 60, Min: 60, Max: 600,
			Desc: "Minutes the bot stays in its difficulty after a switch to the next one did not happen."},
		{Key: "bot.diffReadyV2", Label: "Difficulty readiness v2", Unit: "bool", Default: 0, Min: 0, Max: 1, NoHandicap: true,
			Desc: "[dark-launch] 1 replaces the clvl-only ready-for-next-difficulty check by the per-class predicate (internal/diffready: Baal dead, clvl, life, Nightmare resists, kit, merc, deaths per hour) and makes the hop opt-in: the bot shows readiness in ops why and waits for the owner's go, never hops alone. 0 is off: nothing changes (docs/diffready.md, data/darklaunch.json)."},
		{Key: "bot.savePointMin", Label: "Save point interval", Unit: "min", Default: 20, Min: 0, Max: 120, NoHandicap: true,
			Desc: "Minutes in one game before the bot, in town and idle, saves and re-enters it (a crash loses what came after the last save); 0 is off."},
		{Key: "bot.heapLimitMB", Label: "Heap guard limit", Unit: "MB", Default: 4096, Min: 0, Max: 8192, NoHandicap: true,
			Desc: "Go heap in use above which d2bot writes a heap profile and exits for a supervisor restart (a runaway allocation would otherwise end in the OOM killer); the GC gets a soft limit 25 percent above it. 0 is off. Normal runs use under 100 MB."},
		{Key: "bot.failsafeRung3Max", Label: "Failsafe re-entry cap", Unit: "count", Default: 6, Min: 0, Max: 50, NoHandicap: true,
			Desc: "Leave-and-re-enter cycles of failsafe ladder rung 3 allowed without real progress (XP, kills, levels) before the ladder skips to the pause rung. 0 is unbounded."},
		{Key: "bot.rung3WindowMax", Label: "Re-entry burst cap", Unit: "count", Default: 3, Min: 0, Max: 20, NoHandicap: true,
			Desc: "Failsafe leave-and-re-enter cycles allowed inside the burst window even when kills come between them. One more ends in a field reset (skip the area, no new game). 0 is off."},
		{Key: "bot.rung3WindowMin", Label: "Re-entry burst window", Unit: "min", Default: 20, Min: 1, Max: 240, NoHandicap: true,
			Desc: "Virtual minutes the re-entry burst cap counts over."},
		{Key: "bot.droughtMin", Label: "Kill drought limit", Unit: "min", Default: 8, Min: 0, Max: 60, NoHandicap: true,
			Desc: "Virtual minutes without a kill or XP gain in a field area at healthy life before the bot leaves the area (second time in a row: a fresh game). Keep below the Gate 2 no-kill limit of 10. 0 is off."},
		{Key: "bot.droughtStage2Min", Label: "Kill drought, second stage", Unit: "min", Default: 2, Min: 0, Max: 30, NoHandicap: true,
			Desc: "Virtual minutes after the first drought action (leave the area) before the second (a fresh game). The old rule waited a full bot.droughtMin, which can never help under the Gate 2 limit of 10. 0 uses bot.droughtMin."},
		{Key: "bot.droughtSlowMin", Label: "Slow field window", Unit: "min", Default: 10, Min: 0, Max: 60, NoHandicap: true,
			Desc: "Early game: after this many minutes in one field area with fewer than bot.droughtSlowKills kills in them (a depleted Blood Moor gives a straggler every few minutes, which keeps bot.droughtMin quiet; V, tester realtime drill: 45 kills in 25 min, then one per 1 to 5 min) the bot starts a fresh game, which refills the field. 0 is off."},
		{Key: "bot.droughtSlowKills", Label: "Slow field kills", Unit: "kills", Default: 3, Min: 1, Max: 30, NoHandicap: true,
			Desc: "Kills the slow-field window must hold to go on in the same game."},
		{Key: "bot.droughtSlowMaxLevel", Label: "Slow field below clvl", Unit: "level", Default: 10, Min: 0, Max: 40, NoHandicap: true,
			Desc: "The slow-field rule applies below this clvl only (small early fields deplete; later levels clear on purpose). 0 is off."},
		{Key: "bot.recoveryDryMin", Label: "Dry recovery window", Unit: "min", Default: 5, Min: 0, Max: 30, NoHandicap: true,
			Desc: "While a recovery farms gold, after this many minutes in one field area with fewer than bot.recoveryDryKills kills in them (a depleted Blood Moor: 141 s since the last kill, V realtime Paladin drill) the hero moves on to the next area the recovery rules allow, or starts a fresh game when there is none. The safe-range rules (route guard, kit gate, bans) still decide which areas qualify. 0 is off."},
		{Key: "bot.recoveryDryKills", Label: "Dry recovery kills", Unit: "kills", Default: 5, Min: 1, Max: 60, NoHandicap: true,
			Desc: "Kills the dry-recovery window must hold to stay in the same area."},
		{Key: "bot.recoveryDrySkipMin", Label: "Dry recovery: area skip", Unit: "min", Default: 15, Min: 1, Max: 60, NoHandicap: true,
			Desc: "Minutes the recovery stays out of an area it left as dry (a new game or a respawn refills it)."},
		{Key: "bot.noPortalWaitSec", Label: "No portal: leave after", Unit: "s", Default: 20, Min: 0, Max: 240, NoHandicap: true,
			Desc: "Low life, no potion and the way to town fails because there is no town portal (and no walk back for this area): seconds after the first such failure before the hero leaves the game, which restarts it in town. 240 keeps the old two minute waits."},
		{Key: "bot.portalRetryWaitSec", Label: "Cancelled portal: retry after", Unit: "s", Default: 3, Min: 0, Max: 30, NoHandicap: true,
			Desc: "A town portal cast cancelled before the click (a drink or a retreat cut it, nothing spent) is retried once as a town trip after this many seconds. 0 is off: the next planning tick decides."},
		{Key: "bot.checkpointEnable", Label: "Checkpoint portals", Unit: "bool", Default: 0, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1 casts a town portal on purpose at a danger area while the hero is healthy and nothing is near, retreats to it at low life and returns through it after a death. 0 is off (docs/design/danger-protocol-2026-11-25.md)."},
		{Key: "bot.checkpointMinLifePct", Label: "Checkpoint cast: life at least", Unit: "%", Default: 80, Min: 50, Max: 100, NoHandicap: true,
			Desc: "A checkpoint portal is cast only at or above this life percent."},
		{Key: "bot.checkpointRetreatPct", Label: "Checkpoint retreat: life at most", Unit: "%", Default: 40, Min: 10, Max: 70, NoHandicap: true,
			Desc: "At or below this life percent a live checkpoint within reach is walked to and taken (potions and the crowd rule act first)."},
		{Key: "bot.checkpointGapSec", Label: "Checkpoint cast gap", Unit: "s", Default: 60, Min: 10, Max: 600, NoHandicap: true,
			Desc: "At least this many seconds between two checkpoint casts."},
		{Key: "bot.checkpointMinScrolls", Label: "Checkpoint scroll reserve", Unit: "count", Default: 2, Min: 0, Max: 10, NoHandicap: true,
			Desc: "A checkpoint is cast only with more portal casts than this in the pack, so the last ones stay for escapes."},
		{Key: "bot.checkpointSafeRadius", Label: "Checkpoint cast: nothing hostile within", Unit: "subtiles", Default: 30, Min: 10, Max: 60, NoHandicap: true,
			Desc: "The checkpoint spot must have no hostile this close."},
		{Key: "bot.checkpointEntryDist", Label: "Checkpoint cast: near the entry", Unit: "subtiles", Default: 40, Min: 5, Max: 120, NoHandicap: true,
			Desc: "The cast happens at most this far from where the hero entered the area (the entrance of a cleared stretch)."},
		{Key: "bot.checkpointReach", Label: "Checkpoint retreat reach", Unit: "subtiles", Default: 60, Min: 10, Max: 200, NoHandicap: true,
			Desc: "A retreat to the checkpoint is tried only when the portal is at most this far away."},
		{Key: "bot.checkpointArrivalHostiles", Label: "Checkpoint return: crowd at the exit", Unit: "count", Default: 4, Min: 1, Max: 20, NoHandicap: true,
			Desc: "After a death, arriving through the checkpoint portal into this many hostiles near the exit sends the hero straight back to town."},
		{Key: "bot.freshAfterRecoverySec", Label: "Quiet after recovery", Unit: "s", Default: 180, Min: 0, Max: 900, NoHandicap: true,
			Desc: "No unplanned fresh game (no target left, drought) this long after a lost-game recovery, unless life is below the resume line. A second restart right after the recovery fails the Gate 2 fault criterion. 0 is off."},
		{Key: "bot.emptyFieldSec", Label: "Empty quest field", Unit: "s", Default: 150, Min: 0, Max: 900, NoHandicap: true,
			Desc: "A quest-run area with no hostile in sight and no kill for this long counts as cleared, so the route moves on. Other areas follow the sweep setting. 0 is off."},
		{Key: "bot.clearIdleMin", Label: "Idle leveling field", Unit: "min", Default: 3, Min: 0, Max: 30, NoHandicap: true,
			Desc: "A leveling area where the hero has killed nothing for this long counts as cleared, so the planner moves on to the next area (a human leaves an emptied field). 0 is off."},
		{Key: "bot.failsafePauseMaxSec", Label: "Failsafe pause cap", Unit: "s", Default: 3600, Min: 300, Max: 14400, NoHandicap: true,
			Desc: "Longest failsafe self pause. The pause starts at 5 minutes and doubles with every consecutive pause without real progress."},
		{Key: "bot.flailMoves", Label: "Flail: area changes", Unit: "moves", Default: 6, Min: 0, Max: 100, NoHandicap: true,
			Desc: "This many area changes inside the flail window without a kill or XP count as a loop (any cause: the quest board, area done and waypoint fallbacks undoing each other). The bot then holds in place. 0 is off."},
		{Key: "bot.flailWindowMin", Label: "Flail: window", Unit: "min", Default: 10, Min: 1, Max: 120, NoHandicap: true,
			Desc: "Length of the window in which the area changes of the flail guard are counted."},
		{Key: "bot.flailHoldMin", Label: "Flail: hold", Unit: "min", Default: 5, Min: 1, Max: 120, NoHandicap: true,
			Desc: "How long the bot stays in the current area (fights in the field, rests in town) after a flail was found. Doubles for each repeat within an hour, at most eight times."},
		{Key: "bot.invariants", Label: "Invariant monitor", Unit: "bool", Default: 1, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1 checks about twelve named invariants once per tick (a potion not drunk under the drink line, a field entry without the kit, a refusal counted as a failure, ...) and logs one rate-limited `invariant` line per hit plus per-name counts in the minute line. Logging only, nothing changes for the hero. 0 is off (docs/invariants.md)."},
		{Key: "bot.typedOutcome", Label: "Typed step outcome", Unit: "bool", Default: 1, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1 makes a refusal of the quest board step (early road, route guard, gate: not ready) a typed Refused outcome with a need and a wake fact: it is handled in one place, never counted as a failure, and logged once per change as `step refused`. 0 is the legacy path (the refusal goes through the failure counter). Other paths are not converted yet (docs/outcomes.md)."},
		{Key: "bot.arbiterShadow", Label: "Arbiter shadow", Unit: "bool", Default: 0, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1 runs the intent arbiter beside the goal chain and logs one line per divergence (chain chose A, arbiter would choose B because ...) plus per-family counters in the minute line. It never acts: nothing changes for the hero. 0 is off (docs/arbiter.md)."},
		{Key: "bot.arbiterControl", Label: "Arbiter control", Unit: "bool", Default: 0, Min: 0, Max: 1, NoHandicap: true,
			Desc: "[dark-launch] 1 lets the intent arbiter act for the families in bot.arbiterFamilies: the goal chain's hooks of such a family defer to its decision, and the old chain is the fallback when the arbiter chooses nothing or fails. Needs bot.arbiterShadow 1. 0 is off: nothing changes (docs/arbiter.md, data/darklaunch.json)."},
		{Key: "bot.arbiterFamilies", Label: "Arbiter control families", Unit: "mask", Default: 3, Min: 0, Max: 255, NoHandicap: true,
			Desc: "Arbiter control: which families the arbiter controls, one bit each in the order survive 1, player 2, chores 4, claim 8, goldfarm 16, recovery 32, quest 64, level 128. 3 is survive and player, the first family. Every bit is wired (arbfam entries 1 to 4); with bot.arbiterControl 0 none acts (docs/arbiter.md)."},
		{Key: "bot.tickLogSec", Label: "Tick log", Unit: "s", Default: 0, Min: 0, Max: 60, NoHandicap: true,
			Desc: "[dark-launch] Above 0 writes one `tick` line at most every this many seconds: area, kills, xp, the chain's hook family and the arbiter's world facts, so `d2arbiter -follow` can replay a recorded session. Log only, independent of bot.arbiterShadow. 0 is off (docs/design/ticklog-t4.md, data/darklaunch.json)."},
		{Key: "bot.admission", Label: "Admission function", Unit: "bool", Default: 0, Min: 0, Max: 1, NoHandicap: true,
			Desc: "[dark-launch] 1 lets the admission function (internal/admit, rows in admission.json) decide the early road, poverty, kit floor and lean entry rows at the travel chokepoint and the quest-walk kit at the town exit. 0 keeps the old judges in control; the function runs beside them and logs one `admission shadow` line per disagreement (docs/design/admission-p4.md, data/darklaunch.json)."},
		{Key: "bot.arbiterDwellSec", Label: "Arbiter lease", Unit: "s", Default: 120, Min: 10, Max: 1800, NoHandicap: true,
			Desc: "Arbiter shadow: how long a chosen intent holds its lease; only a higher priority class preempts it inside this time."},
		{Key: "bot.arbiterTravelSec", Label: "Arbiter travel window", Unit: "s", Default: 600, Min: 30, Max: 7200, NoHandicap: true,
			Desc: "Arbiter shadow: an area change that would undo the previous one inside this window is refused unless a kill, XP, gold or belt change happened in between."},
		{Key: "bot.arbiterLogGapSec", Label: "Arbiter log gap", Unit: "s", Default: 60, Min: 1, Max: 3600, NoHandicap: true,
			Desc: "Arbiter shadow: minimum seconds between two divergence lines of the same (chain, arbiter) pair."},
		// Merc and cube chores: switches and policy, not skill, so no handicap profile moves them (NoHandicap).
		{Key: "bot.mercRetryMin", Label: "Merc chore retry", Unit: "min", Default: 10, Min: 1, Max: 60, NoHandicap: true,
			Desc: "Minutes between merc chore attempts; a failed hire or revive is not retried sooner."},
		{Key: "bot.mercHireMinGold", Label: "Merc hire gold", Unit: "gold", Default: 400, Min: 0, Max: 2000, NoHandicap: true,
			Desc: "Gold in Act 2 town below which no hire is attempted."},
		{Key: "bot.mercReviveUnverified", Label: "Merc revive (unverified)", Unit: "bool", Default: 0, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1 lets the town visit click the revive dialog, which is UNVERIFIED live (docs/merc-farm.md). Off until a live probe."},
		{Key: "bot.mercReviveLoopN", Label: "Merc revive loop limit", Unit: "count", Default: 3, Min: 0, Max: 10, NoHandicap: true,
			Desc: "Only with bot.zoneModel 1. After this many revives that each ended in the merc's death within bot.mercReviveLoopMin minutes in the same zone, the model stops reviving him while that zone is its plan (it stays closed for the survival noReviveMinutes, 60). A death after a longer life resets the count. 0 turns the breaker off. Every revive and every death logs the merc's time to death either way."},
		{Key: "bot.mercReviveLoopMin", Label: "Merc revive loop window", Unit: "min", Default: 10, Min: 1, Max: 120, NoHandicap: true,
			Desc: "A merc who dies within this many minutes of a revive counts toward bot.mercReviveLoopN (seed 103 of the screen: 4 revives, each lived about a minute)."},
		{Key: "bot.cubeJobs", Label: "Cube chore", Unit: "bool", Default: 0, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1 runs the gold-free gem and rune upgrade chore in town. UNVERIFIED live (docs/items.md), so off."},
		{Key: "bot.cubeMaxJobs", Label: "Cube jobs per visit", Unit: "count", Default: 6, Min: 1, Max: 12, HigherIsBetter: true, NoHandicap: true,
			Desc: "Transmutes per town visit when the cube chore is on."},
		{Key: "bot.cubeRuneCap", Label: "Cube rune cap", Unit: "rune", Default: 10, Min: 1, Max: 10, HigherIsBetter: true, NoHandicap: true,
			Desc: "Highest rune number the chore produces (10 = Thul; Dol to Hel and above are ladder-flagged, docs/items.md Q1)."},
		{Key: "bot.cubeGemCap", Label: "Cube gem cap", Unit: "grade", Default: 2, Min: 0, Max: 4, HigherIsBetter: true, NoHandicap: true,
			Desc: "Highest gem grade the chore produces: 0 chipped, 1 flawed, 2 normal, 3 flawless, 4 perfect."},
	}
}

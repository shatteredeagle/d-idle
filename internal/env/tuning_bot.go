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
	CorpseRetries   int // corpse retrievals attempted per death (softcore); 0 abandons the corpse
	PortalSettleMs  int // wait in town after chores before returning to the field
	ClearRadius     int // radius of one clear sweep

	// Death policy (PLAN.md "Death is normal"; docs/bot.md "Death").
	CorpseValueMin      int // estimated gold value of the lost gear below which the corpse is abandoned
	SpiralDeaths        int // deaths in one area within SpiralMinutes that trigger the death-spiral breaker
	SpiralMinutes       int // window of the spiral breaker
	DangerBanMin        int // minutes the breaker keeps the character out of an area (a Config.DangerBanSec above 0 overrides)
	NavSkipSec          int // seconds a nav fault (stuck, no exit) keeps the planner out of an area
	StallWarnMin        int // campaign stall ladder (internal/bot/stall.go): virtual minutes without campaign progress before the WARN and a fresh game (0 = ladder off)
	StallExitMin        int // virtual minutes still without progress after the fresh game before the bot exits for a supervisor restart
	AreaIdleSec         int // longest wait in town for area bans that are not a death streak before the least bad area is farmed anyway
	StarterBanMin       int // longest ban of the first field of an act once the character stands in town because of it
	RepeatSiteBanMin    int // minutes a repeat-site death keeps the character out of the area (0 = rule off)
	RepeatSiteEscalate  int // 1: the second repeat-site ban in one area within 12 h lasts 4x, the third 12x (cap 6 h)
	RepeatSiteWindowMin int // window in which a second death in one area counts as a repeat site
	RepeatSiteGapSec    int // a death this soon after the previous one (any area) counts as a repeat site
	CorpseMinValue      int // gold-equivalent below which the in-field corpse run is skipped outright
	PauseDeathsPerHour  int // deaths in the last hour (any area) after which the bot pauses itself
	PovertyTier         int // gear tier bought when rebuilding from vendors after an abandoned corpse: 0 cheapest viable
	GoldFloor           int // gold the town visit must not spend, kept for rebuying gear

	// Leveling (docs/leveler.md, docs/bot.md "Leveling"): points, gear, poverty, difficulty.
	PointsRetrySec   int // wait before spending unspent points again after a failed attempt
	GearCheckSec     int // minimum gap between gear passes (equip from the pack, shop) in town
	GearMinGold      int // gold needed before the bot shops for gear
	GearBudgetPct    int // share of the gold above GearReserve a shopping pass may spend
	GearReserve      int // gold kept for potions when shopping for gear
	SellMinItems     int // sellable pack items that justify a town trip for a starving character
	DifficultyBanMin int // minutes the bot stays in its difficulty after a switch to a higher one failed

	// SavePointMin: the game writes the .d2s only on Save and Exit (V: docs/soak-analysis.md 6c), so a crash loses all
	// progress since. In town, idle, the bot leaves and re-enters the game after this many minutes. 0 = off.
	SavePointMin int

	// HeapLimitMB: d2bot's heap guard (cmd/d2bot/heapguard.go) writes a heap profile and exits for a supervisor restart
	// when the Go heap in use passes this many MB (G1 attempt 16: a runaway took 28 GB in 30 s and the OOM killer ended
	// the process). The Go GC also gets a soft limit 25 percent above it. 0 = off.
	HeapLimitMB int

	// Mercenary chore (docs/merc-farm.md). Hiring clicks a verified dialog; reviving clicks an UNVERIFIED one, so it is off.
	MercRetryMin         int  // minutes between merc chore attempts (a failed hire or revive is not retried sooner)
	MercHireMinGold      int  // gold in Act 2 town below which no hire is attempted
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
		LevelMargin: 0, LeaveSlack: 0, CorpseRetries: 1, PortalSettleMs: 600, ClearRadius: 30,
		CorpseValueMin: 1000, SpiralDeaths: 4, SpiralMinutes: 15, DangerBanMin: 20, StarterBanMin: 3, RepeatSiteBanMin: 30, RepeatSiteEscalate: 1, RepeatSiteWindowMin: 30, RepeatSiteGapSec: 180, CorpseMinValue: 50, NavSkipSec: 150, AreaIdleSec: 60, StallWarnMin: 30, StallExitMin: 30, PauseDeathsPerHour: 8, PovertyTier: 0, GoldFloor: 2000,
		PointsRetrySec: 30, GearCheckSec: 60, GearMinGold: 150, GearBudgetPct: 70, GearReserve: 100,
		SellMinItems: 4, DifficultyBanMin: 60, SavePointMin: 20, HeapLimitMB: 4096,
		MercRetryMin: 10, MercHireMinGold: 400, CubeMaxJobs: 6, CubeRuneCap: 10, CubeGemCap: 2,
	}
}

func botKnobs() []Knob {
	return []Knob{
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
		{Key: "bot.corpseRetries", Label: "Corpse retrievals", Unit: "count", Default: 1, Min: 0, Max: 1, HigherIsBetter: true,
			Desc: "Max attempts to recover the corpse after a softcore death. 0 abandons the gear."},
		{Key: "bot.portalSettleMs", Label: "Town settle time", Unit: "ms", Default: 600, Min: 600, Max: 4500,
			Desc: "Pause in town after chores before going back out."},
		{Key: "bot.clearRadius", Label: "Clear radius", Unit: "subtiles", Default: 30, Min: 12, Max: 30, HigherIsBetter: true,
			Desc: "Reach of one clear sweep. Smaller means more walking between sweeps."},
		{Key: "bot.corpseValueMin", Label: "Corpse value threshold", Unit: "gold", Default: 1000, Min: 1, Max: 20000,
			Desc: "Estimated value of the gear on the corpse below which it is abandoned. Higher abandons more."},
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
			Desc: "Below this estimated gold value the in-field corpse run is skipped entirely (the stricter bot.corpseValueMin still applies above it)."},
		{Key: "bot.navSkipSec", Label: "Nav-stuck area skip", Unit: "s", Default: 150, Min: 30, Max: 900,
			Desc: "How long a navigation fault (stuck, no exit to the area) keeps the planner out of an area. Short on purpose: it is a bot fault, not danger."},
		{Key: "bot.areaIdleSec", Label: "Longest wait for area bans", Unit: "s", Default: 60, Min: 15, Max: 600,
			Desc: "When every candidate area is skipped or banned, the bot waits this long in town at most, then farms the least bad area anyway (a death streak waits at least the starter ban cap)."},
		{Key: "bot.stallWarnMin", Label: "Campaign stall: warn and fresh game", Unit: "min", Default: 30, Min: 0, Max: 240,
			Desc: "Virtual minutes without campaign progress (no kill or xp, no non-town area entered, no purchase, no quest step) before the bot logs 'campaign stalled' and starts a fresh game. 0 turns the stall ladder off. Longest legitimate stretch in the navsoak logs: 19 min."},
		{Key: "bot.stallExitMin", Label: "Campaign stall: exit window", Unit: "min", Default: 30, Min: 5, Max: 240,
			Desc: "Virtual minutes still without campaign progress after the fresh game before the bot exits (status 5) so the supervisor restarts the whole process."},
		{Key: "bot.spiralMinutes", Label: "Spiral breaker window", Unit: "min", Default: 15, Min: 5, Max: 15, HigherIsBetter: true,
			Desc: "Window in which the deaths are counted. Shorter forgets sooner."},
		{Key: "bot.pauseDeathsPerHour", Label: "Pause after deaths per hour", Unit: "deaths", Default: 8, Min: 8, Max: 20,
			Desc: "Deaths in the last hour after which the bot pauses itself and asks the user."},
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
		{Key: "bot.savePointMin", Label: "Save point interval", Unit: "min", Default: 20, Min: 0, Max: 120, NoHandicap: true,
			Desc: "Minutes in one game before the bot, in town and idle, saves and re-enters it (a crash loses what came after the last save); 0 is off."},
		{Key: "bot.heapLimitMB", Label: "Heap guard limit", Unit: "MB", Default: 4096, Min: 0, Max: 8192, NoHandicap: true,
			Desc: "Go heap in use above which d2bot writes a heap profile and exits for a supervisor restart (a runaway allocation would otherwise end in the OOM killer); the GC gets a soft limit 25 percent above it. 0 is off. Normal runs use under 100 MB."},
		// Merc and cube chores: switches and policy, not skill, so no handicap profile moves them (NoHandicap).
		{Key: "bot.mercRetryMin", Label: "Merc chore retry", Unit: "min", Default: 10, Min: 1, Max: 60, NoHandicap: true,
			Desc: "Minutes between merc chore attempts; a failed hire or revive is not retried sooner."},
		{Key: "bot.mercHireMinGold", Label: "Merc hire gold", Unit: "gold", Default: 400, Min: 0, Max: 2000, NoHandicap: true,
			Desc: "Gold in Act 2 town below which no hire is attempted."},
		{Key: "bot.mercReviveUnverified", Label: "Merc revive (unverified)", Unit: "bool", Default: 0, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1 lets the town visit click the revive dialog, which is UNVERIFIED live (docs/merc-farm.md). Off until a live probe."},
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

package env

import "sync"

// ResourceTuning holds the resource-policy knobs (internal/resource, docs/resource.md). The struct lives here so the
// Tuner can read and write it by key ("resource.restStartManaPct"); the knob metadata is registered by cmd/d2bot
// through RegisterKnobs because internal/resource imports env. The potion thresholds are NOT duplicated: they come
// from CombatTuning (resource.FromEnv).
type ResourceTuning struct {
	RestStartManaPct int
	CastsPerMonster  int
	RestMaxSec       int
	EngageLifePct    int
	MinEngageLifePct int
	RetreatDanger    int
	HPReserve        int

	EconomyManaPct  int
	MeleeMinLifePct int
	MeleeMaxPack    int
	LowManaWaitMs   int
	WaitMaxSec      int
	BackoffMaxSteps int // consecutive out-of-mana back-off steps before the melee fallback; 0 = never

	PoorGold      int
	ReserveGold   int
	PotionGoldPct int
	HPFirst       int
	MPFirst       int

	TripHPMin   int
	TripMPMin   int
	TripMinGold int
	FutileStep  int
	TripHealPct int

	MercDeadPack int

	FleeLifePct   int // no potion and no portal: break away from hostiles below this life %
	KitTPGold     int // price of one town portal scroll
	KitPotionGold int // price of the cheapest healing potion
	PovertyMargin int // gold above the kit cost at which poverty mode ends
	PovertyLevels int // poverty farms the plan area for a character this many levels lower

	RetreatBelt      int // in a fight, at or below this many healing potions leave (portal or walk home) instead of drinking; 0 = off
	RetreatCountPack int // 1: spare healing/rejuvenation potions in the pack count for RetreatBelt (a belt fill will move them); 0 = belt only
	ManaPaysCasts    int // below the mana-drink line a caster still skips the potion while its mana pays this many casts of the cheapest spell; 0 = always drink at the line
	MinFieldBeltPct  int // leave town only with at least this % of the planned healing potions; 0 = off

	GearGateMode   int // 0 hard (old binary gate), 1 graded risk (docs/gear-gate.md)
	GearRiskMedium int // risk points at which gear risk is medium
	GearRiskHigh   int // risk points at which gear risk is high and the field is closed
	GearDeathsHigh int // deaths in the hour or in one area from which deaths add gear risk
	EarlyGearMax   int // below this clvl armor is ignored and only weapon-dependent classes need a weapon (docs/sme/early-gear-policy.md); 0 = off
	TripBudget     int // finished town trips allowed per TripWindowMin before a thin belt no longer sends the bot home; 0 = no budget
	TripWindowMin  int // minutes the trip budget looks back
	TripGoldFloor  int // below this gold a thin belt does not trigger a retreat (a trip could not restock) and town buys only one portal scroll; 0 = off

	RecoveryOn           int // 1: a broke character pauses its quest step and farms gold in a safe area (docs/recovery.md); 0 = old behaviour
	RecoveryKitGold      int // recovery starts when gold is below this and the missing kit costs more than the gold
	RecoveryTargetGold   int // recovery ends at this gold (after a sell trip)
	RecoveryMaxMin       int // longest recovery cycle in minutes
	RecoveryCooldownMin  int // minutes after a recovery before the next one may start
	RecoveryPotionFloor  int // healing potions a recovering character may buy (belt total); 0 = no floor, the normal belt plan
	RecoveryManaFloor    int // mana potions a recovering caster may buy (belt plus pack); 0 = none, the old behaviour
	RecoveryXPPick       int // 1: the recovery area is the safe area with the best XP per kill for the level; 0 = the plan area of a character PovertyLevels lower
	RecoveryMaxAreaAbove int // XP pick only: area monster level may be at most clvl plus this
	RecoveryKitGate      int // 1 (default): a recovering character without a healing potion in the belt or without a town portal scroll farms only Blood Moor and Cold Plains and never routes through a dungeon area; 0 = the old choice
	RecoveryXPEarly      int // XP pick only: the area may be entered this many levels below its enter and kit level (0 = the plan levels, the old behaviour)
	PovertySellMin       int // poor character: pack junk items needed for a sell trip (0 = the built-in 4 on foot, 6 with a portal scroll)
	BRAttempts           int // Blood Raven attempts before waiting for more levels; 0 = unlimited
	BRLevelGap           int // levels to gain after the last attempt before the budget is renewed
}

func DefaultResourceTuning() ResourceTuning {
	return ResourceTuning{
		RestStartManaPct: 60, CastsPerMonster: 3, RestMaxSec: 45, EngageLifePct: 65, MinEngageLifePct: 40, RetreatDanger: 2, HPReserve: 1,
		EconomyManaPct: 40, MeleeMinLifePct: 50, MeleeMaxPack: 3, LowManaWaitMs: 700, WaitMaxSec: 8, BackoffMaxSteps: 8,
		PoorGold: 150, ReserveGold: 30, PotionGoldPct: 80, HPFirst: 3, MPFirst: 2,
		TripHPMin: 1, TripMPMin: 0, TripMinGold: 60, FutileStep: 100, TripHealPct: 50,
		MercDeadPack: 2,
		FleeLifePct:  50, KitTPGold: 100, KitPotionGold: 30, PovertyMargin: 60, PovertyLevels: 4,
		RetreatBelt: 0, RetreatCountPack: 1, ManaPaysCasts: 3, MinFieldBeltPct: 0,
		GearGateMode: 1, GearRiskMedium: 3, GearRiskHigh: 6, GearDeathsHigh: 4, EarlyGearMax: 15,
		TripBudget: 3, TripWindowMin: 10, TripGoldFloor: 300,
		RecoveryOn: 1, RecoveryKitGold: 500, RecoveryTargetGold: 900, RecoveryMaxMin: 45, RecoveryCooldownMin: 15, RecoveryPotionFloor: 2, RecoveryManaFloor: 2, RecoveryXPPick: 1, RecoveryKitGate: 1, RecoveryMaxAreaAbove: 1, BRAttempts: 3, BRLevelGap: 2,
	}
}

var (
	extraMu    sync.Mutex
	extraKnobs []func() []Knob
)

// RegisterKnobs adds knobs owned by a package that imports env (so env cannot import it). Call from main or init.
func RegisterKnobs(f func() []Knob) {
	extraMu.Lock()
	extraKnobs = append(extraKnobs, f)
	extraMu.Unlock()
}

func registeredKnobs() []Knob {
	extraMu.Lock()
	defer extraMu.Unlock()
	var k []Knob
	for _, f := range extraKnobs {
		k = append(k, f()...)
	}
	return k
}

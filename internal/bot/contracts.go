package bot

import (
	"context"
	"errors"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/launch"
)

// Consumer-side contracts (docs/archive/wave2.md). The method sets are the agreed ones; the argument and result types are
// declared here so the bot builds before nav, combat, town and loot land. At integration each type becomes an alias of
// the implementer's type (type ClearOpts = combat.ClearOpts) or a thin adapter converts it. Keep them in this file.

// ErrPortalWrongLevel: BackToField came out in another field level than the wanted one (the town portal stands where
// the last scroll was read). The character is in that level; the caller walks the rest.
var ErrPortalWrongLevel = errors.New("bot: portal led to another level")

// Mover walks the character. (nav.Navigator)
type Mover interface {
	GoTo(ctx context.Context, p input.Point) error
	GoToUnit(ctx context.Context, unitID uint32) error
	ExitTo(ctx context.Context, levelNo int) error
	// Route lists the levels to enter, in order, to get from one level to another (nav owns the exit graph).
	Route(from, to int) ([]int, error)
	TakeWaypoint(ctx context.Context, levelNo int) error
	Explore(ctx context.Context, done func() bool) error
	PathTo(p input.Point) ([]input.Point, error)
}

// Fighter kills things and keeps the character alive. (combat.Fighter)
type Fighter interface {
	Clear(ctx context.Context, o ClearOpts) (ClearResult, error)
	Kill(ctx context.Context, unitID uint32) error
	Buff(ctx context.Context) error
	Safety(snap *d2.Snapshot) SafetyAction
	Act(ctx context.Context, a SafetyAction) error
}

// Townsman does town chores and trips. (town.Town)
type Townsman interface {
	ToTown(ctx context.Context) error
	Chores(ctx context.Context, plan ChorePlan) (ChoreReport, error)
	BackToField(ctx context.Context, levelNo int) error
	// Rebuy is poverty mode after an abandoned corpse: potions first, then the cheapest viable gear within budget gold.
	Rebuy(ctx context.Context, budget int) (RebuyReport, error)
	// NeedsTrip is the Townsman's pure check (potions, belt columns, scrolls, pack, durability): go to town now?
	NeedsTrip(snap *d2.Snapshot) (bool, string)
}

// RebuyReport is what a poverty rebuild bought.
type RebuyReport struct {
	Equipped []string
	Spent    int
	Missing  []string
	Blocked  []string
}

// Sentinels the behavior adapters return so the runner reacts correctly (errors.Is).
var (
	// ErrPortalOpened: the fighter opened a town portal in the middle of a fight; start an emergency trip.
	ErrPortalOpened = errors.New("bot: town portal opened")
	// ErrLeftGame: the fighter left the game to survive; the menus handle the rest.
	ErrLeftGame = errors.New("bot: left the game")
)

// Looter picks up and manages items. (loot.Looter)
type Looter interface {
	PickUp(ctx context.Context, o PickOpts) (PickReport, error)
	Evaluate(it d2.Item) Verdict
}

// ClearOpts asks the fighter to kill everything hostile within Radius subtiles of the player.
type ClearOpts struct {
	Radius int
}

// ClearResult reports one sweep.
type ClearResult struct {
	Killed    int
	Remaining int // hostile units still alive in the radius when the sweep ended
}

// SafetyAction is the fighter's pure survival decision for a snapshot.
type SafetyAction int

const (
	SafetyNone SafetyAction = iota
	SafetyDrinkLife
	SafetyDrinkMana
	SafetyDrinkRejuv
	SafetyTownPortal
	SafetyChicken
	SafetyCrowdRetreat
	SafetyCrowdAbandon
)

func (a SafetyAction) String() string {
	return [...]string{"none", "drink-life", "drink-mana", "drink-rejuv", "town-portal", "chicken", "crowd-retreat", "crowd-abandon"}[a]
}

// ChorePlan says what the town visit must do. The townsman may do more (it knows its own state).
type ChorePlan struct {
	Reason   string
	Heal     bool
	Potions  bool
	Repair   bool
	Identify bool
	Sell     bool
	Stash    bool
	Insure   bool // also stash valuables before a dangerous trip
	// Poverty: the gear is gone (abandoned corpse). Rebuy from vendors: potions first, then the cheapest viable
	// weapon and armor of tier GearTier (0 cheapest).
	Poverty   bool
	GearTier  int
	GoldFloor int // gold the visit must not spend
	// PotionFloor > 0 (recovery): buy healing potions only up to this many in the belt, no mana or rejuvenation.
	PotionFloor int
	// Antidotes > 0: top antidote potions up to this many (belt and pack); the Andariel kit (task.BossGate).
	Antidotes int
}

// ChoreReport is what the visit achieved. Done lists chore names that completed.
type ChoreReport struct {
	Done    []string
	Skipped []string
	Blocked []string // poverty rebuy: slots whose affordable stock the character cannot wear yet (strength, dex, level)
	// NeedStr, NeedDex: the requirement of the cheapest blocked item (0 = unknown); the bot passes them to a StatNeeder.
	NeedStr, NeedDex int
}

// StatNeeder is implemented by a Leveler that can spend the next stat points on gear requirements.
type StatNeeder interface {
	NeedStats(str, dex int)
}

// PickOpts selects ground items to pick up around the player.
type PickOpts struct {
	Radius int
	// Names are items taken whatever the pickit says (quest items, by display name); empty = the pickit alone.
	Names []string
}

// PickReport reports one pickup pass.
type PickReport struct {
	Picked  int
	Skipped int
}

// UnitEvaluator is an optional Looter capability: judge a pack item with its base row and the player's gold, so junk gear
// counts as sellable while the character is poor (the bare Evaluate cannot know a weapon from a potion).
type UnitEvaluator interface {
	EvaluateUnit(u d2.Unit, clvl, gold int) Verdict
}

// Verdict is the looter's decision for an item.
type Verdict int

const (
	VerdictKeep Verdict = iota
	VerdictSell
	VerdictDrop
	VerdictStash
)

// Cycler leaves and enters games through the out-of-game menus (launch.Flow). Optional: without it the bot cannot
// start a fresh game to respawn monsters, nor re-enter after a chicken exit.
type Cycler interface {
	// EnterGame gets from the menus into a game (select character, difficulty). Difficulty is "" for the current one.
	EnterGame(ctx context.Context, difficulty string) error
	// NewGame saves, leaves the current game and enters a new one.
	NewGame(ctx context.Context, difficulty string) error
}

// Corpse recovers the corpse after a softcore death. Optional: the bot has a click-the-spot default (UNVERIFIED).
type Corpse interface {
	Retrieve(ctx context.Context, area int, at input.Point) error
}

// Leveler spends stat and skill points. Optional (the leveling executor, not part of the bot core yet).
type Leveler interface {
	Spend(ctx context.Context, snap *d2.Snapshot) error
}

// Gear equips and buys weapons and armor for the build. Optional. (leveler.Leveler through an adapter)
type Gear interface {
	// PendingSwaps is the pure check: how many equip swaps from the pack would Equip make?
	PendingSwaps(snap *d2.Snapshot) int
	// Equip equips every better item in the pack, each swap verified.
	Equip(ctx context.Context) (GearReport, error)
	// Shop buys better gear from the town vendors within budget gold (it does not equip).
	Shop(ctx context.Context, budget int) (GearReport, error)
}

// GearReport is what a gear pass did.
type GearReport struct {
	Swaps  []string
	Bought []string
	Spent  int
}

// Quester claims quest rewards in town (Den of Evil: talk to Akara). Optional; without it a finished quest area is
// logged and the reward stays unclaimed. UNVERIFIED end to end.
type Quester interface {
	Claim(ctx context.Context, quest string) error
}

// Health reports game process health; *launch.Monitor satisfies it.
type Health interface {
	Step(now time.Time) launch.Event
}

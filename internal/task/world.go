package task

import "context"

// Area is a d2 level id. Ids are from docs/goals/execution.md and catalog.md (U: verify against levels.txt at setup).
type Area int

const (
	RogueEncampment Area = 1
	BloodMoor       Area = 2
	ColdPlains      Area = 3
	StonyField      Area = 4
	DarkWood        Area = 5
	BlackMarsh      Area = 6
	DenOfEvil       Area = 8
	BurialGrounds   Area = 17
	ForgottenTower  Area = 20
	TowerCellar5    Area = 25
	Barracks        Area = 28
	Catacombs4      Area = 37
	Tristram        Area = 38
	LutGholein      Area = 40
)

// Quest ids follow docs/archive/goals-catalog.md ("quest.<id>").
const (
	QDen    = "den_of_evil"
	QBurial = "sisters_burial"
	QCain   = "search_for_cain"
	// QCainBoard is the board's own group for the report to Akara (slice S7b); the campaign never lists it.
	QCainBoard = "search_for_cain_board"
	QTower     = "forgotten_tower"
	QTools     = "tools_of_the_trade"
	QAndariel  = "sisters_to_slaughter"
	QAct1Done  = "act1_complete"
	NPCAkara   = "Akara"
	NPCKashya  = "Kashya"
	NPCCharsi  = "Charsi"
	NPCWarriv  = "Warriv"
	NPCCain    = "Deckard Cain"
	BossBR     = "Blood Raven"
	BossSmith  = "The Smith"
	BossCount  = "The Countess"
	BossAndy   = "Andariel"
	ItemMalus  = "Horadric Malus"
	ItemBark   = "Scroll of Inifuss"
	ItemCainSc = "Key to the Cairn Stones" // code bkd, what Akara hands back after reading the scroll (not Cain's own scroll)
	ItemPortal = "Tristram portal"         // pseudo item: the red portal is open
)

// Stage is how far a quest got. Cleared means the objective is met but the reward is not claimed (Den cleared,
// Blood Raven dead): the turn-in step runs on Cleared and not on Done.
type Stage int

const (
	StageNone Stage = iota
	StageStarted
	StageCleared
	StageDone
)

// Evidence says how sure a decision is (tasklayer.md section 2). A Need that rests on EvProxy never proves a reward
// was claimed.
type Evidence int

const (
	EvNone Evidence = iota
	EvProxy
	EvMemory
	EvSave
)

// QuestState is one quest as the world reader sees it.
type QuestState struct {
	Stage Stage
	Ev    Evidence
	// Raw is the quest record's word as read (0 = unknown or untouched). The Stage folds it; a handler that needs one
	// bit of it (Search for Cain: bit 4 = the cairn stones were done, so the tree has given its scroll) reads it here.
	Raw uint16
}

// World is the read-only value one tick decides on (finding F7: the same value goes to Need and Ready of every
// candidate). The bot's Env assembles it from d2.Snapshot, the quest/waypoint readers and the pack.
// KitPlan is the supply plan's answer for the guard kit (internal/supply, town-planner section 5.4): the minima the
// guards compare against, already clamped to what the hero can carry, and the line the plan cannot close.
type KitPlan struct {
	Healing, Mana, Portals int
	GoldCarry              int    // the purse the excursion wants when the stash has it (profile gold_carry; the boss profiles)
	Short                  string // "" = the plan closes every min with the gold owned; else "short of gold: needs N more for healing"
	Hand                   string // the hand-over of a short-gold plan ("needs N more gold for X: sell, then recovery"); "" when the plan is not short of gold
}

// EarlyBlock is why an area of the early road is closed now.
type EarlyBlock struct {
	Why     string
	KitOnly bool
}

type World struct {
	Act         int
	Difficulty  string
	Area        Area
	InTown      bool
	Dead        bool
	Level       int
	Gold        int
	Potions     int // usable healing potions in belt and pack
	HasTP       bool
	KitPlan     *KitPlan              // the town planner's view of the guard kit (knob town.plan >= 2); nil = the constants below
	ManaPotions int                   // mana potions in belt and pack
	Caster      bool                  // the class lives on mana (Sorceress, Necromancer, Druid); the boss gate checks mana for it
	WPs         map[Area]bool         // waypoints activated
	Quests      map[string]QuestState // empty when no quest reader exists (proxies only)
	// SaveOnly: Quests come from the .d2s alone, which only changes on Save and Exit. A step's effect cannot be confirmed
	// in the same session, so a clean Do whose Need stays true is assumed (as without a reader), not counted a failure.
	SaveOnly bool
	Killed   map[string]bool // uniques killed in this game (event proxy, EvProxy)
	Items    map[string]int  // quest items in the pack by name
	Cleared  map[Area]bool   // areas fully cleared this game
	// Swept: areas whose sweep ended (no monster left in reach) while the live quest word still says the objective is
	// open. They are not Cleared (the word is the truth), but the claim probe uses them: one Talk with the giver costs a
	// minute and shows whether the game just has not reported the clear (Akara's Talk pays a pending reward directly).
	Swept map[Area]bool

	// Capabilities the bot knows about its character (not read from the game by the task layer).
	CanTeleport bool // a Teleport key is bound and usable outside town (Summoner, Council moat, Durance shortcuts)
	HasMerc     bool // a living mercenary is in the unit table (replaces the pseudo item "Mercenary")
	// Gated: the gear gate (resource.GearGate) keeps the character out of the field: no weapon, no body armor, or no
	// potions and no gold. Only towns and SafeArea (the first field, where it farms gold) are open to it; GatedWhy says why.
	Gated    bool
	GatedWhy string
	// GatedFix is "potions" when the only thing that gates the character is a missing potion kit that a town visit with its
	// gold fixes (knob quests.boardSM): the guard then sends it to town instead of refusing the step.
	GatedFix string
	// BoardSM mirrors the knob quests.boardSM: guards then return coded faults (Fault.Code) and holds (Fault.Hold).
	BoardSM  bool
	SafeArea Area
	// Banned: areas the death-spiral breaker keeps the character out of for now (travel, clear and kill steps wait).
	Banned map[Area]bool
	// BanNote is the player-facing detail of a ban ("N deaths, until HH:MM"), empty when the bot has none.
	BanNote map[Area]string

	// Blood Raven kit gate inputs (gate.go brFloor/brShort). BRHigh: the class takes the higher caster floor (Sorceress,
	// Druid, Amazon; the Necromancer and melee classes do not). BR holds the knobs, zero fields mean the defaults;
	// BRWaitLevel is the level a failed attempt waits for (0 = none). TPKnown false: the bot cannot read portal charges and
	// HasTP alone counts (U: no quantity reader).
	BRHigh      bool
	BR          BRKnobs
	BRWaitLevel int
	TPCharges   int
	TPKnown     bool

	// AndarielMinLevel is the Andariel clvl floor (Tuning.Boss.AndarielMinLevel); 0 = env.DefaultAndarielMinLevel.
	AndarielMinLevel int
	// AndarielMinLevelMerc is the floor with a living merc (Tuning.Boss.AndarielMinLevelMerc); 0 = the no-merc floor.
	AndarielMinLevelMerc int
	// AndarielRec and AndarielRecMerc are the class's recommended levels (plandata.BossLevels); below them the gate still
	// passes but AndarielWarn says so. 0 = no recommendation (no warning).
	AndarielRec, AndarielRecMerc int
	// AndarielHold: no unattended Andariel attempt before the throwaway fight has won once (Tuning.Boss.AndarielHold); false = off.
	AndarielHold bool
	Antidotes    int // antidote potions in belt and pack
	// MercKnown and MercAlive come from the optional MercAlive hook (gate.go); MercKnown false = no reader, the gate passes.
	MercKnown, MercAlive bool

	// Route guard and imbue knobs (route.go); zero = rule off. RouteMinLevel: clvl that opens the Monastery road without a
	// merc. ImbueMinLevel: clvl floor of the optional Imbue quest; ImbueUseful: the class is melee; ImbueNeedsMerc: also
	// needs a living merc.
	RouteMinLevel, ImbueMinLevel int
	// RouteMinLevelMerc is the road floor with a living merc; 0 = a living merc opens the road at any level.
	RouteMinLevelMerc           int
	ImbueUseful, ImbueNeedsMerc bool
	// RouteBlocked is RouteWhy(w) filled by the bot ("" = open): fieldReady keeps every step into a guarded area waiting.
	RouteBlocked string
	// EarlyBlocked is the early road guard by area (earlyroutes.json, filled by the bot): a step into one of these areas waits with the
	// reason; KitOnly means a chores trip fixes it (the others are level or life, the hero levels first).
	EarlyBlocked map[Area]EarlyBlock
	// WalkBlocked is the kit rule of every quest walk into a field (earlyroutes.json questWalk, filled by the bot).
	WalkBlocked EarlyBlock
	// FirstField is the first field of the hero's act (filled with WalkBlocked): the quest-walk kit rule never closes it, because a hero
	// in town with an empty purse earns nothing anywhere else (Questy 2026-11-26).
	FirstField Area
	// TowerMinLevel (0 = tower guard off) and TowerNeedsMerc are the Forgotten Tower knobs (route.go); TowerBlocked is
	// TowerWhy(w) filled by the bot, kept out of areas 20-25 by fieldReady.
	TowerMinLevel     int
	TowerMinLevelMerc int  // the Forgotten Tower floor with a living merc while TowerNeedsMerc is false; 0 = the no-merc floor
	TowerNeedsMerc    bool // true: a living merc opens the tower at any level
	TowerBlocked      string
	// MaxLife is the character's max life (0 = not readable). TowerMinLife (0 = no life rule) is the no-merc max-life
	// floor of the tower; TowerMinLevelNoLife the clvl floor used instead while MaxLife is unreadable; TowerKitMana the
	// mana potions the tower kit wants (0 = the road's count for a Caster); the bot sets it for a mana-column class only.
	MaxLife, TowerMinLife, TowerMinLevelNoLife, TowerKitMana int
	// BarracksMinLevel and BarracksMinLife are the class's floors for the Tools of the Trade chunk (route.barracks*, the bot
	// fills the melee or the caster pair); 0 = no rule.
	BarracksMinLevel, BarracksMinLife int
	BarracksLifeWaiver                int // levels above BarracksMinLevel after which the life floor is waived (0 = never)
	// DenMinLevel is the Normal clvl floor of Den of Evil for this class (caster or melee knob, bot fills it); 0 = no class floor.
	DenMinLevel int

	NoTownPortal bool // the running step carries Constraints.NoTownPortal (Ancients): no step may plan a town portal
}

// Env reads the world. One call per tick.
type Env interface {
	World(ctx context.Context) (World, error)
}

// Actions is every capability a step may use (tasklayer.md "Deps"); the bot adapts nav, combat, town and loot to it.
// Each call is bounded and idempotent from the caller's view: it may return after partial progress and the next tick
// re-derives. Errors should be wrapped with Wrap so Classify can tell the fault kind.
type Actions interface {
	// Travel gets to the area (nav.ToArea: waypoint when active, else Route+ExitTo; town portal when going home).
	Travel(ctx context.Context, a Area) error
	// ToTown opens a town portal or walks home (town.ToTown).
	ToTown(ctx context.Context) error
	// Clear explores and fights until the whole area is cleared or the bounded sweep ends (nav.Explore + combat.Clear).
	Clear(ctx context.Context, a Area) error
	// KillBoss finds the named unique in the current area, kills it, picks up its quest drops (combat.Kill + loot).
	KillBoss(ctx context.Context, name string) error
	// Talk walks to the NPC, opens dialog and picks the named option ("" = just open and close).
	Talk(ctx context.Context, npc, pick string) error
	// Interact clicks a named object (quest tree, cairn stone, cage, waypoint).
	Interact(ctx context.Context, object string) error
	// PickUp collects a named ground item in range.
	PickUp(ctx context.Context, item string) error
	// ActivateWaypoint walks to and touches the area's waypoint (nav.ActivateWaypoint).
	ActivateWaypoint(ctx context.Context, a Area) error
	// Chores runs the town loop: potions, scrolls, repair, sell (town.Chores).
	Chores(ctx context.Context) error
	// LevelTo runs one bounded slice of the bot's existing reach_level loop toward the level.
	LevelTo(ctx context.Context, level int) error
}

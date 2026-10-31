package boss

import (
	"context"
	"strings"
	"time"
)

// Match identifies the boss in the monster list. Any matching field is enough: Class for plain uniques (Andariel
// 156, Blood Raven 267), UniqueNo or Name for super uniques (Countess, Smith: the base class is shared with
// ordinary monsters, so prefer UniqueNo/Name there). Zero Class and negative UniqueNo are "unset".
type Match struct {
	Class    int
	UniqueNo int // -1 = unset
	Name     string
}

// Is reports whether m is the boss.
func (x Match) Is(m Mon) bool {
	if x.Class > 0 && m.Class == x.Class && (m.Name != "" || x.Name == "") {
		return true
	}
	if x.UniqueNo >= 0 && m.UniqueNo == x.UniqueNo && m.UniqueNo >= 0 {
		return true
	}
	return x.Name != "" && strings.EqualFold(m.Name, x.Name)
}

// Anchor says where the fight happens. Preferred source is a preset (works for random layouts); Hint is a fixed
// world coordinate from Kolbot used only as a cross-check and as a last-resort walk target (S, U for 1.14d).
type Anchor struct {
	Preset   *PresetRef
	Also     []*PresetRef // tried in order when Preset is absent from the level
	Hint     *Pt
	Boss     bool // the boss unit itself is the anchor: no preset, wait/explore until it is in the monster list
	HintSlop int  // subtiles the resolved stand may differ from Hint before a note is recorded (default 60)
	// Teleport: the arena is islands over a void (Arcane Sanctuary), so walking cannot reach the anchor; when the
	// Actions value also implements Teleporter, hop there first and let GoTo cover the last stretch.
	Teleport bool
}

// Teleporter is the optional Actions extension for Teleport characters (nav.Navigator.TeleportTo).
type Teleporter interface {
	TeleportTo(ctx context.Context, p Pt) error
}

// PresetRef selects a preset and the stand point derived from it.
type PresetRef struct {
	Kind, ID int
	Off      Pt        // stand = preset.Pos + Off when no variant matches
	Variants []Variant // layouts told apart by the preset's X (Countess chest: 12565 / 12526)
}

// Variant maps one observed preset X to a stand point (absolute, from Kolbot, S).
type Variant struct {
	AtX   int
	Stand Pt
}

// StandPoint is a position used while fighting. Rel points are offsets from the boss spawn position (the first
// place the boss was seen), so they do not depend on the level's world origin; absolute ones are Kolbot values.
type StandPoint struct {
	Name          string
	Pos           Pt
	Rel           bool
	NeedsTeleport bool // docs/progression/bosses.md section 4: skipped when the character cannot teleport
}

// OuterClear is the pre-fight sweep around the arrival point.
type OuterClear struct {
	Radius int
	Budget time.Duration
}

// Lure pulls the boss towards the position the fight started from before attacking.
type Lure struct {
	Dist int           // wait until the boss is within this many subtiles
	Wait time.Duration // give up luring after this long (stationary bosses never come)
}

// Kite sidesteps between standing points while the boss is close.
type Kite struct {
	Range int           // act when the boss is closer than this
	Every time.Duration // minimum time between steps
	Away  int           // fallback: step this far straight away from the boss when no point fits
}

// RespKind is what a hazard makes the runner do.
type RespKind int

const (
	RespNone RespKind = iota
	RespAntidote
	RespStepAway
	RespClearAdds
	RespRetreat // walk back to where the fight started and drink
	RespDrink
	RespEscape // town portal
	RespAbort  // ErrAborted
)

// Response is one hazard reaction with its parameters.
type Response struct {
	Kind   RespKind
	Radius int           // StepAway: distance; ClearAdds: radius
	Budget time.Duration // ClearAdds: time cap
}

// Ctx is what a hazard sees.
type Ctx struct {
	V      View
	Boss   *Mon // nil before the boss was located
	St     *State
	Script *Script
}

// Hazard is a detector plus a response (docs/progression/bosses.md section 3). Evaluated each loop in list order,
// first one that fires and is off cooldown wins.
type Hazard struct {
	Name     string
	Detect   func(c Ctx) bool
	Do       Response
	Cooldown time.Duration
}

// ExitPlan is what happens after the kill and the loot.
type ExitPlan int

const (
	ExitStay ExitPlan = iota // the task layer decides (next step walks or portals)
	ExitTown                 // ToTown
)

// Hooks are optional callbacks for the integration (events, buffs, logging).
type Hooks struct {
	OnPhase      func(name string)
	BeforeAttack func(ctx context.Context, v View, b Mon) error // buffs, Static Field openers
	OnKill       func(b Mon)
}

// Script is one boss as data plus the shared runner (Run).
type Script struct {
	Name     string // task.BossXxx display name
	Act      int
	WP       int   // waypoint area the run starts from (0 = none)
	Route    []int // areas after the WP, crossed with ExitTo in order
	Arena    int   // final area
	Match    Match
	Anchor   Anchor
	Stand    []StandPoint
	Lure     *Lure
	Kite     *Kite
	Hazards  []Hazard
	MinLevel [3]int // clvl floor per difficulty; below it the run refuses (ErrTooWeak)
	Timeout  time.Duration
	// ExploreBudget overrides Opts.Explore for this script (the Barracks are big: V live, 90 s was not enough for the Smith).
	ExploreBudget time.Duration

	EscapePct        int  // life below this with no potions: town portal
	ChickenPct       int  // life below this: leave the game
	DrinkPct         int  // life below this: drink a healing potion (the script's own line, on top of combat's)
	PreAntidoteBelow int  // drink an antidote before the first attack when poison resist is below this and one exists
	NoTP             bool // Ancients style: never portal out
	MinPotions       int  // approach refuses (ErrAborted) when the character carries fewer healing potions
	RouteMinLifePct  int  // after each route hop, life below this escapes (deep levels without a waypoint)
	// OuterClear kills the monsters around the arrival point before the boss is looked for (Blood Raven's outer skeletons
	// and zombies, SME section 3). Zero Radius = off.
	OuterClear OuterClear
	// RetreatPct: life below this with a potion left walks back to the arrival point (the entry, the nearest chokepoint
	// the script knows) and drinks. The MaxRetreats-th retreat of one run is a town portal instead, which ends the attempt.
	// 0 = off.
	RetreatPct, MaxRetreats int

	PostKillWait time.Duration // let minions show up before looting (prior-art 02: 2 s)
	PostClear    int           // radius cleared around the corpse before looting, 0 = none
	Loot         LootSpec      // Center is filled in with the boss position
	Exit         ExitPlan
}

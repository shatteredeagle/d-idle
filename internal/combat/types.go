// Package combat kills things and keeps the character alive: target selection, skill choice, hotkey binding,
// potions and chicken, positioning. The entry point is Fighter (docs/archive/wave2.md, docs/combat.md).
package combat

import (
	"context"
	"errors"
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// Mover walks the character. Consumer-side subset of the wave 2 Mover contract (nav.Navigator provides it).
type Mover interface {
	GoTo(ctx context.Context, p input.Point) error
	GoToUnit(ctx context.Context, unitID uint32) error
	// PathTo is pure planning on the current grid; an error means the point is not reachable (or not walkable).
	PathTo(p input.Point) ([]input.Point, error)
}

// Syncer is an optional extension of Mover: bring the map up to the player's current position without walking.
// Clear and Kill call it first, so the first reachability check after a position jump (debug goto, portal or waypoint
// arrival, restart) does not run on the previous level's grid and call every monster unreachable (DRILLS5 F4).
type Syncer interface {
	Sync() error
}

// SafetyAction is the survival decision for one snapshot.
type SafetyAction int

const (
	SafetyNone SafetyAction = iota
	DrinkLife
	DrinkMana
	DrinkRejuv
	TownPortal
	Chicken
	CrowdRetreat // a crowd and a thinning life: step away from it (crowd.go)
	CrowdAbandon // a big crowd on a walk: retreat 20+ subtiles, then open a portal
)

func (a SafetyAction) String() string {
	return [...]string{"none", "drink-life", "drink-mana", "drink-rejuv", "town-portal", "chicken", "crowd-retreat", "crowd-abandon"}[a]
}

// Errors returned by Clear / Kill / Act. Use errors.Is.
var (
	ErrDead         = errors.New("combat: character is dead")
	ErrNotInGame    = errors.New("combat: not in game")
	ErrTownPortal   = errors.New("combat: opened a town portal, go to town")
	ErrChicken      = errors.New("combat: left the game to survive")
	ErrTargetGone   = errors.New("combat: target not found")
	ErrNoPotion     = errors.New("combat: no potion of that kind in the belt")
	ErrNoPortal     = errors.New("combat: no town portal scroll or tome")
	ErrNoHotkey     = errors.New("combat: skill has no hotkey")
	ErrBindFailed   = errors.New("combat: could not bind the skill to its hotkey")
	ErrTargetGaveUp = errors.New("combat: gave up on the target")
)

// ClearOpts says what Clear should do.
type ClearOpts struct {
	Radius  int           // subtiles around Origin; 0 = tuning default
	Origin  *input.Point  // nil = the player's position when Clear starts
	Timeout time.Duration // 0 = tuning default
	// Yield marks a call from a walk stalled by monsters: earlier ignore decisions are not applied, the nearest hostile is fought.
	Yield bool
	// Interlude, when set, is called once per fight tick after survival and before a target is chosen, with the
	// distance of the nearest hostile and the pack danger. true means it acted (the snapshot is stale). The bot uses
	// it for the gold grab (docs/goldprio.md); it must decide quickly and never leave the fight stuck.
	Interlude func(ctx context.Context, snap *d2.Snapshot, nearest float64, danger int) bool
}

// ClearResult reports what a Clear (or Kill) did.
type ClearResult struct {
	Kills   int
	Bosses  int
	XP      int64
	Skipped int // targets left alive: immune to every skill, no progress, unreachable
	Avoided int // targets deliberately not fought (danger table)
	// Ignored lists every hostile in scope that was left alive on purpose, with the reason (unreachable, no shot, avoid,
	// immune, ...), including units remembered from earlier Clear calls. The caller moves on instead of calling Clear again.
	Ignored  []Ignored
	Potions  int
	TimedOut bool
	Elapsed  time.Duration
}

func (r ClearResult) String() string {
	return fmt.Sprintf("kills=%d bosses=%d xp=%d skipped=%d avoided=%d ignored=%d potions=%d timedOut=%v in %v",
		r.Kills, r.Bosses, r.XP, r.Skipped, r.Avoided, len(r.Ignored), r.Potions, r.TimedOut, r.Elapsed.Round(time.Millisecond))
}

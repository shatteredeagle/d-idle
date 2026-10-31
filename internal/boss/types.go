// Package boss runs scripted boss fights: reach the arena, find the boss, fight it in phases (lure, kite, attack,
// retreat) under per-boss hazard rules, then pick up the loot. It depends on two small interfaces (World and
// Actions) that INTEGRATE adapts from nav, combat, loot and town, so every behaviour is testable on a fake.
// Tags: V verified, S secondary source (Kolbot), U unverified. All live behaviour here is U until docs/boss.md's
// checks have run. Coordinates are subtiles (5 per tile), like the rest of the bot.
package boss

import (
	"context"
	"errors"
	"time"
)

// Pt is a position in subtiles.
type Pt struct{ X, Y int }

// Dist is the straight-line distance in subtiles.
func (p Pt) Dist(q Pt) int {
	dx, dy := float64(p.X-q.X), float64(p.Y-q.Y)
	return int(isqrt(dx*dx + dy*dy))
}

func isqrt(v float64) float64 {
	if v <= 0 {
		return 0
	}
	x := v
	for i := 0; i < 40; i++ {
		x = (x + v/x) / 2
	}
	return x
}

// Mon is one monster as the script needs it (adapter fills it from d2.Snapshot.Monsters).
type Mon struct {
	ID       uint32
	Class    int    // monstats hcIdx
	UniqueNo int    // superuniques hcIdx for super uniques, else -1 (U: MonsterInfo.UniqueNo)
	Name     string // display name of uniques/bosses, "" otherwise
	Pos      Pt
	Mode     int // d2 monster mode; 0 dying, 12 dead are not alive
	LifePct  int // 0..100
	MaxLife  int // max life in points (stat 7), 0 = unread; Blood Raven's real Normal HP is open (SME: 113 vs 323)
	Minion   bool
	Hostile  bool // from combat.Hostiles: not friendly by combat.IsFriendly (NPC, pets, merc, minions)
}

// Alive is false for dying (0) and dead (12) monsters, like d2.Unit.Alive.
func (m Mon) Alive() bool { return m.Mode != 0 && m.Mode != 12 }

// Danger is a ground hazard the adapter derived from missiles/objects (poison cloud, fire wall ...). Kind is a
// stable lower-case key the hazards match on. U: how each Kind is read from memory is not known yet.
type Danger struct {
	Kind string
	Pos  Pt
}

// Preset is a statically placed unit (d2map.Preset): Kind 1 NPC/monster class, 2 object, 3 tile (U for 1.14d).
type Preset struct {
	Kind int
	ID   int
	Pos  Pt
}

// Preset kinds, copied from d2map so this package stays free of it.
const (
	PresetNPC    = 1
	PresetObject = 2
	PresetTile   = 3
)

// View is one read of the game (adapter: d2.Snapshot + fighter state). One call per decision.
type View struct {
	Area       int
	Difficulty int // 0 Normal 1 Nightmare 2 Hell
	Level      int
	Pos        Pt
	Life, Max  int
	Mana       int
	MaxMana    int
	Dead       bool
	Poisoned   bool // U: poison state is not read yet; adapter may derive from the life curve or a state list
	PoisonRes  int  // character poison resist in percent (stat 45)
	Antidotes  int  // antidote potions in belt/pack
	Potions    int  // healing + rejuvenation potions
	BeltSlots  int  // potion slots of the equipped belt, 0 = unknown
	HasTP      bool
	Monsters   []Mon
	Dangers    []Danger
}

// PotionFloor is the number of healing potions the character must carry for a script that asks for want: a small belt
// (4 slots, no belt item) also holds the mana potions the town chores keep, so it only has to be half full.
func (v View) PotionFloor(want int) int {
	switch {
	case v.BeltSlots <= 0:
		return want
	case v.BeltSlots <= 4:
		return min(want, max(1, v.BeltSlots/2))
	}
	return min(want, v.BeltSlots)
}

// LifePct is life in percent, 100 when Max is unknown.
func (v View) LifePct() int {
	if v.Max <= 0 {
		return 100
	}
	return v.Life * 100 / v.Max
}

// Hostiles are the living hostile monsters, bosses included.
func (v View) Hostiles() []Mon {
	var out []Mon
	for _, m := range v.Monsters {
		if m.Hostile && m.Alive() {
			out = append(out, m)
		}
	}
	return out
}

// World reads the game. Sleep must advance the fake's clock in tests and honour ctx in the bot.
type World interface {
	View(ctx context.Context) (View, error)
	// Presets lists the static units of the current area (nav/d2map). Nil when unknown.
	Presets(ctx context.Context) ([]Preset, error)
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

// LootSpec says what to collect after the kill. Names are item display names; Kinds are classes of items.
type LootSpec struct {
	Center Pt
	Radius int
	Quest  []string // always pick these up (quest items)
	Kinds  []string // "rune", "key", "gem", "unique", "set", "rare", "any" (loot module decides the tiers)
}

// Actions is what a script may do. The task-layer adapter (docs/boss.md section 5) maps each to nav, combat, loot, town.
type Actions interface {
	// Travel goes to the area with the nav planner (waypoint when active), also from town. nav.ToArea.
	Travel(ctx context.Context, area int) error
	// ExitTo walks to the exit into the neighbouring area and crosses. nav.ExitTo.
	ExitTo(ctx context.Context, area int) error
	// GoTo walks to a point (nav GoTo, tolerant of a few subtiles).
	GoTo(ctx context.Context, p Pt) error
	// Explore reveals the current area for up to budget, stopping early when stop() is true (nav.Explore).
	Explore(ctx context.Context, budget time.Duration, stop func() bool) error
	// Attack fights the unit for at most slice, then returns nil (a deadline of its own is not an error). It must
	// return combat.ErrTownPortal/ErrChicken/ErrDead wrapped as ErrEscaped/ErrChicken/ErrDead of this package.
	Attack(ctx context.Context, id uint32, slice time.Duration) error
	// ClearAround fights everything hostile within radius of p for at most budget (combat.Clear).
	ClearAround(ctx context.Context, p Pt, radius int, budget time.Duration) error
	// Antidote drinks one antidote potion.
	Antidote(ctx context.Context) error
	// DrinkHealing drinks the best healing potion available (combat Act(DrinkLife/Rejuv)).
	DrinkHealing(ctx context.Context) error
	// Loot walks over the spec and picks up what matches; returns the number of items taken.
	Loot(ctx context.Context, spec LootSpec) (int, error)
	// ToTown opens a town portal (or walks) and arrives in town. town.ToTown.
	ToTown(ctx context.Context) error
	// Chicken leaves the game (combat Fighter.Leave).
	Chicken(ctx context.Context) error
}

// Errors of a boss run. Map to task Kinds as docs/boss.md section 5 says.
var (
	ErrBossNotFound = errors.New("boss: boss not found in the arena")
	ErrTimeout      = errors.New("boss: run timed out")
	ErrEscaped      = errors.New("boss: escaped to town")
	ErrChicken      = errors.New("boss: left the game")
	ErrDead         = errors.New("boss: character died")
	ErrLeftArena    = errors.New("boss: left the route or arena")
	ErrTooWeak      = errors.New("boss: character level below the script's floor")
	ErrAborted      = errors.New("boss: aborted by a hazard rule")
	ErrBossLost     = errors.New("boss: boss vanished alive")
)

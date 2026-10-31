// Package merc is the pure mercenary policy of the bot: what to hire, when to replace, revive, equip and strip.
// It knows no game, no memory layout and no UI: a backend (internal/mem today, possibly d2d later,
// docs/decisions.md 2026-10-30) fills MercState and HireOffer through Reader and performs decisions through Actor.
// Rules and their evidence tags: docs/merc.md.
package merc

import "context"

// Hireling unit classes (monstats). V: internal/d2/merc.go MercClasses, docs/merc-farm.md 1.1.
const (
	ClassRogue    = 271 // Act 1 Rogue Scout
	ClassDesert   = 338 // Act 2 Desert Mercenary
	ClassIronWolf = 359 // Act 3
	ClassBarb     = 561 // Act 5
)

// Difficulty is the game difficulty.
type Difficulty int

const (
	Normal Difficulty = iota
	Nightmare
	Hell
)

func (d Difficulty) String() string {
	return [...]string{"normal", "nightmare", "hell"}[max(0, min(int(d), 2))]
}

// Slot is a merc equipment slot. V (d2decomped): head, torso, left hand (shield), right hand (weapon).
type Slot int

const (
	SlotHelm Slot = iota
	SlotArmor
	SlotShield
	SlotWeapon
	NumSlots
)

func (s Slot) String() string {
	return [...]string{"helm", "armor", "shield", "weapon"}[max(0, min(int(s), int(NumSlots)-1))]
}

// Kind is the item category the equip rules need.
type Kind int

const (
	KindOther Kind = iota
	KindHelm
	KindArmor
	KindBow
	KindSpear // spear or polearm
	KindSword1H
	KindAxe1H
	KindShield
)

// Item is one worn or candidate item. Zero stats mean absent; the backend fills what it can read.
type Item struct {
	Name, Base string
	Kind       Kind
	Sockets    int
	Ethereal   bool
	Identified bool
	Broken     bool
	Defense    int
	DmgMin     int
	DmgMax     int
	Life       int // +life from the item
	Res        int // sum of the four +resist values
	ReqStr     int
	ReqDex     int
	ReqLevel   int
	W, H       int // inventory footprint in cells
}

// Cells is the inventory footprint, at least 1.
func (it Item) Cells() int { return max(1, it.W) * max(1, it.H) }

// MercState is what the backend reads about the mercenary. Known false means no reader answered (everything else
// is then meaningless and every policy fails closed).
type MercState struct {
	Known bool
	// Owned: the player employs a merc, alive or dead (a dead one still forfeits the free Rogue, d2decomped, S).
	Owned bool
	Class int
	Alive bool
	// LifePct 0..100.
	LifePct int
	Level   int
	Aura    string // hire window text; memory cannot read it, so the caller keeps what it hired
	// Str and Dex are the merc's own stats (0 with StatsKnown false: not read).
	Str, Dex   int
	StatsKnown bool
	Gear       [NumSlots]*Item
	// FreeCells is the free inventory space of the player (for the strip plan), -1 unknown.
	FreeCells int
}

// HireOffer is one row of a hire window.
type HireOffer struct {
	Row   int
	Class int
	Aura  string // aura name; for the Rogue the arrow skill (Cold Arrow, Fire Arrow)
	Level int
	Price int
}

// Reader is what a backend must provide to read. Fakes: internal/merc/sim.
type Reader interface {
	State() (MercState, error)
	// Offers lists the rows of the open hire window; an error when no window is open.
	Offers() ([]HireOffer, error)
}

// Actor is what a backend must provide to act. Every call is one UI-level step the backend verifies itself.
type Actor interface {
	// Hire clicks the offer row and returns the price actually paid (gold delta).
	Hire(ctx context.Context, row int) (paid int, err error)
	// Revive pays for the dead merc at the current NPC and returns the price paid.
	Revive(ctx context.Context) (paid int, err error)
	Equip(ctx context.Context, slot Slot, it Item) error
	// Unequip takes the slot's item off into the player's inventory.
	Unequip(ctx context.Context, slot Slot) error
	// GiveHealing hands one healing potion to the merc (Shift + belt key).
	GiveHealing(ctx context.Context) error
}

// Backend is Reader plus Actor.
type Backend interface {
	Reader
	Actor
}

// Decision is a policy verdict with the reason that goes into the log.
type Decision struct {
	Do     bool
	Reason string
	// StripFirst: take the current merc's gear off before the hire (replace only).
	StripFirst bool
	// LeaveArea: the merc died in the field; stop fighting here (revive decisions only).
	LeaveArea bool
}

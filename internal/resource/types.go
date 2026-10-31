// Package resource is the pure mana/health policy: when to rest, what to cast or hit with, when to drink,
// when a town trip pays off and when a fight must be abandoned. Nothing here touches the game or the clock; callers fill
// Input from a snapshot and act on the returned decision. Design notes and integration requests: docs/resource.md.
//
// Tags: V verified, S secondary source, U unverified (see docs/resource.md for the list).
package resource

import "d2lod-bot/internal/d2"

// Skill is one usable attack, in the caller's preference order.
type Skill struct {
	Name  string
	Cost  int  // mana per use, rounded up; 0 for free and melee
	Melee bool // basic attack with the wielded weapon (always available, costs nothing)
	Ready bool // not on cooldown, key bound
}

// Belt counts potions by kind; tiers are 1 minor, 2 light, 3 normal, 4 greater, 5 super (rejuv: 1 rejuv, 2 full).
type Belt struct {
	HP, MP, RV             int
	HPTier, MPTier, RVTier int // best tier held (0 = none)
	Rows                   int // rows of the worn belt (1 plain, up to 4); 0 means 1
	PackHP, PackRV         int // healing and rejuvenation potions in the backpack (BELTFILL moves them to the belt)
	Slots                  int // potion slots of the worn belt (4 per row); 0 = 4 * Rows
}

// FreeSlots is the number of empty belt slots.
func (b Belt) FreeSlots() int {
	n := b.Slots
	if n <= 0 {
		n = 4 * max(b.Rows, 1)
	}
	return max(n-b.HP-b.MP-b.RV, 0)
}

// Spare is the number of healing and rejuvenation potions in the pack that the belt has room for: the ones a belt
// fill would make usable.
func (b Belt) Spare() int { return min(b.PackHP+b.PackRV, b.FreeSlots()) }

// Input is everything the policy reads. Zero values are safe: unknown merc, no hostiles, no gold.
type Input struct {
	Build         string // leveler table name ("" unknown); selects the mana-heavy belt, see manaheavy.go
	Life, MaxLife int
	Mana, MaxMana int
	Belt          Belt
	HasTP         bool // town portal scroll or tome in the pack
	NearTown      bool // the town is a short walk away (an early area of act 1)
	InTown        bool
	Gold          int
	LastTripGold  int // gold when the last town trip ended; -1 when none yet
	Level         int
	Sellable      int // sellable pack items (junk worth gold), 0 when unknown
	TripsRecent   int // finished town trips inside the trip-budget window (Tuning.TripWindowMin)
	IdleTrips     int // consecutive town trips that bought nothing (no gold); reset by a sale or a purchase
	Class         d2.Class
	Diff          d2.Difficulty
	Area          int
	Poor          bool // poverty mode: no gold to refill potions
	MaxPackDanger int  // with Poor and no healing potion: packs rated above this are not engaged

	Hostiles     int  // awake hostile monsters inside the fight radius
	PackSeen     int  // monsters of a not-yet-engaged pack in view (asleep or far)
	ChampionPack bool // a champion (not a boss) is among the hostiles in range: Tuning.ChampionEarlyPct applies in early Normal
	WeakPack     bool // every hostile in range is at most clvl+1: a weapon swing is enough, retreating is not
	PackDanger   int  // 0 trash, 1 champions/ranged shamans, 2 elite/boss/dangerous affix, 3 deadly (Hell uniques)
	SinceFight   float64
	MercKnown    bool // a merc was hired this run
	MercLifePct  int  // 0 = dead, only meaningful when MercKnown
	MercLevel    int  // merc level (stat 12, U); 0 = unknown
	Skills       []Skill
	BackoffSteps int     // consecutive out-of-mana back-off steps in this fight (combat counts them)
	ManaRegenB   int     // +% mana regeneration from gear
	LifeRegen    float64 // life per second from gear/skills; 0 for an ordinary character (U: no natural life regen)
}

func pct(v, max int) int {
	if max <= 0 {
		return 0
	}
	return v * 100 / max
}

func (in Input) LifePct() int { return pct(in.Life, in.MaxLife) }
func (in Input) ManaPct() int { return pct(in.Mana, in.MaxMana) }

// MercDead reports a hired merc that is dead.
func (in Input) MercDead() bool { return in.MercKnown && in.MercLifePct <= 0 }

// cheapestCast is the lowest positive cost among attack spells, 0 when none (a melee class).
func (in Input) cheapestCast() int {
	best := 0
	for _, s := range in.Skills {
		if s.Melee || s.Cost <= 0 {
			continue
		}
		if best == 0 || s.Cost < best {
			best = s.Cost
		}
	}
	return best
}

// hasCheaper reports whether some attack spell costs less than the first (preferred) one.
func (in Input) hasCheaper() bool {
	first := 0
	for _, s := range in.Skills {
		if s.Melee || s.Cost <= 0 {
			continue
		}
		if first == 0 {
			first = s.Cost
		} else if s.Cost < first {
			return true
		}
	}
	return false
}

func (in Input) hasMelee() bool {
	for _, s := range in.Skills {
		if s.Melee {
			return true
		}
	}
	return false
}

func (in Input) hasFree() bool {
	for _, s := range in.Skills {
		if !s.Melee && s.Cost == 0 && s.Ready {
			return true
		}
	}
	return false
}

// ManaNeed ranks how much a class depends on mana: 0 none (Barbarian), 1 occasional (Amazon, Paladin,
// Assassin), 2 every attack (Sorceress, Necromancer, Druid). docs/classes/*.md section 7 (S).
func ManaNeed(c d2.Class) int {
	switch c {
	case d2.Sorceress, d2.Necromancer, d2.Druid:
		return 2
	case d2.Barbarian:
		return 0
	}
	return 1
}

// manaFullSecs: seconds to refill an empty mana pool from natural regeneration. U: D2 gives a flat fraction of the
// pool per frame, so the time to refill does not depend on pool size; 100 s is the usual quoted figure, class
// charstats ManaRegen differences are ignored. Calibrate with a live measurement (docs/resource.md checklist).
const manaFullSecs = 100.0

// ManaPerSec is the natural regeneration rate.
func ManaPerSec(maxMana, bonusPct int) float64 {
	return float64(maxMana) / manaFullSecs * (1 + float64(bonusPct)/100)
}

// RestSecs estimates the seconds to regenerate from Mana to target (U, see manaFullSecs).
func (in Input) RestSecs(target int) float64 {
	if target <= in.Mana {
		return 0
	}
	r := ManaPerSec(in.MaxMana, in.ManaRegenB)
	if r <= 0 {
		return 1e9
	}
	return float64(target-in.Mana) / r
}

// LifeRestSecs estimates the seconds to reach target life; huge when the character has no regeneration.
func (in Input) LifeRestSecs(target int) float64 {
	if target <= in.Life {
		return 0
	}
	if in.LifeRegen <= 0 {
		return 1e9
	}
	return float64(target-in.Life) / in.LifeRegen
}

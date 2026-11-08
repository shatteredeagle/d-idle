package resource

import (
	"fmt"
	"time"
)

// RecoveryInput is what the recovery decision reads (docs/recovery.md).
type RecoveryInput struct {
	Gold      int
	Shortfall int // KitShortfall: gold the missing portal and potions cost
	Sellable  int // pack items a sell trip would turn into gold
	SellMin   int // sellable items that justify a trip
	Now       time.Time
	Since     time.Time // when the running recovery started
	LastEnd   time.Time // when the last one ended (zero: never)
	Was       bool      // a recovery is running
	Force     bool      // a death streak with an empty purse: start even inside the cooldown
	StartGold int       // gold when the running recovery started
	Stalled   bool      // the last recovery ended as stalled (RecoveryCheck reason stallReason)
	LowKit    bool      // KitLow: the healing potions are under the floor (Gate 1 rerun: 6 virtual minutes fighting with none)
}

// RecoveryVerdict is the answer of RecoveryCheck. Start and End are the edges (log them once).
type RecoveryVerdict struct {
	On, Start, End bool
	Reason         string
}

// RecoveryForceDeaths deaths within RecoveryForceMin minutes with the purse under the kit price start a recovery even
// inside the cooldown (G1 attempt 11: five deaths in seven minutes at gold 0 right after a recovery had ended).
const (
	RecoveryForceDeaths = 3
	RecoveryForceMin    = 30
)

// KitLow reports healing potions (belt plus pack) under RecoveryPotionFloor. With an unaffordable kit that overrides the
// cooldown: a character one potion from none must not keep pushing the quest (Gate 1 rerun, docs/fixbelt.md).
func KitLow(healing int, t Tuning) bool {
	return t.RecoveryPotionFloor > 0 && healing < t.RecoveryPotionFloor
}

// KitReason is the End reason of a recovery whose purpose is met: the missing kit is affordable, the same test that
// keeps a recovery from starting (recoveryEndOnKit; TASK-183: the bar of 500 and the target of 900 are for gear, a
// character at clvl 1 to 5 only needs the portal and the potions).
const KitReason = "kit affordable"

// StallReason is the End reason of a recovery whose gold income is too low to reach the bar (docs/fresh-start-pace.md).
const StallReason = "stalled"

// RecoveryCheck is the recovery state machine. A human with an empty purse does not push the quest: he drops back to easy
// monsters, farms gold and sellable loot, restocks and returns. Start: gold is under RecoveryKitGold and the kit it lacks
// costs more than the gold, outside the cooldown. End: gold reached RecoveryTargetGold and the loot is sold, or the time
// cap ran out (so a dry area cannot hold the character for good).
func RecoveryCheck(in RecoveryInput, t Tuning) RecoveryVerdict {
	if t.RecoveryOn <= 0 {
		return RecoveryVerdict{Reason: "off", End: in.Was}
	}
	if !in.Was {
		cool := time.Duration(t.RecoveryCooldownMin) * time.Minute
		switch {
		case in.Gold >= t.RecoveryKitGold:
			return RecoveryVerdict{Reason: "gold covers the kit"}
		case in.Shortfall <= in.Gold:
			return RecoveryVerdict{Reason: "the missing kit is affordable"}
		case !in.LastEnd.IsZero() && in.Now.Sub(in.LastEnd) < cool && !in.Force && !in.LowKit:
			return RecoveryVerdict{Reason: "cooldown"}
		case in.Stalled && !in.LastEnd.IsZero() && in.Now.Sub(in.LastEnd) < time.Duration(t.RecoveryStallHoldMin)*time.Minute && !in.Force:
			return RecoveryVerdict{Reason: "stalled hold"}
		}
		return RecoveryVerdict{On: true, Start: true, Reason: fmt.Sprintf("gold %d < kit %d", in.Gold, t.RecoveryKitGold)}
	}
	ran := in.Now.Sub(in.Since)
	switch {
	case t.RecoveryMaxMin > 0 && ran >= time.Duration(t.RecoveryMaxMin)*time.Minute:
		return RecoveryVerdict{End: true, Reason: fmt.Sprintf("time cap %d min", t.RecoveryMaxMin)}
	case t.RecoveryStallMin > 0 && ran >= time.Duration(t.RecoveryStallMin)*time.Minute && in.Gold-in.StartGold < t.RecoveryStallGold:
		return RecoveryVerdict{End: true, Reason: StallReason}
	case t.RecoveryEndOnKit > 0 && in.Shortfall <= in.Gold:
		return RecoveryVerdict{End: true, Reason: KitReason}
	case in.Gold >= t.RecoveryTargetGold && in.Sellable < max(in.SellMin, 1):
		return RecoveryVerdict{End: true, Reason: fmt.Sprintf("target %d reached, loot sold", t.RecoveryTargetGold)}
	}
	return RecoveryVerdict{On: true}
}

// RecoveryKitReason prefixes the chore reason of the recovery kit trip (bot counts those trips by it).
const RecoveryKitReason = "recovery restock: no healing potion,"

// RecoveryKitInput is what RecoveryKitTrip reads.
type RecoveryKitInput struct {
	Recovering bool
	InTown     bool
	Healing    int  // healing and rejuvenation potions in belt and pack
	Gold       int  // purse (plus the stash when no potion is left, as choresDue counts it)
	Capped     bool // KitTripCap restock trips already made without a potion to show for it
	WayHome    bool // a portal source, a walkable home area or an activated waypoint
}

// RecoveryKitTrip decides the short town trip of a recovering character with no healing potion: once the gold covers one
// potion (KitPotionGold) it buys the minimal kit (the visit is capped at RecoveryPotionFloor potions, bot.potionFloor)
// and goes back to farming for RecoveryTargetGold. Gate 1 seed 19 farmed 18 virtual minutes at 54..73 gold unprotected.
func RecoveryKitTrip(in RecoveryKitInput, t Tuning) (bool, string) {
	if t.RecoveryOn <= 0 || !in.Recovering || in.InTown || in.Healing > 0 || !in.WayHome || in.Capped ||
		t.KitPotionGold <= 0 || in.Gold < t.KitPotionGold {
		return false, ""
	}
	return true, fmt.Sprintf("%s gold %d", RecoveryKitReason, in.Gold)
}

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
		case !in.LastEnd.IsZero() && in.Now.Sub(in.LastEnd) < cool && !in.Force:
			return RecoveryVerdict{Reason: "cooldown"}
		}
		return RecoveryVerdict{On: true, Start: true, Reason: fmt.Sprintf("gold %d < kit %d", in.Gold, t.RecoveryKitGold)}
	}
	ran := in.Now.Sub(in.Since)
	switch {
	case t.RecoveryMaxMin > 0 && ran >= time.Duration(t.RecoveryMaxMin)*time.Minute:
		return RecoveryVerdict{End: true, Reason: fmt.Sprintf("time cap %d min", t.RecoveryMaxMin)}
	case in.Gold >= t.RecoveryTargetGold && in.Sellable < max(in.SellMin, 1):
		return RecoveryVerdict{End: true, Reason: fmt.Sprintf("target %d reached, loot sold", t.RecoveryTargetGold)}
	}
	return RecoveryVerdict{On: true}
}

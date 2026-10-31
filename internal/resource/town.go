package resource

// TripMethod is how to reach town.
type TripMethod int

const (
	NoTrip TripMethod = iota
	ByPortal
	ByWalk
)

func (m TripMethod) String() string { return [...]string{"none", "portal", "walk"}[m] }

// walkableHome are the act 1 fields from which ToTown can walk back without a scroll: Blood Moor (next to the camp),
// the Den of Evil (a dead end off Blood Moor) and Cold Plains (waypoint, or via Blood Moor); town.walkToTown handles
// exactly these hops. V live for Blood Moor; the Den and Cold Plains hops are U (found offline 2026-10-29: a poor
// clvl 4 Soaky fought 25 minutes in the Den because the policy thought town was too far).
var walkableHome = map[int]bool{2: true, 8: true, 3: true}

// NearTownArea reports whether a character without a scroll can still walk home from area (Input.NearTown).
func NearTownArea(area int) bool { return walkableHome[area] }

// Trip is the TownTripTrigger result.
type Trip struct {
	Go     bool
	Futile bool // a reason exists but a trip cannot fix it (no gold) and it is not free healing
	Method TripMethod
	Reason string
}

// TownTripTrigger decides, outside a fight, whether a trip to town is worth its time (about 60-90 s).
//
// Reasons: healing potions at or below TripHPMin; casters at or below TripMPMin mana potions or mana-starved (rest to
// the cast budget exceeds the timeout); no way to regenerate life and life below TripHealPct (the healer is free). Potion reasons are
// futile when gold is below TripMinGold, or when the last trip ended with gold within FutileStep of now (it could not fix it).
func TownTripTrigger(in Input, t Tuning) Trip {
	if in.InTown {
		return Trip{Reason: "already in town"}
	}
	if in.Hostiles > 0 {
		return Trip{Reason: "fight on (AbortFight handles emergencies)"}
	}
	method := NoTrip
	switch {
	case in.HasTP:
		method = ByPortal
	case in.NearTown:
		method = ByWalk
	}
	var potion, heal string
	if in.Belt.HP <= t.TripHPMin {
		potion = "healing potions low"
	}
	if in.manaNeed() >= 2 && in.Belt.MP <= t.TripMPMin {
		starved := in.ManaPct() < t.RestStartManaPct && in.RestSecs(ManaTarget(in, t)) > float64(t.RestMaxSec)
		if starved && in.Belt.MP == 0 {
			// The healer refills mana as well as life and is free, so a starved caster with no mana potion gets
			// the heal even when too poor to buy anything (live: Soaky "out of mana: backing off" for 10 minutes).
			heal = "mana-starved, free heal"
		}
		if starved || in.Belt.MP == 0 && in.Belt.HP <= t.TripHPMin+1 {
			if potion != "" {
				potion += " and mana-starved"
			} else {
				potion = "mana-starved caster out of mana potions"
			}
		}
	}
	if in.LifeRegen <= 0 && in.LifePct() < t.TripHealPct {
		heal = "life low, cannot regenerate, free heal"
	}
	// Poor and already went to town twice for nothing: the free heal alone no longer justifies a trip while the
	// life is above the never-engage line (the flee<->town loop of the Booti soak, V from the log).
	if potion == "" && heal != "" && in.IdleTrips >= 2 && in.LifePct() >= t.MinEngageLifePct {
		return Trip{Reason: "heal-only trip suppressed after idle trips"}
	}
	if potion == "" && heal == "" {
		return Trip{Reason: "nothing needed"}
	}
	futile := false
	if potion != "" {
		if in.Gold < t.TripMinGold {
			futile = true
		} else if in.LastTripGold >= 0 && in.Gold-in.LastTripGold < t.FutileStep {
			futile = true
		}
	}
	reason := potion
	if heal != "" {
		reason = heal
		if potion != "" && !futile {
			reason = potion + "; " + heal
		}
		futile = false // the heal is free, so the trip is never wasted
	}
	if futile {
		return Trip{Futile: true, Method: method, Reason: reason + " (futile: not enough gold)"}
	}
	if method == NoTrip {
		return Trip{Reason: reason + " (no portal, town too far)"}
	}
	return Trip{Go: true, Method: method, Reason: reason}
}

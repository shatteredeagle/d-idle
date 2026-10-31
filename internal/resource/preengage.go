package resource

// PreAction is what to do before walking into a pack.
type PreAction int

const (
	Proceed     PreAction = iota
	RestHere              // stand still and regenerate (the pack is asleep or far)
	RetreatRest           // walk back out of sight, then regenerate
	DrinkMana             // rest would take too long: one mana potion
	DrinkLife             // one healing potion before engaging
	TownHeal              // free healing (and restock) in town: cannot regenerate and no cheap fix
	Avoid                 // too weak to engage, nothing to fix it: skip this pack
)

func (a PreAction) String() string {
	return [...]string{"proceed", "rest", "retreat-rest", "drink-mana", "drink-life", "town-heal", "avoid"}[a]
}

// PreDecision is the PreEngage result.
type PreDecision struct {
	Action     PreAction
	WaitSec    int // rest timeout for RestHere/RetreatRest (caller re-asks every couple of seconds and stops at Proceed)
	TargetMana int // mana to reach before engaging
	Reason     string
}

// A rest longer than two thirds of RestMaxSec (about 30 s) is not worth it before a harmless pack when a cheaper way to fight exists.
// ManaTarget is the mana worth having before a pack: the larger of the percent floor and the cast budget for the
// pack, capped at 90% of the pool (a full pool is not worth waiting for).
func ManaTarget(in Input, t Tuning) int {
	cost := in.cheapestCast()
	if cost == 0 {
		return 0
	}
	n := in.PackSeen
	if n < 1 {
		n = 1
	}
	need := n * t.CastsPerMonster * cost
	floor := in.MaxMana * t.RestStartManaPct / 100
	if floor > need {
		need = floor
	}
	if hi := in.MaxMana * 90 / 100; need > hi {
		need = hi
	}
	return need
}

// PreEngage decides what to do while a pack is in view but not yet fighting. Called with Hostiles == 0; with hostiles
// already on top of the character it answers Proceed (AttackChoice and AbortFight take over).
func PreEngage(in Input, t Tuning) PreDecision {
	if in.InTown || in.Hostiles > 0 {
		return PreDecision{Action: Proceed, Reason: "fight on or town"}
	}
	// No potion to rescue a bad fight and no gold for one: leave packs above the poverty limit alone.
	if in.Poor && in.Belt.HP == 0 && in.Belt.RV == 0 && in.PackDanger > in.MaxPackDanger {
		return PreDecision{Action: Avoid, Reason: "poor, no potions, pack danger above limit"}
	}
	lp := in.LifePct()
	// Life first: it does not regenerate by itself (U), so resting only helps with gear regeneration.
	if lp < t.EngageLifePct {
		if secs := in.LifeRestSecs(in.MaxLife * t.EngageLifePct / 100); secs <= float64(t.RestMaxSec) {
			return PreDecision{Action: restAction(in, t), WaitSec: int(secs) + 1, Reason: "life regen"}
		}
		switch {
		case in.Belt.HP > t.HPReserve && (lp < t.MinEngageLifePct || in.PackDanger >= 1):
			return PreDecision{Action: DrinkLife, Reason: "life below engage line, spare potion"}
		case in.HasTP || in.NearTown:
			if lp < t.MinEngageLifePct || in.PackDanger >= 1 || in.Belt.HP == 0 {
				return PreDecision{Action: TownHeal, Reason: "life low, free heal"}
			}
		case lp < t.MinEngageLifePct:
			return PreDecision{Action: Avoid, Reason: "life too low, no fix"}
		}
	}
	target := ManaTarget(in, t)
	// One or two trash monsters need their cast budget, not the percent floor: live (i4, clvl 12, no gold for mana
	// potions) every single Quill Rat cost a 40 s rest to 60% of the pool, which regenerates only about 0.5/s.
	if cost := in.cheapestCast(); cost > 0 && in.PackSeen <= 2 && in.PackDanger == 0 && in.Mana >= max(in.PackSeen, 1)*t.CastsPerMonster*cost {
		return PreDecision{Action: Proceed, TargetMana: target, Reason: "trash pack, cast budget enough"}
	}
	if target == 0 || in.Mana >= target {
		return PreDecision{Action: Proceed, TargetMana: target, Reason: "mana enough"}
	}
	secs := in.RestSecs(target)
	// A fallback attack (staff swing, free skill) makes a long rest pointless for a harmless pack: go in with what we
	// have and let AttackChoice drop to the cheap skill (U threshold; Booti paid 30-40 s of rest per pack).
	if secs > float64(t.RestMaxSec)*2/3 && in.PackDanger == 0 && in.Mana >= in.cheapestCast() && (in.hasMelee() || in.hasFree() || in.hasCheaper()) {
		return PreDecision{Action: Proceed, TargetMana: target, Reason: "long rest, cheap fallback skill"}
	}
	if secs <= float64(t.RestMaxSec) {
		return PreDecision{Action: restAction(in, t), WaitSec: int(secs) + 1, TargetMana: target, Reason: "rest to cast budget"}
	}
	// Resting would exceed the timeout. Rest as far as the timeout allows when the pool is merely low; with a spare
	// mana potion drink it; otherwise go in and let AttackChoice use the fallback ladder.
	if in.hasMelee() || in.hasFree() {
		return PreDecision{Action: Proceed, TargetMana: target, Reason: "rest too long, fallback skills"}
	}
	if in.Belt.MP > 0 && in.manaNeed() >= 2 && in.ManaPct() < t.DrinkManaPct*2 {
		return PreDecision{Action: DrinkMana, TargetMana: target, Reason: "rest too long, mana potion"}
	}
	return PreDecision{Action: restAction(in, t), WaitSec: t.RestMaxSec, TargetMana: target, Reason: "rest to timeout, no fallback"}
}

func restAction(in Input, t Tuning) PreAction {
	if in.PackDanger >= t.RetreatDanger {
		return RetreatRest
	}
	return RestHere
}

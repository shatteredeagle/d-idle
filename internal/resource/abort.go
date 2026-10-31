package resource

// AbortKind is the escalation when a fight is lost.
type AbortKind int

const (
	Continue   AbortKind = iota
	RetreatOut           // back away and rest, do not leave the area
	PortalOut            // open a town portal
	LeaveGame            // chicken
	Flee                 // break away from the hostiles (no potion, no portal), keep running while they follow
)

func (a AbortKind) String() string {
	return [...]string{"continue", "retreat", "portal", "leave", "flee"}[a]
}

// AbortDecision is the AbortFight result.
type AbortDecision struct {
	Kind   AbortKind
	Reason string
}

// AbortFight decides whether to stop fighting. It runs after PotionPolicy (a potion can still save the run).
//
//	life <= chicken, nothing to drink: portal when it is still comfortably above the chicken line and a portal exists
//	  (about 1.3 s), else leave the game (about 1.8 s)
//	no potion, no portal, life under FleeLifePct (and above the chicken line): Flee, the caller moves away from the hostiles
//	life and mana gone: life under the drink line with no healing/rejuv potion AND no way to attack (mana below the
//	  cheapest cast, no mana potion/rejuv, no melee/free) with hostiles left: portal, else retreat, else leave
//	merc dead with MercDeadPack or more hostiles, or dangerous pack, life under the engage line: portal or retreat
//	mana gone alone is not an abort: AttackChoice has the ladder.
func AbortFight(in Input, t Tuning) AbortDecision {
	if in.InTown || in.Hostiles <= 0 || in.Life <= 0 {
		return AbortDecision{Reason: "nothing to abort"}
	}
	lp, b := in.LifePct(), in.Belt
	chicken := t.Chicken(in.Diff)
	noHeal := b.HP == 0 && b.RV == 0
	if lp <= chicken && noHeal {
		if in.HasTP && lp > chicken-8 {
			return AbortDecision{Kind: PortalOut, Reason: "life at chicken line, no potion, portal still fast enough"}
		}
		return AbortDecision{Kind: LeaveGame, Reason: "life at chicken line, no potion"}
	}
	if thinBeltRetreat(in, t) {
		return escape(in, "belt thin, life under the drink line")
	}
	// Nothing to drink and no way home: trading blows only ends at the chicken line. Break away early instead.
	if noHeal && !in.HasTP && lp < t.FleeLifePct {
		return AbortDecision{Kind: Flee, Reason: "no potion, no portal, life falling: break away"}
	}
	cheap := in.cheapestCast()
	canAttack := in.hasMelee() || in.hasFree() || b.MP > 0 || b.RV > 0 || cheap == 0 || in.Mana >= cheap
	if noHeal && lp < t.DrinkLifePct && !canAttack {
		return escape(in, "life and mana gone")
	}
	if noHeal && lp < t.DrinkLifePct+10 && cheap > 0 && in.Mana < cheap && !in.hasMelee() {
		return escape(in, "no potions, no mana, no melee")
	}
	if in.MercDead() && lp < t.EngageLifePct && (in.Hostiles >= t.MercDeadPack || in.PackDanger >= 2) {
		return escape(in, "merc dead, hostiles remain, life low")
	}
	return AbortDecision{Reason: "keep fighting"}
}

func escape(in Input, why string) AbortDecision {
	switch {
	case in.HasTP:
		return AbortDecision{Kind: PortalOut, Reason: why + ": portal"}
	case in.NearTown:
		return AbortDecision{Kind: RetreatOut, Reason: why + ": retreat toward town"}
	}
	return AbortDecision{Kind: LeaveGame, Reason: why + ": leave"}
}

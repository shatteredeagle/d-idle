package resource

// AttackKind is what the fighter does this tick.
type AttackKind int

const (
	Cast    AttackKind = iota // cast Skill
	Melee                     // basic attack with the weapon
	Free                      // a zero-cost skill
	Wait                      // do nothing for WaitMs (stop casting, mana regenerates)
	Retreat                   // nothing affordable and unsafe to stand: back off and rest
)

func (k AttackKind) String() string {
	return [...]string{"cast", "melee", "free", "wait", "retreat"}[k]
}

// AttackDecision is the AttackChoice result.
type AttackDecision struct {
	Kind   AttackKind
	Skill  Skill
	WaitMs int
	Reason string
}

// AttackChoice picks the attack with a low-mana ladder (Kolbot LowManaSkill, docs/prior-art/04-combat.md):
//  1. the first ready spell the mana pays for, in preference order; below EconomyManaPct the cheapest one instead
//  2. a ready free skill
//  3. melee, when life, pack size and danger allow it
//  4. stop casting: wait (mana regenerates) while standing is safe, else retreat.
//
// Callers run PotionPolicy first; a mana potion already drunk shows up as more Mana here.
func AttackChoice(in Input, t Tuning) AttackDecision {
	var pick *Skill
	economy := in.ManaPct() < t.EconomyManaPct
	if !economy && in.Hostiles > 0 {
		// Not enough mana for the preferred spell on every monster in view: switch to the cheapest now instead of
		// running dry halfway through the pack.
		for _, s := range in.Skills {
			if !s.Melee && s.Cost > 0 && s.Ready {
				economy = in.Mana < in.Hostiles*t.CastsPerMonster*s.Cost
				break
			}
		}
	}
	for i := range in.Skills {
		s := &in.Skills[i]
		if s.Melee || s.Cost <= 0 || !s.Ready || s.Cost > in.Mana {
			continue
		}
		if pick == nil {
			pick = s
			if !economy {
				break
			}
			continue
		}
		if s.Cost < pick.Cost {
			pick = s
		}
	}
	if pick != nil {
		why := "affordable"
		if economy {
			why = "economy: cheapest"
		}
		return AttackDecision{Kind: Cast, Skill: *pick, Reason: why}
	}
	for _, s := range in.Skills {
		if !s.Melee && s.Cost <= 0 && s.Ready {
			return AttackDecision{Kind: Free, Skill: s, Reason: "free skill"}
		}
	}
	if in.hasMelee() && in.LifePct() >= t.MeleeMinLifePct && (in.Hostiles <= t.MeleeMaxPack || in.WeakPack) && in.PackDanger < 2 {
		for _, s := range in.Skills {
			if s.Melee {
				return AttackDecision{Kind: Melee, Skill: s, Reason: "out of mana, melee fallback"}
			}
		}
	}
	// A back-off loop that has run its course: the pack follows, mana does not come back on the move, and standing
	// still is "unsafe" for ever. Swing the weapon (any pack size) rather than run in circles; danger 3 still flees.
	if t.BackoffMaxSteps > 0 && in.BackoffSteps >= t.BackoffMaxSteps && in.hasMelee() && in.LifePct() >= t.MeleeMinLifePct && in.PackDanger < 3 {
		for _, s := range in.Skills {
			if s.Melee {
				return AttackDecision{Kind: Melee, Skill: s, Reason: "out of mana, back-off cap, melee"}
			}
		}
	}
	if in.LifePct() >= t.MinEngageLifePct && in.PackDanger < 2 && (in.Hostiles <= t.MeleeMaxPack || in.WeakPack) {
		return AttackDecision{Kind: Wait, WaitMs: t.LowManaWaitMs, Reason: "out of mana, stop casting"}
	}
	return AttackDecision{Kind: Retreat, Reason: "out of mana, not safe to stand"}
}

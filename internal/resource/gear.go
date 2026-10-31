package resource

import (
	"fmt"
	"strings"
)

// GearGateLevel is the character level from which missing potions and gold count against the character, and below
// which the old hard gate does not apply. A fresh character has no body armor (U: classic starting kits).
const GearGateLevel = 5

// Gate modes (knob resource.gearGateMode).
const (
	GateHard   = 0 // the old binary gate: no weapon, no body armor or no potions and no gold => gated
	GateGraded = 1 // risk points decide (docs/gear-gate.md)
)

// Risk is the graded danger of playing on with the current kit.
type Risk int

const (
	RiskLow Risk = iota
	RiskMedium
	RiskHigh
)

func (r Risk) String() string { return [...]string{"low", "medium", "high"}[r] }

// GearInput is what the gear gate reads. Zero values of the optional fields mean "unknown, add no points".
type GearInput struct {
	Level   int
	Gold    int
	Potions int  // healing potions in belt and pack
	Weapon  bool // a weapon is equipped (either weapon set)
	Body    bool // a body armor is equipped

	Caster     bool // Sorceress, Necromancer, Druid: spells are the damage, the weapon barely matters
	NeedWeapon bool // the class cannot play without a weapon (NeedsWeapon); matters below EarlyMax only
	Difficulty int  // 0 Normal, 1 Nightmare, 2 Hell
	Act        int  // 1..5, 0 unknown
	MaxLife    int  // 0 unknown
	MinResist  int  // lowest of fire, cold, lightning as displayed (already includes the difficulty penalty)
	AreaMLvl   int  // monster level of the area in question, 0 = none
	Deaths     int  // deaths in the last hour
	DeathsHere int  // deaths in the recent window in the area in question
}

// GearTune are the graded-model knobs.
type GearTune struct {
	Mode       int // GateHard or GateGraded
	MediumAt   int // risk points at which risk is medium
	HighAt     int // risk points at which risk is high (gated)
	DeathsHigh int // deaths in the last hour (or in one area) from which deaths add risk
	EarlyMax   int // below this clvl armor never counts and a weapon only for classes that need one; 0 = policy off
}

// DefaultGearTune are the shipped defaults.
func DefaultGearTune() GearTune {
	return GearTune{Mode: GateGraded, MediumAt: 3, HighAt: 6, DeathsHigh: 4, EarlyMax: DefaultEarlyGearMax}
}

// GearVerdict says how dangerous the kit is and whether the character may enter the dangerous field at all.
type GearVerdict struct {
	Gated    bool // only town and the safest farm area are allowed
	Naked    bool // weapon or body armor missing
	CanRebuy bool // a missing slot is worth a trip and the gold covers the cheapest weapon and armor
	Risk     Risk
	Points   int
	Reason   string
}

// lowLife is the max life below which the difficulty gets dangerous (S: a clvl 9 Sorceress has about 100 life; U: floors).
var lowLife = [3]int{40, 250, 500}

// GearGate evaluates the kit. In GateHard mode it is the death-spiral guard of WP DEATHSPIRAL (V: Soaky 2026-10-29,
// clvl 8, gold 0, no weapon, no body armor, no potions, 4 deaths in 11 minutes). In GateGraded mode a point score
// decides: only HIGH gates, and a caster in Normal with life and potions never does (V: G1 attempts 6-7 froze a clvl 9
// Sorceress for 2 hours on a hard gate).
func GearGate(in GearInput, rebuyGold int, t GearTune) GearVerdict {
	d := min(max(in.Difficulty, 0), 2)
	// Early game (docs/sme/early-gear-policy.md): armor is ignored, and so is the weapon of a class that does not need
	// one. A caster in Normal never counts armor at any level (V: G1 attempts 6-11 gated a clvl 9-11 Sorceress on it).
	early := Early(in.Level, t.EarlyMax)
	missB := !in.Body && !early && !(in.Caster && d == 0)
	missW := !in.Weapon && !(early && !in.NeedWeapon)
	v := GearVerdict{Naked: missW || missB}
	if t.Mode == GateHard {
		return hardGate(in, rebuyGold, v, missW, missB)
	}
	var why []string
	add := func(n int, s string) {
		if n > 0 {
			v.Points += n
			why = append(why, fmt.Sprintf("%s +%d", s, n))
		}
	}
	if missB {
		add(1+2*d, "no body armor")
	}
	if missW {
		if in.Caster {
			add(d, "no weapon")
		} else {
			add(t.HighAt, "no weapon (melee cannot kill)")
		}
	}
	add(d*3, []string{"", "nightmare", "hell"}[d])
	switch {
	case in.Act >= 4:
		add(2, "act 4+")
	case in.Act >= 2:
		add(1, "act 2+")
	}
	if in.MaxLife > 0 && in.MaxLife < lowLife[d] {
		add(2, fmt.Sprintf("max life %d", in.MaxLife))
	}
	if d > 0 && in.MinResist < 0 {
		add(d, fmt.Sprintf("resist %d", in.MinResist))
	}
	if in.AreaMLvl > 0 {
		switch gap := in.AreaMLvl - in.Level; {
		case gap >= 6:
			add(3, fmt.Sprintf("area mlvl %d vs clvl %d", in.AreaMLvl, in.Level))
		case gap >= 3:
			add(1, fmt.Sprintf("area mlvl %d vs clvl %d", in.AreaMLvl, in.Level))
		}
	}
	if in.Level >= GearGateLevel {
		switch {
		case in.Potions == 0 && in.Gold == 0:
			add(t.HighAt, "no potions and no gold")
		case in.Potions == 0:
			add(2, "no potions")
		}
	}
	// A death is normal and expected: a few change nothing, a streak in the hour or in this very area does.
	if in.Deaths >= t.DeathsHigh {
		add(in.Deaths-t.DeathsHigh+2, fmt.Sprintf("%d deaths in the hour", in.Deaths))
	}
	if in.DeathsHere >= t.DeathsHigh {
		add(2, fmt.Sprintf("%d deaths here", in.DeathsHere))
	}
	switch {
	case v.Points >= t.HighAt:
		v.Risk = RiskHigh
	case v.Points >= t.MediumAt:
		v.Risk = RiskMedium
	}
	v.Gated = v.Risk == RiskHigh
	v.Reason = strings.Join(why, ", ")
	if v.Reason == "" {
		v.Reason = "kit fine"
	}
	// A caster without a weapon loses nothing worth a trip; any missing slot that matters is.
	worth := missB || (missW && !in.Caster)
	v.CanRebuy = worth && in.Gold >= rebuyGold
	if v.CanRebuy {
		v.Reason += fmt.Sprintf(" (%d gold covers a rebuy)", in.Gold)
	}
	return v
}

func hardGate(in GearInput, rebuyGold int, v GearVerdict, missW, missB bool) GearVerdict {
	if in.Level < GearGateLevel {
		return v
	}
	destitute := in.Potions == 0 && in.Gold == 0
	if !v.Naked && !destitute {
		return v
	}
	v.Gated, v.Risk = true, RiskHigh
	switch {
	case missW && missB:
		v.Reason = "no weapon and no body armor"
	case missW:
		v.Reason = "no weapon"
	case missB:
		v.Reason = "no body armor"
	default:
		v.Reason = "no potions and no gold"
	}
	v.CanRebuy = v.Naked && in.Gold >= rebuyGold
	if v.CanRebuy {
		v.Reason += fmt.Sprintf(" (%d gold covers a rebuy)", in.Gold)
	}
	return v
}

package resource

import (
	"testing"

	"d2lod-bot/internal/d2"
)

func TestGearGateHard(t *testing.T) {
	kit := GearInput{Level: 8, Gold: 300, Potions: 4, Weapon: true, Body: true}
	cases := []struct {
		name               string
		mut                func(*GearInput)
		gated, naked, shop bool
	}{
		{"healthy clvl 8", nil, false, false, false},
		{"soaky: naked, broke, no potions", func(i *GearInput) { i.Weapon, i.Body, i.Gold, i.Potions = false, false, 0, 0 }, true, true, false},
		{"no weapon only", func(i *GearInput) { i.Weapon = false }, true, true, true},
		{"no body armor, poor", func(i *GearInput) { i.Body, i.Gold = false, 20 }, true, true, false},
		{"destitute but geared", func(i *GearInput) { i.Gold, i.Potions = 0, 0 }, true, false, false},
		{"no potions but gold", func(i *GearInput) { i.Potions = 0 }, false, false, false},
		{"fresh clvl 3 has no armor", func(i *GearInput) { i.Level, i.Body = 3, false }, false, true, false},
	}
	for _, c := range cases {
		in := kit
		if c.mut != nil {
			c.mut(&in)
		}
		v := GearGate(in, 150, GearTune{Mode: GateHard})
		if v.Gated != c.gated || v.Naked != c.naked || v.CanRebuy != c.shop {
			t.Errorf("%s: %+v", c.name, v)
		}
	}
}

func TestGearGateGraded(t *testing.T) {
	tn := DefaultGearTune()
	sorc := GearInput{Level: 9, Gold: 200, Potions: 4, Caster: true, MaxLife: 100, Act: 1}
	barb := GearInput{Level: 9, Gold: 200, Potions: 4, MaxLife: 200, Act: 1, NeedWeapon: true}
	cases := []struct {
		name string
		in   GearInput
		mut  func(*GearInput)
		risk Risk
	}{
		{"sorc naked normal act 1", sorc, nil, RiskLow},
		{"sorc naked normal act 1 three deaths", sorc, func(i *GearInput) { i.Deaths = 3 }, RiskLow},
		{"sorc naked normal act 1 broke but potions", sorc, func(i *GearInput) { i.Gold = 0 }, RiskLow},
		{"sorc naked normal act 4", sorc, func(i *GearInput) { i.Act = 4 }, RiskLow},
		{"barbarian naked", barb, nil, RiskHigh},
		{"barbarian only body missing", barb, func(i *GearInput) { i.Weapon = true }, RiskLow},
		{"sorc naked nightmare", sorc, func(i *GearInput) { i.Level, i.Difficulty, i.MinResist, i.MaxLife = 30, 1, -40, 300 }, RiskHigh},
		{"sorc geared nightmare", sorc, func(i *GearInput) { i.Difficulty, i.Weapon, i.Body, i.MaxLife = 1, true, true, 300 }, RiskMedium},
		{"streak of deaths", sorc, func(i *GearInput) { i.Deaths, i.DeathsHere, i.Act = 6, 4, 2 }, RiskHigh},
		{"potions and gold out", sorc, func(i *GearInput) { i.Potions, i.Gold = 0, 0 }, RiskHigh},
		{"potions out, gold left", sorc, func(i *GearInput) { i.Potions = 0 }, RiskLow},
		{"area far above clvl", sorc, func(i *GearInput) { i.AreaMLvl, i.Act = 20, 2 }, RiskMedium},
	}
	for _, c := range cases {
		in := c.in
		if c.mut != nil {
			c.mut(&in)
		}
		if v := GearGate(in, 150, tn); v.Risk != c.risk || v.Gated != (c.risk == RiskHigh) {
			t.Errorf("%s: want %v got %+v", c.name, c.risk, v)
		}
	}
}

func TestGearGateGradedRebuy(t *testing.T) {
	tn := DefaultGearTune()
	in := GearInput{Level: 30, Gold: 300, Potions: 4, Caster: true, Body: true, Difficulty: 1}
	if v := GearGate(in, 150, tn); v.CanRebuy {
		t.Errorf("caster without weapon is no trip: %+v", v)
	}
	in.Body = false
	if v := GearGate(in, 150, tn); !v.CanRebuy {
		t.Errorf("missing body armor with gold is a trip: %+v", v)
	}
}

func TestEarlyGearPolicy(t *testing.T) {
	tn := DefaultGearTune()
	sorc := GearInput{Level: 11, Gold: 300, Potions: 4, Caster: true, MaxLife: 100, Act: 1}
	if v := GearGate(sorc, 150, tn); v.Gated || v.Naked || v.CanRebuy || v.Risk != RiskLow {
		t.Errorf("naked clvl 11 Sorceress in Normal must be fine, no trip: %+v", v)
	}
	sorc.Gold, sorc.Potions = 0, 0
	if v := GearGate(sorc, 150, tn); !v.Gated {
		t.Errorf("no potions and no gold stays high: %+v", v)
	}
	// a caster in Normal never counts armor, even past the early limit
	late := GearInput{Level: 30, Gold: 300, Potions: 4, Weapon: true, Caster: true, MaxLife: 300, Act: 1}
	if v := GearGate(late, 150, tn); v.Naked || v.CanRebuy || v.Gated {
		t.Errorf("clvl 30 Sorceress in Normal without armor: %+v", v)
	}
	ama := GearInput{Level: 4, Gold: 300, Potions: 2, NeedWeapon: true, Body: false, MaxLife: 100}
	v := GearGate(ama, 150, tn)
	if !v.Gated || !v.Naked || !v.CanRebuy {
		t.Errorf("Amazon without a weapon: high risk and a rebuy: %+v", v)
	}
	ama.Weapon = true
	if v := GearGate(ama, 150, tn); v.Gated || v.Naked || v.CanRebuy {
		t.Errorf("Amazon with a weapon, no armor, clvl 4: %+v", v)
	}
	// policy off (EarlyMax 0): the old rules
	off := tn
	off.EarlyMax = 0
	if v := GearGate(GearInput{Level: 9, Gold: 300, Potions: 2, MaxLife: 100}, 150, off); !v.Naked {
		t.Errorf("policy off: naked: %+v", v)
	}
}

func TestNeedsWeapon(t *testing.T) {
	for c, want := range map[d2.Class]bool{d2.Amazon: true, d2.Paladin: true, d2.Barbarian: true, d2.Assassin: true,
		d2.Sorceress: false, d2.Necromancer: false, d2.Druid: false} {
		if NeedsWeapon(c) != want {
			t.Errorf("%v: want %v", c, want)
		}
	}
}

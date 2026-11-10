package combat

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/gear"
	"d2lod-bot/internal/gear/geartest"
)

// PACKFIGHT: a Paladin with Smite bound and an empty mana pool in a pack of 12 swings the basic attack at once.
func TestPaladinNoManaBigPackSwings(t *testing.T) {
	h, g := newHandGame(t)
	g.p.Class, g.p.Level, g.p.LeftSkill = d2.Paladin, 10, 112
	g.aura = 0
	h.s.mana, h.s.maxM, h.s.manaRegen = 0, 30, 0
	h.s.life, h.s.maxL = 150, 150
	for i := 0; i < 12; i++ {
		h.s.addMon(uint32(10+i), clsFallen, 104+float64(i%4), 100+float64(i/4), 20, 1, 5)
	}
	h.s.meleeDmg = 40
	d := &d2dClick{handGame: g}
	h.f.e.In = d
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	res, err := h.f.Clear(ctx, ClearOpts{})
	t.Logf("res %+v err %v swings %d x %d y %d", res, err, len(d.swings), int(h.s.px), int(h.s.py))
	if res.Kills < 6 || len(d.swings) == 0 {
		t.Fatalf("kills %d swings %d", res.Kills, len(d.swings))
	}
}

// PACKFIGHT: Smite needs a shield (gear.json, V). Without one the skill list used to be empty ("no attack skill
// bound", every target ignored) because the skill dump did not list the basic attack. The profile's "Attack" entry
// must stay usable for a melee class and for a build whose band ends in Attack, whatever the dump says.
func TestAttackStaysWhenGearRulesOutTheBuildSkill(t *testing.T) {
	geartest.Need(t)
	for _, class := range []d2.Class{d2.Paladin, d2.Barbarian} {
		h, g := newHandGame(t)
		g.p.Class, g.p.Level = class, 10
		g.p.Skills = []d2.Skill{{ID: 112, Level: 3}, {ID: 107, Level: 5}} // no id 0, as the live dump
		g.p.LeftSkill = 112
		h.f.eq = &gear.Equipped{Weapon: "hax"}
		for _, sp := range h.f.attackSkills(g.p) {
			if sp.Name == "Attack" {
				return
			}
		}
		t.Errorf("%v: no Attack in %v", class, h.f.attackSkills(g.p))
	}
}

// A swinger backend needs no popup rows to attack with a skill that is not on the left hand.
func TestSwingerNeedsNoListRowsForOffHandSkill(t *testing.T) {
	h, g := newHandGame(t)
	g.p.Level, g.p.LeftSkill = 10, 107 // Sacrifice is on the left; the clvl 10 Paladin list is just Attack, the off-hand skill
	h.f.SetListRows(nil)
	h.f.e.In = &d2dClick{handGame: g}
	found := false
	for _, sp := range h.f.attackSkills(g.p) {
		found = found || sp.Name == "Attack"
	}
	if !found {
		t.Fatalf("Attack missing for a swinger: %v", h.f.attackSkills(g.p))
	}
	h.f.e.In = g
	for _, sp := range h.f.attackSkills(g.p) {
		if sp.Name == "Attack" {
			t.Fatalf("a click backend without list rows cannot select Attack")
		}
	}
}

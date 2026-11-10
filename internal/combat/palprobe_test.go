package combat

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// PALPROBE part B (b): the Paladin combat ladder on the fake game (docs/classes/paladin-probe.md). These run
// offline on the same simulator as TestPaladinFightsWithLeftHand; what the d2d backend does with a held button is
// not testable here (U, card PALHOLD).

// noHold hides HoldClick, like the d2d input (docs/d2d-backend.md G5: no held button).
type noHold struct{ input.Controller }

func skillsOf(ids ...int) []d2.Skill {
	out := []d2.Skill{{ID: 0, Level: 1}}
	for _, id := range ids {
		out = append(out, d2.Skill{ID: id, Level: 3})
	}
	return out
}

// Below clvl 12 (no Zeal) the ladder is Sacrifice then Attack (palprofile); a held button is for Zeal only.
func TestPalProbeHoldClickIsZealOnly(t *testing.T) {
	h, g := newHandGame(t)
	g.p.Level = 10
	g.p.Skills = skillsOf(107, 112, 102, 98) // Sacrifice, Smite, Holy Fire, Might: no Zeal yet
	g.p.LeftSkill = 0
	g.aura = 102
	for i := 0; i < 3; i++ {
		h.s.addMon(uint32(10+i), clsFallen, 106+float64(i)*3, 103, 30, 1, 5)
	}
	h.s.meleeDmg = 12
	res, err := h.f.Clear(context.Background(), ClearOpts{})
	if err != nil || res.Kills != 3 {
		t.Fatalf("%v %v", res, err)
	}
	if g.holds != 0 {
		t.Fatalf("Smite and Attack are clicks, got %d holds", g.holds)
	}
	if g.p.LeftSkill != 107 {
		t.Fatalf("Sacrifice must be the left skill at clvl 10, got %d", g.p.LeftSkill)
	}
}

// payingGame charges the 2 mana of Zeal and Smite per swing (the shared sim only drains mana on right-hand casts).
type payingGame struct{ *handGame }

func (g payingGame) pay() {
	if id := g.p.LeftSkill; id == 106 || id == 112 {
		g.mu.Lock()
		g.mana = max(0, g.mana-2)
		g.mu.Unlock()
	}
}
func (g payingGame) Click(b input.Button, x, y int, m ...input.Mod) error {
	err := g.handGame.Click(b, x, y, m...)
	if b == input.Left && !g.open {
		g.pay()
	}
	return err
}
func (g payingGame) HoldClick(b input.Button, x, y int, d time.Duration) error {
	err := g.handGame.HoldClick(b, x, y, d)
	g.pay()
	return err
}

// Zeal is held while mana lasts, then the ladder drops to the free Attack, then (mana back) up again.
func TestPalProbeLadderFlipsWhenManaDrains(t *testing.T) {
	h, g := newHandGame(t)
	g.p.LeftSkill = 106
	g.aura = 102
	h.s.mana, h.s.manaRegen = 12, 0
	h.f.e.In = payingGame{g}
	for i := 0; i < 6; i++ {
		h.s.addMon(uint32(10+i), clsFallen, 106+float64(i)*2, 103, 60, 0, 5)
	}
	h.s.meleeDmg = 10
	res, err := h.f.Clear(context.Background(), ClearOpts{})
	if err != nil || res.Kills != 6 {
		t.Fatalf("%v %v", res, err)
	}
	if g.holds == 0 {
		t.Fatal("Zeal must be held while mana lasts")
	}
	if g.p.LeftSkill != 0 {
		t.Fatalf("mana drained: the left skill must end on Attack, got %d", g.p.LeftSkill)
	}
}

// d2d degradation: without a held button Zeal is a plain click per tick; the fight still ends, nothing hangs.
func TestPalProbeZealWithoutHoldClickStillFights(t *testing.T) {
	h, g := newHandGame(t)
	g.p.LeftSkill = 106
	g.aura = 102
	h.f.e.In = noHold{Controller: g}
	h.s.addMon(10, clsFallen, 106, 103, 30, 1, 5)
	h.s.meleeDmg = 15
	res, err := h.f.Clear(context.Background(), ClearOpts{})
	if err != nil || res.Kills != 1 || g.holds != 0 {
		t.Fatalf("%v %v holds %d", res, err, g.holds)
	}
}

// Aura choice by what the character knows: the profile ladder Prayer (rest), Holy Fire, Might, Thorns.
func TestPalProbeAuraSelection(t *testing.T) {
	h, _ := newHandGame(t)
	cat := palCatalog()
	h.f.SetCatalog(cat)
	h.f.SetHotkeys(map[int]string{102: "f2", 98: "f1"})
	pick := func(skills []d2.Skill, tg *Target) (string, bool) {
		p := palPlayer()
		p.Level, p.Skills, p.RightSkill = 5, skills, 98
		sp, ok := h.f.wantAura(p, h.f.profileFor(p), tg, false)
		return sp.Name, ok
	}
	if n, ok := pick(skillsOf(98), nil); !ok || n != "Might" {
		t.Errorf("Might only: %q %v", n, ok)
	}
	if n, ok := pick(skillsOf(98, 102), nil); !ok || n != "Holy Fire" {
		t.Errorf("Might + Holy Fire: %q %v", n, ok)
	}
	fi := &Target{HasRes: true, Res: fireImm()}
	if n, ok := pick(skillsOf(98, 102), fi); !ok || n != "Might" {
		t.Errorf("fire immune: %q %v", n, ok)
	}
	if n, ok := pick(skillsOf(102), fi); ok && n == "Holy Fire" {
		t.Errorf("fire immune with only Holy Fire must not pick it")
	}
	if _, ok := pick(skillsOf(), nil); ok {
		t.Error("no aura known: nothing to pick")
	}
}

// Backing off: surrounded and healthy stays and fights; surrounded and hurt (below 65 %) walks away; a single
// attacker at life 40 % or less also walks away (retreatLifePct).
func TestPalProbeMeleeGuardThresholds(t *testing.T) {
	for _, c := range []struct {
		name  string
		life  float64
		mons  int
		moves bool
	}{
		{"surrounded and healthy", 90, 5, false},
		{"surrounded below 65 percent", 60, 5, true},
		{"three around and hurt: under the pack size", 60, 3, false},
		{"one attacker at 40 percent", 40, 1, true},
		{"one attacker at 50 percent", 50, 1, false},
	} {
		h, g := newHandGame(t)
		g.p.LeftSkill, g.aura = 106, 102
		h.s.life = c.life
		for i := 0; i < c.mons; i++ {
			h.s.addMon(uint32(10+i), clsFallen, 101+float64(i%3), 101+float64(i/3), 400, 0, 0)
		}
		snap, _ := g.snap()
		moved, err := h.f.meleeGuard(context.Background(), snap, ProfileFor(d2.Paladin, "").Melee, newFightState())
		if err != nil || moved != c.moves {
			t.Errorf("%s: moved=%v err=%v", c.name, moved, err)
		}
	}
}

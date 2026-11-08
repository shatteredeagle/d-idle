package combat

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// d2dClick models the d2d channel in lockstep (palmelee, V live): a bare left click never lands on the monster, only
// a unit-addressed Swing does.
type d2dClick struct {
	*handGame
	swings []int
}

func (g *d2dClick) Click(b input.Button, x, y int, m ...input.Mod) error {
	if b == input.Left && !g.open && !(x == LeftPopup.BtnX && y == LeftPopup.BtnY) {
		g.clicks = append(g.clicks, input.Screen{X: x, Y: y})
		return nil // swallowed
	}
	return g.handGame.Click(b, x, y, m...)
}

func (g *d2dClick) Swing(skill int, at input.Point) error {
	g.swings = append(g.swings, skill)
	snap, _ := g.snap()
	s := screenOf(snap, at)
	return g.sim.Click(input.Left, s.X, s.Y)
}

// A melee class on a backend whose pointer click does not attack must still kill: the fighter swings by unit id.
// Paladin (profile paladin) and Barbarian (a second build) at clvl 1 with only the basic attack.
func TestMeleeClassSwingsByUnitWhenClickIsInert(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Barbarian} {
		h, g := newHandGame(t)
		g.p.Class = class
		g.p.Level = 1
		g.p.Skills = []d2.Skill{{ID: 0, Level: 1}}
		g.p.LeftSkill, g.p.RightSkill = 0, 0
		g.aura = 0
		h.s.addMon(10, clsFallen, 104, 100, 30, 1, 5)
		h.s.meleeDmg = 40
		d := &d2dClick{handGame: g}
		h.f.e.In = d
		res, err := h.f.Clear(context.Background(), ClearOpts{})
		if err != nil || res.Kills != 1 {
			t.Fatalf("%v: %v %v (swings %v clicks %v)", class, res, err, d.swings, g.clicks)
		}
		if len(d.swings) == 0 || d.swings[0] != 0 {
			t.Fatalf("%v: swings %v", class, d.swings)
		}
	}
}

// A right-hand caster never swings: the Sorceress still casts with the bound key.
func TestCasterDoesNotSwing(t *testing.T) {
	h := newHarness(t)
	d := &d2dSwingRec{Controller: h.f.e.In}
	h.f.e.In = d
	_, _ = h.f.Clear(context.Background(), ClearOpts{})
	if d.swings != 0 {
		t.Fatalf("a Fire Bolt caster swung %d times", d.swings)
	}
}

type d2dSwingRec struct {
	input.Controller
	swings int
}

func (d *d2dSwingRec) Swing(int, input.Point) error { d.swings++; return nil }

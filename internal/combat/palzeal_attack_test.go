package combat

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/input"
)

// palzeal (live 2026-11-27, Doofer): on the d2d backend the main attack follows the profile band by skill id, with no
// hotkey and no left-skill selection: plain Attack before the band's skill is learned, the band's skill from then.
func TestD2dMainAttackFollowsBandByID(t *testing.T) {
	h, g := newHandGame(t)
	d := &d2dClick{handGame: g}
	h.f.e.In = d
	h.f.SetListRows(nil)
	h.f.SetHotkeys(nil)
	g.p.LeftSkill, g.p.RightSkill = 0, 98
	g.p.Level = 11
	g.p.Skills = []d2.Skill{{ID: 0, Level: 1}, {ID: 98, Level: 8}, {ID: 107, Level: 1}} // Attack, Might, Sacrifice
	if a := h.f.attackSkills(g.p); len(a) == 0 || a[0].Name != "Attack" {
		t.Fatalf("clvl 11 without Zeal: %v", a)
	}
	g.p.Level = 12
	g.p.Skills = append(g.p.Skills, d2.Skill{ID: 106, Level: 1})
	a := h.f.attackSkills(g.p)
	if len(a) == 0 || a[0].Name != "Zeal" {
		t.Fatalf("clvl 12 with Zeal: %v", a)
	}
	snap, _ := g.snap()
	if err := h.f.cast(snap, a[0], input.Point{X: 104, Y: 100}); err != nil || len(d.swings) != 1 || d.swings[0] != 106 {
		t.Fatalf("Zeal not swung by id: %v %v", err, d.swings)
	}
	if kv := h.f.lastSkillKV(); len(kv) < 2 || kv[1] != "Zeal" {
		t.Errorf("kill log skill %v", kv)
	}
}

// The second build: a Necromancer's Teeth is a right-hand skill. A Teeth point spent mid-game has no hotkey until the
// next binding; a backend that casts by id uses it at once, a hotkey-only backend still needs the key.
func TestD2dRightHandMainAttackCastByID(t *testing.T) {
	const idTeeth = 67
	h, w := newNecro(t)
	cat := necroCat()
	cat.Skills[0] = &data.Skill{ID: 0, Key: "Attack", Name: "Attack", Range: "h2h", LeftSkill: true}
	cat.Skills[idTeeth] = &data.Skill{ID: idTeeth, Key: "Teeth", Name: "Teeth", Class: "nec", Range: "rng", Mana: 3, ManaShift: 8, MinMana: 3}
	h.f.SetCatalog(cat)
	h.f.SetHotkeys(nil)
	p := w.snap.Player
	p.Level, p.RightSkill = 12, 0
	p.Skills = []d2.Skill{{ID: 0, Level: 1}, {ID: idTeeth, Level: 4}}
	if a := h.f.attackSkills(p); len(a) > 0 && a[0].Name == "Teeth" {
		t.Fatalf("hotkey-only backend picked an unbound right skill: %v", a)
	}
	c := &idCast{nworld: w}
	h.f.e.In = c
	a := h.f.attackSkills(p)
	if len(a) == 0 || a[0].Name != "Teeth" {
		t.Fatalf("cast-by-id backend: %v", a)
	}
	snap, _ := h.f.snapFn()
	if err := h.f.cast(snap, a[0], input.Point{X: 110, Y: 100}); err != nil || len(c.ids) != 1 || c.ids[0] != idTeeth {
		t.Fatalf("Teeth not cast by id: %v %v", err, c.ids)
	}
}

type idCast struct {
	*nworld
	ids []int
}

func (c *idCast) CastSkill(id int, _ input.Point) error { c.ids = append(c.ids, id); return nil }

package combat

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
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

package bot

import (
	"testing"

	"d2lod-bot/internal/d2"
)

// One drink inside the Den must not turn the farm target around (docs/denshuttle.md): the entry needs two belt
// potions, staying needs one. Both classes, same data.
func TestDenKitHasAGapBetweenEntryAndStay(t *testing.T) {
	for _, cl := range []d2.Class{d2.Sorceress, d2.Paladin} {
		for _, c := range []struct {
			area, potions int
			short         bool
		}{
			{2, 1, true},              // outside with one potion: not enough to go in
			{2, 2, false},             // outside with two: may enter
			{areaDenOfEvil, 1, false}, // inside after a drink: stays
			{areaDenOfEvil, 0, true},  // inside with nothing: leaves
			{areaDenOfEvil, 2, false},
		} {
			g, _ := farmRig(t, 5, cl)
			ids := []uint32{}
			for i := 0; i < c.potions; i++ {
				ids = append(ids, uint32(701+i))
			}
			g.w.belt(ids...)
			g.w.set(func(w *world) { w.area = c.area })
			g.r.poll()
			if got := g.r.denKitShort(g.r.latest()); got != c.short {
				t.Errorf("%v area %d potions %d: short %v, want %v", cl, c.area, c.potions, got, c.short)
			}
		}
	}
}

// The order's target stays in the Den after the drink and only moves out when the belt is empty.
func TestGoldFarmStaysInTheDenAfterOneDrink(t *testing.T) {
	for _, cl := range []d2.Class{d2.Sorceress, d2.Paladin} {
		g, c := farmRig(t, 5, cl)
		g.w.belt(701, 702)
		g.w.set(func(w *world) { w.area = areaDenOfEvil })
		g.r.poll()
		c.Start()
		if a, ok, _ := g.r.goldFarmTarget(g.r.latest()); !ok || a != areaDenOfEvil {
			t.Fatalf("%v with two potions: area %d ok %v, want the Den", cl, a, ok)
		}
		g.w.belt(701)
		g.r.poll()
		if a, ok, _ := g.r.goldFarmTarget(g.r.latest()); !ok || a != areaDenOfEvil {
			t.Errorf("%v after one drink: area %d ok %v, want the Den", cl, a, ok)
		}
		g.w.belt()
		g.r.poll()
		if a, ok, _ := g.r.goldFarmTarget(g.r.latest()); !ok || a == areaDenOfEvil {
			t.Errorf("%v with an empty belt: area %d ok %v, want a leave", cl, a, ok)
		}
	}
}

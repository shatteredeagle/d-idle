package combat

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/town/panel"
)

// packPotionGame fakes the inventory panel with one healing potion in the pack (id 88, cell 4,1): a right click on its
// cell while the panel is open drinks it (it leaves the pack, life is restored).
type packPotionGame struct {
	*sim
	open       bool
	gone       bool
	keys       []string
	clicks     []input.Screen
	cell       input.Screen
	noPack     bool
	keepPortal bool
	ignores    int
}

func (g *packPotionGame) Key(name string) error {
	g.keys = append(g.keys, name)
	if name == "i" {
		g.open = !g.open
	}
	return nil
}
func (g *packPotionGame) CloseAll() error          { g.open = false; return nil }
func (g *packPotionGame) MoveMouse(x, y int) error { return nil }
func (g *packPotionGame) Click(b input.Button, x, y int, m ...input.Mod) error {
	at := input.Screen{X: x, Y: y}
	g.clicks = append(g.clicks, at)
	if g.open && b == input.Right && at == g.cell {
		if g.ignores > 0 {
			g.ignores--
			return nil
		}
		g.gone = true
		g.sim.life = g.sim.maxL
	}
	return nil
}
func (g *packPotionGame) snap() (*d2.Snapshot, error) {
	s, _ := g.sim.snapshot()
	if g.open {
		s.UI[d2.UIInventory] = 1
	}
	s.Screen = d2.Point{X: 800, Y: 600}
	if !g.keepPortal { // no portal source unless a test asks for one
		var keep []d2.Unit
		for _, u := range s.Inventory.Items {
			if u.Class != itTSC && u.Class != itTome {
				keep = append(keep, u)
			}
		}
		s.Inventory.Items = keep
	}
	if !g.gone && !g.noPack {
		u := d2.Unit{Type: d2.TypeItem, Class: itHP1, ID: 88, Item: &d2.Item{Where: d2.ContInventory, GX: 4, GY: 1}}
		s.Inventory.Items = append(s.Inventory.Items, u)
	}
	return s, nil
}

func newPackHarness(t *testing.T, class d2.Class) (*harness, *packPotionGame) {
	h := newHarness(t)
	h.s.class = class
	g := &packPotionGame{sim: h.s}
	grid := panel.Grid{Name: "inventory", Cols: 10, Rows: 4, Left: 497, Top: 375, CellW: 29, CellH: 29}
	g.cell = input.Screen{X: grid.Left + 4*29 + 14, Y: grid.Top + 29 + 14}
	h.f.e.In = g
	h.f.snapFn = g.snap
	h.f.gridFn = func(*d2.Snapshot) (panel.Grid, error) { return grid, nil }
	h.f.Portal = nil
	h.s.life = 30 // 30 percent: under combat.drinkRejuvPct (35), over the chicken line (25)
	h.s.addMon(10, clsFallen, 104, 100, 500, 0, 0)
	return h, g
}

// BELTFIX layer 3, the sandbox state: no healing potion in the belt (mana only), one in the pack, life 30 percent in a fight.
// Safety asks for a healing drink and the Fighter drinks it from the pack. Amazon (the harness default) and Sorceress.
func TestPackDrinkWhenBeltHasNoHealing(t *testing.T) {
	for _, class := range []d2.Class{d2.Amazon, d2.Sorceress} {
		h, g := newPackHarness(t, class)
		g.sim.stock(itMP1, 0, 6) // six mana potions in the belt
		snap, _ := g.snap()
		// no scroll in the harness pack: no portal, so the ladder has nothing else to offer
		a := h.f.Safety(snap)
		if a != DrinkLife {
			t.Fatalf("class %v: safety %v, want DrinkLife", class, a)
		}
		if err := h.f.Act(t.Context(), a); err != nil {
			t.Fatalf("class %v: %v", class, err)
		}
		if !g.gone || len(g.keys) == 0 || g.keys[0] != "i" || len(g.clicks) != 1 || g.open {
			t.Errorf("class %v: gone %v keys %v clicks %v open %v", class, g.gone, g.keys, g.clicks, g.open)
		}
	}
}

// A healing potion in the belt is drunk from the belt; the pack is not touched.
func TestPackDrinkNotUsedWhenBeltHasHealing(t *testing.T) {
	h, g := newPackHarness(t, d2.Amazon)
	g.sim.stock(itHP1, 0, 1)
	snap, _ := g.snap()
	if a := h.f.Safety(snap); a != DrinkLife {
		t.Fatalf("safety %v", a)
	}
	if err := h.f.Act(t.Context(), DrinkLife); err != nil {
		t.Fatal(err)
	}
	if g.gone || len(g.clicks) != 0 {
		t.Errorf("pack touched: gone %v clicks %v", g.gone, g.clicks)
	}
}

// Above the rejuvenation line nothing is drunk from the pack, and below the chicken line the character leaves.
func TestPackDrinkLines(t *testing.T) {
	h, g := newPackHarness(t, d2.Amazon)
	g.sim.life = 50
	snap, _ := g.snap()
	if a := h.f.Safety(snap); a == DrinkLife {
		t.Errorf("50 percent: pack drink asked (%v)", a)
	}
	g.sim.life = 20
	snap, _ = g.snap()
	if a := h.f.Safety(snap); a != Chicken {
		t.Errorf("20 percent: %v, want Chicken", a)
	}
}

// A click that does nothing fails with an error and backs off, so the next tick does not reopen the panel.
func TestPackDrinkFailureBacksOff(t *testing.T) {
	h, g := newPackHarness(t, d2.Amazon)
	g.ignores = 99
	snap, _ := g.snap()
	if a := h.f.Safety(snap); a != DrinkLife {
		t.Fatalf("safety %v", a)
	}
	if err := h.f.Act(t.Context(), DrinkLife); err == nil {
		t.Fatal("want an error from an ignored click")
	}
	if a := h.f.Safety(snap); a == DrinkLife {
		t.Error("no backoff after a failed pack drink")
	}
}

// With a scroll in the pack the portal comes first, and the pack potion is drunk before the cast (drinkIfDue).
func TestPackDrinkBeforePortalCast(t *testing.T) {
	h, g := newPackHarness(t, d2.Amazon)
	h.f.Portal = nil
	g.keepPortal = true
	scroll := d2.Unit{Type: d2.TypeItem, Class: itTSC, ID: 77, Item: &d2.Item{Where: d2.ContInventory, GX: 6, GY: 1}}
	inner := g.snap
	h.f.snapFn = func() (*d2.Snapshot, error) {
		s, _ := inner()
		s.Inventory.Items = append(s.Inventory.Items, scroll)
		return s, nil
	}
	_ = h.f.openPortal(t.Context())
	if !g.gone {
		t.Errorf("pack potion not drunk before the cast: clicks %v", g.clicks)
	}
}

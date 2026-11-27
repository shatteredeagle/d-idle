package bot

import (
	"testing"

	"d2lod-bot/internal/d2"
)

// unitLoot judges a pack item by the gold the character has, like the real looter in poor mode.
type unitLoot struct{ fakeLoot }

func (unitLoot) EvaluateUnit(u d2.Unit, clvl, gold int) Verdict {
	if gold < 300 {
		return VerdictSell
	}
	return VerdictDrop
}

func TestSellableUsesUnitEvaluatorWhenAvailable(t *testing.T) {
	g := newRig(t)
	g.r.d.Loot = unitLoot{fakeLoot{w: g.w}}
	mk := func(gold int) *d2.Snapshot {
		return &d2.Snapshot{Player: &d2.Player{Level: 8, Gold: gold}, Inventory: &d2.Inventory{Items: []d2.Unit{
			{Item: &d2.Item{Where: d2.ContInventory}}, {Item: &d2.Item{Where: d2.ContInventory}}, {Item: &d2.Item{Where: d2.ContEquipment}},
		}}}
	}
	if n := g.r.sellable(mk(0)); n != 2 {
		t.Fatalf("poor: %d sellable, want 2 (equipped items never count)", n)
	}
	if n := g.r.sellable(mk(1000)); n != 0 {
		t.Fatalf("rich: %d sellable, want 0", n)
	}
	g.r.d.Loot = fakeLoot{w: g.w}
	if n := g.r.sellable(mk(0)); n != 0 {
		t.Fatalf("plain evaluator keeps behaviour: %d", n)
	}
}

// sellTown answers the town's own sell count, like town.Town.SellableCount.
type sellTown struct {
	Townsman
	n int
}

func (s sellTown) SellableCount(*d2.Snapshot) int { return s.n }

// Questy bed 2026-11-26: the poverty advice counted 4 sellable items, the town's sell chore skipped them (starting items, the
// scroll surplus rule) and sold 0, a sell trip every few minutes for nothing. The bot asks the town for the count. Two classes.
func TestSellableAsksTheTownForTheChoresCount(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		g := newRig(t)
		g.r.d.Loot = unitLoot{fakeLoot{w: g.w}}
		s := &d2.Snapshot{Player: &d2.Player{Class: class, Level: 11, Gold: 24}, Inventory: &d2.Inventory{Items: []d2.Unit{
			{Item: &d2.Item{Where: d2.ContInventory}}, {Item: &d2.Item{Where: d2.ContInventory}}, {Item: &d2.Item{Where: d2.ContInventory}}, {Item: &d2.Item{Where: d2.ContInventory}},
		}}}
		g.r.d.Town = sellTown{g.r.d.Town, 0}
		if n := g.r.sellable(s); n != 0 {
			t.Errorf("%v: the loot verdict says sell but the chore sells nothing: advice counted %d", class, n)
		}
		g.r.d.Town = sellTown{g.r.d.Town, 3}
		if n := g.r.sellable(s); n != 3 {
			t.Errorf("%v: the town's count of 3 was not used: %d", class, n)
		}
	}
}

// Questy 2026-11-26 (28 minutes of frozen xp): the hero stood in the Rogue Encampment with 7 sellable items and 19 gold, and no
// town visit planned the sell chore. A poor hero in town with sellable items sells first. Two classes; with nothing sellable, or
// after a trip the vendor left items unsold, no sell is planned.
func TestPoorHeroInTownSellsFirst(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		for _, tc := range []struct {
			sellable int
			left     int
			want     bool
		}{{7, 0, true}, {0, 0, false}, {7, 7, false}} {
			g := newRig(t)
			g.r.d.Town = sellTown{g.r.d.Town, tc.sellable}
			g.w.set(func(w *world) { w.class, w.level, w.area, w.gold0 = class, 13, 1, 19 })
			g.w.belt() // no potions: poor
			g.r.poll()
			g.r.ob.poor = true
			g.r.setSellLeft(tc.left)
			plan, _, _ := g.r.choresDue(g.r.latest())
			if plan.Sell != tc.want {
				t.Errorf("%v sellable %d left %d: sell planned %v, want %v (%+v)", class, tc.sellable, tc.left, plan.Sell, tc.want, plan)
			}
		}
	}
}

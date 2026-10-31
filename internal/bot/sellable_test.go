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

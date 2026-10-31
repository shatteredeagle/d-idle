package bot

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
)

// gateRig is a level 20 character in the Rogue Encampment with a town portal, an empty belt and the given gold.
func gateRig(t *testing.T, retreat, minPct, gold int) (*rig, *trip, *d2.Snapshot) {
	t.Helper()
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) {
		tu.Resource.RetreatBelt, tu.Resource.MinFieldBeltPct = retreat, minPct
	})
	g.w.set(func(w *world) {
		w.level, w.area, w.state = 20, 1, d2.StateInGame
		w.items = append(w.items, d2.Unit{ID: 900, Type: d2.TypeItem, Class: 5, Item: &d2.Item{Where: d2.ContInventory}})
	})
	g.w.gold(gold)
	g.w.belt() // the fake belt has no rows, so the plan wants 2 potions and none is thin
	s, _ := g.w.snapshot()
	return g, &trip{returnTo: 4, stage: 3}, s
}

// Empty belt, gold to pay: shop again, at most gateRestocks times, then leave with a log line.
func TestFieldGateRestocksThenLeaves(t *testing.T) {
	g, tr, s := gateRig(t, 1, 50, 5000)
	for i := 1; i <= gateRestocks; i++ {
		if !g.r.gateField(s, tr) || tr.stage != 1 || tr.restocks != i || !tr.plan.Potions {
			t.Fatalf("attempt %d: stage %d restocks %d plan %+v", i, tr.stage, tr.restocks, tr.plan)
		}
		tr.stage = 3
	}
	if g.r.gateField(s, tr) || tr.returnTo != 4 {
		t.Fatalf("third time must proceed to the field, returnTo %d", tr.returnTo)
	}
}

// No gold: no shopping, the first field of the act instead, and no loop on repeated calls.
func TestFieldGatePoorGoesToFirstField(t *testing.T) {
	g, tr, s := gateRig(t, 1, 50, 0)
	if g.r.gateField(s, tr) {
		t.Fatal("poor character was sent shopping")
	}
	if tr.returnTo != 2 || tr.restocks != 0 {
		t.Fatalf("returnTo %d restocks %d, want area 2 and none", tr.returnTo, tr.restocks)
	}
	if g.r.gateField(s, tr) || tr.returnTo != 2 {
		t.Fatal("second call changed something")
	}
	if a, ok := g.r.target0(s, control{spec: goalSpec{kind: goals.ReachLevel}}); !ok || a != 2 {
		t.Fatalf("target0 = %d %v, want 2", a, ok)
	}
}

// Knobs off: the gate does nothing at all.
func TestFieldGateOffIsUnchanged(t *testing.T) {
	g, tr, s := gateRig(t, 0, 0, 1) // 1 gold: no gold at all would be the gear gate's destitute case
	before := *tr
	if g.r.gateField(s, tr) || *tr != before {
		t.Fatalf("gate changed the trip: %+v", tr)
	}
}

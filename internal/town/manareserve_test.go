package town_test

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town"
)

// manaVisit runs a potion visit with the default kit prices and returns the mana potions bought.
func manaVisit(t *testing.T, gold int) (int, town.ChoreReport) {
	t.Helper()
	r := audit(t, gold, 12, false)
	r.Env.Tune.Resource.KitPotionGold, r.Env.Tune.Resource.KitTPGold = 30, 100
	r.W.Add("hp1", d2.ContBelt, 0, 0)
	r.W.Add("hp1", d2.ContBelt, 1, 0)
	r.W.Add("tsc", d2.ContInventory, 9, 3)
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true})
	if err != nil {
		t.Fatal(err)
	}
	return boughtLike(rep, "Mana"), rep
}

// Gate 1 20261102: mana potions leave the gold of one healing potion and one scroll (130); above that gold they are bought.
func TestManaLeavesKitReserve(t *testing.T) {
	rich, repR := manaVisit(t, 600)
	if rich < 2 {
		t.Fatalf("600 gold bought %v, want the mana fill", repR.Bought)
	}
	poor, repP := manaVisit(t, 150)
	if poor != 0 {
		t.Fatalf("150 gold bought mana: %v", repP.Bought)
	}
}

// codefix F3: with the scroll in the pack and kitMinHP healing potions in the belt the reserve is zero: 180 gold may become mana.
func TestManaReserveOnlyWhileKitIncomplete(t *testing.T) {
	r := audit(t, 180, 12, false)
	r.Env.Tune.Resource.KitPotionGold, r.Env.Tune.Resource.KitTPGold = 30, 100
	r.Env.Tune.Resource.KitMinHP = 2
	r.W.Add("hp1", d2.ContBelt, 0, 0)
	r.W.Add("hp1", d2.ContBelt, 1, 0)
	r.W.Add("tsc", d2.ContInventory, 9, 3)
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true})
	if err != nil {
		t.Fatal(err)
	}
	if n := boughtLike(rep, "Mana"); n < 1 {
		t.Errorf("complete kit, 180 gold bought %v, want a mana potion (the old reserve refused it)", rep.Bought)
	}
}

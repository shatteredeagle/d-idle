package town_test

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/gear/geartest"
	"d2lod-bot/internal/town"
)

// NAVEXIT21 (Gate 1 ms1 seed 17): clvl 9, empty tome, no potions, 284 gold. The first portal scroll is bought before
// the belt is filled up to its floor; with the tome in the pack buyFirstTP used to return at once ("has a tome").
func TestEmptyTomeGetsAScrollBeforeThePotionsEatTheGold(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	r.W.Player.Gold = 284
	r.W.Player.Level = 9
	tome := r.W.Add("tbk", d2.ContInventory, 9, 3)
	r.W.Stats[tome.Addr] = d2.Stats{70: 0}
	rep, err := r.T.Chores(context.Background(), town.FullPlan())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Bought["Scroll of Town Portal"] == 0 {
		t.Fatalf("no scroll bought for the empty tome: %+v gold after %d", rep.Bought, rep.GoldAfter)
	}
}

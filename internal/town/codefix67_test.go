package town_test

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town"
)

// codefix67 F6: a purse that pays the minimal kit (scroll plus HPFirst potions) needs no stash round, in the default
// build and in a Barbarian build with a larger kit.
func TestNoWithdrawWhenPurseAlreadyPaysTheKit(t *testing.T) {
	for _, tc := range []struct {
		name  string
		class d2.Class
		hp    int
	}{{"default", d2.Sorceress, 0}, {"barbarian", d2.Barbarian, 3}} {
		r, c := withdrawRig(t, 600, 1418, 11, false) // 600 < keep 850, 600 >= 100 + 3 x 30
		r.W.Player.Class = tc.class
		if tc.hp > 0 {
			r.Env.Tune.Resource.HPFirst = tc.hp
		}
		if _, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true, Gold: true}); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		for _, m := range c.moves {
			if m < 0 {
				t.Errorf("%s: withdrawal %v although the purse pays the kit", tc.name, c.moves)
			}
		}
	}
}

// codefix67 F6: one healing count. A rejuvenation potion in the pack is a healing potion for the withdraw chore, as it
// is for the bot side.
func TestNoWithdrawWithPackRejuvenation(t *testing.T) {
	r, c := withdrawRig(t, 6, 1418, 11, false)
	r.W.Add("rvs", d2.ContInventory, 0, 0)
	if _, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true, Gold: true}); err != nil {
		t.Fatal(err)
	}
	for _, m := range c.moves {
		if m < 0 {
			t.Errorf("withdrawal %v with a rejuvenation potion in the pack", c.moves)
		}
	}
}

package town_test

import (
	"context"
	"testing"

	"d2lod-bot/internal/town"
)

// withdrawGold extends the d2dGold controller of fixbank_test.go with the withdrawal direction.
type withdrawGold struct{ *d2dGold }

func (c *withdrawGold) MoveGold(n int, deposit bool) error {
	if deposit {
		return c.d2dGold.MoveGold(n, true)
	}
	c.moves = append(c.moves, -n)
	p := &c.r.W.Player
	if n <= p.StashGold {
		p.StashGold -= n
		p.Gold += n
	}
	return nil
}

func withdrawRig(t *testing.T, purse, bank, clvl int, fullBelt bool) (*rig, *withdrawGold) {
	r := audit(t, purse, clvl, fullBelt)
	c := &d2dGold{Controller: r.In, r: r}
	w := &withdrawGold{c}
	r.Env.In = w
	r.W.Player.StashGold = bank
	return r, w
}

// BELTFIX5, Gate 1 20261103-0338 (clvl 11, minute line 332): 0 healing potions, 6 gold in the pack, 1418 in the stash.
// The stash is reachable: the pack goes back to the keep line (300 + 50 x clvl = 850) and the shop buys the potions.
func TestWithdrawForTheKitWhenNoHealingPotion(t *testing.T) {
	r, c := withdrawRig(t, 6, 1418, 11, false) // empty belt, no potion in the pack
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true, Gold: true})
	if err != nil || rep.Errors != 0 {
		t.Fatalf("%v %+v", err, rep.Steps)
	}
	if len(c.moves) != 1 || c.moves[0] != -844 {
		t.Fatalf("moves %v, want one withdrawal of 844", c.moves)
	}
	if len(rep.Bought) == 0 {
		t.Fatalf("nothing bought after the withdrawal: %+v", rep.Steps)
	}
}

// A full belt, or a pack at the keep line, or an empty stash: no withdrawal.
func TestNoWithdrawWhenPotionsOrPurseOrStash(t *testing.T) {
	for name, f := range map[string]func() (*rig, *withdrawGold){
		"healing potions in the belt": func() (*rig, *withdrawGold) { return withdrawRig(t, 6, 1418, 11, true) },
		"empty stash":                 func() (*rig, *withdrawGold) { return withdrawRig(t, 6, 0, 11, false) },
		"purse at the keep line":      func() (*rig, *withdrawGold) { return withdrawRig(t, 900, 1418, 11, false) },
	} {
		r, c := f()
		if _, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true, Gold: true}); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, m := range c.moves {
			if m < 0 {
				t.Errorf("%s: unexpected withdrawal %v", name, c.moves)
			}
		}
	}
}

// codefix F5: recovery caps the healing potions of a visit by the purse at the trip start; once the withdrawal has
// lifted the purse to the recovery kit money the cap no longer applies.
func TestRecoveryFloorLiftedByWithdrawal(t *testing.T) {
	r, c := withdrawRig(t, 6, 1418, 11, false)
	r.Env.Tune.Resource.RecoveryManaFloor = 0
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true, Gold: true, PotionFloor: 1})
	if err != nil || len(c.moves) != 1 {
		t.Fatalf("%v moves %v", err, c.moves)
	}
	if hp, mp := boughtLike(rep, "Healing"), boughtLike(rep, "Mana"); hp <= 1 || mp == 0 {
		t.Errorf("bought %v after a withdrawal to 850 gold, want more than the floor of 1 and mana", rep.Bought)
	}
	// without a stash the floor holds
	r2 := audit(t, 850, 11, false)
	r2.Env.Tune.Resource.RecoveryManaFloor = 0
	rep2, err := r2.T.Chores(context.Background(), town.ChorePlan{Potions: true, PotionFloor: 1})
	if err != nil {
		t.Fatal(err)
	}
	if hp := boughtLike(rep2, "Healing"); hp > 1 {
		t.Errorf("recovery without a withdrawal bought %v, want at most 1 healing", rep2.Bought)
	}
}

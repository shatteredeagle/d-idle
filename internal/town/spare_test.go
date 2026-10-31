package town_test

import (
	"context"
	"d2lod-bot/internal/gear/geartest"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town"
)

func countCode(r *rig, code string) int {
	n := 0
	for _, u := range r.W.Items {
		if u.Item.Where == d2.ContInventory && r.Env.Data.Items[u.Class].Code == code {
			n++
		}
	}
	return n
}

// G1 attempt 10: a pack full of spare stamina potions and identify scrolls (pickit keep, tier C) was never sold, so every
// trip fixed nothing and loot was left on the ground. A full pack is sold down to SpareKeep of each kind.
func TestFullPackOfSparesIsSoldDown(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	dressed(r.W)
	for x := 0; x < 10; x++ {
		for y := 0; y < 4; y++ {
			code := "vps"
			if y >= 2 {
				code = "isc"
			}
			r.W.Add(code, d2.ContInventory, x, y)
		}
	}
	r.Env.Tune.Town.SpareKeep = 3
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Sell: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("steps %+v", rep.Steps)
	if n := countCode(r, "vps"); n != 3 {
		t.Errorf("%d spare potions left, want SpareKeep 3", n)
	}
	// the first IDScrollTarget scrolls are needed (pickit tier S) and stay on top of the spares
	if n := countCode(r, "isc"); n < 3 || n > 3+r.Env.Tune.Town.IDScrollTarget {
		t.Errorf("%d identify scrolls left", n)
	}
}

// A pack that is not full keeps its spares (they are free to carry).
func TestSparesStayWhileThePackHasRoom(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	dressed(r.W)
	for x := 0; x < 10; x++ {
		r.W.Add("vps", d2.ContInventory, x, 0)
	}
	r.Env.Tune.Town.SpareKeep = 3
	if _, err := r.T.Chores(context.Background(), town.ChorePlan{Sell: true}); err != nil {
		t.Fatal(err)
	}
	if n := countCode(r, "vps"); n != 10 {
		t.Errorf("%d spare potions left, want all 10", n)
	}
}

// Recovery buys healing potions only up to the floor and no mana (the last gold is for the kit).
func TestPotionFloorLimitsTheBuy(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	r.Env.Tune.Resource.RecoveryManaFloor = 0
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true, PotionFloor: 2})
	if err != nil {
		t.Fatal(err)
	}
	hp, mp := 0, 0
	for name, n := range rep.Bought {
		switch {
		case name == "Minor Healing Potion" || name == "Light Healing Potion" || name == "Healing Potion":
			hp += n
		case name == "Minor Mana Potion" || name == "Light Mana Potion" || name == "Mana Potion":
			mp += n
		}
	}
	if hp != 2 || mp != 0 {
		t.Errorf("bought %v, want exactly 2 healing and no mana", rep.Bought)
	}
	full, err := newRig(t).T.Chores(context.Background(), town.ChorePlan{Potions: true})
	if err != nil {
		t.Fatal(err)
	}
	if full.Spent <= rep.Spent {
		t.Errorf("without a floor the belt plan must buy more: %d vs %d", full.Spent, rep.Spent)
	}
}

// MANABACKOFF: a recovering caster still buys a small mana reserve, never more than the floor.
func TestPotionFloorKeepsAManaReserve(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	r.Env.Tune.Resource.RecoveryManaFloor = 1
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true, PotionFloor: 2})
	if err != nil {
		t.Fatal(err)
	}
	mp := 0
	for name, n := range rep.Bought {
		if name == "Minor Mana Potion" || name == "Light Mana Potion" || name == "Mana Potion" {
			mp += n
		}
	}
	if mp != 1 {
		t.Errorf("bought %v, want exactly 1 mana potion", rep.Bought)
	}
}

// INVREJUV: a pack full of rejuvenation potions (the old pickit kept them at prio 75, protected from selling) is sold.
// The cap is 3 and Full Rejuvenation is kept over regular.
func TestFullPackOfRejuvsIsSoldDown(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	dressed(r.W)
	for x := 0; x < 10; x++ {
		for y := 0; y < 4; y++ {
			code := "rvs"
			if x%2 == 1 {
				code = "rvl"
			}
			r.W.Add(code, d2.ContInventory, x, y)
		}
	}
	r.Env.Tune.Loot.RejuvSpare = 3
	if _, err := r.T.Chores(context.Background(), town.ChorePlan{Sell: true}); err != nil {
		t.Fatal(err)
	}
	if n := countCode(r, "rvs") + countCode(r, "rvl"); n != 3 {
		t.Errorf("%d rejuvenation potions left, want the cap 3", n)
	}
	if n := countCode(r, "rvl"); n != 3 {
		t.Errorf("%d Full kept, want all 3 (regular ones go first)", n)
	}
}

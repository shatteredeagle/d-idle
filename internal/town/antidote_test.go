package town_test

import (
	"context"
	"d2lod-bot/internal/gear/geartest"
	"testing"

	"d2lod-bot/internal/town"
)

// The Andariel kit asks the shop for antidotes: the visit tops them up to the target (belt and pack) even when
// potions and scrolls are not wanted, and does not buy beyond it.
func TestShopBuysAntidotesToTarget(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	r.W.Stock(r.Akara, "hp1", "mp1", "tsc", "yps")
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Antidotes: 2})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for name, c := range rep.Bought {
		if name == "Antidote Potion" {
			n += c
		}
	}
	if n != 2 {
		t.Fatalf("bought %v, want 2 antidotes", rep.Bought)
	}
	rep, err = r.T.Chores(context.Background(), town.ChorePlan{Antidotes: 2})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Bought["Antidote Potion"] != 0 {
		t.Errorf("second visit bought more: %v", rep.Bought)
	}
}

func TestShopNoAntidotesWhenBroke(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	r.W.Stock(r.Akara, "yps")
	r.W.Player.Gold = 100 // at the reserve: optional buys never go below it
	rep, _ := r.T.Chores(context.Background(), town.ChorePlan{Antidotes: 2})
	if rep.Bought["Antidote Potion"] != 0 {
		t.Errorf("bought below the reserve: %v", rep.Bought)
	}
}

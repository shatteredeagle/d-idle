package town_test

import (
	"context"
	"d2lod-bot/internal/gear/geartest"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town"
)

func beltRig(t *testing.T, clvl int) *rig {
	r := newRig(t)
	r.W.Player.Level = clvl
	dressed(r.W)
	r.W.Stock(r.Charsi, "lbt", "mbl", "lbl", "vbl")
	return r
}

// From clvl 10 the chore buys the cheapest belt with more slots than the worn one (the sash), after the potions.
func TestBeltBuyCheapestBelt(t *testing.T) {
	geartest.Need(t)
	r := beltRig(t, 12)
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true})
	if err != nil {
		t.Fatal(err)
	}
	s, _ := r.W.Snapshot()
	b := s.Inventory.Equipped(d2.BodyBelt)
	if b == nil || b.Class != codeClass(r, "lbl") {
		t.Fatalf("worn belt %+v, report %+v", b, rep)
	}
	if rep.Bought["belt"] != 1 {
		t.Errorf("bought %v", rep.Bought)
	}
}

func TestBeltBuyOffBelowLevelAndWithKnobOff(t *testing.T) {
	geartest.Need(t)
	r := beltRig(t, 8)
	if _, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true}); err != nil {
		t.Fatal(err)
	}
	if s, _ := r.W.Snapshot(); s.Inventory.Equipped(d2.BodyBelt) != nil {
		t.Fatal("belt bought below clvl 10")
	}
	r = beltRig(t, 20)
	r.Env.Tune.Town.BeltBuyLevel = 0
	if _, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true}); err != nil {
		t.Fatal(err)
	}
	if s, _ := r.W.Snapshot(); s.Inventory.Equipped(d2.BodyBelt) != nil {
		t.Fatal("belt bought with the knob off")
	}
}

// With no gold beyond the kit reserve no belt is bought.
func TestBeltBuyKeepsKitGold(t *testing.T) {
	geartest.Need(t)
	r := beltRig(t, 12)
	r.W.Player.Gold = 150
	if _, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true}); err != nil {
		t.Fatal(err)
	}
	if s, _ := r.W.Snapshot(); s.Inventory.Equipped(d2.BodyBelt) != nil {
		t.Fatal("belt bought without gold for the kit")
	}
}

package resource

import (
	"testing"

	"d2lod-bot/internal/d2"
)

func soaky() Input {
	return Input{Class: d2.Sorceress, Level: 6, Life: 100, MaxLife: 100, Mana: 50, MaxMana: 50, Gold: 400,
		Belt: Belt{HP: 3, MP: 2, Rows: 1}, HasTP: true, Area: 2, NearTown: true, Hostiles: 3, LastTripGold: -1}
}

func TestThinBeltRetreat(t *testing.T) {
	tn := DefaultTuning()
	in := soaky()
	in.Life, in.Belt.HP = 42, 1 // the 41..44 percent drinks of the soak
	if d := PotionPolicy(in, tn); d.Drink != PotionLife {
		t.Fatalf("default (off) must drink: %+v", d)
	}
	if a := AbortFight(in, tn); a.Kind != Continue {
		t.Fatalf("default (off) must not abort: %+v", a)
	}
	tn.RetreatBelt = 1
	if d := PotionPolicy(in, tn); d.Drink != PotionNone {
		t.Fatalf("thin belt near town must leave, not drink: %+v", d)
	}
	if a := AbortFight(in, tn); a.Kind != PortalOut {
		t.Fatalf("want portal: %+v", a)
	}
	in.HasTP = false
	if a := AbortFight(in, tn); a.Kind != RetreatOut {
		t.Fatalf("no portal near town: walk home: %+v", a)
	}
	in.NearTown = false
	if d := PotionPolicy(in, tn); d.Drink != PotionLife {
		t.Fatalf("no way home: drink: %+v", d)
	}
}

func TestThinBeltRetreatBounds(t *testing.T) {
	tn := DefaultTuning()
	tn.RetreatBelt = 1
	in := soaky()
	in.Belt.HP = 1
	in.Life = 19 // the critical drinks of the soak (16, 16, 19) stay drinks
	if d := PotionPolicy(in, tn); d.Drink != PotionLife {
		t.Fatalf("critical life must drink: %+v", d)
	}
	in.Life = 30 // chicken 25 + 8 = 33
	if d := PotionPolicy(in, tn); d.Drink != PotionLife {
		t.Fatalf("near the chicken line must drink: %+v", d)
	}
	in.Life, in.Belt.HP = 42, 3
	if d := PotionPolicy(in, tn); d.Drink != PotionLife {
		t.Fatalf("full belt drinks: %+v", d)
	}
}

func TestFieldGate(t *testing.T) {
	tn := DefaultTuning()
	in := soaky()
	in.Belt.HP = 0
	if v := FieldGate(in, tn); !v.Go {
		t.Fatalf("off by default: %+v", v)
	}
	tn.MinFieldBeltPct = 50
	if v := FieldGate(in, tn); !v.Restock || v.Go {
		t.Fatalf("empty belt with gold: restock, %+v", v)
	}
	in.Gold = 20 // the 11:05 log: gold gone
	if v := FieldGate(in, tn); !v.BloodMoor || v.Go {
		t.Fatalf("poor: Blood Moor only, %+v", v)
	}
	in = soaky()
	in.Belt.HP = 1 // plan: 2 hp columns x 1 row = 2, 1 is half
	if v := FieldGate(in, tn); !v.Go {
		t.Fatalf("half the plan is enough: %+v", v)
	}
	in.HasTP = false
	if v := FieldGate(in, tn); v.Go || !v.Restock {
		t.Fatalf("no portal and gold: shop: %+v", v)
	}
}

// The kit is bought before anything else: with exactly the kit shortfall in gold above the reserve, PlanBuys buys it all
// even though the potion share (80 percent) alone would not cover it.
func TestPlanBuysKeepsKit(t *testing.T) {
	tn := DefaultTuning()
	in := soaky()
	in.Belt = Belt{Rows: 4}
	in.Gold = tn.ReserveGold + KitShortfall(true, 0, 0, tn)
	b := PlanBuys(in, tn, DefaultPrices())
	if b.HP < tn.HPFirst || b.MP < tn.MPFirst {
		t.Fatalf("kit not bought: %+v (gold %d)", b, in.Gold)
	}
}

func TestTripBudget(t *testing.T) {
	tn := DefaultTuning()
	tn.RetreatBelt = 1
	in := soaky()
	in.Life, in.Belt.HP = 42, 1
	if a := AbortFight(in, tn); a.Kind != PortalOut {
		t.Fatalf("within budget: portal, %+v", a)
	}
	in.TripsRecent = tn.TripBudget
	if a := AbortFight(in, tn); a.Kind != Continue {
		t.Fatalf("budget spent: fight on, %+v", a)
	}
	if d := PotionPolicy(in, tn); d.Drink != PotionLife {
		t.Fatalf("budget spent: drink the last potion, %+v", d)
	}
	in.TripsRecent = 0
	in.Gold = tn.TripGoldFloor - 1
	if a := AbortFight(in, tn); a.Kind != Continue {
		t.Fatalf("under the gold floor: fight on, %+v", a)
	}
	if TripBudgetSpent(in, tn) == "" {
		t.Fatal("reason must be set")
	}
	in.Life = 19 // critical life still escapes through the other rules, never via the thin-belt path
	in.Belt.HP = 0
	if a := AbortFight(in, tn); a.Kind == Continue {
		t.Fatalf("no potion at critical life must still abort: %+v", a)
	}
}

func TestThinBeltRetreatCountsPack(t *testing.T) {
	tn := DefaultTuning()
	tn.RetreatBelt = 1
	in := soaky()
	in.Life, in.Belt.HP = 42, 1
	in.Belt.PackHP = 4 // belt 1+2 used of 4 slots: one free slot only
	if d := PotionPolicy(in, tn); d.Drink != PotionLife {
		t.Fatalf("a spare in the pack must keep the bot fighting: %+v", d)
	}
	if a := AbortFight(in, tn); a.Kind != Continue {
		t.Fatalf("no abort with a pack spare: %+v", a)
	}
	in.Belt.Slots = 3 // no free slot: the pack spares cannot be moved
	if d := PotionPolicy(in, tn); d.Drink != PotionNone {
		t.Fatalf("a full belt cannot take the spares, retreat: %+v", d)
	}
	in.Belt.Slots = 0
	tn.RetreatCountPack = 0
	if d := PotionPolicy(in, tn); d.Drink != PotionNone {
		t.Fatalf("knob off: belt only: %+v", d)
	}
}

func TestManaPaysCastsKnob(t *testing.T) {
	tn := DefaultTuning()
	in := soaky()
	in.Mana, in.Belt.MP = 10, 2 // 20 percent of 50, cheapest cast 3: 3 casts left
	in.Skills = []Skill{{Name: "Bolt", Cost: 3, Ready: true}}
	if d := PotionPolicy(in, tn); d.Drink != PotionNone {
		t.Fatalf("default: mana still pays for casts: %+v", d)
	}
	tn.ManaPaysCasts = 0
	if d := PotionPolicy(in, tn); d.Drink != PotionMana {
		t.Fatalf("knob 0 must drink at the line: %+v", d)
	}
}

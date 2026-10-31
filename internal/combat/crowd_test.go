package combat

import (
	"context"
	"testing"

	"d2lod-bot/internal/env"
)

func crowdIn(life, crowd, adj int, mut func(*SafetyInput)) SafetyInput {
	full := Belt{}
	full.Cols[0] = BeltColumn{PotionHealing, 3, 4, false}
	in := SafetyInput{Life: life, Max: 100, Mana: 100, MMax: 100, Belt: full, InFight: true, HasPortal: true, HealReady: true, ManaReady: true,
		RejuvReady: true, Crowd: crowd, Adjacent: adj, CanRetreat: true, Tune: env.DefaultCombatTuning()}
	if mut != nil {
		mut(&in)
	}
	return in
}

func TestCrowdDecide(t *testing.T) {
	for _, c := range []struct {
		name string
		in   SafetyInput
		want SafetyAction
	}{
		{"ordinary fight 60% is left to the normal rules", crowdIn(60, 3, 2, nil), SafetyNone},
		{"ordinary fight 54% drinks as before", crowdIn(54, 3, 2, nil), DrinkLife},
		{"7 hostiles is not a crowd", crowdIn(68, 7, 2, nil), SafetyNone},
		{"crowd 70% portal when not surrounded", crowdIn(70, 8, 2, nil), TownPortal},
		{"crowd 71% fights on", crowdIn(71, 8, 2, nil), SafetyNone},
		{"crowd 62% surrounded drinks (cast would fail)", crowdIn(62, 12, 3, nil), DrinkLife},
		{"crowd 68% surrounded steps away", crowdIn(68, 12, 3, nil), CrowdRetreat},
		{"crowd 68% surrounded cannot step: fights", crowdIn(68, 12, 3, func(i *SafetyInput) { i.CanRetreat = false }), SafetyNone},
		{"crowd 60% no scroll drinks at 65", crowdIn(60, 9, 1, func(i *SafetyInput) { i.HasPortal = false }), DrinkLife},
		{"crowd 64% no scroll no potion", crowdIn(64, 9, 1, func(i *SafetyInput) { i.HasPortal = false; i.Belt = Belt{} }), SafetyNone},
		{"abandon on a walk at full life", crowdIn(100, 15, 0, func(i *SafetyInput) { i.Walking = true }), CrowdAbandon},
		{"15 hostiles in a fight at full life: fight", crowdIn(100, 15, 0, nil), SafetyNone},
		{"15 hostiles in a fight at 69%: abandon", crowdIn(69, 15, 1, nil), CrowdAbandon},
		{"abandon without a scroll steps away", crowdIn(100, 16, 0, func(i *SafetyInput) { i.Walking, i.HasPortal = true, false }), CrowdRetreat},
		{"abandon surrounded and hurt drinks first", crowdIn(60, 18, 4, func(i *SafetyInput) { i.Walking = true }), DrinkLife},
		{"chicken line still wins", crowdIn(25, 20, 4, func(i *SafetyInput) { i.Walking = true; i.Belt = Belt{} }), Chicken},
		{"rule off", crowdIn(60, 20, 4, func(i *SafetyInput) { i.Tune.CrowdN = 0; i.Walking = true }), SafetyNone},
	} {
		if got := Decide(c.in); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

// Through the Fighter: nine monsters 5 to 7 subtiles away at 60% life: portal; the same life with two monsters: nothing.
func TestSafetyCrowdFromSnapshot(t *testing.T) {
	h := newHarness(t)
	h.s.stock(itHP1, 0, 3)
	for i := 0; i < 9; i++ {
		h.s.addMon(uint32(10+i), clsFallen, 105+float64(i%3), 98+float64(i%5), 30, 5, 5)
	}
	snap, _ := h.s.snapshot()
	snap.Player.Life = 60
	if a := h.f.Safety(snap); a != TownPortal {
		t.Fatalf("crowd at 60%%: %v", a)
	}
	h.f.SetWalking(true)
	if a := h.f.Safety(snap); a != TownPortal { // 9 hostiles are below the abandon size 15
		t.Fatalf("walking crowd of 9: %v", a)
	}
	snap.Player.Life = 100
	if a := h.f.Safety(snap); a != SafetyNone {
		t.Fatalf("crowd of 9 at full life on a walk: %v", a)
	}
	few := newHarness(t)
	few.s.stock(itHP1, 0, 3)
	few.s.addMon(10, clsFallen, 106, 100, 30, 5, 5)
	few.s.addMon(11, clsFallen, 107, 101, 30, 5, 5)
	s2, _ := few.s.snapshot()
	s2.Player.Life = 60
	if a := few.f.Safety(s2); a != SafetyNone {
		t.Fatalf("two monsters at 60%%: %v", a)
	}
}

// CrowdAbandon walks away from the crowd (20+ subtiles, bounded steps) and then opens the portal.
func TestCrowdAbandonRetreatsThenPortals(t *testing.T) {
	h := newHarness(t)
	h.s.stock(itHP1, 0, 3)
	for i := 0; i < 16; i++ {
		h.s.addMon(uint32(10+i), clsFallen, 105+float64(i%4), 96+float64(i/2), 300, 0, 0)
	}
	start := h.s.px
	if err := h.f.Act(context.Background(), CrowdAbandon); err != nil {
		t.Fatal(err)
	}
	if !h.s.portalOpen {
		t.Fatal("no portal after the retreat")
	}
	if moved := start - h.s.px; moved < 8 {
		t.Fatalf("moved only %.1f subtiles away from the crowd", moved)
	}
}

func TestDrinkSkipReason(t *testing.T) {
	low := crowdIn(40, 3, 1, nil)
	if got := drinkSkipReason(low, DrinkLife); got != "" {
		t.Errorf("drinking: %q", got)
	}
	if got := drinkSkipReason(crowdIn(80, 3, 1, nil), SafetyNone); got != "" {
		t.Errorf("healthy: %q", got)
	}
	if got := drinkSkipReason(low, TownPortal); got != "chose "+TownPortal.String()+" first" {
		t.Errorf("portal: %q", got)
	}
	in := crowdIn(40, 3, 1, func(i *SafetyInput) { i.Belt = Belt{} })
	if got := drinkSkipReason(in, SafetyNone); got != "no healing potion in the belt" {
		t.Errorf("empty belt: %q", got)
	}
	in = crowdIn(40, 3, 1, func(i *SafetyInput) { i.HealReady, i.RejuvReady = false, false })
	if got := drinkSkipReason(in, SafetyNone); got != "potion cooldown" {
		t.Errorf("cooldown: %q", got)
	}
}

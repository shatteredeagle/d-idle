package combat

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
)

func withEffectRule(tu *env.Tuning) {
	tu.Combat.PotionEmergencyPct = env.DefaultCombatTuning().PotionEmergencyPct
}

func countDrinks(h *harness) int { return h.s.drinks["healing"] }

// hold keeps life at pct and, every PotionGapMs, asks Safety and acts for seconds of game time; returns the healing
// potions drunk.
func holdLife(t *testing.T, h *harness, pct float64, seconds int) int {
	t.Helper()
	for i := 0; i < seconds; i++ {
		h.s.mu.Lock()
		h.s.life = h.s.maxL * pct / 100
		h.s.mu.Unlock()
		snap, _ := h.s.snapshot()
		if a := h.f.Safety(snap); a != SafetyNone {
			if err := h.f.Act(context.Background(), a); err != nil {
				t.Fatal(err)
			}
		}
		h.s.Sleep(time.Second)
	}
	return countDrinks(h)
}

func TestHealingEffectMergesLikeTheGame(t *testing.T) {
	now := time.Unix(1000, 0)
	e := healEffect{}.drink(now, 1, d2.Sorceress)
	if e.End.Sub(now) != 192*gameTick || e.Rate < 3.9 || e.Rate > 3.95 {
		t.Fatalf("hp1 sorceress: %+v", e) // 30 life in 7.68 s = 3.9/s (fid-potions.md Q2)
	}
	pal := healEffect{}.drink(now, 1, d2.Paladin)
	if p := pal; p.Rate < 5.8 || p.Rate > 5.9 {
		t.Fatalf("hp1 paladin rate %.2f, want 5.86 (45 life)", p.Rate)
	}
	again := e.drink(now.Add(time.Second), 1, d2.Sorceress)
	if again.Rate != e.Rate && (again.Rate < e.Rate-0.001 || again.Rate > e.Rate+0.001) {
		t.Fatalf("same tier changed the rate: %.3f -> %.3f", e.Rate, again.Rate)
	}
	if !again.End.After(e.End) {
		t.Fatal("same tier did not extend the duration")
	}
	if up := e.drink(now.Add(time.Second), 5, d2.Sorceress); up.Rate <= e.Rate || up.Tier != 5 {
		t.Fatalf("higher tier: %+v", up)
	}
	if p := e.Pending(now); p < 29.9 || p > 30.1 {
		t.Fatalf("pending %.1f, want 30", p)
	}
	unk := healEffect{}.drink(now, 2, d2.Class(99))
	if unk.Rate != 0 || unk.Tier != 2 {
		t.Fatalf("unknown class must not guess a factor: %+v", unk)
	}
}

// Over a fight at a steady 45 percent life one potion is drunk per effect, not one per second, for two builds.
func TestNoSameTierRedrinkWhileWorking(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		h := newHarness(t, withEffectRule)
		h.s.class = class
		h.s.stock(itHP1, 0, 12)
		n := holdLife(t, h, 45, 10) // an hp1 works 7.68 s
		if n != 2 {
			t.Errorf("%v: %d potions in 10 s at 45%%, want 2 (one, then one after the effect ended)", class, n)
		}
	}
}

func TestEmergencyStillDrinksWhileWorking(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		h := newHarness(t, withEffectRule)
		h.s.class = class
		h.s.stock(itHP1, 0, 12)
		holdLife(t, h, 45, 1)
		if n := holdLife(t, h, 20, 3); n != 4 {
			t.Errorf("%v: %d potions, want 1 + 3 emergency drinks", class, n)
		}
	}
}

// A higher tier in the belt is allowed while a weaker potion works, and it is the lowest tier above the working one.
func TestHigherTierAllowedWhileWorking(t *testing.T) {
	h := newHarness(t, withEffectRule)
	h.s.stock(itHP1, 0, 3)
	h.s.stock(itHP3, 1, 3)
	holdLife(t, h, 45, 1) // lowest tier first: hp1 (column 1)
	if h.f.heal.Tier != 1 {
		t.Fatalf("first drink tier %d, want 1", h.f.heal.Tier)
	}
	holdLife(t, h, 45, 1)
	if h.f.heal.Tier != 3 || countDrinks(h) != 2 {
		t.Fatalf("tier %d after %d drinks, want 3 after 2", h.f.heal.Tier, countDrinks(h))
	}
}

func TestColumnAboveKeepsBeltSelection(t *testing.T) {
	var b Belt
	b.Cols[0] = BeltColumn{Kind: PotionHealing, Tier: 1, Count: 2}
	b.Cols[1] = BeltColumn{Kind: PotionHealing, Tier: 4, Count: 2}
	b.Cols[2] = BeltColumn{Kind: PotionHealing, Tier: 3, Count: 2}
	if c, ok := b.ColumnAbove(PotionHealing, 1); !ok || c != 2 {
		t.Fatalf("above 1: col %d %v, want 2", c, ok)
	}
	if _, ok := b.ColumnAbove(PotionHealing, 4); ok {
		t.Fatal("nothing above tier 4")
	}
	if c, _ := b.ColumnFor(PotionHealing, false); c != 0 {
		t.Fatalf("ColumnFor lowest: %d", c)
	}
	if c, _ := b.ColumnFor(PotionHealing, true); c != 1 {
		t.Fatalf("ColumnFor best: %d", c)
	}
}

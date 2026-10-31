package resource

import (
	"testing"

	"d2lod-bot/internal/d2"
)

func TestHellTuning(t *testing.T) {
	d := DefaultTuning()
	d.ChickenHell = 10
	if h := HellTuning(d, d2.Hell); h.ChickenHell != 25 || h.DrinkRejuvPct < 40 || h.DrinkLifePct < 60 {
		t.Fatalf("%+v", h)
	}
	if HellTuning(d, d2.Nightmare) != d {
		t.Fatal("nightmare changed")
	}
}

func TestPlanForDiff(t *testing.T) {
	n := PlanForDiff(d2.Sorceress, 50, d2.Normal)
	h := PlanForDiff(d2.Sorceress, 50, d2.Hell)
	if n != PlanFor(d2.Sorceress, 50) || h.Cols != PlanFor(d2.Sorceress, 61).Cols || h.PrefTier != n.PrefTier {
		t.Fatalf("%+v %+v", n, h)
	}
	if got := PlanForDiff(d2.Sorceress, 30, d2.Hell); got != PlanFor(d2.Sorceress, 30) {
		t.Fatal("low level unchanged")
	}
}

func TestAntidote(t *testing.T) {
	h := HellCtx{Poisoned: true, Antidotes: 2, PoisonRes: -50}
	if !AntidoteNeeded(d2.Hell, h) || AntidoteNeeded(d2.Nightmare, h) {
		t.Fatal("hell only")
	}
	h.PoisonRes = 40
	if AntidoteNeeded(d2.Hell, h) {
		t.Fatal("resistant")
	}
	if AntidotesToCarry(d2.Hell, 0) != 2 || AntidotesToCarry(d2.Hell, 60) != 0 || AntidotesToCarry(d2.Normal, 0) != 0 {
		t.Fatal("carry")
	}
	in := with(func(i *Input) { i.Diff = d2.Hell; i.Hostiles = 1 })
	if d := PotionPolicyHell(in, DefaultTuning(), HellCtx{Poisoned: true, Antidotes: 1, PoisonRes: 0}); d.Reason != "antidote" {
		t.Fatalf("%+v", d)
	}
}

func TestPotionPolicyHellManaBurn(t *testing.T) {
	in := with(func(i *Input) {
		i.Diff = d2.Hell
		i.Hostiles = 1
		i.Mana = 15 // 33%
		i.Belt.MP = 2
		i.Skills = []Skill{{Name: "Bolt", Cost: 3, Ready: true}}
	})
	// plain policy: mana still pays for casts
	if d := PotionPolicy(in, DefaultTuning()); d.Drink != PotionNone {
		t.Fatalf("%+v", d)
	}
	if d := PotionPolicyHell(in, DefaultTuning(), HellCtx{ManaBurn: true}); d.Drink != PotionMana {
		t.Fatalf("%+v", d)
	}
	if d := PotionPolicyHell(in, DefaultTuning(), HellCtx{}); d.Drink != PotionNone {
		t.Fatalf("%+v", d)
	}
}

func TestManaBurnTracker(t *testing.T) {
	var m ManaBurnTracker
	if m.Observe(0, 100, 100, 0) != ManaOK || m.Observe(1, 90, 100, 0) != ManaOK {
		t.Fatal("small loss")
	}
	if got := m.Observe(2, 60, 100, 0); got != ManaFlee {
		t.Fatalf("40%% in 2s: %v", got)
	}
	// spent mana is not foreign loss
	m.Reset()
	m.Observe(0, 100, 100, 0)
	if got := m.Observe(1, 50, 100, 50); got != ManaOK {
		t.Fatalf("own casts: %v", got)
	}
	// slow loss outside the window
	m.Reset()
	m.Observe(0, 100, 100, 0)
	if got := m.Observe(10, 60, 100, 0); got != ManaOK {
		t.Fatalf("slow: %v", got)
	}
}

func TestManaBurnChickenTwice(t *testing.T) {
	var m ManaBurnTracker
	m.Observe(0, 100, 100, 0)
	m.Observe(1, 10, 100, 0) // burned: 90% gone in 1 s
	if got := m.Observe(2, 12, 100, 0); got == ManaChicken {
		t.Fatal("once is not enough")
	}
	m.Observe(30, 90, 100, 0) // recovered
	m.Observe(31, 90, 100, 0)
	if got := m.Observe(32, 5, 100, 0); got != ManaChicken {
		t.Fatalf("%v", got)
	}
}

// Finding 3: casting down to nothing and drinking again, twice, is not a mana burn.
func TestManaBurnIgnoresOwnCasting(t *testing.T) {
	var m ManaBurnTracker
	mana := 100
	for round := 0; round < 4; round++ {
		now := float64(round * 20)
		m.Observe(now, mana, 100, 0)
		// two fast casts spend 90 mana within the window and the bot reports them
		if got := m.Observe(now+1, 10, 100, 90); got != ManaOK {
			t.Fatalf("round %d: %v", round, got)
		}
		if got := m.Observe(now+10, 70, 100, 0); got != ManaOK { // potion / regen
			t.Fatalf("round %d refill: %v", round, got)
		}
		mana = 70
	}
}

package combat

import (
	"context"
	"errors"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

func TestPackDangerNormalIgnoresResists(t *testing.T) {
	h := newHarness(t)
	ts := []Target{{Affixes: AffixFireEnchanted | AffixLightningEnchanted}}
	p := &d2.Player{Level: 30, Difficulty: d2.Normal}
	p.Resist = d2.Resists{Fire: -50, Cold: -50, Lightning: -50, Poison: -50}
	if got, want := h.f.packDanger(p, ts), CurseDanger(p, PackDanger(ts, d2.Normal, 30)); got != want {
		t.Fatalf("normal danger changed: %d vs %d", got, want)
	}
	p.Difficulty = d2.Hell
	p.Level = 80
	if got, base := h.f.packDanger(p, ts), CurseDanger(p, PackDanger(ts, d2.Hell, 80)); got < base {
		t.Fatalf("hell danger %d below the base %d", got, base)
	}
}

func TestHardTickIsInertInNormal(t *testing.T) {
	h := newHarness(t)
	snap, _ := h.s.snapshot()
	snap.Player.Mana, snap.Player.MaxMana = 1, 100
	st := newFightState()
	hs := []Target{{Affixes: AffixManaBurn, Dist: 4}}
	for i := 0; i < 6; i++ {
		snap.Player.Mana = 1 + i%2*80
		if err := h.f.hardTick(context.Background(), snap, hs, st); err != nil {
			t.Fatalf("normal tick %d: %v", i, err)
		}
	}
}

func TestHardTickChickensOnRepeatedEmptyPoolInHell(t *testing.T) {
	h := newHarness(t)
	left := false
	h.f.Leave = func(ctx context.Context) error { left = true; return nil }
	snap, _ := h.s.snapshot()
	snap.Player.Difficulty, snap.Player.Level = d2.Hell, 80
	snap.Player.MaxMana = 100
	st := newFightState()
	hs := []Target{{Affixes: AffixManaBurn, Dist: 4}}
	var err error
	for _, m := range []int{90, 10, 60, 10} {
		snap.Player.Mana = m
		if err = h.f.hardTick(context.Background(), snap, hs, st); errors.Is(err, ErrChicken) {
			break
		}
	}
	if !errors.Is(err, ErrChicken) || !left {
		t.Fatalf("err %v left %v", err, left)
	}
}

func burnTicks(h *harness, snap *d2.Snapshot, st *fightState, hs []Target, mana ...int) error {
	var err error
	for _, m := range mana {
		snap.Player.Mana = m
		if err = h.f.hardTick(context.Background(), snap, hs, st); err != nil {
			break
		}
	}
	return err
}

func TestManaBurnWatchGates(t *testing.T) {
	for _, tc := range []struct {
		name  string
		class d2.Class
		dist  float64
	}{
		{"barbarian", d2.Barbarian, 4},
		{"mana burn monster out of melee reach", d2.Sorceress, 20},
	} {
		h := newHarness(t)
		h.f.Leave = func(ctx context.Context) error { return nil }
		snap, _ := h.s.snapshot()
		snap.Player.Difficulty, snap.Player.Level, snap.Player.Class, snap.Player.MaxMana = d2.Hell, 80, tc.class, 100
		hs := []Target{{Affixes: AffixManaBurn, Dist: tc.dist}}
		if err := burnTicks(h, snap, newFightState(), hs, 90, 10, 60, 10, 90, 5); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
	}
}

func TestManaChickenCooldown(t *testing.T) {
	h := newHarness(t)
	h.f.Leave = func(ctx context.Context) error { return nil }
	snap, _ := h.s.snapshot()
	snap.Player.Difficulty, snap.Player.Level, snap.Player.Class, snap.Player.MaxMana = d2.Hell, 80, d2.Sorceress, 100
	hs := []Target{{Affixes: AffixManaBurn, Dist: 4}}
	if err := burnTicks(h, snap, newFightState(), hs, 90, 10, 60, 10); !errors.Is(err, ErrChicken) {
		t.Fatalf("first: %v", err)
	}
	// re-entry next to the same pack: no second mana chicken within the cooldown
	if err := burnTicks(h, snap, newFightState(), hs, 90, 10, 60, 10); errors.Is(err, ErrChicken) {
		t.Fatal("second mana chicken inside the cooldown")
	}
}

func TestAntidoteCooldownCoversItsDuration(t *testing.T) {
	h := newHarness(t)
	if !h.f.antidoteReady() {
		t.Fatal("never drunk: ready")
	}
	h.f.lastDrink[PotionAntidote] = h.f.now()
	// poisoned for 20 s, polled every 3 s: no second drink inside the boost window
	for el := time.Duration(0); el < 20*time.Second; el += 3 * time.Second {
		if h.f.antidoteReady() {
			t.Fatalf("ready %v after the drink", el)
		}
		h.s.Sleep(3 * time.Second)
	}
	h.s.Sleep(10 * time.Second)
	if !h.f.antidoteReady() {
		t.Fatal("not ready after the duration")
	}
}

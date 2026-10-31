package combat

import (
	"context"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"errors"
	"math"
	"testing"
	"time"
)

// A far pack is not engaged on an empty pool: the fighter rests until the cast budget is there (PreEngage).
func TestRestsBeforeAFarPack(t *testing.T) {
	h := newHarness(t)
	h.s.mana, h.s.manaRegen = 15, 1.0
	// three of them: one or two trash monsters only need the cast budget (resource.PreEngage), a pack rests to the floor
	for i := 0; i < 3; i++ {
		h.s.addMon(uint32(10+i), clsFallen, float64(100+30+i), 100, 12, 0, 0) // 30 away: in clear radius, outside the fight radius
	}
	start := h.s.Now()
	var firstCast time.Time
	h.s.onCast = func() {
		if firstCast.IsZero() {
			firstCast = h.s.t
		}
	}
	res, err := h.f.Clear(context.Background(), ClearOpts{Timeout: 3 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kills != 3 && !res.TimedOut {
		t.Fatalf("expected the monster dead: %+v", res)
	}
	if firstCast.IsZero() || firstCast.Sub(start) < 5*time.Second {
		t.Fatalf("should have rested to the cast budget first, first cast after %v", firstCast.Sub(start))
	}
}

// With no mana at all and no regeneration the basic attack finishes the fight (AttackChoice melee fallback).
func TestMeleeWhenOutOfMana(t *testing.T) {
	h := newHarness(t)
	h.s.mana, h.s.manaRegen, h.s.meleeDmg = 0, 0, 20
	h.s.addMon(10, clsFallen, 104, 100, 30, 1, 0)
	res, err := h.f.Clear(context.Background(), ClearOpts{Timeout: time.Minute})
	if err != nil || res.Kills != 1 {
		t.Fatalf("melee fallback should kill it: %+v %v", res, err)
	}
	if h.s.melee == 0 || h.s.casts != 0 {
		t.Fatalf("melee %d casts %d", h.s.melee, h.s.casts)
	}
}

// A dangerous pack with no mana and no way to attack but a portal: portal out (AbortFight).
func TestAbortFightPortalsOutWhenHelpless(t *testing.T) {
	h := newHarness(t)
	h.s.mana, h.s.manaRegen = 0, 0
	h.s.life = 45 // under the drink line, no potions
	h.s.addMon(10, clsFallen, 104, 100, 500, 0, 0)
	h.s.left_ = 7 // not "Attack": no melee fallback
	_, err := h.f.Clear(context.Background(), ClearOpts{Timeout: time.Minute})
	if !errors.Is(err, ErrTownPortal) || !h.s.portalOpen {
		t.Fatalf("expected a portal out, got %v (portal %v)", err, h.s.portalOpen)
	}
}

// No potion, no portal, life under the flee line with a monster on top of us: the fighter breaks away at once
// instead of standing in melee until the chicken line (stallfatal section 5: 51 s frozen at 62 -> 22 %).
func TestFleesWithoutPotionOrPortal(t *testing.T) {
	h := newHarness(t, func(e *env.Tuning) { e.Combat.KiteEnabled = 0 })
	h.s.noTP = true
	h.s.life = 45
	h.s.mana, h.s.manaRegen = 0, 0
	h.s.addMon(10, clsFallen, 103, 100, 500, 0, 0)
	_, _ = h.f.Clear(context.Background(), ClearOpts{Timeout: 20 * time.Second})
	if d := math.Hypot(h.s.px-103, h.s.py-100); d < 35 {
		t.Fatalf("the character stayed %.0f from the monster: it stood in melee", d)
	}
	if h.s.left {
		t.Fatal("must not chicken above the critical line")
	}
}

func TestNearTownCoversEveryAct(t *testing.T) {
	for _, a := range []int{2, 3, 8, 41, 76, 104, 110} {
		if !nearTown(a) {
			t.Errorf("area %d must be near town", a)
		}
	}
	for _, a := range []int{0, 1, 4, 40, 42, 109} {
		if nearTown(a) {
			t.Errorf("area %d must not be near town", a)
		}
	}
}

// A poor character without potions leaves a boss pack alone in a Clear, but still fights a named Kill target.
func TestPoorSkipsDangerousPackButKillsNamedTarget(t *testing.T) {
	h := newHarness(t)
	h.f.SetPoverty(true, 0)
	h.s.addMon(10, clsBoss, 130, 100, 40, 0, 0)
	res, err := h.f.Clear(context.Background(), ClearOpts{Timeout: 20 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kills != 0 || h.s.casts != 0 {
		t.Fatalf("poor clear must skip the boss: %+v casts %d", res, h.s.casts)
	}
	h2 := newHarness(t)
	h2.f.SetPoverty(true, 0)
	h2.s.addMon(10, clsBoss, 104, 100, 40, 0, 0)
	if err := h2.f.Kill(context.Background(), 10); err != nil {
		t.Fatalf("named target must still be fought: %v", err)
	}
}

func TestWeakHostiles(t *testing.T) {
	p := &d2.Player{Level: 3, Difficulty: d2.Normal}
	mk := func(lvl int, dist float64) Target {
		m := &data.Monster{}
		m.Level[d2.Normal] = lvl
		return Target{Mon: m, Dist: dist, Unit: d2.Unit{Monster: &d2.MonsterInfo{Flags: 2}}} // 2 = normal
	}
	if !weakHostiles(p, []Target{mk(4, 10), mk(2, 5)}, 30) {
		t.Error("levels <= clvl+1 are weak")
	}
	if weakHostiles(p, []Target{mk(4, 10), mk(5, 5)}, 30) {
		t.Error("a level 5 monster is not weak at clvl 3")
	}
	if !weakHostiles(p, []Target{mk(4, 10), mk(9, 99)}, 30) {
		t.Error("a far monster is outside the radius")
	}
	champ := mk(2, 5)
	champ.Unit.Monster.Flags = 4
	unknown := mk(2, 5)
	unknown.Unit.Monster.Flags = 0
	elite, affixed := mk(2, 5), mk(2, 5)
	elite.Elite, affixed.Affixes = true, 1
	for name, tg := range map[string]Target{"champion": champ, "unknown kind": unknown, "elite": elite, "affixed": affixed} {
		if weakHostiles(p, []Target{mk(2, 5), tg}, 30) {
			t.Errorf("%s must not count as weak", name)
		}
	}
	big := make([]Target, weakPackMax+1)
	for i := range big {
		big[i] = mk(2, 5)
	}
	if weakHostiles(p, big, 30) {
		t.Error("a big pack is not weak")
	}
	if weakHostiles(p, []Target{{Dist: 5}}, 30) || weakHostiles(p, nil, 30) {
		t.Error("unknown class or nobody is not weak")
	}
}

// The break-away is bounded: after fleeMax of unbroken fleeing the fighter chickens instead of kiting forever.
func TestFleeEndsInChicken(t *testing.T) {
	h := newHarness(t, func(e *env.Tuning) { e.Combat.KiteEnabled = 0 })
	h.s.noTP = true
	h.s.life = 45
	h.s.mana, h.s.manaRegen = 0, 0
	h.s.addMon(10, clsFallen, 103, 100, 500, 0, 0)
	st := newFightState()
	ctx := context.Background()
	for i := 0; i < 100 && !h.s.left; i++ {
		h.s.px, h.s.py = 100, 100 // the hostile keeps up
		snap, err := h.f.read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		hs := Hostiles(snap, h.f.hostileCtx(snap.Player))
		if _, err := h.f.abortFight(ctx, snap, hs, st); errors.Is(err, ErrChicken) {
			break
		}
		h.s.advance(2 * time.Second)
	}
	if !h.s.left {
		t.Fatal("expected a chicken after fleeing too long")
	}
	if h.s.t.Sub(newSim().t) < fleeMax {
		t.Fatalf("chickened after only %v", h.s.t.Sub(newSim().t))
	}
}

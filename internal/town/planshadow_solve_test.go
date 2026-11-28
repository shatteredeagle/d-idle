package town

import (
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/supply"
	"d2lod-bot/internal/town/belt"
)

// The shadow plan for two builds with an empty belt and gold for everything: the mana build asks for mana, the melee
// build does not, and both want today's 4 healing potions (profile monastery_road). Runs without game tables.
func TestSolveShadowTwoClasses(t *testing.T) {
	lt, rs := env.DefaultLootTuning(), env.DefaultResourceTuning()
	for _, tc := range []struct {
		class d2.Class
		mana  bool
	}{{d2.Sorceress, true}, {d2.Paladin, false}} {
		band := belt.PlanForClass(tc.class, 20)
		c := supply.Count{Purse: 1500, StashKnown: true, PortalCharges: 2, HasTome: true}
		pl, err := solveShadow(tc.class, "", band, belt.Columns, c, ChorePlan{Profile: "monastery_road"}, &lt, &rs)
		if err != nil {
			t.Fatalf("%v: %v", tc.class, err)
		}
		if pl.Need.Lines[supply.Healing].Min != 4 {
			t.Errorf("%v: healing min %d, want 4", tc.class, pl.Need.Lines[supply.Healing].Min)
		}
		if got := pl.Need.Lines[supply.Mana].Min > 0; got != tc.mana {
			t.Errorf("%v: mana line %v, want %v: %s", tc.class, got, tc.mana, pl.Trace())
		}
		if !strings.Contains(pl.Trace(), "would buy healing") {
			t.Errorf("%v: no healing buy in %q", tc.class, pl.Trace())
		}
	}
}

func TestSolveShadowUnknownProfile(t *testing.T) {
	lt, rs := env.DefaultLootTuning(), env.DefaultResourceTuning()
	_, err := solveShadow(d2.Paladin, "", belt.PlanForClass(d2.Paladin, 5), belt.Columns, supply.Count{}, ChorePlan{Profile: "nope"}, &lt, &rs)
	if err == nil || !strings.Contains(err.Error(), `"nope"`) {
		t.Fatalf("got %v", err)
	}
}

// V live 2026-11-24 19:24: a 4-slot belt holds 3 healing, the boss floor asked for 8 plus a buffer (9), the shop wanted 3 more into
// the pack three times, the buy failed and the Andariel chunk parked on a kit it could not carry. The floor is clamped to what
// belt plus a 2-potion pack reserve carry (pack potions are not drinkable in a fight). Two builds, no game tables.
func TestBossKitIsClampedToBeltPlusPackReserve(t *testing.T) {
	lt, rs := env.DefaultLootTuning(), env.DefaultResourceTuning()
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		band := belt.PlanForClass(class, 20)
		capHP := band.Capacity(belt.HP, belt.Columns)
		pl, err := solveShadow(class, "", band, belt.Columns, supply.Count{Purse: 3000, StashKnown: true, PortalCharges: 2, HasTome: true}, ChorePlan{Profile: "andariel"}, &lt, &rs)
		if err != nil {
			t.Fatalf("%v: %v", class, err)
		}
		if got, want := pl.MinOf(supply.Healing), capHP+2; got != want {
			t.Errorf("%v: boss healing floor %d, want belt %d + 2", class, got, capHP)
		}
		if got := pl.MinOf(supply.GoldCarry); got != 3000 {
			t.Errorf("%v: boss purse %d", class, got)
		}
	}
}

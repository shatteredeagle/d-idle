package resource

import (
	"testing"

	"d2lod-bot/internal/d2"
)

func TestMercReviveCost(t *testing.T) {
	for lvl, want := range map[int]int{0: 0, 30: 6750, 43: 13860, 75: 42180, 82: 50000, 99: 50000} {
		if got := MercReviveCost(lvl); got != want {
			t.Errorf("level %d: %d want %d", lvl, got, want)
		}
	}
}

func TestSetMerc(t *testing.T) {
	var in Input
	in.SetMerc(d2.Merc{}, false)
	if in.MercKnown || in.MercDead() {
		t.Fatal("no merc must not read as dead")
	}
	in.SetMerc(d2.Merc{}, true) // known but vanished: dead
	if !in.MercDead() {
		t.Fatal("vanished known merc is dead")
	}
	in.SetMerc(d2.Merc{Present: true, Alive: true, LifePct: 0, Level: 30}, false)
	if in.MercDead() || in.MercLifePct != 1 || in.MercLevel != 30 {
		t.Fatalf("%+v", in)
	}
	in.SetMerc(d2.Merc{Present: true, Alive: false, LifePct: 40}, false)
	if !in.MercDead() {
		t.Fatal("corpse is dead")
	}
}

func TestMercReviveWanted(t *testing.T) {
	in := Input{MercKnown: true, MercLevel: 30, Gold: 7000}
	if !in.MercReviveWanted(0) || in.MercReviveWanted(500) {
		t.Fatal("gold gate")
	}
	in.MercLifePct = 50
	if in.MercReviveWanted(0) {
		t.Fatal("alive merc needs no revive")
	}
}

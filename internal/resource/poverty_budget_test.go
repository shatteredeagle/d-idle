package resource

import (
	"testing"

	"d2lod-bot/internal/d2"
)

func TestPovertyEntryBand(t *testing.T) {
	tu := DefaultTuning()
	in := Input{HasTP: true, Gold: 50, Belt: Belt{HP: tu.HPFirst - 1}}
	if tu.HPFirst < 2 {
		t.Skip("band needs HPFirst >= 2")
	}
	if PovertyCheck(in, tu, false).On {
		t.Fatalf("one potion below HPFirst must not start poverty")
	}
	if !PovertyCheck(in, tu, true).On {
		t.Fatalf("poverty already on stays on until HPFirst is back")
	}
	in.Belt.HP = 0
	if !PovertyCheck(in, tu, false).On {
		t.Fatalf("an empty healing belt below the kit cost is poverty")
	}
}

func TestManaBuyByClass(t *testing.T) {
	tu := DefaultTuning()
	cost := KitCost(tu)
	for _, tc := range []struct {
		name  string
		class d2.Class
		gold  int
		heal  int
		want  bool
	}{
		{"paladin short healing", d2.Paladin, cost + 500, tu.HPFirst - 1, false},
		{"paladin kit unaffordable after", d2.Paladin, cost + 20, tu.HPFirst, false},
		{"paladin ok", d2.Paladin, cost + 200, tu.HPFirst, true},
		{"sorceress short healing", d2.Sorceress, 100, 0, true}, // her shop reserve rule applies elsewhere
		{"barbarian never", d2.Barbarian, 5000, 9, false},
	} {
		r := ManaBuy(ManaNeed(tc.class), tu)
		if got := ManaBuyOK(r, tu, tc.gold, 60, tc.heal, 0); got != tc.want {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
	if ManaBuyOK(ManaBuy(1, tu), tu, 5000, 60, 9, 2) {
		t.Errorf("a low-mana-use class stops at its per-visit cap")
	}
}

func TestSavingForOrder(t *testing.T) {
	tu := DefaultTuning()
	if s := SavingFor(Input{Gold: 10}, tu, 0); s.Item != "a town portal scroll" || s.Need != tu.KitTPGold-10 {
		t.Errorf("no portal: %+v", s)
	}
	in := Input{HasTP: true, Gold: tu.KitPotionGold, Belt: Belt{}}
	if s := SavingFor(in, tu, 0); s.Item != "healing potions" || s.Need != tu.HPFirst*tu.KitPotionGold-tu.KitPotionGold {
		t.Errorf("potions: %+v", s)
	}
	in.Belt.HP = tu.HPFirst
	in.Gold = 100
	if s := SavingFor(in, tu, 400); s.Item != "the repair" || s.Need != 300 {
		t.Errorf("repair: %+v", s)
	}
	if s := SavingFor(in, tu, 0); s.Item != "" {
		t.Errorf("nothing missing: %+v", s)
	}
}

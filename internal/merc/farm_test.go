package merc

import "testing"

func TestKitRefill(t *testing.T) {
	// defaults: portal 100, potion 30, 4 healing, 3 mana
	if got := KitRefill(100, 30, 4, 3); got != 310 {
		t.Errorf("KitRefill = %d", got)
	}
}

// Potions first: the reserve is 100 percent of one kit refill (was 150 before the mercrevive card), so with a 310 kit
// the merc needs price + 310. Two kits (default refill and a pricier one) pin the bar; gold between the new and the
// old bar revives now.
func TestSpendingOrderReserve(t *testing.T) {
	if DefaultKnobs().KitReservePct != 100 {
		t.Fatalf("default reserve %d, want 100", DefaultKnobs().KitReservePct)
	}
	dead := MercState{Known: true, Owned: true, Level: 10} // 750
	for _, tc := range []struct {
		name string
		kit  int
	}{{"default kit", KitRefill(100, 30, 4, 3)}, {"big kit", KitRefill(100, 60, 8, 6)}} {
		t.Run(tc.name, func(t *testing.T) {
			c := ctx()
			c.Kit = tc.kit
			bar, oldBar := 750+tc.kit, 750+tc.kit*150/100
			if got := ReviveNeed(c, 750); got != bar {
				t.Fatalf("need %d, want price + 100%% kit = %d", got, bar)
			}
			if d, _ := ShouldRevive(dead, withGold(c, bar-1), 0); d.Do || d.Reason != ReasonGold {
				t.Errorf("one gold short of price + kit: %+v", d)
			}
			if d, _ := ShouldRevive(dead, withGold(c, bar), 0); !d.Do {
				t.Errorf("price + kit: %+v", d)
			}
			if d, _ := ShouldRevive(dead, withGold(c, oldBar-1), 0); !d.Do {
				t.Errorf("between new and old bar must revive: %+v", d)
			}
		})
	}
}

func withGold(c Ctx, g int) Ctx { c.Gold = g; return c }

func TestReviveCapBinds(t *testing.T) {
	c := ctx()
	c.Gold = 99999
	dead := MercState{Known: true, Owned: true, Level: 10}
	for n, want := range []bool{true, true, false, false} {
		c.RevivesLastHour = n
		if d, _ := ShouldRevive(dead, c, 0); d.Do != want {
			t.Errorf("%d revives this hour: %+v", n, d)
		}
	}
}

func TestGoldFarm(t *testing.T) {
	k := DefaultKnobs()
	for _, c := range []struct {
		name          string
		missing       bool
		gold, need, l int
		want          bool
	}{
		{"merc alive", false, 0, 1000, 10, false},
		{"missing and poor", true, 300, 1000, 10, true},
		{"missing and rich", true, 1000, 1000, 10, false},
		{"level cap", true, 300, 1000, 17, false}, // cap 18 -> 17 = no-merc Andariel floor, job mercrevive2
		{"old cap level no longer farms", true, 300, 1000, 18, false},
		{"just below the cap", true, 300, 1000, 16, true},
	} {
		if got := GoldFarm(c.missing, c.gold, c.need, c.l, k); got != c.want {
			t.Errorf("%s: %v", c.name, got)
		}
	}
	k.GoldFarmMaxLevel = 0
	if GoldFarm(true, 0, 1000, 5, k) {
		t.Error("knob 0 must turn the farm off")
	}
	for _, a := range []int{2, 3, 4, 8, 9, 13, 17, 18} {
		if !FarmAreas[a] {
			t.Errorf("area %d missing", a)
		}
	}
	if FarmAreas[5] || FarmAreas[7] || FarmAreas[26] {
		t.Error("Dark Wood, Tamoe and the Monastery are not early areas")
	}
}

func TestWaitAndLeave(t *testing.T) {
	alive := MercState{Known: true, Owned: true, Alive: true}
	if !ShouldWaitForMerc(alive, false, 1, 5) || ShouldWaitForMerc(alive, true, 1, 5) || ShouldWaitForMerc(alive, false, 5, 5) {
		t.Error("wait for a living merc that is not near, within the budget")
	}
	dead := MercState{Known: true, Owned: true}
	if ShouldWaitForMerc(dead, false, 0, 5) {
		t.Error("never wait for a dead merc")
	}
	if !LeaveWithMercDown(dead, true) || LeaveWithMercDown(dead, false) || LeaveWithMercDown(alive, true) {
		t.Error("leave only when the crowd rule fires with the merc down")
	}
}

package merc

import "testing"

func TestKitRefill(t *testing.T) {
	// defaults: portal 100, potion 30, 4 healing, 3 mana
	if got := KitRefill(100, 30, 4, 3); got != 310 {
		t.Errorf("KitRefill = %d", got)
	}
}

// Potions first: the reserve is 150 percent of the kit, so with a 310 kit the merc needs price + 465.
func TestSpendingOrderReserve(t *testing.T) {
	c := ctx()
	c.Kit = KitRefill(100, 30, 4, 3)
	dead := MercState{Known: true, Owned: true, Level: 10} // 750
	if d, _ := ShouldRevive(dead, withGold(c, 750+465-1), 0); d.Do || d.Reason != ReasonGold {
		t.Errorf("one gold short of price + 150%% kit: %+v", d)
	}
	if d, _ := ShouldRevive(dead, withGold(c, 750+465), 0); !d.Do {
		t.Errorf("price + 150%% kit: %+v", d)
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
		{"level cap", true, 300, 1000, 16, false},
		{"just below the cap", true, 300, 1000, 15, true},
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

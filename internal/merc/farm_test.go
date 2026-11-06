package merc

import "testing"

func TestKitRefill(t *testing.T) {
	// defaults: portal 100, potion 30, 4 healing, 3 mana
	if got := KitRefill(100, 30, 4, 3); got != 310 {
		t.Errorf("KitRefill = %d", got)
	}
}

// Potions first for a HIRE: the reserve is 100 percent of one kit refill (merc.kitReservePct), so with a 310 kit a hire
// needs price + 310. A REVIVE no longer applies it (SME rule, user 2026-11-06): below clvl 13 it needs the price alone.
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
			if got := GoldNeed(c, 750); got != 750+tc.kit {
				t.Fatalf("hire need %d, want price + 100%% kit = %d", got, 750+tc.kit)
			}
			if got := ReviveNeed(c, 750); got != 750 {
				t.Fatalf("revive need %d, want the bare price 750", got)
			}
			if d, _ := ShouldRevive(dead, withGold(c, 749), 0); d.Do || d.Reason != ReasonGold {
				t.Errorf("one gold short of the price: %+v", d)
			}
			if d, _ := ShouldRevive(dead, withGold(c, 750), 0); !d.Do {
				t.Errorf("the price alone must revive: %+v", d)
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

// g2f seed 17: price 1920 (clvl 16), gold 1930 total: the merc stayed dead 7 hours behind a 155 to 310 kit reserve. SME
// rule (user 2026-11-06): the revive keeps back 0 gold below clvl 13 and 50 from clvl 13; a hire keeps the kit reserve.
func TestReviveReserveBands(t *testing.T) {
	k := DefaultKnobs()
	if k.ReviveReserveLow != 0 || k.ReviveReserveHigh != 50 || k.ReviveReserveClvl != 13 {
		t.Fatalf("defaults %d/%d/%d, want 0/50/13", k.ReviveReserveLow, k.ReviveReserveHigh, k.ReviveReserveClvl)
	}
	dead := MercState{Known: true, Owned: true, Level: 16}
	cases := []struct {
		clvl, gold int
		revive     bool
		need       int
	}{
		{12, 1930, true, 1920},  // below the edge: bare price
		{12, 1919, false, 1920}, // one short
		{13, 1969, false, 1970}, // edge clvl uses the high band
		{14, 1930, false, 1970},
		{14, 1975, true, 1970},
		{0, 1930, false, 1970}, // unknown level: the high band, never silently the low one
	}
	for _, class := range []string{"Sorceress", "Paladin"} {
		for _, full := range []bool{false, true} {
			for _, tc := range cases {
				c := ctx()
				c.PlayerClass, c.KitFull, c.Kit, c.Level = class, full, KitRefill(100, 30, 4, 3), tc.clvl
				c.Knobs = DefaultKnobs()
				c = withGold(c, tc.gold)
				if got := ReviveNeed(c, 1920); got != tc.need {
					t.Fatalf("%s full=%v clvl %d: need %d, want %d", class, full, tc.clvl, got, tc.need)
				}
				if d, _ := ShouldRevive(dead, c, 1920); d.Do != tc.revive || (!d.Do && d.Reason != ReasonGold) {
					t.Errorf("%s full=%v clvl %d gold %d: %+v want revive=%v", class, full, tc.clvl, tc.gold, d, tc.revive)
				}
			}
		}
	}
	// the stop rule stays: 3+ deaths an hour raise the bar to 5x the price plus the reserve
	c := ctx()
	c.Knobs, c.Level, c.DeathsLastHour = DefaultKnobs(), 14, 4
	if got := ReviveNeed(c, 750); got != 5*750+50 {
		t.Errorf("stop bar %d", got)
	}
	// a hire still applies kitReservePct (and the full-kit pct) whatever the clvl
	c = ctx()
	c.Knobs, c.Kit, c.Level = DefaultKnobs(), 310, 12
	if got := GoldNeed(c, 400); got != 710 {
		t.Errorf("hire need %d want 710", got)
	}
	c.KitFull = true
	if got := GoldNeed(c, 400); got != 555 {
		t.Errorf("hire need, full kit %d want 555", got)
	}
}

package merc

import (
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)

func at(min int) time.Time { return t0.Add(time.Duration(min) * time.Minute) }

type step struct {
	min        int
	want       bool
	gold, need int
	on         bool
	ended      string
}

func TestFarmBudget(t *testing.T) {
	k := DefaultKnobs() // budget 40, cooldown 60
	for _, tc := range []struct {
		name  string
		steps []step
	}{
		{"budget expiry then cooldown then re-arm", []step{
			{0, true, 300, 2000, true, FarmEndNone},
			{39, true, 300, 2000, true, FarmEndNone},
			{40, true, 300, 2000, false, FarmEndBudget},
			{41, true, 300, 2000, false, FarmEndNone},
			{99, true, 300, 2000, false, FarmEndNone}, // 40 + 60 = 100
			{100, true, 300, 2000, true, FarmEndNone},
		}},
		{"gold reaches the bar: early end, no cooldown", []step{
			{0, true, 300, 2000, true, FarmEndNone},
			{10, true, 2000, 2000, false, FarmEndGold},
			{11, true, 300, 2000, true, FarmEndNone}, // chores spent it, below 90 percent: arms again
		}},
		{"hysteresis: hovering at the bar does not flap", []step{
			{0, true, 300, 2000, true, FarmEndNone},
			{10, true, 2100, 2000, false, FarmEndGold},
			{11, true, 1900, 2000, false, FarmEndNone}, // 95 percent: still off
			{12, true, 1801, 2000, false, FarmEndNone},
			{13, true, 1799, 2000, true, FarmEndNone}, // below 90 percent
		}},
		{"never arms at or above the bar", []step{{0, true, 2000, 2000, false, FarmEndNone}}},
		{"merc back or level cap ends it without cooldown", []step{
			{0, true, 300, 2000, true, FarmEndNone},
			{5, false, 300, 2000, false, FarmEndWant},
			{6, true, 300, 2000, true, FarmEndNone},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var b FarmBudget
			for _, s := range tc.steps {
				on, ended := b.Step(at(s.min), s.want, s.gold, s.need, k)
				if on != s.on || ended != s.ended {
					t.Fatalf("minute %d: on=%v ended=%q, want on=%v ended=%q", s.min, on, ended, s.on, s.ended)
				}
			}
		})
	}
}

func TestFarmBudgetKnobs(t *testing.T) {
	k := DefaultKnobs()
	k.GoldFarmBudgetMin = 0 // no limit
	var b FarmBudget
	b.Step(at(0), true, 0, 100, k)
	if on, _ := b.Step(at(9999), true, 0, 100, k); !on {
		t.Error("budget 0 must not end the farm")
	}
	k = DefaultKnobs()
	k.GoldFarmBudgetMin, k.GoldFarmCooldownMin = 10, 5
	b = FarmBudget{}
	b.Step(at(0), true, 0, 100, k)
	b.Step(at(10), true, 0, 100, k)
	if on, _ := b.Step(at(14), true, 0, 100, k); on {
		t.Error("inside the 5 minute cooldown")
	}
	if on, _ := b.Step(at(15), true, 0, 100, k); !on {
		t.Error("cooldown over")
	}
}

func TestLogFarmBudget(t *testing.T) {
	got := LogFarmBudget(DefaultKnobs(), 1990, 900)
	if got != "merc: gold farm budget minutes=40 cooldown=60 need=1990 gold=900" || !strings.HasPrefix(got, Prefix) {
		t.Errorf("%q", got)
	}
}

// Hire bar: a full kit (just bought) reserves 50 percent of the refill, not 100. Two kit profiles. A revive ignores the kit.
func TestFullKitReserve(t *testing.T) {
	if k := DefaultKnobs(); k.KitReservePctFullKit != 50 || k.GoldFarmBudgetMin != 40 || k.GoldFarmCooldownMin != 60 || k.GoldFarmMaxLevel != 17 {
		t.Fatalf("defaults %+v", k)
	}
	dead := MercState{Known: true, Owned: true, Level: 10} // 750
	for _, tc := range []struct {
		name string
		kit  int
	}{{"default kit", KitRefill(100, 30, 4, 3)}, {"big kit", KitRefill(100, 60, 8, 6)}} {
		t.Run(tc.name, func(t *testing.T) {
			c := ctx()
			c.Kit = tc.kit
			if got, want := GoldNeed(c, 750), 750+tc.kit; got != want {
				t.Fatalf("empty kit need %d, want %d", got, want)
			}
			c.KitFull = true
			bar := 750 + tc.kit/2
			if got := GoldNeed(c, 750); got != bar {
				t.Fatalf("full kit need %d, want %d", got, bar)
			}
			if got := ReviveNeed(c, 750); got != 750 {
				t.Errorf("a revive does not reserve the kit: %d", got)
			}
			if d, _ := ShouldRevive(dead, withGold(c, 749), 0); d.Do {
				t.Errorf("one short: %+v", d)
			}
			if d, _ := ShouldRevive(dead, withGold(c, 750), 0); !d.Do {
				t.Errorf("at the bar: %+v", d)
			}
		})
	}
}

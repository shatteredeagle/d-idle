package ladder_test

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/ladder"
	"d2lod-bot/internal/resource"
	"d2lod-bot/internal/town"
)

// classes are the two builds of the tests: the default Sorceress and a Paladin. The names come from the catalogue's
// class table through d2.Class.String, never from a literal in the ladder.
var classes = []d2.Class{d2.Sorceress, d2.Paladin}

func newLadder(t testing.TB, src ladder.AnswerSource) (*ladder.Ladder, *decide.ManualClock, *ladder.RecordSink) {
	t.Helper()
	clk := &decide.ManualClock{}
	sink := &ladder.RecordSink{}
	l, err := ladder.Open("", "t0", clk, sink, src, nil)
	if err != nil {
		t.Fatal(err)
	}
	return l, clk, sink
}

func hero(c d2.Class, gold int) ladder.Hero {
	return ladder.Hero{Class: c, AreaClass: "town", Durability: -1, Gold: gold}
}

// Golden parity, t1.repair: with the shipped rule file and nothing learned the ladder repairs exactly when the legacy
// durability test (below town.repairBelowPct, default knob) says so. The grid is every percent 0..100 x gold bands x
// 2 builds x 3 difficulties.
func TestParityRepair(t *testing.T) {
	below := env.DefaultTownTuning().RepairBelowPct
	l, _, _ := newLadder(t, nil)
	n := 0
	for _, c := range classes {
		for _, diff := range []d2.Difficulty{d2.Normal, d2.Nightmare, d2.Hell} {
			for _, gold := range []int{0, 99, 100, 499, 500, 1499, 1500, 9000} {
				for pct := 0; pct <= 100; pct++ {
					h := hero(c, gold)
					h.Diff = diff
					got, err := l.RepairWanted(h, pct)
					if err != nil {
						t.Fatal(err)
					}
					if want := town.RepairDue(pct, below); got != want {
						t.Fatalf("%v diff %v gold %d durability %d: ladder %v, legacy %v", c, diff, gold, pct, got, want)
					}
					n++
				}
			}
		}
	}
	t.Logf("t1.repair parity grid: %d states", n)
}

// Golden parity, t1.tp.lost: buy exactly when the legacy purchase test (resource.BuyKit with the scroll at 100 gold,
// the knob default) buys.
func TestParityTPLost(t *testing.T) {
	price := env.DefaultResourceTuning().KitTPGold
	l, _, _ := newLadder(t, nil)
	n := 0
	for _, c := range classes {
		for _, area := range []string{"town", "field", "boss"} {
			for gold := 0; gold <= 2000; gold++ {
				h := hero(c, gold)
				h.AreaClass = area
				got, err := l.TPBuyWanted(h)
				if err != nil {
					t.Fatal(err)
				}
				want := resource.BuyKit(resource.Input{Gold: gold}, resource.Tuning{KitTPGold: price}).TP
				if got != want {
					t.Fatalf("%v area %s gold %d: ladder %v, legacy %v", c, area, gold, got, want)
				}
				n++
			}
		}
	}
	t.Logf("t1.tp.lost parity grid: %d states", n)
}

// Refused points: the parity test found a state where the default rule file and the legacy code disagree, so the point
// stays on its legacy path. Each case names one witness state; the test fails if the disagreement disappears (then the
// point may be wired).
func TestRefusedPointsDisagree(t *testing.T) {
	cat, err := decide.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	rf, err := decide.ShippedRuleFile()
	if err != nil {
		t.Fatal(err)
	}
	ans := func(point string, o decide.Obs) string {
		ctx, err := cat.Build(point, o, 1, nil)
		if err != nil {
			t.Fatal(err)
		}
		if a, ok := rf.Lookup(ctx); ok {
			return a
		}
		a, _ := cat.Default(ctx)
		return a
	}
	base := func(cat map[string]string, num map[string]float64) decide.Obs {
		c := map[string]string{"act": "1", "diff": "normal", "hc": "normal"}
		for k, v := range cat {
			c[k] = v
		}
		return decide.Obs{Class: "sorceress", Cat: c, Num: num}
	}
	res := resource.FromEnv(env.DefaultTuning())

	// t0.retreat.drink: life 30 percent, kit OK, a pack in melee, healing potions in the belt. Legacy drinks (life under
	// the drink line, a potion held); the rule file portals at 25 to 35 percent.
	in := resource.Input{Life: 30, MaxLife: 100, Hostiles: 2, HasTP: true, Belt: resource.Belt{HP: 5, Rows: 1}, Level: 5, Gold: 500}
	if d := resource.PotionPolicy(in, res); d.Drink != resource.PotionLife {
		t.Fatalf("legacy no longer drinks at 30 percent: %+v", d)
	}
	if got := ans("t0.retreat.drink", base(map[string]string{"kit": "OK", "packdanger": "1"}, map[string]float64{"life": 30})); got != "portal" {
		t.Errorf("t0.retreat.drink: rule file says %q, expected the witness portal", got)
	}
	// t0.retreat.drink at high life: the point has no 'keep fighting' answer; the file answers drink at 90 percent,
	// legacy drinks nothing.
	in.Life = 90
	if d := resource.PotionPolicy(in, res); d.Drink != resource.PotionNone {
		t.Fatalf("legacy drinks at 90 percent: %+v", d)
	}
	if got := ans("t0.retreat.drink", base(map[string]string{"kit": "OK", "packdanger": "1"}, map[string]float64{"life": 90})); got != "drink" {
		t.Errorf("t0.retreat.drink at 90: %q", got)
	}
	// t1.return: legacy FieldGate (knob 0) always lets the hero leave town, the file keeps a kit-NONE hero in town.
	fv := resource.FieldGate(resource.Input{Gold: 0, Class: d2.Sorceress, Level: 1}, res)
	if !fv.Go {
		t.Fatalf("legacy field gate stopped: %+v", fv)
	}
	if got := ans("t1.return", base(map[string]string{"kit": "NONE", "areaclass": "field"}, nil)); got != "no" {
		t.Errorf("t1.return: %q", got)
	}
	// t1.town.trigger: legacy trips on potions (TripHPMin), the file on free pack rows.
	trip := resource.TownTripTrigger(resource.Input{Life: 100, MaxLife: 100, HasTP: true, Gold: 1000, LastTripGold: -1, Belt: resource.Belt{HP: 6, Rows: 1}}, res)
	if trip.Go {
		t.Fatalf("legacy trips with a full belt: %+v", trip)
	}
	if got := ans("t1.town.trigger", base(map[string]string{"kit": "OK"}, map[string]float64{"freerows": 0})); got != "go" {
		t.Errorf("t1.town.trigger: %q", got)
	}
	// t1.restock: legacy refuses a trip under tripGoldFloor (300), the file restocks from 100 gold.
	if resource.TripBudgetSpent(resource.Input{Gold: 150}, res) == "" {
		t.Fatal("legacy trip budget no longer refuses at 150 gold")
	}
	if got := ans("t1.restock", base(map[string]string{"belt": "low"}, map[string]float64{"gold": 150})); got != "yes" {
		t.Errorf("t1.restock: %q", got)
	}
}

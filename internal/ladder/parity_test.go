package ladder_test

import (
	"testing"
	"time"

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
	n, defSplit := 0, 0
	for _, c := range classes {
		for _, diff := range []d2.Difficulty{d2.Normal, d2.Nightmare, d2.Hell} {
			for _, gold := range []int{0, 99, 100, 499, 500, 1499, 1500, 9000} {
				for pct := 0; pct <= 100; pct++ {
					h := hero(c, gold)
					h.Diff = diff
					h.Durability = pct
					want := town.RepairDue(pct, below)
					got, err := l.RepairWanted(h, pct, want)
					if err != nil {
						t.Fatal(err)
					}
					if def := defIs(t, l, ladder.PointRepair, h, decide.Obs{}, "repair"); def != want {
						defSplit++
					}
					if got != want {
						t.Fatalf("%v diff %v gold %d durability %d: ladder %v, legacy %v", c, diff, gold, pct, got, want)
					}
					n++
				}
			}
		}
	}
	if defSplit != 0 {
		t.Errorf("catalogue default differs from the legacy test at %d states", defSplit)
	}
	t.Logf("t1.repair parity grid: %d states", n)
}

// Golden parity, t1.tp.lost: buy exactly when the legacy purchase test (resource.BuyKit with the scroll at 100 gold,
// the knob default) buys.
func TestParityTPLost(t *testing.T) {
	price := env.DefaultResourceTuning().KitTPGold
	l, _, _ := newLadder(t, nil)
	n, defSplit := 0, 0
	for _, c := range classes {
		for _, area := range []string{"town", "field", "boss"} {
			for gold := 0; gold <= 2000; gold++ {
				h := hero(c, gold)
				h.AreaClass = area
				want := resource.BuyKit(resource.Input{Gold: gold}, resource.Tuning{KitTPGold: price}).TP
				got, err := l.TPBuyWanted(h, want)
				if err != nil {
					t.Fatal(err)
				}
				if def := defIs(t, l, ladder.PointTPLost, h, decide.Obs{}, "buy"); def != want {
					defSplit++
				}
				if got != want {
					t.Fatalf("%v area %s gold %d: ladder %v, legacy %v", c, area, gold, got, want)
				}
				n++
			}
		}
	}
	if defSplit != 0 {
		t.Errorf("catalogue default differs from the legacy test at %d states", defSplit)
	}
	t.Logf("t1.tp.lost parity grid: %d states", n)
}

// defIs reports whether the catalogue default at this state is the positive answer pos.
func defIs(t testing.TB, l *ladder.Ladder, point string, h ladder.Hero, extra decide.Obs, pos string) bool {
	t.Helper()
	a, err := l.CatalogueDefault(point, h, extra)
	if err != nil {
		t.Fatal(err)
	}
	return a == pos
}

// Golden parity, t1.town.trigger: over the legacy policy's input space (healing potions in the belt, pack fill, gold,
// difficulty, two builds) the answer equals resource.TownTripTrigger called directly. The call site asks the point only
// when the policy says go, so the grid compares the answer on the Go verdicts and checks that a no verdict is
// reported as such. The catalogue default is compared too: where it differs the legacy answer still wins (counted).
func TestParityTownTrigger(t *testing.T) {
	rt := resource.FromEnv(env.DefaultTuning())
	l, _, _ := newLadder(t, nil)
	n, goes, split := 0, 0, 0
	for _, c := range classes {
		for _, diff := range []d2.Difficulty{d2.Normal, d2.Hell} {
			for hp := 0; hp <= 8; hp++ {
				for _, gold := range []int{0, 59, 60, 299, 300, 1500} {
					for _, pct := range []int{0, 40, 79, 80, 100} {
						in := resource.Input{Class: c, Life: 100, MaxLife: 100, HasTP: true, Gold: gold, LastTripGold: -1,
							Belt: resource.Belt{HP: hp, Rows: 2}, Diff: diff, Level: 12}
						trip := resource.TownTripTrigger(in, rt)
						fv := resource.FieldGate(in, rt)
						h := hero(c, gold)
						h.Diff = diff
						belt := ladder.BeltBand(fv.HaveHP, fv.WantHP)
						got, err := l.TripWanted(h, belt, pct, trip.Go)
						if err != nil {
							t.Fatal(err)
						}
						if got != trip.Go {
							t.Fatalf("%v %v hp %d gold %d pack %d: ladder %v, legacy %v", c, diff, hp, gold, pct, got, trip.Go)
						}
						extra := decide.Obs{Cat: map[string]string{"belt": belt}, Num: map[string]float64{"packused": float64(pct)}}
						if trip.Go {
							goes++
							if !defIs(t, l, ladder.PointTrigger, h, extra, "go") {
								split++
							}
						}
						n++
					}
				}
			}
		}
	}
	t.Logf("t1.town.trigger parity grid: %d states, %d with a Go verdict, catalogue default differs at %d of those", n, goes, split)
}

// Golden parity, t1.return: the answer equals resource.FieldGate's Go. The gate is exercised with the belt gate on
// (MinFieldBeltPct 50, the case where it can refuse) and off (the default), over potions, portal, gold and two builds.
func TestParityReturn(t *testing.T) {
	l, _, _ := newLadder(t, nil)
	n, refusals, split := 0, 0, 0
	for _, minPct := range []int{0, 50} {
		tn := env.DefaultTuning()
		tn.Resource.MinFieldBeltPct = minPct
		rt := resource.FromEnv(tn)
		for _, c := range classes {
			for hp := 0; hp <= 6; hp++ {
				for _, tp := range []bool{false, true} {
					for _, gold := range []int{0, 59, 100, 299, 1000} {
						in := resource.Input{Class: c, Life: 100, MaxLife: 100, HasTP: tp, Gold: gold, Level: 8, InTown: true,
							Belt: resource.Belt{HP: hp, Rows: 1}}
						fv := resource.FieldGate(in, rt)
						kit := ladder.KitOK
						switch {
						case !tp && hp == 0:
							kit = ladder.KitNone
						case !tp || hp < 2:
							kit = ladder.KitThin
						}
						h := hero(c, gold)
						got, err := l.ReturnWanted(h, kit, fv.Go)
						if err != nil {
							t.Fatal(err)
						}
						if got != fv.Go {
							t.Fatalf("minPct %d %v hp %d tp %v gold %d: ladder %v, legacy %v", minPct, c, hp, tp, gold, got, fv.Go)
						}
						if !fv.Go {
							refusals++
						}
						extra := decide.Obs{Cat: map[string]string{"kit": kit, "areaclass": "field"}}
						if defIs(t, l, ladder.PointReturn, h, extra, "yes") != fv.Go {
							split++
						}
						n++
					}
				}
			}
		}
	}
	t.Logf("t1.return parity grid: %d states, %d legacy refusals, catalogue default differs from the gate at %d (the gate wins)", n, refusals, split)
}

// Golden parity, t1.restock: the answer equals the legacy purse test (gold at or above resource.tripGoldFloor, the
// gold clause of TripBudgetSpent) at every gold from 0 to 2000, three belt states, two builds. The catalogue default is
// compared with the same test at its band edges 60 and 300.
func TestParityRestock(t *testing.T) {
	rt := resource.FromEnv(env.DefaultTuning())
	l, _, _ := newLadder(t, nil)
	n, split := 0, 0
	for _, c := range classes {
		for _, belt := range []string{ladder.BeltFull, ladder.BeltHalf, ladder.BeltLow} {
			for gold := 0; gold <= 2000; gold++ {
				want := resource.TripBudgetSpent(resource.Input{Gold: gold}, rt) == ""
				h := hero(c, gold)
				got, err := l.RestockWanted(h, belt, gold, want)
				if err != nil {
					t.Fatal(err)
				}
				if got != want {
					t.Fatalf("%v belt %s gold %d: ladder %v, legacy %v", c, belt, gold, got, want)
				}
				extra := decide.Obs{Cat: map[string]string{"belt": belt}, Num: map[string]float64{"tripgold": float64(gold)}}
				if defIs(t, l, ladder.PointRestock, h, extra, "yes") != want {
					split++
				}
				n++
			}
		}
	}
	if split != 0 {
		t.Errorf("catalogue default differs from the legacy purse test at %d states", split)
	}
	t.Logf("t1.restock parity grid: %d states", n)
}

// A learned rule wins over the legacy answer at every wired point, in either direction, and only at its own context.
func TestLearnedRuleOverridesLegacy(t *testing.T) {
	type call func(l *ladder.Ladder, h ladder.Hero, legacy bool) (bool, error)
	pts := []struct {
		point, keyHas, teach string
		legacy               bool // the legacy answer at the taught context
		ask                  call
	}{
		{ladder.PointRepair, "durability=20-30", "skip", true, func(l *ladder.Ladder, h ladder.Hero, lg bool) (bool, error) { return l.RepairWanted(h, 25, lg) }},
		{ladder.PointTPLost, "gold=lt100", "buy", false, func(l *ladder.Ladder, h ladder.Hero, lg bool) (bool, error) { return l.TPBuyWanted(h, lg) }},
		{ladder.PointReturn, "kit=NONE", "no", true, func(l *ladder.Ladder, h ladder.Hero, lg bool) (bool, error) { return l.ReturnWanted(h, "NONE", lg) }},
		{ladder.PointTrigger, "belt=half", "continue", true, func(l *ladder.Ladder, h ladder.Hero, lg bool) (bool, error) {
			return l.TripWanted(h, ladder.BeltHalf, 10, lg)
		}},
		{ladder.PointRestock, "tripgold=lt60", "yes", false, func(l *ladder.Ladder, h ladder.Hero, lg bool) (bool, error) {
			return l.RestockWanted(h, ladder.BeltLow, 10, lg)
		}},
	}
	for _, p := range pts {
		for _, c := range classes {
			src, err := ladder.ParseScript([]byte(`{"rows":[{"point":"` + p.point + `","keyHas":"` + p.keyHas + `","answer":"` + p.teach + `","confirm":"yes","delaySec":1}]}`))
			if err != nil {
				t.Fatal(err)
			}
			l, clk, _ := newLadder(t, src)
			h := hero(c, 10)
			for i := 0; i < 6; i++ {
				if _, err := p.ask(l, h, p.legacy); err != nil {
					t.Fatalf("%s %v: %v", p.point, c, err)
				}
				clk.Advance(5 * time.Second)
				l.Pump()
			}
			got, err := p.ask(l, h, p.legacy)
			if err != nil {
				t.Fatal(err)
			}
			if got == p.legacy {
				t.Errorf("%s %v: the rule %q did not replace the legacy answer %v (stats %+v)", p.point, c, p.teach, p.legacy, l.Stats())
			}
			if l.Stats().ByRule == 0 {
				t.Errorf("%s %v: no decision came from a rule", p.point, c)
			}
		}
	}
}

// Witnesses of the four points the first slice refused. After the catalogue v2 (internal/decide parity table) the
// default answer equals the legacy action at each witness state; the full grid lives in internal/decide. The points
// stay unwired until the ladder maps the new fields (held, clvl, retreats, tripgold, packused) from the hero.
func TestRefusedPointsNowAgree(t *testing.T) {
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

	in := resource.Input{Life: 30, MaxLife: 100, Hostiles: 2, HasTP: true, Belt: resource.Belt{HP: 5, Rows: 1}, Level: 5, Gold: 500}
	if d := resource.PotionPolicy(in, res); d.Drink != resource.PotionLife {
		t.Fatalf("legacy no longer drinks at 30 percent: %+v", d)
	}
	if got := ans("t0.retreat.drink", base(map[string]string{"kit": "OK", "packdanger": "1", "held": "hp"}, map[string]float64{"life": 30})); got != "drink" {
		t.Errorf("t0.retreat.drink at 30: %q, legacy drinks", got)
	}
	in.Life = 90
	if d := resource.PotionPolicy(in, res); d.Drink != resource.PotionNone {
		t.Fatalf("legacy drinks at 90 percent: %+v", d)
	}
	if got := ans("t0.retreat.drink", base(map[string]string{"kit": "OK", "packdanger": "1", "held": "hp"}, map[string]float64{"life": 90})); got != "fight" {
		t.Errorf("t0.retreat.drink at 90: %q, legacy drinks nothing", got)
	}
	if fv := resource.FieldGate(resource.Input{Gold: 0, Class: d2.Sorceress, Level: 1}, res); !fv.Go {
		t.Fatalf("legacy field gate stopped: %+v", fv)
	}
	if got := ans("t1.return", base(map[string]string{"kit": "NONE", "areaclass": "field"}, nil)); got != "yes" {
		t.Errorf("t1.return: %q", got)
	}
	trip := resource.TownTripTrigger(resource.Input{Life: 100, MaxLife: 100, HasTP: true, Gold: 1000, LastTripGold: -1, Belt: resource.Belt{HP: 6, Rows: 1}}, res)
	if trip.Go {
		t.Fatalf("legacy trips with a full belt: %+v", trip)
	}
	if got := ans("t1.town.trigger", base(map[string]string{"belt": "full"}, map[string]float64{"packused": 10})); got != "continue" {
		t.Errorf("t1.town.trigger: %q", got)
	}
	if resource.TripBudgetSpent(resource.Input{Gold: 150}, res) == "" {
		t.Fatal("legacy trip budget no longer refuses at 150 gold")
	}
	if got := ans("t1.restock", base(map[string]string{"belt": "low"}, map[string]float64{"tripgold": 150})); got != "no" {
		t.Errorf("t1.restock: %q", got)
	}
}

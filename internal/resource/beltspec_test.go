package resource

import (
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
)

func specInput(c d2.Class, lvl, gold int) Input {
	return Input{Class: c, Level: lvl, Gold: gold, HasTP: true, Belt: Belt{Rows: 2}, LastTripGold: -1}
}

func TestSpecTemplateColumns(t *testing.T) {
	hp, mp, rv, none := PotionLife, PotionMana, PotionRejuv, PotionNone
	cases := []struct {
		name string
		spec PlanSpec
		want [4]Potion
	}{
		{"balanced", PlanSpec{Template: 1}, [4]Potion{hp, hp, mp, rv}},
		{"all health", PlanSpec{Template: 2}, [4]Potion{hp, hp, hp, hp}},
		{"all mana", PlanSpec{Template: 3}, [4]Potion{mp, mp, mp, mp}},
		{"3:1", PlanSpec{Template: 4}, [4]Potion{hp, hp, hp, mp}},
		{"1:1", PlanSpec{Template: 5}, [4]Potion{hp, hp, mp, mp}},
		{"rejuv only", PlanSpec{Template: 6}, [4]Potion{rv, rv, rv, rv}},
		{"column override", PlanSpec{Template: 2, Cols: [4]int{0, 0, ColMana, ColEmpty}}, [4]Potion{hp, hp, mp, none}},
		{"override on the build plan", PlanSpec{Cols: [4]int{ColRejuv, 0, 0, 0}}, [4]Potion{rv, hp, mp, mp}},
	}
	for _, c := range cases {
		for _, class := range []d2.Class{d2.Sorceress, d2.Barbarian} {
			p, err := c.spec.Apply(PlanFor(class, 12))
			if err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
			if class == d2.Sorceress && c.name == "override on the build plan" {
				if p.Cols != c.want {
					t.Errorf("%s %v: cols %v want %v", c.name, class, p.Cols, c.want)
				}
				continue
			}
			if c.name != "override on the build plan" && p.Cols != c.want {
				t.Errorf("%s %v: cols %v want %v", c.name, class, p.Cols, c.want)
			}
		}
	}
}

func TestSpecErrorsNameTheValue(t *testing.T) {
	if _, err := TemplateID("mana only"); err == nil || !strings.Contains(err.Error(), `"mana only"`) || !strings.Contains(err.Error(), "all mana") {
		t.Fatalf("unknown template name: %v", err)
	}
	for name, want := range map[string]int{"": 0, "default": 0, "All Mana": 3, "3:1": 4, "rejuv only": 6} {
		if got, err := TemplateID(name); err != nil || got != want {
			t.Errorf("TemplateID(%q) = %d, %v", name, got, err)
		}
	}
	if _, err := (PlanSpec{Template: 9}).Apply(PlanFor(d2.Sorceress, 5)); err == nil || !strings.Contains(err.Error(), "id 9") {
		t.Fatalf("unknown template id: %v", err)
	}
	if _, err := (PlanSpec{Cols: [4]int{0, 7, 0, 0}}).Apply(PlanFor(d2.Sorceress, 5)); err == nil || !strings.Contains(err.Error(), "beltCol2") {
		t.Fatalf("unknown column value: %v", err)
	}
}

// The default spec is the old behaviour: PlanBuys with the zero spec equals the shopping list of the build's own plan
// (numbers from origin/main before this change, same fixtures).
func TestPlanBuysDefaultUnchanged(t *testing.T) {
	tn := DefaultTuning()
	pr := DefaultPrices()
	for _, c := range []struct {
		class        d2.Class
		lvl, gold    int
		hp, mp, rv   int
		tierHP, cost int
	}{
		{d2.Sorceress, 12, 1000, 4, 4, 0, 2, 480},
		{d2.Barbarian, 12, 1000, 8, 0, 0, 2, 480},
		{d2.Paladin, 12, 1000, 6, 2, 0, 2, 480},
	} {
		b := PlanBuys(specInput(c.class, c.lvl, c.gold), tn, pr)
		if b.HP != c.hp || b.MP != c.mp || b.RV != c.rv || b.TierHP != c.tierHP && c.hp > 0 || b.Cost != c.cost {
			t.Errorf("%v: %+v, want hp %d mp %d rv %d cost %d", c.class, b, c.hp, c.mp, c.rv, c.cost)
		}
	}
}

func TestPlanBuysByTemplate(t *testing.T) {
	pr := DefaultPrices()
	cases := []struct {
		name       string
		class      d2.Class
		tpl        int
		gold       int
		hp, mp, rv int
	}{
		// clvl 12, two belt rows: each column holds two potions
		{"sorc all mana", d2.Sorceress, 3, 1000, 0, 8, 0},
		{"barb all mana", d2.Barbarian, 3, 1000, 0, 8, 0}, // a Barbarian with a mana plan still buys what the plan says
		{"sorc all health", d2.Sorceress, 2, 1000, 8, 0, 0},
		{"pal 1:1", d2.Paladin, 5, 1000, 4, 4, 0},
		{"sorc 3:1", d2.Sorceress, 4, 1000, 6, 2, 0},
		{"sorc balanced", d2.Sorceress, 1, 1000, 4, 2, 2},
		{"sorc rejuv only", d2.Sorceress, 6, 10000, 0, 0, 8},
	}
	for _, c := range cases {
		tn := DefaultTuning()
		tn.BeltPlan = c.tpl
		b := PlanBuys(specInput(c.class, 12, c.gold), tn, pr)
		if b.HP != c.hp || b.MP != c.mp || b.RV != c.rv {
			t.Errorf("%s: bought hp %d mp %d rv %d, want %d %d %d (%+v)", c.name, b.HP, b.MP, b.RV, c.hp, c.mp, c.rv, b)
		}
	}
}

// Gold is short: healing comes first unless the plan has none; the cap by gold holds and an excluded kind is never bought.
func TestPlanBuysGoldShort(t *testing.T) {
	pr := DefaultPrices()
	tn := DefaultTuning()
	in := specInput(d2.Sorceress, 12, 150)
	b := PlanBuys(in, tn, pr)
	if b.HP == 0 || b.Cost > in.Gold-tn.ReserveGold {
		t.Fatalf("default poor: %+v", b)
	}
	tn.BeltPlan = 3
	b = PlanBuys(in, tn, pr)
	if b.HP != 0 || b.MP == 0 || b.Cost > in.Gold-tn.ReserveGold {
		t.Fatalf("all mana poor: %+v", b)
	}
	tn.BeltPlan = 2
	b = PlanBuys(in, tn, pr)
	if b.MP != 0 || b.RV != 0 || b.HP == 0 {
		t.Fatalf("all health poor must not buy mana: %+v", b)
	}
}

// An all-mana belt is not "low" for lacking healing potions; a plan with a healing column still is.
func TestKitHealingWaivesAnAllManaPlan(t *testing.T) {
	tn := DefaultTuning()
	mana, _ := PlanSpec{Template: 3}.Apply(PlanFor(d2.Sorceress, 12))
	if got := KitHealing(mana, 0, tn); KitLow(got, tn) {
		t.Errorf("all-mana plan reported low (healing %d)", got)
	}
	if sf := KitShortfall(true, KitHealing(mana, 0, tn), tn.MPFirst, tn); sf != 0 {
		t.Errorf("all-mana shortfall with mana in hand = %d", sf)
	}
	std := PlanFor(d2.Sorceress, 12)
	if got := KitHealing(std, 0, tn); !KitLow(got, tn) {
		t.Errorf("default plan with no healing must stay low (healing %d)", got)
	}
	rej, _ := PlanSpec{Template: 6}.Apply(PlanFor(d2.Sorceress, 12))
	if !rej.HasHealing() || !KitLow(KitHealing(rej, 0, tn), tn) {
		t.Errorf("a rejuvenation plan is a healing plan")
	}
}

// The field gate has nothing to demand of an all-mana belt (want 0 healing), and still demands a portal elsewhere.
func TestFieldGateAllMana(t *testing.T) {
	tn := DefaultTuning()
	tn.MinFieldBeltPct = 50
	tn.BeltPlan = 3
	if v := FieldGate(specInput(d2.Sorceress, 12, 500), tn); !v.Go {
		t.Fatalf("%+v", v)
	}
	tn.BeltPlan = 0
	if v := FieldGate(specInput(d2.Sorceress, 12, 500), tn); v.Go {
		t.Fatalf("empty default belt must not go: %+v", v)
	}
}

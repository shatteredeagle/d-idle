package combat

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
)

func tgtKey(key string, a Affix) Target {
	return Target{Mon: &data.Monster{Key: key}, Affixes: a}
}

func TestPlanManaBurn(t *testing.T) {
	p := PlanForTarget(tgtKey("fallen1", AffixManaBurn), d2.Sorceress, d2.Hell, 80)
	if !p.NoMelee || !p.KillFirst || p.KeepRange < 6 || p.ManaReserve != 40 {
		t.Fatalf("%+v", p)
	}
	if n := PlanForTarget(tgtKey("fallen1", AffixManaBurn), d2.Sorceress, d2.Normal, 20); n.NoMelee {
		t.Fatalf("normal: %+v", n)
	}
	if b := PlanForTarget(tgtKey("fallen1", AffixManaBurn), d2.Barbarian, d2.Hell, 80); b.ManaReserve != 0 || b.NoMelee {
		t.Fatalf("barbarian has no mana need: %+v", b)
	}
}

func TestPlanKiting(t *testing.T) {
	for _, a := range []Affix{AffixExtraFast, AffixTeleportation} {
		if PlanForTarget(tgtKey("x", a), d2.Sorceress, d2.Hell, 80).Kite {
			t.Fatalf("kite allowed for %v", a)
		}
	}
	if !PlanForTarget(tgtKey("x", AffixStoneSkin), d2.Sorceress, d2.Hell, 80).Kite {
		t.Fatal("stone skin may be kited")
	}
}

func TestPlanLightningAndCursed(t *testing.T) {
	p := PlanForTarget(tgtKey("x", AffixLightningEnchanted|AffixCursed), d2.Paladin, d2.Hell, 70)
	if p.KeepRange < lightningSpacing || !p.NoMelee || !p.KillFirst {
		t.Fatalf("%+v", p)
	}
	if PlanForTarget(tgtKey("x", AffixLightningEnchanted), d2.Paladin, d2.Hell, 80).NoMelee {
		t.Fatal("clvl 80 may melee")
	}
}

func TestPlanMonsterRules(t *testing.T) {
	cases := map[string]func(Plan) bool{
		"doomknight3": func(p Plan) bool { return p.KillFirst && p.KeepRange >= 5 },
		"fetish5":     func(p Plan) bool { return p.KeepRange >= dollSpacing },
		"fingermage2": func(p Plan) bool { return p.NoMelee },
		"megademon6":  func(p Plan) bool { return p.Skip },
		"unraveler4":  func(p Plan) bool { return p.KillFirst },
		"fallen1":     func(p Plan) bool { return p == Plan{Kite: true} },
	}
	for k, ok := range cases {
		if p := PlanForTarget(tgtKey(k, 0), d2.Sorceress, d2.Hell, 85); !ok(p) {
			t.Errorf("%s: %+v", k, p)
		}
	}
}

func sk(name string, e data.DamageType, cost int) SkillSpec {
	return SkillSpec{Name: name, Elem: e, Cost: cost, Kind: KindBolt}
}

func TestPickByResist(t *testing.T) {
	order := []SkillSpec{sk("Blizzard", data.Cold, 20), sk("Fire Ball", data.Fire, 20), sk("Lightning", data.Lightning, 20)}
	// magic resistant unique with cold 80, fire 20: switch to fire
	tg := &Target{HasRes: true, Res: [6]int{0, 0, 20, 90, 80, 0}}
	got, why := PickByResist(order, tg, 100, nil)
	if why != PickOK || got.Name != "Fire Ball" {
		t.Fatalf("%v %v", got.Name, why)
	}
	// mild difference keeps the preferred skill
	tg = &Target{HasRes: true, Res: [6]int{0, 0, 50, 55, 60, 0}}
	if got, _ := PickByResist(order, tg, 100, nil); got.Name != "Blizzard" {
		t.Fatalf("%v", got.Name)
	}
	// immune preferred skill falls through as before
	tg = &Target{HasRes: true, Res: [6]int{0, 0, 0, 0, 100, 0}}
	if got, _ := PickByResist(order, tg, 100, nil); got.Name != "Fire Ball" {
		t.Fatalf("%v", got.Name)
	}
	if _, why := PickByResist(order, &Target{HasRes: true, Res: [6]int{0, 0, 100, 100, 100, 0}}, 100, nil); why != PickImmune {
		t.Fatalf("%v", why)
	}
}

func TestStoneSkinPrefersElement(t *testing.T) {
	order := []SkillSpec{sk("Zeal", data.Physical, 0), sk("Holy Fire", data.Fire, 0)}
	tg := &Target{Affixes: AffixStoneSkin}
	if f := DamageFactor(order[0], tg); f != 0.5 {
		t.Fatalf("%v", f)
	}
	tg = &Target{Affixes: AffixStoneSkin, HasRes: true, Res: [6]int{0, 0, 90, 0, 0, 0}}
	// physical 0.5 vs fire 0.1: keep physical
	if got, _ := PickByResist(order, tg, 10, nil); got.Name != "Zeal" {
		t.Fatalf("%v", got.Name)
	}
}

func TestImmuneWorkaround(t *testing.T) {
	cold := &Target{HasRes: true, Res: [6]int{0, 0, 30, 0, 150, 0}}
	w := ImmuneWorkaround(d2.Sorceress, cold)
	if len(w.Skills) == 0 || w.Skills[0] != "Chain Lightning" && w.Skills[0] != "Fire Ball" {
		t.Fatalf("%+v", w)
	}
	for _, s := range w.Skills {
		if s == "Blizzard" {
			t.Fatalf("cold skill offered against cold immune: %v", w.Skills)
		}
	}
	all := &Target{HasRes: true, Elite: true, Res: [6]int{0, 0, 100, 100, 100, 0}}
	if w := ImmuneWorkaround(d2.Sorceress, all); !w.Merc || !w.Skip || len(w.Skills) != 0 {
		t.Fatalf("%+v", w)
	}
	phys := &Target{HasRes: true, Res: [6]int{100, 0, 0, 0, 0, 0}}
	w = ImmuneWorkaround(d2.Paladin, phys)
	if len(w.Skills) == 0 || w.Skills[0] == "Zeal" {
		t.Fatalf("%+v", w)
	}
}

func TestResistPenaltyAndDanger(t *testing.T) {
	if ResistPenalty(d2.Normal) != 0 || ResistPenalty(d2.Nightmare) != -40 || ResistPenalty(d2.Hell) != -100 {
		t.Fatal("penalty")
	}
	r := PlayerResists{Fire: 80, Cold: 75, Lightning: 30, Poison: 0}
	if e := r.Effective(d2.Hell); e.Fire != -25 || e.Cold != -25 || e.Lightning != -70 || e.Poison != -100 {
		t.Fatalf("%+v", e)
	}
	ts := []Target{{Affixes: AffixLightningEnchanted}}
	if got := ResistDanger(1, ts, r, d2.Hell); got != 2 {
		t.Fatalf("%d", got)
	}
	if got := ResistDanger(3, ts, r, d2.Hell); got != 3 {
		t.Fatalf("cap %d", got)
	}
	ok := PlayerResists{Fire: 75, Cold: 75, Lightning: 75}
	if got := ResistDanger(1, ts, ok, d2.Nightmare); got != 1 {
		t.Fatalf("%d", got)
	}
}

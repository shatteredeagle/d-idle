package resource

import (
	"testing"

	"d2lod-bot/internal/d2"
)

var (
	bolt  = Skill{Name: "Fire Bolt", Cost: 3, Ready: true}
	ice   = Skill{Name: "Ice Blast", Cost: 6, Ready: true}
	staff = Skill{Name: "Staff", Melee: true, Ready: true}
)

func sorc() Input {
	return Input{Life: 60, MaxLife: 60, Mana: 45, MaxMana: 45, Class: d2.Sorceress, Level: 8, Gold: 300, LastTripGold: -1,
		Belt: Belt{HP: 3, MP: 1}, HasTP: true, Skills: []Skill{bolt, staff}}
}

func with(f func(*Input)) Input { in := sorc(); f(&in); return in }

func TestPreEngage(t *testing.T) {
	tu := DefaultTuning()
	cases := []struct {
		name string
		in   Input
		want PreAction
	}{
		{"full", sorc(), Proceed},
		{"hostiles on top", with(func(i *Input) { i.Hostiles = 2; i.Mana = 0 }), Proceed},
		{"low mana rests, no fallback", with(func(i *Input) { i.Mana = 12; i.PackSeen = 2; i.Skills = []Skill{bolt} }), RestHere},
		{"long rest skipped with staff", with(func(i *Input) { i.Mana = 12; i.PackSeen = 2 }), Proceed},
		{"long rest skipped with cheaper spell", with(func(i *Input) { i.Mana = 12; i.PackSeen = 2; i.Skills = []Skill{ice, bolt} }), Proceed},
		{"dangerous pack retreats", with(func(i *Input) { i.Mana = 12; i.PackSeen = 2; i.PackDanger = 2 }), RetreatRest},
		{"rest too long but melee fallback", with(func(i *Input) { i.Mana = 0; i.MaxMana = 400; i.PackSeen = 3 }), Proceed},
		{"rest too long, mana potion, no fallback", with(func(i *Input) {
			i.Mana = 2
			i.MaxMana = 400
			i.Skills = []Skill{bolt}
			i.PackSeen = 3
		}), DrinkMana},
		{"rest too long, nothing", with(func(i *Input) {
			i.Mana = 2
			i.MaxMana = 400
			i.Skills = []Skill{bolt}
			i.Belt.MP = 0
			i.PackSeen = 3
		}), RestHere},
		{"melee class ignores mana", with(func(i *Input) { i.Class = d2.Barbarian; i.Skills = []Skill{staff}; i.Mana = 0 }), Proceed},
		{"hurt, spare potion, dangerous", with(func(i *Input) { i.Life = 20; i.PackDanger = 1 }), DrinkLife},
		{"hurt, one potion, portal", with(func(i *Input) { i.Life = 20; i.Belt.HP = 1 }), TownHeal},
		{"hurt, no way back", with(func(i *Input) { i.Life = 15; i.Belt.HP = 1; i.HasTP = false }), Avoid},
		{"hurt but regenerating", with(func(i *Input) { i.Life = 30; i.LifeRegen = 2 }), RestHere},
		{"in town", with(func(i *Input) { i.InTown = true; i.Mana = 0 }), Proceed},
	}
	for _, c := range cases {
		if got := PreEngage(c.in, tu); got.Action != c.want {
			t.Errorf("%s: got %v (%s), want %v", c.name, got.Action, got.Reason, c.want)
		}
	}
	d := PreEngage(with(func(i *Input) { i.Mana = 12; i.PackSeen = 2; i.Skills = []Skill{bolt} }), tu)
	if d.WaitSec <= 0 || d.WaitSec > tu.RestMaxSec || d.TargetMana < 27 {
		t.Errorf("rest decision %+v", d)
	}
}

func TestAttackChoiceSwitchesCheapWhenPackOutlastsMana(t *testing.T) {
	tu := DefaultTuning()
	in := with(func(i *Input) { i.Skills = []Skill{ice, bolt}; i.Mana = 30; i.MaxMana = 45; i.Hostiles = 1 })
	if d := AttackChoice(in, tu); d.Skill.Name != "Ice Blast" {
		t.Fatalf("one monster, mana enough: %+v", d)
	}
	in.Hostiles = 6
	if d := AttackChoice(in, tu); d.Skill.Name != "Fire Bolt" {
		t.Fatalf("six monsters, mana short for the main spell: %+v", d)
	}
}

func TestAttackChoice(t *testing.T) {
	tu := DefaultTuning()
	cases := []struct {
		name  string
		in    Input
		kind  AttackKind
		skill string
	}{
		{"plenty", with(func(i *Input) { i.Skills = []Skill{ice, bolt, staff}; i.Hostiles = 2 }), Cast, "Ice Blast"},
		{"economy picks cheapest", with(func(i *Input) { i.Skills = []Skill{ice, bolt, staff}; i.Mana = 10; i.Hostiles = 2 }), Cast, "Fire Bolt"},
		{"cannot pay main, pays cheap", with(func(i *Input) { i.Skills = []Skill{ice, bolt}; i.Mana = 4; i.Hostiles = 1 }), Cast, "Fire Bolt"},
		{"empty: melee", with(func(i *Input) { i.Mana = 1; i.Hostiles = 2 }), Melee, "Staff"},
		{"empty, big pack: retreat", with(func(i *Input) { i.Mana = 1; i.Hostiles = 5 }), Retreat, ""},
		{"empty, no staff: wait", with(func(i *Input) { i.Mana = 1; i.Hostiles = 2; i.Skills = []Skill{bolt} }), Wait, ""},
		{"empty, hurt, no staff: retreat", with(func(i *Input) { i.Mana = 1; i.Hostiles = 2; i.Skills = []Skill{bolt}; i.Life = 15 }), Retreat, ""},
		{"empty, big weak pack: melee", with(func(i *Input) { i.Mana = 1; i.Hostiles = 5; i.WeakPack = true }), Melee, "Staff"},
		{"empty, big weak pack, no staff: wait", with(func(i *Input) { i.Mana = 1; i.Hostiles = 5; i.WeakPack = true; i.Skills = []Skill{bolt} }), Wait, ""},
		{"empty, weak but elite: retreat", with(func(i *Input) { i.Mana = 1; i.Hostiles = 5; i.WeakPack = true; i.PackDanger = 2 }), Retreat, ""},
		{"empty, elite: retreat", with(func(i *Input) { i.Mana = 1; i.Hostiles = 2; i.PackDanger = 2 }), Retreat, ""},
		{"free skill beats melee", with(func(i *Input) {
			i.Mana = 0
			i.Hostiles = 1
			i.Skills = []Skill{bolt, {Name: "Zap", Ready: true}, staff}
		}), Free, "Zap"},
		{"cooldown skipped", with(func(i *Input) { b := bolt; b.Ready = false; i.Skills = []Skill{b, ice}; i.Hostiles = 1 }), Cast, "Ice Blast"},
	}
	for _, c := range cases {
		got := AttackChoice(c.in, tu)
		if got.Kind != c.kind || (c.skill != "" && got.Skill.Name != c.skill) {
			t.Errorf("%s: got %v %q (%s)", c.name, got.Kind, got.Skill.Name, got.Reason)
		}
	}
}

func TestPotionPolicy(t *testing.T) {
	tu := DefaultTuning()
	cases := []struct {
		name string
		in   Input
		want Potion
	}{
		{"healthy", with(func(i *Input) { i.Hostiles = 2 }), PotionNone},
		{"life low", with(func(i *Input) { i.Hostiles = 2; i.Life = 30 }), PotionLife},
		{"life low out of fight, above engage line", with(func(i *Input) { i.Life = 30; i.MaxLife = 60 }), PotionNone},
		{"critical prefers rejuv", with(func(i *Input) { i.Hostiles = 2; i.Life = 10; i.Belt.RV = 1 }), PotionRejuv},
		{"critical heal", with(func(i *Input) { i.Hostiles = 2; i.Life = 10 }), PotionLife},
		{"mana low, rich", with(func(i *Input) { i.Hostiles = 2; i.Mana = 2; i.Skills = []Skill{bolt} }), PotionMana},
		{"mana low, poor with staff keeps potion", with(func(i *Input) { i.Hostiles = 2; i.Mana = 2; i.Gold = 40 }), PotionNone},
		{"mana low, poor but elite", with(func(i *Input) { i.Hostiles = 2; i.Mana = 2; i.Gold = 40; i.PackDanger = 2 }), PotionMana},
		{"mana ok for casts", with(func(i *Input) { i.Hostiles = 2; i.Mana = 11 }), PotionNone},
		{"barbarian never mana", with(func(i *Input) { i.Class = d2.Barbarian; i.Hostiles = 2; i.Mana = 0 }), PotionNone},
		{"no potions", with(func(i *Input) { i.Hostiles = 2; i.Life = 10; i.Belt = Belt{} }), PotionNone},
		{"town", with(func(i *Input) { i.InTown = true; i.Life = 5 }), PotionNone},
	}
	for _, c := range cases {
		if got := PotionPolicy(c.in, tu); got.Drink != c.want {
			t.Errorf("%s: got %v (%s), want %v", c.name, got.Drink, got.Reason, c.want)
		}
	}
}

func TestPlanFor(t *testing.T) {
	cnt := func(p BeltPlan, k Potion) (n int) {
		for _, c := range p.Cols {
			if c == k {
				n++
			}
		}
		return
	}
	if p := PlanFor(d2.Sorceress, 8); cnt(p, PotionMana) != 2 || p.PrefTier != 2 || p.MaxTier != 2 {
		t.Errorf("sorc 8: %+v", p)
	}
	if p := PlanFor(d2.Barbarian, 20); cnt(p, PotionMana) != 0 || p.PrefTier != 3 {
		t.Errorf("barb 20: %+v", p)
	}
	if p := PlanFor(d2.Paladin, 40); cnt(p, PotionRejuv) != 1 || p.PrefTier != 4 {
		t.Errorf("pal 40: %+v", p)
	}
	if p := PlanFor(d2.Sorceress, 70); cnt(p, PotionRejuv) != 2 || p.PrefTier != 5 {
		t.Errorf("sorc 70: %+v", p)
	}
	if p := PlanFor(d2.Sorceress, 1); p.PrefTier != 1 {
		t.Errorf("sorc 1: %+v", p)
	}
}

func TestPlanBuys(t *testing.T) {
	tu, pr := DefaultTuning(), DefaultPrices()
	b := PlanBuys(with(func(i *Input) { i.Belt = Belt{}; i.Gold = 5000; i.Level = 20 }), tu, pr)
	if b.TierHP != 3 || b.HP != 2 || b.MP != 2 || b.Cost > b.Spendable {
		t.Errorf("rich clvl 20 sorc: %+v", b)
	}
	b = PlanBuys(with(func(i *Input) { i.Belt = Belt{}; i.Gold = 100; i.Level = 20 }), tu, pr)
	if b.TierHP != 1 || b.HP == 0 || b.Cost > 70 {
		t.Errorf("poor clvl 20: %+v", b) // 100 - 30 reserve, minor potions only
	}
	b = PlanBuys(with(func(i *Input) { i.Belt = Belt{}; i.Gold = 100; i.Level = 20; i.HasTP = false }), tu, pr)
	if b.Spendable != 0 || b.HP != 0 {
		t.Errorf("no TP scroll eats the budget: %+v", b)
	}
	b = PlanBuys(with(func(i *Input) { i.Belt = Belt{HP: 2, MP: 2}; i.Gold = 5000 }), tu, pr)
	if b.HP != 0 || b.MP != 0 {
		t.Errorf("full belt: %+v", b)
	}
	// poor: healing before mana
	b = PlanBuys(with(func(i *Input) { i.Belt = Belt{}; i.Gold = 75; i.Level = 5 }), tu, pr)
	if b.HP < 1 || b.MP > b.HP {
		t.Errorf("poor priority: %+v", b)
	}
	b = PlanBuys(with(func(i *Input) { i.Belt = Belt{Rows: 2}; i.Gold = 2000; i.Level = 8 }), tu, pr)
	if b.HP != 4 {
		t.Errorf("two rows hold 4 healing potions: %+v", b)
	}
}

func TestTownTripTrigger(t *testing.T) {
	tu := DefaultTuning()
	cases := []struct {
		name         string
		in           Input
		goes, futile bool
		method       TripMethod
	}{
		{"fine", sorc(), false, false, NoTrip},
		{"no healing potions, gold", with(func(i *Input) { i.Belt.HP = 0 }), true, false, ByPortal},
		{"no healing potions, broke", with(func(i *Input) { i.Belt.HP = 0; i.Gold = 10 }), false, true, ByPortal},
		{"futile: last trip could not fix it", with(func(i *Input) { i.Belt.HP = 0; i.Gold = 120; i.LastTripGold = 100 }), false, true, ByPortal},
		{"gold grew since", with(func(i *Input) { i.Belt.HP = 0; i.Gold = 400; i.LastTripGold = 100 }), true, false, ByPortal},
		{"caster out of mana potions and starved", with(func(i *Input) {
			i.Belt.MP = 0
			i.Belt.HP = 1
			i.MaxMana = 400
			i.Mana = 20
		}), true, false, ByPortal},
		{"walk when no portal", with(func(i *Input) { i.Belt.HP = 0; i.HasTP = false; i.NearTown = true }), true, false, ByWalk},
		{"no way", with(func(i *Input) { i.Belt.HP = 0; i.HasTP = false }), false, false, NoTrip},
		{"free heal even when broke", with(func(i *Input) { i.Life = 20; i.Gold = 0 }), true, false, ByPortal},
		{"fight on", with(func(i *Input) { i.Belt.HP = 0; i.Hostiles = 1 }), false, false, NoTrip},
		{"poor clvl 4 sorc, no kit, hurt, in the Den", with(func(i *Input) {
			i.Belt = Belt{}
			i.HasTP = false
			i.Gold = 0
			i.Level, i.Life, i.MaxLife = 4, 27, 73
			i.NearTown = NearTownArea(8)
		}), true, false, ByWalk},
		{"poor, no kit, out of mana, healthy", with(func(i *Input) {
			i.Belt = Belt{}
			i.HasTP = false
			i.Gold = 0
			i.Level, i.Mana, i.MaxMana = 4, 1, 30
			i.Skills = []Skill{bolt}
			i.NearTown = true
		}), true, false, ByWalk},
		{"in town", with(func(i *Input) { i.Belt.HP = 0; i.InTown = true }), false, false, NoTrip},
	}
	for _, c := range cases {
		got := TownTripTrigger(c.in, tu)
		if got.Go != c.goes || got.Futile != c.futile || (c.method != NoTrip && got.Method != c.method) {
			t.Errorf("%s: got %+v", c.name, got)
		}
	}
}

func TestAbortFight(t *testing.T) {
	tu := DefaultTuning()
	fight := func(f func(*Input)) Input {
		return with(func(i *Input) { i.Hostiles = 3; f(i) })
	}
	cases := []struct {
		name string
		in   Input
		want AbortKind
	}{
		{"fine", fight(func(i *Input) {}), Continue},
		{"chicken, potion left", fight(func(i *Input) { i.Life = 10 }), Continue},
		{"chicken line, none left, portal", fight(func(i *Input) { i.Life = 15; i.Belt = Belt{} }), PortalOut},
		{"far below chicken, none left", fight(func(i *Input) { i.Life = 5; i.Belt = Belt{} }), LeaveGame},
		{"no portal chicken", fight(func(i *Input) { i.Life = 14; i.Belt = Belt{}; i.HasTP = false }), LeaveGame},
		{"life and mana gone, no melee", fight(func(i *Input) { i.Life = 25; i.Belt = Belt{}; i.Mana = 0; i.Skills = []Skill{bolt} }), PortalOut},
		{"life gone, mana ok", fight(func(i *Input) { i.Life = 25; i.Belt = Belt{} }), Continue},
		{"no potions, no mana, melee, still ok", fight(func(i *Input) { i.Life = 40; i.Belt = Belt{}; i.Mana = 0 }), Continue},
		{"no portal, no potion: flee", fight(func(i *Input) {
			i.Life = 25
			i.Belt = Belt{}
			i.Mana = 0
			i.Skills = []Skill{bolt}
			i.HasTP = false
			i.NearTown = true
		}), Flee},
		{"merc dead, pack, hurt", fight(func(i *Input) { i.MercKnown = true; i.MercLifePct = 0; i.Life = 30 }), PortalOut},
		{"merc dead, healthy", fight(func(i *Input) { i.MercKnown = true; i.MercLifePct = 0 }), Continue},
		{"merc unknown", fight(func(i *Input) { i.Life = 30 }), Continue},
		{"hell chicken line is higher", fight(func(i *Input) { i.Diff = d2.Hell; i.Life = 18; i.Belt = Belt{} }), PortalOut},
		{"no hostiles", with(func(i *Input) { i.Life = 5 }), Continue},
	}
	for _, c := range cases {
		if got := AbortFight(c.in, tu); got.Kind != c.want {
			t.Errorf("%s: got %v (%s), want %v", c.name, got.Kind, got.Reason, c.want)
		}
	}
}

func TestKnobs(t *testing.T) {
	seen := map[string]bool{}
	for _, k := range Knobs() {
		if seen[k.Key] || k.Default < k.Min || k.Default > k.Max || k.Min >= k.Max {
			t.Errorf("bad knob %+v", k)
		}
		seen[k.Key] = true
	}
	if len(seen) < 20 {
		t.Errorf("only %d knobs", len(seen))
	}
}

func TestRegenModel(t *testing.T) {
	in := Input{Mana: 0, MaxMana: 100}
	if s := in.RestSecs(50); s < 49 || s > 51 {
		t.Errorf("rest 50%%: %.1fs", s)
	}
	in.ManaRegenB = 100
	if s := in.RestSecs(50); s < 24 || s > 26 {
		t.Errorf("rest 50%% with +100%%: %.1fs", s)
	}
	if (Input{}).LifeRestSecs(10) < 1e8 {
		t.Error("no life regen should never finish")
	}
}

func TestAbortFightFleesWithoutPotionOrPortal(t *testing.T) {
	tu := DefaultTuning()
	in := Input{Life: 40, MaxLife: 100, Hostiles: 2, Class: d2.Sorceress, Skills: []Skill{{Name: "Attack", Melee: true, Ready: true}}}
	if got := AbortFight(in, tu); got.Kind != Flee {
		t.Fatalf("no potion no portal at 40%%: %v", got)
	}
	in.Life = 5 // at the real critical line the last resort stays the chicken
	if got := AbortFight(in, tu); got.Kind != LeaveGame {
		t.Fatalf("critical: %v", got)
	}
	in.Life = 40
	in.HasTP = true
	if got := AbortFight(in, tu); got.Kind == Flee {
		t.Fatalf("a portal holder must not flee: %v", got)
	}
	in.HasTP = false
	in.Belt.HP = 1
	if got := AbortFight(in, tu); got.Kind == Flee {
		t.Fatalf("a potion holder must not flee: %v", got)
	}
}

func TestPovertyCheck(t *testing.T) {
	tu := DefaultTuning() // kit 100 + 3*30 = 190, margin 60
	if c := KitCost(tu); c != 190 {
		t.Fatalf("kit cost %d", c)
	}
	in := Input{Gold: 0}
	if p := PovertyCheck(in, tu, false); !p.On || p.Missing != 190 {
		t.Fatalf("broke: %+v", p)
	}
	in.Gold = 190
	if p := PovertyCheck(in, tu, false); p.On {
		t.Fatalf("affordable should not enter: %+v", p)
	}
	if p := PovertyCheck(in, tu, true); !p.On {
		t.Fatalf("hysteresis: still on below cost+margin: %+v", p)
	}
	in.Gold = 250
	if p := PovertyCheck(in, tu, true); p.On {
		t.Fatalf("should leave at cost+margin: %+v", p)
	}
	in.Gold, in.HasTP, in.Belt.HP = 0, true, 3
	if p := PovertyCheck(in, tu, true); p.On {
		t.Fatalf("complete kit is never poor: %+v", p)
	}
}

func TestBuyKitScrollFirst(t *testing.T) {
	tu := DefaultTuning()
	if k := BuyKit(Input{Gold: 120}, tu); !k.TP || k.HP != 0 {
		t.Fatalf("120 gold: %+v", k)
	}
	if k := BuyKit(Input{Gold: 50}, tu); k.TP || k.HP != 1 {
		t.Fatalf("50 gold cannot afford the scroll: %+v", k)
	}
	if k := BuyKit(Input{Gold: 400}, tu); !k.TP || k.HP != 3 {
		t.Fatalf("400 gold: %+v", k)
	}
}

func TestPovertyAdviceSellsThenPlaysLean(t *testing.T) {
	tu := DefaultTuning()
	in := with(func(i *Input) { i.Gold = 10; i.HasTP = true; i.Belt = Belt{}; i.Sellable = 5 })
	p := PovertyCheck(in, tu, false)
	if a := PovertyAdvice(in, p, 4); a.Action != SellTrip {
		t.Fatalf("5 sellable items: %+v", a)
	}
	in.Sellable = 2
	if a := PovertyAdvice(in, p, 4); a.Action != PlayLean {
		t.Fatalf("2 sellable items: %+v", a)
	}
	in.Sellable, in.IdleTrips = 9, 2
	if a := PovertyAdvice(in, p, 4); a.Action != PlayLean {
		t.Fatalf("two idle trips already: %+v", a)
	}
	if a := PovertyAdvice(sorc(), PovertyCheck(sorc(), tu, false), 4); a.Action != PovertyNone {
		t.Fatalf("not poor: %+v", a)
	}
}

func TestHealOnlyTripSuppressedAfterIdleTrips(t *testing.T) {
	tu := DefaultTuning()
	in := with(func(i *Input) { i.Life = 27; i.MaxLife = 60; i.Belt = Belt{HP: 3, MP: 1}; i.Gold = 5 })
	if tr := TownTripTrigger(in, tu); !tr.Go {
		t.Skipf("baseline does not trip (%s); nothing to suppress", tr.Reason)
	}
	in.IdleTrips = 2
	if tr := TownTripTrigger(in, tu); tr.Go {
		t.Fatalf("heal-only trip after idle trips: %+v", tr)
	}
}

func TestNearTownArea(t *testing.T) {
	for area, want := range map[int]bool{2: true, 8: true, 3: true, 1: false, 17: false, 5: false} {
		if NearTownArea(area) != want {
			t.Errorf("area %d: want %v", area, want)
		}
	}
}

// REJUVUSE: against a champion/boss pack a belt rejuvenation is drunk at the boss line (50 %), against trash only at 35 %.
func TestBossRejuvLine(t *testing.T) {
	tn := DefaultTuning()
	in := Input{Life: 45, MaxLife: 100, Belt: Belt{HP: 2, RV: 1, Slots: 4}, Hostiles: 3, Level: 20, Class: d2.Barbarian}
	if d := PotionPolicy(in, tn); d.Drink == PotionRejuv {
		t.Errorf("trash at 45 %% drank rejuvenation: %+v", d)
	}
	in.PackDanger = 2
	if d := PotionPolicy(in, tn); d.Drink != PotionRejuv {
		t.Errorf("boss at 45 %% did not drink rejuvenation: %+v", d)
	}
	tn.DrinkRejuvBossPct = 0
	if d := PotionPolicy(in, tn); d.Drink == PotionRejuv {
		t.Errorf("knob 0 must disable the boss line: %+v", d)
	}
}

// MANABACKOFF: a weak caster in a big pack at danger 2 ran in circles ("backing off" x200 on the Fresha soak); after
// BackoffMaxSteps it swings the weapon, but a deadly pack (danger 3) or low life still flees.
func TestBackoffCapFallsBackToMelee(t *testing.T) {
	tu := DefaultTuning()
	in := Input{Life: 100, MaxLife: 100, Mana: 1, MaxMana: 45, Hostiles: 9, PackDanger: 2,
		Skills: []Skill{{Name: "Charged Bolt", Cost: 3, Ready: true}, {Name: "Attack", Melee: true, Ready: true}}}
	if d := AttackChoice(in, tu); d.Kind != Retreat {
		t.Fatalf("before the cap: %v, want retreat", d)
	}
	in.BackoffSteps = tu.BackoffMaxSteps
	if d := AttackChoice(in, tu); d.Kind != Melee {
		t.Errorf("at the cap: %v, want melee", d)
	}
	in.PackDanger = 3
	if d := AttackChoice(in, tu); d.Kind != Retreat {
		t.Errorf("danger 3: %v, want retreat", d)
	}
	in.PackDanger = 2
	in.Life = 30
	if d := AttackChoice(in, tu); d.Kind != Retreat {
		t.Errorf("low life: %v, want retreat", d)
	}
	tu.BackoffMaxSteps = 0
	in.Life = 100
	in.BackoffSteps = 99
	if d := AttackChoice(in, tu); d.Kind != Retreat {
		t.Errorf("cap off: %v, want retreat", d)
	}
}

func TestChampionEarlyTightensDrinkLines(t *testing.T) {
	tu := DefaultTuning()
	champ := func(lvl int, diff d2.Difficulty, life int, rv int) Input {
		return with(func(i *Input) {
			i.Hostiles, i.PackDanger, i.ChampionPack, i.Level, i.Diff = 2, 1, true, lvl, diff
			i.Life, i.MaxLife, i.Belt.RV = life, 100, rv
		})
	}
	// healing line 55 -> 65 against a champion in early Normal
	if got := PotionPolicy(champ(10, d2.Normal, 62, 0), tu); got.Drink != PotionLife {
		t.Fatalf("clvl 10 champion at 62%%: %v", got)
	}
	// rejuv line 50 -> 60
	if got := PotionPolicy(champ(10, d2.Normal, 58, 1), tu); got.Drink != PotionRejuv {
		t.Fatalf("clvl 10 champion at 58%% with rejuv: %v", got)
	}
	for name, in := range map[string]Input{
		"clvl 15":     champ(15, d2.Normal, 62, 0),
		"nightmare":   champ(10, d2.Nightmare, 62, 0),
		"no champion": with(func(i *Input) { i.Hostiles, i.Level, i.Life, i.MaxLife = 2, 10, 62, 100 }),
	} {
		if got := PotionPolicy(in, tu); got.Drink != PotionNone {
			t.Errorf("%s: unchanged line expected, drank %v", name, got.Drink)
		}
	}
	tu.ChampionEarlyPct = 0
	if got := PotionPolicy(champ(10, d2.Normal, 62, 0), tu); got.Drink != PotionNone {
		t.Fatalf("knob 0 = off: %v", got)
	}
}

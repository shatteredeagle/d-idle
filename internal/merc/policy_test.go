package merc

import "testing"

func ctx() Ctx {
	return Ctx{Gold: 3000, Kit: 300, Act: 2, PlayerClass: "Sorceress", HaveTP: true, HavePotions: true, Knobs: DefaultKnobs(), FirstLutVisit: true}
}

func TestFormulas(t *testing.T) {
	for _, c := range []struct{ clvl, r, want int }{{8, 0, 3}, {8, 4, 7}, {3, 0, 2}, {20, 2, 17}} {
		if got := OfferedLevel(c.clvl, c.r); got != c.want {
			t.Errorf("OfferedLevel(%d,%d)=%d want %d", c.clvl, c.r, got, c.want)
		}
	}
	for _, c := range []struct{ base, off, row, want int }{{350, 9, 9, 350}, {350, 11, 9, 455}, {350, 5, 9, 350}, {100, 10, 0, 250}} {
		if got := HireCost(c.base, c.off, c.row); got != c.want {
			t.Errorf("HireCost(%v)=%d want %d", c, got, c.want)
		}
	}
	for _, c := range []struct{ lvl, want int }{{0, 0}, {9, 600}, {10, 750}, {11, 900}, {60, 27000}, {200, 50000}} {
		if got := ReviveCost(c.lvl); got != c.want {
			t.Errorf("ReviveCost(%d)=%d want %d", c.lvl, got, c.want)
		}
	}
}

func TestShouldClaimFreeRogue(t *testing.T) {
	ok := FreeRogueCtx{AtKashya: true, BloodRavenDone: true}
	for _, c := range []struct {
		name string
		s    MercState
		c    FreeRogueCtx
		do   bool
		why  string
	}{
		{"all good", MercState{Known: true}, ok, true, ReasonOK},
		{"unknown state still claims", MercState{}, ok, true, ReasonOK},
		{"merc alive", MercState{Known: true, Owned: true, Alive: true}, ok, false, ReasonOwned},
		{"merc dead still forfeits", MercState{Known: true, Owned: true}, ok, false, ReasonOwned},
		{"not at kashya", MercState{Known: true}, FreeRogueCtx{BloodRavenDone: true}, false, ReasonNotAtKashya},
		{"quest open", MercState{Known: true}, FreeRogueCtx{AtKashya: true}, false, ReasonQuestOpen},
		{"nightmare", MercState{Known: true}, FreeRogueCtx{AtKashya: true, BloodRavenDone: true, Difficulty: Nightmare}, false, ReasonNotNormal},
	} {
		if d := ShouldClaimFreeRogue(c.s, c.c); d.Do != c.do || d.Reason != c.why {
			t.Errorf("%s: %+v", c.name, d)
		}
	}
}

func TestAuraPrefs(t *testing.T) {
	for _, c := range []struct {
		class string
		d     Difficulty
		bow   bool
		want  string
	}{
		{"Sorceress", Normal, false, "Defiance,Prayer"},
		{"Necromancer", Hell, false, "Defiance,Prayer"},
		{"Paladin", Normal, false, "Defiance,Prayer"},
		{"Amazon", Normal, false, "Defiance,Prayer"},
		{"Amazon", Normal, true, "Defiance,Prayer,Blessed Aim"},
		{"Barbarian", Nightmare, true, "Might,Defiance,Prayer"},
		{"Druid", Nightmare, false, "Holy Freeze,Defiance,Prayer"},
		{"Amazon", Nightmare, true, "Might,Defiance,Prayer,Blessed Aim"},
		{"Assassin", Hell, false, "Defiance,Prayer"},
	} {
		got := ""
		for i, a := range AuraPrefs(c.class, c.d, c.bow) {
			if i > 0 {
				got += ","
			}
			got += a
		}
		if got != c.want {
			t.Errorf("%v: %s want %s", c, got, c.want)
		}
	}
	if AuraRank([]string{"Defiance"}, "defiance") != 0 || AuraRank([]string{"Defiance"}, "") != -1 {
		t.Error("AuraRank")
	}
}

func TestShouldHire(t *testing.T) {
	none := MercState{Known: true}
	rogue := HireOffer{Class: ClassRogue, Aura: "Cold Arrow", Level: 5, Price: 400}
	desert := HireOffer{Class: ClassDesert, Aura: "Defiance", Level: 12, Price: 600}
	mut := func(f func(*Ctx)) Ctx { c := ctx(); f(&c); return c }
	for _, c := range []struct {
		name string
		o    HireOffer
		s    MercState
		c    Ctx
		do   bool
		why  string
	}{
		{"rogue before skip", rogue, none, ctx(), false, ReasonFreePending},
		{"rogue fallback", rogue, none, mut(func(c *Ctx) { c.BloodRavenSkipped = true }), true, ReasonOK},
		{"rogue fallback poor", rogue, none, mut(func(c *Ctx) { c.BloodRavenSkipped = true; c.Gold = 1400 }), false, ReasonGold},
		{"rogue exact 3x", rogue, none, mut(func(c *Ctx) { c.BloodRavenSkipped = true; c.Gold = 1500 }), true, ReasonOK},
		{"rogue fire arrow", HireOffer{Class: ClassRogue, Aura: "Fire Arrow", Price: 100}, none, mut(func(c *Ctx) { c.BloodRavenSkipped = true }), false, ReasonAura},
		{"rogue free only knob", rogue, none, mut(func(c *Ctx) { c.BloodRavenSkipped = true; c.Knobs.FreeRogueOnly = true }), false, ReasonFreeOnly},
		{"owned", desert, MercState{Known: true, Owned: true}, ctx(), false, ReasonOwned},
		{"desert defiance", desert, none, ctx(), true, ReasonOK},
		{"desert might in normal", HireOffer{Class: ClassDesert, Aura: "Might", Price: 100}, none, ctx(), false, ReasonAura},
		{"desert might nightmare barb", HireOffer{Class: ClassDesert, Aura: "Might", Price: 100}, none, mut(func(c *Ctx) { c.PlayerClass = "Barbarian"; c.Difficulty = Nightmare }), true, ReasonOK},
		{"desert might nightmare caster ok? no", HireOffer{Class: ClassDesert, Aura: "Might", Price: 100}, none, mut(func(c *Ctx) { c.Difficulty = Nightmare }), false, ReasonAura},
		{"blessed aim sorc", HireOffer{Class: ClassDesert, Aura: "Blessed Aim", Price: 100}, none, ctx(), false, ReasonAura},
		{"blessed aim bow amazon", HireOffer{Class: ClassDesert, Aura: "Blessed Aim", Price: 100}, none, mut(func(c *Ctx) { c.PlayerClass = "Amazon"; c.BowOrJavelin = true }), true, ReasonOK},
		{"kit eats gold", desert, none, mut(func(c *Ctx) { c.Gold = 2000; c.Kit = 300 }), false, ReasonGold},
		{"zero price", HireOffer{Class: ClassDesert, Aura: "Defiance"}, none, ctx(), false, ReasonGold},
		{"iron wolf", HireOffer{Class: ClassIronWolf, Aura: "Defiance", Price: 10}, none, ctx(), false, ReasonClass},
	} {
		if d := ShouldHire(c.o, c.s, c.c); d.Do != c.do || d.Reason != c.why {
			t.Errorf("%s: %+v", c.name, d)
		}
	}
}

func TestShouldReplace(t *testing.T) {
	gear := &Item{Name: "Skull Cap"}
	rogue := MercState{Known: true, Owned: true, Alive: true, Class: ClassRogue}
	rogueGear := rogue
	rogueGear.Gear[SlotHelm] = gear
	desertPrayer := MercState{Known: true, Owned: true, Alive: true, Class: ClassDesert, Aura: "Prayer"}
	desertDef := desertPrayer
	desertDef.Aura = "Defiance"
	desertUnk := desertPrayer
	desertUnk.Aura = ""
	desertThorns := desertPrayer
	desertThorns.Aura = "Thorns"
	deadRogue := rogueGear
	deadRogue.Alive = false
	offer := HireOffer{Class: ClassDesert, Aura: "Defiance", Price: 600}
	mut := func(f func(*Ctx)) Ctx { c := ctx(); f(&c); return c }
	for _, c := range []struct {
		name  string
		cur   MercState
		o     HireOffer
		c     Ctx
		do    bool
		why   string
		strip bool
	}{
		{"rogue no gear", rogue, offer, ctx(), true, ReasonOK, false},
		{"rogue with gear strips", rogueGear, offer, ctx(), true, ReasonOK, true},
		{"dead merc, nothing to strip", deadRogue, offer, ctx(), true, ReasonOK, false},
		{"prayer to defiance", desertPrayer, offer, ctx(), true, ReasonOK, false},
		{"defiance to defiance", desertDef, offer, ctx(), false, ReasonNoBetter, false},
		{"unknown current aura", desertUnk, offer, ctx(), false, ReasonNoBetter, false},
		{"unacceptable current aura", desertThorns, offer, ctx(), true, ReasonOK, false},
		{"not owned", MercState{Known: true}, offer, ctx(), false, ReasonNotOwned, false},
		{"unknown", MercState{}, offer, ctx(), false, ReasonUnknown, false},
		{"knob off", rogue, offer, mut(func(c *Ctx) { c.Knobs.ReplaceAtAct2 = false }), false, ReasonKnobOff, false},
		{"act 1", rogue, offer, mut(func(c *Ctx) { c.Act = 1 }), false, ReasonNotAct2, false},
		{"second visit", rogue, offer, mut(func(c *Ctx) { c.FirstLutVisit = false }), false, ReasonNotFirstVisit, false},
		{"gold 3x after kit", rogue, offer, mut(func(c *Ctx) { c.Gold = 2099 }), false, ReasonGold, false},
		{"gold exactly 3x after kit", rogue, offer, mut(func(c *Ctx) { c.Gold = 2100 }), true, ReasonOK, false},
		{"rogue offer", rogue, HireOffer{Class: ClassRogue, Aura: "Cold Arrow", Price: 1}, ctx(), false, ReasonClass, false},
		{"bad aura", rogue, HireOffer{Class: ClassDesert, Aura: "Might", Price: 1}, ctx(), false, ReasonAura, false},
		{"nightmare might upgrades defiance", desertDef, HireOffer{Class: ClassDesert, Aura: "Might", Price: 900},
			mut(func(c *Ctx) { c.PlayerClass = "Paladin"; c.Difficulty = Nightmare }), true, ReasonOK, false},
	} {
		d := ShouldReplace(c.cur, c.o, c.c)
		if d.Do != c.do || d.Reason != c.why || d.StripFirst != c.strip {
			t.Errorf("%s: %+v", c.name, d)
		}
	}
}

func TestShouldRevive(t *testing.T) {
	dead := MercState{Known: true, Owned: true, Level: 10} // price 750
	mut := func(f func(*Ctx)) Ctx { c := ctx(); f(&c); return c }
	for _, c := range []struct {
		name  string
		s     MercState
		c     Ctx
		price int
		do    bool
		why   string
		cost  int
	}{
		{"ok", dead, mut(func(c *Ctx) { c.Gold = 1500 }), 0, true, ReasonOK, 750},
		{"gold short", dead, mut(func(c *Ctx) { c.Gold = 1499 }), 0, false, ReasonGold, 750},
		{"quote overrides formula", dead, mut(func(c *Ctx) { c.Gold = 1500 }), 800, false, ReasonGold, 800},
		{"cap", dead, mut(func(c *Ctx) { c.RevivesLastHour = 2 }), 0, false, ReasonCap, 750},
		{"one more allowed", dead, mut(func(c *Ctx) { c.RevivesLastHour = 1 }), 0, true, ReasonOK, 750},
		{"no tp", dead, mut(func(c *Ctx) { c.HaveTP = false }), 0, false, ReasonKit, 750},
		{"no potions", dead, mut(func(c *Ctx) { c.HavePotions = false }), 0, false, ReasonKit, 750},
		{"3 deaths still normal", dead, mut(func(c *Ctx) { c.DeathsLastHour = 3; c.Gold = 1500 }), 0, true, ReasonOK, 750},
		{"4 deaths need 5x", dead, mut(func(c *Ctx) { c.DeathsLastHour = 4; c.Gold = 3749 }), 0, false, ReasonStopped, 750},
		{"4 deaths 5x ok", dead, mut(func(c *Ctx) { c.DeathsLastHour = 4; c.Gold = 3750 }), 0, true, ReasonOK, 750},
		{"alive", MercState{Known: true, Owned: true, Alive: true, Level: 10}, ctx(), 0, false, ReasonAlive, 0},
		{"not owned", MercState{Known: true}, ctx(), 0, false, ReasonNotOwned, 0},
		{"unknown", MercState{}, ctx(), 0, false, ReasonUnknown, 0},
		{"level unknown", MercState{Known: true, Owned: true}, ctx(), 0, false, ReasonNoLevel, 0},
		{"level unknown but quoted", MercState{Known: true, Owned: true}, ctx(), 600, true, ReasonOK, 600},
		{"huge price", MercState{Known: true, Owned: true, Level: 200}, mut(func(c *Ctx) { c.Gold = 99999 }), 0, false, ReasonGold, 50000},
	} {
		d, cost := ShouldRevive(c.s, c.c, c.price)
		if d.Do != c.do || d.Reason != c.why || cost != c.cost {
			t.Errorf("%s: %+v cost %d", c.name, d, cost)
		}
	}
}

func TestMercDied(t *testing.T) {
	for _, c := range []struct{ was, now, town, leave bool }{
		{true, false, false, true}, {true, false, true, false}, {false, false, false, false}, {true, true, false, false}, {false, true, false, false},
	} {
		if d := MercDied(c.was, c.now, c.town); d.LeaveArea != c.leave || d.Do != c.leave {
			t.Errorf("%+v: %+v", c, d)
		}
	}
}

func TestMercAliveGate(t *testing.T) {
	dead := MercState{Known: true, Owned: true}
	alive := MercState{Known: true, Owned: true, Alive: true}
	for _, c := range []struct {
		name  string
		s     MercState
		gold  int
		floor int
		want  int
	}{
		{"dead poor", dead, 999, 15, 17}, {"dead exactly 1000", dead, 1000, 15, 15}, {"dead rich", dead, 5000, 15, 15},
		{"alive poor", alive, 0, 15, 15}, {"unknown", MercState{}, 0, 15, 15}, {"none owned", MercState{Known: true}, 0, 15, 15},
	} {
		if got := MercAliveGate(c.floor, c.s, c.gold); got != c.want {
			t.Errorf("%s: %d want %d", c.name, got, c.want)
		}
	}
}

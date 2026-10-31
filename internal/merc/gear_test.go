package merc

import "testing"

func owned(class, level int) MercState {
	return MercState{Known: true, Owned: true, Alive: true, Class: class, Level: level, Str: 60, Dex: 50, StatsKnown: true, LifePct: 100}
}

func cap(def int) Item {
	return Item{Name: "Cap", Kind: KindHelm, Identified: true, Defense: def, W: 2, H: 2}
}

func TestScore(t *testing.T) {
	a := Item{Defense: 50}
	if Score(a, Normal) != 50 {
		t.Error("defense only")
	}
	if Score(Item{Defense: 50, Life: 10}, Normal) != 80 {
		t.Error("life weighs 3")
	}
	if Score(Item{DmgMin: 10, DmgMax: 20}, Normal) != 30 {
		t.Error("damage")
	}
	if Score(Item{Defense: 10, Res: 20}, Normal) != 10 || Score(Item{Defense: 10, Res: 20}, Hell) != 50 {
		t.Error("res only in hell")
	}
	if got := Score(Item{Defense: 100, Ethereal: true}, Normal); got != 105 {
		t.Errorf("ethereal %v", got)
	}
	if Score(Item{Sockets: 2}, Normal) != 10 {
		t.Error("sockets")
	}
}

func TestAllowed(t *testing.T) {
	bow, spear, sword, shield := Item{Kind: KindBow}, Item{Kind: KindSpear}, Item{Kind: KindSword1H}, Item{Kind: KindShield}
	for _, c := range []struct {
		class int
		slot  Slot
		it    Item
		want  bool
	}{
		{ClassRogue, SlotWeapon, bow, true}, {ClassRogue, SlotWeapon, spear, false}, {ClassRogue, SlotWeapon, sword, false},
		{ClassDesert, SlotWeapon, spear, true}, {ClassDesert, SlotWeapon, bow, false},
		{ClassIronWolf, SlotWeapon, sword, true}, {ClassIronWolf, SlotShield, shield, true}, {ClassDesert, SlotShield, shield, false},
		{ClassBarb, SlotWeapon, sword, true}, {ClassBarb, SlotWeapon, Item{Kind: KindAxe1H}, false},
		{ClassRogue, SlotHelm, Item{Kind: KindHelm}, true}, {ClassRogue, SlotHelm, Item{Kind: KindArmor}, false},
		{ClassDesert, SlotArmor, Item{Kind: KindArmor}, true}, {ClassDesert, SlotArmor, Item{Kind: KindHelm}, false},
		{999, SlotWeapon, bow, false},
	} {
		if got := Allowed(c.class, c.slot, c.it); got != c.want {
			t.Errorf("%+v: %v", c, got)
		}
	}
}

func TestMeetsReq(t *testing.T) {
	m := owned(ClassRogue, 10)
	base := Item{Identified: true, ReqLevel: 10, ReqStr: 60, ReqDex: 50}
	noStats := m
	noStats.StatsKnown = false
	for _, c := range []struct {
		name   string
		it     Item
		m      MercState
		strDex bool
		ok     bool
		why    string
	}{
		{"exact", base, m, true, true, "ok"},
		{"level", Item{Identified: true, ReqLevel: 11}, m, false, false, "level"},
		{"str", Item{Identified: true, ReqStr: 61}, m, true, false, "str"},
		{"str ignored", Item{Identified: true, ReqStr: 61}, m, false, true, "ok"},
		{"dex", Item{Identified: true, ReqDex: 51}, m, true, false, "dex"},
		{"stats unknown fails closed", base, noStats, true, false, "stats unknown"},
		{"stats unknown level only", base, noStats, false, true, "ok"},
		{"unidentified", Item{}, m, false, false, "unidentified"},
		{"broken", Item{Identified: true, Broken: true}, m, false, false, "broken"},
	} {
		if r := MeetsReq(c.it, c.m, c.strDex); r.OK != c.ok || r.Reason != c.why {
			t.Errorf("%s: %+v", c.name, r)
		}
	}
}

func TestEquipIfBetter(t *testing.T) {
	k := DefaultKnobs()
	m := owned(ClassDesert, 10)
	worn := cap(50)
	m.Gear[SlotHelm] = &worn
	for _, c := range []struct {
		name string
		m    MercState
		slot Slot
		cand Item
		do   bool
		why  string
		gain int
	}{
		{"empty slot", owned(ClassDesert, 10), SlotHelm, cap(30), true, "empty slot", 0},
		{"empty worthless", owned(ClassDesert, 10), SlotHelm, cap(0), false, "worthless", 0},
		{"gain 10 pct", m, SlotHelm, cap(55), true, "better by 10%", 10},
		{"gain 8 pct", m, SlotHelm, cap(54), false, "gain 8% < 10%", 8},
		{"worse", m, SlotHelm, cap(40), false, "gain -20% < 10%", -20},
		{"much better", m, SlotHelm, cap(100), true, "better by 100%", 100},
		{"wrong slot kind", m, SlotArmor, cap(100), false, "not allowed", 0},
		{"unidentified", m, SlotHelm, Item{Kind: KindHelm, Defense: 99}, false, "unidentified", 0},
		{"level too high", m, SlotHelm, Item{Kind: KindHelm, Identified: true, Defense: 99, ReqLevel: 30}, false, "level", 0},
		{"str too high", m, SlotHelm, Item{Kind: KindHelm, Identified: true, Defense: 99, ReqStr: 90}, false, "str", 0},
		{"not owned", MercState{Known: true}, SlotHelm, cap(99), false, ReasonUnknown, 0},
		{"life wins", m, SlotHelm, Item{Kind: KindHelm, Identified: true, Defense: 30, Life: 15}, true, "better by 50%", 50},
	} {
		ch := EquipIfBetter(c.m, c.slot, c.cand, Normal, k)
		if ch.Do != c.do || ch.Reason != c.why || ch.GainPct != c.gain {
			t.Errorf("%s: %+v", c.name, ch)
		}
	}
	// the rule moves with the knob
	k.SwapGainPct = 50
	if EquipIfBetter(m, SlotHelm, cap(60), Normal, k).Do {
		t.Error("20% gain passes a 50% rule")
	}
	// worn item worth nothing is replaced by anything of value
	z := cap(0)
	m.Gear[SlotHelm] = &z
	if ch := EquipIfBetter(m, SlotHelm, cap(1), Normal, DefaultKnobs()); !ch.Do {
		t.Errorf("%+v", ch)
	}
}

func TestPlanStrip(t *testing.T) {
	m := owned(ClassDesert, 10)
	if p := PlanStrip(m, 40); len(p.Steps) != 0 || p.CellsNeeded != 0 || p.StashCells != 0 {
		t.Errorf("empty merc: %+v", p)
	}
	h, a, w := Item{Name: "Helm", W: 2, H: 2}, Item{Name: "Armor", W: 2, H: 3}, Item{Name: "Polearm", W: 2, H: 4}
	m.Gear[SlotHelm], m.Gear[SlotArmor], m.Gear[SlotWeapon] = &h, &a, &w
	for _, c := range []struct{ free, stash int }{{40, 0}, {18, 0}, {10, 8}, {0, 18}, {-1, 18}} {
		p := PlanStrip(m, c.free)
		if p.CellsNeeded != 18 || p.StashCells != c.stash || len(p.Steps) != 3 {
			t.Errorf("free %d: %+v", c.free, p)
		}
		if p.Steps[0].Slot != SlotWeapon || p.Steps[2].Slot != SlotHelm {
			t.Errorf("order %+v", p.Steps)
		}
	}
	if (Item{}).Cells() != 1 {
		t.Error("unknown footprint counts one cell")
	}
}

func TestShouldGiveHealing(t *testing.T) {
	k := DefaultKnobs()
	m := owned(ClassRogue, 10)
	at := func(l int) MercState { s := m; s.LifePct = l; return s }
	fight := PotionCtx{HealingPotions: 3}
	for _, c := range []struct {
		name string
		m    MercState
		c    PotionCtx
		want bool
	}{
		{"39", at(39), fight, true}, {"40", at(40), fight, false}, {"1", at(1), fight, true},
		{"dead", func() MercState { s := at(10); s.Alive = false; return s }(), fight, false},
		{"zero life unread", at(0), fight, false},
		{"trivial fight", at(10), PotionCtx{FightTrivial: true, HealingPotions: 3}, false},
		{"no potion", at(10), PotionCtx{}, false}, {"unknown", MercState{LifePct: 10}, fight, false},
	} {
		if got := ShouldGiveHealing(c.m, c.c, k); got != c.want {
			t.Errorf("%s: %v", c.name, got)
		}
	}
	if BuyPotionsForMerc() {
		t.Error("never buy for the merc")
	}
}

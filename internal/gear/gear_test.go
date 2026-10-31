package gear

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestIsA(t *testing.T) {
	needTables(t)
	for _, c := range []struct {
		typ, anc string
		want     bool
	}{
		{"abow", "bow", true}, {"abow", "miss", true}, {"bow", "miss", true}, {"xbow", "miss", true},
		{"ajav", "jave", true}, {"jave", "thro", true}, {"aspe", "spea", true}, {"spea", "jave", false},
		{"swor", "miss", false}, {"mboq", "bowq", true}, {"bowq", "xboq", false},
	} {
		if got := IsA(c.typ, c.anc); got != c.want {
			t.Errorf("IsA(%s,%s)=%v", c.typ, c.anc, got)
		}
	}
}

func TestSkillUsable(t *testing.T) {
	needTables(t)
	bow := Equipped{Weapon: "hbw", Offhand: "aqv", OffhandQty: 300}
	jav := Equipped{Weapon: "jav", WeaponQty: 40, Offhand: "buc"}
	spear := Equipped{Weapon: "spr"}
	staff := Equipped{Weapon: "sst"}
	cases := []struct {
		skill string
		e     Equipped
		want  bool
	}{
		{"Fire Arrow", bow, true}, {"Multiple Shot", bow, true}, {"Exploding Arrow", bow, true},
		{"Poison Javelin", bow, false}, {"Lightning Fury", bow, false}, {"Jab", bow, false}, // jave is a spea per Equiv2 (V itemtypes), so melee spear skills accept javelins
		{"Fire Arrow", jav, false}, {"Poison Javelin", jav, true}, {"Lightning Bolt", jav, true}, {"Charged Strike", jav, true},
		{"Jab", spear, true}, {"Charged Strike", spear, true}, {"Fire Arrow", spear, false}, {"Poison Javelin", spear, false},
		{"Fire Arrow", Equipped{}, false}, {"Teeth", staff, true}, {"Bone Spear", Equipped{}, true}, {"Fire Bolt", staff, true},
		{"Fire Arrow", Equipped{Weapon: "hbw"}, false},                                  // no quiver
		{"Fire Arrow", Equipped{Weapon: "hbw", Offhand: "aqv"}, false},                  // empty quiver
		{"Fire Arrow", Equipped{Weapon: "hbw", Offhand: "cqv", OffhandQty: 100}, false}, // bolts for a bow
		{"Fire Arrow", Equipped{Weapon: "mxb", Offhand: "cqv", OffhandQty: 100}, true},  // crossbow + bolts (itypea miss)
		{"Poison Javelin", Equipped{Weapon: "jav"}, false},                              // empty stack
		{"Poison Javelin", Equipped{Weapon: "am5", WeaponQty: 10}, true},                // Maiden Javelin is a jave
		{"Poison Dagger", Equipped{Weapon: "dgr"}, true}, {"Poison Dagger", Equipped{Weapon: "sst"}, false},
		{"Throw", jav, true}, {"Throw", staff, false}, {"Unknown Skill", Equipped{}, true},
	}
	for _, c := range cases {
		if got, why := SkillUsable(c.skill, c.e); got != c.want {
			t.Errorf("SkillUsable(%q,%+v)=%v (%s)", c.skill, c.e, got, why)
		}
	}
}

func TestWeaponProblem(t *testing.T) {
	needTables(t)
	cases := []struct {
		class string
		e     Equipped
		want  string // substring, "" none
	}{
		{"Amazon", Equipped{Weapon: "hbw", Offhand: "aqv", OffhandQty: 100}, ""},
		{"amazon", Equipped{Weapon: "jav", WeaponQty: 60, Offhand: "buc"}, ""},
		{"Amazon", Equipped{Weapon: "spr"}, ""},
		{"Amazon", Equipped{}, "no weapon"},
		{"Amazon", Equipped{Weapon: "hbw", Offhand: "buc"}, "non-quiver"},
		{"Amazon", Equipped{Weapon: "hbw"}, "without quiver"},
		{"Amazon", Equipped{Weapon: "hbw", Offhand: "aqv"}, "empty"},
		{"Amazon", Equipped{Weapon: "hbw", Offhand: "cqv", OffhandQty: 9}, "does not fit"},
		{"Amazon", Equipped{Weapon: "jav"}, "empty"},
		{"Amazon", Equipped{Weapon: "am1", Offhand: "aqv", OffhandQty: 5}, ""},
		{"Sorceress", Equipped{Weapon: "sst"}, ""},
		{"Sorceress", Equipped{}, ""},
		{"Necromancer", Equipped{Weapon: "wnd", Offhand: "buc"}, ""},
		{"Necromancer", Equipped{Weapon: "am1", Offhand: "aqv", OffhandQty: 5}, "restricted"},
		{"Paladin", Equipped{Weapon: "ssd", Offhand: "buc"}, ""},
		{"Necromancer", Equipped{Weapon: "zzzz"}, ""},
	}
	for _, c := range cases {
		got := WeaponProblem(c.class, c.e)
		if (c.want == "") != (got == "") || !strings.Contains(got, c.want) {
			t.Errorf("WeaponProblem(%s,%+v)=%q want %q", c.class, c.e, got, c.want)
		}
	}
}

func TestAmmo(t *testing.T) {
	needTables(t)
	for _, c := range []struct {
		e    Equipped
		low  int
		isLo bool
	}{
		{Equipped{Weapon: "hbw", Offhand: "aqv", OffhandQty: 350}, 70, false},
		{Equipped{Weapon: "hbw", Offhand: "aqv", OffhandQty: 70}, 70, true},
		{Equipped{Weapon: "mxb", Offhand: "cqv", OffhandQty: 51}, 50, false},
		{Equipped{Weapon: "mxb", Offhand: "cqv", OffhandQty: 50}, 50, true},
		{Equipped{Weapon: "jav", WeaponQty: 21}, 20, false},
		{Equipped{Weapon: "jav", WeaponQty: 20}, 20, true},
	} {
		low, ok := LowAmmo(c.e)
		if !ok || low != c.low || AmmoIsLow(c.e) != c.isLo {
			t.Errorf("%+v low=%d ok=%v isLow=%v", c.e, low, ok, AmmoIsLow(c.e))
		}
	}
	if _, ok := LowAmmo(Equipped{Weapon: "sst"}); ok {
		t.Error("staff has no ammo")
	}
	if b, code, ok := AmmoFor("lwb"); !ok || code != "aqv" || b.MaxStack != 350 {
		t.Errorf("AmmoFor bow: %v %v %v", b, code, ok)
	}
	if b, code, ok := AmmoFor("hxb"); !ok || code != "cqv" || b.MaxStack != 250 {
		t.Errorf("AmmoFor xbow: %v %v %v", b, code, ok)
	}
	if !func() bool { b, _ := BaseByCode("aqv"); return b.PermStore }() {
		t.Error("Arrows must be a PermStoreItem")
	}
}

func TestRestock(t *testing.T) {
	needTables(t)
	cases := []struct {
		e    Equipped
		want string // purchase code or ""
	}{
		{Equipped{Weapon: "hbw", Offhand: "aqv", OffhandQty: 300}, ""},
		{Equipped{Weapon: "hbw", Offhand: "aqv", OffhandQty: 10}, "aqv"},
		{Equipped{Weapon: "hbw", Offhand: "buc"}, "aqv"},
		{Equipped{Weapon: "hbw", Offhand: "cqv", OffhandQty: 99}, "aqv"},
		{Equipped{Weapon: "mxb"}, "cqv"},
		{Equipped{Weapon: "jav", WeaponQty: 5}, "jav"},
		{Equipped{Weapon: "sst"}, ""},
	}
	for _, c := range cases {
		p := Restock(c.e)
		got := ""
		if len(p) > 0 {
			got = p[0].Code
		}
		if got != c.want {
			t.Errorf("Restock(%+v)=%v want %q", c.e, p, c.want)
		}
	}
}

func TestStartPurchasesAndItems(t *testing.T) {
	needTables(t)
	p := StartPurchases("Amazon", "bow")
	if len(p) != 2 || p[0].Code != "sbw" || p[1].Code != "aqv" || p[1].Qty != 350 {
		t.Errorf("bow start %v", p)
	}
	if p := StartPurchases("Amazon", "javelin"); len(p) != 1 || p[0].Code != "jav" || p[0].Qty != 60 {
		t.Errorf("javelin start %v", p)
	}
	if StartPurchases("Necromancer", "") != nil {
		t.Error("necromancer buys nothing for weapons")
	}
	// the start kit must not be a WeaponProblem for the class that owns it, except the Amazon with a bare javelin stack
	for class, want := range map[string]string{"Amazon": "jav", "Necromancer": "wnd", "Sorceress": "sst", "Paladin": "ssd"} {
		var weapon string
		var off string
		var qty int
		for _, it := range StartItems(class) {
			switch it.Loc {
			case "rarm":
				weapon = it.Code
				qty = 30
			case "larm":
				off = it.Code
			}
		}
		if weapon != want {
			t.Errorf("%s starts with %q want %q", class, weapon, want)
		}
		if p := WeaponProblem(class, Equipped{Weapon: weapon, WeaponQty: qty, Offhand: off}); p != "" {
			t.Errorf("%s start kit is a problem: %s", class, p)
		}
	}
	c, ok := ClassStart("Amazon")
	if !ok || c.Str != 20 || c.Dex != 25 || c.Vit != 20 || c.Ene != 15 {
		t.Errorf("amazon start %+v", c)
	}
	if c, _ := ClassStart("Necromancer"); c.Str != 15 || c.Dex != 25 || c.Ene != 25 || c.Start != "raise skeleton" {
		t.Errorf("necro start %+v", c)
	}
}

func TestPickLauncher(t *testing.T) {
	needTables(t)
	stock := []string{"sbw", "hbw", "lbw", "cbw", "lxb", "am1", "spr", "lwb"}
	for _, c := range []struct {
		class        string
		str, dex, lv int
		want         string
	}{
		{"Amazon", 20, 25, 5, "sbw"},  // hbw needs dex 28, long bow str 22, light crossbow str 21
		{"Amazon", 20, 28, 5, "hbw"},  // hunter's bow 350 beats short bow
		{"Amazon", 30, 45, 14, "am1"}, // Stag Bow: clvl 14, str 30, dex 45 (V weapons.txt)
		{"Amazon", 20, 15, 1, "sbw"},
		{"Amazon", 5, 5, 1, ""},
		{"Necromancer", 60, 99, 99, "lwb"}, // class-restricted am1 skipped
	} {
		got := PickLauncher(stock, c.class, c.str, c.dex, c.lv)
		if got != c.want {
			b, _ := BaseByCode(c.want)
			t.Errorf("PickLauncher(%v)=%q want %q (%+v)", c, got, c.want, b)
		}
	}
}

func TestBuilds(t *testing.T) {
	needTables(t)
	skills := load().Skills
	for _, name := range Builds() {
		b, err := BuildJSON(name)
		if err != nil {
			t.Fatal(err)
		}
		var f struct {
			Build string `json:"build"`
			Class string `json:"class"`
			Tags  string `json:"tags"`
			Rows  []struct {
				Clvl                                  int
				Strength, Dexterity, Vitality, Energy int
				Skills                                []struct {
					Skill  string
					Points int
				}
			} `json:"rows"`
			Hotkeys []struct {
				Key, Skill, Hand string
			} `json:"hotkeys"`
			Main []MainAttack `json:"main_attack"`
		}
		if err := json.Unmarshal(b, &f); err != nil {
			t.Fatal(name, err)
		}
		if f.Build != name || f.Class == "" || !strings.Contains(f.Tags, "V") {
			t.Errorf("%s: header %q %q", name, f.Build, f.Class)
		}
		cc := ClassCode(f.Class)
		last := tableMax(name)
		if len(f.Rows) != last-1 || f.Rows[0].Clvl != 2 || f.Rows[len(f.Rows)-1].Clvl != last {
			t.Fatalf("%s: rows %d", name, len(f.Rows))
		}
		lv := map[string]int{}
		points := 0
		for i, r := range f.Rows {
			if r.Clvl != i+2 {
				t.Errorf("%s: row %d is clvl %d", name, i, r.Clvl)
			}
			if r.Strength+r.Dexterity+r.Vitality+r.Energy != 5 {
				t.Errorf("%s clvl %d: stat points != 5", name, r.Clvl)
			}
			for _, sp := range r.Skills {
				s, ok := skills[sp.Skill]
				if !ok || s.Class != cc {
					t.Errorf("%s clvl %d: %q not a %s skill", name, r.Clvl, sp.Skill, cc)
					continue
				}
				for k := 0; k < sp.Points; k++ {
					if r.Clvl < s.ReqLevel {
						t.Errorf("%s clvl %d: %s needs clvl %d", name, r.Clvl, sp.Skill, s.ReqLevel)
					}
					for _, q := range s.Req {
						if lv[q] == 0 {
							t.Errorf("%s clvl %d: %s needs %s first", name, r.Clvl, sp.Skill, q)
						}
					}
					lv[sp.Skill]++
					points++
					if lv[sp.Skill] > 20 {
						t.Errorf("%s: %s past 20", name, sp.Skill)
					}
				}
			}
		}
		// one point per level (29) plus the Den, Radament and Izual quest points when the table folds them in
		if points < last-1 || points > last+15 {
			t.Errorf("%s: %d skill points", name, points)
		}
		for _, h := range f.Hotkeys {
			if _, ok := skills[h.Skill]; !ok || (h.Hand != "left" && h.Hand != "right") {
				t.Errorf("%s: hotkey %+v", name, h)
			}
		}
		// main attack ranges cover 1..last without gaps
		next := 1
		for _, m := range f.Main {
			if m.From != next {
				t.Errorf("%s: main attack gap at %d (range starts %d)", name, next, m.From)
			}
			next = m.Until + 1
			if _, ok := skills[m.Skill]; !ok && m.Skill != "Throw" {
				t.Errorf("%s: main attack %q unknown", name, m.Skill)
			}
		}
		if next != last+1 {
			t.Errorf("%s: main attack ends at %d", name, next-1)
		}
	}
}

// tableMax is the last clvl a build table covers: 90 for the endgame tables, 30 for the Normal ones.
func tableMax(name string) int {
	if strings.HasSuffix(name, "_endgame90") {
		return 90
	}
	return 30
}

func TestBuildsMatchWeapons(t *testing.T) {
	needTables(t)
	// the main attack of every level range must be usable with the weapon the range names, and with a weapon of
	// another family it must not be (bow skills vs javelin).
	for _, name := range Builds() {
		for clvl := 1; clvl <= tableMax(name); clvl++ {
			m, err := MainAttackAt(name, clvl)
			if err != nil {
				t.Fatal(err)
			}
			var e Equipped
			switch m.Weapon {
			case "bow":
				e = Equipped{Weapon: "hbw", Offhand: "aqv", OffhandQty: 300}
			case "jave":
				e = Equipped{Weapon: "jav", WeaponQty: 50}
			default:
				e = Equipped{Weapon: "wnd"}
				if m.Skill == "Smite" {
					e = Equipped{Weapon: "ssd", Offhand: "buc"} // the Paladin start kit; Smite needs a shield
				}
			}
			if ok, why := SkillUsable(m.Skill, e); !ok {
				t.Errorf("%s clvl %d: main attack %s unusable with its own weapon: %s", name, clvl, m.Skill, why)
			}
		}
	}
	if DefaultBuild("Amazon") == "" || DefaultBuild("necromancer") == "" || DefaultBuild("Druid") != "druid_fire_normal30" || DefaultBuild("assassin") != "assassin_fireblast_normal30" || DefaultBuild("barbarian") != "barbarian_melee_normal30" {
		t.Error("DefaultBuild")
	}
	if _, err := BuildJSON("nope"); err == nil {
		t.Error("unknown build must fail")
	}
}

func TestVendors(t *testing.T) {
	needTables(t)
	if v, ok := VendorForAct(1); !ok || v.NPC != 154 || !v.Verified {
		t.Errorf("act 1 vendor %+v", v)
	}
	if _, ok := VendorForAct(6); ok {
		t.Error("act 6")
	}
}

func TestDruidAssassinSkillsNeedNoWeapon(t *testing.T) {
	needTables(t)
	// the two new default builds use only skill attacks: usable bare-handed, with a staff and with claws
	for _, e := range []Equipped{{}, {Weapon: "sst"}, {Weapon: "clw"}} {
		for _, s := range []string{"Firestorm", "Molten Boulder", "Eruption", "Fire Trauma", "Wake of Fire Sentry", "Quickness"} {
			if ok, why := SkillUsable(s, e); !ok {
				t.Errorf("%s with %q: %s", s, e.Weapon, why)
			}
		}
	}
	// claw-only martial arts must refuse a staff (V data.json itypea h2h), so a martial arts build needs claws
	if ok, _ := SkillUsable("Fists of Fire", Equipped{Weapon: "sst"}); ok {
		t.Error("Fists of Fire with a staff")
	}
	if ok, why := SkillUsable("Fists of Fire", Equipped{Weapon: "clw"}); !ok {
		t.Error(why)
	}
}

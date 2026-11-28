package d2

import "testing"

func TestClassTraitsTable(t *testing.T) {
	casters := map[Class]bool{Sorceress: true, Necromancer: true, Druid: true}
	for c := Amazon; c <= Assassin; c++ {
		if c.Caster() != casters[c] {
			t.Errorf("%s caster %v", c, c.Caster())
		}
		want := 1
		switch {
		case casters[c]:
			want = 2
		case c == Barbarian:
			want = 0
		}
		if c.ManaNeed() != want {
			t.Errorf("%s manaNeed %d want %d", c, c.ManaNeed(), want)
		}
	}
	if _, err := TraitsOf(Class(9)); err == nil {
		t.Error("an unknown class must be an error")
	}
	if _, err := TraitsByName("Warlock"); err == nil {
		t.Error("an unknown class name must be an error")
	}
}

func TestMercAuraFor(t *testing.T) {
	if a, ok, err := MercAuraFor(Necromancer, "Nightmare"); err != nil || !ok || a != "Might" {
		t.Fatalf("necro nightmare: %q %v %v", a, ok, err)
	}
	if _, ok, err := MercAuraFor(Sorceress, "normal"); err != nil || ok {
		t.Fatalf("sorceress has no row: %v %v", ok, err)
	}
	if _, _, err := MercAuraFor(Class(42), "normal"); err == nil {
		t.Fatal("unknown class must error")
	}
}

// Review 8 A1/A3: the merc aura defaults are data (the "default" row), for every difficulty and the Act 1 Rogue.
func TestDefaultMercAura(t *testing.T) {
	for _, k := range []string{"normal", "nightmare", "hell", "rogue"} {
		if a, err := DefaultMercAura(k); err != nil || a == "" {
			t.Errorf("%s: %q %v", k, a, err)
		}
	}
	if _, err := DefaultMercAura("inferno"); err == nil {
		t.Error("an unknown key must be an error")
	}
	// two classes: the Necromancer has its own row, the Paladin has none and gets no silent class value
	if a, ok, _ := MercAuraFor(Necromancer, "nightmare"); !ok || a == "" {
		t.Errorf("necromancer nightmare %q %v", a, ok)
	}
	if _, ok, err := MercAuraFor(Paladin, "nightmare"); ok || err != nil {
		t.Errorf("paladin has no class row: %v %v", ok, err)
	}
}

// Card cls2: the Amazon and the Assassin carry their own Act 2 aura per difficulty, and each is an aura the Desert
// mercenary offers in that difficulty in LoD 1.14d (docs/sme/merc-auras.md section 1, V there: Might, Holy Freeze and
// Thorns are Nightmare-only). Source of the picks (S): docs/sme/class-amazon.md and class-assassin.md section 5.
func TestMercAuraHireableInItsDifficulty(t *testing.T) {
	offered := map[string]map[string]bool{
		"normal":    {"Prayer": true, "Defiance": true, "Blessed Aim": true},
		"nightmare": {"Thorns": true, "Holy Freeze": true, "Might": true},
		"hell":      {"Prayer": true, "Defiance": true, "Blessed Aim": true},
	}
	for name, tr := range classTraits {
		for diff, aura := range tr.MercAura {
			if o, ok := offered[diff]; ok && !o[aura] {
				t.Errorf("%s: %s is not hireable in %s", name, aura, diff)
			}
		}
	}
	want := map[Class]string{Amazon: "Might", Assassin: "Holy Freeze"}
	for c, w := range want {
		for _, diff := range []string{"normal", "nightmare", "hell"} {
			a, ok, err := MercAuraFor(c, diff)
			if err != nil || !ok {
				t.Fatalf("%s %s: no aura row (ok=%v err=%v)", c, diff, ok, err)
			}
			if diff == "nightmare" && a != w {
				t.Errorf("%s nightmare aura = %s, want %s", c, a, w)
			}
		}
	}
}

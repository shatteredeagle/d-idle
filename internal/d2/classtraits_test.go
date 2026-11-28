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

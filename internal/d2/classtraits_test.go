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

package d2

import "testing"

func mercSnap(units ...Unit) *Snapshot {
	return &Snapshot{Player: &Player{X: 100, Y: 100}, Monsters: units}
}

func TestFindMerc(t *testing.T) {
	mon := func(life int) *MonsterInfo { return &MonsterInfo{LifePct: life} }
	cases := []struct {
		name string
		s    *Snapshot
		want Merc
	}{
		{"nil", nil, Merc{}},
		{"none", mercSnap(Unit{Class: 148, X: 100, Y: 100}), Merc{}},
		{"560 is not a merc", mercSnap(Unit{Class: 560}), Merc{}},
		{"alive", mercSnap(Unit{Class: 338, ID: 7, Mode: MonModeStand, X: 103, Y: 100, Monster: mon(80)}),
			Merc{Present: true, Alive: true, Class: 338, ID: 7, X: 103, Y: 100, LifePct: 80}},
		{"dead corpse", mercSnap(Unit{Class: 271, Mode: MonModeDead, Monster: mon(0)}),
			Merc{Present: true, Class: 271}},
		{"dead keeps no life", mercSnap(Unit{Class: 271, Mode: MonModeDeath, Monster: mon(30)}),
			Merc{Present: true, Class: 271}},
		{"nearest of two", mercSnap(
			Unit{Class: 359, ID: 1, Mode: MonModeStand, X: 150, Y: 100, Monster: mon(50)},
			Unit{Class: 561, ID: 2, Mode: MonModeStand, X: 105, Y: 100, Monster: mon(60)}),
			Merc{Present: true, Alive: true, Class: 561, ID: 2, X: 105, Y: 100, LifePct: 60}},
		{"alive beats corpse", mercSnap(
			Unit{Class: 271, ID: 1, Mode: MonModeDead, X: 100, Y: 100},
			Unit{Class: 271, ID: 2, Mode: MonModeStand, X: 190, Y: 100, Monster: mon(100)}),
			Merc{Present: true, Alive: true, Class: 271, ID: 2, X: 190, Y: 100, LifePct: 100}},
	}
	for _, c := range cases {
		if got := FindMerc(c.s); got != c.want {
			t.Errorf("%s: got %+v want %+v", c.name, got, c.want)
		}
	}
}

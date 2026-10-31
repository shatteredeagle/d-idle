package gear

import "testing"

// Paladin start kit (Short Sword + Buckler, V live on Pally 2026-10-29) must be fine; other classes' gear must be flagged.
func TestWeaponProblemPaladin(t *testing.T) {
	needTables(t)
	for _, c := range []struct {
		name string
		e    Equipped
		bad  bool
	}{
		{"start kit", Equipped{Weapon: "ssd", Offhand: "buc"}, false},
		{"empty hands", Equipped{}, false},
		{"scepter+auric shield absent", Equipped{Weapon: "scp", Offhand: "buc"}, false},
		{"amazon bow", Equipped{Weapon: "sbw"}, true},
		{"sorc orb offhand", Equipped{Weapon: "ssd", Offhand: "ob1"}, true},
	} {
		got := WeaponProblem("paladin", c.e)
		if (got != "") != c.bad {
			t.Errorf("%s: problem=%q want bad=%v", c.name, got, c.bad)
		}
	}
}

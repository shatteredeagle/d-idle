package combat

import (
	"d2lod-bot/internal/gear/geartest"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/gear"
)

func TestGearFilterAmazon(t *testing.T) {
	geartest.Need(t)
	h := newHarness(t)
	f := h.f
	f.e.Data = &data.DB{Items: []*data.ItemBase{{Code: "jav"}, {Code: "sbw"}, {Code: "aqv"}, {Code: "buc"}}}
	qty := map[uint32]int32{}
	f.statsFn = func(a uint32) (d2.Stats, error) { return d2.Stats{70: qty[a]}, nil }
	f.GearBuild = "amazon_bow_normal30"
	item := func(cls, body int, addr uint32) d2.Unit {
		return d2.Unit{Addr: addr, Class: cls, Item: &d2.Item{Where: d2.ContEquipment, Body: body}}
	}
	snap := func(items ...d2.Unit) *d2.Snapshot {
		return &d2.Snapshot{Player: &d2.Player{Class: d2.Amazon, Level: 8}, Inventory: &d2.Inventory{Items: items}}
	}
	specs := []SkillSpec{{Name: "Multiple Shot"}, {Name: "Fire Arrow"}, {Name: "Jab"}, {Name: "Attack"}}
	names := func(s []SkillSpec) (o []string) {
		for _, x := range s {
			o = append(o, x.Name)
		}
		return
	}

	qty[1] = 40
	// Javelin + buckler: bow skills are refused (smoke-amazon finding 2), Jab stays.
	s := snap(item(0, d2.BodyRHand, 1), item(3, d2.BodyLHand, 2))
	f.noteGear(s)
	if got := names(f.gearFilter(s.Player, specs)); len(got) != 2 || got[0] != "Jab" || got[1] != "Attack" {
		t.Fatalf("javelin: %v", got)
	}

	// Short bow + arrows: bow skills pass, main attack goes first.
	qty[3] = 300
	s = snap(item(1, d2.BodyRHand, 4), item(2, d2.BodyLHand, 3))
	f.noteGear(s)
	got := names(f.gearFilter(s.Player, specs))
	if len(got) < 3 || got[0] != "Fire Arrow" || f.WeaponProblem() != "" || f.AmmoLow() {
		t.Fatalf("bow: %v problem %q", got, f.WeaponProblem())
	}

	// Empty quiver: weapon problem, no skills, logged once.
	qty[3] = 0
	f.noteGear(s)
	if f.WeaponProblem() == "" || f.gearFilter(s.Player, specs) != nil {
		t.Fatalf("empty quiver should refuse, problem %q", f.WeaponProblem())
	}
	// Unknown gear never refuses.
	f.noteGear(&d2.Snapshot{Player: &d2.Player{Class: d2.Amazon}})
	if len(f.gearFilter(s.Player, specs)) != 4 {
		t.Fatal("no inventory must not filter")
	}
}

func TestFrenzyNeedsAMeleePair(t *testing.T) {
	geartest.Need(t)
	cases := []struct {
		e    gear.Equipped
		want bool
	}{
		{gear.Equipped{Weapon: "hax", Offhand: "hax"}, true},
		{gear.Equipped{Weapon: "hax"}, false},
		{gear.Equipped{Weapon: "hax", Offhand: "buc"}, false},
		{gear.Equipped{}, false},
	}
	for _, c := range cases {
		if got := dualWieldOK("Frenzy", c.e); got != c.want {
			t.Errorf("Frenzy %+v = %v", c.e, got)
		}
	}
	if !dualWieldOK("Bash", gear.Equipped{Weapon: "hax"}) {
		t.Error("Bash is not a dual-wield skill")
	}
}

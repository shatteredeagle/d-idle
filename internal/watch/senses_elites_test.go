package watch

import (
	"reflect"
	"testing"

	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/senses"
	"d2lod-bot/internal/unlock"
)

// eliteSnap: a champion with two mods, Bishibosh (superunique), a rare unique with an id the table lacks, a plain Fallen.
func eliteSnap(t *testing.T) snapSrc {
	t.Helper()
	snap, ok := sensesFixture(t).Snapshot()
	if !ok {
		t.Fatal("no snapshot")
	}
	snap.Unit = []senses.Unit{
		{ID: 1, Kind: senses.KindMonster, X: 3, Y: 1, Name: "Fallen", LifePct: 100, Elite: senses.EliteChampion,
			Mods: []string{decide.ModName(5), decide.ModName(7)}},
		{ID: 2, Kind: senses.KindMonster, X: 4, Y: 1, Name: "Skeleton", Title: "Bishibosh", LifePct: 100, Elite: senses.EliteSuperunique,
			Mods: []string{decide.ModName(6), decide.ModName(9)}},
		{ID: 3, Kind: senses.KindMonster, X: 2, Y: 2, Name: "Zombie", LifePct: 100, Elite: senses.EliteUnique,
			Mods: []string{decide.ModName(34)}},
		{ID: 4, Kind: senses.KindMonster, X: 5, Y: 1, Name: "Fallen", LifePct: 100},
	}
	return snapSrc{snap}
}

func pick(us []PublicUnit, id uint32) PublicUnit {
	for _, u := range us {
		if u.ID == id {
			return u
		}
	}
	return PublicUnit{}
}

func TestSensesElitesFixture(t *testing.T) {
	us := (&Senses{Src: eliteSnap(t), Lock: FixedTier(4)}).Reply(0).Units
	if c := pick(us, 1); c.Elite != senses.EliteChampion || !reflect.DeepEqual(c.Mods, []string{"Extra Strong", "Cursed"}) {
		t.Fatalf("champion with two mods: %+v", c)
	}
	if s := pick(us, 2); s.Elite != senses.EliteSuperunique || s.Name != "Bishibosh" || !reflect.DeepEqual(s.Mods, []string{"Extra Fast", "Fire Enchanted"}) {
		t.Fatalf("superunique: %+v", s)
	}
	if u := pick(us, 3); u.Elite != senses.EliteUnique || !reflect.DeepEqual(u.Mods, []string{"unknown mod #34"}) {
		t.Fatalf("unknown mod id: %+v", u)
	}
	if p := pick(us, 4); p.Elite != "" || p.Mods != nil {
		t.Fatalf("plain monster marked: %+v", p)
	}
	// Tier 3 alone shows none of it.
	for _, u := range (&Senses{Src: eliteSnap(t), Lock: FixedTier(3)}).Reply(0).Units {
		if u.Elite != "" || u.Mods != nil {
			t.Fatalf("tier 3 leaks %+v", u)
		}
	}
}

// First encounter keeps the surprise: with only a champion in the Codex the superunique is still a plain dot, and the
// other way round; checked for both hero classes.
func TestSensesElitesFirstEncounter(t *testing.T) {
	champ := unlock.Event{Type: unlock.EvCodexFirst, Sub: "champion", ID: "champion_pack", Name: "Cursed"}
	bish := unlock.Event{Type: unlock.EvCodexFirst, Sub: "monster", ID: "bishibosh", Name: "Bishibosh"}
	fallen := unlock.Event{Type: unlock.EvCodexFirst, Sub: "monster", ID: "fallen", Name: "Fallen"} // opens tiers 2 and 3
	for _, class := range []string{"sorceress", "paladin"} {
		for _, tc := range []struct {
			evs           []unlock.Event
			champ, superU bool
		}{{[]unlock.Event{champ}, true, false}, {[]unlock.Event{bish}, false, true}, {[]unlock.Event{champ, bish}, true, true}} {
			l, err := unlock.Shipped()
			if err != nil {
				t.Fatal(err)
			}
			eng, err := unlock.Open(t.TempDir(), class, l, nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, ev := range append([]unlock.Event{fallen}, tc.evs...) {
				if _, err := eng.Observe(ev); err != nil {
					t.Fatal(err)
				}
			}
			us := (&Senses{Src: eliteSnap(t), Lock: LadderUnlocker{Engine: eng}}).Reply(0).Units
			if got := pick(us, 1).Elite != ""; got != tc.champ {
				t.Fatalf("%s %v: champion marked=%v", class, tc.evs, got)
			}
			s := pick(us, 2)
			if got := s.Elite != ""; got != tc.superU || (!tc.superU && (s.Name != "" || s.Mods != nil)) {
				t.Fatalf("%s %v: superunique marked=%v %+v", class, tc.evs, got, s)
			}
		}
	}
}

package watch

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"d2lod-bot/internal/senses"
	"d2lod-bot/internal/unlock"
)

// unitFixture: a Fallen being attacked, a plain Zombie, a mercenary, an NPC, a champion with two mods and Corpsefire.
func unitFixture(t *testing.T, merc bool) snapSrc {
	t.Helper()
	snap, ok := sensesFixture(t).Snapshot()
	if !ok {
		t.Fatal("no snapshot")
	}
	snap.Unit = []senses.Unit{
		{ID: 1, Kind: senses.KindMonster, X: 3, Y: 1, Class: 5, Name: "Fallen", LifePct: 55, Target: true},
		{ID: 2, Kind: senses.KindMonster, X: 4, Y: 1, Class: 6, Name: "Zombie", LifePct: 100},
		{ID: 3, Kind: senses.KindNPC, X: 1, Y: 1, Class: 150, Name: "Kashya"},
		{ID: 4, Kind: senses.KindMonster, X: 2, Y: 2, Class: 7, Name: "Fallen", LifePct: 100, Elite: senses.EliteChampion, Mods: []string{"Cursed", "Extra Fast"}},
		{ID: 5, Kind: senses.KindMonster, X: 5, Y: 1, Class: 8, Name: "Zombie", Title: "Corpsefire", LifePct: 90, Elite: senses.EliteSuperunique, Mods: []string{"Fire Enchanted", "unknown mod #99"}},
	}
	if merc {
		snap.Unit = append(snap.Unit, senses.Unit{ID: 6, Kind: senses.KindMerc, X: 1, Y: 2, Class: 271, Name: "Rogue"})
	}
	return snapSrc{snap}
}

func unitsOf(t *testing.T, s *Senses) []PublicUnit {
	t.Helper()
	return s.Reply(0).Units
}

// Golden JSON of the reply per ladder-less tier (FixedTier n): 0 off, 1 map only, 2 dots only, 3 target bar, 4 elites.
func TestSensesUnitsGoldenTiers(t *testing.T) {
	for tier := 0; tier <= 4; tier++ {
		s := &Senses{Src: unitFixture(t, true), Lock: FixedTier(tier)}
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/api/senses", nil))
		golden(t, fmt.Sprintf("senses-units-tier%d", tier), w.Body.String())
	}
}

func TestSensesUnitsTierPrivacy(t *testing.T) {
	two := unitsOf(t, &Senses{Src: unitFixture(t, true), Lock: FixedTier(2)})
	for _, u := range two {
		if u.Name != "" || u.LifePct != 0 || u.Target || u.Elite != "" || u.Mods != nil {
			t.Fatalf("tier 2 leaks %+v", u)
		}
	}
	three := unitsOf(t, &Senses{Src: unitFixture(t, true), Lock: FixedTier(3)})
	named := 0
	for _, u := range three {
		if u.Elite != "" || u.Mods != nil {
			t.Fatalf("tier 3 leaks elite data %+v", u)
		}
		if u.Name != "" {
			named++
			if !u.Target || u.LifePct != 55 {
				t.Fatalf("name for a unit that is not the target: %+v", u)
			}
		}
	}
	if named != 1 {
		t.Fatalf("tier 3 names %d units, want only the target", named)
	}
}

// Two heroes: a Sorceress without a mercenary and a Paladin with one; the merc is a blue dot, never named or barred.
func TestSensesUnitsMercPresence(t *testing.T) {
	for _, tc := range []struct {
		class string
		merc  bool
	}{{"sorceress", false}, {"paladin", true}} {
		us := unitsOf(t, &Senses{Src: unitFixture(t, tc.merc), Lock: FixedTier(4)})
		n := 0
		for _, u := range us {
			if u.Kind == senses.KindMerc {
				n++
				if u.Name != "" || u.LifePct != 0 {
					t.Fatalf("%s: merc detail %+v", tc.class, u)
				}
			}
		}
		if (n == 1) != tc.merc {
			t.Fatalf("%s: %d mercs", tc.class, n)
		}
	}
}

// ladderSenses opens the tiers by Codex firsts of the shipped ladder, for a hero class.
func ladderSenses(t *testing.T, class string, evs ...unlock.Event) *Senses {
	t.Helper()
	l, err := unlock.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	eng, err := unlock.Open(t.TempDir(), class, l, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range evs {
		if _, err := eng.Observe(ev); err != nil {
			t.Fatal(err)
		}
	}
	return &Senses{Src: unitFixture(t, class == "paladin"), Lock: LadderUnlocker{Engine: eng}}
}

func TestSensesUnitsSpoilerRule(t *testing.T) {
	fallen := unlock.Event{Type: unlock.EvCodexFirst, Sub: "monster", ID: "fallen", Name: "Fallen"}
	champ := unlock.Event{Type: unlock.EvCodexFirst, Sub: "champion", ID: "champion_pack", Name: "Cursed"}
	corpse := unlock.Event{Type: unlock.EvCodexFirst, Sub: "monster", ID: "corpsefire", Name: "Corpsefire"}
	byID := func(us []PublicUnit, id uint32) PublicUnit {
		for _, u := range us {
			if u.ID == id {
				return u
			}
		}
		t.Fatalf("unit %d missing", id)
		return PublicUnit{}
	}
	for _, class := range []string{"sorceress", "paladin"} {
		// Only Fallen met: tiers 2 and 3 are open, tier 4 is not. The Fallen target shows a bar; no elite data at all.
		us := unitsOf(t, ladderSenses(t, class, fallen))
		if u := byID(us, 1); u.Name != "Fallen" || u.LifePct != 55 {
			t.Fatalf("%s: %+v", class, u)
		}
		for _, id := range []uint32{4, 5} {
			if u := byID(us, id); u.Elite != "" || u.Mods != nil || u.Name != "" {
				t.Fatalf("%s: elite %d leaks before tier 4: %+v", class, id, u)
			}
		}
		// A first champion opens tier 4 for champions only; Corpsefire (never met) stays a plain dot.
		us = unitsOf(t, ladderSenses(t, class, fallen, champ))
		if u := byID(us, 4); u.Elite != senses.EliteChampion || len(u.Mods) != 2 {
			t.Fatalf("%s: champion %+v", class, u)
		}
		if u := byID(us, 5); u.Elite != "" || u.Mods != nil || u.Name != "" {
			t.Fatalf("%s: never-met superunique shows: %+v", class, u)
		}
		// Meeting Corpsefire marks him with his own name.
		us = unitsOf(t, ladderSenses(t, class, fallen, champ, corpse))
		if u := byID(us, 5); u.Elite != senses.EliteSuperunique || u.Name != "Corpsefire" || len(u.Mods) != 2 {
			t.Fatalf("%s: superunique %+v", class, u)
		}
	}
	// Tier 3 is per class: the Zombie target of a hero who never met a Zombie stays a plain dot.
	s := ladderSenses(t, "sorceress", fallen)
	snap, _ := s.Src.Snapshot()
	snap.Unit[0], snap.Unit[1] = snap.Unit[1], snap.Unit[0]
	snap.Unit[0].Target, snap.Unit[1].Target = true, false
	s.Src = snapSrc{snap}
	if u := byID(unitsOf(t, s), 2); u.Name != "" || u.Target || u.LifePct != 0 {
		t.Fatalf("unmet class shows detail: %+v", u)
	}
	// Elite data of an unmet kind is also absent from the raw bytes, not only from the typed field.
	w := httptest.NewRecorder()
	ladderSenses(t, "sorceress", fallen).Handler().ServeHTTP(w, httptest.NewRequest("GET", "/api/senses", nil))
	for _, leak := range []string{"Corpsefire", "Cursed", "Fire Enchanted", `"elite"`} {
		if strings.Contains(w.Body.String(), leak) {
			t.Fatalf("reply bytes contain %q before tier 4", leak)
		}
	}
}

func TestSensesUnitsConcurrentPollers(t *testing.T) {
	src := &swapSrc{s: unitFixture(t, true).s}
	s := &Senses{Src: src, Lock: FixedTier(4)}
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				if g == 0 {
					src.set(i)
					continue
				}
				r := s.Reply(0)
				if _, err := json.Marshal(r); err != nil {
					t.Error(err)
				}
			}
		}(g)
	}
	wg.Wait()
}

type swapSrc struct {
	mu sync.Mutex
	s  senses.Snapshot
}

func (f *swapSrc) Snapshot() (senses.Snapshot, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.s, true
}

// set swaps in a new unit slice (the publisher never mutates one in place).
func (f *swapSrc) set(i int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.s.Unit = []senses.Unit{{ID: uint32(i), Kind: senses.KindMonster, X: 3, Y: 1}}
}

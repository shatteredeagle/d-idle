package combat

import (
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
)

func TestLookupProfileByID(t *testing.T) {
	for _, c := range []struct {
		class d2.Class
		id    Build
		want  string
	}{
		{d2.Sorceress, "", "sorceress/blizzard"}, // the zero value is the old default
		{d2.Sorceress, BuildBlizzard, "sorceress/blizzard"},
		{d2.Sorceress, BuildLightning, "sorceress/lightning"},
		{d2.Sorceress, "sorceress/lightning", "sorceress/lightning"},
		{d2.Paladin, "", "paladin"},
		{d2.Paladin, BuildBlizzard, "paladin"}, // legacy ids still reach a single-profile class
		{d2.Paladin, "paladin", "paladin"},
	} {
		p, err := LookupProfile(c.class, c.id)
		if err != nil || p.ProfileID() != c.want {
			t.Errorf("%v %q: %v %v, want %s", c.class, c.id, p, err, c.want)
		}
	}
}

func TestLookupProfileUnknownIsAnError(t *testing.T) {
	for _, c := range []struct {
		class d2.Class
		id    Build
	}{
		{d2.Class(42), ""},          // unknown class: never the Sorceress
		{d2.Sorceress, "poison"},    // unknown profile of a known class
		{d2.Paladin, "zealot-lite"}, // single-profile classes only know their own id
	} {
		if p, err := LookupProfile(c.class, c.id); err == nil || p != nil {
			t.Errorf("%v %q: got %v, want an error", c.class, c.id, p)
		}
	}
}

func TestSelectProfileCheckedErrorsAndFallbackLogs(t *testing.T) {
	cat := testCatalog()
	if _, err := SelectProfileChecked(&d2.Player{Class: d2.Class(42)}, cat, ""); err == nil {
		t.Fatal("unknown class must be an error")
	}
	if _, err := SelectProfileChecked(&d2.Player{Class: d2.Sorceress}, cat, "poison"); err == nil {
		t.Fatal("unknown profile must be an error")
	}
	// the lenient form still returns something the fight loop can use
	if pr, _ := SelectProfile(&d2.Player{Class: d2.Sorceress}, cat, "poison"); pr == nil || pr.Class != "sorceress" {
		t.Fatalf("%v", pr)
	}
}

// The sorceress/fire line (sorc2): the second Sorceress build selects it by name, Fire Ball leads once owned.
func TestSorceressFireProfile(t *testing.T) {
	pr, err := LookupProfile(d2.Sorceress, "fire")
	if err != nil || pr == nil || pr.Bands[0].Requires != "Fire Ball" || pr.Bands[0].Attack[0] != "Fire Ball" {
		t.Fatalf("fire profile: %v %+v", err, pr)
	}
}

// necro1: the Necromancer summoner line is a second profile selected by name; the class default stays the Teeth line,
// and an unknown Necromancer profile is an error that names it, never another class or the default.
func TestNecromancerSummonerProfile(t *testing.T) {
	def, err := LookupProfile(d2.Necromancer, "")
	if err != nil || def.Build != "" || def.Bands[0].Attack[0] != "Bone Spear" {
		t.Fatalf("necromancer default changed: %v %+v", err, def)
	}
	sm, err := LookupProfile(d2.Necromancer, "summoner")
	if err != nil || sm.ProfileID() != "necromancer/summoner" {
		t.Fatalf("summoner profile: %v %+v", err, sm)
	}
	if _, ok := sm.Layout["Raise Skeleton"]; !ok {
		t.Errorf("summoner layout has no Raise Skeleton key: %v", sm.Layout)
	}
	// the summoner never ranks the Bone line as an attack
	for _, b := range sm.Bands {
		for _, a := range b.Attack {
			if a == "Bone Spear" {
				t.Errorf("summoner band attacks with %s", a)
			}
		}
	}
	for _, c := range []struct {
		class d2.Class
		id    Build
	}{{d2.Necromancer, "bone"}, {d2.Paladin, "summoner"}, {d2.Sorceress, "summoner"}, {d2.Class(42), "summoner"}} {
		if p, err := LookupProfile(c.class, c.id); err == nil || p != nil {
			t.Errorf("%v %q: got %v, want an error", c.class, c.id, p)
		}
	}
}

// hammer1: the Paladin Hammerdin line is a second profile selected by name; the class default stays the Zeal and
// Sacrifice line, and an unknown Paladin profile (or another class asking for "hammerdin") is an error that names it.
func TestPaladinHammerdinProfile(t *testing.T) {
	def, err := LookupProfile(d2.Paladin, "")
	if err != nil || def.Build != "" || def.Bands[1].Attack[0] != "Zeal" {
		t.Fatalf("paladin default changed: %v %+v", err, def)
	}
	hd, err := LookupProfile(d2.Paladin, "hammerdin")
	if err != nil || hd.ProfileID() != "paladin/hammerdin" {
		t.Fatalf("hammerdin profile: %v %+v", err, hd)
	}
	if hd.Bands[0].Attack[0] != "Blessed Hammer" {
		t.Errorf("hammerdin does not lead with Blessed Hammer: %v", hd.Bands[0].Attack)
	}
	for _, n := range []string{"Blessed Hammer", "Holy Shield", "Concentration", "Vigor"} {
		if _, ok := hd.Layout[n]; !ok {
			t.Errorf("hammerdin layout has no key for %s: %v", n, hd.Layout)
		}
	}
	// never a Zeal or Sacrifice attack, and no life price
	for _, b := range hd.Bands {
		for _, a := range b.Attack {
			if a == "Zeal" || a == "Sacrifice" {
				t.Errorf("hammerdin band attacks with %s", a)
			}
		}
	}
	if len(hd.LifePrice) != 0 {
		t.Errorf("hammerdin pays life for %v", hd.LifePrice)
	}
	for _, c := range []struct {
		class d2.Class
		id    Build
	}{{d2.Paladin, "zealot"}, {d2.Necromancer, "hammerdin"}, {d2.Sorceress, "hammerdin"}, {d2.Class(42), "hammerdin"}} {
		if p, err := LookupProfile(c.class, c.id); err == nil || p != nil {
			t.Errorf("%v %q: got %v, want an error", c.class, c.id, p)
		}
	}
}

// barb1: the Barbarian Frenzy line is a second profile selected by name; the class default stays the Bash, Double Swing
// and Frenzy-from-18 line, and an unknown Barbarian profile (or another class asking for "frenzy") is an error that names it.
func TestBarbarianFrenzyProfile(t *testing.T) {
	def, err := LookupProfile(d2.Barbarian, "")
	if err != nil || def.Build != "" || def.Bands[0].Attack[0] != "Frenzy" || def.Bands[0].MinLevel != 18 {
		t.Fatalf("barbarian default changed: %v %+v", err, def)
	}
	fr, err := LookupProfile(d2.Barbarian, "frenzy")
	if err != nil || fr.ProfileID() != "barbarian/frenzy" {
		t.Fatalf("frenzy profile: %v %+v", err, fr)
	}
	if fr.Bands[0].Attack[0] != "Frenzy" || fr.Bands[0].MinLevel != 24 {
		t.Errorf("frenzy does not lead with Frenzy from 24: %+v", fr.Bands[0])
	}
	for _, n := range []string{"Bash", "Double Swing", "Frenzy", "Battle Orders", "Shout"} {
		if _, ok := fr.Layout[n]; !ok {
			t.Errorf("frenzy layout has no key for %s: %v", n, fr.Layout)
		}
	}
	// no unsupported skill (Whirlwind, War Cry, Leap) in any band or buff, and no passive (Iron Skin) in the buff ladder
	for _, b := range fr.Bands {
		for _, a := range b.Attack {
			if a == "Whirlwind" || a == "Leap Attack" || a == "Double Throw" {
				t.Errorf("frenzy band attacks with %s", a)
			}
		}
	}
	for _, g := range fr.Buffs {
		for _, n := range g {
			if n == "Iron Skin" || n == "War Cry" {
				t.Errorf("frenzy buff ladder lists %s", n)
			}
		}
	}
	for _, c := range []struct {
		class d2.Class
		id    Build
	}{{d2.Barbarian, "whirlwind"}, {d2.Paladin, "frenzy"}, {d2.Necromancer, "frenzy"}, {d2.Sorceress, "frenzy"}, {d2.Class(42), "frenzy"}} {
		if p, err := LookupProfile(c.class, c.id); err == nil || p != nil {
			t.Errorf("%v %q: got %v, want an error", c.class, c.id, p)
		} else if !strings.Contains(err.Error(), string(c.id)) && c.class != d2.Class(42) {
			t.Errorf("%v %q: error %q does not name the profile", c.class, c.id, err)
		}
	}
}

// druid1: the Druid fire line is a second profile selected by name; the class default stays as it was, and an unknown
// Druid profile (or another class asking for "fire") is an error that names it. Data only: no ground-target code.
func TestDruidFireProfile(t *testing.T) {
	def, err := LookupProfile(d2.Druid, "")
	if err != nil || def.Build != "" || def.Bands[0].Attack[0] != "Eruption" || def.Bands[0].MinLevel != 12 {
		t.Fatalf("druid default changed: %v %+v", err, def)
	}
	fr, err := LookupProfile(d2.Druid, "fire")
	if err != nil || fr.ProfileID() != "druid/fire" {
		t.Fatalf("fire profile: %v %+v", err, fr)
	}
	if fr.Bands[0].Attack[0] != "Eruption" || fr.Bands[0].MinLevel != 12 || fr.Bands[0].Requires != "Eruption" {
		t.Errorf("fire does not lead with Eruption from 12: %+v", fr.Bands[0])
	}
	for _, n := range []string{"Firestorm", "Molten Boulder", "Eruption", "Cyclone Armor", "Oak Sage", "Volcano"} {
		if _, ok := fr.Layout[n]; !ok {
			t.Errorf("fire layout has no key for %s: %v", n, fr.Layout)
		}
	}
	// no unsupported skill (Tornado, Hurricane, Armageddon, a form skill) in any band or buff
	bad := map[string]bool{"Tornado": true, "Hurricane": true, "Armageddon": true, "Wearwolf": true, "Wearbear": true, "Maul": true, "Fury": true}
	for _, b := range fr.Bands {
		for _, a := range b.Attack {
			if bad[a] {
				t.Errorf("fire band attacks with %s", a)
			}
		}
	}
	for _, g := range fr.Buffs {
		for _, n := range g {
			if bad[n] {
				t.Errorf("fire buff ladder lists %s", n)
			}
		}
	}
	for _, c := range []struct {
		class d2.Class
		id    Build
	}{{d2.Druid, "wind"}, {d2.Paladin, "fire"}, {d2.Necromancer, "fire"}, {d2.Barbarian, "fire"}, {d2.Class(42), "fire"}} {
		if p, err := LookupProfile(c.class, c.id); err == nil || p != nil {
			t.Errorf("%v %q: got %v, want an error", c.class, c.id, p)
		} else if !strings.Contains(err.Error(), string(c.id)) && c.class != d2.Class(42) {
			t.Errorf("%v %q: error %q does not name the profile", c.class, c.id, err)
		}
	}
}

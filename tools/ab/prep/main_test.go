package main

import (
	"strings"
	"testing"

	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/scenario/hero"
)

type fakeTables struct{ ids map[string]int }

func (f *fakeTables) SkillID(name string) (int, bool) {
	if f.ids == nil {
		f.ids = map[string]int{"Attack": 0}
	}
	if _, ok := f.ids[name]; !ok {
		f.ids[name] = len(f.ids)
	}
	return f.ids[name], true
}
func (f *fakeTables) SkillIndex(_ string, id int) (int, bool) { return id, id > 0 }
func (f *fakeTables) Rule(string) leveler.Rule                { return nil }
func (f *fakeTables) StarterWeapon(string) string             { return "" }

func lines(v []hero.Verb) []string {
	var out []string
	for _, x := range v {
		out = append(out, x.Line)
	}
	return out
}

func TestVerbsBanked(t *testing.T) {
	_, v, err := Verbs(Setup{Hero: hero.Decl{Profile: hero.Banked, Class: "Sorceress", Level: 11, Weapon: "none"}, Gold: 500, MercType: 0}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(lines(v), "; ")
	for _, want := range []string{"debug stat 12 11", "debug statpts 50", "debug skillpts 10", "debug stat 14 500", "debug stat 15 0", "debug merc hire 0"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
	if strings.Contains(got, "cmd ") || strings.Contains(got, "debug equip") {
		t.Errorf("banked hero without weapon: %s", got)
	}
}

// The survival hero of both A/B classes is a played hero: its points are spent and its attack bound before the save.
func TestVerbsPlayedTwoClasses(t *testing.T) {
	for _, class := range []string{"Paladin", "Sorceress"} {
		h, v, err := Verbs(Setup{Hero: hero.Decl{Profile: hero.Played, Class: class, Level: 12, Weapon: "hax", QuestSkillPts: 1}, MercType: -1}, &fakeTables{})
		if err != nil {
			t.Fatal(err)
		}
		got := strings.Join(lines(v), "; ")
		for _, want := range []string{"debug equip hax 4", "cmd stat ", "cmd skillpt ", "cmd select "} {
			if !strings.Contains(got, want) {
				t.Errorf("%s: missing %q in %s", class, want, got)
			}
		}
		if strings.Contains(got, "merc hire") {
			t.Errorf("%s: merc -1 hired one", class)
		}
		r := Record{Run: "sa-margin211", Hero: h}
		if l := r.HeroLine(); !strings.HasPrefix(l, "hero: run=sa-margin211 profile=played") || !strings.Contains(l, "selfcheck=ok") {
			t.Errorf("hero line %q", l)
		}
	}
}

func TestClassName(t *testing.T) {
	if n, err := ClassName(2); err != nil || n != "Paladin" {
		t.Errorf("2: %q %v", n, err)
	}
	if _, err := ClassName(9); err == nil {
		t.Error("9: no error")
	}
}

func TestWeaponReadback(t *testing.T) {
	ok := `{"v":1,"stats":{"weapons":[{"code":"hax","slot":4,"mindam":3,"maxdam":6}],"hero":{"hands":[{"slot":4,"min":3,"max":6}]}}}`
	if s, err := WeaponReadback(ok, "hax"); err != nil || s == "" {
		t.Errorf("worn weapon: %q, %v", s, err)
	}
	if _, err := WeaponReadback(ok, "swd"); err == nil {
		t.Error("another code must fail")
	}
	old := `{"v":1,"stats":{"base":{"str":25}}}`
	if _, err := WeaponReadback(old, "hax"); err == nil {
		t.Error("a dump without weapons must fail when one was asked for")
	}
	if _, err := WeaponReadback(old, ""); err != nil {
		t.Errorf("no weapon asked: %v", err)
	}
}

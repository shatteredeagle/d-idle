package diffready

import (
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
)

func must(t *testing.T) *Table {
	t.Helper()
	tb, err := Shipped()
	if err != nil {
		t.Fatal(err)
	}
	return tb
}

func good(c d2.Class) Hero {
	return Hero{Class: c, Level: 45, MaxLife: 800, Res: d2.Resists{Fire: 60, Cold: 60, Lightning: 60, Poison: 40},
		Baal: Yes, KeySkillLevel: 25, Healing: 5, ManaOrRejuv: 5, Portals: 2, Merc: Yes}
}

func TestNightmareTwoClasses(t *testing.T) {
	tb, err := Shipped()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []d2.Class{d2.Sorceress, d2.Barbarian} {
		v, err := tb.Check("nightmare", good(c))
		if err != nil || !v.Ready {
			t.Fatalf("%v: %+v %v", c, v, err)
		}
	}
	// the same life is enough for a Sorceress (370) and not for a Barbarian (690)
	h := good(d2.Sorceress)
	h.MaxLife = 400
	if v, _ := tb.Check("nightmare", h); !v.Ready {
		t.Errorf("sorceress at 400 life: %s", v.Why())
	}
	h.Class = d2.Barbarian
	v, _ := tb.Check("nightmare", h)
	if v.Ready || !strings.Contains(v.Why(), "max life 400, need 690") {
		t.Errorf("barbarian at 400 life: %s", v.Why())
	}
}

func TestNightmareMissingList(t *testing.T) {
	tb := must(t)
	h := good(d2.Sorceress)
	h.Res.Fire = 40 // 40-40 = 0 post penalty, need 20
	h.Baal = No
	h.Merc = No
	h.Level = 30
	v, _ := tb.Check("nightmare", h)
	if v.Ready || len(v.Missing) != 4 {
		t.Fatalf("%+v", v)
	}
	for _, w := range []string{"Baal", "clvl 30, need 38", "fire resist 0 after the nightmare penalty, need 20", "mercenary"} {
		if !strings.Contains(v.Why(), w) {
			t.Errorf("%q missing from %s", w, v.Why())
		}
	}
}

func TestHellHasNoResistGate(t *testing.T) {
	tb := must(t)
	h := good(d2.Sorceress)
	h.Level, h.MaxLife = 80, 900
	h.Res = d2.Resists{} // raw 0: post penalty -100, would fail any resist gate
	v, err := tb.Check("hell", h)
	if err != nil || !v.Ready || v.Tag != "U" {
		t.Fatalf("%+v %v", v, err)
	}
	h.Level = 60
	if v, _ := tb.Check("hell", h); v.Ready {
		t.Error("clvl 60 must not be ready for Hell")
	}
}

func TestUnreadTermsAreListedNotBlocking(t *testing.T) {
	tb := must(t)
	h := good(d2.Sorceress)
	h.Baal, h.KeySkillLevel = Unread, -1
	v, _ := tb.Check("nightmare", h)
	if !v.Ready || len(v.Unknown) != 2 || !strings.Contains(v.Why(), "unread") {
		t.Errorf("%+v", v)
	}
}

// Must fail: unknown class or difficulty is an error that names it, never a fallback to the Sorceress row.
func TestUnknownErrors(t *testing.T) {
	tb := must(t)
	if _, err := tb.Check("nightmare", good(d2.Class(9))); err == nil || !strings.Contains(err.Error(), "Class(9)") {
		t.Errorf("class: %v", err)
	}
	if _, err := tb.Check("normal", good(d2.Sorceress)); err == nil || !strings.Contains(err.Error(), `"normal"`) {
		t.Errorf("difficulty: %v", err)
	}
	if _, err := Parse([]byte(`{"targets":{"hell":{"bogus":1}}}`)); err == nil {
		t.Error("unknown field must fail")
	}
}

func TestAllClassesHaveBothRows(t *testing.T) {
	tb := must(t)
	for c := d2.Amazon; c <= d2.Assassin; c++ {
		for _, d := range []string{"nightmare", "hell"} {
			if _, err := tb.Check(d, good(c)); err != nil {
				t.Errorf("%v %s: %v", c, d, err)
			}
		}
	}
}

func TestConsent(t *testing.T) {
	var c Consent
	if c.Take("nightmare") {
		t.Error("no go yet")
	}
	c.Grant("nightmare")
	if !c.Granted("hell") == false || !c.Granted("nightmare") || !c.Take("nightmare") || c.Take("nightmare") {
		t.Error("go is per target and consumed once")
	}
}

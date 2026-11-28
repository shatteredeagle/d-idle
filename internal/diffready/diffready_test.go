package diffready

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	now := time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC)
	if c.Take("nightmare", "Ann", now) {
		t.Error("no go yet")
	}
	c.Grant("nightmare", "Ann", now)
	if c.Granted("hell", "Ann", now) || !c.Granted("nightmare", "Ann", now) || !c.Take("nightmare", "Ann", now) || c.Take("nightmare", "Ann", now) {
		t.Error("go is per target and consumed once")
	}
}

// A go is for one hero and ends after GoTTL; the file carries both, and a foreign or ended go is dropped with a line.
func TestConsentHeroAndExpiry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "diffgo.json")
	now := time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC)
	var lines []string
	c := Consent{Logf: func(msg string, args ...any) { lines = append(lines, fmt.Sprint(append([]any{msg}, args...)...)) }}
	if err := c.Persist(path); err != nil {
		t.Fatal(err)
	}
	c.Grant("nightmare", "Ann", now)
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), `"hero":"Ann"`) || !strings.Contains(string(b), `"expires":"2026-12-04T12:00:00Z"`) {
		t.Fatalf("file lacks the hero or the 7-day end: %s", b)
	}
	if c.Take("nightmare", "Bob", now) {
		t.Fatal("another hero used the go")
	}
	if len(lines) != 1 || !strings.Contains(lines[0], "another hero") {
		t.Fatalf("log lines %q", lines)
	}
	// Fable review 12 C3: a look by another hero, or by a snapshot with no name, never erases the owner's go, and the
	// polled read logs the same refusal once.
	for i := 0; i < 3; i++ {
		if c.Granted("nightmare", "Bob", now) {
			t.Fatal("a foreign look used the go")
		}
	}
	if len(lines) != 1 {
		t.Fatalf("a polled foreign look must log once, got %q", lines)
	}
	if c.Granted("nightmare", "", now) {
		t.Fatal("a snapshot without a hero name used the go")
	}
	if !c.Granted("nightmare", "Ann", now) {
		t.Fatal("a look with another or no hero name erased the owner's go")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("the go's file was removed by a read")
	}
	lines = nil
	c.Keep("")
	c.Grant("hell", "Ann", now)
	var c2 Consent // a restart
	c2.Logf = c.Logf
	if err := c2.Persist(path); err != nil {
		t.Fatal(err)
	}
	if !c2.Granted("hell", "Ann", now.Add(GoTTL-time.Second)) {
		t.Fatal("a go inside its week was lost")
	}
	if c2.Take("hell", "Ann", now.Add(GoTTL)) || len(lines) != 1 || !strings.Contains(lines[0], "ended") {
		t.Fatalf("an ended go was used or not logged: %q", lines)
	}
	if err := os.WriteFile(path, []byte(`{"granted":["nightmare"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var c3 Consent
	if err := c3.Persist(path); err == nil || c3.Granted("nightmare", "Ann", now) {
		t.Fatal("the old file shape (no hero, no end) must not count as a go")
	}
}

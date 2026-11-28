package main

import (
	"bytes"
	"strings"
	"testing"

	"d2lod-bot/internal/scenario"
)

// The smoke tier on the shipped scenarios: the marker line, with four preps and four ledger lines.
func TestSmokeOnShippedData(t *testing.T) {
	var b bytes.Buffer
	if err := cmdSmoke([]string{"-dir", dir, "-golden", "../../" + defaultGolden}, &b); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(b.String(), "smoke: static ") || !strings.HasSuffix(b.String(), ", prep 4, ledger 4, blocked 0\n") {
		t.Fatal(b.String())
	}
}

// The must-fail arm: a d2d whose dump lacks the hands (the 2026-11-27 defect) blocks the played hero on the attack row
// and leaves the banked one alone, so a smoke that cannot fail is caught too.
func TestSmokePrepCatchesMissingHands(t *testing.T) {
	scs, err := loadAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range smokeRows {
		s, err := findScenario(scs, r.ID)
		if err != nil {
			t.Fatal(err)
		}
		res := smokePrep(s, r, &smokeGame{NoHands: true})
		played := s.HeroProfile() == scenario.HeroPlayed
		if played != (res.Status == scenario.Blocked) {
			t.Fatalf("%s (%s): %s %s", r.Key(), s.HeroProfile(), res.Status, res.Reason)
		}
		if played && !strings.Contains(res.Reason, "selfcheck: attack") {
			t.Fatalf("%s: %s", r.Key(), res.Reason)
		}
	}
}

// The pack model refuses what does not fit, as the lab d2d did for town.plan.belt-refill-from-pack.
func TestSmokeGamePackRoom(t *testing.T) {
	g := &smokeGame{}
	for i := 0; i < 10; i++ {
		if got := g.handle("debug give cap pack"); got != "ok\n" {
			t.Fatalf("cap %d: %q", i, got)
		}
	}
	if got := g.handle("debug give hp1 pack"); got != "err no room\n" {
		t.Fatalf("%q", got)
	}
	if got := g.handle("debug nosuchverb"); !strings.HasPrefix(got, "err unknown verb") {
		t.Fatalf("%q", got)
	}
}

// The golden files follow the scenarios: flipping one scenario's profile (what a changed default did to 207 of them)
// is named by the check, on two classes.
func TestSetupGoldenCatchesAProfileChange(t *testing.T) {
	scs, err := loadAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	golden := "../../" + defaultGolden
	if err := checkGolden(scs, golden); err != nil {
		t.Fatal(err)
	}
	for i, s := range scs {
		if s.ID != "town.resupply" { // Paladin and Sorceress, banked, level 8
			continue
		}
		scs[i].Pre.HeroProfile = scenario.HeroPlayed
		err := checkGolden(scs, golden)
		if err == nil || !strings.Contains(err.Error(), "town.resupply Paladin") || !strings.Contains(err.Error(), "town.resupply Sorceress") {
			t.Fatalf("a profile change went unseen: %v", err)
		}
		return
	}
	t.Fatal("town.resupply is gone: pick another scenario")
}

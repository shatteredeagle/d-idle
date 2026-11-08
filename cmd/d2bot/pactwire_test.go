package main

import (
	"context"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/pact"
	"errors"
	"strings"
	"testing"

	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/unlock"
	"d2lod-bot/internal/watch"
)

// Pacts are off in gates and measured runs: nothing is subscribed or started (a nil manager would panic otherwise).
func TestStartPactsOffInGatesAndMeasuredRuns(t *testing.T) {
	for _, env := range []map[string]string{{"D2BOT_PACTS": "off"}, {"D2BOT_RUN": "g2c"}} {
		startPacts(context.Background(), ladderOptions{}, &ladderRig{}, nil, nil, t.TempDir(), func(k string) string { return env[k] })
	}
}

func TestPactRefsNameTheMissingRow(t *testing.T) {
	cat, _ := decide.Shipped()
	tbl, err := decide.ShippedPolicy(cat)
	if err != nil {
		t.Fatal(err)
	}
	// a sheet needs a store; the refs only read the table through DefaultOption, which a bare table can answer
	st, err := decide.Open(t.TempDir(), "pact", cat, &decide.ManualClock{})
	if err != nil {
		t.Fatal(err)
	}
	ps, err := watch.NewPolicySheet(st, cat, tbl, func() string { return "paladin" }, func() float64 { return 0 }, watch.NewAwayLog(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	r := pactRefs(ps)
	if err := r.Option("repair", "low"); err != nil {
		t.Error(err)
	}
	if err := r.Option("nosuchrow", "x"); err == nil || !strings.Contains(err.Error(), "nosuchrow") {
		t.Errorf("unknown row: %v", err)
	}
}

// The desk behind /api/pacts: consent needs the chosen rank and the typed name; Hardcore has its own typed name (the
// hero's) and activation is refused, for two classes.
func TestPactDeskConsentAndHardcore(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		cat, err := pact.Shipped(pact.Refs{})
		if err != nil {
			t.Fatal(err)
		}
		ctl, err := pact.New(pact.Config{Cat: cat, Class: class})
		if err != nil {
			t.Fatal(err)
		}
		if err := ctl.Select([]pact.Pick{{ID: "blood_ravens_tithe", Rank: 1}}); err != nil {
			t.Fatal(err)
		}
		hc, _ := pact.OpenHardcore("", nil)
		d := &pactDesk{ctl: ctl, hc: hc}
		d.setHero(goals.Character{Name: "Zed", Class: class, Difficulty: "Nightmare"})
		if err := d.Consent("blood_ravens_tithe", 3, 4, "Blood Raven's Tithe"); err == nil {
			t.Errorf("%s: wrong rank accepted", class)
		}
		if err := d.Consent("blood_ravens_tithe", 1, 4, "Zed"); err == nil {
			t.Errorf("%s: the hero's name consented a Pact", class)
		}
		if err := d.Consent("blood_ravens_tithe", 1, 4, "Blood Raven's Tithe"); err != nil {
			t.Errorf("%s: %v", class, err)
		}
		if err := d.HardcoreConfirm("Blood Raven's Tithe"); err == nil {
			t.Errorf("%s: a Pact name confirmed hardcore", class)
		}
		if err := d.HardcoreConfirm("zed"); err != nil {
			t.Errorf("%s: %v", class, err)
		}
		if err := d.HardcoreActivate(); !errors.Is(err, watch.ErrPactInactive) {
			t.Errorf("%s: activate: %v", class, err)
		}
	}
}

func TestLadderDifficulty(t *testing.T) {
	if ladderDifficulty("Hell") != "hell" || ladderDifficulty("Nightmare") != "nightmare" || ladderDifficulty("") != "normal" {
		t.Error("ladder difficulty names")
	}
}

func TestChooseThroughDeskFollowsLadder(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		cat, _ := pact.Shipped(pact.Refs{})
		lad, err := unlock.Shipped()
		if err != nil {
			t.Fatal(err)
		}
		eng, _ := unlock.Open("", "t", lad, nil)
		ctl, _ := pact.New(pact.Config{Cat: cat, Class: class, Gate: eng})
		hc, _ := pact.OpenHardcore("", nil)
		d := &pactDesk{ctl: ctl, hc: hc}
		if err := d.Choose([]watch.PactPick{{ID: "blood_ravens_tithe", Rank: 1}}); err == nil || !strings.Contains(err.Error(), "locked") {
			t.Errorf("%s: locked Pact chosen: %v", class, err)
		}
		if _, err := eng.Observe(unlock.Event{Type: unlock.EvBossKill, Name: "Mephisto", Sub: "normal"}); err != nil {
			t.Fatal(err)
		}
		if err := d.Choose([]watch.PactPick{{ID: "blood_ravens_tithe", Rank: 1}}); err == nil {
			t.Errorf("%s: another boss opened the Pact", class)
		}
		eng.Observe(unlock.Event{Type: unlock.EvBossKill, Name: "Andariel", Sub: ladderDifficulty("Hell")})
		if err := d.Choose([]watch.PactPick{{ID: "blood_ravens_tithe", Rank: 1}}); err != nil {
			t.Errorf("%s: %v", class, err)
		}
		v, err := d.Catalogue("andariels_shadow:1")
		if err != nil || v.(pact.View).Proposed == nil {
			t.Errorf("%s: %v %v", class, v, err)
		}
		if _, err := d.Catalogue("junk"); err == nil {
			t.Errorf("%s: junk proposal accepted", class)
		}
		if v, _ := d.Catalogue("-"); v.(pact.View).Proposed == nil || len(v.(pact.View).Proposed.Picks) != 0 {
			t.Errorf("%s: the empty proposal", class)
		}
	}
}

func TestCostLineWithoutAndWithCounters(t *testing.T) {
	c := pact.CostLine{Name: "Griswold's Debt", Rank: 1, Effect: "repairs wait", Row: "repair", Start: 100, End: -1, Reward: pact.Reward{TallyPct: 5}}
	if l := costLine(pact.CostLine{Name: "X", Rank: 1, Effect: "e", Start: -1}, nil); !strings.Contains(l, "not acted yet") || !strings.Contains(l, "tally +0%") {
		t.Error(l)
	}
	if l := costLine(c, nil); !strings.Contains(l, "No counter") {
		t.Error(l)
	}
	cat, _ := decide.Shipped()
	tbl, _ := decide.ShippedPolicy(cat)
	st, _ := decide.Open(t.TempDir(), "pact", cat, &decide.ManualClock{})
	ps, err := watch.NewPolicySheet(st, cat, tbl, func() string { return "paladin" }, func() float64 { return 700 }, watch.NewAwayLog(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	if l := costLine(c, ps); !strings.Contains(l, "After, 10 min") || !strings.Contains(l, "Before, 10 min") {
		t.Error(l)
	}
}

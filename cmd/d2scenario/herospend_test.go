package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dctl/fake"
	"d2lod-bot/internal/scenario"
	"d2lod-bot/internal/scenario/hero"
)

// The default hero of a Proving Ground scenario is played (points spent, main attack bound), on two classes; banked is
// only the declared profile, a raw point write or a points scenario.
func TestHeroProfileDefaultPlayed(t *testing.T) {
	for _, class := range []string{"Sorceress", "Necromancer"} {
		s := fightScenario(12)
		if p := s.HeroProfile(); p != scenario.HeroPlayed {
			t.Fatalf("%s: default profile %q", class, p)
		}
		su, err := scenario.BuildSetup(s, class, 1)
		if err != nil || su.HeroProfile != scenario.HeroPlayed {
			t.Fatalf("%s: setup %q, %v", class, su.HeroProfile, err)
		}
		h, _ := scenarioHero(s, class, hero.NameTables())
		if h.SkillPts >= 11 || h.Attack == "" {
			t.Errorf("%s: played hero left %d skill points banked, attack %q", class, h.SkillPts, h.Attack)
		}
	}
	for name, mod := range map[string]func(*scenario.Scenario){
		"declared":   func(s *scenario.Scenario) { s.Pre.HeroProfile = scenario.HeroBanked },
		"raw points": func(s *scenario.Scenario) { s.Pre.Stats = []scenario.StatSet{{Stat: 5, Value: 3}} },
		"points":     func(s *scenario.Scenario) { s.Covers = []string{"points.autoalloc"} },
	} {
		s := fightScenario(12)
		mod(&s)
		if p := s.HeroProfile(); p != scenario.HeroBanked {
			t.Errorf("%s: profile %q, want banked", name, p)
		}
		if h, _ := scenarioHero(s, "Necromancer", hero.NameTables()); h.Attack != "" || h.SkillPts == 0 {
			t.Errorf("%s: banked hero has attack %q and %d skill points", name, h.Attack, h.SkillPts)
		}
	}
}

// spendPoints sends the player commands of the build table to the prep d2d for the played profile and nothing for
// the banked one (the old behaviour that left every skill unlearned).
func TestSpendPointsSendsBuildCommands(t *testing.T) {
	for _, class := range []string{"Sorceress", "Necromancer"} {
		for _, profile := range []string{scenario.HeroPlayed, scenario.HeroBanked} {
			path, cleanup, err := fake.TempSocket()
			if err != nil {
				t.Fatal(err)
			}
			srv, err := fake.New(path, fake.Sample(), fake.Options{})
			if err != nil {
				cleanup()
				t.Fatal(err)
			}
			c := d2dctl.New(path, d2dctl.Options{Timeout: 2 * time.Second})
			s := fightScenario(12)
			s.Pre.HeroProfile = profile
			su, _ := scenario.BuildSetup(s, class, 1)
			ledger := filepath.Join(t.TempDir(), "ledger.jsonl")
			err = spendPoints(context.Background(), c, s, class, su, hero.NameTables(), ledger)
			lines := strings.Join(srv.Lines(), "\n")
			c.Close()
			srv.Close()
			cleanup()
			if err != nil {
				t.Fatalf("%s %s: %v", class, profile, err)
			}
			has := strings.Contains(lines, "cmd skillpt")
			if has != (profile == scenario.HeroPlayed) || (profile == scenario.HeroPlayed && !strings.Contains(lines, "cmd select")) {
				t.Errorf("%s %s: commands %q", class, profile, lines)
			}
		}
	}
}

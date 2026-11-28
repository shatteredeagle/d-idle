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

// A scenario without pre.hero_profile is banked (as before the played profile existed); played is opt-in, on two classes.
func TestHeroProfileDefaultBanked(t *testing.T) {
	for _, class := range []string{"Sorceress", "Necromancer", "Paladin"} {
		s := fightScenario(12)
		if p := s.HeroProfile(); p != scenario.HeroBanked {
			t.Fatalf("%s: default profile %q", class, p)
		}
		su, err := scenario.BuildSetup(s, class, 1)
		if err != nil || su.HeroProfile != scenario.HeroBanked {
			t.Fatalf("%s: setup %q, %v", class, su.HeroProfile, err)
		}
		if h, _ := scenarioHero(s, class, hero.NameTables()); h.Attack != "" || h.SkillPts != 11 {
			t.Errorf("%s: banked hero has attack %q and %d skill points", class, h.Attack, h.SkillPts)
		}
		s.Pre.HeroProfile = scenario.HeroPlayed
		if h, _ := scenarioHero(s, class, hero.NameTables()); class != "Paladin" && (h.SkillPts >= 11 || h.Attack == "") {
			t.Errorf("%s: played hero left %d skill points banked, attack %q", class, h.SkillPts, h.Attack)
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

// A run blocked before the setup ran (a failed self-check) still gets its ledger line: the tag is the scenario's own.
func TestBlockedRunLedgerTagFromScenario(t *testing.T) {
	for _, lvl := range []int{12, 1} {
		su, err := scenario.BuildSetup(fightScenario(lvl), "Paladin", 11)
		if err != nil || scenario.SetupTag(su.Tags) == "" {
			t.Fatalf("level %d: tags %v, %v", lvl, su.Tags, err)
		}
		if lvl == 12 && scenario.SetupTag(su.Tags) != scenario.SetupCheated {
			t.Errorf("level setup writes are CHEATED, got %v", su.Tags)
		}
	}
}

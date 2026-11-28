package main

import (
	"strings"
	"testing"

	"d2lod-bot/internal/scenario"
	"d2lod-bot/internal/scenario/hero"
)

func fightScenario(level int) scenario.Scenario {
	return scenario.Scenario{ID: "t.fight", Covers: []string{"combat.fight"},
		Pre:     scenario.Pre{HeroProfile: scenario.HeroBanked, Level: level, Gold: 2000, Items: []scenario.Item{{Code: "hp1", Count: 4}, {Code: "tsc"}}, Belt: []string{"hp1", "mp1"}},
		Trigger: scenario.Trigger{Goals: "reach:20"}}
}

// The static half runs offline on two classes; each failing row has its own must-fail case.
func TestStaticCheckRows(t *testing.T) {
	for _, class := range []string{"Sorceress", "Paladin"} {
		s := fightScenario(12)
		su, err := scenario.BuildSetup(s, class, 1)
		if err != nil {
			t.Fatal(err)
		}
		if err := staticCheck(s, class, su, hero.NameTables()); err != nil {
			t.Errorf("%s: %v", class, err)
		}

		met := fightScenario(20)
		if err := staticCheck(met, class, su, hero.NameTables()); err == nil || !strings.Contains(err.Error(), "goal:") {
			t.Errorf("%s: reach goal met at the start: %v", class, err)
		}
		met.Trigger.Driver = "board-taps" // the board drives the bot: the goal is only a fallback
		if err := staticCheck(met, class, su, hero.NameTables()); err != nil {
			t.Errorf("%s: board-driven: %v", class, err)
		}

		bad := su
		bad.Verbs = append([]scenario.Verb{{Line: "debug give rin 4"}}, su.Verbs...)
		bad.Verbs = append(bad.Verbs, scenario.Verb{Line: "debug goto 8"})
		if err := staticCheck(s, class, bad, hero.NameTables()); err == nil || !strings.Contains(err.Error(), "order:") {
			t.Errorf("%s: ground item before the position verb: %v", class, err)
		}

		other := fightScenario(12)
		other.Trigger.Build = "no-such-build"
		if err := staticCheck(other, class, su, hero.NameTables()); err == nil || !strings.Contains(err.Error(), "build:") {
			t.Errorf("%s: unknown build: %v", class, err)
		}
	}
	nec := fightScenario(4)
	nec.Pre.HeroProfile = scenario.HeroPlayed // the attack row is the played profile's
	nec.Trigger.Build = "necromancer_summoner_normal30"
	su, _ := scenario.BuildSetup(nec, "Necromancer", 1)
	if err := staticCheck(nec, "Necromancer", su, hero.NameTables()); err == nil || !strings.Contains(err.Error(), "attack:") {
		t.Errorf("summoner clvl 4 fighting without its main attack: %v", err)
	}
	nec.Pre.HeroProfile = scenario.HeroBanked // banked: no attack row
	if err := staticCheck(nec, "Necromancer", su, hero.NameTables()); err != nil {
		t.Errorf("a points scenario is banked, no attack row: %v", err)
	}
}

// The runtime half: the hero the scenario declares passes against the state a correct prep leaves, and must fail when
// the points were spent (or banked away), the purse is empty or the potions are missing.
func TestScenarioHeroSelfCheck(t *testing.T) {
	for _, class := range []string{"Sorceress", "Paladin"} {
		s := fightScenario(12)
		// fightScenario is banked: the state below is a banked hero's
		h, f := scenarioHero(s, class, hero.NameTables())
		if f.Purse != 2000 || f.Potions != 6 {
			t.Fatalf("floors %+v", f)
		}
		lvl, sp, kp, gold, pots := 12, 55, 11, 2000, 6
		o := hero.Observed{Level: &lvl, StatPts: &sp, SkillPts: &kp, Gold: &gold, Potions: &pots}
		if bad := hero.Failed(h.SelfCheckFloors(o, f)); bad != "" {
			t.Errorf("%s: %s", class, bad)
		}
		spent, none := 0, 0
		o.SkillPts, o.Gold = &spent, &none
		if bad := hero.Failed(h.SelfCheckFloors(o, f)); bad != "skillpts,purse" {
			t.Errorf("%s: failed %q", class, bad)
		}
	}
	s := fightScenario(12)
	s.Pre.Stats = []scenario.StatSet{{Stat: 4, Value: 0}}
	if h, _ := scenarioHero(s, "Paladin", hero.NameTables()); h.StatPts != 0 {
		t.Errorf("a raw statpts write sets the expectation: %d", h.StatPts)
	}
}

// The pack row: ten caps and twelve potions (the first town.plan.belt-refill-from-pack) cannot fit 40 cells and are
// refused at T0, on two classes; the shipped five caps pass, and the fake d2d of the smoke prepares that hero.
func TestStaticCheckPackRoom(t *testing.T) {
	for _, class := range []string{"Paladin", "Sorceress"} {
		s := fightScenario(16)
		s.Pre.Items = []scenario.Item{{Code: "hp1", Count: 8}, {Code: "mp1", Count: 4}, {Code: "cap", Count: 10}}
		su, err := scenario.BuildSetup(s, class, 1)
		if err != nil {
			t.Fatal(err)
		}
		if err := staticCheck(s, class, su, hero.NameTables()); err == nil || !strings.Contains(err.Error(), "pack: the declared pack items need at least 52 cells") {
			t.Errorf("%s: an overfull pack: %v", class, err)
		}
		s.Pre.Items[2].Count = 5
		if err := staticCheck(s, class, su, hero.NameTables()); err != nil {
			t.Errorf("%s: 32 cells: %v", class, err)
		}
	}
	scs, err := loadAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	s, err := findScenario(scs, "town.plan.belt-refill-from-pack")
	if err != nil {
		t.Fatal(err)
	}
	for _, class := range s.Matrix.Classes {
		if res := smokePrep(s, scenario.Run{ID: s.ID, Class: class, Seed: 11}, &smokeGame{}); res.Status == scenario.Blocked {
			t.Errorf("%s: %s", class, res.Reason)
		}
	}
}

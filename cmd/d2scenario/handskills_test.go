package main

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/scenario"
	"d2lod-bot/internal/scenario/hero"
)

// handRows are shipped drills with a pre.bind, a pre.summon or both, on four classes.
var handRows = []scenario.Run{
	{ID: "combat.whirlwind-clears", Class: "Barbarian", Seed: 11},
	{ID: "combat.hold-interruptible", Class: "Paladin", Seed: 11},
	{ID: "combat.corpse-explodes", Class: "Necromancer", Seed: 11},
	{ID: "combat.pets-hold", Class: "Druid", Seed: 11},
}

// The prep half of the drills on the fake d2d: the declared hand skill and the setup summon pass the self-check, and
// each has its must-fail arm (a d2d that ignores the select, one that summons nothing): the run is BLOCKED naming the
// row, never handed to the bot with the wrong hero.
func TestPrepBindAndSummon(t *testing.T) {
	scs, err := loadAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	binds, summons := 0, 0
	for _, r := range handRows {
		s, err := findScenario(scs, r.ID)
		if err != nil {
			t.Fatal(err)
		}
		if res := smokePrep(s, r, &smokeGame{}); res.Status == scenario.Blocked {
			t.Errorf("%s: BLOCKED %s", r.Key(), res.Reason)
		} else if !slices.Contains(res.Tags, "CHEATED") {
			t.Errorf("%s: tags %v lack CHEATED", r.Key(), res.Tags)
		}
		if len(s.Pre.Bind) > 0 {
			binds++
			res := smokePrep(s, r, &smokeGame{DeafSelect: true})
			if res.Status != scenario.Blocked || !strings.Contains(res.Reason, "selfcheck: ") || !strings.Contains(res.Reason, "bind") {
				t.Errorf("%s: a d2d that ignores cmd select: %s %s", r.Key(), res.Status, res.Reason)
			}
		}
		if len(s.Pre.Summon) > 0 {
			summons++
			res := smokePrep(s, r, &smokeGame{NoSummon: true})
			if res.Status != scenario.Blocked || !strings.Contains(res.Reason, "selfcheck: ") || !strings.Contains(res.Reason, "minions") {
				t.Errorf("%s: a d2d that summons nothing: %s %s", r.Key(), res.Status, res.Reason)
			}
		}
	}
	if binds < 2 || summons < 2 {
		t.Fatalf("%d bind rows and %d summon rows: want two classes of each", binds, summons)
	}
}

// otherClass are tables in which no skill belongs to the hero's class.
type otherClass struct{ hero.Tables }

func (otherClass) SkillIndex(string, int) (int, bool) { return 0, false }

// summonMinions casts with the player's command, N times, each cast a SETUP ledger line; a skill that is not the
// class's is an error naming the skill and the class, and nothing is cast. Two classes.
func TestSummonMinionsCastsAndLedgers(t *testing.T) {
	for _, c := range []struct{ class, skill string }{{"Necromancer", "Clay Golem"}, {"Druid", "Oak Sage"}} {
		g, err := startSmokeGame(c.class)
		if err != nil {
			t.Fatal(err)
		}
		cl := d2dctl.New(g.Path(), d2dctl.Options{Timeout: 2 * time.Second})
		s := scenario.Scenario{ID: "x.summon", Pre: scenario.Pre{HeroProfile: scenario.HeroPlayed, Level: 24, Start: scenario.StartField, Area: 2,
			Summon: []scenario.Summon{{Skill: c.skill, N: 3}}}}
		su, err := scenario.BuildSetup(s, c.class, 1)
		if err != nil {
			t.Fatal(err)
		}
		ledger := filepath.Join(t.TempDir(), "ledger.jsonl")
		tabs := hero.NameTables()
		id, _ := tabs.SkillID(c.skill)
		if err := summonMinions(context.Background(), cl, s, c.class, su, otherClass{tabs}, ledger); err == nil ||
			!strings.Contains(err.Error(), c.skill) || !strings.Contains(err.Error(), c.class) {
			t.Errorf("%s: a skill of another class: %v", c.class, err)
		}
		if g.minions != 0 {
			t.Errorf("%s: %d casts before the class check", c.class, g.minions)
		}
		if err := summonMinions(context.Background(), cl, s, c.class, su, tabs, ledger); err != nil {
			t.Fatalf("%s: %v", c.class, err)
		}
		cl.Close()
		g.Close()
		b, _ := os.ReadFile(ledger)
		if n := strings.Count(string(b), `"reason":"SETUP: scenario x.summon`); n != 3 || g.minions != 3 {
			t.Errorf("%s: %d SETUP ledger lines, %d casts, want 3 and 3:\n%s", c.class, n, g.minions, b)
		}
		casts := 0
		for _, l := range g.lines {
			if f := strings.Fields(l); len(f) == 5 && f[0] == "cmd" && f[1] == "skill" {
				casts++
				if f[2] != strconv.Itoa(id) || f[3] != "32.5" || f[4] != "20.5" { // two cells east of the model's hero
					t.Errorf("%s: cast line %q", c.class, l)
				}
			}
		}
		if casts != 3 {
			t.Errorf("%s: %d cast lines in %v", c.class, casts, g.lines)
		}
	}
}

// The static half refuses a bind or a summon whose skill the build never learns when no setup points are declared,
// and playedHero names a skill the tables do not know. Two classes.
func TestStaticCheckHandSkills(t *testing.T) {
	for _, class := range []string{"Necromancer", "Sorceress"} {
		c := struct{ class, learned string }{class: class}
		s := fightScenario(12)
		s.Pre.HeroProfile, s.Pre.Start, s.Pre.Area = scenario.HeroPlayed, scenario.StartField, 2
		if h, err := playedHero(s, class, hero.NameTables()); err != nil || h.Attack == "" {
			t.Fatalf("%s: %v", class, err)
		} else {
			c.learned = h.Attack // the build's own attack: a skill the plan gives a point
		}
		s.Pre.Bind = []scenario.HandBind{{Skill: c.learned}}
		su, err := scenario.BuildSetup(s, c.class, 1)
		if err != nil {
			t.Fatal(err)
		}
		if err := staticCheck(s, c.class, su, hero.NameTables()); err != nil {
			t.Errorf("%s: a learned skill refused: %v", c.class, err)
		}
		s.Pre.Bind = []scenario.HandBind{{Skill: "No Such Skill"}}
		if err := staticCheck(s, c.class, su, hero.NameTables()); err == nil || !strings.Contains(err.Error(), "pre.bind: No Such Skill gets no point") {
			t.Errorf("%s: an unlearned bind: %v", c.class, err)
		}
		s.Pre.Bind, s.Pre.Summon = nil, []scenario.Summon{{Skill: "No Such Summon"}}
		if err := staticCheck(s, c.class, su, hero.NameTables()); err == nil || !strings.Contains(err.Error(), "pre.summon: No Such Summon gets no point") {
			t.Errorf("%s: an unlearned summon: %v", c.class, err)
		}
		s.Pre.Bind, s.Pre.Summon = []scenario.HandBind{{Skill: c.learned}}, nil
		if _, err := playedHero(s, c.class, otherClass{hero.NameTables()}); err == nil {
			t.Errorf("%s: the build's skills are not the class's and the hero resolved", c.class)
		}
	}
}

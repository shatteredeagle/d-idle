package main

import (
	"encoding/json"
	"testing"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/gear"
)

func TestPlanForClassJoinsCombatAndLeveler(t *testing.T) {
	cases := []struct {
		class   d2.Class
		cfg     string
		profile string
		build   combat.Build
		lv      string
	}{
		{d2.Paladin, "", "paladin", "paladin", "paladin-normal30"}, // the table names the class default profile (review 8 fix 4)
		{d2.Sorceress, "", "sorceress/lightning", combat.BuildLightning, "fallback"},
		{d2.Sorceress, "primary", "sorceress/blizzard", combat.BuildBlizzard, "primary"},
	}
	for _, c := range cases {
		p := planForClass(c.class, c.cfg, 12, map[string]int{"Static Field": 1, "Holy Fire": 2})
		if p.Profile != c.profile || p.CombatBuild != c.build || p.LevelerBuild != c.lv {
			t.Errorf("%s/%q: %+v", c.class, c.cfg, p)
		}
	}
	// the barbarian has a leveler table since agent/barblive
	if p := planForClass(d2.Barbarian, "", 1, nil); p.LevelerBuild == "" || p.Profile == "" {
		t.Errorf("barbarian %+v", p)
	}
	// BUILDV3: an explicit table name selects the lightning combat line from the table's own data key
	if p := planForClass(d2.Sorceress, "fallback-lightning-v3", 1, nil); p.LevelerBuild != "fallback-lightning-v3" || p.CombatBuild != combat.BuildLightning {
		t.Errorf("v3 by name: %+v", p)
	}
	// BUILDV2: a fresh character gets the v2 tables, one that owns a skill the new table dropped keeps the old one
	for _, c := range []struct {
		class d2.Class
		lvl   int
		owned map[string]int
		lv    string
	}{
		{d2.Sorceress, 1, nil, "sorceress_normal30"},
		{d2.Sorceress, 4, map[string]int{"Charged Bolt": 3}, "sorceress_normal30"},
		{d2.Sorceress, 12, map[string]int{"Charged Bolt": 9, "Static Field": 1}, "fallback"},
		{d2.Paladin, 1, nil, "paladin-normal30-v2"},
		{d2.Paladin, 9, map[string]int{"Might": 3, "Holy Fire": 3}, "paladin-normal30"},
	} {
		p := planForClass(c.class, "", c.lvl, c.owned)
		if p.LevelerBuild != c.lv {
			t.Errorf("%s clvl %d: table %q, want %q", c.class, c.lvl, p.LevelerBuild, c.lv)
		}
		want := combat.BuildLightning // the legacy "fallback" table; sorceress_normal30 names the cold line
		if c.lv == "sorceress_normal30" {
			want = combat.Build("cold")
		}
		if c.class == d2.Sorceress && p.CombatBuild != want {
			t.Errorf("sorceress %s: combat build %v", c.lv, p.CombatBuild)
		}
	}
}

func TestPlanForClassProfileFromTable(t *testing.T) {
	// the END Sorceress table declares no combat key: Blizzard profile; the v2 and v3 lightning tables declare it
	if p := planForClass(d2.Sorceress, "sorceress_blizzard_endgame90", 30, nil); p.Err != nil || p.Profile != "sorceress/blizzard" {
		t.Errorf("end table: %+v", p)
	}
	if p := planForClass(d2.Sorceress, "fallback-lightning-v2", 1, nil); p.Err != nil || p.Profile != "sorceress/lightning" {
		t.Errorf("v2: %+v", p)
	}
	// a Paladin selected by table: its own single profile, no error
	if p := planForClass(d2.Paladin, "paladin-normal30", 5, nil); p.Err != nil || p.Profile != "paladin" {
		t.Errorf("paladin: %+v", p)
	}
	// an unknown class is an error in the plan, not a Sorceress profile
	if p := planForClass(d2.Class(42), "", 1, nil); p.Err == nil || p.Profile != "" {
		t.Errorf("unknown class: %+v", p)
	}
}

// A default table's main attack must have a key in the combat profile its class plan resolves: a skill without a layout
// key is never bound and so never cast (the Necromancer default table named Corpse Explosion as its main attack from
// clvl 15 but carried no "combat" key, so it fought with the layout that has no Corpse Explosion key; card necro1).
func TestDefaultTableMainAttackHasAKeyInItsProfile(t *testing.T) {
	for _, c := range []d2.Class{d2.Necromancer, d2.Druid} {
		p := planForClass(c, "", 1, nil)
		if p.Err != nil {
			t.Fatalf("%s: %v", c, p.Err)
		}
		raw, err := gear.BuildJSON(p.LevelerBuild)
		if err != nil {
			t.Fatalf("%s: %v", p.LevelerBuild, err)
		}
		var tbl struct {
			Main []gear.MainAttack `json:"main_attack"`
		}
		if err := json.Unmarshal(raw, &tbl); err != nil {
			t.Fatal(err)
		}
		prof, err := combat.LookupProfile(c, p.CombatBuild)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range tbl.Main {
			if _, ok := prof.Layout[m.Skill]; !ok {
				t.Errorf("%s table %s: main attack %q has no key in profile %s", c, p.LevelerBuild, m.Skill, prof.ProfileID())
			}
		}
	}
}

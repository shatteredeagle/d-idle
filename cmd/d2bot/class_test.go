package main

import (
	"testing"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
)

func TestPlanForClassJoinsCombatAndLeveler(t *testing.T) {
	cases := []struct {
		class   d2.Class
		cfg     string
		profile string
		build   combat.Build
		lv      string
	}{
		{d2.Paladin, "", "paladin", combat.BuildBlizzard, "paladin-normal30"},
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
	// BUILDV2: a fresh character gets the v2 tables, one that owns a skill the new table dropped keeps the old one
	for _, c := range []struct {
		class d2.Class
		lvl   int
		owned map[string]int
		lv    string
	}{
		{d2.Sorceress, 1, nil, "fallback-lightning-v2"},
		{d2.Sorceress, 4, map[string]int{"Charged Bolt": 3}, "fallback-lightning-v2"},
		{d2.Sorceress, 12, map[string]int{"Charged Bolt": 9, "Static Field": 1}, "fallback"},
		{d2.Paladin, 1, nil, "paladin-normal30-v2"},
		{d2.Paladin, 9, map[string]int{"Might": 3, "Holy Fire": 3}, "paladin-normal30"},
	} {
		p := planForClass(c.class, "", c.lvl, c.owned)
		if p.LevelerBuild != c.lv {
			t.Errorf("%s clvl %d: table %q, want %q", c.class, c.lvl, p.LevelerBuild, c.lv)
		}
		if c.class == d2.Sorceress && p.CombatBuild != combat.BuildLightning {
			t.Errorf("sorceress %s: combat build %v", c.lv, p.CombatBuild)
		}
	}
}

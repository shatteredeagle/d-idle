package main

import (
	"strings"
	"testing"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
)

// sorcready2 (user decision 2026-11-10): a fresh Sorceress defaults to sorceress_normal30, whose table names the cold
// combat profile; a fresh Paladin keeps paladin-normal30-v2 (sorcready had pinned the old v2 default).
func TestFreshSorceressDefaultsToNormal30(t *testing.T) {
	p := planForClass(d2.Sorceress, "", 1, nil)
	if p.Err != nil || p.LevelerBuild != "sorceress_normal30" || p.CombatBuild != combat.Build("cold") || p.Profile != "sorceress/cold" {
		t.Errorf("fresh sorceress %+v", p)
	}
	if p := planForClass(d2.Paladin, "", 1, nil); p.LevelerBuild != "paladin-normal30-v2" || p.Err != nil || p.Profile != "paladin" {
		t.Errorf("fresh paladin %+v", p)
	}
}

// A table written for another class is refused, and the error names both classes (two classes, both directions).
func TestPlanForClassRefusesAnotherClassesTable(t *testing.T) {
	for _, c := range []struct {
		class d2.Class
		table string
		owner string
	}{
		{d2.Paladin, "sorceress_normal30", "sorceress"},
		{d2.Sorceress, "paladin-normal30-v2", "paladin"},
	} {
		p := planForClass(c.class, c.table, 1, nil)
		if p.Err == nil || !strings.Contains(p.Err.Error(), c.class.String()) || !strings.Contains(p.Err.Error(), c.owner) || !strings.Contains(p.Err.Error(), c.table) {
			t.Errorf("%s on %s: %v", c.class, c.table, p.Err)
		}
	}
}

package main

import (
	"testing"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
)

// sorcready: the setup wizard's class pick (D2D_CLASS) never writes a leveler table (wizard.env holds name, class
// index and goals only, the wizard.env assertions in cmd/d2dsetup/d2dsetup_test.go), so a fresh Sorceress plays the BUILDV2 table
// fallback-lightning-v2 and a fresh Paladin paladin-normal30-v2. sorceress_normal30 is used only when config
// levelerBuild names it; then the table's combat key picks the lightning profile. Pinned here so a change to the
// default is a visible decision (it is an open question for the user, docs/sorceress-early.md).
func TestClassPickDoesNotSelectNormal30(t *testing.T) {
	if p := planForClass(d2.Sorceress, "", 1, nil); p.LevelerBuild != "fallback-lightning-v2" || p.Err != nil {
		t.Errorf("fresh sorceress %+v", p)
	}
	if p := planForClass(d2.Paladin, "", 1, nil); p.LevelerBuild != "paladin-normal30-v2" || p.Err != nil {
		t.Errorf("fresh paladin %+v", p)
	}
	p := planForClass(d2.Sorceress, "sorceress_normal30", 1, nil)
	if p.Err != nil || p.LevelerBuild != "sorceress_normal30" || p.CombatBuild != combat.BuildLightning || p.Profile != "sorceress/lightning" {
		t.Errorf("explicit sorceress_normal30 %+v", p)
	}
}

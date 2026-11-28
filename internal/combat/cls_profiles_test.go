package combat

import (
	"testing"

	"d2lod-bot/internal/d2"
)

// Card cls1: the stance of a melee build is its own data. The War Cry Barbarian stands in the pack (the nova needs the
// monsters on top of the hero, S docs/probes/sme-other-classes-review-2026-11-27.md Barbarian change 3), the Frenzy
// Barbarian keeps the single-target numbers; the Druid's two builds lead with different schools at the same level.
func TestBuildsOfOneClassCarryTheirOwnStance(t *testing.T) {
	war, frenzy := ProfileFor(d2.Barbarian, "warcry"), ProfileFor(d2.Barbarian, "frenzy")
	if war == nil || frenzy == nil || war.Melee == nil || frenzy.Melee == nil {
		t.Fatal("barbarian warcry and frenzy profiles need a melee block")
	}
	if war.Melee.Surround <= frenzy.Melee.Surround || war.Melee.SurroundLifePct >= frenzy.Melee.SurroundLifePct {
		t.Errorf("warcry backs out of a surround at %d monsters under %d%% life, frenzy at %d under %d%%: the nova build must hold longer",
			war.Melee.Surround, war.Melee.SurroundLifePct, frenzy.Melee.Surround, frenzy.Melee.SurroundLifePct)
	}
	all := func(string) bool { return true }
	fire, tornado := ProfileFor(d2.Druid, "fire"), ProfileFor(d2.Druid, "tornado")
	if fire == nil || tornado == nil {
		t.Fatal("druid fire and tornado profiles are missing")
	}
	if fire.Melee != nil || tornado.Melee != nil {
		t.Error("a druid caster profile carries a melee block")
	}
	f, w := fire.AttackNames(24, all), tornado.AttackNames(24, all)
	if len(f) == 0 || len(w) == 0 || f[0] == w[0] {
		t.Errorf("druid builds lead with the same attack at clvl 24: fire %v, tornado %v", f, w)
	}
}

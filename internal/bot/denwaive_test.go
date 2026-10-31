package bot

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goals"
)

// A clvl 1 Sorceress whose only open field (Blood Moor) is marked cleared must not be sent into the Den of Evil: the
// floor yield is for a character at or near its floor, not for a level 1 caster (D2DPACE, 4 live d2d sessions).
func TestDenFloorDoesNotYieldFarBelowFloor(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 1)
	g.r.e.Tune.Campaign.DenMinLevelCaster, g.r.e.Tune.Campaign.DenMinLevelMelee = 5, 3
	g.r.poll()
	s := g.r.latest()
	s.Player.Class, s.Player.Difficulty = d2.Sorceress, d2.Normal
	g.r.loop.cleared = map[int]bool{2: true}
	a, ok := g.r.target0(s, control{spec: goalSpec{kind: goals.ReachLevel}})
	if ok && a == areaDenOfEvil {
		t.Fatalf("clvl 1 caster sent to the Den (area %d)", a)
	}
}

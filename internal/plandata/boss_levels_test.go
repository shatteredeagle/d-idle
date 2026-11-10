package plandata

import (
	"strings"
	"testing"
)

func TestBossLevelsRows(t *testing.T) {
	for _, c := range []string{"paladin", "sorceress", "necromancer", "amazon", "druid", "assassin", "barbarian"} {
		r, err := BossLevels(GateAndariel, c)
		if err != nil {
			t.Fatalf("%s: %v", c, err)
		}
		if r.RecommendedMerc > r.Recommended || r.FloorMerc > r.Floor || r.Floor > r.Recommended || r.FloorMerc > r.RecommendedMerc || r.FloorMerc <= 0 {
			t.Errorf("%s: inconsistent row %+v", c, r)
		}
	}
	if r, _ := BossLevels(GateAndariel, "paladin"); r.Recommended != 17 || r.RecommendedMerc != 15 {
		t.Errorf("paladin starting data %+v, want 17 / 15", r)
	}
	if r, _ := BossLevels(GateAndariel, "sorceress"); r.Recommended <= 17 {
		t.Errorf("sorceress is the weaker class at Andariel, got %+v", r)
	}
}

func TestBossLevelsUnknownNamesIt(t *testing.T) {
	if _, err := BossLevels(GateAndariel, "warlock"); err == nil || !strings.Contains(err.Error(), "warlock") {
		t.Errorf("unknown class: %v", err)
	}
	if _, err := BossLevels("mephisto", "paladin"); err == nil || !strings.Contains(err.Error(), "mephisto") {
		t.Errorf("unknown gate: %v", err)
	}
	if LowestFloor(GateAndariel) != 13 {
		t.Errorf("lowest floor %d", LowestFloor(GateAndariel))
	}
}

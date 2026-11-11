package combat

import (
	"testing"

	"d2lod-bot/internal/d2"
)

// 12 Fallen in a ring around the hero and a shaman behind them: a melee class swings at the nearest, a ranged class
// still opens on the shaman (it casts over the pack).
func TestRankShamanBehindThePack(t *testing.T) {
	var ts []Target
	for i := 0; i < 12; i++ {
		ts = append(ts, Target{Unit: d2.Unit{ID: uint32(10 + i)}, Name: "Fallen", Dist: 3 + float64(i%3)})
	}
	ts = append(ts, Target{Unit: d2.Unit{ID: 99}, Name: "Fallen Shaman", Dist: 14, KillFast: true})
	for _, tc := range []struct {
		class string
		melee bool
		want  uint32
	}{{"Paladin", true, 10}, {"Sorceress", false, 99}} {
		got := RankFor(ts, 100, 0, tc.melee)[0].Unit.ID
		if got != tc.want {
			t.Errorf("%s: first target %d, want %d", tc.class, got, tc.want)
		}
	}
	// the shaman within reach still comes first for a melee class
	near := append([]Target(nil), ts...)
	near[12].Dist = 6
	if got := RankFor(near, 100, 0, true)[0].Unit.ID; got != 99 {
		t.Errorf("shaman in reach: first target %d, want 99", got)
	}
}

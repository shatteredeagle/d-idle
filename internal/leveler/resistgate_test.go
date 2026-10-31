package leveler

import (
	"testing"

	"d2lod-bot/internal/d2"
)

func TestResistGate(t *testing.T) {
	for _, c := range []struct {
		name string
		d    d2.Difficulty
		r    d2.Resists
		ok   bool
	}{
		{"normal passes naked", d2.Normal, d2.Resists{}, true},
		{"nightmare 60 raw is +20", d2.Nightmare, d2.Resists{Fire: 60, Cold: 60, Lightning: 60, Poison: 40}, true},
		{"nightmare 59 fire is +19", d2.Nightmare, d2.Resists{Fire: 59, Cold: 60, Lightning: 60, Poison: 40}, false},
		{"nightmare poison 39 is -1", d2.Nightmare, d2.Resists{Fire: 75, Cold: 75, Lightning: 75, Poison: 39}, false},
		{"hell 125 raw is +25", d2.Hell, d2.Resists{Fire: 125, Cold: 125, Lightning: 125, Poison: 100}, true},
		{"hell 124 cold is +24", d2.Hell, d2.Resists{Fire: 125, Cold: 124, Lightning: 125, Poison: 100}, false},
		{"hell maxed 75 raw is -25", d2.Hell, d2.Resists{Fire: 75, Cold: 75, Lightning: 75, Poison: 75}, false},
		{"hell poison 99 is -1", d2.Hell, d2.Resists{Fire: 125, Cold: 125, Lightning: 125, Poison: 99}, false},
	} {
		got := ResistGate(c.d, c.r)
		if got.OK != c.ok || (!got.OK && got.Why == "") {
			t.Errorf("%s: %+v", c.name, got)
		}
	}
	if g := ResistGate(d2.Hell, d2.Resists{Fire: 200, Cold: 200, Lightning: 200, Poison: 200}); g.Post[0] != 75 {
		t.Errorf("final value capped at 75, got %v", g.Post)
	}
}

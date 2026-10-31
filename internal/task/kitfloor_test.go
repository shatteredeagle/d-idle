package task_test

import (
	"testing"

	"d2lod-bot/internal/task"
)

func TestDenFloorNeedsKit(t *testing.T) {
	c := task.NormalAct1(task.Act1Options{})
	cases := []struct {
		name string
		w    task.World
		den  bool
	}{
		{"clvl 3 no kit levels first", task.World{Level: 3}, false},
		{"clvl 5 no kit enters", task.World{Level: 5}, true},
		{"clvl 2 with scroll enters", task.World{Level: 2, HasTP: true}, true},
		{"clvl 2 with 3 potions enters", task.World{Level: 2, Potions: 3}, true},
		{"clvl 2 with 2 potions levels first", task.World{Level: 2, Potions: 2}, false},
	}
	for _, tc := range cases {
		p := c.Plan(tc.w, nil)
		if got := p.Group == task.QDen; got != tc.den {
			t.Errorf("%s: plan %q group %q, want den=%v", tc.name, p.ID, p.Group, tc.den)
		}
	}
}

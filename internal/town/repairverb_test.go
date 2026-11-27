package town_test

import (
	"errors"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

type repairStub struct {
	useStub
	ids    []int
	clicks int
	fail   error
}

func (r *repairStub) Repair(id int) error { r.ids = append(r.ids, id); return r.fail }
func (r *repairStub) Click(input.Button, int, int, ...input.Mod) error {
	r.clicks++
	return nil
}

func TestRepairAllByVerb(t *testing.T) {
	for _, cls := range []d2.Class{d2.Sorceress, d2.Barbarian} {
		for _, tc := range []struct {
			name         string
			on           bool
			caps         []string
			fail         error
			verbs, click int
		}{
			{"verb", true, []string{input.CapRepair}, nil, 1, 0},
			{"knob off", false, []string{input.CapRepair}, nil, 0, 1},
			{"no cap", true, nil, nil, 0, 1},
			{"older build", true, []string{input.CapRepair}, errors.New("d2d: unknown cmd 'repair'"), 1, 1},
		} {
			c, _ := useTown(cls, tc.on, tc.caps...)
			r := &repairStub{fail: tc.fail}
			c.setIn(r)
			if err := c.t.PressRepairAllForTest(); err != nil {
				t.Fatalf("%v %s: %v", cls, tc.name, err)
			}
			if len(r.ids) != tc.verbs || r.clicks != tc.click || (tc.verbs > 0 && r.ids[0] != input.RepairAllItems) {
				t.Fatalf("%v %s: verbs %v clicks %d", cls, tc.name, r.ids, r.clicks)
			}
		}
	}
}

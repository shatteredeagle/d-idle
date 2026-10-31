package task_test

import (
	"strings"
	"testing"

	"d2lod-bot/internal/task"
)

func denWorld(lvl, floor int) task.World {
	return task.World{Act: 1, Difficulty: "normal", Level: lvl, Gold: 300, Potions: 5, HasTP: true, DenMinLevel: floor,
		Quests: map[string]task.QuestState{}}
}

// A caster below the floor levels in Blood Moor instead of entering the Den; at the floor the Den is the first quest.
func TestDenWaitsForCasterFloor(t *testing.T) {
	c := task.NormalAct1(task.Act1Options{})
	for _, lvl := range []int{1, 3, 4} {
		p := c.Plan(denWorld(lvl, 5), nil)
		if p.Group == task.QDen || p.ID != "level.5" || p.Why != "den: waiting for clvl 5 in Blood Moor" {
			t.Errorf("clvl %d caster: plan %q group %q why %q", lvl, p.ID, p.Group, p.Why)
		}
	}
	if p := c.Plan(denWorld(5, 5), nil); p.Group != task.QDen {
		t.Errorf("clvl 5 caster: plan %q, want the Den", p.ID)
	}
}

func TestDenMeleeEntersAtThree(t *testing.T) {
	c := task.NormalAct1(task.Act1Options{})
	if p := c.Plan(denWorld(2, 3), nil); p.ID != "level.3" || !strings.HasPrefix(p.Why, "den: waiting") {
		t.Errorf("clvl 2 melee: %q %q", p.ID, p.Why)
	}
	if p := c.Plan(denWorld(3, 3), nil); p.Group != task.QDen {
		t.Errorf("clvl 3 melee: plan %q", p.ID)
	}
}

// The Den stays the first quest: above the floor nothing else precedes it, and other difficulties ignore the floor.
func TestDenStaysFirstAndNormalOnly(t *testing.T) {
	c := task.NormalAct1(task.Act1Options{})
	if p := c.Plan(denWorld(12, 5), nil); p.Group != task.QDen {
		t.Errorf("clvl 12 with the Den open: %q", p.ID)
	}
	w := denWorld(1, 5)
	w.DenMinLevel = 0
	if p := c.Plan(w, nil); p.Group != task.QDen {
		t.Errorf("no class floor: %q", p.ID)
	}
}

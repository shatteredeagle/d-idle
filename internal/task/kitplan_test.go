package task_test

import (
	"strings"
	"testing"

	"d2lod-bot/internal/task"
)

// town.plan >= 2: a guard waits only on a line the plan can close. A 4-slot belt that can carry three healing potions
// made the old guard (min 4) unsatisfiable for ever; the plan clamps the minimum to the carry capacity.
func TestKitPlanClampedMinOpensTheRoad(t *testing.T) {
	for _, tc := range []struct {
		name   string
		caster bool
	}{{"paladin", false}, {"sorceress", true}} {
		mut := func(w *task.World) { w.Caster, w.Potions, w.ManaPotions = tc.caster, 3, 3 }
		old := routeWorld(22, mut)
		if task.RouteWhy(old) == "" {
			t.Fatalf("%s: control, the constants must block 3 healing potions", tc.name)
		}
		planned := routeWorld(22, mut)
		planned.KitPlan = &task.KitPlan{Healing: 3, Mana: 2, Portals: 2}
		if why := task.RouteWhy(planned); why != "" {
			t.Errorf("%s: the clamped plan minimum 3 must open the road, got %q", tc.name, why)
		}
	}
}

func TestKitPlanUnclosableLineIsNamedNotRetried(t *testing.T) {
	for _, tc := range []struct {
		name   string
		caster bool
	}{{"paladin", false}, {"sorceress", true}} {
		w := routeWorld(22, func(w *task.World) { w.Caster, w.Potions = tc.caster, 1 })
		w.KitPlan = &task.KitPlan{Healing: 4, Mana: 2, Portals: 2, Short: "short of gold: needs 90 more for healing"}
		w.RouteBlocked = task.RouteWhy(w)
		f := task.StepTravel("t", "travel", 7, func(task.World) bool { return false }).Ready(w)
		if f == nil || f.Kind != task.KindDanger || !strings.Contains(f.Why, "needs 90 more for healing") || len(f.Insert) != 0 {
			t.Errorf("%s: fault %+v, want danger naming the line and no chores insert", tc.name, f)
		}
		w.KitPlan.Short = "" // gold in hand: the plan can close it, so the fix is a chores trip
		f = task.StepTravel("t", "travel", 7, func(task.World) bool { return false }).Ready(w)
		if f == nil || f.Kind != task.KindResource || len(f.Insert) != 1 {
			t.Errorf("%s: fault %+v, want a resource fault inserting chores", tc.name, f)
		}
	}
}

func TestStepChoresNeedFollowsThePlan(t *testing.T) {
	s := task.StepChores("c")
	w := routeWorld(22, func(w *task.World) { w.Potions = 3 })
	if need, _ := s.Need(w); !need {
		t.Fatal("old minimum 4 wants a trip at 3")
	}
	w.KitPlan = &task.KitPlan{Healing: 3}
	if need, _ := s.Need(w); need {
		t.Error("plan minimum 3 is met")
	}
	w.Potions, w.KitPlan.Healing, w.KitPlan.Short = 1, 4, "short of gold: needs 60 more for healing"
	if need, _ := s.Need(w); need {
		t.Error("a trip the plan cannot pay must not run just to end 'ran but is still needed'")
	}
}

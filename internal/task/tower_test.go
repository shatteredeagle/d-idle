package task_test

import (
	"strings"
	"testing"

	"d2lod-bot/internal/task"
)

func towerOn(w *task.World) { w.TowerMinLevel, w.TowerNeedsMerc = 20, true }

// the G1 attempt 17 Soaky: clvl 14 caster, Den, Burial and Tools done, no merc, empty belt.
func soaky(mut func(*task.World)) task.World {
	return routeWorld(14, func(w *task.World) {
		towerOn(w)
		w.Caster, w.Potions = true, 0
		w.Quests[task.QTools] = task.QuestState{Stage: task.StageDone}
		if mut != nil {
			mut(w)
		}
	})
}

func TestTowerSkippedNoMercBelowFloor(t *testing.T) {
	w := soaky(nil)
	if why := task.TowerSkipWhy(w); why != "no merc, clvl 14 < 20" {
		t.Fatalf("skip why %q", why)
	}
	if line := task.TowerLine(w); line != "route guard: blocked tower (no merc, clvl 14 < 20)" {
		t.Errorf("line %q", line)
	}
	p := task.NormalAct1(task.Act1Options{}).Plan(w, nil)
	if p.Group == task.QTower || strings.Contains(p.ID, task.QTower) {
		t.Errorf("tower not skipped: %q", p.ID)
	}
}

func TestTowerOpenWithMercOrLevel(t *testing.T) {
	merc := soaky(func(w *task.World) { w.HasMerc = true })
	if why := task.TowerSkipWhy(merc); why != "" {
		t.Errorf("merc alive: %q", why)
	}
	if why := task.TowerKitWhy(merc); why == "" {
		t.Error("empty belt must still fail the kit")
	}
	full := soaky(func(w *task.World) { w.HasMerc, w.Potions = true, 5 })
	if why := task.TowerWhy(full); why != "" {
		t.Errorf("merc and kit: %q", why)
	}
	if p := task.NormalAct1(task.Act1Options{}).Plan(full, nil); p.Group != task.QTower {
		t.Errorf("merc and kit plan %q", p.ID)
	}
	if why := task.TowerSkipWhy(soaky(func(w *task.World) { w.Level = 20 })); why != "" {
		t.Errorf("clvl 20: %q", why)
	}
	// needsMerc off: the level alone decides, a merc does not open it
	if why := task.TowerSkipWhy(soaky(func(w *task.World) { w.TowerNeedsMerc, w.HasMerc = false, true })); why != "clvl 14 < 20" {
		t.Errorf("needsMerc off: %q", why)
	}
	if why := task.TowerWhy(soaky(func(w *task.World) { w.TowerMinLevel = 0 })); why != "" {
		t.Errorf("guard off: %q", why)
	}
	if why := task.TowerWhy(soaky(func(w *task.World) { w.Difficulty = "nightmare" })); why != "" {
		t.Errorf("nightmare: %q", why)
	}
}

func TestTowerNeverBlocksAndariel(t *testing.T) {
	// optional: with the tower skipped the chain continues to Andariel once the road is open (clvl 20)
	w := routeWorld(20, func(w *task.World) {
		w.Quests[task.QTools] = task.QuestState{Stage: task.StageDone}
		towerOn(w)
		w.TowerMinLevel = 25 // tower closed for clvl 20, the road is not
	})
	if p := task.NormalAct1(task.Act1Options{}).Plan(w, nil); p.Group != task.QAndariel {
		t.Errorf("plan %q group %q", p.ID, p.Group)
	}
}

func TestTowerGuardedAreas(t *testing.T) {
	for a := task.Area(0); a < 60; a++ {
		want := a >= 20 && a <= 25
		if task.TowerGuarded(a) != want {
			t.Errorf("area %d guarded = %v", a, !want)
		}
	}
}

func TestFieldReadyKeepsTowerAreasClosed(t *testing.T) {
	w := soaky(func(w *task.World) { w.TowerBlocked = task.TowerWhy(*w) })
	if f := task.StepTravel("t", "t", task.TowerCellar5, func(task.World) bool { return false }).Ready(w); f == nil || f.Kind != task.KindDanger {
		t.Errorf("tower cellar 5 with a closed tower: %v", f)
	}
}

package task_test

import (
	"strings"
	"testing"

	"d2lod-bot/internal/task"
)

func guardOn(w *task.World) {
	w.RouteMinLevel, w.ImbueMinLevel, w.ImbueNeedsMerc, w.ImbueUseful = 20, 20, true, true
}

// a Normal Act 1 character with Den and Blood Raven done, Tools and Tower open (the Soaky situation).
func routeWorld(lvl int, mut func(*task.World)) task.World {
	w := task.World{Act: 1, Difficulty: "normal", Level: lvl, Gold: 500, Potions: 5, HasTP: true, Antidotes: 2, ManaPotions: 3,
		Quests: map[string]task.QuestState{task.QDen: {Stage: task.StageDone}, task.QBurial: {Stage: task.StageDone}}}
	guardOn(&w)
	if mut != nil {
		mut(&w)
	}
	return w
}

func TestImbueSkippedBelow20OrNoMerc(t *testing.T) {
	c := task.NormalAct1(task.Act1Options{})
	for name, mut := range map[string]func(*task.World){
		"clvl 12 melee":     func(w *task.World) {},
		"clvl 25 no merc":   func(w *task.World) { w.Level = 25 },
		"sorceress clvl 25": func(w *task.World) { w.Level = 25; w.ImbueUseful = false; w.HasMerc = true },
	} {
		w := routeWorld(12, mut)
		if p := c.Plan(w, nil); p.Group == task.QTools {
			t.Errorf("%s: plan %q, Tools must be skipped", name, p.ID)
		}
		if task.ImbueSkipWhy(w) == "" {
			t.Errorf("%s: no skip reason", name)
		}
	}
	w := routeWorld(20, func(w *task.World) { w.HasMerc = true })
	if task.ImbueSkipWhy(w) != "" {
		t.Errorf("clvl 20 with merc must do the imbue: %q", task.ImbueSkipWhy(w))
	}
	if p := c.Plan(w, nil); p.Group != task.QTools {
		t.Errorf("clvl 20 merc plan %q", p.ID)
	}
	// knob off: needs no merc
	w = routeWorld(20, func(w *task.World) { w.ImbueNeedsMerc = false })
	if task.ImbueSkipWhy(w) != "" {
		t.Error("imbueNeedsMerc off, clvl 20: not skipped")
	}
	// the chain is never blocked: with Tower done a skipped Tools lets the plan go on
	w = routeWorld(12, func(w *task.World) { w.Quests[task.QTower] = task.QuestState{Stage: task.StageDone} })
	if p := c.Plan(w, nil); p.Group == task.QTools || p.Stuck {
		t.Errorf("plan %+v", p)
	}
}

func TestRouteGuardMonastery(t *testing.T) {
	g, _ := task.GateFor(task.QTools)
	w := routeWorld(12, func(w *task.World) { w.ImbueMinLevel = 0 }) // imbue rule off to look at the guard alone
	if why := g.NotReady(w); !strings.Contains(why, "route guard: blocked monastery (no merc, clvl 12 < 20)") {
		t.Errorf("no merc clvl 12: %q", why)
	}
	if line := task.RouteLine(w); line != "route guard: blocked monastery (no merc, clvl 12 < 20)" {
		t.Errorf("line %q", line)
	}
	// a living merc (unit table or hook) opens it with the kit; unknown counts as no merc
	for name, mut := range map[string]func(*task.World){
		"hasmerc": func(w *task.World) { w.HasMerc = true },
		"hook":    func(w *task.World) { w.MercKnown, w.MercAlive = true, true },
		"clvl 20": func(w *task.World) { w.Level = 20 },
	} {
		w := routeWorld(12, mut)
		w.ImbueMinLevel = 0
		if why := g.NotReady(w); why != "" {
			t.Errorf("%s: %q", name, why)
		}
	}
	w2 := routeWorld(15, func(w *task.World) { w.MercKnown, w.MercAlive = false, true }) // not known: no merc
	if task.RouteLevelWhy(w2) == "" {
		t.Error("unknown merc must count as no merc")
	}
	// kit
	for name, mut := range map[string]func(*task.World){
		"3 healing":   func(w *task.World) { w.HasMerc = true; w.Potions = 3 },
		"no portal":   func(w *task.World) { w.HasMerc = true; w.HasTP = false },
		"1 charge":    func(w *task.World) { w.HasMerc = true; w.TPKnown, w.TPCharges = true, 1 },
		"caster mana": func(w *task.World) { w.HasMerc = true; w.Caster, w.ManaPotions = true, 1 },
	} {
		if why := g.NotReady(routeWorld(12, mut)); !strings.HasPrefix(why, "route guard: kit") {
			t.Errorf("%s: %q", name, why)
		}
	}
	// guard off (zero knob) and Nightmare change nothing
	if why := g.NotReady(routeWorld(12, func(w *task.World) { w.RouteMinLevel = 0; w.ImbueMinLevel = 0 })); why != "" {
		t.Errorf("guard off: %q", why)
	}
}

func TestRouteGuardBlocksAndarielAndLevelsInstead(t *testing.T) {
	c := task.NormalAct1(task.Act1Options{})
	done := func(w *task.World) {
		for _, q := range []string{task.QTools, task.QTower} {
			w.Quests[q] = task.QuestState{Stage: task.StageDone}
		}
	}
	// clvl 16 is above the Andariel floor 15, but with no merc the road is closed: a level step, not the quest
	w := routeWorld(16, func(w *task.World) { done(w); w.AndarielMinLevel = 15 }) // the old numbers: 20 / any merc / 15
	p := c.Plan(w, nil)
	if p.Group == task.QAndariel || p.Group == task.QAct1Done || !strings.HasPrefix(p.ID, "level.") {
		t.Fatalf("clvl 16 no merc: plan %q group %q", p.ID, p.Group)
	}
	if p.ID != "level.20" {
		t.Errorf("level target %q, want level.20", p.ID)
	}
	if p := c.Plan(routeWorld(16, func(w *task.World) { done(w); w.HasMerc = true; w.AndarielMinLevel = 15 }), nil); p.Group != task.QAndariel {
		t.Errorf("merc alive: %q", p.ID)
	}
	if p := c.Plan(routeWorld(20, done), nil); p.Group != task.QAndariel {
		t.Errorf("clvl 20: %q", p.ID)
	}
	g, _ := task.GateFor(task.QAndariel)
	if why := g.NotReady(routeWorld(16, func(w *task.World) { done(w); w.AndarielMinLevel = 15 })); !strings.Contains(why, "route guard") {
		t.Errorf("andariel gate: %q", why)
	}
}

func TestRouteGuardedAreas(t *testing.T) {
	for a := task.Area(0); a < 60; a++ {
		want := a == 7 || (a >= 26 && a <= 37)
		if task.RouteGuarded(a) != want {
			t.Errorf("area %d guarded=%v", a, task.RouteGuarded(a))
		}
	}
}

func TestFieldReadyHonoursTheClosedRoad(t *testing.T) {
	w := routeWorld(12, nil)
	w.RouteBlocked = task.RouteWhy(w)
	ready := func(a task.Area) *task.Fault {
		return task.StepTravel("t", "travel", a, func(task.World) bool { return false }).Ready(w)
	}
	if f := ready(7); f == nil || f.Kind != task.KindDanger || !strings.Contains(f.Why, "route guard") {
		t.Fatalf("fault %+v", f)
	}
	if ready(3) != nil {
		t.Error("Cold Plains stays open")
	}
	w.RouteBlocked = ""
	if ready(7) != nil {
		t.Error("open road")
	}
}

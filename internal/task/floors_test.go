package task_test

import (
	"strings"
	"testing"

	"d2lod-bot/internal/env"
	"d2lod-bot/internal/task"
)

// floorWorld is a Normal character with the ADR 0011 knob defaults, the full kit and a merc state.
func floorWorld(lvl int, merc string, mut func(*task.World)) task.World {
	rt, bt := env.DefaultRouteTuning(), env.DefaultBossTuning()
	w := task.World{Act: 1, Difficulty: "normal", Level: lvl, Gold: 500, Potions: 5, HasTP: true, ManaPotions: 3, Antidotes: 2,
		RouteMinLevel: rt.MonasteryMinLevel, RouteMinLevelMerc: rt.MonasteryMinLevelMerc,
		TowerMinLevel: rt.TowerMinLevel, TowerMinLevelMerc: rt.TowerMinLevelMerc, TowerNeedsMerc: rt.TowerNeedsMerc != 0,
		AndarielMinLevel: bt.AndarielMinLevel, AndarielMinLevelMerc: bt.AndarielMinLevelMerc,
		Quests: map[string]task.QuestState{}}
	switch merc {
	case "alive":
		w.HasMerc, w.MercKnown, w.MercAlive = true, true, true
	case "dead":
		w.MercKnown = true
	}
	if mut != nil {
		mut(&w)
	}
	return w
}

// TestGuardFloorTable: gate x merc state x clvl at the boundaries 13/14, 15/16, 14/15 and 16/17 (ADR 0011).
func TestGuardFloorTable(t *testing.T) {
	andyGate, _ := task.GateFor(task.QAndariel)
	cases := []struct {
		merc          string
		lvl           int
		road, tw, ady bool // open?
	}{
		{"none", 13, false, false, false}, {"none", 14, false, false, false}, {"none", 15, false, false, false},
		{"none", 16, true, true, false}, {"none", 17, true, true, true},
		{"dead", 15, false, false, false}, {"dead", 16, true, true, false}, {"dead", 17, true, true, true},
		{"alive", 13, false, false, false}, {"alive", 14, true, true, false}, {"alive", 15, true, true, true}, {"alive", 16, true, true, true},
	}
	for _, c := range cases {
		w := floorWorld(c.lvl, c.merc, nil)
		if got := task.RouteWhy(w) == ""; got != c.road {
			t.Errorf("road merc=%s clvl %d: open=%v (%q)", c.merc, c.lvl, got, task.RouteWhy(w))
		}
		if got := task.TowerWhy(w) == ""; got != c.tw {
			t.Errorf("tower merc=%s clvl %d: open=%v (%q)", c.merc, c.lvl, got, task.TowerWhy(w))
		}
		w.AndarielHold = false
		if got := andyGate.NotReady(w) == ""; got != c.ady {
			t.Errorf("andariel merc=%s clvl %d: ready=%v (%q)", c.merc, c.lvl, got, andyGate.NotReady(w))
		}
	}
}

func TestGuardKitStaysOnEveryGuard(t *testing.T) {
	thin := func(w *task.World) { w.Potions = 2 }
	if why := task.RouteWhy(floorWorld(15, "alive", thin)); !strings.Contains(why, "healing potions") {
		t.Errorf("road kit: %q", why)
	}
	if why := task.TowerWhy(floorWorld(15, "alive", thin)); !strings.Contains(why, "healing potions") {
		t.Errorf("tower kit: %q", why)
	}
	g, _ := task.GateFor(task.QAndariel)
	if why := g.NotReady(floorWorld(17, "none", func(w *task.World) { thin(w); w.AndarielHold = false })); why == "" {
		t.Error("andariel kit")
	}
}

func TestAndarielNeedsTwoAntidotes(t *testing.T) {
	g, _ := task.GateFor(task.QAndariel)
	for _, n := range []int{0, 1} {
		if why := g.NotReady(floorWorld(17, "none", func(w *task.World) { w.Antidotes = n; w.AndarielHold = false })); !strings.Contains(why, "antidotes") {
			t.Errorf("%d antidotes: %q", n, why)
		}
	}
	if why := g.NotReady(floorWorld(17, "none", func(w *task.World) { w.AndarielHold = false })); why != "" {
		t.Errorf("2 antidotes: %q", why)
	}
}

func TestAndarielUnattendedHold(t *testing.T) {
	c := task.NormalAct1(task.Act1Options{})
	done := func(w *task.World) {
		for _, q := range []string{task.QDen, task.QBurial, task.QTools, task.QTower} {
			w.Quests[q] = task.QuestState{Stage: task.StageDone}
		}
		w.AndarielHold = true
	}
	// everything else ready, clvl above both floors: the campaign still refuses and levels
	if p := c.Plan(floorWorld(20, "alive", done), nil); p.Group == task.QAndariel || !strings.HasPrefix(p.ID, "level.") {
		t.Errorf("hold on: plan %q", p.ID)
	}
	g, _ := task.GateFor(task.QAndariel)
	w := floorWorld(20, "alive", done)
	if why := g.NotReady(w); !strings.Contains(why, "hold") {
		t.Errorf("gate: %q", why)
	}
	if f := g.NotReady(w); f == "" {
		t.Fatal("gate must refuse")
	}
	if got := task.AndarielLine(w); !strings.Contains(got, "unattended hold") {
		t.Errorf("line %q", got)
	}
	// knob 0 (the fight has won) lifts it; a dead Andariel never holds Act 2
	if p := c.Plan(floorWorld(20, "alive", func(w *task.World) { done(w); w.AndarielHold = false }), nil); p.Group != task.QAndariel {
		t.Errorf("hold off: %q", p.ID)
	}
	killed := floorWorld(20, "alive", func(w *task.World) {
		done(w)
		w.Killed = map[string]bool{task.BossAndy: true}
		w.Quests[task.QAndariel] = task.QuestState{Stage: task.StageDone}
	})
	if p := c.Plan(killed, nil); p.Group == task.QAndariel {
		t.Errorf("andariel done: %q", p.ID)
	}
	// not Normal: untouched
	if why := g.NotReady(floorWorld(70, "none", func(w *task.World) { done(w); w.Difficulty = "hell" })); strings.Contains(why, "hold") {
		t.Errorf("hell: %q", why)
	}
}

func TestEffectiveFloorLine(t *testing.T) {
	for _, c := range []struct {
		merc, want string
		w          task.World
	}{
		{"none", "route guard: monastery floor=16 (merc=none)", floorWorld(10, "none", nil)},
		{"alive", "route guard: monastery floor=14 (merc=alive)", floorWorld(10, "alive", nil)},
		{"dead", "route guard: monastery floor=16 (merc=dead)", floorWorld(10, "dead", nil)},
	} {
		if got := task.FloorLine("monastery", task.MonasteryFloor(c.w), c.w); got != c.want {
			t.Errorf("%s: %q want %q", c.merc, got, c.want)
		}
	}
	if got := task.FloorLine("andariel", task.AndarielFloor(floorWorld(10, "alive", nil)), floorWorld(10, "alive", nil)); got != "route guard: andariel floor=15 (merc=alive)" {
		t.Errorf("andariel %q", got)
	}
	if got := task.FloorLine("tower", task.TowerFloor(floorWorld(10, "none", nil)), floorWorld(10, "none", nil)); got != "route guard: tower floor=16 (merc=none)" {
		t.Errorf("tower %q", got)
	}
}

// TestOldNumbersByKnobs: 20 / a merc opens at any level / Andariel 15 reproduce the old decisions (the dead-merc +2 is
// not reproducible: it is replaced, 17 without a merc equals the old 15 plus 2).
func TestOldNumbersByKnobs(t *testing.T) {
	old := func(w *task.World) {
		w.RouteMinLevel, w.RouteMinLevelMerc = 20, 0
		w.TowerMinLevel, w.TowerMinLevelMerc, w.TowerNeedsMerc = 20, 0, true
		w.AndarielMinLevel, w.AndarielMinLevelMerc, w.AndarielHold = 15, 0, false
	}
	if why := task.RouteWhy(floorWorld(12, "none", old)); !strings.HasPrefix(why, "no merc, clvl 12 < 20") {
		t.Errorf("no merc: %q", why)
	}
	if why := task.RouteWhy(floorWorld(5, "alive", old)); why != "" {
		t.Errorf("merc any level: %q", why)
	}
	if why := task.TowerWhy(floorWorld(5, "alive", old)); why != "" {
		t.Errorf("tower merc any level: %q", why)
	}
	if why := task.TowerWhy(floorWorld(19, "none", old)); why == "" {
		t.Error("tower no merc 19")
	}
	g, _ := task.GateFor(task.QAndariel)
	if why := g.NotReady(floorWorld(20, "none", old)); why != "" {
		t.Errorf("andariel clvl 20: %q", why)
	}
	if why := g.NotReady(floorWorld(15, "alive", old)); why != "" {
		t.Errorf("andariel merc clvl 15: %q", why)
	}
}

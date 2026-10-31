package task_test

import (
	"strings"
	"testing"

	"d2lod-bot/internal/env"
	"d2lod-bot/internal/task"
)

// andyWorld is a Normal Act 1 character with the first four quests done and a full Andariel kit.
func andyWorld(lvl int, mut func(*task.World)) task.World {
	w := task.World{Act: 1, Difficulty: "normal", Level: lvl, Gold: 500, Potions: 4, HasTP: true, Antidotes: 2,
		Quests: map[string]task.QuestState{}}
	for _, q := range []string{task.QDen, task.QBurial, task.QTools, task.QTower} {
		w.Quests[q] = task.QuestState{Stage: task.StageDone}
	}
	if mut != nil {
		mut(&w)
	}
	return w
}

func TestAndarielKnobFloor(t *testing.T) {
	c := task.NormalAct1(task.Act1Options{})
	d := env.DefaultAndarielMinLevel
	if p := c.Plan(andyWorld(d-1, nil), nil); p.Group == task.QAndariel || p.ID == "" {
		t.Errorf("clvl %d plan %q, want a level step", d-1, p.ID)
	}
	if p := c.Plan(andyWorld(d, nil), nil); p.Group != task.QAndariel {
		t.Errorf("clvl %d plan %q, want andariel", d, p.ID)
	}
	// knob 22 reproduces the old hold
	old := func(w *task.World) { w.AndarielMinLevel = 22 }
	if p := c.Plan(andyWorld(21, old), nil); p.Group == task.QAndariel {
		t.Errorf("knob 22 at clvl 21: %q", p.ID)
	}
	if p := c.Plan(andyWorld(22, old), nil); p.Group != task.QAndariel {
		t.Errorf("knob 22 at clvl 22: %q", p.ID)
	}
	// the knob is Normal only: Nightmare keeps its static floor
	nm := task.NightmareAll(false)
	for _, e := range nm.Entries {
		if e.Quest == task.QAndariel && e.MinLevel < 40 {
			t.Errorf("nightmare andariel floor %d", e.MinLevel)
		}
	}
}

func TestAndarielKitGate(t *testing.T) {
	g, _ := task.GateFor(task.QAndariel)
	full := andyWorld(env.DefaultAndarielMinLevel, nil)
	if why := g.NotReady(full); why != "" {
		t.Fatalf("full kit refused: %s", why)
	}
	for name, mut := range map[string]func(*task.World){
		"3 potions":        func(w *task.World) { w.Potions = 3 },
		"no town portal":   func(w *task.World) { w.HasTP = false },
		"no antidotes":     func(w *task.World) { w.Antidotes = 0 },
		"caster no mana":   func(w *task.World) { w.Caster = true },
		"clvl below floor": func(w *task.World) { w.Level-- },
	} {
		if why := g.NotReady(andyWorld(env.DefaultAndarielMinLevel, mut)); why == "" {
			t.Errorf("%s: gate passed", name)
		}
	}
	// antidotes are only demanded while affordable
	if why := g.NotReady(andyWorld(env.DefaultAndarielMinLevel, func(w *task.World) { w.Antidotes, w.Gold = 0, 20 })); why != "" {
		t.Errorf("broke character refused for antidotes: %s", why)
	}
	// a short kit is a chores insert (resource fault) at the floor, a short level is a danger fault
	ready := task.Handler(task.QAndariel)[1].Step.Ready // travel step
	if f := ready(andyWorld(env.DefaultAndarielMinLevel, func(w *task.World) { w.Potions = 0 })); f == nil || f.Kind != task.KindResource || len(f.Insert) == 0 {
		t.Errorf("short kit fault: %+v", f)
	}
	if f := ready(andyWorld(env.DefaultAndarielMinLevel-1, nil)); f == nil || f.Kind != task.KindDanger {
		t.Errorf("short level fault: %+v", f)
	}
}

func TestAndarielMercHook(t *testing.T) {
	d := env.DefaultAndarielMinLevel
	if task.MercAlive != nil {
		t.Fatal("hook must default to unknown")
	}
	c := task.NormalAct1(task.Act1Options{})
	// ADR 0011: a dead merc counts as none, whatever the gold: the no-merc floor, no +2
	for _, gold := range []int{100, 5000} {
		dead := func(w *task.World) { w.MercKnown, w.MercAlive, w.Gold = true, false, gold }
		if p := c.Plan(andyWorld(d-1, dead), nil); p.Group == task.QAndariel {
			t.Errorf("dead merc gold %d clvl %d: %q", gold, d-1, p.ID)
		}
		if p := c.Plan(andyWorld(d, dead), nil); p.Group != task.QAndariel {
			t.Errorf("dead merc gold %d: floor is the no-merc %d, got %q", gold, d, p.ID)
		}
	}
	unknown := func(w *task.World) { w.MercKnown, w.MercAlive, w.Gold = false, false, 100 }
	if p := c.Plan(andyWorld(d, unknown), nil); p.Group != task.QAndariel {
		t.Errorf("unknown merc is none, at the floor it passes: %q", p.ID)
	}
	if p := c.Plan(andyWorld(d-1, unknown), nil); p.Group == task.QAndariel {
		t.Errorf("unknown merc below the no-merc floor: %q", p.ID)
	}
}

func TestAndarielLines(t *testing.T) {
	d := env.DefaultAndarielMinLevel
	if got := task.AndarielLine(andyWorld(d-3, nil)); got != "boss: held for andariel (clvl 14 < floor 17)" {
		t.Errorf("held line %q", got)
	}
	if got := task.AndarielLine(andyWorld(d, nil)); got != "boss: andariel gate pass clvl=17 kit=ok" {
		t.Errorf("pass line %q", got)
	}
	got := task.AndarielLine(andyWorld(d+1, func(w *task.World) { w.HasTP = false }))
	if !strings.HasPrefix(got, "boss: andariel gate pass clvl=18 kit=short(") || !strings.Contains(got, "no town portal") {
		t.Errorf("short line %q", got)
	}
}

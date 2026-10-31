package task_test

import (
	"testing"

	"d2lod-bot/internal/task"
)

// The Soaky log 11:05 to 11:12 (soak-analysis R1): clvl 6, belt 0 refuses; kit and clvl 8 allows.
func TestBossGateTable(t *testing.T) {
	kit := func(mut func(*task.World)) task.World {
		w := task.World{Level: 8, Potions: 4, HasTP: true, ManaPotions: 2, Caster: true, Gold: 300}
		if mut != nil {
			mut(&w)
		}
		return w
	}
	cases := []struct {
		name  string
		quest string
		w     task.World
		ready bool
	}{
		{"soaky 11:06 clvl 6 belt 0", task.QBurial, task.World{Level: 6, HasTP: true, Caster: true}, false},
		{"clvl 7 full kit", task.QBurial, kit(func(w *task.World) { w.Level = 7 }), false},
		{"clvl 8 full kit", task.QBurial, kit(nil), true},
		{"clvl 8 belt 0", task.QBurial, kit(func(w *task.World) { w.Potions = 0 }), false},
		{"clvl 8 no TP", task.QBurial, kit(func(w *task.World) { w.HasTP = false }), false},
		{"caster without mana potions", task.QBurial, kit(func(w *task.World) { w.ManaPotions = 0 }), false},
		{"melee needs no mana potions", task.QBurial, kit(func(w *task.World) { w.ManaPotions, w.Caster = 0, false }), true},
		{"sorc clvl 8 waits for 9", task.QBurial, kit(func(w *task.World) { w.BRHigh = true }), false},
		{"sorc clvl 9 kit", task.QBurial, kit(func(w *task.World) { w.BRHigh, w.Level = true, 9 }), true},
		{"gold 249", task.QBurial, kit(func(w *task.World) { w.Gold = 249 }), false},
		{"gold 250", task.QBurial, kit(func(w *task.World) { w.Gold = 250 }), true},
		{"one portal scroll", task.QBurial, kit(func(w *task.World) { w.TPKnown, w.TPCharges = true, 1 }), false},
		{"tome 2 charges", task.QBurial, kit(func(w *task.World) { w.TPKnown, w.TPCharges = true, 2 }), true},
		{"wait level after a failure", task.QBurial, kit(func(w *task.World) { w.BRWaitLevel = 9 }), false},
		{"knobs raise the floor", task.QBurial, kit(func(w *task.World) { w.BR = task.BRKnobs{MeleeLevel: 10} }), false},
		{"smith clvl 8 kit", task.QTools, kit(nil), true},
		{"countess clvl 13", task.QTower, kit(func(w *task.World) { w.Level = 13 }), false},
		{"andariel 3 potions", task.QAndariel, kit(func(w *task.World) { w.Level = 22; w.Potions = 3; w.ManaPotions = 3 }), false},
		{"andariel full", task.QAndariel, kit(func(w *task.World) { w.Level = 22; w.ManaPotions = 3; w.Gold = 0 }), true},
	}
	for _, c := range cases {
		g, ok := task.GateFor(c.quest)
		if !ok {
			t.Fatalf("no gate for %s", c.quest)
		}
		if why := g.NotReady(c.w); (why == "") != c.ready {
			t.Errorf("%s: NotReady = %q, want ready=%v", c.name, why, c.ready)
		}
	}
}

// The planner never hands Blood Raven to the executor below the floor, and a kit-less character at the floor gets a
// chores insert from the step's Ready, never the fight.
func TestBurialGateInPlanAndReady(t *testing.T) {
	c := task.NormalAct1(task.Act1Options{SkipTools: true, SkipTower: true})
	base := func(lvl, pots int, tp bool) task.World {
		return task.World{Act: 1, Level: lvl, Potions: pots, HasTP: tp, Gold: 300, Quests: map[string]task.QuestState{
			task.QDen: {Stage: task.StageDone}}}
	}
	if p := c.Plan(base(6, 0, true), nil); p.Group == task.QBurial || p.ID != "level.8" {
		t.Errorf("clvl 6 belt 0 plan %q group %q, want level.8", p.ID, p.Group)
	}
	if p := c.Plan(base(7, 9, true), nil); p.Group == task.QBurial {
		t.Errorf("clvl 7 plan %q", p.ID)
	}
	p := c.Plan(base(8, 0, true), nil)
	if p.Group != task.QBurial {
		t.Fatalf("clvl 8 plan %q", p.ID)
	}
	var travel task.Step
	for _, n := range p.Nodes {
		if n.Step.ID == "quest.sisters_burial/to-burial" {
			travel = n.Step
		}
	}
	if f := travel.Ready(base(8, 0, true)); f == nil || f.Kind != task.KindResource || len(f.Insert) != 1 {
		t.Errorf("belt 0 at the floor: %v", f)
	}
	if f := travel.Ready(base(8, 4, true)); f != nil {
		t.Errorf("clvl 8 with kit faulted: %v", f)
	}
}

// COMBATLOOP kit gate: the Barracks walk (Monastery Gate border deaths, attempt 11: 2 potions, no scroll) is not started
// with a short kit; the step's Ready inserts a chores trip, it is a planning block and no fault of the quest.
func TestBarracksKitGate(t *testing.T) {
	var travel task.Step
	for _, n := range task.Handler(task.QTools) {
		if n.Step.ID == "quest."+task.QTools+"/to-barracks" {
			travel = n.Step
		}
	}
	if travel.ID == "" {
		t.Fatal("no barracks step")
	}
	w := task.World{Act: 1, Level: 11, Potions: 2, HasTP: false, ManaPotions: 2, Caster: true}
	if f := travel.Ready(w); f == nil || f.Kind != task.KindResource || len(f.Insert) != 1 {
		t.Fatalf("2 potions, no scroll: %v", f)
	}
	w.Potions, w.HasTP = 4, true
	if f := travel.Ready(w); f != nil {
		t.Fatalf("full kit faulted: %v", f)
	}
	w.ManaPotions = 0
	if f := travel.Ready(w); f == nil {
		t.Fatal("a caster without mana potions went")
	}
}

// Blood Raven plan rules: a Sorceress levels to 9 before the quest, a short purse is a danger fault (the planner levels
// and farms, a chores trip would buy nothing) while a short kit is a chores insert.
func TestBurialClassFloorAndGoldFault(t *testing.T) {
	c := task.NormalAct1(task.Act1Options{SkipTools: true, SkipTower: true})
	w := task.World{Act: 1, Level: 8, Potions: 5, HasTP: true, Gold: 400, BRHigh: true, Caster: true, ManaPotions: 3,
		Quests: map[string]task.QuestState{task.QDen: {Stage: task.StageDone}}}
	if p := c.Plan(w, nil); p.ID != "level.9" {
		t.Errorf("sorceress clvl 8 plan %q, want level.9", p.ID)
	}
	w.BRHigh = false
	if p := c.Plan(w, nil); p.Group != task.QBurial {
		t.Errorf("necromancer clvl 8 plan %q, want the quest", p.ID)
	}
	var travel task.Step
	for _, n := range c.Plan(w, nil).Nodes {
		if n.Step.ID == "quest.sisters_burial/to-burial" {
			travel = n.Step
		}
	}
	w.Gold = 100
	if f := travel.Ready(w); f == nil || f.Kind != task.KindDanger || len(f.Insert) != 0 {
		t.Errorf("gold 100: %+v", f)
	}
	w.Gold, w.ManaPotions = 400, 0
	if f := travel.Ready(w); f == nil || f.Kind != task.KindResource || len(f.Insert) != 1 {
		t.Errorf("no mana potions: %+v", f)
	}
}

// BRFIX: Blood Raven skipped in Act 1 (gold or kit short) must not be planned again once the character is in Act 2.
func TestBloodRavenNotRetriedFromAct2(t *testing.T) {
	c := task.NormalAct1(task.Act1Options{SkipTools: true, SkipTower: true})
	w := task.World{Act: 2, Level: 16, Gold: 0, Potions: 4, HasTP: true, Quests: map[string]task.QuestState{
		task.QDen: {Stage: task.StageDone}, task.QAndariel: {Stage: task.StageDone}, task.QAct1Done: {Stage: task.StageDone}}}
	if p := c.Plan(w, nil); p.Group == task.QBurial || p.Stuck {
		t.Errorf("act 2 plan %q group %q stuck=%v: Blood Raven must be skipped, not retried", p.ID, p.Group, p.Stuck)
	}
	w.Act, w.Quests[task.QAct1Done] = 1, task.QuestState{}
	w.Quests[task.QAndariel] = task.QuestState{}
	w.Gold = 300
	if p := c.Plan(w, nil); p.Group != task.QBurial {
		t.Errorf("act 1 with gold and kit: plan %q, want Blood Raven", p.ID)
	}
}

package task_test

import (
	"encoding/json"
	"testing"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/plandata"
	"d2lod-bot/internal/task"
)

var campDiffs = []struct {
	name string
	diff d2.Difficulty
	tmpl string
}{{"normal", d2.Normal, "normal_all"}, {"nightmare", d2.Nightmare, "nightmare_all"}, {"hell", d2.Hell, "hell_all"}}

// Per campaign: every entry has an act (1-5) in non-decreasing order, the act's hop entry carries the act's highest
// floor, and the floor is never below the entry gate.
func TestCampaignFloorsMonotoneAndGated(t *testing.T) {
	for _, d := range campDiffs {
		c := task.Template(d.tmpl, false)
		act, top := 1, map[int]int{}
		for _, e := range c.Entries {
			if e.Act < act || e.Act > 5 {
				t.Errorf("%s %s: act %d after act %d", d.tmpl, e.Quest, e.Act, act)
			}
			act = e.Act
			top[act] = max(top[act], e.MinLevel)
			if g := combat.EntryGate(d.diff, e.Act, 0, 0, false); e.MinLevel < g.Floor {
				t.Errorf("%s %s floor %d below the act %d entry gate %d", d.tmpl, e.Quest, e.MinLevel, e.Act, g.Floor)
			}
		}
		// Not asserted: Hell act 2's top (72) is below act 1's (75, the dungeon floor, U); harmless, the character
		// is already above it when it gets there.
		// the hop entry of acts 1-4 sits at the act's top floor, so the character is as strong as the act demands
		last := map[int]task.Entry{}
		for _, e := range c.Entries {
			last[e.Act] = e
		}
		for a := 1; a <= 4; a++ {
			if last[a].MinLevel != top[a] {
				t.Errorf("%s: act %d hop %s floor %d, act top %d", d.tmpl, a, last[a].Quest, last[a].MinLevel, top[a])
			}
		}
	}
}

// The progression file must answer every level step: no clvl between the first entry floor and the campaign's top
// floor may fall in a hole where no farming step covers it (enter <= clvl <= leave).
func TestCampaignFloorsHaveNoProgressionGap(t *testing.T) {
	for _, d := range campDiffs[:3] {
		raw, err := plandata.FS.ReadFile("progression-" + d.name + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var v map[string]json.RawMessage
		if err := json.Unmarshal(raw, &v); err != nil {
			t.Fatal(err)
		}
		var steps []struct {
			Name string `json:"name"`
			E    int    `json:"enter_clvl"`
			L    int    `json:"leave_clvl"`
		}
		if err := json.Unmarshal(v[d.name], &steps); err != nil {
			t.Fatal(err)
		}
		c := task.Template(d.tmpl, false)
		first, top := c.Entries[0].MinLevel, 0
		for _, e := range c.Entries {
			top = max(top, e.MinLevel)
		}
		for l := first; l <= top; l++ {
			covered := false
			for _, s := range steps {
				if s.E <= l && l <= s.L {
					covered = true
				}
			}
			if !covered {
				t.Errorf("%s: clvl %d is covered by no progression step", d.name, l)
			}
		}
	}
}

// The first entry floor is what ReadyForNext asks for: it rises with each difficulty and stays within reach of the
// previous difficulty's top floor (the gap is the farming stretch the progression file covers).
func TestDifficultyEntryFloorsChain(t *testing.T) {
	var prevTop, prevFirst int
	for i, d := range campDiffs {
		c := task.Template(d.tmpl, false)
		top := 0
		for _, e := range c.Entries {
			top = max(top, e.MinLevel)
		}
		if i > 0 && c.Entries[0].MinLevel <= prevFirst {
			t.Errorf("%s first floor %d not above the previous difficulty's %d", d.tmpl, c.Entries[0].MinLevel, prevFirst)
		}
		if i > 0 && c.Entries[0].MinLevel > prevTop+30 {
			t.Errorf("%s first floor %d far above the previous top %d: the old difficulty needs a farming stretch", d.tmpl, c.Entries[0].MinLevel, prevTop)
		}
		prevFirst = c.Entries[0].MinLevel
		prevTop = top
	}
}

// Regression: the Act 3 hop group re-fights Mephisto, so in Nightmare it must not start below Mephisto's floor.
func TestNightmareHopDoesNotBypassTheBossFloor(t *testing.T) {
	c := task.Template("nightmare_all", false)
	w := nmWorld("nightmare", 52)
	w.Act = 3
	done := []string{task.QDen, task.QBurial, task.QTools, task.QTower, task.QAndariel, task.QAct1Done, task.QMercAct2, task.QRadament,
		task.QCube, task.QViper, task.QShaft, task.QStaff, task.QDuriel, task.QAct2Done, task.QMercAct3, task.QEye, task.QBrain,
		task.QHeart, task.QCouncil, task.QKhalim}
	for _, q := range done {
		w.Quests[q] = task.QuestState{Stage: task.StageDone, Ev: task.EvMemory}
	}
	if p := c.Plan(w, nil); p.Group != "" {
		t.Fatalf("clvl 52 in Nightmare Act 3 should level toward Mephisto's 55, got %+v", p)
	}
	w.Level = 55
	if p := c.Plan(w, nil); p.Group != task.QGuardian {
		t.Fatalf("clvl 55: %+v", p)
	}
}

// Regression: a character in Act 1 never gets an Act 2+ quest, however high its level.
func TestPlanNeverStartsALaterActQuest(t *testing.T) {
	c := task.Template("normal_all", false)
	w := nmWorld("normal", 40)
	for _, q := range []string{task.QDen, task.QBurial, task.QTools, task.QTower} {
		w.Quests[q] = task.QuestState{Stage: task.StageDone, Ev: task.EvMemory}
	}
	for act, want := range map[int]string{1: task.QAndariel} {
		w.Act = act
		if p := c.Plan(w, nil); p.Group != want {
			t.Fatalf("act %d: %+v", act, p)
		}
	}
	w.Quests[task.QAndariel] = task.QuestState{Stage: task.StageCleared, Ev: task.EvMemory}
	if p := c.Plan(w, nil); p.Group != task.QAct1Done {
		t.Fatalf("after Andariel: %+v", p)
	}
}

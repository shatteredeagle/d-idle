package task_test

import (
	"encoding/json"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/plandata"
	"d2lod-bot/internal/task"
)

func nmWorld(diff string, level int, done ...string) task.World {
	w := task.World{Act: 1, Difficulty: diff, Area: task.RogueEncampment, InTown: true, Level: level, Potions: 6, HasTP: true,
		Quests: map[string]task.QuestState{}}
	for _, q := range done {
		w.Quests[q] = task.QuestState{Stage: task.StageDone, Ev: task.EvSave}
	}
	return w
}

func TestNightmareTemplates(t *testing.T) {
	for _, name := range []string{"nightmare_all", "hell_all"} {
		if c := task.Template(name, false); c == nil || len(c.Entries) != len(task.Template("normal_all", false).Entries) {
			t.Fatalf("%s: %+v", name, c)
		}
	}
	nm, hell := task.Template("nightmare_all", false), task.Template("hell_all", false)
	for i, e := range nm.Entries {
		if h := hell.Entries[i]; h.Quest != e.Quest || h.MinLevel < e.MinLevel || e.MinLevel < 36 {
			t.Errorf("%s: nm %d hell %d", e.Quest, e.MinLevel, h.MinLevel)
		}
	}
}

func TestNightmarePlanGatesAndDifficulty(t *testing.T) {
	c := task.Template("nightmare_all", false)
	if p := c.Plan(nmWorld("normal", 80), nil); !p.Stuck {
		t.Fatalf("normal world not held: %+v", p)
	}
	// Normal quests all done does not count in Nightmare: quests are per difficulty.
	p := c.Plan(nmWorld("nightmare", 40), nil)
	if p.Group != task.QDen {
		t.Fatalf("clvl 40 want den: %+v", p)
	}
	if p := c.Plan(nmWorld("nightmare", 20), nil); p.Group != "" || p.Done || p.Stuck {
		t.Fatalf("clvl 20 want a level step: %+v", p)
	}
	// Andariel's floor is 45 even though the act floor is 36.
	done := []string{task.QDen, task.QBurial, task.QTools, task.QTower}
	if p := c.Plan(nmWorld("nightmare", 44, done...), nil); p.Group == task.QAndariel {
		t.Fatalf("andariel at 44: %+v", p)
	}
	if p := c.Plan(nmWorld("nightmare", 45, done...), nil); p.Group != task.QAndariel {
		t.Fatalf("andariel at 45: %+v", p)
	}
	// Hell Den (a cave) needs the dungeon floor.
	h := task.Template("hell_all", false)
	if p := h.Plan(nmWorld("hell", 70), nil); p.Group == task.QDen {
		t.Fatalf("hell den at 70: %+v", p)
	}
	if p := h.Plan(nmWorld("hell", 75), nil); p.Group != task.QDen {
		t.Fatalf("hell den at 75: %+v", p)
	}
}

func TestNextDifficulty(t *testing.T) {
	for _, c := range []struct {
		cur  string
		baal bool
		want string
	}{{"normal", false, ""}, {"normal", true, "nightmare"}, {"", true, "nightmare"}, {"nightmare", true, "hell"}, {"hell", true, ""}} {
		if got := task.NextDifficulty(c.cur, c.baal); got != c.want {
			t.Errorf("%+v got %q", c, got)
		}
	}
	if ok, _ := task.ReadyForNext("nightmare", 35, nil); ok {
		t.Error("nightmare at 35")
	}
	if ok, _ := task.ReadyForNext("nightmare", 40, nil); !ok {
		t.Error("nightmare at 40")
	}
	if ok, _ := task.ReadyForNext("hell", 70, nil); ok {
		t.Error("hell at 70")
	}
}

// The level step between quests must be answerable by the progression file: every campaign floor is reachable by
// farming the listed areas (some step leaves at or above it, or the floor is the first entry level).
func TestFloorsCoveredByProgression(t *testing.T) {
	for diff, name := range map[string]string{"nightmare": "nightmare_all", "hell": "hell_all"} {
		raw, err := plandata.FS.ReadFile("progression-" + diff + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var v map[string]json.RawMessage
		if err := json.Unmarshal(raw, &v); err != nil {
			t.Fatal(err)
		}
		var steps []struct {
			Enter int `json:"enter_clvl"`
			Leave int `json:"leave_clvl"`
		}
		if err := json.Unmarshal(v[diff], &steps); err != nil {
			t.Fatal(err)
		}
		top := 0
		for _, s := range steps {
			top = max(top, s.Leave)
		}
		for _, e := range task.Template(name, false).Entries {
			if e.MinLevel > top {
				t.Errorf("%s %s floor %d above the progression top %d", diff, e.Quest, e.MinLevel, top)
			}
		}
	}
}

func TestReadyForNextResistGate(t *testing.T) {
	good := d2.Resists{Fire: 60, Cold: 60, Lightning: 60, Poison: 50}
	if ok, why := task.ReadyForNext("nightmare", 40, &good); !ok {
		t.Errorf("capped resists refused: %s", why)
	}
	low := good
	low.Cold = 40 // 40-40 = 0 after the Nightmare penalty
	if ok, why := task.ReadyForNext("nightmare", 40, &low); ok || why == "" {
		t.Errorf("low cold passed: %v %q", ok, why)
	}
	if ok, _ := task.ReadyForNext("hell", 80, &good); ok { // 60-100 < 25
		t.Error("hell passed with 60 resists")
	}
}

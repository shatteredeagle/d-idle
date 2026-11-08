package watch

import (
	"encoding/json"
	"strings"
	"testing"

	questdata "d2lod-bot/data/quests"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/task"
	"d2lod-bot/internal/unlock"
)

func questsOf(r *funRig) QuestLogView {
	var v QuestLogView
	r.getJSON(QuestLogRoute, &v)
	return v
}

func questEv(id string, stage, diff int, seq uint64) goals.Event {
	return goals.Event{Type: goals.EvQuest, Quest: id, Stage: stage, Done: stage >= 3, Diff: diff, Seq: seq}
}

func TestQuestLogGoldenTwoClasses(t *testing.T) {
	for _, class := range []string{"Sorceress", "Paladin"} {
		t.Run(class, func(t *testing.T) {
			r, eng := shippedRig(t, class)
			v := questsOf(r)
			if v.Open || v.Teaser == "" || len(v.Quests) != 0 || v.NotGiven != 0 {
				t.Fatalf("locked view: %+v", v)
			}
			// a quest of another act and a quest not yet given change nothing, and the log stays shut
			r.feed(questEv("radament", 1, 0, 1), questEv(task.QTools, 0, 0, 2))
			if eng.Allowed(QuestLogID) {
				t.Fatal("opened without the Den")
			}
			r.feed(questEv(task.QDen, 1, 0, 3)) // Akara hands it out
			if !eng.Allowed(QuestLogID) {
				t.Fatal("the Den quest did not open the quest log")
			}
			r.feed(questEv(task.QTools, 1, 0, 4), questEv(task.QDen, 2, 0, 5))
			b, _ := json.Marshal(questsOf(r))
			golden(t, "questlog-"+strings.ToLower(class), string(b)+"\n")
			found := false
			for _, l := range storyLines(r) {
				found = found || strings.Contains(l, "Unlocked: Quest log.")
			}
			if !found {
				t.Fatalf("no reveal line: %v", storyLines(r))
			}
		})
	}
}

func TestQuestLogStagesReplayAndDifficulty(t *testing.T) {
	r, _ := shippedRig(t, "Sorceress")
	r.feed(questEv(task.QDen, 1, 0, 1), questEv(task.QDen, 2, 0, 2))
	r.feed(questEv(task.QDen, 1, 0, 3)) // a lower stage arriving late: no step back
	v := questsOf(r)
	if len(v.Quests) != 1 || v.Quests[0].State != "cleared" || v.Quests[0].Objective != "Return to Akara." {
		t.Fatalf("%+v", v.Quests)
	}
	r.feed(questEv(task.QDen, 3, 0, 4))
	v = questsOf(r)
	if v.Quests[0].State != "done" || v.Quests[0].Reward != "One skill point." || v.Difficulty != "normal" || v.NotGiven != 5 {
		t.Fatalf("%+v", v)
	}
	r.feed(questEv(task.QDen, 1, 1, 5)) // the same quest again in Nightmare: shown from its start, no normal reward promised
	v = questsOf(r)
	if v.Difficulty != "nightmare" || v.Quests[0].State != "started" || v.Quests[0].Reward != "" {
		t.Fatalf("%+v", v)
	}
	if got := len(r.fun.Unlocks.Errors()); got != 0 {
		t.Fatal(r.fun.Unlocks.Errors())
	}
}

func TestQuestLogUnknownRowAndTable(t *testing.T) {
	l, _ := unlock.Parse([]byte(`{"schema":"unlock.ladder/1","entries":[
 {"id":"chronicle","group":"ui","trigger":{"kind":"kills_of_kind","args":{"kind":"*","min":1}},"reveal":{"title":"C","text":"c"}}]}`))
	eng, _ := unlock.Open("", "p", l, nil)
	if _, err := NewQuestLog(&Unlocks{Engine: eng}, nil, nil); err == nil || !strings.Contains(err.Error(), `"questlog"`) {
		t.Fatalf("missing row: %v", err)
	}
	good, err := ParseQuestTable(questdata.JSON)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range good.Quests { // every id is one the task layer emits
		switch q.ID {
		case task.QDen, task.QBurial, task.QTools, task.QCain, task.QTower, task.QAndariel:
		default:
			t.Errorf("quest %q is not a task layer id", q.ID)
		}
	}
	mut := func(f func(*QuestTable)) error {
		var c QuestTable
		b, _ := json.Marshal(good)
		json.Unmarshal(b, &c)
		f(&c)
		b, _ = json.Marshal(c)
		_, err := ParseQuestTable(b)
		return err
	}
	if err := mut(func(c *QuestTable) { c.Quests[1].ID = c.Quests[0].ID }); err == nil || !strings.Contains(err.Error(), `duplicate quest "den_of_evil"`) {
		t.Errorf("duplicate: %v", err)
	}
	if err := mut(func(c *QuestTable) { delete(c.Quests[0].Stages, "cleared") }); err == nil || !strings.Contains(err.Error(), `"den_of_evil" stage "cleared"`) {
		t.Errorf("missing stage: %v", err)
	}
	if err := mut(func(c *QuestTable) { c.Quests[0].Reward = map[string]string{"easy": "x"} }); err == nil || !strings.Contains(err.Error(), `unknown difficulty "easy"`) {
		t.Errorf("difficulty: %v", err)
	}
}

func TestQuestStageTriggerValidation(t *testing.T) {
	for _, bad := range []string{
		`{"kind":"quest_stage","args":{"quest":"x","min":4}}`,
		`{"kind":"quest_stage","args":{"min":1}}`,
	} {
		doc := `{"schema":"unlock.ladder/1","entries":[{"id":"a","group":"ui","trigger":` + bad + `,"reveal":{"title":"t","text":"x"}}]}`
		if _, err := unlock.Parse([]byte(doc)); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
}

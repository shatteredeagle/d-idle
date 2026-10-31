package main

import (
	"os"
	"path/filepath"
	"testing"

	"d2lod-bot/internal/quest"
	"d2lod-bot/internal/saves"
	"d2lod-bot/internal/task"
)

func TestSaveQuestReaderMapsWordsAndKeepsPriorKnowledgeOnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Booti.d2s"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	var p saves.Progress
	p.Difficulty[0].Quests = []saves.QuestState{
		{ID: "den_of_evil", Word: 0x0001},              // done
		{ID: "sisters_burial_grounds", Word: 0x000c},   // started
		{ID: "tools_of_the_trade", Word: 0},            // untouched: not reported
		{ID: "forgotten_tower", Word: 0x0002 | 0x0004}, // started with the reward pending: cleared
		{ID: "sisters_to_the_slaughter", Word: 0x8001}, // done earlier
	}
	p.Difficulty[0].Waypoints = []saves.Waypoint{{Level: 3, Active: true}, {Level: 4}}
	r := newSaveQuestReader(func() string { return dir }, func() string { return "Booti" })
	r.read = func(string) (saves.Progress, error) { return p, nil }
	v, ok := r.Read(0)
	if !ok {
		t.Fatal("no view")
	}
	want := map[string]task.Stage{task.QDen: task.StageDone, task.QBurial: task.StageStarted, task.QTower: task.StageCleared, task.QAndariel: task.StageDone}
	for q, st := range want {
		if got := v.Quests[q]; got.Stage != st || got.Ev != task.EvSave {
			t.Errorf("%s: %+v want stage %d from the save", q, got, st)
		}
	}
	if _, ok := v.Quests[task.QTools]; ok {
		t.Error("an untouched quest must be left to the proxies")
	}
	if !v.WPs[3] || v.WPs[4] {
		t.Errorf("waypoints %v", v.WPs)
	}
	if _, ok := newSaveQuestReader(func() string { return dir }, func() string { return "" }).Read(0); ok {
		t.Error("unknown character name must not read")
	}
}

// fakeMem is a memQuests with fixed words.
type fakeMem struct {
	words map[string]uint16
	wps   map[int]bool
}

func (f fakeMem) Word(_ int, id string) (uint16, bool) { w, ok := f.words[id]; return w, ok }
func (fakeMem) Evidence() quest.Evidence               { return quest.EvMemory }
func (f fakeMem) Waypoint(_ int, a int) (bool, bool) {
	on, ok := f.wps[a]
	return on, ok
}

func TestQuestReaderPrefersMemoryOverTheSave(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "Booti.d2s"), []byte("x"), 0o644)
	var p saves.Progress
	p.Difficulty[0].Quests = []saves.QuestState{
		{ID: "den_of_evil", Word: 0x000c},            // the save lags: started
		{ID: "sisters_burial_grounds", Word: 0x0001}, // the save says done, memory does not know it
	}
	p.Difficulty[0].Waypoints = []saves.Waypoint{{Level: 3, Active: false}, {Level: 4, Active: true}}
	r := newSaveQuestReader(func() string { return dir }, func() string { return "Booti" })
	r.read = func(string) (saves.Progress, error) { return p, nil }
	r.mem = fakeMem{words: map[string]uint16{"den_of_evil": 0x0001}, wps: map[int]bool{3: true}}
	v, ok := r.Read(0)
	if !ok {
		t.Fatal("no view")
	}
	if got := v.Quests[task.QDen]; got.Stage != task.StageDone || got.Ev != task.EvMemory {
		t.Errorf("den %+v want done from memory", got)
	}
	if got := v.Quests[task.QBurial]; got.Stage != task.StageDone || got.Ev != task.EvSave {
		t.Errorf("burial %+v want the save as fallback", got)
	}
	if !v.WPs[3] || !v.WPs[4] {
		t.Errorf("waypoints %v", v.WPs)
	}
	if r.status("den_of_evil") != quest.Done {
		t.Error("dialog row status must use memory")
	}
	// memory alone works without a readable save
	r2 := newSaveQuestReader(func() string { return dir }, func() string { return "" })
	r2.mem = r.mem
	if v, ok := r2.Read(0); !ok || v.Quests[task.QDen].Ev != task.EvMemory {
		t.Errorf("memory-only read %v %v", v, ok)
	}
}

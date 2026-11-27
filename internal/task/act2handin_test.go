package task_test

import (
	"strings"
	"testing"

	"d2lod-bot/internal/task"
)

func TestAct2HandinSpecLoads(t *testing.T) {
	if err := task.Act2HandinSpecErr(); err != nil {
		t.Fatal(err)
	}
}

// handinNeeded lists which of the hand-in talks (Tyrael, Jerhyn, Meshif) are needed in w.
func handinNeeded(w task.World) string {
	var out []string
	for _, q := range []string{task.QDuriel, task.QAct2Done} {
		for _, n := range task.Handler(q) {
			for _, k := range []string{"tyrael", "jerhyn", "meshif"} {
				if strings.HasSuffix(n.Step.ID, "/"+k) {
					if need, _ := n.Step.Need(w); need {
						out = append(out, k)
					}
				}
			}
		}
	}
	return strings.Join(out, ",")
}

// The Seven Tombs word through the hand-in (bit order S from the d2d port): staff placed 0x0004, Duriel dead +0x0020,
// Tyrael +0x2000, Jerhyn +0x0010, Meshif leaves 0x0001. Exactly one talk is needed at each point, in that order; the
// kill holds the stage at Cleared throughout, so a stage-only check fails here (Tyrael stays needed, Meshif opens early).
func TestAct2HandinFollowsTheQuestBits(t *testing.T) {
	type pt struct {
		word  uint16
		stage task.Stage
		act   int
		area  task.Area
		town  bool
		want  string
	}
	seq := []pt{
		{0x0004, task.StageStarted, 2, task.CanyonOfMagi, false, ""},
		{0x0024, task.StageCleared, 2, task.DurielsLair, false, "tyrael"},
		{0x2024, task.StageCleared, 2, task.LutGholein, true, "jerhyn"},
		{0x2034, task.StageCleared, 2, task.LutGholein, true, "meshif"},
		{0x0001, task.StageDone, 2, task.LutGholein, true, "meshif"}, // bit 0 set, the boat has not left yet
		{0x0001, task.StageDone, 3, task.KurastDocks, true, ""},
	}
	for _, caster := range []bool{false, true} {
		for _, p := range seq {
			w := task.World{Act: p.act, Level: 28, Caster: caster, Area: p.area, InTown: p.town, Potions: 20,
				Items: map[string]int{task.ItemStaffPlaced: 1}, Killed: map[string]bool{task.BossDuriel: p.word&0x0020 != 0},
				WPs: map[task.Area]bool{}, Quests: map[string]task.QuestState{
					task.QStaff:  {Stage: task.StageDone, Ev: task.EvMemory},
					task.QDuriel: {Stage: p.stage, Raw: p.word, Ev: task.EvMemory}}}
			if got := handinNeeded(w); got != p.want {
				t.Errorf("caster %v word %#04x act %d: needed %q, want %q", caster, p.word, p.act, got, p.want)
			}
		}
	}
}

// The Act 2 word may read done before Jerhyn's line: Jerhyn is still needed and Meshif waits for his bit.
func TestAct2HandinJerhynBeforeTheBoatDespiteTheActWord(t *testing.T) {
	for _, caster := range []bool{false, true} {
		w := task.World{Act: 2, Level: 28, Caster: caster, Area: task.LutGholein, InTown: true, Potions: 20,
			Items: map[string]int{task.ItemStaffPlaced: 1}, Killed: map[string]bool{task.BossDuriel: true}, WPs: map[task.Area]bool{},
			Quests: map[string]task.QuestState{
				task.QDuriel:   {Stage: task.StageCleared, Raw: 0x2024, Ev: task.EvMemory},
				task.QAct2Word: {Stage: task.StageDone, Ev: task.EvMemory}}}
		if got := handinNeeded(w); got != "jerhyn" {
			t.Errorf("caster %v: needed %q, want jerhyn", caster, got)
		}
	}
}

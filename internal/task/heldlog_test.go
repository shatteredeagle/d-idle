package task_test

import (
	"fmt"
	"strings"
	"testing"

	"d2lod-bot/internal/task"
)

// The held line changes with the clvl only, so the bot's change-only logger (noteAndariel) emits one line per clvl.
// Logging must not change the gate decision.
func TestHeldLineOncePerClvl(t *testing.T) {
	hold := func(w *task.World) {
		for _, q := range []string{task.QDen, task.QBurial, task.QTools, task.QTower} {
			w.Quests[q] = task.QuestState{Stage: task.StageDone}
		}
		w.AndarielHold = true
	}
	for _, merc := range []string{"alive", "dead"} {
		g, _ := task.GateFor(task.QAndariel)
		var last string
		lines := 0
		for _, lvl := range []int{8, 8, 8, 14, 14, 16, 16, 16, 17} {
			w := floorWorld(lvl, merc, hold)
			before := g.NotReady(w)
			line := task.AndarielLine(w)
			if after := g.NotReady(w); after != before || before == "" {
				t.Errorf("%s clvl %d: gate changed %q -> %q", merc, lvl, before, after)
			}
			if !strings.HasPrefix(line, "boss: held for andariel (") {
				t.Fatalf("%s clvl %d: line %q", merc, lvl, line)
			}
			want := fmt.Sprintf(" clvl=%d floor=%d merc=%s", lvl, task.AndarielFloor(w), merc)
			if !strings.HasSuffix(line, want) {
				t.Errorf("%s: line %q lacks suffix %q", merc, line, want)
			}
			if line != last {
				lines++
				last = line
			}
		}
		if lines != 4 { // clvl 8, 14, 16, 17
			t.Errorf("%s: %d distinct lines, want 4", merc, lines)
		}
	}
}

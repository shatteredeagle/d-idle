package task_test

import (
	"context"
	"testing"

	"d2lod-bot/internal/quest"
	"d2lod-bot/internal/task"
	"d2lod-bot/internal/task/sim"
)

// denGame is the logged Den state of the soak (2026-10-29): the sweep ends, the live quest word stays 0x001c (started, not
// pending), and only a Talk with Akara moves it. payAt says how the fake Akara behaves on the probe.
type denGame struct {
	*sim.Game
	stage task.Stage
	swept bool
	pays  bool // the Talk pays the reward (word 0x001c -> done)
}

func (g *denGame) World(ctx context.Context) (task.World, error) {
	w, _ := g.Game.World(ctx)
	w.Quests = map[string]task.QuestState{task.QDen: {Stage: g.stage, Ev: task.EvMemory}}
	w.Cleared = map[task.Area]bool{}
	w.Swept = map[task.Area]bool{task.DenOfEvil: g.swept}
	return w, nil
}

func (g *denGame) Clear(ctx context.Context, a task.Area) error {
	g.Log = append(g.Log, "clear")
	g.swept = true
	return nil
}

func (g *denGame) Talk(ctx context.Context, npc, pick string) error {
	g.Log = append(g.Log, "talk:"+npc+":"+pick)
	if npc == task.NPCAkara && pick == "quest" && g.stage == task.StageNone {
		g.stage = task.StageStarted
	}
	if npc == task.NPCAkara && pick == "reward" && g.pays && g.stage == task.StageStarted {
		g.stage = task.StageDone
	}
	return nil
}

func TestDenClaimProbeFixtureWords(t *testing.T) {
	// V (docs: quest.dialog subWhen): 0x001c is started without the pending bit, 0x001e is the reward pending, 0x8002 / 0x0001 done.
	for _, tc := range []struct {
		w       uint16
		st      quest.Status
		pending bool
	}{{0x001c, quest.Started, false}, {0x001e, quest.Started, true}, {0x8002, quest.Done, true}, {0x0001, quest.Done, false}} {
		if st, p := quest.Interpret(tc.w); st != tc.st || p != tc.pending {
			t.Errorf("word %#06x: got %v/%v want %v/%v", tc.w, st, p, tc.st, tc.pending)
		}
	}
}

func driveDen(t *testing.T, g *denGame, ticks int) {
	t.Helper()
	ex := task.NewExecutor(g, g)
	c := task.NormalAct1(task.Act1Options{SkipTools: true, SkipTower: true})
	g.SetLevel(6)
	for i := 0; i < ticks; i++ {
		r, err := ex.Tick(context.Background(), c)
		if err != nil {
			t.Fatal(err)
		}
		if r.Group != task.QDen && r.Group != "" {
			return // the campaign moved on to the next quest
		}
	}
}

func count(log []string, s string) int {
	n := 0
	for _, l := range log {
		if l == s {
			n++
		}
	}
	return n
}

// A sweep that leaves the word open is followed by exactly one Akara visit; when it pays, the Den is done and never claimed again.
func TestDenClaimProbePaysAndIsIdempotent(t *testing.T) {
	g := &denGame{Game: sim.New(true), stage: task.StageStarted, pays: true}
	driveDen(t, g, 60)
	if g.stage != task.StageDone {
		t.Fatalf("stage %v, log %v", g.stage, g.Log)
	}
	if n := count(g.Log, "talk:Akara:reward"); n != 1 {
		t.Fatalf("%d claims, want 1; log %v", n, g.Log)
	}
	if c, p := count(g.Log, "clear"), pos(g.Log, "talk:Akara:reward"); c != 1 || p < pos(g.Log, "clear") {
		t.Fatalf("order wrong: log %v", g.Log)
	}
}

// Akara that does not pay: the probe is optional and gives up after the executor's failure budget instead of looping.
func TestDenClaimProbeGivesUp(t *testing.T) {
	g := &denGame{Game: sim.New(true), stage: task.StageStarted}
	driveDen(t, g, 120)
	if n := count(g.Log, "talk:Akara:reward"); n == 0 || n > 4 {
		t.Fatalf("%d probes, want 1..4; log %v", n, g.Log)
	}
}

// Without a finished sweep there is no probe (no Akara trip while monsters are still being hunted).
func TestDenClaimProbeNeedsASweep(t *testing.T) {
	w := task.World{Quests: map[string]task.QuestState{task.QDen: {Stage: task.StageStarted, Ev: task.EvMemory}}, Swept: map[task.Area]bool{}}
	for _, n := range task.Handler(task.QDen) {
		if n.Step.ID == "quest.den_of_evil/claim-probe" {
			if need, _ := n.Step.Need(w); need {
				t.Fatal("probe needed before any sweep")
			}
			w.Swept[task.DenOfEvil] = true
			if need, _ := n.Step.Need(w); !need {
				t.Fatal("probe not needed after a sweep with the word open")
			}
			return
		}
	}
	t.Fatal("no claim-probe node")
}

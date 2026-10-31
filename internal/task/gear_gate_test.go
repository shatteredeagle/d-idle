package task_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"d2lod-bot/internal/task"
	"d2lod-bot/internal/task/sim"
)

// drive ticks the Normal Act 1 campaign of a clvl 8 character with a full kit and returns the sim log.
func driveGear(t *testing.T, mut func(w *task.World)) *sim.Game {
	t.Helper()
	g := sim.New(true)
	g.SetLevel(8)
	g.XPStep = 0 // leveling does not change the picture
	g.W.Potions, g.W.HasTP, g.W.InTown = 10, true, true
	g.MarkDone(task.QDen)
	mut(&g.W)
	ex := task.NewExecutor(g, g)
	c := task.NormalAct1(task.Act1Options{SkipTower: true})
	for i := 0; i < 40; i++ {
		if _, err := ex.Tick(context.Background(), c); err != nil {
			t.Fatal(err)
		}
	}
	return g
}

func fieldTrips(g *sim.Game) []string {
	var out []string
	for _, l := range g.Log {
		var a int
		if _, err := fmt.Sscanf(l, "travel:%d", &a); err == nil && !(a == 1) {
			out = append(out, l)
		}
	}
	return out
}

// V: Soaky 2026-10-29. A gated (naked / destitute) character never gets a field travel, a clear or a boss kill, whatever
// quest is next; the healthy twin runs the same campaign into the field as before.
func TestGatedCharacterStaysOutOfTheField(t *testing.T) {
	healthy := driveGear(t, func(w *task.World) {})
	if len(fieldTrips(healthy)) == 0 {
		t.Fatalf("healthy clvl 8 never left town: %v", healthy.Log)
	}
	gated := driveGear(t, func(w *task.World) {
		w.Gated, w.GatedWhy, w.SafeArea = true, "no weapon and no body armor", 2
	})
	for _, l := range gated.Log {
		var a int
		if _, err := fmt.Sscanf(l, "travel:%d", &a); err == nil && a != 2 {
			t.Fatalf("gated character travelled: %v", gated.Log)
		}
		if len(l) > 5 && (l[:5] == "clear" || l[:4] == "kill") {
			t.Fatalf("gated character fought: %v", gated.Log)
		}
	}
}

func TestBannedAreaIsNotEntered(t *testing.T) {
	g := driveGear(t, func(w *task.World) {
		w.Banned = map[task.Area]bool{task.Barracks: true, task.BurialGrounds: true, task.DarkWood: true}
	})
	for _, a := range []task.Area{task.Barracks, task.BurialGrounds} {
		if g.Calls[fmt.Sprintf("travel:%d", a)] > 0 {
			t.Fatalf("travelled to banned area %d: %v", a, g.Log)
		}
	}
}

// G1 attempt 4: a gated character looped talk-charsi retry/skipped and 'Go to the Barracks' not ready. It must not start
// the Tools quest at all: no Charsi greeting, no waypoint trip. A healthy twin still talks to her.
func TestGatedCharacterDoesNotStartToolsQuest(t *testing.T) {
	gated := driveGear(t, func(w *task.World) {
		w.Gated, w.GatedWhy, w.SafeArea = true, "no weapon and no body armor", 2
	})
	for _, l := range gated.Log {
		if strings.HasPrefix(l, "talk") || strings.HasPrefix(l, "wp") {
			t.Fatalf("gated character started a quest step: %v", gated.Log)
		}
	}
	healthy := driveGear(t, func(w *task.World) {})
	talked := false
	for _, l := range healthy.Log {
		talked = talked || strings.HasPrefix(l, "talk")
	}
	if !talked {
		t.Fatalf("healthy character never talked: %v", healthy.Log)
	}
}

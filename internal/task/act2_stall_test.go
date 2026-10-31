package task_test

import (
	"context"
	"testing"

	"d2lod-bot/internal/task"
)

// stallGame fails Travel with a stuck error until a recovery ran.
type stallGame struct {
	*act2Game
	tele      bool // TeleportToward works
	fails     int  // Travel failures left
	cleared   int
	teleports int
}

func (g *stallGame) Travel(ctx context.Context, a task.Area) error {
	if g.fails > 0 {
		g.fails--
		return task.Errorf(task.KindUI, "nav stuck")
	}
	return g.act2Game.Travel(ctx, a)
}
func (g *stallGame) ClearAround(context.Context) error { g.cleared++; return nil }
func (g *stallGame) TeleportToward(ctx context.Context, a task.Area) error {
	g.teleports++
	if !g.tele {
		return task.ErrNoTeleport
	}
	return g.act2Game.Travel(ctx, a)
}

func runStall(g task.Actions) error {
	s := task.StepTravelA2("t", "travel", task.FarOasis, func(task.World) bool { return false })
	return s.Do(context.Background(), g)
}

func TestStallClearsThenRetries(t *testing.T) {
	g := &stallGame{act2Game: newAct2(false, 20), fails: 1}
	if err := runStall(g); err != nil {
		t.Fatal(err)
	}
	if g.cleared != 1 || g.teleports != 1 {
		t.Fatalf("cleared %d teleports %d", g.cleared, g.teleports)
	}
}

func TestStallPrefersTeleport(t *testing.T) {
	g := &stallGame{act2Game: newAct2(false, 20), fails: 1, tele: true}
	if err := runStall(g); err != nil {
		t.Fatal(err)
	}
	if g.cleared != 0 || g.teleports != 1 {
		t.Fatalf("cleared %d teleports %d", g.cleared, g.teleports)
	}
}

func TestStallGivesUpAfterTwoRounds(t *testing.T) {
	g := &stallGame{act2Game: newAct2(false, 20), fails: 99}
	if err := runStall(g); err == nil {
		t.Fatal("want error")
	}
	if g.cleared != 2 {
		t.Fatalf("cleared %d", g.cleared)
	}
}

func TestPrereqIsNotRecovered(t *testing.T) {
	g := &stallGame{act2Game: newAct2(false, 20)}
	n := &prereqGame{stallGame: g}
	if err := runStall(n); err == nil || g.cleared != 0 {
		t.Fatalf("err %v cleared %d", err, g.cleared)
	}
}

type prereqGame struct{ *stallGame }

func (prereqGame) Travel(context.Context, task.Area) error { return task.Errorf(task.KindPrereq, "no") }

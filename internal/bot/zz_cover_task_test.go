package bot

import (
	"context"
	"errors"
	"testing"
	"time"

	"d2lod-bot/internal/task"
)

func primed(t *testing.T) (*rig, taskActs) {
	t.Helper()
	g := newRig(t)
	g.r.d.Task = &TaskDeps{}
	if _, err := (taskEnv{g.r}).World(context.Background()); err != nil {
		t.Fatal(err)
	}
	return g, taskActs{r: g.r}
}

func TestCoverTagClassifiesFaults(t *testing.T) {
	_, acts := primed(t)
	if acts.tag(nil) != nil {
		t.Fatal("nil must stay nil")
	}
	cases := []struct {
		err  error
		want task.Kind
	}{
		{ErrLeftGame, task.KindDeath},
		{ErrPortalOpened, task.KindFatal},
		{errors.New("nav: no route from 1 to 99"), task.KindNotFound},
	}
	for _, c := range cases {
		if got := task.Classify(acts.tag(c.err)); got != c.want {
			t.Errorf("%v: %v want %v", c.err, got, c.want)
		}
	}
	if err := acts.tag(errBoom); !errors.Is(err, errBoom) {
		t.Fatalf("plain errors pass through: %v", err)
	}
}

func TestCoverToTownWithoutPortalIsAResourceFault(t *testing.T) {
	g, acts := primed(t)
	g.town.toTownErr = errors.New("town: no town portal scroll")
	g.w.set(func(w *world) { w.area = 2 })
	g.r.poll()
	err := acts.ToTown(context.Background())
	if task.Classify(err) != task.KindResource {
		t.Fatalf("got %v (%v)", err, task.Classify(err))
	}
}

func TestCoverToTownHonoursCancellation(t *testing.T) {
	g, acts := primed(t)
	g.town.toTownBlock = true
	g.w.set(func(w *world) { w.area = 2 })
	g.r.poll()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- acts.ToTown(ctx) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("blocked ToTown reported success")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ToTown ignores ctx")
	}
}

func TestCoverChoresFailureIsAResourceFault(t *testing.T) {
	g, acts := primed(t)
	g.town.choresErr = errors.New("vendor refused")
	if err := acts.Chores(context.Background()); task.Classify(err) != task.KindResource {
		t.Fatalf("got %v", err)
	}
}

func TestCoverKillBossBranches(t *testing.T) {
	ctx := context.Background()
	t.Run("not found after exploring", func(t *testing.T) {
		g, acts := primed(t)
		err := acts.KillBoss(ctx, "Andariel")
		if task.Classify(err) != task.KindNotFound {
			t.Fatalf("got %v", err)
		}
		if g.w.count("explore") != 1 {
			t.Fatalf("explore calls %d", g.w.count("explore"))
		}
	})
	t.Run("already killed this run counts as done", func(t *testing.T) {
		_, acts := primed(t)
		acts.r.noteKill("Andariel")
		if err := acts.KillBoss(ctx, "Andariel"); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("explore error propagates", func(t *testing.T) {
		g, acts := primed(t)
		g.mover.exploreFn = func(context.Context, func() bool) error { return errBoom }
		if err := acts.KillBoss(ctx, "Andariel"); !errors.Is(err, errBoom) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("seen, killed, looted", func(t *testing.T) {
		g, acts := primed(t)
		g.w.addMonster(900, 12, 5, 0)
		g.r.poll()
		if err := acts.KillBoss(ctx, "Andariel"); err != nil {
			t.Fatal(err)
		}
		if g.w.count("kill 900") != 1 || g.w.count("pickup") == 0 {
			t.Fatalf("calls %q", g.w.log())
		}
		if !g.r.killedBoss("Andariel") || !g.r.loop.task.killed["Andariel"] {
			t.Fatal("kill not recorded")
		}
	})
	t.Run("fighter leaves the game", func(t *testing.T) {
		g, acts := primed(t)
		g.w.addMonster(900, 12, 5, 0)
		g.r.poll()
		g.r.d.Fighter = killFails{g.fight, ErrLeftGame}
		err := acts.KillBoss(ctx, "Andariel")
		if task.Classify(err) != task.KindDeath {
			t.Fatalf("got %v", err)
		}
		if g.r.killedBoss("Andariel") {
			t.Fatal("a chicken exit must not count as a kill")
		}
	})
}

type killFails struct {
	*fakeFighter
	err error
}

func (k killFails) Kill(context.Context, uint32) error { return k.err }

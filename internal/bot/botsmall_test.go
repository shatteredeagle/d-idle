package bot

import (
	"context"
	"testing"

	"d2lod-bot/internal/goals"
)

func TestRestartCount(t *testing.T) {
	for _, c := range []struct {
		name                        string
		games, planned, recycles, w int
	}{
		{"first entry only", 1, 0, 0, 0},
		{"menu observer sees recycles", 18, 0, 17, 17},
		{"d2d: no menu poll, 17 recycles", 1, 0, 17, 17},
		{"planned save points are not restarts", 5, 4, 0, 0},
		{"crash re-entries count on Game.exe", 6, 1, 2, 4},
		{"never negative", 0, 3, 0, 0},
	} {
		if got := restartCount(c.games, c.planned, c.recycles); got != c.w {
			t.Errorf("%s: restartCount(%d,%d,%d) = %d, want %d", c.name, c.games, c.planned, c.recycles, got, c.w)
		}
	}
}

// The fresh game counts as a restart even though the observer never saw the menus (d2d backend).
func TestFreshGameCountsRecycleWithoutObserver(t *testing.T) {
	g := soakyRig(t, 2)
	g.r.d.Cycler = fakeCycler{g.w}
	g.r.syncGames()
	g.r.loop.cleared = map[int]bool{2: true}
	g.w.set(func(w *world) { w.area = 1 })
	g.r.poll()
	acts := taskActs{r: g.r, c: control{goal: &goals.Goal{}}}
	for i := 0; i < 3; i++ {
		g.r.loop.cleared = map[int]bool{2: true}
		if _, err := acts.r.freshGameToLevel(context.Background(), g.r.latest()); err != nil {
			t.Fatal(err)
		}
	}
	g.r.obMu.Lock()
	n, rec, planned := g.r.ob.newGames, g.r.ob.recycles, g.r.ob.plannedGames
	g.r.obMu.Unlock()
	if rec != 3 || restartCount(n, planned, rec) < 3 {
		t.Fatalf("recycles %d newGames %d: restarts %d, want >= 3", rec, n, restartCount(n, planned, rec))
	}
}

// The observer path (syncGames, a new game seen through the menus) forgets the same per-game state as the fresh-game path.
func TestSyncGamesForgetsViaSharedHelper(t *testing.T) {
	g := soakyRig(t, 2)
	g.r.syncGames()
	g.r.loop.cleared = map[int]bool{2: true}
	g.r.loop.sweepKills = map[int]int{2: 5}
	g.r.loop.stepArea, g.r.loop.needBuff, g.r.loop.waitNote = 7, false, true
	g.r.obMu.Lock()
	g.r.ob.newGames++
	g.r.obMu.Unlock()
	g.r.syncGames()
	l := &g.r.loop
	if len(l.cleared) != 0 || l.sweepKills != nil || l.stepArea != 0 || !l.needBuff || l.waitNote {
		t.Fatalf("per-game state survived: %+v", l.cleared)
	}
}

package bot

import (
	"context"
	"sync/atomic"
	"testing"

	"d2lod-bot/internal/env"
)

// fakePlayerGear counts queued actions like invdesk.Desk does; RunPending logs a call and empties the queue.
type fakePlayerGear struct {
	w       *world
	pending atomic.Int32
	runs    atomic.Int32
}

func (f *fakePlayerGear) Pending() int { return int(f.pending.Load()) }
func (f *fakePlayerGear) RunPending(ctx context.Context) []string {
	f.w.call("player-actions")
	f.runs.Add(1)
	f.pending.Store(0)
	return []string{"trash thing done"}
}

// In town the queue runs at once; in the field it waits until nothing hostile is near.
func TestPlayerActionsRunInTownAndWaitForTheFieldFight(t *testing.T) {
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) {})
	pg := &fakePlayerGear{w: g.w}
	pg.pending.Store(1)
	g.r.d.PlayerGear = pg
	g.mover.exploreFn = blockExplore
	g.run()
	defer g.finish()
	g.r.Start(reach(10))
	g.waitFor("town run", func() bool { return pg.runs.Load() == 1 })
	g.waitFor("in the field", func() bool { return g.w.count("explore") > 0 })
	g.w.addMonster(7, 10, 8, 0)
	pg.pending.Store(1)
	g.waitFor("run after the fight", func() bool { return pg.runs.Load() >= 2 })
	g.w.mu.Lock()
	calls := append([]string(nil), g.w.calls...)
	g.w.mu.Unlock()
	last, clear := -1, -1
	for i, c := range calls {
		if c == "player-actions" {
			last = i
		}
		if len(c) >= 5 && c[:5] == "clear" {
			clear = i
		}
	}
	if clear < 0 || last < clear {
		t.Fatalf("player actions ran while a hostile was near: %v", calls)
	}
}

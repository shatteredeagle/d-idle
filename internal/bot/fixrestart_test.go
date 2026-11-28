package bot

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"d2lod-bot/internal/env"
	"d2lod-bot/internal/launch"
)

// d2d: a game lost and restarted by the bot (no menu poll, no recycle) counts as exactly one restart on the minute
// line, and a recycle in the same run is not counted twice.
func TestGameLostRestartCountsOnceWithoutMenus(t *testing.T) {
	g := failsafeRig(t, func(c *Config, d *Deps, tu *env.Tuning) { c.MaxFailsafeRestarts = 3 })
	var restarted atomic.Bool
	g.r.d.Restart = func(ctx context.Context) error { restarted.Store(true); return nil }
	g.mover.exploreFn = func(ctx context.Context, done func() bool) error {
		g.w.set(func(w *world) { w.x += 50; w.exp += 10 })
		g.e.Clock.Sleep(time.Second)
		return nil
	}
	g.w.set(func(w *world) { w.area = 2 })
	g.run()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("exploring", func() bool { return g.w.count("explore") > 2 })
	g.r.obMu.Lock()
	before := restartCount(g.r.ob.newGames, g.r.ob.plannedGames, g.r.ob.recycles, g.r.ob.lostRestarts)
	g.r.obMu.Unlock()
	g.health.mu.Lock()
	g.health.ev = launch.EventExited
	g.health.mu.Unlock()
	g.waitFor("restart", func() bool { return restarted.Load() })
	g.health.mu.Lock()
	g.health.ev = launch.EventNone
	g.health.mu.Unlock()
	n := g.w.count("explore")
	g.waitFor("working again", func() bool { return g.w.count("explore") > n+2 })
	g.r.obMu.Lock()
	after := restartCount(g.r.ob.newGames, g.r.ob.plannedGames, g.r.ob.recycles, g.r.ob.lostRestarts)
	lost, rec := g.r.ob.lostRestarts, g.r.ob.recycles
	g.r.obMu.Unlock()
	g.finish()
	// The farm goal also starts fresh games (recycles, counted by newGameRun); the lost game adds exactly one on top.
	if lost != 1 || after != rec+lost || after < before+1 {
		t.Fatalf("lost %d recycles %d restarts %d -> %d, want lost 1 and restarts = recycles + lost", lost, rec, before, after)
	}
}

// xpRisk is the exp gained since the last save when the save file can be read (d2d: d2dSavedExp), not the whole exp.
func TestXPRiskIsSinceLastSaveAcrossAKill(t *testing.T) {
	var saved uint32 = 1000
	g := newRig(t, func(_ *Config, d *Deps, _ *env.Tuning) {
		d.SavedExp = func() (uint32, bool) { return saved, true }
	})
	now := time.Unix(1000, 0)
	risk := func() int64 {
		g.r.obMu.Lock()
		defer g.r.obMu.Unlock()
		return g.r.ob.xpAtRiskLocked(g.r.d.SavedExp)
	}
	g.w.set(func(w *world) { w.exp = 1500 })
	g.observeAt(now)
	if got := risk(); got != 500 {
		t.Fatalf("saved 1000, exp 1500: xpRisk %d, want 500", got)
	}
	saved = 1500 // Save and Exit rewrote the file
	if got := risk(); got != 0 {
		t.Fatalf("after a save: xpRisk %d, want 0", got)
	}
	g.w.set(func(w *world) { w.exp = 1800 })
	g.observeAt(now.Add(time.Second))
	// kill -9: the 300 unsaved are gone; the character reloads from the save (no menu seen on d2d)
	g.r.noteGameLost()
	g.w.set(func(w *world) { w.exp = 1500 })
	g.observeAt(now.Add(2 * time.Second))
	g.observeAt(now.Add(3 * time.Second)) // a fall counts after a second consistent read
	if got := risk(); got != 0 {
		t.Fatalf("after the kill and re-entry: xpRisk %d, want 0 (the saved value)", got)
	}
}

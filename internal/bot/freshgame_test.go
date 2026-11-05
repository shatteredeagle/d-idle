package bot

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

// freshCycles runs the fresh-game slice every 3 virtual minutes (the live cadence of the seed 13 loop) for the given
// time on a character of the given class; gain is the experience each new game yields before the next check.
func freshCycles(t *testing.T, class d2.Class, hours int, gain uint32) (restarts int, g *rig) {
	g = soakyRig(t, 2)
	g.r.d.Cycler = fakeCycler{g.w}
	g.r.syncGames()
	g.w.set(func(w *world) { w.area = 1; w.exp = 1000 })
	g.r.poll()
	end := g.r.e.Clock.Now().Add(time.Duration(hours) * time.Hour)
	for g.r.e.Clock.Now().Before(end) {
		g.r.loop.cleared = map[int]bool{2: true}
		g.w.set(func(w *world) { w.area = 1 })
		g.r.poll()
		s := *g.r.latest()
		p := *s.Player
		p.Class = class
		s.Player = &p
		if _, err := g.r.freshGameToLevel(context.Background(), &s); err != nil {
			t.Fatal(err)
		}
		g.w.set(func(w *world) { w.exp += gain })
		g.r.e.Clock.Sleep(3 * time.Minute)
	}
	return g.w.count("newgame normal"), g
}

// Seed 13: 75 restarts in 4 h with no XP. With the bound the count is the three gap-spaced ones plus one blocked probe
// per route.freshBlockedSec, for a caster and a non-caster alike (the rule holds no class data).
func TestFreshRestartsBoundedWithoutXP(t *testing.T) {
	for _, c := range []d2.Class{d2.Sorceress, d2.Barbarian} {
		n, g := freshCycles(t, c, 4, 0)
		rt := g.r.e.Tune.Route
		max := rt.FreshMaxNoGain + 4*3600/rt.FreshBlockedSec + 1
		if n < 2 || n > max {
			t.Errorf("class %v: %d restarts in 4 h, want 2..%d", c, n, max)
		}
		if g.r.loop.freshNoGain < rt.FreshMaxNoGain {
			t.Errorf("class %v: no-gain count %d never reached the bound %d", c, g.r.loop.freshNoGain, rt.FreshMaxNoGain)
		}
	}
}

// A cycle that gains experience resets the count: restarts keep the old 3-minute cadence only as far as the gap allows
// nothing, because a gaining cycle is never held back.
func TestFreshRestartCountResetsOnXP(t *testing.T) {
	for _, c := range []d2.Class{d2.Sorceress, d2.Barbarian} {
		n, g := freshCycles(t, c, 1, 500)
		if g.r.loop.freshNoGain != 0 || n < 15 {
			t.Errorf("class %v: %d restarts, noGain %d: a gaining cycle must not be held back", c, n, g.r.loop.freshNoGain)
		}
	}
}

// Stopping the run during the blocked wait returns at once and does not start a game.
func TestFreshBlockedWaitHonoursStop(t *testing.T) {
	g := soakyRig(t, 2)
	g.r.d.Cycler = fakeCycler{g.w}
	g.r.syncGames()
	g.w.set(func(w *world) { w.area = 1 })
	g.r.poll()
	g.r.loop.cleared = map[int]bool{2: true}
	if _, err := g.r.freshGameToLevel(context.Background(), g.r.latest()); err != nil {
		t.Fatal(err)
	}
	g.r.loop.cleared = map[int]bool{2: true}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	g.r.poll()
	if done, _ := g.r.freshGameToLevel(ctx, g.r.latest()); !done {
		t.Fatal("not done")
	}
	if n := g.w.count("newgame normal"); n != 1 {
		t.Fatalf("%d games, want 1", n)
	}
}

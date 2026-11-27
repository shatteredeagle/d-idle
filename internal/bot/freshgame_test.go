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

// Questy 2026-11-26: a broke clvl 13 hero stood 28 minutes in a dry Blood Moor (skipped as dry-recovery, nothing cleared). Monsters do
// not respawn within a game, so a dry farm area needs a new game right away: the xp-gain bound and its 30 minute blocked wait do not
// apply (a floor of five minutes between two such restarts), and the new game lifts the dry skip. Two classes.
func TestDryAreaGetsAFreshGameRightAway(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		g := soakyRig(t, 2)
		g.r.d.Cycler = fakeCycler{g.w}
		g.r.syncGames()
		g.w.set(func(w *world) { w.area = 1; w.exp = 1000; w.class, w.level, w.gold0 = class, 13, 19 })
		g.r.poll()
		// three earlier no-gain restarts: the blocked state of the xp bound
		g.r.loop.freshNoGain, g.r.loop.freshAt, g.r.loop.freshExp = 3, g.r.e.Clock.Now().Add(-time.Minute), 1000
		g.r.skipArea(2, 15*time.Minute, whyDryRecovery)
		before := g.w.count("newgame normal")
		start := g.r.e.Clock.Now()
		done, err := g.r.freshGameToLevel(context.Background(), g.r.latest())
		if err != nil || !done || g.w.count("newgame normal") != before+1 {
			t.Fatalf("%v: done %v err %v new games %d -> %d: no fresh game for a dry area", class, done, err, before, g.w.count("newgame normal"))
		}
		if waited := g.r.e.Clock.Now().Sub(start); waited > 6*time.Minute {
			t.Errorf("%v: waited %v virtual, want at most the five minute floor", class, waited)
		}
		if g.r.loop.freshNoGain != 3 {
			t.Errorf("%v: a dry restart changed the no-gain count to %d", class, g.r.loop.freshNoGain)
		}
		if len(g.r.dryAreas()) != 0 {
			t.Errorf("%v: the dry skip outlived the new game: %v", class, g.r.dryAreas())
		}
		// nothing dry and nothing cleared: no fresh game
		g2 := soakyRig(t, 2)
		g2.r.d.Cycler = fakeCycler{g2.w}
		g2.r.poll()
		if done, _ := g2.r.freshGameToLevel(context.Background(), g2.r.latest()); done || g2.w.count("newgame normal") != 0 {
			t.Errorf("%v: a fresh game without a cleared or dry area", class)
		}
	}
}

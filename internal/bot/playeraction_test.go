package bot

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"d2lod-bot/internal/d2"
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

// errandGear is a desk with one queued town errand: it can only run in town, so the field reports nothing pending.
type errandGear struct {
	w    *world
	done atomic.Bool
	kind string
}

func (f *errandGear) Pending() int {
	if f.done.Load() || f.w.area != 1 {
		return 0
	}
	return 1
}
func (f *errandGear) RunPending(ctx context.Context) []string {
	f.w.call("player-actions")
	f.done.Store(true)
	return []string{"buyid done"}
}
func (f *errandGear) NextKind() string { return f.kind }
func (f *errandGear) ErrandsWaiting() int {
	if f.done.Load() {
		return 0
	}
	return 1
}

// A queued town errand takes the hero home at the next safe moment (not while a hostile is near) and runs in town, for a
// Paladin and a Sorceress; the Why line names the errand.
func TestErrandTripGoesHomeAtASafeMoment(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) {})
		g.w.set(func(w *world) { w.class, w.area = class, 2 })
		eg := &errandGear{w: g.w, kind: "buyid"}
		g.r.d.PlayerGear = eg
		g.mover.exploreFn = blockExplore
		g.w.addMonster(7, 10, 8, 0) // a fight is on when the errand is queued
		g.run()
		g.r.Start(reach(10))
		g.waitFor("errand done in town", func() bool { return eg.done.Load() })
		g.w.mu.Lock()
		calls := append([]string(nil), g.w.calls...)
		g.w.mu.Unlock()
		home, ran, clear := -1, -1, -1
		for i, c := range calls {
			switch {
			case c == "totown" && home < 0:
				home = i
			case c == "player-actions":
				ran = i
			case len(c) >= 5 && c[:5] == "clear" && clear < 0:
				clear = i
			}
		}
		if home < 0 || ran < home || clear < 0 || home < clear {
			t.Fatalf("%v: went home at %d, errand at %d, fight cleared at %d: %v", class, home, ran, clear, calls)
		}
		g.finish()
	}
}

// The errand trip does not start without a waiting errand, and a failed walk home backs off.
func TestErrandTripBacksOffAfterAFailedWalk(t *testing.T) {
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) {})
	eg := &errandGear{w: g.w, kind: "cain"}
	g.r.d.PlayerGear = eg
	g.town.toTownErr = errors.New("no way home")
	g.mover.exploreFn = blockExplore
	g.run()
	defer g.finish()
	g.r.Start(reach(10))
	g.waitFor("first try", func() bool { return g.w.count("totown") >= 1 })
	if n := g.w.count("totown"); n != 1 {
		t.Fatalf("totown %d", n)
	}
}

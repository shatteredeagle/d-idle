package bot

import (
	"strings"
	"sync/atomic"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
)

type leftLoot struct {
	fakeLoot
	n *atomic.Int64
}

func (l leftLoot) LeftBehind() int  { return int(l.n.Load()) }
func (l leftLoot) ClearLeftBehind() { l.n.Store(0) }

// Valuables left on the ground for lack of room raise a trip when the pack is 80 percent full, even below the
// ordinary inventory trip line (InvFullPct), and not when the pack has room.
func TestLeftBehindValuablesRaiseATripFromEightyPercent(t *testing.T) {
	var n atomic.Int64
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { d.Loot = leftLoot{fakeLoot{}, &n} })
	fill := func(plates int) {
		g.w.set(func(w *world) {
			w.area, w.gold0 = 8, 0
			w.items = nil
			for i := 0; i < plates; i++ { // Plate Mail is 2x3 in the fake catalog
				w.items = append(w.items, d2.Unit{ID: uint32(800 + i), Type: d2.TypeItem, Class: 3, Item: &d2.Item{Where: d2.ContInventory}})
			}
		})
	}
	fill(5) // 30 of 40 cells = 75 percent: below the line
	n.Store(2)
	s, _ := g.w.snapshot()
	if plan, _, ok := g.r.choresDue(s); ok && strings.Contains(plan.Reason, "left on the ground") {
		t.Fatalf("75%% full must not raise it: %+v", plan)
	}
	fill(6) // 36 cells = 90 percent
	s, _ = g.w.snapshot()
	plan, _, ok := g.r.choresDue(s)
	if !ok || !plan.Sell || !plan.Stash || !strings.Contains(plan.Reason, "left on the ground") {
		t.Fatalf("90%% full with left-behind valuables: ok=%v plan=%+v", ok, plan)
	}
	n.Store(0)
	g.w.set(func(w *world) { w.items = w.items[:0] })
	fill(5)
	s, _ = g.w.snapshot()
	if plan, _, ok := g.r.choresDue(s); ok && strings.Contains(plan.Reason, "left on the ground") {
		t.Fatalf("nothing left behind must not raise it: %+v", plan)
	}
}

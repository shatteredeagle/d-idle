package bot

import (
	"testing"

	"d2lod-bot/internal/d2"
)

// countTown counts the town's sell judgements.
type countTown struct {
	Townsman
	calls *int
}

func (c countTown) SellableCount(*d2.Snapshot) int { *c.calls++; return 2 }

// sellsnap (goroutine dump 2026-11-27): resInput judged the whole pack on every call, many times a tick, and the watch
// API's guard world (readiness, risk, propose) did it per request behind the backend's client lock. Now one judgement per
// snapshot, a new one when the pack changes, and the guard world never judges. Two classes.
func TestSellCountOncePerSnapshotAndNeverFromTheGuardWorld(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		g := newRig(t)
		g.r.d.Loot = unitLoot{fakeLoot{w: g.w}}
		calls := 0
		g.r.d.Town = countTown{g.r.d.Town, &calls}
		s := &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{Class: class, Level: 11, Gold: 24}, Inventory: &d2.Inventory{Items: []d2.Unit{
			{ID: 1, Item: &d2.Item{Where: d2.ContInventory}}, {ID: 2, Item: &d2.Item{Where: d2.ContInventory}},
		}}}
		for i := 0; i < 5; i++ {
			if n := g.r.resInput(s).Sellable; n != 2 {
				t.Fatalf("%v: sellable %d, want 2", class, n)
			}
		}
		if calls != 1 {
			t.Errorf("%v: 5 resInput calls on one snapshot judged the pack %d times, want 1", class, calls)
		}
		s.Inventory.Items = append(s.Inventory.Items, d2.Unit{ID: 3, Item: &d2.Item{Where: d2.ContInventory}})
		g.r.resInput(s)
		if calls != 2 {
			t.Errorf("%v: a changed pack must be judged again: %d judgements, want 2", class, calls)
		}
		for i := 0; i < 5; i++ {
			g.r.guardWorld(&d2.Snapshot{State: d2.StateInGame, Player: s.Player, Inventory: s.Inventory})
		}
		if calls != 2 {
			t.Errorf("%v: the guard world judged the pack (%d judgements, want 2): the watch API must read the cached count", class, calls-2)
		}
		if n := g.r.sellableCached(); n != 2 {
			t.Errorf("%v: cached count %d, want 2", class, n)
		}
	}
}

func BenchmarkResInputSellable(b *testing.B) {
	g := newRig(&testing.T{})
	calls := 0
	g.r.d.Town = countTown{g.r.d.Town, &calls}
	s := &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{Class: d2.Sorceress, Level: 11}, Inventory: &d2.Inventory{}}
	for i := 0; i < 40; i++ {
		s.Inventory.Items = append(s.Inventory.Items, d2.Unit{ID: uint32(i + 1), X: i % 10, Y: i / 10, Item: &d2.Item{Where: d2.ContInventory}})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.r.resInput(s)
	}
	b.ReportMetric(float64(calls)/float64(b.N), "judgements/op")
}

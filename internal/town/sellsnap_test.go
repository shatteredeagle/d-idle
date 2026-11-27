package town_test

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town"
	"d2lod-bot/internal/town/sim"
)

// countingSource wraps the sim's snapshot so a test can count game reads.
func countingSource(tw *town.Town, w *sim.World) *int {
	n := new(int)
	tw.UseSource(town.Source{Snapshot: func() (*d2.Snapshot, error) { *n++; return w.Snapshot() }})
	return n
}

// sellsnap (goroutine dump 2026-11-27): SellableCount judged every pack item with EvaluateUnit, and on d2d each one took
// a full world snapshot behind the client mutex. The count is now judged against the snapshot it is given: no read at
// all, however many items the pack holds. Two classes.
func TestSellableCountTakesNoSnapshot(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		r := newRig(t)
		r.W.Player.Class = class
		dressed(r.W)
		for i := 0; i < 6; i++ {
			r.W.Add("cap", d2.ContInventory, 2*i%10, 2*(i/5))
		}
		s, _ := r.W.Snapshot()
		reads := countingSource(r.T, r.W)
		r.T.SellableCount(s)
		if *reads != 0 {
			t.Errorf("%v: SellableCount took %d snapshots for 6 pack items, want 0 (it has the snapshot)", class, *reads)
		}
	}
}

func BenchmarkSellableCount(b *testing.B) {
	r := sim.NewRig(b)
	tw := town.New(r.Env, r.Mv)
	tw.UseSource(town.Source{Snapshot: r.W.Snapshot, Items: r.W.AllItems, Owner: r.W.Owner, Stats: r.W.UnitStats})
	dressed(r.W)
	for i := 0; i < 10; i++ {
		r.W.Add("cap", d2.ContInventory, 2*i%10, 2*(i/5))
	}
	s, _ := r.W.Snapshot()
	reads := countingSource(tw, r.W)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tw.SellableCount(s)
	}
	b.ReportMetric(float64(*reads)/float64(b.N), "snapshots/op")
}

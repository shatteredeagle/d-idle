package invdesk

import (
	"context"
	"sync"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/status"
)

// reEq stands for the leveler: its equip check asks the desk whether the slot is pinned, as the live wiring does.
type reEq struct{ d *Desk }

func (r *reEq) CheckPlayerEquip(*d2.Snapshot, uint32, int) (string, error) {
	r.d.Pinned(d2.BodyRHand)
	r.d.MarkOf(d2.Unit{ID: 1})
	return "x", nil
}
func (r *reEq) PlayerEquip(context.Context, uint32, int) error { return nil }

func within(t *testing.T, what string, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { f(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("%s deadlocked: a desk callback reached the desk while it held its lock", what)
	}
}

// Regression for the live freeze of 2026-11-10: View held d.mu while the equip check called Pinned (same mutex).
func TestCallbacksMayCallBackIntoTheDesk(t *testing.T) {
	d, _, _ := rig(t, d2.Paladin, town1)
	d.SetEquipper(&reEq{d})
	for _, cls := range []d2.Class{d2.Paladin, d2.Sorceress} {
		d.c.Snap().Player.Class = cls
		within(t, "View", func() { d.View() })
		within(t, "RequestEquip", func() { d.RequestEquip(1, "rhand") })
		within(t, "SetMark", func() { d.SetMark(1, MarkSell) })
	}
	d.SetWhy(func(u d2.Unit) string { d.Pinned(0); d.MarkOf(u); return "w" })
	d.SetSellBlock(func(u d2.Unit) string { d.Pinned(0); return "" })
	within(t, "View with Why", func() { d.View() })
	within(t, "SetMark with SellBlock", func() { d.SetMark(1, MarkSell) })
}

func TestViewAndPinnedConcurrent(t *testing.T) {
	d, _, _ := rig(t, d2.Paladin, town1)
	d.SetEquipper(&reEq{d})
	var _ = status.Item{}
	within(t, "stress", func() {
		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					d.View()
					d.Pinned(d2.BodyRHand)
					d.SetMark(1, MarkKeep)
				}
			}()
		}
		wg.Wait()
	})
}

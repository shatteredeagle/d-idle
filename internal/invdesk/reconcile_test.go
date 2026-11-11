package invdesk

import (
	"path/filepath"
	"testing"

	"d2lod-bot/internal/d2"
)

func unitAt(id uint32, class, q, ilvl, gx, gy int, where d2.Container) d2.Unit {
	return d2.Unit{ID: id, Class: class, Item: &d2.Item{Where: where, Quality: q, ILvl: ilvl, GX: gx, GY: gy}}
}

// A game restart renumbers the items: the marks follow them by base and grid cell, the neighbours stay unmarked, an
// identified item (print changed) keeps its mark, and the new ids survive a reload of the state file.
func TestReconcileFollowsRenumberedItems(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ui.json")
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		d, err := New(Config{Path: path})
		if err != nil {
			t.Fatal(err)
		}
		d.st = state{}
		before := []d2.Unit{
			unitAt(21, 339, 1, 1, 0, 0, d2.ContInventory),
			unitAt(22, 339, 1, 1, 1, 0, d2.ContInventory),
			unitAt(23, 314, 1, 4, 2, 0, d2.ContInventory),
			unitAt(24, 339, 1, 1, 0, 0, d2.ContStash),
		}
		d.c.Snap = func() *d2.Snapshot {
			return &d2.Snapshot{Player: &d2.Player{Class: class}, Inventory: &d2.Inventory{Items: before}}
		}
		if err := d.SetMark(21, MarkSell); err != nil {
			t.Fatal(err)
		}
		if err := d.SetMark(23, MarkSell); err != nil {
			t.Fatal(err)
		}
		after := []d2.Unit{
			unitAt(5, 339, 1, 1, 0, 0, d2.ContInventory),
			unitAt(6, 339, 1, 1, 1, 0, d2.ContInventory),
			unitAt(7, 314, 4, 4, 2, 0, d2.ContInventory), // identified: quality moved, ilvl the same
			unitAt(8, 339, 1, 1, 0, 0, d2.ContStash),
		}
		if n := d.Reconcile(after); n != 2 {
			t.Fatalf("%v: reconciled %d marks, want 2", class, n)
		}
		want := map[uint32]string{5: MarkSell, 6: "", 7: MarkSell, 8: ""}
		for _, u := range after {
			if got := d.MarkOf(u); got != want[u.ID] {
				t.Errorf("%v: id %d mark %q, want %q", class, u.ID, got, want[u.ID])
			}
		}
		d2, err := New(Config{Path: path})
		if err != nil {
			t.Fatal(err)
		}
		if d2.MarkOf(after[0]) != MarkSell {
			t.Errorf("%v: the rebound mark was not saved", class)
		}
		// an unrelated item at the same cell is not marked when the original is still around under its own id
		if n := d.Reconcile(after); n != 0 {
			t.Errorf("second pass changed %d marks", n)
		}
		d.Forget(5)
		if d.MarkOf(after[0]) != "" {
			t.Error("Forget kept the mark")
		}
	}
}

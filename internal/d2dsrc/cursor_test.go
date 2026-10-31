package d2dsrc

import (
	"testing"

	"d2lod-bot/internal/d2"
)

func TestCursorFromHeldCode(t *testing.T) {
	s := &Source{}
	mk := func(held string, items ...d2.Unit) *builder {
		return &builder{s: s, rp: newRep(&s.trk), snap: &d2.Snapshot{Inventory: &d2.Inventory{Items: items}}, held: held}
	}
	u := d2.Unit{ID: 6, Class: -1, Item: &d2.Item{Where: d2.ContInventory}}
	mk("", u).cursor() // remember the pack
	b := mk("tsc")     // item 6 gone, held
	b.cursor()
	if c := b.snap.Inventory.Cursor; c == nil || c.ID != 6 || c.Item.Where != d2.ContCursor {
		t.Fatalf("cursor %+v", c)
	}
	b = mk("tsc") // still held next read
	b.cursor()
	if b.snap.Inventory.Cursor == nil {
		t.Fatal("cursor lost while held")
	}
	b = mk("", u) // put back
	b.cursor()
	if b.snap.Inventory.Cursor != nil {
		t.Fatal("cursor set after the put")
	}
}

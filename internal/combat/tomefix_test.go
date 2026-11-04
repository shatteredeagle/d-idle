package combat

import (
	"errors"
	"testing"

	"d2lod-bot/internal/d2"
)

// tomefix: offline audit of the false "portal empty" verdict (docs/tomefix.md). Each case builds the field condition
// with fakes and asserts the verdict stays "not empty" while the tome has charges.

// byID answers like d2denv.Reader.UnitStats: from a map keyed by unit id (Addr == ID on d2d), ErrUnsupported otherwise.
func byID(m map[uint32]int32) Stats {
	return func(id uint32) (d2.Stats, error) {
		q, ok := m[id]
		if !ok {
			return nil, d2.ErrUnsupported
		}
		if q > 0 {
			return d2.Stats{statQuantity: q}, nil
		}
		return d2.Stats{}, nil
	}
}

func tomeAt(id uint32, where d2.Container, x, y int) d2.Unit {
	u := pItem(int(id), itTome, id)
	u.Item.Where, u.X, u.Y, u.Item.GX, u.Item.GY = where, x, y, x, y
	return u
}

func TestTomeFixTomeWithChargesIsASource(t *testing.T) {
	stats := byID(map[uint32]int32{9: 13})
	s := packWith(tomeAt(9, d2.ContInventory, 9, 3))
	if u := PortalSource(s, pickCat(), stats); u == nil || u.ID != 9 {
		t.Fatalf("tome with 13 charges not a source: %v", u)
	}
}

// A shifted snapshot: the tome moved cell (or the pack was re-listed in another order) but the stats come from the
// previous tick. The key is the unit id, not the cell, so the count still matches.
func TestTomeFixShiftedOrStaleSnapshotKeepsCount(t *testing.T) {
	old := byID(map[uint32]int32{9: 13, 4: 2}) // previous tick's extras
	for _, s := range []*d2.Snapshot{
		packWith(tomeAt(9, d2.ContInventory, 0, 0)),
		packWith(pItem(4, itTSC, 4), tomeAt(9, d2.ContInventory, 9, 3)),
		packWith(tomeAt(9, d2.ContInventory, 9, 3), pItem(4, itTSC, 4)),
	} {
		if u := PortalSource(s, pickCat(), old); u == nil {
			t.Errorf("no source for %+v", s.Inventory.Items)
		}
	}
	if q := PortalQty(&d2.Unit{ID: 9, Addr: 9, Class: itTome}, pickCat(), old); q != 13 {
		t.Errorf("qty %d", q)
	}
}

// Unit-id collision: a ground item with the tome's id lives in Extras.Ground, never in the map UnitStats reads, so the
// stats map still holds the tome (the d2dsrc/d2denv tests pin the producer side).
func TestTomeFixGroundCollisionDoesNotChangeQty(t *testing.T) {
	stats := byID(map[uint32]int32{9: 13}) // ground id 9 is not in this map
	s := packWith(tomeAt(9, d2.ContInventory, 9, 3))
	s.Ground = []d2.Unit{{ID: 9, Addr: 9, Type: d2.TypeItem, Class: itTSC}}
	if PortalSource(s, pickCat(), stats) == nil {
		t.Fatal("ground unit with the tome's id emptied the source")
	}
}

// Item between pack and cursor: PortalQty reads the count wherever the tome is; PortalSource only takes the main
// pack, so a tome on the cursor reads "no source" for that look. That is by design (the cast clicks the pack cell) and
// the verdict debounce (bot/tpdebounce.go) carries a short excursion. Pinned so a change shows up here.
func TestTomeFixCursorTomeKeepsQtyButIsNotPackSource(t *testing.T) {
	stats := byID(map[uint32]int32{9: 13})
	cur := tomeAt(9, d2.ContCursor, 0, 0)
	if q := PortalQty(&cur, pickCat(), stats); q != 13 {
		t.Errorf("cursor tome qty %d, want 13", q)
	}
	if PortalSource(packWith(cur), pickCat(), stats) != nil {
		t.Error("cursor tome taken as a pack source")
	}
	back := tomeAt(9, d2.ContInventory, 9, 3)
	if PortalSource(packWith(back), pickCat(), stats) == nil {
		t.Error("tome back in the pack is not a source")
	}
}

// Panel open or closed does not enter the verdict: the snapshot's UI flags are not read by either function.
func TestTomeFixPanelStateIsIrrelevant(t *testing.T) {
	stats := byID(map[uint32]int32{9: 13})
	for _, open := range []uint32{0, 1} {
		s := packWith(tomeAt(9, d2.ContInventory, 9, 3))
		s.UI[d2.UIInventory] = open
		if PortalSource(s, pickCat(), stats) == nil {
			t.Errorf("panel %d: no source", open)
		}
	}
}

// The dropout itself: the unit id is absent from the stats map (extras lost) or its stat list errors. This is the
// ONLY way PortalQty returns 0 for a pack tome; the test fixes that contract so the fix must stay on the producer side.
func TestTomeFixOnlyMissingExtrasReadEmpty(t *testing.T) {
	cat := pickCat()
	tome := tomeAt(9, d2.ContInventory, 9, 3)
	if PortalQty(&tome, cat, byID(map[uint32]int32{})) != 0 {
		t.Error("absent extras should read empty (contract)")
	}
	if PortalQty(&tome, cat, func(uint32) (d2.Stats, error) { return nil, errors.New("x") }) != 0 {
		t.Error("stat error should read empty (contract)")
	}
	if PortalQty(&tome, cat, nil) != 1 {
		t.Error("no stats source must count as usable")
	}
}

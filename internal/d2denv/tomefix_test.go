package d2denv

import (
	"testing"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dsrc"
)

// tomefix (docs/tomefix.md): the Reader half of the tome verdict. UnitStats answers from the LAST snapshot's
// Extras.Items by unit id; these cases pin what the field can do to that map.

type tomeCat struct{}

func (tomeCat) Item(class int) combat.ItemClass {
	return combat.ItemClass{Portal: true, Tome: class == 530}
}

func tomeResult(items map[uint32]d2dsrc.ItemExtra, ground map[uint32]d2dsrc.ItemExtra, inv ...d2.Unit) *d2dsrc.Result {
	return &d2dsrc.Result{
		Snap:   &d2.Snapshot{Inventory: &d2.Inventory{Items: inv}},
		Extras: d2dsrc.Extras{Items: items, Ground: ground},
	}
}

func packTome(id uint32, where d2.Container) d2.Unit {
	return d2.Unit{ID: id, Addr: id, Type: d2.TypeItem, Class: 530, Item: &d2.Item{Where: where}}
}

func TestTomeFixReaderKeepsTomeAcrossGroundCollision(t *testing.T) {
	r := NewReader(nil)
	res := tomeResult(
		map[uint32]d2dsrc.ItemExtra{9: {Code: "tbk", Qty: 13, Sockets: -1}},
		map[uint32]d2dsrc.ItemExtra{9: {Code: "gld", Sockets: -1}}, // ground id 9 as well
		packTome(9, d2.ContInventory))
	r.last = res
	if combat.PortalSource(res.Snap, tomeCat{}, r.UnitStats) == nil {
		t.Fatal("ground id collision emptied the tome")
	}
}

// A newer look replaces `last`; a caller still holding the older snapshot asks by the same id and gets the new
// count. A tome that spends a charge between the two looks reads the lower number, never 0 while charges remain.
func TestTomeFixReaderStaleSnapshotReadsCurrentCount(t *testing.T) {
	r := NewReader(nil)
	old := tomeResult(map[uint32]d2dsrc.ItemExtra{9: {Code: "tbk", Qty: 13, Sockets: -1}}, nil, packTome(9, d2.ContInventory))
	r.last = old
	r.last = tomeResult(map[uint32]d2dsrc.ItemExtra{9: {Code: "tbk", Qty: 12, Sockets: -1}}, nil, packTome(9, d2.ContInventory))
	if q := combat.PortalQty(&old.Snap.Inventory.Items[0], tomeCat{}, r.UnitStats); q != 12 {
		t.Fatalf("qty %d, want 12", q)
	}
}

// The tome on the cursor keeps its extras (dump lists it with loc cursor), so the count is readable; it is only not
// a pack source for that look.
func TestTomeFixReaderCursorTomeKeepsExtras(t *testing.T) {
	r := NewReader(nil)
	cur := packTome(9, d2.ContCursor)
	res := tomeResult(map[uint32]d2dsrc.ItemExtra{9: {Code: "tbk", Qty: 13, Sockets: -1}}, nil, cur)
	r.last = res
	if q := combat.PortalQty(&cur, tomeCat{}, r.UnitStats); q != 13 {
		t.Fatalf("cursor qty %d", q)
	}
}

// Contract of the remaining producer-side gaps (U live): a stock "items" look (no dump) or a record without qty reads
// as an empty tome by design, because a real empty tome has no qty either.
func TestTomeFixReaderNoQtyRecordReadsEmpty(t *testing.T) {
	r := NewReader(nil)
	res := tomeResult(map[uint32]d2dsrc.ItemExtra{9: {Code: "tbk", Sockets: -1}}, nil, packTome(9, d2.ContInventory))
	r.last = res
	if combat.PortalSource(res.Snap, tomeCat{}, r.UnitStats) != nil {
		t.Fatal("a record without qty must read empty (indistinguishable from a real empty tome)")
	}
}

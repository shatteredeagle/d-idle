package panel_test

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town/panel"
	"d2lod-bot/internal/town/sim"
)

type (
	Pt = panel.Pt
)

func TestLayoutMatchesSecondSet(t *testing.T) {
	db := sim.RealDB(t)
	l, err := panel.NewLayout(db, d2.Sorceress, Pt{X: 800, Y: 600}, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		g   panel.Grid
		row string
	}{{l.Inventory, "Sorceress2"}, {l.Stash, "Big Bank Page2"}, {l.Cube, "Transmogrify Box2"}, {l.Vendor, "Monster2"}} {
		v := db.InventoryLayout(c.row)
		if v == nil || v.GridRect.Left != c.g.Left || v.GridRect.Top != c.g.Top || v.Cols != c.g.Cols || v.Rows != c.g.Rows {
			t.Errorf("%s: layout %+v vs data %+v", c.g.Name, c.g, v)
		}
	}
	if p := l.Inventory.Cell(0, 0); p != (Pt{X: 419 + 14, Y: 315 + 14}) {
		t.Errorf("cell %v", p)
	}
	if col, row, ok := l.Inventory.CellAt(l.Inventory.ItemCenter(3, 2, 1, 3)); !ok || col != 3 || row != 3 {
		t.Errorf("CellAt %d %d %v", col, row, ok)
	}
	if len(l.Slots) < 8 {
		t.Errorf("slots %v", l.Slots)
	}
}

func TestOccupancy(t *testing.T) {
	o := panel.NewOccupancy(10, 4)
	o.Place(0, 0, 1, 3)
	if c, r, ok := o.FindSpot(2, 2); !ok || c != 1 || r != 0 {
		t.Fatalf("spot %d,%d %v", c, r, ok)
	}
	if o.Fits(9, 3, 2, 1) || !o.Fits(9, 3, 1, 1) || o.Fits(0, 2, 1, 1) {
		t.Fatal("Fits")
	}
	if o.FreeCells() != 37 || o.UsedPct() != 7 {
		t.Fatalf("free %d used %d", o.FreeCells(), o.UsedPct())
	}
	o.Remove(0, 0, 1, 3)
	if o.FreeCells() != 40 {
		t.Fatal("Remove")
	}
	full := panel.NewOccupancy(2, 2)
	full.Place(0, 0, 2, 2)
	if _, _, ok := full.FindSpot(1, 1); ok {
		t.Fatal("full grid")
	}
}

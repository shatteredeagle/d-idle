// Package panel holds the screen geometry of the item grids (backpack, stash, cube, vendor, belt) and a
// free-space model of a grid. It is pure: no memory, no input. Shared by internal/town and internal/loot.
//
// Geometry comes from inventory.txt via internal/data. The plain rows are the 640x480 layout; the game
// centres that layout in a bigger window, so for 800x600 every rectangle shifts by (80,60). The data's
// own "...2" rows (the 800x600 set) agree with plain+offset (checked by TestLayoutMatchesSecondSet on the
// real tables). Whether the running client really draws at those pixels is tagged in docs/town.md.
package panel

import (
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
)

// Pt is a client-area pixel.
type Pt struct{ X, Y int }

// Grid is one rectangular item grid on screen.
type Grid struct {
	Name         string
	Cols, Rows   int
	Left, Top    int // pixel of the top-left corner of cell (0,0)
	CellW, CellH int
}

// Cell returns the pixel centre of cell (col,row).
func (g Grid) Cell(col, row int) Pt {
	return Pt{g.Left + col*g.CellW + g.CellW/2, g.Top + row*g.CellH + g.CellH/2}
}

// ItemCenter returns the pixel centre of an item whose top-left cell is (col,row) and which spans w x h cells.
func (g Grid) ItemCenter(col, row, w, h int) Pt {
	return Pt{g.Left + col*g.CellW + w*g.CellW/2, g.Top + row*g.CellH + h*g.CellH/2}
}

// CellAt returns the cell under pixel p, or ok=false outside the grid.
func (g Grid) CellAt(p Pt) (col, row int, ok bool) {
	if p.X < g.Left || p.Y < g.Top {
		return 0, 0, false
	}
	col, row = (p.X-g.Left)/g.CellW, (p.Y-g.Top)/g.CellH
	return col, row, col < g.Cols && row < g.Rows
}

// Layout is every grid the town code clicks, for one character class and window size.
type Layout struct {
	Screen    Pt
	Inventory Grid
	Stash     Grid
	Cube      Grid
	Vendor    Grid // the NPC trade panel (inventory.txt "Monster")
	// Panels (outer rectangles) for hit tests: a click must land inside to count as "on the panel".
	InvPanel, LeftPanel Rect
	// Offset is the shift applied to the 640x480 base layout.
	Offset Pt
	// Slots are the equipment slot rectangles keyed by inventory.txt names (head, torso, rHand, lHand, neck,
	// belt, feet, gloves, rArm, lArm), already offset to screen pixels.
	Slots map[string]Rect
}

// Center of a rectangle.
func (r Rect) Center() Pt { return Pt{(r.L + r.R) / 2, (r.T + r.B) / 2} }

// Rect is a pixel rectangle, right/bottom exclusive.
type Rect struct{ L, T, R, B int }

func (r Rect) Contains(p Pt) bool { return p.X >= r.L && p.X < r.R && p.Y >= r.T && p.Y < r.B }

// Class names used by inventory.txt for the character rows.
func classRow(c d2.Class) string {
	switch c {
	case d2.Amazon:
		return "Amazon"
	case d2.Sorceress:
		return "Sorceress"
	case d2.Necromancer:
		return "Necromancer"
	case d2.Paladin:
		return "Paladin"
	case d2.Barbarian:
		return "Barbarian"
	case d2.Druid:
		return "Druid"
	case d2.Assassin:
		return "Assassin"
	}
	return "Sorceress"
}

// NewLayout builds the layout from the data tables for a window of the given size.
// bigStash selects the 6x8 LoD stash ("Big Bank Page 1"); false the classic 6x4. LoD uses true.
func NewLayout(db *data.DB, class d2.Class, screen Pt, bigStash bool) (*Layout, error) {
	off := Pt{(screen.X - 640) / 2, (screen.Y - 480) / 2}
	grid := func(name, row string) (Grid, data.Rect, error) {
		v := db.InventoryLayout(row)
		if v == nil || v.Cols <= 0 || v.CellW <= 0 {
			return Grid{}, data.Rect{}, errMissing(row)
		}
		return Grid{Name: name, Cols: v.Cols, Rows: v.Rows, Left: v.GridRect.Left + off.X, Top: v.GridRect.Top + off.Y,
			CellW: v.CellW, CellH: v.CellH}, v.Panel, nil
	}
	var l Layout
	var err error
	var ip, lp data.Rect
	if l.Inventory, ip, err = grid("inventory", classRow(class)); err != nil {
		return nil, err
	}
	stash := "Bank Page 1"
	if bigStash {
		stash = "Big Bank Page 1"
	}
	if l.Stash, lp, err = grid("stash", stash); err != nil {
		return nil, err
	}
	if l.Cube, _, err = grid("cube", "Transmogrify Box Page 1"); err != nil {
		return nil, err
	}
	if l.Vendor, _, err = grid("vendor", "Monster"); err != nil {
		return nil, err
	}
	// The stash and cube panels share the left half; their data rows only describe the lower part, so the
	// hit-test rectangle is the whole left panel (width 321, full height of the inventory panel).
	l.InvPanel = Rect{ip.Left + off.X, ip.Top + off.Y, ip.Right + off.X, ip.Bottom + off.Y}
	l.LeftPanel = Rect{lp.Left + off.X, ip.Top + off.Y, lp.Right + off.X, ip.Bottom + off.Y}
	l.Screen, l.Offset = screen, off
	l.Slots = map[string]Rect{}
	if v := db.InventoryLayout(classRow(class)); v != nil {
		for n, r := range v.Slots {
			l.Slots[n] = Rect{r.Left + off.X, r.Top + off.Y, r.Right + off.X, r.Bottom + off.Y}
		}
	}
	return &l, nil
}

type missingErr string

func (e missingErr) Error() string { return "panel: no inventory.txt row " + string(e) }
func errMissing(row string) error  { return missingErr(row) }

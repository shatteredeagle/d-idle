package data

import "strings"

// Rect is a pixel rectangle as inventory.txt / belts.txt give it (in the
// 640x480 UI coordinate space the game scales from).
type Rect struct{ Left, Right, Top, Bottom, Width, Height int }

// Inventory is one inventory.txt row: the layout of a container panel. The
// row names are "Amazon".."Assassin" (a player's inventory + equipment),
// "Bank Page 1" (stash), "Big Bank Page 1" (expansion stash), "Transmogrify
// Box Page 1" (Horadric Cube), "Trade Page 1", "Hireling", "Monster", ...
// The "...2" rows (e.g. "Amazon2") are the same panels for the other UI
// resolution mode.
type Inventory struct {
	Class string
	Panel Rect // invLeft..invBottom
	// Grid: Cols x Rows cells (gridX, gridY) inside GridRect; each cell is
	// CellW x CellH pixels.
	Cols, Rows   int
	GridRect     Rect
	CellW, CellH int
	// Slots are the equipment slot boxes keyed by: rArm torso lArm head neck
	// rHand lHand belt feet gloves.
	Slots map[string]Rect
}

var slotNames = []string{"rArm", "torso", "lArm", "head", "neck", "rHand", "lHand", "belt", "feet", "gloves"}

var inventoryCols = func() []string {
	c := []string{"class", "invLeft", "invRight", "invTop", "invBottom", "gridX", "gridY", "gridLeft", "gridRight", "gridTop", "gridBottom", "gridBoxWidth", "gridBoxHeight"}
	for _, s := range slotNames {
		c = append(c, s+"Left", s+"Right", s+"Top", s+"Bottom", s+"Width", s+"Height")
	}
	return c
}()

func rectOf(r Row, p string) Rect {
	return Rect{r.Int(p + "Left"), r.Int(p + "Right"), r.Int(p + "Top"), r.Int(p + "Bottom"), r.Int(p + "Width"), r.Int(p + "Height")}
}

func loadInventory(t *Table) []*Inventory {
	var out []*Inventory
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		name := r.Str("class")
		if name == "" || name == "Expansion" {
			continue
		}
		inv := &Inventory{
			Class: name, Cols: r.Int("gridX"), Rows: r.Int("gridY"), CellW: r.Int("gridBoxWidth"), CellH: r.Int("gridBoxHeight"),
			Panel:    Rect{r.Int("invLeft"), r.Int("invRight"), r.Int("invTop"), r.Int("invBottom"), 0, 0},
			GridRect: Rect{r.Int("gridLeft"), r.Int("gridRight"), r.Int("gridTop"), r.Int("gridBottom"), 0, 0},
			Slots:    map[string]Rect{},
		}
		inv.Panel.Width, inv.Panel.Height = inv.Panel.Right-inv.Panel.Left, inv.Panel.Bottom-inv.Panel.Top
		inv.GridRect.Width, inv.GridRect.Height = inv.GridRect.Right-inv.GridRect.Left, inv.GridRect.Bottom-inv.GridRect.Top
		for _, s := range slotNames {
			// -1 marks "no such slot" (stash, cube, ...); 0 for all is blank.
			if rc := rectOf(r, s); rc != (Rect{}) && rc.Left >= 0 {
				inv.Slots[s] = rc
			}
		}
		out = append(out, inv)
	}
	return out
}

// CellCenter returns the pixel centre of grid cell (col,row).
func (v *Inventory) CellCenter(col, row int) (x, y int) {
	return v.GridRect.Left + col*v.CellW + v.CellW/2, v.GridRect.Top + row*v.CellH + v.CellH/2
}

// Belt is one belts.txt row: a belt type's potion slot boxes. Names are the
// item belt classes: "default" (no belt), "sash" (8), "belt" (12), ...
type Belt struct {
	Name  string
	Slots int
	BoxW  int
	BoxH  int
	Boxes []Rect // numboxes entries; Width/Height filled from BoxW/BoxH
}

var beltCols = func() []string {
	c := []string{"name", "numboxes", "boxwidth", "boxheight"}
	for i := 1; i <= 16; i++ {
		n := "box" + itoa(i)
		c = append(c, n+"left", n+"right", n+"top", n+"bottom")
	}
	return c
}()

func loadBelts(t *Table) []*Belt {
	var out []*Belt
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		name := r.Str("name")
		if name == "" {
			continue
		}
		b := &Belt{Name: strings.ToLower(name), Slots: r.Int("numboxes"), BoxW: r.Int("boxwidth"), BoxH: r.Int("boxheight")}
		for k := 1; k <= b.Slots && k <= 16; k++ {
			n := "box" + itoa(k)
			b.Boxes = append(b.Boxes, Rect{r.Int(n + "left"), r.Int(n + "right"), r.Int(n + "top"), r.Int(n + "bottom"), b.BoxW, b.BoxH})
		}
		out = append(out, b)
	}
	return out
}

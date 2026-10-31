package leveler

// Screen geometry of the stat panel and the skill tree at 800x600, measured on a game-window screenshot of the
// Sorceress (docs/leveler.md "Geometry"). The panels are anchored to the screen edges; other window sizes are UNVERIFIED.

// Pt is a client-area pixel.
type Pt struct{ X, Y int }

// Geometry holds the click points.
type Geometry struct {
	StatButton [numAttr]Pt // the red + beside Strength, Dexterity, Vitality, Energy (character panel, key C)
	TabX       int         // x of the three skill tab buttons (skill tree, key T)
	TabY       [3]int      // y of the tab for skilldesc SkillPage 1, 2, 3 (the page order runs bottom to top)
	IconX0     int         // icon centre of column 1
	IconDX     int
	IconY0     int // icon centre of row 1
	IconDY     int
}

// DefaultGeometry is the 800x600 layout.
func DefaultGeometry() Geometry {
	return Geometry{
		StatButton: [numAttr]Pt{{220, 145}, {220, 210}, {220, 294}, {220, 354}},
		TabX:       680,
		TabY:       [3]int{420, 320, 218}, // page 1 = Fire (bottom tab), 2 = Lightning, 3 = Cold (top)
		IconX0:     437, IconDX: 70, IconY0: 95, IconDY: 67,
	}
}

// Tab returns the click point of tab page 1..3.
func (g Geometry) Tab(page int) Pt { return Pt{g.TabX, g.TabY[page-1]} }

// Icon returns the centre of the skill icon at row 1..6, column 1..3.
func (g Geometry) Icon(row, col int) Pt {
	return Pt{g.IconX0 + (col-1)*g.IconDX, g.IconY0 + (row-1)*g.IconDY}
}

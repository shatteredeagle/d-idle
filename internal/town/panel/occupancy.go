package panel

// Occupancy tracks which cells of a grid are taken. It is rebuilt from memory before each decision;
// nothing here is authoritative about the game.
type Occupancy struct {
	Cols, Rows int
	taken      []bool
}

func NewOccupancy(cols, rows int) *Occupancy {
	return &Occupancy{Cols: cols, Rows: rows, taken: make([]bool, cols*rows)}
}

func (o *Occupancy) in(c, r int) bool { return c >= 0 && r >= 0 && c < o.Cols && r < o.Rows }

// Fits reports whether a w x h item with its top-left at (c,r) is inside the grid and all cells are free.
func (o *Occupancy) Fits(c, r, w, h int) bool {
	if w <= 0 || h <= 0 || !o.in(c, r) || !o.in(c+w-1, r+h-1) {
		return false
	}
	for y := r; y < r+h; y++ {
		for x := c; x < c+w; x++ {
			if o.taken[y*o.Cols+x] {
				return false
			}
		}
	}
	return true
}

// Place marks the cells taken (cells outside the grid are ignored).
func (o *Occupancy) Place(c, r, w, h int) { o.set(c, r, w, h, true) }

// Remove frees the cells.
func (o *Occupancy) Remove(c, r, w, h int) { o.set(c, r, w, h, false) }

func (o *Occupancy) set(c, r, w, h int, v bool) {
	for y := r; y < r+h; y++ {
		for x := c; x < c+w; x++ {
			if o.in(x, y) {
				o.taken[y*o.Cols+x] = v
			}
		}
	}
}

// FindSpot returns the first free top-left position for a w x h item, scanning row by row (the game's
// own auto-placement scans column by column inside rows; either order is a valid placement).
func (o *Occupancy) FindSpot(w, h int) (c, r int, ok bool) {
	for r = 0; r+h <= o.Rows; r++ {
		for c = 0; c+w <= o.Cols; c++ {
			if o.Fits(c, r, w, h) {
				return c, r, true
			}
		}
	}
	return 0, 0, false
}

// FreeCells counts free cells.
func (o *Occupancy) FreeCells() int {
	n := 0
	for _, t := range o.taken {
		if !t {
			n++
		}
	}
	return n
}

// UsedPct is the percentage of cells taken (0..100).
func (o *Occupancy) UsedPct() int {
	if len(o.taken) == 0 {
		return 100
	}
	return (len(o.taken) - o.FreeCells()) * 100 / len(o.taken)
}

// Clone copies the occupancy.
func (o *Occupancy) Clone() *Occupancy {
	c := *o
	c.taken = append([]bool(nil), o.taken...)
	return &c
}

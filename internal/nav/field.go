package nav

import (
	"math"

	"d2lod-bot/internal/d2map"
)

// Cell flag bits of a Field.
const (
	fWalk  uint8 = 1 // known and BlockWalk clear
	fKnown uint8 = 2 // some loaded room covered the cell
	fOpt   uint8 = 4 // unknown, but inside the level rectangle: optimistic planning may cross it
)

// costUnit is the fixed-point scale of wall and dynamic penalties (1/8 of a plain step).
const costUnit = 8

// Field is the planner's dense copy of a window of a d2map.Grid: walkability, clearance from known walls and a
// per-cell cost overlay for dynamic things. Unknown cells are neither walkable nor walls.
type Field struct {
	X0, Y0, W, H int
	flags        []uint8
	clr          []uint8 // distance to the nearest known wall in half-cells, capped at 2*clearCap
	wall         []uint8 // wall penalty, costUnit per extra step
	dyn          []uint8 // dynamic penalty, costUnit per extra step (monsters); cleared by ClearDyn
	touched      []int32
	cut          []cutCell // cells AddBlock took out of the walkable set; ClearDyn puts them back
	grid         *d2map.Grid
	version      uint64
	clearCap     int
	wallW        float64
	passMask     uint8 // cells with any of these bits are passable (fWalk, plus fOpt when planning optimistically)
}

const clearCapMax = 8

// maxFieldCells bounds one planner window (the Grid cap; real levels are under 2000 on a side, S). A
// torn Level struct can pass the reader's per-side check with a rectangle of hundreds of millions of cells.
const maxFieldCells = 4096 * 4096 // the d2map.Grid cap, so any known area fits

func (f *Field) idx(x, y int) int {
	x -= f.X0
	y -= f.Y0
	if x < 0 || y < 0 || x >= f.W || y >= f.H {
		return -1
	}
	return y*f.W + x
}

func (f *Field) pt(i int32) (int, int) { return int(i)%f.W + f.X0, int(i)/f.W + f.Y0 }

// Walkable reports whether the cell is known and not blocked.
func (f *Field) Walkable(x, y int) bool {
	i := f.idx(x, y)
	return i >= 0 && f.flags[i]&fWalk != 0
}

// Known reports whether the cell was covered by a loaded room.
func (f *Field) Known(x, y int) bool {
	i := f.idx(x, y)
	return i >= 0 && f.flags[i]&fKnown != 0
}

// Clearance is the distance in cells from (x,y) to the nearest known wall (capped).
func (f *Field) Clearance(x, y int) float64 {
	i := f.idx(x, y)
	if i < 0 {
		return 0
	}
	return float64(f.clr[i]) / 2
}

// unknownCost is the extra cost of crossing an unknown cell when planning optimistically.
const unknownCost = 4 // costUnit/8 = 0.5 per step

// buildField copies the grid into a new Field. The window is the grid's known area united with level (the level
// rectangle, may be empty); unknown cells inside level are marked fOpt.
func buildField(g *d2map.Grid, level d2map.Rect, clearance, wallWeight float64) *Field {
	b := g.Bounds()
	r := b
	switch {
	case b.Empty():
		r = level
	case !level.Empty():
		x0, y0 := min(b.X, level.X), min(b.Y, level.Y)
		x1, y1 := max(b.Right(), level.Right()), max(b.Bottom(), level.Bottom())
		r = d2map.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
	}
	r, level = clampWindow(r, b, level)
	f := &Field{grid: g, version: g.Version(), clearCap: min(clearCapMax, max(0, int(math.Ceil(clearance)))), wallW: wallWeight, passMask: fWalk}
	if r.Empty() {
		return f
	}
	f.X0, f.Y0, f.W, f.H = r.X, r.Y, r.W, r.H
	n := f.W * f.H
	f.flags = make([]uint8, n)
	f.clr = make([]uint8, n)
	f.wall = make([]uint8, n)
	f.dyn = make([]uint8, n)
	var objects []int32
	for y := 0; y < f.H; y++ {
		row := y * f.W
		for x := 0; x < f.W; x++ {
			fl, ok := g.Flags(f.X0+x, f.Y0+y)
			if !ok {
				if level.Contains(f.X0+x, f.Y0+y) {
					f.flags[row+x] = fOpt
				}
				continue
			}
			v := fKnown
			if fl&d2map.BlockWalk == 0 {
				v |= fWalk
				if fl&d2map.Object != 0 {
					objects = append(objects, int32(row+x))
				}
			}
			f.flags[row+x] = v
		}
	}
	f.computeClearance()
	// An object's centre cell reads walkable but its footprint is solid (docs/map.md): make those cells costly.
	for _, i := range objects {
		f.wall[i] = max(f.wall[i], 16)
	}
	return f
}

// clampWindow keeps the planner window within maxFieldCells. An absurd level rectangle is cut to a margin around the
// known area (optimistic planning keeps working near it) or dropped; with nothing known the window is empty.
func clampWindow(r, known, level d2map.Rect) (d2map.Rect, d2map.Rect) {
	if int64(r.W)*int64(r.H) <= maxFieldCells {
		return r, level
	}
	const margin = 400
	if known.Empty() {
		return d2map.Rect{}, d2map.Rect{}
	}
	near := d2map.Rect{X: known.X - margin, Y: known.Y - margin, W: known.W + 2*margin, H: known.H + 2*margin}
	x0, y0 := max(near.X, level.X), max(near.Y, level.Y)
	x1, y1 := min(near.Right(), level.Right()), min(near.Bottom(), level.Bottom())
	if x1 > x0 && y1 > y0 {
		level = d2map.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
		x0, y0 = min(known.X, level.X), min(known.Y, level.Y)
		x1, y1 = max(known.Right(), level.Right()), max(known.Bottom(), level.Bottom())
		if w := (d2map.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}); int64(w.W)*int64(w.H) <= maxFieldCells {
			return w, level
		}
	}
	return known, d2map.Rect{}
}

// computeClearance runs a two-pass chamfer distance transform (orthogonal 2, diagonal 3, so half-cells) from the
// known blocked cells and turns it into the wall penalty.
func (f *Field) computeClearance() {
	const inf = uint8(250)
	W, H := f.W, f.H
	capv := uint8(2 * f.clearCap)
	for i, v := range f.flags {
		if v&fKnown != 0 && v&fWalk == 0 {
			f.clr[i] = 0
		} else {
			f.clr[i] = inf
		}
	}
	upd := func(i, j int, c uint8) {
		if f.clr[j] < inf {
			if v := f.clr[j] + c; v < f.clr[i] {
				f.clr[i] = v
			}
		}
	}
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			i := y*W + x
			if f.clr[i] == 0 {
				continue
			}
			if x > 0 {
				upd(i, i-1, 2)
			}
			if y > 0 {
				upd(i, i-W, 2)
				if x > 0 {
					upd(i, i-W-1, 3)
				}
				if x < W-1 {
					upd(i, i-W+1, 3)
				}
			}
		}
	}
	for y := H - 1; y >= 0; y-- {
		for x := W - 1; x >= 0; x-- {
			i := y*W + x
			if f.clr[i] == 0 {
				continue
			}
			if x < W-1 {
				upd(i, i+1, 2)
			}
			if y < H-1 {
				upd(i, i+W, 2)
				if x < W-1 {
					upd(i, i+W+1, 3)
				}
				if x > 0 {
					upd(i, i+W-1, 3)
				}
			}
		}
	}
	for i, c := range f.clr {
		if c > capv {
			c = capv
			f.clr[i] = c
		}
		// penalty for clearance below clearCap cells; cells ON a wall edge (clearance 1) pay the most
		if miss := float64(f.clearCap) - float64(c)/2; miss > 0 && f.flags[i]&fWalk != 0 {
			f.wall[i] = uint8(math.Min(250, math.Round(miss*f.wallW*costUnit)))
		}
		if f.flags[i]&fOpt != 0 {
			f.wall[i] = unknownCost
		}
	}
}

// AddDisc adds a penalty that fades linearly from peak at (cx,cy) to zero at radius r (subtiles).
func (f *Field) AddDisc(cx, cy int, r, peak float64) {
	if r <= 0 || peak <= 0 {
		return
	}
	ri := int(math.Ceil(r))
	for y := cy - ri; y <= cy+ri; y++ {
		for x := cx - ri; x <= cx+ri; x++ {
			i := f.idx(x, y)
			if i < 0 || f.flags[i]&fWalk == 0 {
				continue
			}
			d := math.Hypot(float64(x-cx), float64(y-cy))
			if d >= r {
				continue
			}
			add := peak * (1 - d/r) * costUnit
			if v := float64(f.dyn[i]) + add; v > 250 {
				f.dyn[i] = 250
			} else {
				if f.dyn[i] == 0 {
					f.touched = append(f.touched, int32(i))
				}
				f.dyn[i] = uint8(v)
			}
		}
	}
}

type cutCell struct {
	i  int32
	fl uint8
}

// AddBlock makes the walkable cells within r of (cx,cy) impassable until the next ClearDyn: a hard version of
// AddDisc for cells where the character was seen not to get through although the map says it can.
func (f *Field) AddBlock(cx, cy int, r float64) {
	ri := int(math.Ceil(r))
	for y := cy - ri; y <= cy+ri; y++ {
		for x := cx - ri; x <= cx+ri; x++ {
			i := f.idx(x, y)
			if i < 0 || f.flags[i]&fWalk == 0 || math.Hypot(float64(x-cx), float64(y-cy)) > r {
				continue
			}
			f.cut = append(f.cut, cutCell{int32(i), f.flags[i]})
			f.flags[i] &^= fWalk | fOpt
		}
	}
}

// ClearDyn removes every dynamic penalty and hard block.
func (f *Field) ClearDyn() {
	for k := len(f.cut) - 1; k >= 0; k-- {
		f.flags[f.cut[k].i] = f.cut[k].fl
	}
	f.cut = f.cut[:0]
	for _, i := range f.touched {
		f.dyn[i] = 0
	}
	f.touched = f.touched[:0]
}

// nearestWalkable finds the walkable cell closest to (x,y) within radius rings; ok is false when none exists.
func (f *Field) nearestWalkable(x, y, radius int) (int, int, bool) {
	if f.Walkable(x, y) {
		return x, y, true
	}
	bestD, bx, by := math.MaxFloat64, 0, 0
	for r := 1; r <= radius; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if max(abs(dx), abs(dy)) != r || !f.Walkable(x+dx, y+dy) {
					continue
				}
				if d := float64(dx*dx + dy*dy); d < bestD {
					bestD, bx, by = d, x+dx, y+dy
				}
			}
		}
		if bestD < math.MaxFloat64 && float64(r*r) >= bestD {
			return bx, by, true
		}
	}
	return bx, by, bestD < math.MaxFloat64
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

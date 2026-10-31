package d2map

import "strings"

// Collision flag bits of one subtile cell (WORD). Bit positions: D2BS ActMap.h / Kolbot sdk.js.
// Static terrain bits are 0x1 (BlockWalk), 0x2, 0x4, 0x8, 0x10, 0x20; the higher bits describe
// dynamic things standing on the cell. See docs/map.md for what was seen live.
const (
	BlockWalk     uint16 = 0x0001 // Kolbot's walkability rule: cell is unwalkable when set
	BlockLOS      uint16 = 0x0002
	Wall          uint16 = 0x0004
	BlockPlayer   uint16 = 0x0008
	AlternateTile uint16 = 0x0010
	Blank         uint16 = 0x0020
	Missile       uint16 = 0x0040
	Player        uint16 = 0x0080
	NPCLocation   uint16 = 0x0100
	Item          uint16 = 0x0200
	Object        uint16 = 0x0400
	ClosedDoor    uint16 = 0x0800
	NPCCollision  uint16 = 0x1000
	FriendlyNPC   uint16 = 0x2000
	Unknown4000   uint16 = 0x4000
	DeadBody      uint16 = 0x8000 // also used by portals
)

// DynamicMask covers the bits that describe things currently standing on a cell (players,
// monsters, items, missiles, corpses) rather than terrain.
const DynamicMask = Missile | Player | NPCLocation | Item | NPCCollision | FriendlyNPC | DeadBody

// Grid is a merged collision grid keyed by absolute subtile coordinates. Cells that no loaded
// room has covered are unknown (Known reports false, Walkable reports false).
// A Grid is not safe for concurrent use; Tracker hands out snapshots via Clone.
type Grid struct {
	x0, y0, w, h int
	cells        []uint16
	known        []bool
	nknown       int
	version      uint64
}

func NewGrid() *Grid { return &Grid{} }

// Version increments on every change that alters a cell, so callers can cache derived data.
func (g *Grid) Version() uint64 { return g.version }

// KnownCount is the number of known cells.
func (g *Grid) KnownCount() int { return g.nknown }

// Bounds is the bounding rectangle of the storage (every known cell is inside it). Empty if
// nothing is known.
func (g *Grid) Bounds() Rect {
	if g.nknown == 0 {
		return Rect{}
	}
	return Rect{g.x0, g.y0, g.w, g.h}
}

func (g *Grid) idx(x, y int) int {
	if x < g.x0 || y < g.y0 || x >= g.x0+g.w || y >= g.y0+g.h {
		return -1
	}
	return (y-g.y0)*g.w + (x - g.x0)
}

// Known reports whether any loaded room covered (x,y).
func (g *Grid) Known(x, y int) bool {
	i := g.idx(x, y)
	return i >= 0 && g.known[i]
}

// Flags returns the collision word at (x,y); ok is false for unknown cells.
func (g *Grid) Flags(x, y int) (flags uint16, ok bool) {
	i := g.idx(x, y)
	if i < 0 || !g.known[i] {
		return 0, false
	}
	return g.cells[i], true
}

// Walkable is the Kolbot rule: known and BlockWalk clear.
func (g *Grid) Walkable(x, y int) bool {
	f, ok := g.Flags(x, y)
	return ok && f&BlockWalk == 0
}

// grow enlarges storage to include r.
func (g *Grid) grow(r Rect) {
	if r.Empty() {
		return
	}
	if g.w == 0 {
		g.x0, g.y0, g.w, g.h = r.X, r.Y, r.W, r.H
		g.cells = make([]uint16, r.W*r.H)
		g.known = make([]bool, r.W*r.H)
		return
	}
	nx0, ny0 := min(g.x0, r.X), min(g.y0, r.Y)
	nx1, ny1 := max(g.x0+g.w, r.Right()), max(g.y0+g.h, r.Bottom())
	if nx0 == g.x0 && ny0 == g.y0 && nx1 == g.x0+g.w && ny1 == g.y0+g.h {
		return
	}
	nw, nh := nx1-nx0, ny1-ny0
	cells := make([]uint16, nw*nh)
	known := make([]bool, nw*nh)
	for y := 0; y < g.h; y++ {
		o := (g.y0+y-ny0)*nw + (g.x0 - nx0)
		copy(cells[o:o+g.w], g.cells[y*g.w:(y+1)*g.w])
		copy(known[o:o+g.w], g.known[y*g.w:(y+1)*g.w])
	}
	g.x0, g.y0, g.w, g.h, g.cells, g.known = nx0, ny0, nw, nh, cells, known
}

// maxGridCells bounds one level grid (4096 x 4096 subtiles; real levels are under 2000 on a side). A torn read of a
// freed room can give a coherent but absurd rectangle that would otherwise make grow allocate gigabytes or panic.
const maxGridCells = 4096 * 4096

// fits reports whether the grid including r stays within maxGridCells (overflow-safe).
func (g *Grid) fits(r Rect) bool {
	const lim = 1 << 30
	if r.W > lim || r.H > lim || r.X < -lim || r.X > lim || r.Y < -lim || r.Y > lim {
		return false
	}
	x0, y0, x1, y1 := r.X, r.Y, r.Right(), r.Bottom()
	if g.w > 0 {
		x0, y0 = min(x0, g.x0), min(y0, g.y0)
		x1, y1 = max(x1, g.x0+g.w), max(y1, g.y0+g.h)
	}
	return int64(x1-x0)*int64(y1-y0) <= maxGridCells
}

// SetBlock merges a rectangle of cells (row-major, len == r.W*r.H); the new data overwrites any
// overlap. It returns whether any cell changed.
func (g *Grid) SetBlock(r Rect, cells []uint16) bool {
	if r.Empty() || len(cells) != r.W*r.H || !g.fits(r) {
		return false
	}
	g.grow(r)
	changed := false
	for y := 0; y < r.H; y++ {
		o := (r.Y+y-g.y0)*g.w + (r.X - g.x0)
		for x := 0; x < r.W; x++ {
			c := cells[y*r.W+x]
			if !g.known[o+x] {
				g.known[o+x] = true
				g.nknown++
				changed = true
			}
			if g.cells[o+x] != c {
				g.cells[o+x] = c
				changed = true
			}
		}
	}
	if changed {
		g.version++
	}
	return changed
}

// Clone returns an independent copy.
func (g *Grid) Clone() *Grid {
	c := *g
	c.cells = append([]uint16(nil), g.cells...)
	c.known = append([]bool(nil), g.known...)
	return &c
}

// Region copies the half-open window [x0,x1) x [y0,y1) into a new Grid. Unknown cells stay
// unknown; the result's Bounds is the window clipped to this grid's storage.
func (g *Grid) Region(x0, y0, x1, y1 int) *Grid {
	out := NewGrid()
	x0, y0 = max(x0, g.x0), max(y0, g.y0)
	x1, y1 = min(x1, g.x0+g.w), min(y1, g.y0+g.h)
	if x1 <= x0 || y1 <= y0 {
		return out
	}
	out.grow(Rect{x0, y0, x1 - x0, y1 - y0})
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if i := g.idx(x, y); g.known[i] {
				o := out.idx(x, y)
				out.cells[o], out.known[o] = g.cells[i], true
				out.nknown++
			}
		}
	}
	out.version = 1
	return out
}

// ParseASCII builds a grid from rows of text for tests: '#' blocked (BlockWalk), '.' walkable,
// ' ' or '?' unknown. Row 0 is y0, column 0 is x0.
func ParseASCII(x0, y0 int, rows ...string) *Grid {
	g := NewGrid()
	for dy, row := range rows {
		for dx, ch := range row {
			switch ch {
			case '#':
				g.SetBlock(Rect{x0 + dx, y0 + dy, 1, 1}, []uint16{BlockWalk})
			case '.':
				g.SetBlock(Rect{x0 + dx, y0 + dy, 1, 1}, []uint16{0})
			}
		}
	}
	return g
}

// ASCII renders the window r (clipped to Bounds when r is empty) one character per subtile,
// north (small y) at the top. Marks override cells. Legend: ' ' unknown, '.' walkable,
// '#' BlockWalk, '+' closed door, '~' walkable but another flag set (monster, object...).
func (g *Grid) ASCII(r Rect, marks map[Point]rune) string {
	if r.Empty() {
		r = g.Bounds()
	}
	var sb strings.Builder
	for y := r.Y; y < r.Bottom(); y++ {
		for x := r.X; x < r.Right(); x++ {
			ch := ' '
			if f, ok := g.Flags(x, y); ok {
				switch {
				case f&BlockWalk != 0:
					ch = '#'
				case f&ClosedDoor != 0:
					ch = '+'
				case f != 0:
					ch = '~'
				default:
					ch = '.'
				}
			}
			if m, ok := marks[Point{x, y}]; ok {
				ch = m
			}
			sb.WriteRune(ch)
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

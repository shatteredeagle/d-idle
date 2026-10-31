package nav

import (
	"math"
	"sync"
)

// Planner runs A* over a Field. It owns reusable scratch arrays (generation-stamped, so a search never clears
// them). Plan is serialized by mu: the guard's Fighter.Safety (PathTo) can run beside the acting goroutine on
// Game.exe, and unlocked that corrupted the heap (index out of range in pop). The unexported search methods are
// not locked: only call them under Plan or on a private Planner.
type Planner struct {
	mu    sync.Mutex
	g     []float32
	stamp []uint32
	from  []uint8
	gen   uint32
	heap  []hnode

	// Stats of the last search, for benchmarks.
	Expanded int
	Coarse   bool
	// Truncated is set when the last search hit maxExpandAbs and returned its best partial path.
	Truncated bool
	maxExp    int  // expansion cap for the next search, 0 = none
	aborted   bool // the last search hit maxExp
}

type hnode struct {
	f float32
	i int32
}

var dirs = [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {-1, 1}, {1, -1}, {-1, -1}}

const sqrt2 = 1.4142135

// maxExpandAbs caps the expansions of any single search (a full flood of a maxFieldCells window).
var maxExpandAbs = maxFieldCells // a var so tests can lower it

func (p *Planner) push(n hnode) {
	h := append(p.heap, n)
	i := len(h) - 1
	for i > 0 {
		par := (i - 1) / 2
		if h[par].f <= h[i].f {
			break
		}
		h[par], h[i] = h[i], h[par]
		i = par
	}
	p.heap = h
}

func (p *Planner) pop() hnode {
	h := p.heap
	top := h[0]
	n := len(h) - 1
	h[0] = h[n]
	h = h[:n]
	i := 0
	for {
		l := 2*i + 1
		if l >= n {
			break
		}
		if r := l + 1; r < n && h[r].f < h[l].f {
			l = r
		}
		if h[i].f <= h[l].f {
			break
		}
		h[i], h[l] = h[l], h[i]
		i = l
	}
	p.heap = h
	return top
}

func (p *Planner) prepare(n int) {
	if len(p.g) < n {
		p.g = make([]float32, n)
		p.stamp = make([]uint32, n)
		p.from = make([]uint8, n)
		p.gen = 0
	}
	p.gen++
	if p.gen == 0 {
		for i := range p.stamp {
			p.stamp[i] = 0
		}
		p.gen = 1
	}
	p.heap = p.heap[:0]
	p.Expanded = 0
	p.aborted, p.Truncated = false, false
}

func octile(dx, dy int) float32 {
	dx, dy = abs(dx), abs(dy)
	if dx < dy {
		dx, dy = dy, dx
	}
	return float32(dx-dy) + sqrt2*float32(dy)
}

// search is weighted A* from (sx,sy) to the goal cell (gx,gy); it stops when a node within reach (cells) of the
// goal is popped. mask, if non-nil, restricts expansion to cells where mask[idx] is true. hw >= 1 inflates the
// heuristic. When the goal cannot be reached the path to the explored cell nearest the goal is returned with
// partial = true.
func (p *Planner) search(f *Field, sx, sy, gx, gy int, reach float32, hw float32, mask []bool) (path []int32, partial bool) {
	si := f.idx(sx, sy)
	if si < 0 || f.flags[si]&f.passMask == 0 {
		return nil, false
	}
	p.prepare(f.W * f.H)
	p.g[si], p.stamp[si], p.from[si] = 0, p.gen, 255
	p.push(hnode{hw * octile(gx-sx, gy-sy), int32(si)})
	best, bestH := int32(si), octile(gx-sx, gy-sy)
	W := f.W
	for len(p.heap) > 0 {
		cur := p.pop()
		ci := cur.i
		cx, cy := int(ci)%W, int(ci)/W
		gcur := p.g[ci]
		// stale heap entry: f no longer matches the best g
		h := octile(f.X0+cx-gx, f.Y0+cy-gy)
		if cur.f > gcur+hw*h+0.01+0.002*h {
			continue
		}
		p.Expanded++
		if p.maxExp > 0 && p.Expanded > p.maxExp {
			p.aborted = true
			return nil, true
		}
		if p.Expanded > maxExpandAbs {
			// no search of a real level needs this many; give the best partial path instead of running away
			p.Truncated = true
			break
		}
		if h < bestH {
			best, bestH = ci, h
		}
		if h <= reach {
			return p.trace(f, ci), false
		}
		for d, dv := range dirs {
			nx, ny := cx+dv[0], cy+dv[1]
			if nx < 0 || ny < 0 || nx >= W || ny >= f.H {
				continue
			}
			ni := ny*W + nx
			if f.flags[ni]&f.passMask == 0 {
				continue
			}
			if mask != nil && !mask[ni] {
				continue
			}
			step := float32(1)
			if d >= 4 {
				if f.flags[cy*W+nx]&f.passMask == 0 || f.flags[ny*W+cx]&f.passMask == 0 {
					continue
				}
				step = sqrt2
			}
			ng := gcur + step*(1+float32(f.wall[ni]+f.dyn[ni])/costUnit)
			if p.stamp[ni] == p.gen && p.g[ni] <= ng {
				continue
			}
			p.stamp[ni], p.g[ni], p.from[ni] = p.gen, ng, uint8(d)
			nh := octile(f.X0+nx-gx, f.Y0+ny-gy)
			p.push(hnode{ng + hw*nh + 0.002*nh, int32(ni)})
		}
	}
	if best == int32(si) {
		return []int32{int32(si)}, true
	}
	return p.trace(f, best), true
}

func (p *Planner) trace(f *Field, end int32) []int32 {
	var rev []int32
	for i := end; len(rev) <= len(p.from); { // a parent cycle (a bug) must end, not allocate forever
		rev = append(rev, i)
		d := p.from[i]
		if d == 255 {
			break
		}
		x, y := int(i)%f.W-dirs[d][0], int(i)/f.W-dirs[d][1]
		i = int32(y*f.W + x)
	}
	for l, r := 0, len(rev)-1; l < r; l, r = l+1, r-1 {
		rev[l], rev[r] = rev[r], rev[l]
	}
	return rev
}

// coarsen shrinks f by factor b: a block is walkable when at least half of its cells are.
func coarsen(f *Field, b int) *Field {
	c := &Field{X0: f.X0 / b, Y0: f.Y0 / b, passMask: fWalk}
	if f.X0 < 0 || f.Y0 < 0 { // coordinates are positive in D2; keep the guard simple
		c.X0, c.Y0 = 0, 0
	}
	c.W = (f.X0+f.W+b-1)/b - c.X0
	c.H = (f.Y0+f.H+b-1)/b - c.Y0
	n := c.W * c.H
	c.flags = make([]uint8, n)
	c.wall = make([]uint8, n)
	c.dyn = make([]uint8, n)
	cnt := make([]uint8, n)
	tot := make([]uint8, n)
	for y := 0; y < f.H; y++ {
		for x := 0; x < f.W; x++ {
			i := y*f.W + x
			ci := ((f.Y0+y)/b-c.Y0)*c.W + (f.X0+x)/b - c.X0
			tot[ci]++
			if f.flags[i]&fWalk != 0 && f.clr[i] >= 2 {
				cnt[ci]++
			}
		}
	}
	for i := range c.flags {
		if tot[i] > 0 && int(cnt[i])*2 >= int(tot[i]) {
			c.flags[i] = fWalk | fKnown
		}
	}
	return c
}

const (
	coarseBlock = 5      // one tile
	coarseMin   = 180000 // fields smaller than this are searched directly
	fineBudget  = 40000  // expansions of plain A* before switching to the corridor
)

// Plan finds the cell path from (sx,sy) to (gx,gy) in f; see search for partial. Large fields are searched
// coarse-to-fine: a path over 5x5 blocks defines a corridor, and the exact search runs inside it. If the corridor
// fails, the exact search runs on the whole field.
func (p *Planner) Plan(f *Field, sx, sy, gx, gy int, reach float64, hw float64) (path []int32, partial bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Coarse = false
	if f.W*f.H >= coarseMin && hypotI(gx-sx, gy-sy) > 60 {
		// Cheap first: open terrain is solved by plain A* in a few thousand expansions. Only a search that
		// grows past the cap (walls, mazes) pays for the coarse corridor.
		p.maxExp = fineBudget
		path, partial = p.search(f, sx, sy, gx, gy, float32(reach), float32(hw), nil)
		p.maxExp = 0
		if !p.aborted {
			return path, partial
		}
		if m := p.corridor(f, sx, sy, gx, gy); m != nil {
			p.Coarse = true
			path, partial = p.search(f, sx, sy, gx, gy, float32(reach), float32(hw), m)
			if !partial && path != nil {
				return path, false
			}
		}
		p.Coarse = false
	}
	return p.search(f, sx, sy, gx, gy, float32(reach), float32(hw), nil)
}

func hypotI(a, b int) float64 { return math.Hypot(float64(a), float64(b)) }

// corridor returns a fine-grid mask of the blocks near the coarse path, or nil if the coarse search fails.
func (p *Planner) corridor(f *Field, sx, sy, gx, gy int) []bool {
	const b = coarseBlock
	c := coarsen(f, b)
	csx, csy := sx/b, sy/b
	cgx, cgy := gx/b, gy/b
	// start and goal blocks must be walkable even if mostly wall
	for _, q := range [][2]int{{csx, csy}, {cgx, cgy}} {
		if i := c.idx(q[0], q[1]); i >= 0 {
			c.flags[i] = fWalk | fKnown
		}
	}
	var sub Planner
	cp, partial := sub.search(c, csx, csy, cgx, cgy, 0, 1, nil)
	if partial || len(cp) == 0 {
		return nil
	}
	mask := make([]bool, f.W*f.H)
	const rad = 2 // blocks of slack on both sides
	for _, ci := range cp {
		bx, by := c.pt(ci)
		for dy := -rad; dy <= rad; dy++ {
			for dx := -rad; dx <= rad; dx++ {
				for y := (by + dy) * b; y < (by+dy+1)*b; y++ {
					for x := (bx + dx) * b; x < (bx+dx+1)*b; x++ {
						if i := f.idx(x, y); i >= 0 {
							mask[i] = true
						}
					}
				}
			}
		}
	}
	return mask
}

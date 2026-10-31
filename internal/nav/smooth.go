package nav

import (
	"math"
	"math/rand"

	"d2lod-bot/internal/input"
)

// walkLine steps through every cell the line touches (never cutting a corner) and returns its cost, in the same
// units as the A* path cost, and whether every cell is walkable with at least minClr half-cells of clearance.
func (f *Field) walkLine(x0, y0, x1, y1 int, minClr uint8) (float64, bool) {
	i0, i1 := f.idx(x0, y0), f.idx(x1, y1)
	if i0 < 0 || i1 < 0 {
		return 0, false
	}
	if c := f.clr[i0]; c < minClr {
		minClr = c
	}
	if c := f.clr[i1]; c < minClr {
		minClr = c
	}
	ok := func(x, y int) bool {
		i := f.idx(x, y)
		return i >= 0 && f.flags[i]&f.passMask != 0 && f.clr[i] >= minClr
	}
	dx, dy := abs(x1-x0), abs(y1-y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	x, y := x0, y0
	e := dx - dy
	cost := 0.0
	for {
		if !ok(x, y) {
			return 0, false
		}
		if x == x1 && y == y1 {
			return cost, true
		}
		e2 := 2 * e
		stepX, stepY := e2 > -dy, e2 < dx
		step := 1.0
		if stepX && stepY {
			if !ok(x+sx, y) || !ok(x, y+sy) {
				return 0, false
			}
			step = sqrt2
		}
		if stepX {
			e -= dy
			x += sx
		}
		if stepY {
			e += dx
			y += sy
		}
		i := f.idx(x, y)
		cost += step * (1 + float64(int(f.wall[i])+int(f.dyn[i]))/costUnit)
	}
}

// los reports whether the straight line stays on walkable cells with at least minClr half-cells of clearance.
func (f *Field) los(x0, y0, x1, y1 int, minClr uint8) bool {
	_, ok := f.walkLine(x0, y0, x1, y1, minClr)
	return ok
}

type pt struct{ x, y int }

func (f *Field) points(path []int32) []pt {
	out := make([]pt, len(path))
	for i, c := range path {
		x, y := f.pt(c)
		out[i] = pt{x, y}
	}
	return out
}

// smooth is string pulling: from each anchor jump to the farthest later point that is in line of sight and whose
// straight line costs no more than the A* path between them (so shortcuts never pull the path back against a wall
// or through a monster the planner went around). The farthest point is found by doubling then bisecting, which
// assumes line of sight is roughly monotone along the path.
func (f *Field) smooth(p []pt, prefix []float64) []pt {
	if len(p) <= 2 {
		return p
	}
	good := func(a, j int) bool {
		c, ok := f.walkLine(p[a].x, p[a].y, p[j].x, p[j].y, 0)
		return ok && c <= prefix[j]-prefix[a]+0.5
	}
	out := []pt{p[0]}
	a := 0
	for a < len(p)-1 {
		lo, hi := a+1, -1 // lo is known good (adjacent cells), hi the first failure
		for step := 1; ; step *= 2 {
			j := a + step
			if j >= len(p) {
				j = len(p) - 1
				if good(a, j) {
					lo = j
				} else {
					hi = j
				}
				break
			}
			if !good(a, j) {
				hi = j
				break
			}
			lo = j
		}
		for hi > 0 && hi-lo > 1 {
			m := (lo + hi) / 2
			if good(a, m) {
				lo = m
			} else {
				hi = m
			}
		}
		out = append(out, p[lo])
		a = lo
	}
	return out
}

// pathPrefix returns the cumulative A* cost along the cell path.
func (f *Field) pathPrefix(p []pt) []float64 {
	out := make([]float64, len(p))
	for i := 1; i < len(p); i++ {
		step := 1.0
		if p[i].x != p[i-1].x && p[i].y != p[i-1].y {
			step = sqrt2
		}
		k := f.idx(p[i].x, p[i].y)
		out[i] = out[i-1] + step*(1+float64(int(f.wall[k])+int(f.dyn[k]))/costUnit)
	}
	return out
}

// resample returns points along the polyline every spacing subtiles, always ending on the last vertex. The start
// vertex is not included.
func resample(p []pt, spacing float64, ok func(a, b pt) bool) []pt {
	if len(p) < 2 {
		return nil
	}
	var out []pt
	acc := 0.0 // distance walked since the last emitted point
	for i := 1; i < len(p); i++ {
		ax, ay := float64(p[i-1].x), float64(p[i-1].y)
		bx, by := float64(p[i].x), float64(p[i].y)
		seg := math.Hypot(bx-ax, by-ay)
		pos := 0.0
		for seg > 0 && acc+(seg-pos) >= spacing {
			pos += spacing - acc
			acc = 0
			t := pos / seg
			c := pt{int(math.Round(ax + (bx-ax)*t)), int(math.Round(ay + (by-ay)*t))}
			last := p[0]
			if len(out) > 0 {
				last = out[len(out)-1]
			}
			if ok != nil && !ok(last, c) { // rounding stepped across a corner: fall back to the segment's start vertex
				c = p[i-1]
			}
			out = append(out, c)
		}
		acc += seg - pos
		if i < len(p)-1 && acc >= 1 { // corners are always waypoints, so consecutive waypoints stay in line of sight
			out = append(out, p[i])
			acc = 0
		}
	}
	end := p[len(p)-1]
	if n := len(out); n > 0 && out[n-1] == end {
		return out
	}
	if n := len(out); n > 0 && math.Hypot(float64(out[n-1].x-end.x), float64(out[n-1].y-end.y)) < spacing/2 {
		out = out[:n-1]
	}
	return append(out, end)
}

// jitter displaces interior waypoints sideways by up to amp subtiles, keeping a displacement only if the segment
// from the previous waypoint stays clear. This is the "sloppy walker" handicap.
func (f *Field) jitter(start pt, wps []pt, amp float64, rng *rand.Rand) []pt {
	if amp < 0.5 || len(wps) < 2 {
		return wps
	}
	out := make([]pt, len(wps))
	copy(out, wps)
	prev := start
	for i := 0; i < len(out)-1; i++ {
		dx, dy := float64(out[i+1].x-prev.x), float64(out[i+1].y-prev.y)
		l := math.Hypot(dx, dy)
		if l > 0 {
			off := (rng.Float64()*2 - 1) * amp
			c := pt{out[i].x + int(math.Round(-dy/l*off)), out[i].y + int(math.Round(dx/l*off))}
			if f.los(prev.x, prev.y, c.x, c.y, 2) {
				out[i] = c
			}
		}
		prev = out[i]
	}
	return out
}

func toInput(p []pt) []input.Point {
	out := make([]input.Point, len(p))
	for i, q := range p {
		out[i] = input.Point{X: q.x, Y: q.y}
	}
	return out
}

// pathLength is the Euclidean length of a polyline starting at from.
func pathLength(from input.Point, p []input.Point) float64 {
	l, prev := 0.0, from
	for _, q := range p {
		l += math.Hypot(float64(q.X-prev.X), float64(q.Y-prev.Y))
		prev = q
	}
	return l
}

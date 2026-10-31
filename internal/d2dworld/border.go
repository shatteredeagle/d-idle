package d2dworld

import (
	"context"
	"fmt"
	"strings"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2map"
)

// Placement is one line of the fork verb "debug layout": where a level of the player's act sits, in act tiles (the
// units of Levels.txt Offset and of the wx wy of "debug level"; 5 subtiles per tile).
type Placement struct{ Level, X, Y, W, H int }

// Near is one line of the fork verb "debug nearby": a level linked past the current level's edge so far (d2d builds
// neighbours on a worker thread, so the list grows after arrival). DX, DY are its origin minus ours, in tiles.
type Near struct{ Level, DX, DY int }

// Layouter is the optional part of a Verbs that knows the act layout. Only the private fork has these two verbs
// (docs/d2d-backend.md Phase 3); with stock d2d, or an "err" reply, the world has no Adjacent entries and nav finds no
// outdoor gate.
type Layouter interface {
	Layout(ctx context.Context) ([]Placement, error)
	Nearby(ctx context.Context) ([]Near, error)
}

// prober is the part of *d2dctl.Client used to look across a border ("debug blocked" past the edge consults the
// neighbour level's collision in d2d, V in the d2d source: Level::blocked).
type prober interface {
	Blocked(ctx context.Context, x, y float64) (bool, error)
}

// Client adds the fork verbs to a *d2dctl.Client. Use it as the Verbs of New.
type Client struct{ *d2dctl.Client }

func (c Client) Layout(ctx context.Context) ([]Placement, error) {
	r, err := c.Debug(ctx, "layout")
	if err != nil {
		return nil, err
	}
	return parseInts(r.Lines, 5, func(v []int) Placement { return Placement{v[0], v[1], v[2], v[3], v[4]} })
}

func (c Client) Nearby(ctx context.Context) ([]Near, error) {
	r, err := c.Debug(ctx, "nearby")
	if err != nil {
		return nil, err
	}
	return parseInts(r.Lines, 3, func(v []int) Near { return Near{v[0], v[1], v[2]} })
}

func parseInts[T any](lines []string, n int, mk func([]int) T) ([]T, error) {
	var out []T
	for _, l := range lines {
		f := strings.Fields(l)
		if len(f) < n {
			return nil, fmt.Errorf("want %d fields: %q", n, l)
		}
		v := make([]int, n)
		for i := range v {
			if _, err := fmt.Sscanf(f[i], "%d", &v[i]); err != nil {
				return nil, fmt.Errorf("%q: %w", l, err)
			}
		}
		out = append(out, mk(v))
	}
	return out, nil
}

// abut returns the neighbour rectangles of an act layout that touch the level (tiles, same frame as own) with the
// side the shared edge lies on. Touching means the edges coincide and the other axis overlaps; the DRLG placements
// (docs/research drlg.md: "clear of every earlier level except its link, gap >= 0") put linked levels edge to edge.
type abutting struct {
	p    Placement
	side int // 0 below (neighbour south), 1 left, 2 above, 3 right (the DRLG's directions)
	lo   int // overlap along the shared edge, in tiles, [lo,hi) in the frame of own
	hi   int
}

func abut(own Placement, all []Placement) []abutting {
	var out []abutting
	for _, p := range all {
		if p.Level == own.Level || p.W <= 0 || p.H <= 0 {
			continue
		}
		xlo, xhi := max(own.X, p.X), min(own.X+own.W, p.X+p.W)
		ylo, yhi := max(own.Y, p.Y), min(own.Y+own.H, p.Y+p.H)
		switch {
		case p.Y == own.Y+own.H && xhi > xlo:
			out = append(out, abutting{p, 0, xlo, xhi})
		case p.X+p.W == own.X && yhi > ylo:
			out = append(out, abutting{p, 1, ylo, yhi})
		case p.Y+p.H == own.Y && xhi > xlo:
			out = append(out, abutting{p, 2, xlo, xhi})
		case p.X == own.X+own.W && yhi > ylo:
			out = append(out, abutting{p, 3, ylo, yhi})
		}
	}
	return out
}

// adjacent builds the d2map.Adjacent entries of one level. own and all are layout placements in act tiles; self is the
// level's layout rectangle in subtiles: the strip is its last row or column, like Game.exe's Level rectangle (the d2d
// grid is one tile larger, debug level says 57x41 for the 56x40 town, V live, and that margin is open beside the wall). linked, when non-nil, restricts the result to levels d2d has
// already linked (their collision answers "debug blocked" past the edge); pending counts abutting levels that are
// not linked yet. isOpen decides per subtile position inside the level at the shared edge whether the cell across
// can be walked (nil: trust the inside cell only).
func adjacent(own Placement, self d2map.Rect, all []Placement, linked map[int]bool, grid *d2map.Grid, across func(side, sx, sy int) bool) (adj []d2map.Adjacent, pending, unlinked int) {
	for _, a := range abut(own, all) {
		if linked != nil && !linked[a.p.Level] {
			pending++
			unlinked++
			continue
		}
		nb := d2map.Rect{X: self.X + 5*(a.p.X-own.X), Y: self.Y + 5*(a.p.Y-own.Y), W: 5 * a.p.W, H: 5 * a.p.H}
		// the strip of this level one subtile wide along the shared edge, over the overlap
		var strip d2map.Rect
		switch a.side {
		case 0:
			strip = d2map.Rect{X: self.X + 5*(a.lo-own.X), Y: self.Bottom() - 1, W: 5 * (a.hi - a.lo), H: 1}
		case 2:
			strip = d2map.Rect{X: self.X + 5*(a.lo-own.X), Y: self.Y, W: 5 * (a.hi - a.lo), H: 1}
		case 1:
			strip = d2map.Rect{X: self.X, Y: self.Y + 5*(a.lo-own.Y), W: 1, H: 5 * (a.hi - a.lo)}
		default:
			strip = d2map.Rect{X: self.Right() - 1, Y: self.Y + 5*(a.lo-own.Y), W: 1, H: 5 * (a.hi - a.lo)}
		}
		horizontal := strip.W > strip.H // the strip runs along x
		n := strip.W
		if !horizontal {
			n = strip.H
		}
		runStart := -1
		before := len(adj)
		flush := func(end int) {
			if runStart < 0 {
				return
			}
			c := strip
			if horizontal {
				c.X, c.W = strip.X+runStart, end-runStart
			} else {
				c.Y, c.H = strip.Y+runStart, end-runStart
			}
			adj = append(adj, d2map.Adjacent{LevelNo: a.p.Level, Rect: nb, Contact: c})
			runStart = -1
		}
		const step = 5 // the probe grain: one tile; a gate is at least that wide
		at := func(i int) (int, int) {
			if horizontal {
				return strip.X + i, strip.Y
			}
			return strip.X, strip.Y + i
		}
		tiles := map[int]bool{}
		tileOpen := func(i int) bool {
			if across == nil {
				return true
			}
			k := i / step
			v, ok := tiles[k]
			if !ok {
				sx, sy := at(min(k*step+step/2, n-1))
				v = across(a.side, sx, sy)
				tiles[k] = v
			}
			return v
		}
		for i := 0; i < n; i++ {
			sx, sy := at(i)
			open := grid.Walkable(sx, sy) && tileOpen(i)
			if open && runStart < 0 {
				runStart = i
			}
			if !open {
				flush(i)
			}
		}
		flush(n)
		if linked != nil && across != nil && len(adj) == before {
			// listed as linked but no cell is open across yet: d2d may not have its collision built (V live: the
			// level list showed Cold Plains before the first probe could see it), so keep looking, like a pending link.
			pending++
		}
	}
	return adj, pending, unlinked
}

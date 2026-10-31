package nav

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

// A world source that hands out whole level maps (d2dworld) implements FullyKnown. Explore then has no unknown
// cells to reveal and would report ErrExplored at once, so the area would count as cleared without a single
// monster met. roam defines coverage by visiting reachable ground instead (docs/d2d-pace.md section 'roam').
type fullyKnown interface{ FullyKnown() bool }

const (
	// roamSpacing is the distance between coverage points: about the monster wake/aggro radius (S: ~7 cells in
	// 1.14d), so one visit per point wakes everything of its region. U: not measured on d2d.
	roamSpacing = 7
	// roamCover is how close the character must come for a point to count as visited.
	roamCover = 5
	// roamPasses caps the legs of one level at roamPasses times the number of points, so it cannot roam forever.
	roamPasses = 3
)

// coverage is the roam state of one level of one game. It survives across Explore calls (the bot fights between
// legs and calls Explore again), and is rebuilt when the level map changes (new game).
type coverage struct {
	lm     *d2map.LevelMap // identity of the level in this game (d2dworld swaps the Grid inside it on refresh)
	level  int
	pts    []input.Point
	seen   []bool
	failed []bool
	legs   int
}

func (c *coverage) visited() int {
	k := 0
	for i := range c.pts {
		if c.seen[i] {
			k++
		}
	}
	return k
}

// Coverage reports how many coverage points of the current level were visited and how many there are
// (0, 0 when roaming is not in use).
func (n *Navigator) Coverage() (visited, total int) {
	if n.cov == nil {
		return 0, 0
	}
	return n.cov.visited(), len(n.cov.pts)
}

func (n *Navigator) fullyKnown() bool {
	f, ok := n.w.(fullyKnown)
	return ok && f.FullyKnown()
}

// samplePoints picks one reachable walkable cell per spacing x spacing block (the one closest to the block centre).
func samplePoints(f *Field, dist []int32) []input.Point {
	type pick struct {
		i int32
		d int
	}
	best := map[[2]int]pick{}
	for i, d := range dist {
		if d < 0 {
			continue
		}
		x, y := f.pt(int32(i))
		b := [2]int{floorDiv(x, roamSpacing), floorDiv(y, roamSpacing)}
		c := abs(x-(b[0]*roamSpacing+roamSpacing/2)) + abs(y-(b[1]*roamSpacing+roamSpacing/2))
		if p, ok := best[b]; !ok || c < p.d {
			best[b] = pick{int32(i), c}
		}
	}
	pts := make([]input.Point, 0, len(best))
	for _, p := range best {
		x, y := f.pt(p.i)
		pts = append(pts, input.Point{X: x, Y: y})
	}
	sort.Slice(pts, func(i, j int) bool { return pts[i].X < pts[j].X || pts[i].X == pts[j].X && pts[i].Y < pts[j].Y })
	return pts
}

// floodHops is the 8-neighbour walking distance from (sx,sy) to every walkable cell of f (-1: unreachable).
func floodHops(f *Field, sx, sy int) []int32 {
	dist := make([]int32, f.W*f.H)
	for i := range dist {
		dist[i] = -1
	}
	s := int32(f.idx(sx, sy))
	dist[s] = 0
	queue := append(make([]int32, 0, 1024), s)
	for head := 0; head < len(queue); head++ {
		ci := queue[head]
		cx, cy := f.pt(ci)
		for d, dv := range dirs {
			nx, ny := cx+dv[0], cy+dv[1]
			ni := f.idx(nx, ny)
			if ni < 0 || dist[ni] >= 0 || f.flags[ni]&fWalk == 0 {
				continue
			}
			if d >= 4 && (!f.Walkable(cx+dv[0], cy) || !f.Walkable(cx, cy+dv[1])) {
				continue
			}
			dist[ni] = dist[ci] + 1
			queue = append(queue, int32(ni))
		}
	}
	return dist
}

// roamExitClear keeps coverage points away from tile exits (caves, stairs): walking to a point next to the Den of Evil
// entrance entered the Den, the guard walked back, and the roam started over in a loop (D2DLOOP r2, V live).
const roamExitClear = 7

func dropNearExits(pts []input.Point, exits []d2map.Exit) []input.Point {
	if len(exits) == 0 {
		return pts
	}
	out := pts[:0:0]
next:
	for _, p := range pts {
		for _, e := range exits {
			if max(abs(p.X-e.Pos.X), abs(p.Y-e.Pos.Y)) <= roamExitClear {
				continue next
			}
		}
		out = append(out, p)
	}
	return out
}

func (c *coverage) mark(p input.Point) {
	for i, q := range c.pts {
		if !c.seen[i] && max(abs(q.X-p.X), abs(q.Y-p.Y)) <= roamCover {
			c.seen[i] = true
		}
	}
}

// roam visits the coverage points of the level nearest-first until done() reports true (nil), every point was
// visited or given up (ErrExplored), or the leg or time cap is hit.
func (n *Navigator) roam(ctx context.Context, done func() bool) error {
	t := n.tune()
	deadline := n.clk.Now().Add(t.ExploreLimit())
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if done() {
			return nil
		}
		if n.clk.Now().After(deadline) {
			return fmt.Errorf("%w: roaming for %v", ErrTimeout, t.ExploreLimit())
		}
		o, err := n.observeWait(ctx)
		if err != nil {
			return err
		}
		if err := n.guard(o); err != nil {
			return err
		}
		f := n.fieldFor(o.lm)
		sx, sy, ok := f.nearestWalkable(o.pos.X, o.pos.Y, 8)
		if !ok {
			return ErrExplored
		}
		dist := floodHops(f, sx, sy)
		c := n.cov
		if c == nil || c.lm != o.lm {
			pts := dropNearExits(samplePoints(f, dist), o.lm.Exits)
			c = &coverage{lm: o.lm, level: o.level, pts: pts, seen: make([]bool, len(pts)), failed: make([]bool, len(pts))}
			n.cov = c
			n.log.Info("roam: coverage points", "level", o.level, "points", len(pts), "exits", len(o.lm.Exits))
		}
		c.mark(o.pos)
		pick, bestD := -1, int32(0)
		for i, p := range c.pts {
			if c.seen[i] || c.failed[i] {
				continue
			}
			if len(o.lm.Exits) > 0 && len(dropNearExits([]input.Point{p}, o.lm.Exits)) == 0 {
				c.seen[i] = true // the exits may have been fetched after the points were made
				continue
			}
			if j := f.idx(p.X, p.Y); j >= 0 && dist[j] >= 0 && (pick < 0 || dist[j] < bestD) {
				pick, bestD = i, dist[j]
			}
		}
		if pick < 0 || c.legs >= roamPasses*len(c.pts)+roamPasses {
			n.log.Info("roam: done", "visited", c.visited(), "points", len(c.pts), "legs", c.legs)
			return ErrExplored
		}
		c.legs++
		target := c.pts[pick]
		stop := func(o *obs) bool {
			c.mark(o.pos)
			return done() || c.seen[pick]
		}
		err = n.walk(ctx, func(*obs) (input.Point, float64, error) { return target, 3, nil },
			walkOpts{stopWhen: stop, timeout: 90 * time.Second})
		switch {
		case err == nil:
			if !done() {
				c.seen[pick] = true
			}
		case errors.Is(err, ErrStuck), errors.Is(err, ErrNoPath), errors.Is(err, ErrTimeout):
			c.failed[pick] = true
			n.log.Info("roam leg failed", "to", target, "err", err)
		default:
			return err
		}
	}
}

package nav

import (
	"context"
	"errors"
	"sort"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

const probeSeen = 40 // a target this close to a place already visited counts as visited

// FarProbe visits the far parts of a big level (a Tal Rasha tomb: the walk grid only knows the streamed rooms, so
// Explore says "nothing left" long before the far chambers were seen) so the client streams them in. Each Step goes
// to one unvisited target. Targets are the centres of the level's static Room2 list when it knows rooms the grid
// does not (U: whether unstreamed rooms are listed), else probe points on the level rectangle. Far targets first, so the
// trip through the arms streams the rooms on the way.
type FarProbe struct {
	n       *Navigator
	level   int
	visited []input.Point
}

func (n *Navigator) NewFarProbe() *FarProbe { return &FarProbe{n: n} }

// Step goes to the next target and reports whether more targets remain. A hop or walk that fails (blocked arm, no
// teleport) still counts the target as visited, so the probe always ends.
func (p *FarProbe) Step(ctx context.Context) (more bool, err error) {
	o, err := p.n.observeWait(ctx)
	if err != nil {
		return false, err
	}
	if p.level != o.level {
		p.level, p.visited = o.level, nil
	}
	p.visited = append(p.visited, o.pos)
	targets := FarTargets(o.lm, o.pos, p.visited)
	if len(targets) == 0 {
		return false, nil
	}
	t := targets[0]
	p.visited = append(p.visited, t)
	if p.n.hasTeleport() {
		err = p.n.TeleportTo(ctx, t.X, t.Y)
	} else {
		stop := float64(probeSeen)
		err = p.n.ExploreToward(ctx, t, func() bool {
			c, e := p.n.in.Pos()
			return e == nil && dist(c, t) <= stop
		})
	}
	if err != nil && !errors.Is(err, ErrNoPath) && !errors.Is(err, ErrExplored) {
		return false, err
	}
	return len(targets) > 1, nil
}

// FarTargets lists unvisited probe points of the level, farthest from pos first. Pure, for tests.
func FarTargets(lm *d2map.LevelMap, pos input.Point, visited []input.Point) []input.Point {
	if lm == nil {
		return nil
	}
	var cand []input.Point
	for _, r := range lm.Rooms {
		if !r.Rect.Empty() {
			cand = append(cand, input.Point{X: r.Rect.X + r.Rect.W/2, Y: r.Rect.Y + r.Rect.H/2})
		}
	}
	if len(cand) == 0 && !lm.Rect.Empty() {
		const in = 30
		r := lm.Rect
		xs := [3]int{r.X + in, r.X + r.W/2, r.Right() - in}
		ys := [3]int{r.Y + in, r.Y + r.H/2, r.Bottom() - in}
		for _, x := range xs {
			for _, y := range ys {
				if x != xs[1] || y != ys[1] {
					cand = append(cand, input.Point{X: x, Y: y})
				}
			}
		}
	}
	var out []input.Point
	for _, c := range cand {
		seen := false
		for _, v := range visited {
			if dist(c, v) <= probeSeen {
				seen = true
				break
			}
		}
		if !seen {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return dist(out[i], pos) > dist(out[j], pos) })
	return out
}

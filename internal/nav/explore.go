package nav

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

// Explore walks to frontiers (known walkable cells next to unknown cells inside the level rectangle) until
// done() reports true, so the client loads more rooms. Without a done callback it runs until nothing is left
// (ErrExplored) or the explore timeout. Frontiers near places the character already stood are skipped: if the
// client did not load more there, more walking will not help (dead end).
func (n *Navigator) Explore(ctx context.Context, done func() bool) error {
	return n.explore(ctx, done, nil)
}

// ExploreToward is Explore with informed scoring biased toward hint (for example the level's far side or a
// position known from another source).
func (n *Navigator) ExploreToward(ctx context.Context, hint input.Point, done func() bool) error {
	return n.explore(ctx, done, &hint)
}

type frontier struct {
	pos    input.Point
	hops   int
	count  int
	bucket [2]int
	score  float64
}

func (n *Navigator) explore(ctx context.Context, done func() bool, hint *input.Point) error {
	t := n.tune()
	if done == nil {
		done = func() bool { return false }
	}
	n.visited = map[[2]int]bool{}
	n.failed = map[[2]int]int{}
	n.teleBad = map[[2]int]int{}
	n.teleFails = 0
	if n.fullyKnown() {
		return n.roam(ctx, done)
	}
	deadline := n.clk.Now().Add(t.ExploreLimit())
	var lastDir *input.Point // direction of the last leg, for the informed heading bonus
	legs := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if done() {
			return nil
		}
		if n.clk.Now().After(deadline) {
			return fmt.Errorf("%w: exploring for %v", ErrTimeout, t.ExploreLimit())
		}
		o, err := n.observeWait(ctx)
		if err != nil {
			return err
		}
		if err := n.guard(o); err != nil {
			return err
		}
		n.markVisited(o.pos)
		if n.hasTeleport() {
			handled, err := n.teleLeg(ctx, o, hint)
			if err != nil {
				return err
			}
			if handled {
				continue
			}
		}
		fr, ok := n.pickFrontier(o, hint, lastDir)
		if !ok {
			return ErrExplored
		}
		legs++
		n.log.Debug("nav explore leg", "n", legs, "from", o.pos, "to", fr.pos, "hops", fr.hops, "count", fr.count)
		start := o.pos
		stop := func(o *obs) bool {
			n.markVisited(o.pos)
			return done() || !isFrontier(o.lm, fr.pos)
		}
		err = n.walk(ctx, func(*obs) (input.Point, float64, error) { return fr.pos, 3, nil },
			walkOpts{stopWhen: stop, timeout: 90 * time.Second})
		switch {
		case err == nil:
		case errors.Is(err, ErrStuck), errors.Is(err, ErrNoPath), errors.Is(err, ErrTimeout):
			n.failed[fr.bucket]++
			n.log.Info("nav explore leg failed", "to", fr.pos, "err", err)
		default:
			return err
		}
		n.failed[fr.bucket]++ // inspected: a frontier that survives a visit is a dead end
		if cur, err := n.in.Pos(); err == nil {
			d := input.Point{X: cur.X - start.X, Y: cur.Y - start.Y}
			if d.X != 0 || d.Y != 0 {
				lastDir = &d
			}
		}
	}
}

func (n *Navigator) bucket(p input.Point) [2]int {
	b := max(2, int(n.tune().ExploreBucket))
	return [2]int{p.X / b, p.Y / b}
}

func (n *Navigator) markVisited(p input.Point) { n.visited[n.bucket(p)] = true }

// isFrontier reports whether the walkable cell p still has an unknown 4-neighbour inside the level.
func isFrontier(lm *d2map.LevelMap, p input.Point) bool {
	g := lm.Grid
	if !g.Walkable(p.X, p.Y) {
		return false
	}
	for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		x, y := p.X+d[0], p.Y+d[1]
		if !g.Known(x, y) && lm.Rect.Contains(x, y) {
			return true
		}
	}
	return false
}

// pickFrontier floods the walkable region from the player, groups reachable frontier cells into buckets and
// returns the best bucket's nearest cell. Greedy mode takes the nearest; informed mode prefers big frontiers and
// continuing in the same direction (or toward hint).
func (n *Navigator) pickFrontier(o *obs, hint *input.Point, lastDir *input.Point) (frontier, bool) {
	t := n.tune()
	f := n.fieldFor(o.lm)
	if f.W == 0 {
		return frontier{}, false
	}
	sx, sy, ok := f.nearestWalkable(o.pos.X, o.pos.Y, 8)
	if !ok {
		return frontier{}, false
	}
	dist := make([]int32, f.W*f.H)
	for i := range dist {
		dist[i] = -1
	}
	queue := make([]int32, 0, 1024)
	s := int32(f.idx(sx, sy))
	dist[s] = 0
	queue = append(queue, s)
	best := map[[2]int]*frontier{}
	for head := 0; head < len(queue); head++ {
		ci := queue[head]
		cx, cy := f.pt(ci)
		if f.hasUnknownNeighbour(cx, cy) {
			b := n.bucket(input.Point{X: cx, Y: cy})
			if !n.visited[b] && n.failed[b] < 1 {
				fr := best[b]
				if fr == nil {
					fr = &frontier{pos: input.Point{X: cx, Y: cy}, hops: int(dist[ci]), bucket: b}
					best[b] = fr
				}
				fr.count++
			}
		}
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
	var pick *frontier
	for _, fr := range best {
		fr.score = float64(fr.hops)
		if t.ExploreInformed {
			fr.score -= 1.5 * math.Min(float64(fr.count), 30)
			switch {
			case hint != nil:
				fr.score -= 0.5 * (math.Hypot(float64(hint.X-o.pos.X), float64(hint.Y-o.pos.Y)) -
					math.Hypot(float64(hint.X-fr.pos.X), float64(hint.Y-fr.pos.Y)))
			case lastDir != nil:
				dx, dy := float64(fr.pos.X-o.pos.X), float64(fr.pos.Y-o.pos.Y)
				ln, lm := math.Hypot(dx, dy), math.Hypot(float64(lastDir.X), float64(lastDir.Y))
				if ln > 0 && lm > 0 {
					fr.score -= 12 * (dx*float64(lastDir.X) + dy*float64(lastDir.Y)) / (ln * lm)
				}
			}
		}
		if pick == nil || fr.score < pick.score || fr.score == pick.score && fr.bucket[0] < pick.bucket[0] {
			pick = fr
		}
	}
	if pick == nil {
		return frontier{}, false
	}
	return *pick, true
}

// hasUnknownNeighbour is true when a 4-neighbour of the cell is unknown but inside the level rectangle.
func (f *Field) hasUnknownNeighbour(x, y int) bool {
	for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		if i := f.idx(x+d[0], y+d[1]); i >= 0 && f.flags[i]&fOpt != 0 {
			return true
		}
	}
	return false
}

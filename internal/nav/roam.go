package nav

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
)

// A world source that hands out whole level maps (d2dworld) implements FullyKnown. Explore then has no unknown
// cells to reveal and would report ErrExplored at once, so the area would count as cleared without a single
// monster met. roam defines coverage by visiting reachable ground instead (docs/d2d-pace.md section 'roam').
type fullyKnown interface{ FullyKnown() bool }

const (
	// roamSpacing and roamCover are the LEGACY point spacing and visit radius (the first-pass guess of 2026-10-30,
	// superseded by docs/roamspacing.md); the live values are the knobs nav.roamSpacing and nav.roamCover.
	roamSpacing = 7
	roamCover   = 5
	// roamPasses caps the legs of one level at roamPasses times the number of points, so it cannot roam forever.
	roamPasses = 3
)

// coverage is the roam state of one level of one game. It survives across Explore calls (the bot fights between
// legs and calls Explore again), and is rebuilt when the level map changes (new game).
type coverage struct {
	lm      *d2map.LevelMap // identity of the level in this game (d2dworld swaps the Grid inside it on refresh)
	level   int
	pts     []input.Point
	seen    []bool
	failed  []bool
	held    int // legs the mouth gate held on this level (ROAMLEG); not a target problem, see docs/roamleg.md
	legs    int
	spacing int // the grid of pts and the visit radius, fixed when the points were made (knobs read at that time)
	cover   int
	tgt     input.Point // the point of the leg being walked, for CoverageView only
	hasTgt  bool
	noted   bool // notedAt is the last position NotePosition used
	notedAt input.Point
}

// visited counts the seen points; covMu must be held.
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
	n.covMu.Lock()
	defer n.covMu.Unlock()
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
func samplePoints(f *Field, dist []int32, spacing int) []input.Point {
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
		b := [2]int{floorDiv(x, spacing), floorDiv(y, spacing)}
		c := abs(x-(b[0]*spacing+spacing/2)) + abs(y-(b[1]*spacing+spacing/2))
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

// dropNearExits keeps coverage points away from tile exits (caves, stairs): walking to a point next to the Den of Evil
// entrance entered the Den, the guard walked back, and the roam started over in a loop (D2DLOOP r2, V live).
//
// FIXMOUTH3: the filter radius is Euclidean 17, not 7 Chebyshev: the roam position 13 from the Cave Level 1 exit (V DRILLS17)
// was kept and its first click fired the trigger. 17 is above mouthApproachR and above exitAvoidR+2 (penalizeExits ignores
// an exit whose goal lies within that, so a target at 15 got no detour around the mouth: sim-reproduced). The target is
// filtered rather than every walk guarded out to 17; a coverage point near the mouth is not visited (monsters there wake
// up and come anyway).

const roamExitClear = exitAvoidR + 3

func dropNearExits(pts []input.Point, exits []d2map.Exit) []input.Point {
	if len(exits) == 0 {
		return pts
	}
	out := pts[:0:0]
next:
	for _, p := range pts {
		for _, e := range exits {
			if math.Hypot(float64(p.X-e.Pos.X), float64(p.Y-e.Pos.Y)) <= roamExitClear {
				continue next
			}
		}
		out = append(out, p)
	}
	return out
}

// roamParams are the spacing and visit radius from the knobs, clamped so cover <= spacing and both are >= 1.
func roamParams(t env.NavTuning) (spacing, cover int) {
	spacing = max(1, int(math.Round(t.RoamSpacing)))
	cover = min(spacing, max(1, int(math.Round(t.RoamCover))))
	return
}

// mark flags every point within the visit radius of p; covMu must be held. It reports whether any point changed.
func (c *coverage) mark(p input.Point) (changed bool) {
	for i, q := range c.pts {
		if !c.seen[i] && max(abs(q.X-p.X), abs(q.Y-p.Y)) <= c.cover {
			c.seen[i] = true
			changed = true
		}
	}
	return
}

func (n *Navigator) markPos(c *coverage, p input.Point) {
	n.covMu.Lock()
	c.mark(p)
	n.covMu.Unlock()
}

// selectCov makes the remembered coverage of lm's level the current one (nil when the level was not roamed in this
// game). The memory is per level and lives as long as the game: leaving for town, another zone or a portal trip and
// coming back keeps every point seen. A new game is detected by the level map identity: the world builds new
// LevelMap values per game, so an entry whose map differs is stale and dropped. Nav has no class or build data.
func (n *Navigator) selectCov(lm *d2map.LevelMap) {
	n.covMu.Lock()
	defer n.covMu.Unlock()
	c := n.covs[lm.No]
	if c != nil && c.lm != lm {
		delete(n.covs, lm.No)
		c = nil
	}
	n.cov = c
}

// SetOnCoverChange registers f, called after NotePosition changed the coverage (the minimap republishes it).
func (n *Navigator) SetOnCoverChange(f func()) { n.onCover = f }

// NotePosition marks the coverage points within the visit radius of pos as seen, whatever the hero is doing (fighting,
// picking up, travelling): ground passed during a detour counts as covered, so the roam does not walk back to it. It is
// safe to call from any goroutine, does nothing without a coverage of this level, and allocates nothing.
func (n *Navigator) NotePosition(level int, pos input.Point) {
	n.covMu.Lock()
	c := n.cov
	if c == nil || c.level != level || (c.noted && max(abs(pos.X-c.notedAt.X), abs(pos.Y-c.notedAt.Y)) < 1) {
		n.covMu.Unlock()
		return
	}
	c.noted, c.notedAt = true, pos
	changed := c.mark(pos)
	f := n.onCover
	n.covMu.Unlock()
	if changed && f != nil {
		f()
	}
}

// ErrRoamHeld ends a roam episode because the mouth gate held a leg. It matches ErrExplored, the result the old code
// reached after failing every target, so callers behave as before; Is(err, ErrRoamHeld) tells the two apart.
var ErrRoamHeld = fmt.Errorf("%w: roam leg held by the mouth gate", ErrExplored)

// RoamHeld reports the legs held by the mouth gate on the current level (0 when roaming is not in use).
func (n *Navigator) RoamHeld() int {
	n.covMu.Lock()
	defer n.covMu.Unlock()
	if n.cov == nil {
		return 0
	}
	return n.cov.held
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
		n.covMu.Lock()
		c := n.covs[o.lm.No]
		sp, cv := roamParams(t)
		if c == nil || c.lm != o.lm {
			pts := dropNearExits(samplePoints(f, dist, sp), o.lm.Exits)
			c = &coverage{lm: o.lm, level: o.level, spacing: sp, cover: cv, pts: pts, seen: make([]bool, len(pts)), failed: make([]bool, len(pts))}
			if n.covs == nil {
				n.covs = map[int]*coverage{}
			}
			n.covs[o.lm.No] = c
			n.log.Info("roam: coverage points", "level", o.level, "points", len(pts), "spacing", sp, "cover", cv, "exits", len(o.lm.Exits))
		}
		n.cov = c
		c.mark(o.pos)
		pick, bestD := -1, int32(0) // covMu is held through the pick
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
			vis, legs := c.visited(), c.legs
			n.covMu.Unlock()
			n.log.Info("roam: done", "visited", vis, "points", len(c.pts), "legs", legs)
			return ErrExplored
		}
		c.legs++
		target := c.pts[pick]
		c.tgt, c.hasTgt = target, true
		n.covMu.Unlock()
		stop := func(o *obs) bool {
			n.covMu.Lock()
			c.mark(o.pos)
			s := c.seen[pick]
			n.covMu.Unlock()
			return done() || s
		}
		err = n.walk(ctx, func(*obs) (input.Point, float64, error) { return target, 3, nil },
			walkOpts{stopWhen: stop, timeout: 90 * time.Second, pursuit: true})
		// done is a bot callback (it reads the bot's snapshot under obMu, which the observer holds while it calls
		// NotePosition): never call it with covMu held (deadlock found 2026-11-11). Lock order: covMu is a leaf.
		legDone := err == nil && done()
		n.covMu.Lock()
		c.hasTgt = false
		switch {
		case err == nil:
			if !legDone {
				c.seen[pick] = true
			}
			n.covMu.Unlock()
		case errors.Is(err, ErrMouthGate):
			// ROAMLEG: the gate refuses to move the character nearer the exit from where it stands. Every other target
			// would fail the same way, so end the episode and keep the target.
			c.held++
			held, legs := c.held, c.legs
			n.covMu.Unlock()
			n.log.Debug("roam: held by the mouth gate, ending the episode", "to", target, "held", held, "legs", legs)
			n.noteFail("gatehold", "Roam leg held by the mouth gate", target.X, target.Y)
			return fmt.Errorf("%w: roam held by the mouth gate at %v", ErrRoamHeld, o.pos)
		case errors.Is(err, ErrStuck), errors.Is(err, ErrNoPath), errors.Is(err, ErrTimeout):
			c.failed[pick] = true
			n.covMu.Unlock()
			n.log.Info("roam leg failed", "to", target, "err", err)
			n.noteFail("roamfail", "Roam leg failed: "+err.Error(), target.X, target.Y)
		default:
			n.covMu.Unlock()
			return err
		}
	}
}

// OnFailure, when set, is told of roam legs that failed or were held by the mouth gate (the watch minimap's failure
// markers). Passive: it runs on the navigator's goroutine and must not block or call back into the navigator.
func (n *Navigator) noteFail(kind, text string, x, y int) {
	if n.OnFailure != nil {
		n.OnFailure(kind, text, x, y)
	}
}

// SetOnFailure installs OnFailure (a method so wiring can reach it through an interface).
func (n *Navigator) SetOnFailure(fn func(kind, text string, x, y int)) { n.OnFailure = fn }

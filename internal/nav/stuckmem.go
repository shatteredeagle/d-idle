package nav

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"d2lod-bot/internal/input"
)

// Stuck-spot memory. LIVE (soak24 G1, 2026-10-30): the same walk in Burial Grounds (area 17) to the Cold Plains
// gate stalled at {5459 4549} in two different attempts hours apart and burned the whole ladder both times,
// because every walk started over with the same plan and the same blind side steps. A spot that stalled lately
// is remembered: the next stall there skips the re-click, blocks the cells ahead for planning (hard, not a cost)
// and looks for a way round with a spiral of nearby cells.
const (
	stuckMemory  = 10 * time.Minute // a stall repeated within this window counts as the same problem
	stuckRadius  = 10.0             // subtiles: stalls this close are the same spot
	hardBlockTTL = 3 * time.Minute  // how long cells ahead of a repeated stall stay impassable for the planner
)

type stuckSpot struct {
	level  int
	at     input.Point
	last   time.Time
	walk   int // walk sequence number of the last stall, so one walk's own retries do not count as a repeat
	hits   int
	tried  []input.Point // spiral cells already tried here
	method string        // what got the character moving last time ("" while unresolved)
}

// noteStuck records a stall at p and returns the spot, plus whether an earlier walk already stalled there.
func (n *Navigator) noteStuck(level int, p input.Point) (*stuckSpot, bool) {
	now := n.clk.Now()
	live := n.spots[:0]
	for _, s := range n.spots {
		if now.Sub(s.last) < stuckMemory {
			live = append(live, s)
		}
	}
	n.spots = live
	for _, s := range n.spots {
		if s.level == level && dist(s.at, p) <= stuckRadius {
			repeat := s.walk != n.walkSeq
			s.last, s.walk = now, n.walkSeq
			if repeat {
				s.hits++
			}
			return s, repeat
		}
	}
	s := &stuckSpot{level: level, at: p, last: now, walk: n.walkSeq, hits: 1}
	n.spots = append(n.spots, s)
	return s, false
}

// hardBlock makes the cells within r of p impassable for planning for hardBlockTTL (never a cell next to the character
// or the goal, so the walk can always leave and arrive).
func (n *Navigator) hardBlock(p input.Point, r float64) {
	n.hard = append(n.hard, blocked{at: pt{p.X, p.Y}, expires: n.clk.Now().Add(hardBlockTTL), r: r})
	n.log.Info("nav: blacklisted cell", "at", p, "radius", r, "ttlSec", int(hardBlockTTL.Seconds()))
}

// applyHard blocks the live hard cells on f, except within keep subtiles of start or goal.
func (n *Navigator) applyHard(f *Field, start, goal input.Point) int {
	now := n.clk.Now()
	live := n.hard[:0]
	applied := 0
	for _, b := range n.hard {
		if !b.expires.After(now) {
			continue
		}
		live = append(live, b)
		p := input.Point{X: b.at.x, Y: b.at.y}
		if dist(p, start) <= 3 || dist(p, goal) <= b.r+1 {
			continue
		}
		f.AddBlock(b.at.x, b.at.y, b.r)
		applied++
	}
	n.hard = live
	return applied
}

// spiral clicks a walkable, untried cell to one side of the heading (the obstacle is ahead; cells beyond it are
// behind the same wall), alternating sides on each call, nearer rings first. Returns false when nothing qualifies.
func (n *Navigator) spiral(o *obs, f *Field, s *stuckSpot, heading float64) bool {
	for side := 0; side < 2; side++ {
		sgn := 1.0
		if (len(s.tried)+side)%2 == 1 {
			sgn = -1
		}
		for _, r := range []float64{4, 6, 8, 11, 14} {
			for _, off := range []float64{90, 70, 110, 55, 125} {
				a := heading + sgn*off*math.Pi/180
				p := input.Point{X: o.pos.X + int(math.Round(r*math.Cos(a))), Y: o.pos.Y + int(math.Round(r*math.Sin(a)))}
				if !f.Walkable(p.X, p.Y) || f.Clearance(p.X, p.Y) < 1.5 || tried(s, p) || !f.los(o.pos.X, o.pos.Y, p.X, p.Y, 0) {
					continue
				}
				s.tried = append(s.tried, p)
				_ = n.in.ClickWorld(p)
				return true
			}
		}
	}
	return false
}

func tried(s *stuckSpot, p input.Point) bool {
	for _, q := range s.tried {
		if dist(p, q) < 3 {
			return true
		}
	}
	return false
}

// altGates returns approach cells along the same border gap as g, away from g.at: when the walk to the middle of
// a gap keeps stalling, the cells beside it may be reachable.
func (n *Navigator) altGates(o *obs, levelNo int, g gate) []gate {
	var out []gate
	off := input.Point{X: g.push.X - g.at.X, Y: g.push.Y - g.at.Y}
	for _, c := range o.lm.BorderCells(levelNo) {
		p := input.Point{X: c.X, Y: c.Y}
		if d := dist(p, g.at); d < 5 || d > 16 || n.gateIsBad(o.level, levelNo, p) {
			continue
		}
		out = append(out, gate{at: p, push: input.Point{X: p.X + off.X, Y: p.Y + off.Y}, border: true})
	}
	sort.Slice(out, func(i, j int) bool { return dist(o.pos, out[i].at) < dist(o.pos, out[j].at) })
	if len(out) > 2 {
		out = []gate{out[0], out[len(out)-1]}
	}
	return out
}

// exitUnstick runs after the walk to g.at ended in ErrStuck. It reports done when the level changed. Order:
// another approach cell of the same gap, then the Escape hook (town portal), after which the caller re-plans
// from wherever that led (ErrLevelChanged).
func (n *Navigator) exitUnstick(ctx context.Context, o *obs, levelNo int, g gate) (done bool, err error) {
	{
		n.markStalledGate(o.level, levelNo, g.at)
		for _, a := range n.altGates(o, levelNo, g) {
			n.log.Info("nav exit: other approach cell", "was", g.at, "now", a.at)
			cur, oerr := n.observeWait(ctx)
			if oerr != nil {
				return false, oerr
			}
			if cur.level != o.level {
				return false, nil
			}
			werr := n.walk(ctx, func(*obs) (input.Point, float64, error) { return a.at, 2, nil }, walkOpts{})
			if werr == nil && n.pushThrough(ctx, a, levelNo) {
				n.learn(o.level, levelNo)
				n.clk.Sleep(500 * time.Millisecond)
				return true, nil
			}
			if werr != nil && !errors.Is(werr, ErrStuck) && !errors.Is(werr, ErrNoPath) {
				return false, werr
			}
			n.markStalledGate(o.level, levelNo, a.at)
		}
	}
	if n.Escape != nil && !n.escaping {
		n.escaping = true
		defer func() { n.escaping = false }()
		n.log.Warn("nav: stuck at the exit twice, escaping by town portal", "level", o.level, "to", levelNo)
		if eerr := n.Escape(ctx); eerr != nil {
			return false, fmt.Errorf("%w: escape failed: %v", ErrStuck, eerr)
		}
		n.invalidate()
		if no := n.levelNow(); no != 0 && no != o.level {
			return false, fmt.Errorf("%w: escaped from level %d to %d after a repeated stall", ErrLevelChanged, o.level, no)
		}
	}
	return false, nil
}

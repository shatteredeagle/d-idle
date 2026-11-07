package nav

import (
	"fmt"
	"math"
	"time"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/input"
)

// FIXMOUTH4 (docs/fixmouth4.md): the cave-mouth guard is an invariant on every world click nav sends while a stay
// guard holds, not a filter per planner. fixmouth to fixmouth3 checked the plan at its start; the character was then
// moved by replans, the stuck ladder, fights inside a walk (Yield) and basic-attack swings, and the next unchecked click
// fired the trigger (V DRILLS18: a step-away replanned from 3.2 and its first click, 5.4 from the exit, fired it).
const (
	// mouthTriggerR: a click whose straight segment comes within this many subtiles of an unwanted exit may fire it.
	// V (DRILLS16 to 18 logs): click targets that fired the trigger lay 13.0 (Den), 12.1, 11.7 and 11.2 (Cave Level 1)
	// and 5.4 (Den) from the listed exit position; click targets 16.2 to 18 away never did. 15 covers 13.0 with 2 margin and stays below 16.2. The real
	// bound and shape of the trigger area (seed-dependent, docs/d2d-loop.md) are UNVERIFIED, as is whether the game's own
	// short path from a click bends into it.
	mouthTriggerR = 15.0
	// gateMinClip: a clipped click shorter than this is not worth sending; the gate holds instead.
	gateMinClip = 2.0
	// gateMaxRewrites rewritten clicks per walk, then the gate holds (the walk fails with ErrMouthGate): a goal behind
	// the mouth must not turn into a shuttle between the boundary and the away point.
	gateMaxRewrites = 8
	// gateUsefulClip: a clip shorter than this is only a crawl to the boundary; a detour around the disc is tried first.
	gateUsefulClip = 8.0
	// gateHoldEscalate: the same click (same from and to) refused this many times within gateHoldWindow means the walk
	// will not get out by itself. V (g2g logs, docs/research/g2g-diagnosis.md): one short step refused 40 to 150 times,
	// and a hero at (114,169) in the Burial Grounds and one at (87,287) in the Black Marsh left only through a restart.
	gateHoldEscalate = 3
	gateHoldWindow   = 120 * time.Second
	// gatePocketFor: after an escalation, holds from inside a trigger disc are answered with the pocket rule for this
	// long, so a walk out of the pocket does not need three refusals per step.
	gatePocketFor = 30 * time.Second
	// gateDetourMargin: the detour waypoint lies this far outside the bound.
	gateDetourMargin = 5.0
)

// ErrMouthGate: the click would cross the trigger area of an unwanted exit and no safe replacement exists. It wraps
// ErrNoPath, so roam and explore drop the target and the Fighter counts a failed approach.
var ErrMouthGate = fmt.Errorf("%w: the click would cross the trigger area of an unwanted exit", ErrNoPath)

// mouthGate wraps the controller of a Navigator: every ClickWorld passes the gate (New installs it).
type mouthGate struct {
	input.Controller
	n *Navigator
}

func (g mouthGate) ClickWorld(p input.Point) error {
	q, err := g.n.gateClick(g.Controller, p)
	if err != nil {
		return err
	}
	return g.Controller.ClickWorld(q)
}

func exitPoint(p d2map.Point) input.Point { return input.Point{X: p.X, Y: p.Y} }

// gatedExits lists the unwanted exits of the current level while the gate is active: not in town, and the stay guard
// holds for the level or a warp just brought us here. The config mouthGuard is not consulted: overworld levels list
// exits on their edges and gating them would clip walks along the edge for no observed failure.
func (n *Navigator) gatedExits() (*d2map.LevelMap, []input.Point) {
	if n.w == nil {
		return nil, nil
	}
	lm := n.w.Current()
	if lm == nil || data.IsTown(lm.No) || !(n.arrival || n.stayLevel == lm.No) {
		return nil, nil
	}
	var out []input.Point
	for _, e := range lm.Exits {
		if e.Dest != n.wantExit {
			out = append(out, exitPoint(e.Pos))
		}
	}
	return lm, out
}

// MouthClickOK reports whether a click at to, sent while the character stands at from, keeps the cave-mouth invariant.
// The Fighter asks it before a swing (a left click walks the character to the monster).
func (n *Navigator) MouthClickOK(from, to input.Point) bool {
	_, exits := n.gatedExits()
	return clickOKAll(from, to, exits)
}

// mouthClickOK is the invariant for one exit. The click target must lie outside the bound: every logged fired click had
// its target inside it, outward ones included (V: from 3.2 the click at 5.4 fired, DRILLS18; from 3.2 the second
// step-away waypoint at 12.1 fired, DRILLS17), while long step-away clicks to targets 20 and more away never fired. From
// outside the bound the straight segment must stay outside too (the game walks roughly that line); from inside
// (arrival offset, after a swing or knockback) it may only move outward (leavesExit).
func mouthClickOK(from, to, ex input.Point) bool {
	if dist(to, ex) < mouthTriggerR {
		return false
	}
	d0 := dist(from, ex)
	if d0 >= mouthTriggerR {
		return segDist(ex, from, to) >= mouthTriggerR
	}
	return leavesExit(from, to, d2map.Point{X: ex.X, Y: ex.Y}, d0)
}

func clickOKAll(from, to input.Point, exits []input.Point) bool {
	for _, e := range exits {
		if !mouthClickOK(from, to, e) {
			return false
		}
	}
	return true
}

// gateClick returns the click to send instead of p: p itself, a clipped or away-rewritten click, or ErrMouthGate.
func (n *Navigator) gateClick(ctl input.Controller, p input.Point) (input.Point, error) {
	lm, exits := n.gatedExits()
	if len(exits) == 0 {
		return p, nil
	}
	from, err := ctl.Pos()
	if err != nil {
		from = n.lastPos
	}
	if clickOKAll(from, p, exits) {
		return p, nil
	}
	if n.gateRewrites >= gateMaxRewrites {
		return n.hold(lm, from, p, exits, "rewrite cap")
	}
	q, how := n.gateRewrite(lm, from, p, exits)
	if how == "" {
		return n.hold(lm, from, p, exits, "no safe click")
	}
	n.gateRewrites++
	n.log.Info("nav mouth gate: click rewritten", "from", from, "to", p, "as", q, "how", how, "rewrites", n.gateRewrites)
	return q, nil
}

// gateRewrite: from outside the bound the click is clipped at the boundary; from inside it goes along the away vector
// (exit to character), turned by up to 90 degrees, 20, 25 then 15 subtiles, onto a walkable cell (the target must end
// outside the bound). how is "" when nothing
// passes the invariant.
func (n *Navigator) gateRewrite(lm *d2map.LevelMap, from, to input.Point, exits []input.Point) (q input.Point, how string) {
	ex, best := exits[0], math.MaxFloat64
	for _, e := range exits {
		if d := dist(from, e); !mouthClickOK(from, to, e) && d < best {
			ex, best = e, d
		}
	}
	d0 := dist(from, ex)
	if d0 >= mouthTriggerR {
		c, ok := clipOutside(from, to, ex)
		ok = ok && dist(from, c) >= gateMinClip && clickOKAll(from, c, exits)
		if ok && dist(from, c) >= gateUsefulClip {
			return c, "clip"
		}
		// A clip to the boundary only creeps up to the disc; the next click from there clips to nothing and the walk
		// failed (g2e: 104 / 295 / 47 refusals per seed). Go around the disc instead.
		if w, found := n.gateDetour(lm, from, to, ex, exits); found {
			return w, "detour"
		}
		if ok {
			return c, "clip"
		}
		return input.Point{}, ""
	}
	if d0 < 0.5 {
		return input.Point{}, ""
	}
	var f *Field
	if lm != nil {
		f = n.fieldFor(lm)
	}
	ux, uy := float64(from.X-ex.X)/d0, float64(from.Y-ex.Y)/d0
	// first pass: a straight walkable line (the character walks it as clicked); second pass: any walkable target
	for pass := 0; pass < 2; pass++ {
		for _, l := range []float64{mouthAway, mouthAway + 5, 15} {
			for _, deg := range []float64{0, 30, -30, 60, -60, 90, -90} {
				a := deg * math.Pi / 180
				vx, vy := ux*math.Cos(a)-uy*math.Sin(a), ux*math.Sin(a)+uy*math.Cos(a)
				c := input.Point{X: from.X + int(math.Round(vx*l)), Y: from.Y + int(math.Round(vy*l))}
				if f != nil {
					x, y, ok := f.nearestWalkable(c.X, c.Y, 3)
					if !ok || pass == 0 && !f.los(from.X, from.Y, x, y, 0) {
						continue
					}
					c = input.Point{X: x, Y: y}
				}
				if clickOKAll(from, c, exits) {
					return c, "away"
				}
			}
		}
		if f == nil {
			break
		}
	}
	return input.Point{}, ""
}

// clipOutside returns the last point of the segment from to that stays outside the bound around ex (half a subtile
// of margin), rounded to a cell that still passes the invariant.
func clipOutside(from, to, ex input.Point) (input.Point, bool) {
	wx, wy := float64(from.X-ex.X), float64(from.Y-ex.Y)
	vx, vy := float64(to.X-from.X), float64(to.Y-from.Y)
	a := vx*vx + vy*vy
	if a == 0 {
		return input.Point{}, false
	}
	b := 2 * (wx*vx + wy*vy)
	c := wx*wx + wy*wy - mouthTriggerR*mouthTriggerR
	disc := b*b - 4*a*c
	if disc < 0 {
		return input.Point{}, false
	}
	l := math.Sqrt(a)
	t := math.Min(1, (-b-math.Sqrt(disc))/(2*a)-0.5/l)
	for ; t > 0; t -= 0.25 / l {
		q := input.Point{X: from.X + int(math.Round(t*vx)), Y: from.Y + int(math.Round(t*vy))}
		if mouthClickOK(from, q, ex) {
			return q, true
		}
	}
	return input.Point{}, false
}

// gateDetour returns a waypoint on a ring just outside the bound around ex, reached from here without touching any
// gated disc, (the next click is gated again, so a long way
// round takes a few of them). It prefers waypoints from which the target is reachable by one more safe click. The ring
// is the shortest way around a disc; the side is chosen by the total length via the waypoint.
func (n *Navigator) gateDetour(lm *d2map.LevelMap, from, to, ex input.Point, exits []input.Point) (input.Point, bool) {
	d0 := dist(from, ex)
	if d0 < 0.5 || dist(to, ex) < mouthTriggerR {
		return input.Point{}, false // a target inside the bound is never reached by a detour
	}
	var f *Field
	if lm != nil {
		f = n.fieldFor(lm)
	}
	ux, uy := float64(from.X-ex.X)/d0, float64(from.Y-ex.Y)/d0
	r := mouthTriggerR + gateDetourMargin
	var best input.Point
	bestCost, bestThrough, found := math.MaxFloat64, false, false
	for _, deg := range []float64{25, 45, 65, 90, -25, -45, -65, -90} {
		a := deg * math.Pi / 180
		c := input.Point{
			X: ex.X + int(math.Round(r*(ux*math.Cos(a)-uy*math.Sin(a)))),
			Y: ex.Y + int(math.Round(r*(ux*math.Sin(a)+uy*math.Cos(a)))),
		}
		if f != nil {
			x, y, ok := f.nearestWalkable(c.X, c.Y, 3)
			if !ok {
				continue
			}
			c = input.Point{X: x, Y: y}
		}
		if dist(from, c) < gateMinClip || !clickOKAll(from, c, exits) {
			continue
		}
		through := clickOKAll(c, to, exits)
		cost := dist(from, c) + dist(c, to)
		if through && !bestThrough || through == bestThrough && cost < bestCost {
			best, bestCost, bestThrough, found = c, cost, through, true
		}
	}
	return best, found
}

func (n *Navigator) refuse() error {
	n.gateRefused.Add(1)
	return ErrMouthGate
}

// gateHoldState remembers the last refused click so a repeat can be told from a first refusal.
type gateHoldState struct {
	from, to    input.Point
	count       int
	at          time.Time
	pocketUntil time.Time
}

// hold answers a click the gate cannot rewrite. A first refusal stays a refusal (ErrMouthGate, the caller drops the
// target). The same click refused gateHoldEscalate times, or any hold while the pocket window is open, escalates when
// the hero stands inside the trigger disc of an unwanted exit (or the rewrite cap shuttled it there): the refusal then
// only repeats until the failsafe ladder restarts the game (g2g seeds 13 and 17). The pocket rule moves the hero
// without getting closer to any exit, and when no such cell exists (a dead end whose only way out passes the exit)
// sends the click itself: a fired trigger is a level change the walk already handles (ErrLevelChanged, bounce note),
// a restart is not. A target inside the bound with the hero outside is never escalated: that is a permanent refusal.
func (n *Navigator) hold(lm *d2map.LevelMap, from, p input.Point, exits []input.Point, why string) (input.Point, error) {
	now := n.clk.Now()
	h := &n.gateHolds
	if h.from == from && h.to == p && now.Sub(h.at) < gateHoldWindow {
		h.count++
	} else {
		h.from, h.to, h.count = from, p, 1
	}
	h.at = now
	inside := false
	for _, e := range exits {
		if dist(from, e) < mouthTriggerR {
			inside = true
		}
	}
	if (h.count >= gateHoldEscalate || now.Before(h.pocketUntil)) && (inside || why == "rewrite cap") {
		h.pocketUntil = now.Add(gatePocketFor)
		q, how := n.pocketClick(lm, from, p, exits)
		n.log.Info("nav mouth gate: pocket escape", "from", from, "to", p, "as", q, "how", how, "why", why, "holds", h.count)
		return q, nil
	}
	n.log.Info("nav mouth gate: click held", "from", from, "to", p, "why", why, "holds", h.count)
	return input.Point{}, n.refuse()
}

// pocketClick: the walkable cell 4 to 20 subtiles away, in line of sight, that gets no closer to any exit the hero is
// inside and is farthest from the nearest one; the requested click when there is none.
func (n *Navigator) pocketClick(lm *d2map.LevelMap, from, p input.Point, exits []input.Point) (input.Point, string) {
	var f *Field
	if lm != nil {
		f = n.fieldFor(lm)
	}
	var best input.Point
	bestScore, found := -math.MaxFloat64, false
	for _, l := range []float64{4, 8, 12, 20} {
		for deg := 0.0; deg < 360; deg += 22.5 {
			a := deg * math.Pi / 180
			c := input.Point{X: from.X + int(math.Round(l*math.Cos(a))), Y: from.Y + int(math.Round(l*math.Sin(a)))}
			if f != nil {
				x, y, ok := f.nearestWalkable(c.X, c.Y, 2)
				if !ok || !f.los(from.X, from.Y, x, y, 0) {
					continue
				}
				c = input.Point{X: x, Y: y}
			}
			if dist(from, c) < gateMinClip {
				continue
			}
			score, ok := math.MaxFloat64, true
			for _, e := range exits {
				d0 := dist(from, e)
				if d0 >= mouthTriggerR {
					continue
				}
				if d := segDist(e, from, c); d < d0-0.5 || dist(c, e) < d0 {
					ok = false
					break
				}
				score = math.Min(score, dist(c, e))
			}
			if ok && score > bestScore {
				best, bestScore, found = c, score, true
			}
		}
	}
	if found {
		return best, "away-sideways"
	}
	return p, "through"
}

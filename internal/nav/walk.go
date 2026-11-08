package nav

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"runtime"
	"strings"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
)

type walkOpts struct {
	ignoreUnit uint32           // unit id left out of the monster penalties
	stopWhen   func(*obs) bool  // walk ends successfully when true (checked every tick)
	timeout    time.Duration    // 0: tuning WalkTimeout
	onPlan     func(planResult) // test and debug hook
	escape     bool             // a stalled walk tries the Escape hook (town portal) before it gives up (GoTo, GoToUnit)
	oneClick   bool             // click the goal itself while it is in line of sight (the step away from a cave mouth, FIXMOUTH3)
	flee       *fleeState       // the step away from a cave mouth: re-checked on every replan (FIXMOUTH4)
}

// fleeState is the step-away's goal and the exit it flees. The walk's target reads goal, so a replan can move it.
type fleeState struct {
	exit      input.Point
	goal      input.Point
	retargets int
}

// fleeRetargets bounds how often one step-away picks a new goal after a replan.
const fleeRetargets = 2

type walkState struct {
	res      planResult
	have     bool
	planAt   time.Time
	planGoal input.Point
	sig      uint64
	cur      int
	base     int // waypoint the normal reach would have reached; the stuck ladder works from it, not from cur (WALKSMOOTH)

	lastClick   time.Time
	lastClickPt input.Point
	clicked     bool
	prev        input.Point
	still       int

	anchorPos  input.Point
	anchorTime time.Time
	rung       int
	side       int
	stuckAt    input.Point
	yields     int
	spot       *stuckSpot // memory of this stall (stuckmem.go)
	repeat     bool       // an earlier walk stalled at this spot too: use the alternative ladder
	method     string     // the last recovery tried; reported once the character moves again
	fights     int        // stuck-ladder fight hand-overs of this walk (stuckfight.go)
	why        string     // reason of the last plan, for the border log
	escTried   bool       // the Escape hook ran (or was refused) during this walk
}

// walk is the executor loop. tgt is evaluated every tick, so a moving target (GoToUnit) is followed.
func (n *Navigator) walk(ctx context.Context, tgt func(*obs) (input.Point, float64, error), wo walkOpts) (werr error) {
	t := n.tune()
	timeout := wo.timeout
	if timeout <= 0 {
		timeout = t.WalkTimeout()
	}
	deadline := n.clk.Now().Add(timeout)
	n.walkSeq++
	n.gateRewrites = 0
	if n.log.Enabled(ctx, slog.LevelDebug) {
		var pcs [6]uintptr
		k := runtime.Callers(2, pcs[:])
		fr := runtime.CallersFrames(pcs[:k])
		var who []string
		for f, more := fr.Next(); more && len(who) < 4; f, more = fr.Next() {
			who = append(who, f.Function)
		}
		n.log.Debug("nav walk", "callers", strings.Join(who, " < "))
	}
	var w walkState
	tr := &walkTrace{n: n}
	defer func() { tr.end(tr.last, n.clk.Now(), werr) }()
	mouthSeq := 0
	w.anchorTime = n.clk.Now()
	first := true
	startLevel := 0
	defer func() {
		if startLevel != 0 {
			n.walkLevel = startLevel // a walk that ended in another level (trigger fired) leaves the old one here
		}
	}()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		now := n.clk.Now()
		if now.After(deadline) {
			return fmt.Errorf("%w: walk did not finish in %v", ErrTimeout, timeout)
		}
		t = n.tune()
		obsAt := n.clk.Now()
		o, err := n.observeWait(ctx)
		if err != nil {
			return err
		}
		obsDur := n.clk.Now().Sub(obsAt)
		if err := n.guard(o); err != nil {
			return err
		}
		if startLevel == 0 {
			n.noteBorderBack(o.level)
			startLevel = o.level
			if n.walkLevel != 0 && n.walkLevel != o.level {
				n.arrival = true // the first walk in a level reached by a warp
			}
			n.walkLevel = o.level
		} else if o.level != startLevel {
			n.lastCross = crossNote{from: startLevel, to: o.level, at: now, reason: w.crossReason()}
			return fmt.Errorf("%w: %d -> %d", ErrLevelChanged, startLevel, o.level) // a tile fired early; coordinates of the old level mean nothing now
		}
		if wo.stopWhen != nil && wo.stopWhen(o) {
			return nil
		}
		if !n.inMouth && mouthSeq != n.walkSeq {
			mouthSeq = n.walkSeq
			away, ok := n.mouthStep(o)
			if ok {
				n.log.Info("nav: stepping away from a cave mouth", "from", o.pos, "to", away)
				n.inMouth = true
				fl := &fleeState{exit: input.Point{X: n.mouthLast.x, Y: n.mouthLast.y}, goal: away}
				err := n.walk(ctx, func(*obs) (input.Point, float64, error) { return fl.goal, 3, nil }, walkOpts{timeout: 10 * time.Second, oneClick: true, flee: fl})
				n.inMouth = false
				// FIXMOUTHLOOP2: the nested walk bumped walkSeq; without this the next iteration saw mouthSeq != walkSeq
				// and stepped away again, about 7 ms per round until the timeout (V DRILLS20). One attempt per walk.
				mouthSeq = n.walkSeq
				if errors.Is(err, ErrLevelChanged) {
					n.noteMouthBounce() // the same step again would fire the same trigger (FIXNAV)
				}
				if errors.Is(err, ErrLevelChanged) || errors.Is(err, ErrMouthGate) || ctx.Err() != nil {
					return err // the gate held: the caller (Fighter.move) backs off instead of walking from here again
				}
				continue
			}
		}
		goal, tol, err := tgt(o)
		if errors.Is(err, errNothingAhead) {
			n.clk.Sleep(t.Poll())
			continue
		}
		if err != nil {
			return err
		}
		if dist(o.pos, goal) <= tol {
			return nil
		}
		if w.have && !w.res.partial && w.res.snapped && dist(o.pos, w.res.goal) <= 1.5 {
			return nil // the goal itself is blocked: standing on the nearest walkable cell is as close as it gets
		}
		if first {
			w.prev, w.anchorPos, first = o.pos, o.pos, false
		}
		if o.snap.UI.AnyPanelOpen() {
			if err := n.in.CloseAll(); err != nil {
				return err
			}
			w.clicked = false
			n.clk.Sleep(t.Poll())
			continue
		}

		// --- plan
		replan := !w.have || dist(goal, w.planGoal) > math.Max(3, tol)
		why := "first" // why this plan was made, for the "nav plan" line (TELETUNE3: which trigger split two walks)
		if w.have && replan {
			why = "goal"
		}
		every := t.Replan()
		if w.res.optimistic {
			every = max(every/3, 400*time.Millisecond) // the plan crosses unknown ground: look again soon
		}
		if !replan && now.Sub(w.planAt) >= every {
			if o.lm.Grid.KnownCount() != w.res.known {
				replan, why = true, "map"
			} else if n.monsterSig(o, w.res.wps, w.cur) != w.sig {
				replan, why = true, "monsters"
			}
		}
		if !replan && deviation(o.pos, w.res.wps) > 2*t.WaypointSpacing+4 {
			replan, why = true, "deviation"
		}
		if w.have && w.res.partial && dist(o.pos, w.res.goal) <= 2.5 {
			if o.lm.Grid.KnownCount() == w.res.known {
				return fmt.Errorf("%w: reached the closest reachable point %v to %v", ErrNoPath, o.pos, goal)
			}
			replan, why = true, "partial"
		}
		if replan {
			planAt := n.clk.Now()
			res, err := n.plan(o.lm, o.pos, goal, tol, o.snap.Monsters, wo.ignoreUnit)
			if err != nil {
				return err
			}
			tr.plan(n.clk.Now().Sub(planAt), why, len(res.wps))
			w.res, w.have, w.planAt, w.planGoal, w.cur, w.base, w.why = res, true, now, goal, 0, 0, why
			w.sig = n.monsterSig(o, res.wps, 0)
			w.clicked = false
			if wo.onPlan != nil {
				wo.onPlan(res)
			}
			n.log.Debug("nav plan", "from", o.pos, "to", goal, "wps", len(res.wps), "partial", res.partial, "cells", res.cells, "why", why)
			if n.fleeRetarget(o, res, wo.flee) {
				w.have = false
				continue
			}
		}

		// --- follow: advance along the waypoints, but only while the next one is in line of sight. Cutting
		// the corner toward a waypoint behind a wall makes the character push against it.
		f := n.fieldFor(o.lm)
		sees := func(a, b input.Point) bool {
			if w.res.optimistic {
				f.passMask = fWalk | fOpt
				defer func() { f.passMask = fWalk }()
			}
			return f.los(a.X, a.Y, b.X, b.Y, 0)
		}
		reach := math.Min(6, math.Max(2, t.WaypointSpacing*0.5))
		for w.base < len(w.res.wps)-1 && dist(o.pos, w.res.wps[w.base]) <= reach && sees(o.pos, w.res.wps[w.base+1]) {
			w.base++
		}
		w.cur = max(w.cur, w.base)
		for w.cur < len(w.res.wps)-1 && dist(o.pos, w.res.wps[w.cur]) <= reach && sees(o.pos, w.res.wps[w.cur+1]) {
			w.cur++
		}
		if w.still >= 2 {
			// WALKSMOOTH: the character stood still for two polls. d2d ends a move short of its target (V: every
			// "nav stuck" line of a live Paladin session had the waypoint 5 to 6 subtiles away, beyond reach), and
			// a reclick of the same target does nothing. A waypoint within stopReach of a standing character is
			// reached: go on to the next one now instead of waiting for the 1.2 s stuck ladder.
			stop := math.Max(reach, t.WaypointSpacing*stopReachSpacings)
			for w.cur < len(w.res.wps)-1 && dist(o.pos, w.res.wps[w.cur]) <= stop && sees(o.pos, w.res.wps[w.cur+1]) {
				w.cur++
			}
		}
		if len(w.res.wps) == 0 {
			return emptyPathErr()
		}
		wp := w.res.wps[w.cur]
		if wo.oneClick && sees(o.pos, goal) {
			wp = goal // one long click: no waypoint click lands inside the trigger area on the way out
		}
		if !sees(o.pos, wp) {
			switch {
			case w.cur > 0 && sees(o.pos, w.res.wps[w.cur-1]):
				w.cur--
				w.base = min(w.base, w.cur)
				wp = w.res.wps[w.cur]
			case now.Sub(w.planAt) > 300*time.Millisecond: // off the path: plan again from here
				w.have = false
				continue
			}
		}
		wpStuck := w.res.wps[min(w.base, w.cur)] // the stuck ladder works from the normal waypoint, not from a lookahead or stop-short click
		if wo.oneClick {
			wpStuck = wp
		}
		if !wo.oneClick && sees(o.pos, wp) {
			wp = n.lookahead(o, &w, sees, t.WaypointSpacing)
		}
		moved := dist(o.pos, w.prev) > 0.25
		if moved {
			w.still = 0
		} else {
			w.still++
		}
		w.prev = o.pos
		tr.poll(o.pos, now, obsDur, w.still >= 2)
		if !w.clicked || wp != w.lastClickPt || now.Sub(w.lastClick) > 3*time.Second ||
			(w.still >= 2 && now.Sub(w.lastClick) >= 250*time.Millisecond) {
			reason := "retry"
			switch {
			case !w.clicked:
				reason = "first"
			case wp != w.lastClickPt:
				reason = "newwp"
			case w.still >= 2:
				reason = "still"
			}
			clickAt := n.clk.Now()
			if err := n.in.ClickWorld(wp); err != nil {
				return fmt.Errorf("nav: click %v: %w", wp, err)
			}
			tr.click(o.pos, wp, clickAt, reason, n.clk.Now().Sub(clickAt), w.cur, len(w.res.wps))
			w.clicked, w.lastClick, w.lastClickPt = true, now, wp
			n.log.Debug("nav click", "pos", o.pos, "wp", wp, "cur", w.cur, "of", len(w.res.wps))
		}

		// --- stuck ladder
		if dist(o.pos, w.anchorPos) >= 2.5 {
			w.anchorPos, w.anchorTime = o.pos, now
			if w.rung > 0 && dist(o.pos, w.stuckAt) > 12 { // a side step is not an escape: the ladder resets only after a real one
				w.rung = 0
			}
			if w.method != "" && dist(o.pos, w.stuckAt) >= 4 {
				n.log.Info("nav: stuck recovered", "at", w.stuckAt, "method", w.method, "rung", w.rung, "repeat", w.repeat)
				if w.spot != nil {
					w.spot.method = w.method
				}
				w.method = ""
			}
		} else if now.Sub(w.anchorTime) >= n.stuckAfter(t, &w, o.pos) {
			// a pocket (a pinned character, a sealed corner): the ladder moved us nowhere, leave by portal now
			if wo.escape && !w.escTried && w.rung >= pinnedRungs && dist(o.pos, w.stuckAt) < pinnedRadius {
				if err := n.walkEscape(ctx, o, &w); err != nil {
					return err
				}
			}
			if w.rung >= t.MaxRecoveries {
				if wo.escape && !w.escTried && dist(o.pos, goal) > tol*1.5 {
					if err := n.walkEscape(ctx, o, &w); err != nil {
						return err
					}
				}
				if dist(o.pos, goal) <= tol*1.5 {
					return nil // pushing against the target's own footprint: close enough
				}
				return fmt.Errorf("%w at %v going to %v (after %d recoveries)", ErrStuck, o.pos, goal, w.rung)
			}
			n.log.Info("nav stuck", "pos", o.pos, "wp", wp, "rung", w.rung)
			if done, err := n.stuckFight(ctx, o, &w); err != nil {
				return err
			} else if done {
				continue
			}
			if near := n.packNear(o); len(near) > 0 && n.Yield != nil && w.yields < maxYields {
				w.yields++
				n.log.Info("nav stalled by monsters, yielding", "pos", o.pos, "near", len(near))
				if err := n.Yield(ctx, near); err != nil {
					return err
				}
				n.invalidate()
				w.have, w.clicked, w.rung = false, false, 0
				w.anchorPos, w.anchorTime = o.pos, n.clk.Now()
				continue
			}
			if w.rung == 0 {
				w.stuckAt = o.pos
				if w.spot == nil || dist(w.spot.at, o.pos) > stuckRadius {
					w.spot, w.repeat = n.noteStuck(o.level, o.pos)
					if w.repeat {
						n.log.Info("nav: stuck again at the same spot", "pos", o.pos, "hits", w.spot.hits, "lastMethod", w.spot.method)
					}
				}
			}
			n.recover(o, &w, wpStuck)
			w.rung++
			if int32(w.rung) > n.peakRung.Load() {
				n.peakRung.Store(int32(w.rung))
			}
			w.anchorTime = n.clk.Now()
		}
		n.clk.Sleep(t.Poll())
	}
}

// stopReachSpacings and lookaheadSpacings (WALKSMOOTH, in waypoint spacings): a standing character counts a waypoint
// within stopReachSpacings x spacing as reached; the click goes to the farthest waypoint within lookaheadSpacings x
// spacing that the path leads straight to, so the next move is always issued before the previous one ends.
const (
	stopReachSpacings  = 0.8
	lookaheadSpacings  = 2.5
	lookaheadStraightR = 2.0 // intermediate waypoints may lie this far off the chord
)

// lookahead returns the waypoint to click: wps[cur], or a later one when the character sees it, it is not farther
// than lookaheadSpacings x spacing, the waypoints between lie on the chord (the chord is the planned path) and the
// mouth gate lets the click through. A click that ends before the next one is sent makes the character stand.
func (n *Navigator) lookahead(o *obs, w *walkState, sees func(a, b input.Point) bool, spacing float64) input.Point {
	wps := w.res.wps
	best := wps[w.cur]
	far := spacing * lookaheadSpacings
	_, exits := n.gatedExits()
	for j := w.cur + 1; j < len(wps) && dist(o.pos, wps[j]) <= far; j++ {
		straight := true
		for k := w.cur; k < j && straight; k++ {
			straight = segDist(wps[k], o.pos, wps[j]) <= lookaheadStraightR
		}
		if !straight || !sees(o.pos, wps[j]) || !clickOKAll(o.pos, wps[j], exits) {
			break
		}
		best = wps[j]
	}
	return best
}

// pinnedRungs and pinnedRadius: after this many rungs of the ladder with the character still within pinnedRadius
// (subtiles) of where it first stalled, it is in a pocket and the Escape hook runs before the remaining rungs.
const (
	pinnedRungs  = 3
	pinnedRadius = 3.0
)

// stuckAfter is how long the character may stand still before the next rung. Once the ladder has moved it nowhere
// (the rungs after the first, still within pinnedRadius of the first stall) the wait is halved: more rungs of the
// same ladder do not need the full wait to show that nothing moves. Game time (the navigator's clock).
func (n *Navigator) stuckAfter(t env.NavTuning, w *walkState, pos input.Point) time.Duration {
	d := t.Stuck()
	if w.rung >= 1 && dist(pos, w.stuckAt) < pinnedRadius {
		d /= 2
	}
	return d
}

// walkEscape runs the Escape hook once per walk. A new level ends the walk with ErrLevelChanged (the caller re-plans
// from there); a refused or failed escape (no portal, in town) is logged and the ladder carries on.
func (n *Navigator) walkEscape(ctx context.Context, o *obs, w *walkState) error {
	w.escTried = true
	if n.Escape == nil || n.escaping {
		return nil
	}
	n.escaping = true
	defer func() { n.escaping = false }()
	n.log.Warn("nav: stalled in a pocket, escaping by town portal", "level", o.level, "pos", o.pos, "rung", w.rung)
	if err := n.Escape(ctx); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		n.log.Warn("nav: escape failed, going on with the ladder", "err", err)
		return nil
	}
	n.invalidate()
	if no := n.levelNow(); no != 0 && no != o.level {
		return fmt.Errorf("%w: escaped from level %d to %d after a stall in a pocket", ErrLevelChanged, o.level, no)
	}
	return nil
}

// deviation is the distance from p to the nearest waypoint segment point (waypoints only; cheap and conservative).
func deviation(p input.Point, wps []input.Point) float64 {
	best := math.MaxFloat64
	for i, q := range wps {
		d := dist(p, q)
		if i > 0 {
			d = math.Min(d, segDist(p, wps[i-1], q))
		}
		best = math.Min(best, d)
	}
	return best
}

func segDist(p, a, b input.Point) float64 {
	ax, ay, bx, by := float64(a.X), float64(a.Y), float64(b.X), float64(b.Y)
	dx, dy := bx-ax, by-ay
	l2 := dx*dx + dy*dy
	if l2 == 0 {
		return dist(p, a)
	}
	t := math.Max(0, math.Min(1, ((float64(p.X)-ax)*dx+(float64(p.Y)-ay)*dy)/l2))
	return math.Hypot(float64(p.X)-(ax+t*dx), float64(p.Y)-(ay+t*dy))
}

// monsterSig is a cheap hash of the ids of living monsters close to the next stretch of path.
func (n *Navigator) monsterSig(o *obs, wps []input.Point, cur int) uint64 {
	r := n.tune().MonsterRadius
	if r <= 0 || len(wps) == 0 {
		return 0
	}
	end := min(len(wps), cur+6)
	var h uint64
	for i := range o.snap.Monsters {
		u := &o.snap.Monsters[i]
		if !u.Alive() {
			continue
		}
		up := input.Point{X: u.X, Y: u.Y}
		near := dist(up, o.pos) < r+2
		for j := cur; j < end && !near; j++ {
			near = dist(up, wps[j]) < r
		}
		if near {
			h ^= uint64(u.ID)*0x9E3779B97F4A7C15 + 1
		}
	}
	return h
}

// recover climbs one rung of the stuck ladder. Rungs: re-click, side step, block the cells ahead and replan,
// open a door, wide side step, random hop. Each rung acts, then the loop watches for progress again.
func (n *Navigator) recover(o *obs, w *walkState, wp input.Point) {
	heading := math.Atan2(float64(wp.Y-o.pos.Y), float64(wp.X-o.pos.X))
	if w.repeat && w.rung < 2 {
		// The plain ladder already failed here once: block the cells ahead for the planner and move to open ground
		// instead of re-clicking the same pixels.
		if w.rung == 0 {
			// what is in the way is just ahead; the longer it keeps stopping us, the wider the patch we avoid
			r := math.Min(7, 2.5+1.5*float64(w.spot.hits))
			k := math.Min(4, dist(o.pos, wp))
			n.hardBlock(input.Point{X: o.pos.X + int(math.Round(k*math.Cos(heading))), Y: o.pos.Y + int(math.Round(k*math.Sin(heading)))}, r)
		}
		if n.spiral(o, n.fieldFor(o.lm), w.spot, heading) {
			w.method = "spiral"
			w.have = false
			n.waitStill(2 * time.Second)
			return
		}
	}
	if w.rung > 0 { // whatever stopped us is somewhere ahead: make those cells expensive and plan around them
		for _, k := range []float64{2, 4} {
			p := pt{o.pos.X + int(math.Round(k*math.Cos(heading))), o.pos.Y + int(math.Round(k*math.Sin(heading)))}
			n.black = append(n.black, blocked{at: p, expires: n.clk.Now().Add(30 * time.Second)})
		}
		w.have = false
	}
	f := n.fieldFor(o.lm)
	pack := n.packNear(o)
	w.method = [...]string{"reclick", "sidestep", "replan", "door", "wide-sidestep", "hop"}[w.rung%6]
	switch w.rung % 6 {
	case 0:
		_ = n.in.ClickWorld(wp)
	case 1, 4:
		if len(pack) > 0 && n.retreat(f, o.pos, pack, w) {
			w.have = false
			break
		}
		d := 5.0
		if w.rung%6 == 4 {
			d = 10
		}
		n.sidestep(f, o.pos, heading, d, w)
		w.have = false
	case 2:
		w.have = false // the cells ahead are already blocked above
	case 3:
		n.tryDoor(o, heading)
		w.have = false
	case 5:
		for i := 0; i < 12; i++ {
			a := n.rng.Float64() * 2 * math.Pi
			r := 10 + n.rng.Float64()*4
			p := input.Point{X: o.pos.X + int(r*math.Cos(a)), Y: o.pos.Y + int(r*math.Sin(a))}
			if f.Walkable(p.X, p.Y) && f.Clearance(p.X, p.Y) >= 1 {
				_ = n.in.ClickWorld(p)
				break
			}
		}
		w.have = false
	}
	n.waitStill(2 * time.Second)
}

// waitStill sleeps until the character stops moving (or max passes), so a recovery step finishes before the
// loop plans again from a half-way position.
func (n *Navigator) waitStill(max time.Duration) {
	t := n.tune()
	end := n.clk.Now().Add(max)
	n.clk.Sleep(2 * t.Poll())
	prev, err := n.in.Pos()
	for err == nil && n.clk.Now().Before(end) {
		n.clk.Sleep(t.Poll())
		cur, err2 := n.in.Pos()
		if err2 != nil || dist(cur, prev) < 0.3 {
			return
		}
		prev = cur
	}
}

// sidestep clicks a walkable point d subtiles to one side of the heading, alternating sides.
func (n *Navigator) sidestep(f *Field, pos input.Point, heading, d float64, w *walkState) {
	for try := 0; try < 2; try++ {
		w.side = -w.side
		if w.side == 0 {
			w.side = 1
		}
		a := heading + float64(w.side)*math.Pi/2
		p := input.Point{X: pos.X + int(math.Round(d*math.Cos(a))), Y: pos.Y + int(math.Round(d*math.Sin(a)))}
		if f.Walkable(p.X, p.Y) {
			_ = n.in.ClickWorld(p)
			return
		}
	}
}

// tryDoor interacts with the closest door object in front of the player. UNVERIFIED live (no door was met in
// town or the Blood Moor); selection uses objects.txt IsDoor from the data package.
func (n *Navigator) tryDoor(o *obs, heading float64) {
	if n.e == nil || n.e.Data == nil {
		return
	}
	var best *d2.Unit
	bestD := 12.0
	for i := range o.snap.Objects {
		u := &o.snap.Objects[i]
		ob := n.e.Data.ObjectByID(u.Class)
		if ob == nil || !ob.IsDoor {
			continue
		}
		d := dist(o.pos, input.Point{X: u.X, Y: u.Y})
		if d < bestD {
			best, bestD = u, d
		}
	}
	if best == nil {
		return
	}
	tg := input.Target{Type: uint32(d2.TypeObject), ID: best.ID, Pos: input.Point{X: best.X, Y: best.Y}}
	if err := n.in.Interact(tg); err != nil {
		n.log.Info("nav door interact failed", "err", err)
	}
	n.clk.Sleep(600 * time.Millisecond)
	n.invalidate()
}

// CorpseApproach walks to a remembered corpse position with extra caution: paths keep twice the usual distance
// from monsters, and the walk stops with ErrDanger (naming the nearest monster in the error) as soon as a living
// monster is within CautionRadius of the character or of the corpse. Death during the walk is ErrDead. The
// caller decides what to do next (wait, fight, give up). Getting back to the level first is ToArea.
func (n *Navigator) CorpseApproach(ctx context.Context, corpse input.Point) error {
	t := n.tune()
	n.caution = 2
	defer func() { n.caution = 1 }()
	var danger error
	check := func(o *obs) bool {
		for i := range o.snap.Monsters {
			u := &o.snap.Monsters[i]
			if !u.Alive() || u.Type != d2.TypeMonster || (u.Monster != nil && n.isTownNPC(u)) {
				continue
			}
			up := input.Point{X: u.X, Y: u.Y}
			if d := math.Min(dist(up, o.pos), dist(up, corpse)); d <= t.CautionRadius {
				danger = fmt.Errorf("%w: monster class %d (id %d) %.0f subtiles from the walk", ErrDanger, u.Class, u.ID, d)
				return true
			}
		}
		return false
	}
	err := n.walk(ctx, func(*obs) (input.Point, float64, error) { return corpse, t.ArriveTolerance + 1, nil },
		walkOpts{stopWhen: check})
	if danger != nil {
		return danger
	}
	return err
}

// isTownNPC: NPCs per monstats.txt are harmless; without data every monster counts as dangerous.
func (n *Navigator) isTownNPC(u *d2.Unit) bool {
	if n.e == nil || n.e.Data == nil {
		return false
	}
	m := n.e.Data.MonsterByClassID(u.Class)
	return m != nil && m.NPC
}

// maxYields caps the combat yields of one walk.
const maxYields = 4

// packRadius is how close a living monster must be to count as the thing a stalled walk is stuck in.
const packRadius = 8.0

// packNear lists the living non-NPC monsters within packRadius of the character.
func (n *Navigator) packNear(o *obs) []d2.Unit {
	var out []d2.Unit
	for i := range o.snap.Monsters {
		u := &o.snap.Monsters[i]
		if !u.Alive() || u.Type != d2.TypeMonster || n.isTownNPC(u) {
			continue
		}
		if dist(input.Point{X: u.X, Y: u.Y}, o.pos) <= packRadius {
			out = append(out, *u)
		}
	}
	return out
}

// retreat walks away from the pack's centre (a walkable point 10, then 6, then 3 subtiles opposite to it). The
// pack cells are blacklisted so the next plan goes around. It reports whether a click was made.
func (n *Navigator) retreat(f *Field, pos input.Point, pack []d2.Unit, w *walkState) bool {
	var cx, cy float64
	for _, u := range pack {
		cx += float64(u.X)
		cy += float64(u.Y)
	}
	cx /= float64(len(pack))
	cy /= float64(len(pack))
	n.black = append(n.black, blocked{at: pt{int(cx), int(cy)}, expires: n.clk.Now().Add(30 * time.Second)})
	away := math.Atan2(float64(pos.Y)-cy, float64(pos.X)-cx)
	for _, d := range []float64{10, 6, 3} {
		for _, da := range []float64{0, math.Pi / 4, -math.Pi / 4} {
			a := away + da
			p := input.Point{X: pos.X + int(math.Round(d*math.Cos(a))), Y: pos.Y + int(math.Round(d*math.Sin(a)))}
			if f.Walkable(p.X, p.Y) && f.Clearance(p.X, p.Y) >= 1 {
				_ = n.in.ClickWorld(p)
				return true
			}
		}
	}
	return false
}

// mouthStep answers "arrived on a tile exit": d2d (V live, D2DLOOP) puts the character from a cave inside the warp's
// trigger area, which reaches about 2 tiles to the sides it faces, and any step inside fires it again (the bot then
// shuttled between the Blood Moor and the Den at 10 Hz). Walking straight away from the warp's position was safe on both
// seeds probed, so a walk that starts within mouthR subtiles of an exit it does not want first goes mouthAway subtiles that way.
//
// FIXMOUTH: the walk that starts right after a warp arrival uses mouthArrivalR instead. V (DRILLS14 logs, Den of Evil 4 of 4):
// the arrival (40,145) is 11.2 subtiles from the listed exit (35,135), so mouthR 9 skipped the step-away and the first roam
// click fired the trigger 12 ms later; Cave Level 1 arrives at the same (+5,+10) offset. Positions 16.2 to 18 from the exit
// did not fire (Cave Level 1 log). 16 covers the offset with a margin and stays UNVERIFIED as a trigger bound. A global
// mouthR of 14 was rejected: it would push every walk that passes within 14 subtiles of an exit and widen NearMouth for fights.
const (
	mouthR        = 9.0
	mouthArrivalR = 16.0
	mouthAway     = 20.0
	// mouthApproachR: the radius inside which a click may fire an unwanted exit (NearMouth, the roam target filter).
	// FIXMOUTH3, V (DRILLS16/17 logs): clicks fired the trigger 13.0 from the exit (Den, click (42,146), exit (35,135))
	// and 11.7 and 11.2 from it (Cave Level 1); clicks 16.2 to 18 away never did. 15 covers 13.0 with 2 margin and stays
	// below 16.2. The real trigger bound stays U (docs/fixmouth3.md).
	mouthApproachR = mouthTriggerR
)

// SetMouthGuard turns NearMouth on. d2d only (arrival inside a warp's trigger area is a d2d behavior, docs/d2d-loop.md);
// off by default so the Game.exe path is unchanged.
func (n *Navigator) SetMouthGuard(on bool) { n.mouthGuard = on }

// NearMouth reports whether p lies in the trigger area of a tile exit the walk does not want (within mouthR subtiles
// of its position). Fighters use it to not walk toward a monster standing at a cave mouth: the approach entered the cave,
// the bounce put the character back inside the trigger and the fight became a border shuttle (KILLFIX: Fallen packs
// at the Den mouth, 8 ms between "approach" and "Entered Den of Evil"; penalizeExits skips exits near the goal).
func (n *Navigator) NearMouth(p input.Point) bool {
	if n.w == nil {
		return false
	}
	lm := n.w.Current()
	if lm == nil || !(n.mouthGuard || n.avoiding() || n.arrival || n.stayLevel == lm.No) {
		return false
	}
	for _, e := range lm.Exits {
		if e.Dest == n.wantExit {
			continue
		}
		if math.Hypot(float64(p.X-e.Pos.X), float64(p.Y-e.Pos.Y)) <= mouthApproachR {
			return true
		}
	}
	return false
}

func (n *Navigator) mouthStep(o *obs) (input.Point, bool) {
	r := mouthR
	if n.arrival || n.stayLevel == o.level {
		r = mouthArrivalR
	}
	if n.log.Enabled(context.Background(), slog.LevelDebug) {
		n.log.Debug("nav mouth check", "pos", o.pos, "level", o.level, "exits", len(o.lm.Exits), "wantExit", n.wantExit, "arrival", n.arrival, "stay", n.stayLevel == o.level, "r", r)
	}
	inside := false
	for _, e := range o.lm.Exits {
		if e.Dest == n.wantExit {
			continue
		}
		dx, dy := float64(o.pos.X-e.Pos.X), float64(o.pos.Y-e.Pos.Y)
		d := math.Hypot(dx, dy)
		if n.log.Enabled(context.Background(), slog.LevelDebug) {
			n.log.Debug("nav mouth exit", "dest", e.Dest, "exit", e.Pos, "dist", math.Round(d*10)/10, "skip", d > r || d < 0.5)
		}
		if d > r || d < 0.5 {
			continue
		}
		inside = true
		if !data.IsTown(o.level) {
			// FIXMOUTH3: a level whose mouth the walker had to step away from keeps its guard for the whole stay (DRILLS17:
			// the one-shot arrival flag ended after the first step-away and the engage approach at 10.0 bounced).
			n.stayLevel = o.level
		}
		// FIXNAV: after a bounce (the step fired a trigger) the next attempt from this exit turns the direction: the
		// trigger area is seed-dependent (docs/d2d-loop.md), the same step repeated 12 times in the post-fix soak.
		key := mouthKey{o.level, e.Pos.X, e.Pos.Y}
		k := n.mouthBounces(key)
		n.mouthLast, n.mouthLastOK = key, true
		f := n.fieldFor(o.lm)
		base := mouthTurns[k%len(mouthTurns)] * math.Pi / 180
		// FIXMOUTH3: the turned direction, then its opposite when the straight path would cross the trigger.
		for _, ang := range []float64{base, base + math.Pi} {
			ux, uy := dx/d, dy/d
			ux, uy = ux*math.Cos(ang)-uy*math.Sin(ang), ux*math.Sin(ang)+uy*math.Cos(ang)
			tx, ty := o.pos.X+int(math.Round(ux*mouthAway)), o.pos.Y+int(math.Round(uy*mouthAway))
			x, y, ok := f.nearestWalkable(tx, ty, 6)
			if ok && leavesExit(o.pos, input.Point{X: x, Y: y}, e.Pos, d) && (k > 0 || math.Hypot(float64(x-e.Pos.X), float64(y-e.Pos.Y)) >= d+8) {
				return input.Point{X: x, Y: y}, true
			}
		}
		// Walls in the straight line (cave corridors): the nearest reachable cell that is at least 8 subtiles
		// farther from the warp. Without it the walk went on toward the warp and the cave shuttle began (KILLFIX).
		if p, found := n.mouthStepFlood(f, o.pos, e.Pos.X, e.Pos.Y, d); found {
			return p, true
		}
		n.log.Info("nav: next to a cave mouth, no way away found", "pos", o.pos, "exit", e.Pos, "dest", e.Dest, "dist", int(d))
	}
	if n.arrival && !inside {
		// FIXMOUTH2: the arrival radius holds for the whole stay (V DRILLS16: a roam walk 13 from the exit after Clear
		// had consumed the flag fired the trigger) and ends once a walk starts clear of every unwanted exit.
		n.arrival = false
	}
	return input.Point{}, false
}

// fleeRetarget: a step-away that replanned (after a stall, a fight inside it, knockback) checks the first click it
// would send from where the character stands NOW against the mouth invariant (mouthClickOK); when it breaks it, it
// computes a new away goal from here (FIXMOUTH4, V DRILLS18: the replan from 3.2 first clicked 5.4 from the exit and fired it). The mouth gate would
// rewrite that click anyway; this keeps the walk's own goal from fighting the gate.
func (n *Navigator) fleeRetarget(o *obs, res planResult, fl *fleeState) bool {
	if fl == nil || fl.retargets >= fleeRetargets || len(res.wps) == 0 {
		return false
	}
	wp := res.wps[0]
	for _, p := range res.wps {
		if dist(o.pos, p) > 1 {
			wp = p
			break
		}
	}
	if f := n.fieldFor(o.lm); f.los(o.pos.X, o.pos.Y, fl.goal.X, fl.goal.Y, 0) {
		wp = fl.goal // oneClick: the walk clicks the goal itself
	}
	d := dist(o.pos, fl.exit)
	if mouthClickOK(o.pos, wp, fl.exit) {
		return false
	}
	fl.retargets++
	p, ok := n.mouthStep(o)
	if !ok || p == fl.goal {
		return false
	}
	n.log.Info("nav: step-away replan heads to the mouth, new goal", "pos", o.pos, "wp", wp, "old", fl.goal, "new", p, "dist", math.Round(d*10)/10)
	fl.goal = p
	return true
}

// leavesExit: the straight path from a to b never gets nearer to the exit than a was (d, minus half a subtile of slack):
// it neither passes the exit nor starts by walking toward it. FIXMOUTH3 proof case: from (256,148), 3.2 from the exit
// (255,145), the second click fired the trigger.
func leavesExit(a, b input.Point, e d2map.Point, d float64) bool {
	ax, ay := float64(a.X), float64(a.Y)
	vx, vy := float64(b.X)-ax, float64(b.Y)-ay
	l2 := vx*vx + vy*vy
	t := 0.0
	if l2 > 0 {
		t = math.Max(0, math.Min(1, ((float64(e.X)-ax)*vx+(float64(e.Y)-ay)*vy)/l2))
	}
	return math.Hypot(ax+t*vx-float64(e.X), ay+t*vy-float64(e.Y)) >= d-0.5
}

// mouthStepFlood returns the walkable cell within mouthFlood hops of pos that lies at least 8 subtiles farther from
// the warp at ex than pos does, nearest first (ties: farthest from the warp).
func (n *Navigator) mouthStepFlood(f *Field, pos input.Point, exX, exY int, d float64) (input.Point, bool) {
	sx, sy, ok := f.nearestWalkable(pos.X, pos.Y, 4)
	if !ok {
		return input.Point{}, false
	}
	hops := floodHops(f, sx, sy)
	best, bestH, bestD := input.Point{}, int32(-1), 0.0
	for i, h := range hops {
		if h < 0 || h > mouthFlood || (bestH >= 0 && h > bestH) {
			continue
		}
		x, y := f.pt(int32(i))
		dd := math.Hypot(float64(x-exX), float64(y-exY))
		if dd < d+8 || f.Clearance(x, y) < 1 {
			continue
		}
		if bestH < 0 || h < bestH || dd > bestD {
			best, bestH, bestD = input.Point{X: x, Y: y}, h, dd
		}
	}
	return best, bestH >= 0
}

const mouthFlood = 40

// mouthTurns are the rotations (degrees) of the step-away direction, tried in order after each bounce.
var mouthTurns = []float64{0, 90, -90, 45, -45, 135, -135, 180}

type mouthKey struct{ level, x, y int }

type mouthFail struct {
	n  int
	at time.Time
}

const mouthFailTTL = 10 * time.Minute

func (n *Navigator) mouthBounces(k mouthKey) int {
	if f, ok := n.mouthFails[k]; ok && n.clk.Now().Sub(f.at) < mouthFailTTL {
		return f.n
	}
	return 0
}

// noteMouthBounce counts a step-away that ended in another level against the exit it fled from.
func (n *Navigator) noteMouthBounce() {
	if !n.mouthLastOK {
		return
	}
	if n.mouthFails == nil {
		n.mouthFails = map[mouthKey]mouthFail{}
	}
	n.mouthFails[n.mouthLast] = mouthFail{n.mouthBounces(n.mouthLast) + 1, n.clk.Now()}
}

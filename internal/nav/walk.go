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
	"d2lod-bot/internal/input"
)

type walkOpts struct {
	ignoreUnit uint32           // unit id left out of the monster penalties
	stopWhen   func(*obs) bool  // walk ends successfully when true (checked every tick)
	timeout    time.Duration    // 0: tuning WalkTimeout
	onPlan     func(planResult) // test and debug hook
}

type walkState struct {
	res      planResult
	have     bool
	planAt   time.Time
	planGoal input.Point
	sig      uint64
	cur      int

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
}

// walk is the executor loop. tgt is evaluated every tick, so a moving target (GoToUnit) is followed.
func (n *Navigator) walk(ctx context.Context, tgt func(*obs) (input.Point, float64, error), wo walkOpts) error {
	t := n.tune()
	timeout := wo.timeout
	if timeout <= 0 {
		timeout = t.WalkTimeout()
	}
	deadline := n.clk.Now().Add(timeout)
	n.walkSeq++
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
	mouthSeq := 0
	w.anchorTime = n.clk.Now()
	first := true
	startLevel := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		now := n.clk.Now()
		if now.After(deadline) {
			return fmt.Errorf("%w: walk did not finish in %v", ErrTimeout, timeout)
		}
		t = n.tune()
		o, err := n.observeWait(ctx)
		if err != nil {
			return err
		}
		if err := n.guard(o); err != nil {
			return err
		}
		if startLevel == 0 {
			startLevel = o.level
		} else if o.level != startLevel {
			return fmt.Errorf("%w: %d -> %d", ErrLevelChanged, startLevel, o.level) // a tile fired early; coordinates of the old level mean nothing now
		}
		if wo.stopWhen != nil && wo.stopWhen(o) {
			return nil
		}
		if !n.inMouth && mouthSeq != n.walkSeq {
			mouthSeq = n.walkSeq
			if away, ok := n.mouthStep(o); ok {
				n.log.Info("nav: stepping away from a cave mouth", "from", o.pos, "to", away)
				n.inMouth = true
				err := n.walk(ctx, func(*obs) (input.Point, float64, error) { return away, 3, nil }, walkOpts{timeout: 10 * time.Second})
				n.inMouth = false
				if errors.Is(err, ErrLevelChanged) || ctx.Err() != nil {
					return err
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
		every := t.Replan()
		if w.res.optimistic {
			every = max(every/3, 400*time.Millisecond) // the plan crosses unknown ground: look again soon
		}
		if !replan && now.Sub(w.planAt) >= every {
			replan = o.lm.Grid.KnownCount() != w.res.known || n.monsterSig(o, w.res.wps, w.cur) != w.sig
		}
		if !replan && deviation(o.pos, w.res.wps) > 2*t.WaypointSpacing+4 {
			replan = true
		}
		if w.have && w.res.partial && dist(o.pos, w.res.goal) <= 2.5 {
			if o.lm.Grid.KnownCount() == w.res.known {
				return fmt.Errorf("%w: reached the closest reachable point %v to %v", ErrNoPath, o.pos, goal)
			}
			replan = true
		}
		if replan {
			res, err := n.plan(o.lm, o.pos, goal, tol, o.snap.Monsters, wo.ignoreUnit)
			if err != nil {
				return err
			}
			w.res, w.have, w.planAt, w.planGoal, w.cur = res, true, now, goal, 0
			w.sig = n.monsterSig(o, res.wps, 0)
			w.clicked = false
			if wo.onPlan != nil {
				wo.onPlan(res)
			}
			n.log.Debug("nav plan", "from", o.pos, "to", goal, "wps", len(res.wps), "partial", res.partial, "cells", res.cells)
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
		for w.cur < len(w.res.wps)-1 && dist(o.pos, w.res.wps[w.cur]) <= reach && sees(o.pos, w.res.wps[w.cur+1]) {
			w.cur++
		}
		if len(w.res.wps) == 0 {
			return fmt.Errorf("%w: empty path", ErrNoPath)
		}
		wp := w.res.wps[w.cur]
		if !sees(o.pos, wp) {
			switch {
			case w.cur > 0 && sees(o.pos, w.res.wps[w.cur-1]):
				w.cur--
				wp = w.res.wps[w.cur]
			case now.Sub(w.planAt) > 300*time.Millisecond: // off the path: plan again from here
				w.have = false
				continue
			}
		}
		moved := dist(o.pos, w.prev) > 0.25
		if moved {
			w.still = 0
		} else {
			w.still++
		}
		w.prev = o.pos
		if !w.clicked || wp != w.lastClickPt || now.Sub(w.lastClick) > 3*time.Second ||
			(w.still >= 2 && now.Sub(w.lastClick) >= 250*time.Millisecond) {
			if err := n.in.ClickWorld(wp); err != nil {
				return fmt.Errorf("nav: click %v: %w", wp, err)
			}
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
		} else if now.Sub(w.anchorTime) >= t.Stuck() {
			if w.rung >= t.MaxRecoveries {
				if dist(o.pos, goal) <= tol*1.5 {
					return nil // pushing against the target's own footprint: close enough
				}
				return fmt.Errorf("%w at %v going to %v (after %d recoveries)", ErrStuck, o.pos, goal, w.rung)
			}
			n.log.Info("nav stuck", "pos", o.pos, "wp", wp, "rung", w.rung)
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
			n.recover(o, &w, wp)
			w.rung++
			w.anchorTime = n.clk.Now()
		}
		n.clk.Sleep(t.Poll())
	}
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
const (
	mouthR    = 9.0
	mouthAway = 20.0
)

// SetMouthGuard turns NearMouth on. d2d only (arrival inside a warp's trigger area is a d2d behavior, docs/d2d-loop.md);
// off by default so the Game.exe path is unchanged.
func (n *Navigator) SetMouthGuard(on bool) { n.mouthGuard = on }

// NearMouth reports whether p lies in the trigger area of a tile exit the walk does not want (within mouthR subtiles
// of its position). Fighters use it to not walk toward a monster standing at a cave mouth: the approach entered the cave,
// the bounce put the character back inside the trigger and the fight became a border shuttle (KILLFIX: Fallen packs
// at the Den mouth, 8 ms between "approach" and "Entered Den of Evil"; penalizeExits skips exits near the goal).
func (n *Navigator) NearMouth(p input.Point) bool {
	if !(n.mouthGuard || n.avoiding()) || n.w == nil {
		return false
	}
	lm := n.w.Current()
	if lm == nil {
		return false
	}
	for _, e := range lm.Exits {
		if e.Dest == n.wantExit {
			continue
		}
		if math.Hypot(float64(p.X-e.Pos.X), float64(p.Y-e.Pos.Y)) <= mouthR {
			return true
		}
	}
	return false
}

func (n *Navigator) mouthStep(o *obs) (input.Point, bool) {
	for _, e := range o.lm.Exits {
		if e.Dest == n.wantExit {
			continue
		}
		dx, dy := float64(o.pos.X-e.Pos.X), float64(o.pos.Y-e.Pos.Y)
		d := math.Hypot(dx, dy)
		if d > mouthR || d < 0.5 {
			continue
		}
		tx, ty := o.pos.X+int(math.Round(dx/d*mouthAway)), o.pos.Y+int(math.Round(dy/d*mouthAway))
		f := n.fieldFor(o.lm)
		x, y, ok := f.nearestWalkable(tx, ty, 6)
		if !ok || math.Hypot(float64(x-e.Pos.X), float64(y-e.Pos.Y)) < d+8 {
			// Walls in the straight line (cave corridors): the nearest reachable cell that is at least 8 subtiles
			// farther from the warp. Without it the walk went on toward the warp and the cave shuttle began (KILLFIX).
			if p, found := n.mouthStepFlood(f, o.pos, e.Pos.X, e.Pos.Y, d); found {
				return p, true
			}
			n.log.Info("nav: next to a cave mouth, no way away found", "pos", o.pos, "exit", e.Pos, "dest", e.Dest, "dist", int(d))
			continue
		}
		return input.Point{X: x, Y: y}, true
	}
	return input.Point{}, false
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

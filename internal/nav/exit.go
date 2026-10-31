package nav

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

// gate is a way into a neighbouring level: stand on at, then keep walking toward push.
type gate struct {
	at, push input.Point
	border   bool
}

// ExitTo leaves the current level for the neighbouring level levelNo. Outdoor links are borders: the cells of
// this level that touch the neighbour (d2map LevelMap.BorderCells; the Rogue Encampment's 15-cell gap to the
// Blood Moor at x=5960, y=4945..4959 is VERIFIED LIVE by MAP). Tile exits (LevelMap.Exits: caves, stairs) are
// UNVERIFIED on this client. If no way out is known yet, the level is explored until one shows up.
func (n *Navigator) ExitTo(ctx context.Context, levelNo int) error {
	prev := n.wantExit
	n.wantExit = levelNo
	defer func() { n.wantExit = prev }()
	t := n.tune()
	deadline := n.clk.Now().Add(2 * t.WalkTimeout())
	settled := false
	start := n.levelNow()
	if n.recentNoExit(start, levelNo) {
		return fmt.Errorf("%w: level %d from %d (nothing new since the last search)", ErrNoExit, levelNo, start)
	}
	fruitless := 0
	for attempt := 0; attempt < 8; attempt++ {
		if n.clk.Now().After(deadline) {
			return fmt.Errorf("%w: leaving for level %d", ErrTimeout, levelNo)
		}
		o, err := n.observeWait(ctx)
		if err != nil {
			return err
		}
		if err := n.guard(o); err != nil {
			return err
		}
		if o.level == levelNo {
			return nil
		}
		if start != 0 && o.level != start {
			// The walk crossed into a level that is neither where we started nor the target (VERIFIED LIVE: a border push
			// from the Monastery Gate fell back into the Tamoe Highland). Exploring THAT level for an exit to levelNo and
			// failing with ErrNoExit made ToArea mark the edge start->levelNo bad for the rest of the game.
			n.learn(start, o.level)
			return fmt.Errorf("%w: left level %d for %d but ended in %d", ErrLevelChanged, start, levelNo, o.level)
		}
		g, ok := n.findGate(o, levelNo)
		if !ok {
			n.log.Info("nav no exit known yet, exploring", "from", o.level, "to", levelNo)
			known := o.lm.Grid.KnownCount()
			explore := n.explore
			if n.exploreFn != nil {
				explore = n.exploreFn
			}
			err := explore(ctx, n.gateReachable(levelNo), nil)
			if err == nil {
				// Exploring claims success but findGate still finds no way (LIVE: 6800 retries in Cold Plains with the
				// character standing still): without movement or new map twice in a row there is nothing to find.
				if now, perr := n.observeWait(ctx); perr == nil && dist(now.pos, o.pos) < 3 && now.lm.Grid.KnownCount() == known {
					fruitless++
				} else {
					fruitless = 0
				}
				if fruitless >= 2 {
					n.markNoExit(o.level, levelNo, known)
					return fmt.Errorf("%w: level %d from %d (exploring made no progress)", ErrNoExit, levelNo, o.level)
				}
				continue
			}
			if errors.Is(err, ErrExplored) {
				// Right after an arrival the client may still be loading rooms: give the map one chance to grow.
				if !settled && n.mapGrows(ctx, 3*time.Second) {
					settled = true
					continue
				}
				n.markNoExit(o.level, levelNo, known)
				return fmt.Errorf("%w: level %d from %d", ErrNoExit, levelNo, o.level)
			}
			return err
		}
		n.log.Info("nav exit", "to", levelNo, "gate", g.at, "push", g.push, "border", g.border)
		if n.hasTeleport() && !g.border {
			if err := n.teleportToward(ctx, g.at, 10); err != nil {
				return err
			}
		}
		err = n.walk(ctx, func(*obs) (input.Point, float64, error) { return g.at, 2, nil }, walkOpts{})
		// Standing next to a tile exit (the return tile of the level we arrived by) the planner has nothing to walk: that
		// is "arrived", the push and step-off below take it from here (D2DLOOP r4, V live: Den of Evil, "no path: empty
		// path" five times in a row, then the failsafe ladder).
		if errors.Is(err, ErrNoPath) && !g.border && dist(o.pos, g.at) <= 6 && strings.Contains(err.Error(), "empty path") {
			err = nil
		}
		if err != nil {
			if errors.Is(err, ErrLevelChanged) { // the exit tile fired before we reached its middle
				if no := n.levelNow(); no != 0 {
					n.learn(o.level, no)
					if no == levelNo {
						n.clk.Sleep(500 * time.Millisecond)
						return nil
					}
					return fmt.Errorf("%w: left level %d for %d but ended in %d", ErrLevelChanged, o.level, levelNo, no)
				}
				continue
			}
			if errors.Is(err, ErrStuck) {
				if ok, uerr := n.exitUnstick(ctx, o, levelNo, g); ok {
					return nil
				} else if uerr != nil {
					return uerr
				}
			}
			return err
		}
		if n.pushThrough(ctx, g, levelNo) {
			n.learn(o.level, levelNo)
			n.clk.Sleep(500 * time.Millisecond) // let the new level's rooms load before anything acts
			return nil
		}
		if !g.border {
			// Arriving by a tile puts the character ON the return tile, and standing on it triggers nothing:
			// step off to a walkable cell a few subtiles away and walk back onto it.
			if off, ok := n.stepOff(o, g.at); ok {
				n.log.Info("nav exit tile did not fire, stepping off and back", "tile", g.at, "off", off)
				_ = n.GoTo(ctx, off)
				if err := n.walk(ctx, func(*obs) (input.Point, float64, error) { return g.at, 1, nil }, walkOpts{}); err == nil || errors.Is(err, ErrStuck) {
					if n.pushThrough(ctx, g, levelNo) {
						n.learn(o.level, levelNo)
						n.clk.Sleep(500 * time.Millisecond)
						return nil
					}
				}
			}
		}
		if !g.border && n.gateClicks(ctx, levelNo) {
			n.learn(o.level, levelNo)
			n.clk.Sleep(500 * time.Millisecond)
			return nil
		}
		n.markBadGate(o.level, levelNo, g.at) // this gap does not lead out (a wall cell the neighbour's grid did not show yet): try the next
	}
	return fmt.Errorf("%w: walking out to level %d did not change the level", ErrNoExit, levelNo)
}

// noExitMark records that a search for an exit from->to found nothing with this much of the level known.
type noExitMark struct {
	from, to, known int
	until           time.Time
}

const (
	exitCooldown   = 2 * time.Minute
	stalledGateTTL = 2 * time.Minute
)

func (n *Navigator) markNoExit(from, to, known int) {
	n.noExit = append(n.noExit, noExitMark{from, to, known, n.clk.Now().Add(exitCooldown)})
}

// recentNoExit: the same search failed lately and the map has not grown since, so repeating it only burns time and
// log lines; callers get ErrNoExit at once and can pick another area.
func (n *Navigator) recentNoExit(from, to int) bool {
	lm := n.w.Current()
	if lm == nil {
		return false
	}
	now := n.clk.Now()
	for _, m := range n.noExit {
		if m.from == from && m.to == to && now.Before(m.until) && m.known == lm.Grid.KnownCount() {
			return true
		}
	}
	return false
}

func hasGate(lm *d2map.LevelMap, levelNo int) bool {
	for _, e := range lm.Exits {
		if e.Dest == levelNo {
			return true
		}
	}
	return len(lm.BorderCells(levelNo)) > 0
}

// findGate picks the nearest reachable gate to levelNo.
func (n *Navigator) findGate(o *obs, levelNo int) (gate, bool) {
	var cands []gate
	for _, e := range o.lm.Exits {
		if e.Dest == levelNo {
			cands = append(cands, gate{at: input.Point{X: e.Pos.X, Y: e.Pos.Y}, push: input.Point{X: e.Pos.X, Y: e.Pos.Y}})
		}
	}
	cands = append(cands, n.borderGates(o.lm, levelNo)...)
	sort.Slice(cands, func(i, j int) bool { return dist(o.pos, cands[i].at) < dist(o.pos, cands[j].at) })
	for _, c := range cands {
		if n.gateIsBad(o.level, levelNo, c.at) {
			continue
		}
		res, err := n.plan(o.lm, o.pos, c.at, 3, o.snap.Monsters, 0)
		if err == nil && !res.partial && !res.optimistic {
			return c, true
		}
	}
	return gate{}, false
}

// borderGates groups the border cells toward levelNo into gaps and returns the middle of each.
func (n *Navigator) borderGates(lm *d2map.LevelMap, levelNo int) []gate {
	cells := lm.BorderCells(levelNo)
	if len(cells) == 0 {
		return nil
	}
	var adj *d2map.Adjacent
	for i := range lm.Adjacent {
		if lm.Adjacent[i].LevelNo == levelNo {
			adj = &lm.Adjacent[i]
		}
	}
	vertical := adj == nil || adj.Contact.W <= adj.Contact.H // strip is thin in x: the neighbour lies east or west
	push := n.tune().ExitPush
	var dx, dy float64
	if adj != nil {
		cx, cy := float64(adj.Rect.X)+float64(adj.Rect.W)/2, float64(adj.Rect.Y)+float64(adj.Rect.H)/2
		mx, my := float64(adj.Contact.X)+float64(adj.Contact.W)/2, float64(adj.Contact.Y)+float64(adj.Contact.H)/2
		if vertical {
			dx = math.Copysign(1, cx-mx)
		} else {
			dy = math.Copysign(1, cy-my)
		}
	}
	// A border cell is a gate only if the cell across is walkable too: a wide level can touch a narrow gap
	// (Blood Moor side of the Rogue Encampment gate: the wall cells beside the gap also touch the town).
	if nb := n.w.Level(levelNo); nb != nil && nb.Grid != nil && (dx != 0 || dy != 0) {
		kept := cells[:0:0]
		for _, c := range cells {
			ax, ay := c.X+int(dx), c.Y+int(dy)
			if nb.Grid.Known(ax, ay) && !nb.Grid.Walkable(ax, ay) {
				continue
			}
			kept = append(kept, c)
		}
		cells = kept
	}
	// order along the strip, split where cells are not contiguous
	sort.Slice(cells, func(i, j int) bool {
		if vertical {
			return cells[i].Y < cells[j].Y || cells[i].Y == cells[j].Y && cells[i].X < cells[j].X
		}
		return cells[i].X < cells[j].X || cells[i].X == cells[j].X && cells[i].Y < cells[j].Y
	})
	var gates []gate
	var run []d2map.Point
	flush := func() {
		if len(run) == 0 {
			return
		}
		m := run[len(run)/2]
		gates = append(gates, gate{at: input.Point{X: m.X, Y: m.Y},
			push: input.Point{X: m.X + int(dx*push), Y: m.Y + int(dy*push)}, border: true})
		run = nil
	}
	for i, c := range cells {
		if i > 0 {
			p := cells[i-1]
			gap := c.X - p.X
			if vertical {
				gap = c.Y - p.Y
			}
			if gap > 2 || gap < 0 {
				flush()
			}
		}
		run = append(run, c)
	}
	flush()
	return gates
}

// gateClicks is the last resort at a tile exit the character stands on without the level changing: plain screen
// clicks, no world projection. V live (ACT2LIVE6, tomb 69 gate at 12524,5285, a pillared cave mouth at a cliff foot):
// the tile never fires from a walk or a world click, but a click at the top left of the screen (moves the character a
// few subtiles in front of the pillars) followed by plain clicks up-left of the character (340,175) and (300,190)
// enters the cave. The points are screen points at 800x600 with the character near the centre; U for other exits.
func (n *Navigator) gateClicks(ctx context.Context, levelNo int) bool {
	for _, p := range [][2]int{{60, 100}, {340, 175}, {300, 190}} {
		if ctx.Err() != nil {
			return false
		}
		if err := n.in.Click(input.Left, p[0], p[1]); err != nil {
			return false
		}
		if n.waitArea(ctx, 3*time.Second, levelNo) == nil {
			return true
		}
	}
	return false
}

// pushThrough keeps walking toward g.push until the player's area is levelNo (or a few seconds pass).
func (n *Navigator) pushThrough(ctx context.Context, g gate, levelNo int) bool {
	t := n.tune()
	end := n.clk.Now().Add(6 * time.Second)
	for n.clk.Now().Before(end) && ctx.Err() == nil {
		_ = n.in.ClickWorld(g.push)
		n.clk.Sleep(5 * t.Poll())
		snap, err := n.st.Snapshot()
		if err == nil && snap.State == d2.StateInGame && snap.Player != nil && snap.Player.Area == levelNo {
			return true
		}
		if err == nil && snap.State != d2.StateInGame { // loading screen: a level change is under way
			if o, err := n.observeWait(ctx); err == nil && o.level == levelNo {
				return true
			}
		}
	}
	return false
}

// ToArea gets to level levelNo by the cheapest means known: the waypoint if the current level has one and the
// target is a waypoint level, else walking the chain of neighbouring levels from levels.txt (Vis columns,
// UNVERIFIED as a complete adjacency list). Crossing acts is not supported (ErrNoExit).
func (n *Navigator) ToArea(ctx context.Context, levelNo int) error {
	o, err := n.observeWait(ctx)
	if err != nil {
		return err
	}
	if o.level == levelNo {
		return nil
	}
	// A waypoint is only worth it for more than one hop: finding and walking to it costs a minute on an
	// unexplored level (VERIFIED LIVE: Cold Plains to Stony Field, the neighbour, spent 60 s on a waypoint that was
	// not even activated) and a one-hop walk is shorter.
	hops := 2
	if r, err := n.Route(o.level, levelNo); err == nil {
		hops = len(r)
	}
	if _, _, ok := waypointSlot(levelNo); ok && len(o.lm.Waypoints()) > 0 && hops > 1 {
		if err := n.TakeWaypoint(ctx, levelNo); err == nil {
			return nil
		} else if errors.Is(err, ErrDead) || errors.Is(err, ErrAborted) || ctx.Err() != nil {
			return err
		} else {
			n.log.Info("nav waypoint failed, walking", "to", levelNo, "err", err)
		}
	}
	cur := o.level
	for reroute := 0; reroute < 4; reroute++ {
		route, err := n.Route(cur, levelNo)
		if err != nil {
			return err
		}
		ok := true
		for _, next := range route {
			from := n.levelNow()
			if err := n.ExitTo(ctx, next); err != nil {
				if errors.Is(err, ErrLevelChanged) { // ended up somewhere else: plan again from there, nothing is bad
					if no := n.levelNow(); no != 0 {
						cur, ok = no, false
						break
					}
				}
				if errors.Is(err, ErrNoExit) && from != 0 && n.levelNow() == from {
					n.markBad(from, next) // the edge was a guess (hint or levels.txt) and does not exist: route around it
					cur, ok = from, false
					break
				}
				return err
			}
		}
		if ok {
			return nil
		}
	}
	return fmt.Errorf("%w: no working route to level %d", ErrNoExit, levelNo)
}

// stepOff finds a walkable cell 6 to 12 subtiles from p with a clear line back to it.
func (n *Navigator) stepOff(o *obs, p input.Point) (input.Point, bool) {
	f := n.fieldFor(o.lm)
	for r := 6; r <= 12; r += 3 {
		for k := 0; k < 16; k++ {
			a := float64(k) * math.Pi / 8
			q := input.Point{X: p.X + int(math.Round(float64(r)*math.Cos(a))), Y: p.Y + int(math.Round(float64(r)*math.Sin(a)))}
			if o.lm.Grid.Walkable(q.X, q.Y) && f.los(p.X, p.Y, q.X, q.Y, 0) {
				return q, true
			}
		}
	}
	return input.Point{}, false
}

// gateReachable returns an explore stop condition: true once a way to levelNo is known AND reachable over known
// cells (a tile preset can appear in a room far from anything walkable yet). The check reruns only when the
// known area grew.
func (n *Navigator) gateReachable(levelNo int) func() bool {
	lastKnown, last := -1, false
	return func() bool {
		lm := n.w.Current()
		if lm == nil || !hasGate(lm, levelNo) {
			return false
		}
		if kc := lm.Grid.KnownCount(); kc != lastKnown {
			lastKnown = kc
			snap, err := n.st.Snapshot()
			if err != nil || snap.State != d2.StateInGame || snap.Player == nil {
				return false
			}
			o := &obs{snap: snap, lm: lm, pos: input.Point{X: snap.Player.X, Y: snap.Player.Y}, level: snap.Player.Area}
			_, last = n.findGate(o, levelNo)
		}
		return last
	}
}

type badGate struct {
	from, to int
	at       input.Point
	until    time.Time // zero: for the rest of the game
}

func (n *Navigator) markBadGate(from, to int, at input.Point) {
	n.badGates = append(n.badGates, badGate{from, to, at, time.Time{}})
}

// markStalledGate is a bad gate for stalledGateTTL only: a walk that stalled on the way to it (a pack, a ghost
// obstacle) says nothing about whether the gap leads out.
func (n *Navigator) markStalledGate(from, to int, at input.Point) {
	n.badGates = append(n.badGates, badGate{from, to, at, n.clk.Now().Add(stalledGateTTL)})
}

// gateIsBad: a gate within 6 subtiles of one that already failed.
func (n *Navigator) gateIsBad(from, to int, at input.Point) bool {
	for _, b := range n.badGates {
		if b.from == from && b.to == to && dist(b.at, at) <= 6 && (b.until.IsZero() || n.clk.Now().Before(b.until)) {
			return true
		}
	}
	return false
}

// mapGrows waits up to d and reports whether the current level's known area changed in that time.
func (n *Navigator) mapGrows(ctx context.Context, d time.Duration) bool {
	lm := n.w.Current()
	if lm == nil {
		return false
	}
	start := lm.Grid.KnownCount()
	end := n.clk.Now().Add(d)
	for n.clk.Now().Before(end) && ctx.Err() == nil {
		n.clk.Sleep(n.tune().Poll())
		if _, err := n.observeWait(ctx); err != nil {
			return false
		}
		if cur := n.w.Current(); cur != nil && cur.Grid.KnownCount() != start {
			return true
		}
	}
	return false
}

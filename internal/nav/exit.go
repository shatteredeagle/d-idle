package nav

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

// timeoutProgress is how much nearer to the gate (subtiles) a timed out walk must have come for ExitTo to walk on.
const timeoutProgress = 30.0

// gate is a way into a neighbouring level: stand on at, then keep walking toward push.
type gate struct {
	at, push input.Point
	border   bool
	depth    int         // cells between the gate and the shared edge (0 on a proven strip): the shallowest frontier cell is tried first
	unproven bool        // an unproven strip: d2d has not linked the neighbour yet, so standing here is what links it
	exact    *[2]float64 // the point (d2d cells) a one-pixel Select box warp is taken from; see pushThrough
}

// exitWalkOpts: the gate is the goal of the walk toward it, so the unwanted-exit gate lets a click that stays outside the core through
// (see the call).
func exitWalkOpts(g gate) walkOpts { return goalOpts(g.at, true) }

// goalOpts is a walk toward p with p as its goal: every walk of an exit leg (the gate, its door, the strip before it) has
// one, or the unwanted-exit gate holds a corridor that passes the arrival stairs for good (Halls of the Dead Level 1 and
// Sewers Level 2, V drill.act2.* runs: the door walk was refused 100 times 13 subtiles from the stairs).
func goalOpts(p input.Point, pursuit bool) walkOpts { return walkOpts{pursuit: pursuit, goal: &p} }

// borderLinger is how long the character keeps pushing at an unproven border gate before the next one is tried: d2d links
// the neighbour while the character stands near its edge (lab devp27: after about 15 s of virtual time; live 2026-11-24: not
// within 40 s), and rotating to another strip at once walked the character away from the edge for good (lab seeds 3, 7, 11).
const borderLinger = 75 * time.Second

// lingerAtBorder pushes at g again and again, reading the level afresh each round, until the level changes, the border
// becomes readable (the caller then plans with the proven gate) or borderLinger passes. Each round logs what d2d shows.
func (n *Navigator) lingerAtBorder(ctx context.Context, g gate, levelNo int) bool {
	end := n.clk.Now().Add(borderLinger)
	for round := 1; n.clk.Now().Before(end) && ctx.Err() == nil; round++ {
		if r, ok := n.w.(interface{ Refetch() }); ok {
			r.Refetch()
		}
		if _, err := n.observeWait(ctx); err != nil {
			return false
		}
		if lm := n.w.Current(); lm != nil && round%3 == 1 {
			n.log.Info("nav: lingering at an unproven border", "to", levelNo, "at", g.at, "round", round, "pending", lm.Pending,
				"unproven", len(lm.Unproven), "adjacent", len(lm.Adjacent), "borderCells", len(lm.BorderCells(levelNo)))
		}
		if n.gateShown(levelNo) {
			return false
		}
		if n.pushThrough(ctx, g, levelNo) {
			return true
		}
	}
	return false
}

// exitMaxDoors bounds the doors one ExitTo call opens without counting them as attempts.
const exitMaxDoors = 24

// ExitTo leaves the current level for the neighbouring level levelNo. Outdoor links are borders: the cells of
// this level that touch the neighbour (d2map LevelMap.BorderCells; the Rogue Encampment's 15-cell gap to the
// Blood Moor at x=5960, y=4945..4959 is VERIFIED LIVE by MAP). Tile exits (LevelMap.Exits: caves, stairs) are
// UNVERIFIED on this client. If no way out is known yet, the level is explored until one shows up.
func (n *Navigator) ExitTo(ctx context.Context, levelNo int) error {
	prev := n.wantExit
	n.wantExit = levelNo
	defer func() { n.wantExit = prev }()
	t := n.tune()
	// The door budgets are per call: a level read again after a town trip has its doors shut again (V live, scenario
	// act1.tools.malus: the Monastery Gate doors were refused on the third visit and the step parked).
	n.doorTried, n.doorApproach = nil, nil
	deadline := n.clk.Now().Add(2 * t.WalkTimeout())
	settled := false
	start := n.levelNow()
	if n.recentNoExit(start, levelNo) && !n.gateShown(levelNo) {
		return fmt.Errorf("%w: level %d from %d (nothing new since the last search)", ErrNoExit, levelNo, start)
	}
	fruitless, approached, doors := 0, 0, 0
	// A door opened is progress, not a failed attempt: a maze level (Jail Level 1, V live 2026-11-24) shuts the stairs behind
	// eight doors in a row. The doors are bounded on their own (exitMaxDoors) and by the call's deadline.
	doorOpened := func() bool {
		doors++
		return doors <= exitMaxDoors
	}
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
		if !ok && n.openGateDoor(ctx, o, levelNo) {
			if doorOpened() {
				attempt--
			}
			continue
		}
		if !ok && n.badTTL() > 0 && n.fullyKnown() {
			// The map shows a way to levelNo (d2dworld hands out the whole level and its neighbours) but no plan reaches it
			// from here: the character stands in a pocket of the walkable grid. Exploring cannot find anything on a map
			// that is fully known, and an ErrNoExit would condemn the link for everyone (TOWNWEDGE). Walk to the nearest
			// cell that connects to the gate instead.
			if target, pocket := n.pocketTarget(o, levelNo); pocket {
				n.log.Info("nav: cut off from the gate, stepping to the nearest connected cell", "to", levelNo, "from", o.pos, "cell", target)
				ok, refused := n.leavePocket(ctx, o, target)
				if ok {
					continue
				}
				if refused {
					return fmt.Errorf("%w: gate to level %d is on the map but not reachable from %v (cause=exit_trigger: every way to a connected cell passes an unwanted exit's trigger area)", ErrNoPath, levelNo, o.pos)
				}
				return fmt.Errorf("%w: gate to level %d is on the map but not reachable from %v (cause=cut_off, no way out found)", ErrNoPath, levelNo, o.pos)
			}
		}
		if !ok && approached < 3 {
			// The map lists the neighbour as linked but cannot read across the edge from here (its collision is built near
			// the player): walk to its strip, the border shows up on the way. The map is fully known, so exploring finds nothing.
			if cell, found := n.unprovenTarget(o, levelNo); found {
				approached++
				n.log.Info("nav: border not readable from here, walking to its strip", "to", levelNo, "from", o.pos, "cell", cell)
				err := n.walk(ctx, func(*obs) (input.Point, float64, error) { return cell, 3, nil }, goalOpts(cell, true))
				if err != nil && !errors.Is(err, ErrAlreadyThere) {
					if errors.Is(err, ErrLevelChanged) {
						if no := n.levelNow(); no != 0 {
							n.learn(o.level, no)
							if no == levelNo {
								return nil
							}
						}
					}
					return err
				}
				continue
			}
		}
		if !ok && approached > 0 {
			// The strip was walked to and d2d still cannot read across it (the neighbour is linked, pending stays 1: Barracks
			// back to the Outer Cloister, Proving Ground nav.barracks-return 2026-11-23, both classes). The border is crossed by
			// walking over it, so the unproven strip cells are the gate: the push beyond them changes the level.
			all := n.gatesFromCells(o.lm, levelNo, o.lm.UnprovenCells(levelNo), o.lm.Unproven)
			us := all[:0:0]
			for _, c := range all {
				if !n.gateIsBad(o.level, levelNo, c.at) { // a strip that did not lead out is not tried again at once: rotate along the border
					us = append(us, c)
				}
			}
			if len(us) > 0 {
				// the shallowest frontier first (a notch in the wall is where the way out is), then the nearest
				sort.Slice(us, func(i, j int) bool {
					if us[i].depth != us[j].depth {
						return us[i].depth < us[j].depth
					}
					return dist(o.pos, us[i].at) < dist(o.pos, us[j].at)
				})
				// a sealed pocket of the frontier (a cell no plan reaches) sorts first as often as not: take the first reachable one
				pick := 0
				for i, c := range us {
					if res, err := n.plan(o.lm, o.pos, c.at, 3, o.snap.Monsters, 0); err == nil && !res.partial && !res.optimistic {
						pick = i
						break
					}
				}
				g, ok = us[pick], true
				g.unproven = true
				n.log.Info("nav: crossing an unproven border strip", "to", levelNo, "strip", g.at, "push", g.push, "gates", len(us), "tried", len(all)-len(us), "scan", o.lm.BorderNote)
			}
		}
		if !ok {
			// The facts behind "no gate": what the map shows toward levelNo and why none is usable (card andariel1).
			n.log.Info("nav no usable gate", "from", o.level, "to", levelNo, "exits", len(o.lm.Exits), "adjacent", len(o.lm.Adjacent),
				"borderCells", len(o.lm.BorderCells(levelNo)), "unproven", len(o.lm.UnprovenCells(levelNo)), "pending", o.lm.Pending,
				"cands", len(n.gateCands(o, levelNo)), "known", o.lm.Grid.KnownCount(), "fullyKnown", n.fullyKnown(), "badTTL", n.badTTL() > 0)
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
				// A border that d2dworld still probes (a linked neighbour whose collision is not built yet) is late, not
				// missing: andariel1, Monastery Gate to Outer Cloister, "no exit" 0.2 s after arrival with pending 1.
				if n.pendingGate(ctx, o.lm, levelNo) {
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
		// The gate is the walk's goal: a corridor that passes the stairs of the level we came by (Sewers Level 2, 11
		// subtiles from the arrival stairs) must not be refused for good by the unwanted-exit gate (mouthgate.go).
		err = n.walk(ctx, func(*obs) (input.Point, float64, error) { return g.at, 2, nil }, exitWalkOpts(g))
		// Standing next to a tile exit (the return tile of the level we arrived by) the planner has nothing to walk: that
		// is "arrived", the push and step-off below take it from here (D2DLOOP r4, V live: Den of Evil, "no path: empty
		// path" five times in a row, then the failsafe ladder).
		if errors.Is(err, ErrAlreadyThere) && !g.border && dist(o.pos, g.at) <= 6 {
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
			if errors.Is(err, ErrNoPath) && n.doorAfterShortWalk(ctx, levelNo) {
				if doorOpened() {
					attempt--
				}
				continue
			}
			// A long outdoor leg with fights on the way outlasts one walk (V drill.act3s.lam_esen: 513 subtiles of path, 330 net in
			// 2 minutes, the board parked the step after three such timeouts): a walk that timed out but came 30 subtiles nearer
			// the gate goes on; the call's own deadline and attempt count still bound it.
			if errors.Is(err, ErrTimeout) {
				if cur, oerr := n.observeWait(ctx); oerr == nil && cur.level == o.level && dist(cur.pos, g.at) < dist(o.pos, g.at)-timeoutProgress {
					continue
				}
			}
			return err
		}
		if g.unproven && n.openDoorAtGate(ctx, o, g, levelNo) {
			continue // a door shuts this gate: it is open now (or was tried); plan again with the new grid
		}
		if n.pushThrough(ctx, g, levelNo) || g.unproven && n.lingerAtBorder(ctx, g, levelNo) {
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
		if ctx.Err() != nil {
			// The caller took the hero away (the point spender, a pause) while it stood on the tile: that says nothing about
			// the gap (V drill.act3s.lam_esen.bazaar: the Ruined Temple stairs were marked bad at the tile, the next three
			// tries found no gate and the chunk parked).
			return ctx.Err()
		}
		if g.border && n.pendingGate(ctx, n.w.Current(), levelNo) {
			continue // the neighbour was still being linked: its border shows now, no gate is condemned for it
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

// gateShown reports that the current level map shows a way to levelNo now (an earlier "nothing found" mark is stale then:
// the map did not grow, but its borders did).
func (n *Navigator) gateShown(levelNo int) bool {
	lm := n.w.Current()
	return lm != nil && lm.Grid != nil && hasGate(lm, levelNo)
}

// pendingWait bounds the wait for a late border (nav clock: virtual in a lockstep run, which steps the game).
const pendingWait = 10 * time.Second

// unprovenPlans bounds the plans unprovenTarget tries.
const unprovenPlans = 60

// unprovenTarget picks the nearest strip cell toward levelNo (see LevelMap.Unproven) the planner can reach.
func (n *Navigator) unprovenTarget(o *obs, levelNo int) (input.Point, bool) {
	cells := o.lm.UnprovenCells(levelNo)
	sort.Slice(cells, func(i, j int) bool {
		return math.Hypot(float64(cells[i].X-o.pos.X), float64(cells[i].Y-o.pos.Y)) < math.Hypot(float64(cells[j].X-o.pos.X), float64(cells[j].Y-o.pos.Y))
	})
	// Neighbours of a cell that was planned plan alike, so each is skipped; a rank-based sample (every fifth cell) could
	// skip all four cells of a real gap when a sealed pocket of the frontier sorts before them (barracks-return, 131 cells).
	var tried []input.Point
	for _, c := range cells {
		p := input.Point{X: c.X, Y: c.Y}
		near := false
		for _, t := range tried {
			if abs(t.X-p.X) <= 2 && abs(t.Y-p.Y) <= 2 {
				near = true
				break
			}
		}
		if near {
			continue
		}
		if tried = append(tried, p); len(tried) > unprovenPlans {
			break
		}
		res, err := n.plan(o.lm, o.pos, p, 3, o.snap.Monsters, 0)
		if err == nil && !res.partial && !res.optimistic {
			return p, true
		}
	}
	return input.Point{}, false
}

// pendingGate waits while the level still has borders it cannot read, for a gate to levelNo to show up. It reports
// whether one did.
func (n *Navigator) pendingGate(ctx context.Context, lm *d2map.LevelMap, levelNo int) bool {
	if lm == nil || lm.Pending == 0 {
		return false
	}
	end := n.clk.Now().Add(pendingWait)
	for n.clk.Now().Before(end) && ctx.Err() == nil {
		n.clk.Sleep(n.tune().Poll())
		if _, err := n.observeWait(ctx); err != nil {
			return false
		}
		if n.gateShown(levelNo) {
			return true
		}
		if cur := n.w.Current(); cur == nil || cur.Pending == 0 {
			return false
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
	for _, c := range n.gateCands(o, levelNo) {
		res, err := n.plan(o.lm, o.pos, c.at, 3, o.snap.Monsters, 0)
		if err == nil && !res.partial && !res.optimistic {
			return c, true
		}
	}
	return gate{}, false
}

// gateCands lists the usable (not marked bad) gates to levelNo on the current map, nearest first.
func (n *Navigator) gateCands(o *obs, levelNo int) []gate {
	var cands []gate
	for _, e := range o.lm.Exits {
		if e.Dest == levelNo {
			g := gate{at: input.Point{X: e.Pos.X, Y: e.Pos.Y}, push: input.Point{X: e.Pos.X, Y: e.Pos.Y}}
			if e.HasClick { // the push click must land in the warp's Select box (d2dctl.Warp)
				g.push = input.Point{X: e.Click.X, Y: e.Click.Y}
			}
			if e.HasExact {
				ex := e.Exact
				g.exact = &ex
			}
			cands = append(cands, g)
		}
	}
	cands = append(cands, n.borderGates(o.lm, levelNo)...)
	sort.Slice(cands, func(i, j int) bool { return dist(o.pos, cands[i].at) < dist(o.pos, cands[j].at) })
	out := cands[:0:0]
	for _, c := range cands {
		if !n.gateIsBad(o.level, levelNo, c.at) {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		// Only dead gates are left: trying one beats exploring a level whose map is known.
		for _, c := range cands {
			if !n.gateBad(o.level, levelNo, c.at, false) {
				out = append(out, c)
			}
		}
	}
	return out
}

// borderGates groups the border cells toward levelNo into gaps and returns the middle of each.
func (n *Navigator) borderGates(lm *d2map.LevelMap, levelNo int) []gate {
	return n.gatesFromCells(lm, levelNo, lm.BorderCells(levelNo), lm.Adjacent)
}

// gatesFromCells is borderGates for any set of strip cells; adjacent says where the neighbour lies (the proven list, or the
// unproven one when d2d cannot read across the edge from here).
func (n *Navigator) gatesFromCells(lm *d2map.LevelMap, levelNo int, cells []d2map.Point, adjacent []d2map.Adjacent) []gate {
	if len(cells) == 0 {
		return nil
	}
	var adj *d2map.Adjacent
	for i := range adjacent {
		if adjacent[i].LevelNo == levelNo {
			adj = &adjacent[i]
		}
	}
	vertical := adj == nil || adj.Contact.W <= adj.Contact.H // strip is thin in x: the neighbour lies east or west
	if len(cells) > 1 {
		// the cells decide, not the last contact rectangle: a frontier of cells at several depths is wide along the edge
		minX, maxX, minY, maxY := cells[0].X, cells[0].X, cells[0].Y, cells[0].Y
		for _, c := range cells {
			minX, maxX, minY, maxY = min(minX, c.X), max(maxX, c.X), min(minY, c.Y), max(maxY, c.Y)
		}
		vertical = maxX-minX <= maxY-minY
	}
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
	// The cell across is looked up in the neighbour's own grid, so it is moved from this level's frame to the
	// neighbour's: adj.Rect is where the neighbour lies in our frame, nb.Rect its own frame (zero origin without
	// d2dworld's OriginTiles). Seed 17 (docs/navexit22.md): the Blood Moor's gate cells were tested against the
	// camp's wall column, every cell was dropped and the walk to town failed for good.
	if nb := n.w.Level(levelNo); nb != nil && nb.Grid != nil && adj != nil && (dx != 0 || dy != 0) {
		kept := cells[:0:0]
		for _, c := range cells {
			ax, ay := c.X+int(dx)-adj.Rect.X+nb.Rect.X, c.Y+int(dy)-adj.Rect.Y+nb.Rect.Y
			if nb.Grid.Known(ax, ay) && !nb.Grid.Walkable(ax, ay) {
				continue
			}
			kept = append(kept, c)
		}
		if len(kept) > 0 {
			cells = kept
		}
		// else: every cell looks blocked across the edge in the remembered grid of the neighbour. That grid was read from another side or
		// with a door shut, or lies a few cells off, so it is not allowed to refuse the only gates there are (V live 2026-11-24 20:35Z, the
		// second crossing of the Inner Cloister to the Cathedral: borderCells 3, cands 0, "no exit to that level" three times)
	}
	// depthOf is how far a frontier cell lies inside the level from the shared edge (0 on a proven strip)
	depthOf := func(c d2map.Point) int {
		if adj == nil {
			return 0
		}
		switch {
		case dy < 0:
			return max(c.Y-adj.Rect.Bottom(), 0)
		case dy > 0:
			return max(adj.Rect.Y-1-c.Y, 0)
		case dx < 0:
			return max(c.X-adj.Rect.Right(), 0)
		case dx > 0:
			return max(adj.Rect.X-1-c.X, 0)
		}
		return 0
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
		px, py := m.X+int(dx*push), m.Y+int(dy*push)
		if adj != nil {
			// a cell deeper inside than the edge row (the frontier of an unproven strip) pushes from the neighbour's edge, so the
			// push lands inside the neighbour, not on the wall between
			switch {
			case dy < 0 && m.Y > adj.Rect.Bottom():
				py = adj.Rect.Bottom() - int(push)
			case dy > 0 && m.Y < adj.Rect.Y-1:
				py = adj.Rect.Y + int(push)
			case dx < 0 && m.X > adj.Rect.Right():
				px = adj.Rect.Right() - int(push)
			case dx > 0 && m.X < adj.Rect.X-1:
				px = adj.Rect.X + int(push)
			}
		}
		gates = append(gates, gate{at: input.Point{X: m.X, Y: m.Y}, push: input.Point{X: px, Y: py}, border: true, depth: depthOf(m)})
		run = nil
	}
	for i, c := range cells {
		if i > 0 {
			p := cells[i-1]
			gap := c.X - p.X
			if vertical {
				gap = c.Y - p.Y
			}
			if gap > 2 || gap < 0 || depthOf(c) != depthOf(p) { // a notch in the wall (a shallower run) is a gate of its own
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
// cellMover is the part of a controller that moves to a fractional d2d cell (d2dinput).
type cellMover interface{ MoveCell(x, y float64) error }

func (n *Navigator) pushThrough(ctx context.Context, g gate, levelNo int) bool {
	t := n.tune()
	end := n.clk.Now().Add(6 * time.Second)
	for n.clk.Now().Before(end) && ctx.Err() == nil {
		if n.cm != nil && g.exact != nil {
			_ = n.cm.MoveCell(g.exact[0], g.exact[1])
		} else {
			_ = n.in.ClickWorld(g.push)
		}
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
				if errors.Is(err, ErrNoExit) && from != 0 && n.levelNow() == from && !(n.badTTL() > 0 && n.edgeShown(from, next)) {
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
	lm       *d2map.LevelMap
	dead     bool // survives learn() and a re-entered game, see markDeadGate
}

// markBadGate: a gap that did not lead out. It expires like a bad edge (BadEdgeTTLSec, and when the game is
// re-entered); with the knob at 0 it lasts for the rest of the run.
func (n *Navigator) markBadGate(from, to int, at input.Point) {
	b := badGate{from: from, to: to, at: at}
	if ttl := n.badTTL(); ttl > 0 {
		b.until, b.lm = n.clk.Now().Add(ttl), n.lvl(from)
	}
	n.badGates = append(n.badGates, b)
}

// markStalledGate is a bad gate for stalledGateTTL only: a walk that stalled on the way to it (a pack, a ghost
// obstacle) says nothing about whether the gap leads out.
func (n *Navigator) markStalledGate(from, to int, at input.Point) {
	n.badGates = append(n.badGates, badGate{from: from, to: to, at: at, until: n.clk.Now().Add(stalledGateTTL)})
}

// deadGateTTL: how long a gate that stalled every approach and forced the escape stays out of the plans.
const deadGateTTL = 30 * time.Minute

// markDeadGate: a gate whose every approach stalled until the escape hook fired. Unlike a stalled gate it is not
// forgotten after stalledGateTTL, after a crossing elsewhere (learn) or after a re-entered game, because the town
// walk then picked it again every time (soak kit i2: 5342,6280 in the Rogue Encampment, three escapes in 10 min).
func (n *Navigator) markDeadGate(from, to int, at input.Point) {
	n.badGates = append(n.badGates, badGate{from: from, to: to, at: at, until: n.clk.Now().Add(deadGateTTL), dead: true})
}

// gateIsBad: a gate within 6 subtiles of one that already failed.
func (n *Navigator) gateIsBad(from, to int, at input.Point) bool {
	return n.gateBad(from, to, at, true)
}

func (n *Navigator) gateBad(from, to int, at input.Point, withDead bool) bool {
	for _, b := range n.badGates {
		if b.dead && !withDead {
			continue
		}
		if b.from == from && b.to == to && dist(b.at, at) <= 6 && (b.until.IsZero() || n.clk.Now().Before(b.until) && (b.lm == nil || n.lvl(from) == b.lm)) {
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

// pocketTarget decides whether the character is cut off from every usable gate to levelNo on the known grid, and if
// so returns the connected cell nearest to it. pocket is false when the start connects to a gate (the plan failed
// for another reason) or when there is no gate or no start cell to reason about.
func (n *Navigator) pocketTarget(o *obs, levelNo int) (input.Point, bool) {
	cands := n.gateCands(o, levelNo)
	if len(cands) == 0 {
		return input.Point{}, false
	}
	n.planMu.Lock()
	defer n.planMu.Unlock()
	f := n.fieldFor(o.lm)
	f.ClearDyn()
	sx, sy, ok := f.nearestWalkable(o.pos.X, o.pos.Y, 8)
	if !ok {
		return input.Point{}, false
	}
	si := f.idx(sx, sy)
	strip := otherStrips(o.lm, levelNo)
	// A cell the mouth gate would refuse a click to (inside an unwanted exit's disc) is the last choice: leavePocket clicks
	// straight toward it, a refused click moves nobody, and the nearest cell is often just across the barrier beside the stairs
	// (card cellar3gate, Tower Cellar 3). safe is the best cell with a clickable line, best the best of all.
	best, bestD, found := input.Point{}, math.MaxFloat64, false
	safe, safeD, haveSafe := input.Point{}, math.MaxFloat64, false
	for _, c := range cands {
		gx, gy, ok := f.nearestWalkable(c.at.X, c.at.Y, 3)
		if !ok {
			continue
		}
		hops := floodHops(f, gx, gy)
		if hops[si] >= 0 {
			return input.Point{}, false
		}
		for i, h := range hops {
			if h < 0 {
				continue
			}
			x, y := f.pt(int32(i))
			if f.Clearance(x, y) < 1.5 { // a cell off the wall: the character is not pushed back into it
				continue
			}
			if strip.near(x, y) { // a step onto another level's strip leaves the level
				continue
			}
			p := input.Point{X: x, Y: y}
			d := dist(o.pos, p)
			if d < bestD {
				best, bestD, found = p, d, true
			}
			if d < safeD && n.MouthClickOK(o.pos, p) {
				safe, safeD, haveSafe = p, d, true
			}
		}
	}
	if haveSafe {
		return safe, true
	}
	return best, found
}

// leavePocket clicks toward target in steps of at most 10 subtiles (a click on a far cell the game cannot path to
// goes nowhere) until the character is within 1.5 of it or the level changed. It reports whether it got there, and
// whether the mouth gate refused a click (the caller names that cause; it used to be dropped and the loop stood still).
func (n *Navigator) leavePocket(ctx context.Context, o *obs, target input.Point) (ok, refused bool) {
	t := n.tune()
	for i := 0; i < 8; i++ {
		cur, err := n.observeWait(ctx)
		if err != nil || cur.level != o.level {
			return err == nil, false
		}
		d := dist(cur.pos, target)
		if d <= 1.5 {
			return true, false
		}
		k := math.Min(d, 10) / d
		if err := n.in.ClickWorld(input.Point{X: cur.pos.X + int(math.Round(float64(target.X-cur.pos.X)*k)), Y: cur.pos.Y + int(math.Round(float64(target.Y-cur.pos.Y)*k))}); errors.Is(err, ErrMouthGate) {
			return false, true
		}
		n.clk.Sleep(8 * t.Poll())
	}
	return false, false
}

// stripSet is the border cells of a level toward the neighbours other than the wanted one.
type stripSet map[[2]int]bool

// otherStrips collects them; near tells whether a cell lies on or within stripMargin subtiles of one.
func otherStrips(lm *d2map.LevelMap, wanted int) stripSet {
	s := stripSet{}
	for _, a := range lm.Adjacent {
		if a.LevelNo == wanted {
			continue
		}
		for _, c := range lm.BorderCells(a.LevelNo) {
			s[[2]int{c.X, c.Y}] = true
		}
	}
	return s
}

const stripMargin = 2

func (s stripSet) near(x, y int) bool {
	if len(s) == 0 {
		return false
	}
	for dy := -stripMargin; dy <= stripMargin; dy++ {
		for dx := -stripMargin; dx <= stripMargin; dx++ {
			if s[[2]int{x + dx, y + dy}] {
				return true
			}
		}
	}
	return false
}

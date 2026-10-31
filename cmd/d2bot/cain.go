package main

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/quest"
	"d2lod-bot/internal/task"
)

// Act 1 object work: the cairn stones of Stony Field and the "click each of several identical objects once" rule.
// Tags: classes 17..21 are S (Kolbot StoneAlpha..StoneLambda), that a click lights a stone as a mode change and that a
// wrong stone resets the lit ones is U (docs/actbot.md live checks).

// CairnStones is the Interact name task.cainNodes passes for the five stones.
const CairnStones = "cairn-stones"

const stonyFieldArea = 4

// cairnClasses are the five stones (S, Kolbot). quest.ObjCairnStone (17) is the first.
var cairnClasses = []int{quest.ObjCairnStone, quest.ObjCairnStone + 1, quest.ObjCairnStone + 2, quest.ObjCairnStone + 3, quest.ObjCairnStone + 4}

// onceSet remembers which object ids were already clicked, per area, so three calls for "cage" click three different
// cages and Cain's gibbet is not clicked twice. A new area forgets (ids are only unique within a game, and a visit of
// the area after a restart must be able to click again).
type onceSet struct {
	mu   sync.Mutex
	area int
	ids  map[uint32]bool
}

func (s *onceSet) add(area int, id uint32) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.area != area || s.ids == nil {
		s.area, s.ids = area, map[uint32]bool{}
	}
	s.ids[id] = true
}

func (s *onceSet) has(area int, id uint32) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.area == area && s.ids[id]
}

func (o objectAdapter) findUnclicked(ctx context.Context, class, area int) (*d2.Unit, error) {
	return o.findNearSkip(ctx, class, nil, func(u *d2.Unit) bool { return o.clicked.has(area, u.ID) })
}

// stoneOps is what the stone search needs from the game, so the search can be tested without one.
type stoneOps interface {
	Click(ctx context.Context, i int) error // click stone i (0..4)
	Lit(i int) bool                         // stone i shows the lit state (after the click settled)
	Portal() bool                           // the Tristram portal exists
}

// solveCairn finds the stone order by backtracking: a stone clicked at the right position stays lit, a wrong one
// resets all of them (U), so each position costs at most the remaining candidates plus a re-click of the confirmed
// prefix. At most 5+4+3+2+1 tries plus prefix re-clicks, bounded by maxClicks.
func solveCairn(ctx context.Context, ops stoneOps, n, maxClicks int) error {
	var order []int
	clicks := 0
	click := func(i int) error {
		clicks++
		if clicks > maxClicks {
			return fmt.Errorf("cairn stones: %d clicks without a portal", maxClicks)
		}
		return ops.Click(ctx, i)
	}
	for len(order) < n {
		placed := false
		for c := 0; c < n && !placed; c++ {
			if contains(order, c) {
				continue
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			// a reset (or the very first try) leaves the prefix unlit: click it again in order
			for _, p := range order {
				if !ops.Lit(p) {
					for _, q := range order {
						if err := click(q); err != nil {
							return err
						}
					}
					break
				}
			}
			if err := click(c); err != nil {
				return err
			}
			if ops.Portal() {
				return nil
			}
			if ops.Lit(c) {
				order = append(order, c)
				placed = true
			}
		}
		if !placed {
			return fmt.Errorf("cairn stones: no stone stays lit after %v (the lit state is not readable from the mode, U)", order)
		}
	}
	if !ops.Portal() {
		return fmt.Errorf("cairn stones: all five clicked in order %v but no portal", order)
	}
	return nil
}

func contains(s []int, v int) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// cairnStones opens the Tristram portal and walks through it. With the portal already open (a restart in the same
// game) it only enters.
func (o objectAdapter) cairnStones(ctx context.Context) error {
	s0, err := o.e.Snapshot()
	if err != nil {
		return err
	}
	if s0.Player.Area == quest.AreaTristram {
		return nil
	}
	if s0.Player.Area != stonyFieldArea {
		return task.Errorf(task.KindPrereq, "the cairn stones are in Stony Field, the character is in area %d", s0.Player.Area)
	}
	if !hasClass(s0, quest.ObjRedPortal) {
		ops := &liveStones{o: o}
		if err := ops.load(ctx); err != nil {
			return task.Wrap(task.KindNotFound, err)
		}
		if err := solveCairn(ctx, ops, len(ops.stones), 30); err != nil {
			return task.Wrap(task.KindTimeout, err)
		}
	}
	return o.Interact(ctx, "Tristram portal")
}

func hasClass(s *d2.Snapshot, class int) bool {
	for i := range s.Objects {
		if s.Objects[i].Class == class {
			return true
		}
	}
	return false
}

// liveStones is stoneOps over the running game.
type liveStones struct {
	o      objectAdapter
	stones []d2.Unit // sorted by class, the click order of the search
	init   map[uint32]int
}

func (l *liveStones) load(ctx context.Context) error {
	for _, c := range cairnClasses {
		u, err := l.o.find(ctx, c)
		if err != nil {
			return fmt.Errorf("stone class %d: %w", c, err)
		}
		l.stones = append(l.stones, *u)
	}
	sort.Slice(l.stones, func(a, b int) bool { return l.stones[a].Class < l.stones[b].Class })
	l.init = map[uint32]int{}
	for _, u := range l.stones {
		l.init[u.ID] = u.Mode
	}
	return nil
}

func (l *liveStones) Click(ctx context.Context, i int) error {
	u := l.stones[i]
	if err := l.o.click(ctx, &u); err != nil {
		return err
	}
	l.o.e.Clock.Sleep(900 * time.Millisecond)
	return nil
}

func (l *liveStones) Lit(i int) bool {
	s, err := l.o.e.Snapshot()
	if err != nil {
		return false
	}
	id := l.stones[i].ID
	for j := range s.Objects {
		if s.Objects[j].ID == id {
			return s.Objects[j].Mode != l.init[id]
		}
	}
	return false
}

func (l *liveStones) Portal() bool {
	s, err := l.o.e.Snapshot()
	return err == nil && hasClass(s, quest.ObjRedPortal)
}

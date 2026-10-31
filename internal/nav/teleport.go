package nav

import (
	"context"
	"math"
	"time"

	"d2lod-bot/internal/input"
)

// Teleport-aware movement. Set Navigator.TeleportKey (the hotkey Teleport sits on, "f2" for the Sorceress plan)
// and Explore, ExploreToward and ExitTo hop with CastAt along frontier cells instead of walking. A cast moves
// about 18 subtiles and lands only on walkable cells (V live, docs/act2.md "Live results"), walls in between do
// not matter, so the hop target is chosen from the known grid by distance, not by walking reachability. Every hop
// is checked by the position afterwards; cells where nothing happened are not tried again. Without a key, or when
// hopping does not work (no mana, no usable cell), the callers walk as before.

const (
	defaultTeleportRange = 18
	defaultTeleportMana  = 24 // Teleport level 1; later levels are cheaper, so this is only an upper bound
	teleportSettle       = 700 * time.Millisecond
	teleportManaWait     = 40 * time.Second
	teleportMinHop       = 4
	teleportBadHops      = 3 // consecutive hops that did not move the character
)

func (n *Navigator) hasTeleport() bool { return n.TeleportKey != "" }

func (n *Navigator) teleRange() float64 {
	if n.TeleportRange > 0 {
		return n.TeleportRange
	}
	return defaultTeleportRange
}

func (n *Navigator) teleMana() int {
	if n.TeleportMana > 0 {
		return n.TeleportMana
	}
	return defaultTeleportMana
}

// waitMana returns true when there is mana for one cast. Reading MaxMana 0 means the stat is unknown: cast anyway.
func (n *Navigator) waitMana(ctx context.Context, o *obs) (bool, error) {
	need := n.teleMana()
	deadline := n.clk.Now().Add(teleportManaWait)
	for {
		p := o.snap.Player
		if p.MaxMana <= 0 || p.Mana >= need {
			return true, nil
		}
		if n.Refill != nil {
			if err := n.Refill(ctx, p.Mana, need); err != nil {
				return false, err
			}
		} else {
			n.clk.Sleep(time.Second)
		}
		if n.clk.Now().After(deadline) {
			return false, nil
		}
		var err error
		if o, err = n.observeWait(ctx); err != nil {
			return false, err
		}
		if err := n.guard(o); err != nil {
			return false, err
		}
	}
}

// hopCell picks the walkable known cell within cast range that lands closest to target and gets at least 2
// subtiles nearer than the current position.
func (n *Navigator) hopCell(o *obs, target input.Point) (input.Point, bool) {
	g := o.lm.Grid
	r := n.teleRange()
	ri := int(r)
	here := dist(o.pos, target)
	best, bestD, bestFrom := input.Point{}, math.MaxFloat64, 0.0
	for dy := -ri; dy <= ri; dy++ {
		for dx := -ri; dx <= ri; dx++ {
			from := math.Hypot(float64(dx), float64(dy))
			if from > r || from < teleportMinHop {
				continue
			}
			c := input.Point{X: o.pos.X + dx, Y: o.pos.Y + dy}
			if !g.Walkable(c.X, c.Y) || n.teleBad[[2]int{c.X, c.Y}] > 0 {
				continue
			}
			d := dist(c, target)
			if d > here-2 {
				continue
			}
			if d < bestD-0.01 || d < bestD+0.01 && from > bestFrom {
				best, bestD, bestFrom = c, d, from
			}
		}
	}
	return best, bestD < math.MaxFloat64
}

// hop casts Teleport once toward target. moved is true when the character ended up somewhere else. ok is false
// when no hop could be made (no mana, no candidate cell); the caller should walk.
func (n *Navigator) hop(ctx context.Context, o *obs, target input.Point) (moved bool, ok bool, err error) {
	cell, found := n.hopCell(o, target)
	if !found {
		return false, false, nil
	}
	return n.hopAt(ctx, o, cell)
}

// hopAt casts Teleport at cell and checks the result by position.
func (n *Navigator) hopAt(ctx context.Context, o *obs, cell input.Point) (moved bool, ok bool, err error) {
	if enough, err := n.waitMana(ctx, o); err != nil || !enough {
		return false, false, err
	}
	if err := n.in.CastAt(n.TeleportKey, cell); err != nil {
		return false, false, err
	}
	n.clk.Sleep(teleportSettle)
	after, err := n.observeWait(ctx)
	if err != nil {
		return false, false, err
	}
	n.markVisited(after.pos)
	if dist(after.pos, o.pos) < 2 {
		n.teleBad[[2]int{cell.X, cell.Y}]++
		n.log.Info("nav teleport did not move", "to", cell)
		return false, true, nil
	}
	n.log.Debug("nav teleport", "from", o.pos, "to", after.pos, "aimed", cell)
	return true, true, nil
}

// teleportToward hops toward p until within stop subtiles or hops stop working; the caller walks the rest.
func (n *Navigator) teleportToward(ctx context.Context, p input.Point, stop float64) error {
	if !n.hasTeleport() {
		return nil
	}
	bad := 0
	for i := 0; i < 60; i++ {
		o, err := n.observeWait(ctx)
		if err != nil {
			return err
		}
		if err := n.guard(o); err != nil {
			return err
		}
		if dist(o.pos, p) <= stop {
			return nil
		}
		moved, ok, err := n.hop(ctx, o, p)
		if err != nil || !ok {
			return err
		}
		if moved {
			bad = 0
		} else if bad++; bad >= teleportBadHops {
			return nil
		}
	}
	return nil
}

// teleLeg is one exploration step by Teleport: it picks the best frontier bucket anywhere on the known grid
// (reachable by walking or not) and hops toward it. handled is false when the caller should walk this leg.
func (n *Navigator) teleLeg(ctx context.Context, o *obs, hint *input.Point) (handled bool, err error) {
	if n.teleFails >= teleportBadHops {
		return false, nil
	}
	f := n.fieldFor(o.lm)
	if f.W == 0 {
		return false, nil
	}
	type cand struct {
		pos   input.Point
		count int
	}
	buckets := map[[2]int]*cand{}
	for i := int32(0); int(i) < f.W*f.H; i++ {
		if f.flags[i]&fWalk == 0 {
			continue
		}
		x, y := f.pt(i)
		if !f.hasUnknownNeighbour(x, y) {
			continue
		}
		b := n.bucket(input.Point{X: x, Y: y})
		if n.visited[b] || n.failed[b] >= 1 {
			continue
		}
		c := buckets[b]
		if c == nil {
			c = &cand{pos: input.Point{X: x, Y: y}}
			buckets[b] = c
		}
		c.count++
	}
	var pick *cand
	var pickB [2]int
	pickScore := math.MaxFloat64
	for b, c := range buckets {
		s := dist(o.pos, c.pos) - 0.5*math.Min(float64(c.count), 30)
		if hint != nil {
			s -= 1.0 * (dist(o.pos, *hint) - dist(c.pos, *hint))
		}
		if s < pickScore || s == pickScore && b[0] < pickB[0] {
			pick, pickB, pickScore = c, b, s
		}
	}
	if pick == nil {
		return false, nil
	}
	moved, ok, err := n.hop(ctx, o, pick.pos)
	if err != nil {
		return false, err
	}
	if !ok {
		// No greedy hop: islands separated by void need a planned route or a bounded blind cast.
		target := pick.pos
		if hint != nil {
			target = *hint
		}
		if moved, ok, err = n.teleStep(ctx, o, target); err != nil {
			return false, err
		}
	}
	if !ok {
		n.failed[pickB]++
		return false, nil
	}
	if !moved {
		n.teleFails++
		return true, nil
	}
	n.teleFails = 0
	if after, err := n.observe(); err == nil && dist(after.pos, pick.pos) < 8 {
		n.failed[pickB]++ // inspected: a frontier that survives a visit is a dead end
	}
	return true, nil
}

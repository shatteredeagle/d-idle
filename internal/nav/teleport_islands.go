package nav

import (
	"container/heap"
	"context"
	"fmt"
	"math"

	"d2lod-bot/internal/input"
)

// Island-hopping Teleport. Levels such as the Arcane Sanctuary (74) are platforms ("islands") over a void with no
// room data in between (V live, docs/act2.md Y3/Y4). A hop lands only on walkable cells, so the known walkable cells
// form a graph: two cells are joined when they are within cast range, whatever lies between them. TeleportTo plans
// over that graph with A* (one hop = cost 1) and re-plans after every hop because rooms stream in while the
// character moves. When the target's island is not known yet, the planner walks to the known cell nearest the
// target and then casts "blind" at unknown cells on the target's side (blindCell): the cast lands only if the
// server finds walkable ground there (U live: Y4 saw blind casts stop at the first void edge, so a blind cast
// that moves the character still counts as progress). Every cast is verified by position and failed cells are
// remembered with a radius, so the budget bounds the cost of a wrong guess.

const (
	islandStep      = 4  // spacing of interior planner nodes; rim cells are always nodes
	blindBadRadius  = 3  // cells around a blind cast that did nothing are not retried
	blindFailBudget = 12 // blind casts without any effect before TeleportTo gives up
	teleportMaxHops = 80
	teleportStall   = 8 // hops without getting 2 subtiles closer to the target
)

// TeleportTo moves the character to within a few subtiles of (x, y) by Teleport, over void if needed. It needs
// TeleportKey. ErrNoPath means the hops stopped working or stopped making progress; the caller may walk the rest.
func (n *Navigator) TeleportTo(ctx context.Context, x, y int) error {
	return n.TeleportToStop(ctx, x, y, 5)
}

// TeleportToStop is TeleportTo with an explicit arrival radius in subtiles.
func (n *Navigator) TeleportToStop(ctx context.Context, x, y int, stop float64) error {
	if !n.hasTeleport() {
		return fmt.Errorf("%w: no teleport key", ErrNoPath)
	}
	target := input.Point{X: x, Y: y}
	n.teleBad = map[[2]int]int{}
	best, stall, blindFails := math.MaxFloat64, 0, 0
	for i := 0; i < teleportMaxHops; i++ {
		o, err := n.observeWait(ctx)
		if err != nil {
			return err
		}
		if err := n.guard(o); err != nil {
			return err
		}
		d := dist(o.pos, target)
		if d <= stop {
			return nil
		}
		if d < best-2 {
			best, stall = d, 0
		} else if stall++; stall >= teleportStall {
			return fmt.Errorf("%w: teleport stalled %.0f from (%d,%d)", ErrNoPath, d, x, y)
		}
		if d <= n.teleRange() && o.lm.Grid.Walkable(x, y) {
			if moved, _, err := n.hopAt(ctx, o, target); err != nil {
				return err
			} else if moved {
				continue
			}
		}
		moved, ok, err := n.teleStep(ctx, o, target)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: no teleport hop toward (%d,%d) from %v", ErrNoPath, x, y, o.pos)
		}
		if moved {
			blindFails = 0
		} else if blindFails++; blindFails >= blindFailBudget {
			return fmt.Errorf("%w: %d casts toward (%d,%d) did nothing", ErrNoPath, blindFails, x, y)
		}
	}
	return fmt.Errorf("%w: teleport hop limit", ErrNoPath)
}

// teleStep makes one planned hop toward target: along the island graph when the known cells lead closer, else a
// blind cast. ok is false when there was nothing to cast at.
func (n *Navigator) teleStep(ctx context.Context, o *obs, target input.Point) (moved, ok bool, err error) {
	if path, _ := n.planHops(o, target); len(path) > 0 {
		return n.hopAt(ctx, o, path[0])
	}
	if cell, found := n.blindCell(o, target); found {
		moved, ok, err = n.hopAt(ctx, o, cell)
		if err == nil && ok && !moved {
			n.badAround(cell)
		}
		return moved, ok, err
	}
	return false, false, nil
}

func (n *Navigator) badAround(c input.Point) {
	for dy := -blindBadRadius; dy <= blindBadRadius; dy++ {
		for dx := -blindBadRadius; dx <= blindBadRadius; dx++ {
			n.teleBad[[2]int{c.X + dx, c.Y + dy}]++
		}
	}
}

type hopItem struct {
	id int32
	f  float64
}
type hopQueue []hopItem

func (q hopQueue) Len() int           { return len(q) }
func (q hopQueue) Less(i, j int) bool { return q[i].f < q[j].f }
func (q hopQueue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *hopQueue) Push(x any)        { *q = append(*q, x.(hopItem)) }
func (q *hopQueue) Pop() any {
	old := *q
	it := old[len(old)-1]
	*q = old[:len(old)-1]
	return it
}

// planNodes are the planner's landing cells: every islandStep-th known walkable cell plus every walkable cell on
// an island's rim (a 4-neighbour that is not walkable), because the rim is what bridges the gaps.
func (n *Navigator) planNodes(o *obs) []input.Point {
	g := o.lm.Grid
	b := g.Bounds()
	var nodes []input.Point
	for y := b.Y; y < b.Bottom(); y++ {
		for x := b.X; x < b.Right(); x++ {
			if !g.Walkable(x, y) || n.teleBad[[2]int{x, y}] > 0 {
				continue
			}
			if (x-b.X)%islandStep == 0 && (y-b.Y)%islandStep == 0 ||
				!g.Walkable(x+1, y) || !g.Walkable(x-1, y) || !g.Walkable(x, y+1) || !g.Walkable(x, y-1) {
				nodes = append(nodes, input.Point{X: x, Y: y})
			}
		}
	}
	return nodes
}

// planHops runs A* over the known walkable cells (one hop = cost 1) and returns the landing cells from the first
// hop on. When no known cell is close enough to target, the path leads to the reachable cell nearest the target
// (reached false); the path is empty when the character already stands at that best cell.
func (n *Navigator) planHops(o *obs, target input.Point) (path []input.Point, reached bool) {
	nodes := n.planNodes(o)
	if len(nodes) == 0 {
		return nil, false
	}
	r := n.teleRange()
	cs := int(math.Ceil(r))
	key := func(p input.Point) [2]int { return [2]int{floorDiv(p.X, cs), floorDiv(p.Y, cs)} }
	grid := map[[2]int][]int32{}
	for i, p := range nodes {
		k := key(p)
		grid[k] = append(grid[k], int32(i))
	}
	near := func(p input.Point, fn func(id int32, d float64)) {
		k := key(p)
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				for _, id := range grid[[2]int{k[0] + dx, k[1] + dy}] {
					if d := dist(p, nodes[id]); d <= r && d >= teleportMinHop {
						fn(id, d)
					}
				}
			}
		}
	}
	goalR := 3.0
	if o.lm.Grid.Walkable(target.X, target.Y) {
		goalR = math.Max(goalR, r-1) // the last hop goes straight to the target
	}
	h := func(p input.Point) float64 { return math.Ceil(math.Max(0, dist(p, target)-goalR) / r) }

	gs := make([]float32, len(nodes))
	from := make([]int32, len(nodes))
	closed := make([]bool, len(nodes))
	for i := range gs {
		gs[i], from[i] = -1, -1
	}
	var open hopQueue
	push := func(id int32, cost float32, par int32) {
		if gs[id] >= 0 && gs[id] <= cost {
			return
		}
		gs[id], from[id] = cost, par
		heap.Push(&open, hopItem{id, float64(cost) + h(nodes[id])})
	}
	near(o.pos, func(id int32, d float64) { push(id, 1+float32(d)*0.001, -1) })
	bestID, bestD := int32(-1), dist(o.pos, target)-2
	goal := int32(-1)
	for open.Len() > 0 {
		id := heap.Pop(&open).(hopItem).id
		if closed[id] {
			continue
		}
		closed[id] = true
		c := nodes[id]
		if d := dist(c, target); d <= goalR {
			goal = id
			break
		} else if d < bestD {
			bestID, bestD = id, d
		}
		near(c, func(nid int32, d float64) {
			if !closed[nid] {
				push(nid, gs[id]+1+float32(d)*0.001, id)
			}
		})
	}
	end := goal
	if end < 0 {
		end = bestID
	}
	if end < 0 {
		return nil, false
	}
	for id := end; id >= 0; id = from[id] {
		path = append([]input.Point{nodes[id]}, path...)
	}
	return path, goal >= 0
}

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// blindCell picks an unknown cell within cast range on the target's side, at least 2 subtiles nearer than the
// character, to cast at when no known cell leads on. Unknown cells inside a static Room2 rectangle (a room of the
// level that has not streamed in) are preferred: the level exists there, only its collision is not loaded.
func (n *Navigator) blindCell(o *obs, target input.Point) (input.Point, bool) {
	g := o.lm.Grid
	r := n.teleRange()
	ri := int(r)
	here := dist(o.pos, target)
	inRoom := func(c input.Point) bool {
		for _, rm := range o.lm.Rooms {
			if rm.Rect.Contains(c.X, c.Y) {
				return true
			}
		}
		return false
	}
	best, bestS, found := input.Point{}, math.MaxFloat64, false
	for dy := -ri; dy <= ri; dy++ {
		for dx := -ri; dx <= ri; dx++ {
			if f := math.Hypot(float64(dx), float64(dy)); f > r || f < teleportMinHop+2 {
				continue
			}
			c := input.Point{X: o.pos.X + dx, Y: o.pos.Y + dy}
			if g.Known(c.X, c.Y) || n.teleBad[[2]int{c.X, c.Y}] > 0 || !o.lm.Rect.Empty() && !o.lm.Rect.Contains(c.X, c.Y) {
				continue
			}
			d := dist(c, target)
			if d > here-2 {
				continue
			}
			if s := d - map[bool]float64{true: 6, false: 0}[inRoom(c)]; s < bestS {
				best, bestS, found = c, s, true
			}
		}
	}
	return best, found
}

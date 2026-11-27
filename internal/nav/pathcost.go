package nav

import (
	"math"

	"d2lod-bot/internal/input"
)

// PathCost is the length in subtiles of the planned path between two points of the current map, without moving or
// advancing the map tracker (the town's stage 5 route prices its legs with it). false when there is no map, no path or
// only a partial one: the caller then falls back to the straight line.
func (n *Navigator) PathCost(from, to input.Point) (int, bool) {
	lm := n.w.Current()
	if lm == nil {
		return 0, false
	}
	res, err := n.plan(lm, from, to, n.tune().ArriveTolerance, nil, 0)
	if err != nil || res.partial || res.optimistic {
		return 0, false
	}
	var d float64
	at := from
	for _, w := range res.wps {
		d += math.Hypot(float64(w.X-at.X), float64(w.Y-at.Y))
		at = w
	}
	return int(math.Round(d)), true
}

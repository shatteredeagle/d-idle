package main

import (
	"fmt"
	"math"
	"time"

	"d2lod-bot/internal/input"
)

// gotoNear is a crude obstacle-dodging walk for measurement runs (the real pathing is internal/nav):
// step toward the goal in short hops, fanning out to other headings when a hop makes no progress.
func (e *env) gotoNear(gx, gy int, tol float64) bool {
	for iter := 0; iter < 60; iter++ {
		p, err := e.r.Player()
		if err != nil {
			return false
		}
		d := math.Hypot(float64(gx)-p.X, float64(gy)-p.Y)
		if d <= tol {
			return true
		}
		ang := math.Atan2(float64(gy)-p.Y, float64(gx)-p.X)
		hop := math.Min(d, 12)
		moved := false
		for _, da := range []float64{0, 0.5, -0.5, 1.0, -1.0, 1.5, -1.5, 2.2, -2.2} {
			a := ang + da
			tx, ty := int(p.X+hop*math.Cos(a)), int(p.Y+hop*math.Sin(a))
			if e.nearNPC(float64(tx), float64(ty), 3) {
				continue
			}
			e.g.ClickWorld(input.Point{X: tx, Y: ty})
			time.Sleep(900 * time.Millisecond)
			q, _ := e.waitIdle()
			if math.Hypot(q.X-p.X, q.Y-p.Y) > 2 {
				moved = true
				break
			}
		}
		if !moved {
			fmt.Println("goto: boxed in at", p.X, p.Y)
			return false
		}
	}
	return false
}

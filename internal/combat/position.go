package combat

import (
	"math"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// Pure geometry for positioning. All coordinates are subtiles.

func pdist(a, b input.Point) float64 { return math.Hypot(float64(a.X-b.X), float64(a.Y-b.Y)) }

// approachPoint is the point at distance want from target on the line toward from.
func approachPoint(from, target input.Point, want float64) input.Point {
	d := pdist(from, target)
	if d < 0.5 {
		return from
	}
	f := want / d
	return input.Point{
		X: target.X + int(math.Round(float64(from.X-target.X)*f)),
		Y: target.Y + int(math.Round(float64(from.Y-target.Y)*f)),
	}
}

var retreatAngles = []float64{0, 30, -30, 60, -60, 90, -90, 120, -120, 180}

// awayPoint finds a point about step subtiles from the player that keeps farther from every threat than the player
// is now, preferring the direction straight away from the nearest ones. valid says whether a point is walkable /
// reachable (Mover.PathTo); hazards are points to stay clear of. ok is false when nothing improves the situation.
func awayPoint(player input.Point, threats, hazards []input.Point, step int, valid func(input.Point) bool) (input.Point, bool) {
	if len(threats) == 0 || step <= 0 {
		return player, false
	}
	// direction: away from the centroid of the nearest threats
	var cx, cy float64
	n := 0
	for _, t := range threats {
		if pdist(player, t) < 30 {
			cx += float64(t.X)
			cy += float64(t.Y)
			n++
		}
	}
	if n == 0 {
		return player, false
	}
	cx /= float64(n)
	cy /= float64(n)
	base := math.Atan2(float64(player.Y)-cy, float64(player.X)-cx)
	cur := minDist(player, threats)
	best, bestScore, found := player, -1e9, false
	for _, frac := range []float64{1, 0.66, 0.4} {
		for _, deg := range retreatAngles {
			a := base + deg*math.Pi/180
			l := float64(step) * frac
			c := input.Point{X: player.X + int(math.Round(math.Cos(a)*l)), Y: player.Y + int(math.Round(math.Sin(a)*l))}
			if c == player {
				continue
			}
			md := minDist(c, threats)
			if md <= cur+1 {
				continue
			}
			score := md - math.Abs(deg)/45
			for _, h := range hazards {
				if pdist(c, h) < 5 {
					score -= 20
				}
			}
			if score > bestScore && (valid == nil || valid(c)) {
				best, bestScore, found = c, score, true
			}
		}
		if found {
			return best, true
		}
	}
	return player, false
}

func minDist(p input.Point, ps []input.Point) float64 {
	m := math.MaxFloat64
	for _, q := range ps {
		m = math.Min(m, pdist(p, q))
	}
	return m
}

// hostileMissiles lists missile positions that are not the player's or a friendly unit's.
func hostileMissiles(snap *d2.Snapshot) []input.Point {
	if snap == nil || snap.Player == nil {
		return nil
	}
	friendly := Friendlies(snap, nil)
	var out []input.Point
	for _, m := range snap.Missiles {
		if _, ok := friendly[m.OwnerID]; ok || (m.OwnerID == snap.Player.ID && m.OwnerType == 0) {
			continue
		}
		out = append(out, input.Point{X: m.X, Y: m.Y})
	}
	return out
}

// onScreen reports whether a world point can be clicked with the current viewport.
func onScreen(snap *d2.Snapshot, w input.Point) bool {
	s := screenOf(snap, w)
	return input.Calibrated.InView(s)
}

// screenOf projects a world point (tile centre) with the viewport variables, as input.Game does.
func screenOf(snap *d2.Snapshot, w input.Point) input.Screen {
	x, y := input.Calibrated.WorldToScreenVP(snap.Viewport.X, snap.Viewport.Y, float64(w.X)+0.5, float64(w.Y)+0.5)
	return input.Screen{X: int(math.Round(x)), Y: int(math.Round(y))}
}

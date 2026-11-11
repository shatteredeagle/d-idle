package senses

import "time"

// Trail retention and sampling (docs/senses-map.md "Trail"). 5 minutes at one point per 250 ms is 1200 points at most;
// the cap keeps memory tiny even if the clock misbehaves.
const (
	TrailRetention = 5 * time.Minute
	TrailMinGap    = 250 * time.Millisecond
	TrailCap       = 1200
)

// TrailPoint is one sample of the hero position. Entry marks the first sample after the hero came into the area.
type TrailPoint struct {
	At    time.Time
	X, Y  int
	Area  int
	Entry bool
}

// Trail is the ring of hero positions. It records a sample at most every TrailMinGap and only when the position
// changed, so a stall is a long gap between two samples. Not safe for concurrent use; Publisher holds its lock.
type Trail struct {
	pts []TrailPoint
}

// Add records the hero at (x, y) in area at time now. It reports whether a sample was stored.
func (t *Trail) Add(now time.Time, area, x, y int) bool {
	n := len(t.pts)
	entry := n == 0 || t.pts[n-1].Area != area
	if !entry {
		last := t.pts[n-1]
		if (last.X == x && last.Y == y) || now.Sub(last.At) < TrailMinGap {
			return false
		}
	}
	t.pts = append(t.pts, TrailPoint{At: now, X: x, Y: y, Area: area, Entry: entry})
	cut := 0
	for cut < len(t.pts) && now.Sub(t.pts[cut].At) > TrailRetention {
		cut++
	}
	if over := len(t.pts) - cut - TrailCap; over > 0 {
		cut += over
	}
	if cut > 0 {
		t.pts = append(t.pts[:0], t.pts[cut:]...)
	}
	return true
}

// Area returns the samples of one area still inside the retention at now, oldest first.
func (t *Trail) Area(now time.Time, area int) []TrailPoint {
	var out []TrailPoint
	for _, p := range t.pts {
		if p.Area == area && now.Sub(p.At) <= TrailRetention {
			out = append(out, p)
		}
	}
	return out
}

package nav

// Coverage point states of CoverageView.
const (
	CoverUnseen = "unseen"
	CoverSeen   = "seen"
	CoverFailed = "failed"
)

// CoverPoint is one roam coverage point in level coordinates.
type CoverPoint struct {
	X, Y  int
	State string
}

// CoverageView is a read-only copy of the roam state of the current level (roam.go), for the watch minimap. Call it
// from the goroutine that drives the navigator (the OnObserve hook does); it never changes the roam.
type CoverageView struct {
	Level    int
	Spacing  int // distance between points, subtiles
	Cover    int // a point counts as seen when the hero is within this Chebyshev radius
	Pts      []CoverPoint
	Target   *CoverPoint // the point of the leg in progress; nil between legs
	Explored bool        // every point is seen or given up
	Legs     int
	Held     int
}

// CoverageView returns the roam coverage of the current level; ok is false when roaming is not in use (nothing built).
func (n *Navigator) CoverageView() (CoverageView, bool) {
	n.covMu.Lock()
	defer n.covMu.Unlock()
	c := n.cov
	if c == nil || len(c.pts) == 0 {
		return CoverageView{}, false
	}
	v := CoverageView{Level: c.level, Spacing: c.spacing, Cover: c.cover, Legs: c.legs, Held: c.held, Explored: true,
		Pts: make([]CoverPoint, len(c.pts))}
	for i, p := range c.pts {
		st := CoverUnseen
		switch {
		case c.seen[i]:
			st = CoverSeen
		case c.failed[i]:
			st = CoverFailed
		default:
			v.Explored = false
		}
		v.Pts[i] = CoverPoint{p.X, p.Y, st}
	}
	if c.hasTgt {
		v.Target = &CoverPoint{c.tgt.X, c.tgt.Y, CoverUnseen}
	}
	return v, true
}

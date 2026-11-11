package senses

import "math"

// MaxCoverCells caps the coverage cells published. A big outdoor level has about 1600 roam points (V live, Blood Moor
// 1593), and the points are sorted left to right, so a cap that truncated drew only the west strip: SetCoverage
// downsamples uniformly instead, and Coverage.Cell grows to keep the picture covering the zone.
const MaxCoverCells = 2000

// Cover cell states (the third number of a cell).
const (
	CoverUnseen = 0
	CoverSeen   = 1
	CoverFailed = 2
)

// Coverage is the navigator's roam coverage of one level (nav.Navigator.CoverageView), copied by the bot wiring.
// Cells are {x, y, state} in the level's coordinates; Target is the point of the leg in progress or nil.
type Coverage struct {
	Area     int
	Cell     int // spacing between points, subtiles
	Cover    int // radius within which a point counts as seen
	Cells    [][3]int
	Total    int // points before the cap
	Drawn    int // cells published (equals Total unless downsampled)
	Seen     int // points seen and failed, counted over all points before any downsampling
	Failed   int
	Target   *Point
	Explored bool
	Legs     int
	Held     int
}

// SetCoverage stores the latest coverage; a zero Cells list clears it. Called from the nav goroutine at about 2 Hz.
func (p *Publisher) SetCoverage(c Coverage) {
	p.mu.Lock()
	defer p.mu.Unlock()
	c.Seen, c.Failed = 0, 0
	for _, q := range c.Cells {
		switch q[2] {
		case CoverSeen:
			c.Seen++
		case CoverFailed:
			c.Failed++
		}
	}
	if c.Total < len(c.Cells) {
		c.Total = len(c.Cells)
	}
	if n := len(c.Cells); n > MaxCoverCells {
		out := make([][3]int, MaxCoverCells)
		for i := range out {
			out[i] = c.Cells[i*n/MaxCoverCells]
		}
		c.Cell *= int(math.Ceil(math.Sqrt(float64(n) / MaxCoverCells))) // each kept cell stands for its neighbours too
		c.Cells = out
	}
	c.Drawn = len(c.Cells)
	p.cover = c
}

// CoverageOf returns the coverage when it belongs to the area the hero is in now.
func (p *Publisher) CoverageOf() (Coverage, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.have || len(p.cover.Cells) == 0 || p.cover.Area != p.areaNo {
		return Coverage{}, false
	}
	return p.cover, true
}

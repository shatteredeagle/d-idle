package senses

// MaxCoverCells caps the coverage cells published (a level has some dozens of roam points; the cap is a guard).
const MaxCoverCells = 400

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
	Total    int // cells before the cap
	Target   *Point
	Explored bool
	Legs     int
	Held     int
}

// SetCoverage stores the latest coverage; a zero Cells list clears it. Called from the nav goroutine at about 2 Hz.
func (p *Publisher) SetCoverage(c Coverage) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(c.Cells) > MaxCoverCells {
		c.Cells = c.Cells[:MaxCoverCells]
	}
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

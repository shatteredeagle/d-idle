// Package senses is the read-only snapshot of the area the hero stands in, built for the watch screen's side-panel
// minimap (docs/senses-map.md, docs/research/info-unlocks.md). It reads a d2map level map and never feeds anything
// back to the bot.
package senses

import (
	"fmt"
	"sync"
	"time"

	"d2lod-bot/internal/d2map"
)

// Schema names the JSON shape of Map.
const Schema = "watch.senses/1"

// Cell values of the run-length encoded grid.
const (
	Unknown  = 0 // no loaded room covered the cell
	Walkable = 1
	Blocked  = 2
)

// MaxSide bounds the longer side of the sent grid; a bigger level is downsampled by an integer Step (a block is
// Walkable when any of its cells is, so thin paths survive).
const MaxSide = 256

// Point is a subtile position.
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Exit is a level transition at a position; Dest is the destination level number.
type Exit struct {
	Dest int `json:"dest"`
	X    int `json:"x"`
	Y    int `json:"y"`
}

// Map is the static part of one area. It is re-sent only when Version changes.
type Map struct {
	Schema    string  `json:"schema"`
	AreaID    int     `json:"areaId"`
	Area      string  `json:"area,omitempty"`
	Version   uint64  `json:"version"`
	X0        int     `json:"x0"` // subtile of the top-left grid cell
	Y0        int     `json:"y0"`
	Step      int     `json:"step"` // subtiles per grid cell
	W         int     `json:"w"`    // grid size in cells
	H         int     `json:"h"`
	RLE       []int   `json:"rle"` // value,length pairs in row-major order
	Exits     []Exit  `json:"exits"`
	Waypoints []Point `json:"waypoints"`
}

// Snapshot is what Publisher hands out: the map, the hero (2 Hz) and the optional units (SetUnits).
type Snapshot struct {
	Map  Map    `json:"map"`
	Hero Point  `json:"hero"`
	Unit []Unit `json:"units,omitempty"`
	// Neighbours are visited areas stitched beside Map (adjacency.go); empty when none is known or aligned.
	Neighbours []Neighbour `json:"neighbours,omitempty"`
}

// Build makes the map of lm. area is the display name ("" is fine). An empty or missing grid is an error naming the
// area id.
func Build(lm *d2map.LevelMap, area string, version uint64) (Map, error) {
	return build(lm, area, version, nil)
}

// build is Build with a visibility mask: cells, exits and waypoints that vis rejects are left out (nil shows all).
func build(lm *d2map.LevelMap, area string, version uint64, vis func(x, y int) bool) (Map, error) {
	if lm == nil || lm.Grid == nil || lm.Grid.KnownCount() == 0 {
		no := -1
		if lm != nil {
			no = lm.No
		}
		return Map{}, fmt.Errorf("senses: no geometry for area %d", no)
	}
	b := lm.Grid.Bounds()
	step := 1
	for max(b.W, b.H)/step > MaxSide {
		step++
	}
	w, h := (b.W+step-1)/step, (b.H+step-1)/step
	cells := make([]int, w*h)
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			f, ok := lm.Grid.Flags(b.X+x, b.Y+y)
			if !ok || (vis != nil && !vis(b.X+x, b.Y+y)) {
				continue
			}
			v := Blocked
			if f&d2map.BlockWalk == 0 {
				v = Walkable
			}
			i := (y/step)*w + x/step
			if cells[i] != Walkable {
				cells[i] = v
			}
		}
	}
	m := Map{Schema: Schema, AreaID: lm.No, Area: area, Version: version, X0: b.X, Y0: b.Y, Step: step, W: w, H: h,
		RLE: Encode(cells), Exits: []Exit{}, Waypoints: []Point{}}
	for _, e := range lm.Exits {
		if vis != nil && !vis(e.Pos.X, e.Pos.Y) {
			continue
		}
		m.Exits = append(m.Exits, Exit{e.Dest, e.Pos.X, e.Pos.Y})
	}
	for _, p := range lm.Waypoints() {
		if vis != nil && !vis(p.Pos.X, p.Pos.Y) {
			continue
		}
		m.Waypoints = append(m.Waypoints, Point{p.Pos.X, p.Pos.Y})
	}
	return m, nil
}

// Encode run-length encodes cells as value,length pairs.
func Encode(cells []int) []int {
	var out []int
	for i := 0; i < len(cells); {
		j := i
		for j < len(cells) && cells[j] == cells[i] {
			j++
		}
		out = append(out, cells[i], j-i)
		i = j
	}
	return out
}

// Decode expands rle into n cells. A malformed stream (odd length, bad value, wrong total) is an error.
func Decode(rle []int, n int) ([]int, error) {
	if len(rle)%2 != 0 {
		return nil, fmt.Errorf("senses: odd rle length %d", len(rle))
	}
	out := make([]int, 0, n)
	for i := 0; i < len(rle); i += 2 {
		v, c := rle[i], rle[i+1]
		if v < Unknown || v > Blocked || c <= 0 || len(out)+c > n {
			return nil, fmt.Errorf("senses: bad rle run %d,%d", v, c)
		}
		for ; c > 0; c-- {
			out = append(out, v)
		}
	}
	if len(out) != n {
		return nil, fmt.Errorf("senses: rle covers %d cells, want %d", len(out), n)
	}
	return out, nil
}

// RevealRadius is how far around the hero cells are revealed, in subtiles. S: about one screen half-width; the
// value is a guess to tune live.
const RevealRadius = 24

// fogBlock is the fog granularity in subtiles; blocks are absolute, so they survive a change of the sent Step.
const fogBlock = 4

type block struct{ x, y int32 }

// Publisher is the bot's side: Observe is called on each area change and at about 2 Hz for the hero. It keeps the
// last snapshot and the visited blocks of every area of the session (fog of war: the map shows only what the hero has
// been near). Safe for concurrent use; it never blocks the caller on a reader.
type Publisher struct {
	mu      sync.Mutex
	snap    Snapshot
	have    bool
	areaVer uint64 // grid version the map was built from
	seq     uint64
	areaNo  int
	seen    map[int]map[block]struct{}
	adj     *Adjacency     // nil: no stitching
	visits  map[int]*visit // last fogged map of each area of the session
	visitNo uint64
	skipped []error
	trail   Trail
	act     uint8
	events  Events
	samples Samples
	cover   Coverage
	now     func() time.Time // test clock; nil is time.Now
}

// SetAdjacency sets the table of areas that are stitched into one canvas; nil turns stitching off (the default).
func (p *Publisher) SetAdjacency(a *Adjacency) {
	p.mu.Lock()
	p.adj = a
	p.mu.Unlock()
}

// Skipped returns why the last observation left visited neighbours out (alignment errors name the pair).
func (p *Publisher) Skipped() []error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]error(nil), p.skipped...)
}

// refreshNeighbours records the current map as a visit (changed: the hero just entered this area) and recomputes the
// neighbour list, which can grow after the map did not (d2map learns the neighbours as rooms load). Holding p.mu.
func (p *Publisher) refreshNeighbours(lm *d2map.LevelMap, changed bool) {
	if p.adj == nil {
		return
	}
	if p.visits == nil {
		p.visits = map[int]*visit{}
	}
	if changed {
		p.visitNo++
	}
	p.visits[lm.No] = &visit{m: p.snap.Map, rect: lm.Rect, seq: p.visitNo}
	p.snap.Neighbours, p.skipped = neighbours(p.adj, lm, p.visits)
}

// Observe records the hero position, reveals the cells around it and rebuilds the map when the area, the grid or the
// revealed set changed. A level without geometry returns the error and keeps the previous snapshot.
func (p *Publisher) Observe(lm *d2map.LevelMap, area string, hero d2map.Point) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if lm == nil || lm.Grid == nil || lm.Grid.KnownCount() == 0 {
		_, err := Build(lm, area, 0)
		return err
	}
	if p.seen == nil {
		p.seen = map[int]map[block]struct{}{}
	}
	sv := p.seen[lm.No]
	if sv == nil {
		sv = map[block]struct{}{}
		p.seen[lm.No] = sv
	}
	p.trail.Add(p.clock(), lm.No, hero.X, hero.Y, p.act)
	grew := reveal(sv, hero)
	changed := !p.have || lm.No != p.areaNo
	if !grew && !changed && lm.Grid.Version() == p.areaVer {
		p.snap.Hero = Point{hero.X, hero.Y}
		p.refreshNeighbours(lm, false)
		return nil
	}
	p.seq++
	m, err := build(lm, area, p.seq, func(x, y int) bool { _, ok := sv[block{int32(floorDiv(x)), int32(floorDiv(y))}]; return ok })
	if err != nil {
		p.seq--
		return err
	}
	// A new map keeps the units only inside the same area: a new area starts with none until SetUnits runs again.
	var keep []Unit
	if p.have && lm.No == p.areaNo {
		keep = p.snap.Unit
	}
	p.snap = Snapshot{Map: m, Hero: Point{hero.X, hero.Y}, Unit: keep}
	p.have, p.areaNo, p.areaVer = true, lm.No, lm.Grid.Version()
	p.refreshNeighbours(lm, changed)
	return nil
}

func floorDiv(v int) int {
	if v < 0 {
		return (v - fogBlock + 1) / fogBlock
	}
	return v / fogBlock
}

// reveal marks the blocks within RevealRadius of c and reports whether any was new.
func reveal(sv map[block]struct{}, c d2map.Point) bool {
	grew := false
	for y := c.Y - RevealRadius; y <= c.Y+RevealRadius; y += fogBlock {
		for x := c.X - RevealRadius; x <= c.X+RevealRadius; x += fogBlock {
			if (x-c.X)*(x-c.X)+(y-c.Y)*(y-c.Y) > RevealRadius*RevealRadius {
				continue
			}
			b := block{int32(floorDiv(x)), int32(floorDiv(y))}
			if _, ok := sv[b]; !ok {
				sv[b] = struct{}{}
				grew = true
			}
		}
	}
	return grew
}

// Snapshot returns the last observation; ok is false before the first.
func (p *Publisher) Snapshot() (Snapshot, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.snap, p.have
}

func (p *Publisher) clock() time.Time {
	if p.now != nil {
		return p.now()
	}
	return time.Now()
}

// TrailOf returns the trail of the current area (oldest first) and the time it was read; ok is false before the
// first observation.
func (p *Publisher) TrailOf() (pts []TrailPoint, area int, now time.Time, ok bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	now = p.clock()
	return p.trail.Area(now, p.areaNo), p.areaNo, now, p.have
}

package senses

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"

	"d2lod-bot/internal/d2map"
)

//go:embed adjacency.json
var adjacencyJSON []byte

// MaxNeighbours bounds how many visited neighbours are drawn beside the current area; MaxNeighbourRLE bounds the
// size of one neighbour's grid (ints of its run-length list) so the payload stays small. A larger one is not stitched.
const (
	MaxNeighbours   = 2
	MaxNeighbourRLE = 20000
)

// Pair is one stitchable pair of areas (order is irrelevant). Evidence is V (recorded image), S (secondary) or U.
type Pair struct {
	A        int    `json:"a"`
	B        int    `json:"b"`
	Evidence string `json:"evidence"`
	Note     string `json:"note,omitempty"`
}

// Adjacency is the table of areas that may share a canvas.
type Adjacency struct {
	pairs map[[2]int]Pair
}

// ParseAdjacency reads the table. A duplicate or self pair is an error naming it.
func ParseAdjacency(b []byte) (*Adjacency, error) {
	var f struct {
		Schema string `json:"schema"`
		Pairs  []Pair `json:"pairs"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("senses: adjacency table: %w", err)
	}
	a := &Adjacency{pairs: map[[2]int]Pair{}}
	for _, p := range f.Pairs {
		k := pairKey(p.A, p.B)
		if p.A == p.B {
			return nil, fmt.Errorf("senses: adjacency pair %d<->%d joins an area to itself", p.A, p.B)
		}
		if _, dup := a.pairs[k]; dup {
			return nil, fmt.Errorf("senses: adjacency pair %d<->%d listed twice", k[0], k[1])
		}
		a.pairs[k] = p
	}
	return a, nil
}

// ShippedAdjacency parses the embedded table.
func ShippedAdjacency() (*Adjacency, error) { return ParseAdjacency(adjacencyJSON) }

func pairKey(a, b int) [2]int {
	if a > b {
		a, b = b, a
	}
	return [2]int{a, b}
}

// Has reports whether the pair may be stitched.
func (a *Adjacency) Has(x, y int) bool {
	if a == nil {
		return false
	}
	_, ok := a.pairs[pairKey(x, y)]
	return ok
}

// Neighbour is a visited area drawn beside the current one. DX and DY are added to the neighbour's subtile
// coordinates (map x0/y0, exits, waypoints) to put them in the current area's coordinate space. Map is the fogged
// grid as of the neighbour's last observation.
type Neighbour struct {
	AreaID  int    `json:"areaId"`
	Version uint64 `json:"version"`
	DX      int    `json:"dx"`
	DY      int    `json:"dy"`
	Map     Map    `json:"-"`
}

// visit is what the Publisher remembers of an area it has left or stands in.
type visit struct {
	m    Map
	rect d2map.Rect
	seq  uint64 // recency, from the Publisher's visit counter
}

// align derives the offset of the neighbour (recorded rect nb) into the frame of the current level lm. The current
// level's own Adjacent list gives the neighbour's rectangle in the current frame; the difference to the rectangle
// recorded when the neighbour was visited is the offset (zero when both frames are absolute, Game.exe). Errors name
// the pair; the caller then leaves the neighbour out ("not stitched").
func align(lm *d2map.LevelMap, no int, nb d2map.Rect) (dx, dy int, err error) {
	for _, a := range lm.Adjacent {
		if a.LevelNo != no {
			continue
		}
		if a.Rect.W != nb.W || a.Rect.H != nb.H {
			return 0, 0, fmt.Errorf("senses: adjacency %d<->%d: neighbour is %dx%d beside, %dx%d recorded", lm.No, no, a.Rect.W, a.Rect.H, nb.W, nb.H)
		}
		return a.Rect.X - nb.X, a.Rect.Y - nb.Y, nil
	}
	return 0, 0, fmt.Errorf("senses: adjacency %d<->%d: level %d reports no neighbour %d", lm.No, no, lm.No, no)
}

// neighbours lists the visited areas to draw beside lm, most recent first, and the pairs that were skipped with
// the reason (for tests and logs). Only pairs of the table are tried.
func neighbours(adj *Adjacency, lm *d2map.LevelMap, seen map[int]*visit) ([]Neighbour, []error) {
	type cand struct {
		no int
		v  *visit
	}
	var cs []cand
	for no, v := range seen {
		if no != lm.No && adj.Has(lm.No, no) {
			cs = append(cs, cand{no, v})
		}
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].v.seq > cs[j].v.seq })
	var out []Neighbour
	var skipped []error
	for _, c := range cs {
		if len(out) == MaxNeighbours {
			break
		}
		if len(c.v.m.RLE) > MaxNeighbourRLE {
			skipped = append(skipped, fmt.Errorf("senses: adjacency %d<->%d: neighbour grid too large (%d runs)", lm.No, c.no, len(c.v.m.RLE)))
			continue
		}
		dx, dy, err := align(lm, c.no, c.v.rect)
		if err != nil {
			skipped = append(skipped, err)
			continue
		}
		out = append(out, Neighbour{AreaID: c.no, Version: c.v.m.Version, DX: dx, DY: dy, Map: c.v.m})
	}
	return out, skipped
}

package nav

import (
	"fmt"
	"math"
	"os"
	"testing"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
)

// The roam spacing study (docs/roamspacing.md): replay the nearest-first roam on a level field for a spacing/cover pair
// and measure how much ground ends up near a visited position. Run: ROAMSTUDY=1 go test -run RoamStudy -v ./internal/nav

type studyRow struct {
	pts, legs      int
	path           int     // 8-neighbour hops walked
	f30, f20, f12  float64 // share of reachable walkable cells within that Euclidean distance of a walked cell
	maxGap         float64 // farthest reachable cell from any walked cell (capped at studyCap)
	pockets        int     // connected regions of cells farther than 20 from any walked cell
	cells, skipped int
}

const studyCap = 44

func studyRoam(f *Field, sx, sy, spacing, cover int) studyRow {
	sx, sy, _ = f.nearestWalkable(sx, sy, 8)
	dist := floodHops(f, sx, sy)
	pts := samplePoints(f, dist, spacing)
	seen := make([]bool, len(pts))
	walked := map[int32]bool{}
	cur := int32(f.idx(sx, sy))
	var row studyRow
	row.pts = len(pts)
	mark := func(i int32) {
		walked[i] = true
		x, y := f.pt(i)
		for k, q := range pts {
			if !seen[k] && max(abs(q.X-x), abs(q.Y-y)) <= cover {
				seen[k] = true
			}
		}
	}
	mark(cur)
	for {
		// BFS from cur with parents
		par := make([]int32, f.W*f.H)
		for i := range par {
			par[i] = -2
		}
		par[cur] = -1
		queue := []int32{cur}
		pick := int32(-1)
		for h := 0; h < len(queue) && pick < 0; h++ {
			ci := queue[h]
			cx, cy := f.pt(ci)
			for k, q := range pts {
				if !seen[k] && q.X == cx && q.Y == cy {
					pick = ci
					break
				}
			}
			if pick >= 0 {
				break
			}
			for d, dv := range dirs {
				nx, ny := cx+dv[0], cy+dv[1]
				ni := f.idx(nx, ny)
				if ni < 0 || par[ni] != -2 || f.flags[ni]&fWalk == 0 {
					continue
				}
				if d >= 4 && (!f.Walkable(cx+dv[0], cy) || !f.Walkable(cx, cy+dv[1])) {
					continue
				}
				par[ni] = ci
				queue = append(queue, int32(ni))
			}
		}
		if pick < 0 {
			break
		}
		var path []int32
		for p := pick; p != cur; p = par[p] {
			path = append(path, p)
		}
		for i := len(path) - 1; i >= 0; i-- {
			mark(path[i]) // the roam marks on every observation along the walk
		}
		row.path += len(path)
		row.legs++
		cur = pick
	}
	// distance of every reachable cell to the walked set, capped
	gap := make([]float32, f.W*f.H)
	for i := range gap {
		gap[i] = studyCap + 1
	}
	r := studyCap
	for i := range walked {
		x, y := f.pt(i)
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				j := f.idx(x+dx, y+dy)
				if j < 0 {
					continue
				}
				if d := float32(math.Hypot(float64(dx), float64(dy))); d < gap[j] {
					gap[j] = d
				}
			}
		}
	}
	var n30, n20, n12 int
	far := make([]bool, f.W*f.H)
	for i, d := range dist {
		if d < 0 {
			continue
		}
		row.cells++
		g := gap[i]
		if g <= 30 {
			n30++
		}
		if g <= 20 {
			n20++
		} else {
			far[i] = true
		}
		if g <= 12 {
			n12++
		}
		row.maxGap = math.Max(row.maxGap, float64(min(g, studyCap)))
	}
	row.f30, row.f20, row.f12 = float64(n30)/float64(row.cells), float64(n20)/float64(row.cells), float64(n12)/float64(row.cells)
	for i := range far { // flood the far cells into pockets
		if !far[i] {
			continue
		}
		row.pockets++
		st := []int{i}
		far[i] = false
		for len(st) > 0 {
			c := st[len(st)-1]
			st = st[:len(st)-1]
			cx, cy := f.pt(int32(c))
			for _, dv := range dirs {
				if j := f.idx(cx+dv[0], cy+dv[1]); j >= 0 && far[j] {
					far[j] = false
					st = append(st, j)
				}
			}
		}
	}
	return row
}

// caveGrid is a deterministic cave-like level: a 3-tile-wide winding corridor spine with side rooms and dead-end pockets.
func caveGrid() (*d2map.Grid, input.Point) {
	g := d2map.NewGrid()
	cells := make([]uint16, 240*240)
	for i := range cells {
		cells[i] = d2map.BlockWalk
	}
	open := func(x0, y0, w, h int) {
		for y := y0; y < y0+h; y++ {
			for x := x0; x < x0+w; x++ {
				if x >= 0 && y >= 0 && x < 240 && y < 240 {
					cells[y*240+x] = 0
				}
			}
		}
	}
	for r := 0; r < 6; r++ { // serpentine corridors 6 wide, rooms and dead ends hanging off
		y := 10 + r*38
		open(10, y, 220, 6)
		if r < 5 {
			x := 14
			if r%2 == 0 {
				x = 220
			}
			open(x, y, 6, 44)
		}
		for k := 0; k < 4; k++ {
			x := 30 + k*50 + r*7%20
			open(x, y+6, 5, 14+k*6) // dead-end stubs of growing length
			open(x-6, y+6+14+k*6, 17, 10)
		}
	}
	g.SetBlock(d2map.Rect{X: 2000, Y: 2000, W: 240, H: 240}, cells)
	return g, input.Point{X: 2012, Y: 2012}
}

func studyFields(t *testing.T) map[string]func() (*Field, int, int) {
	return map[string]func() (*Field, int, int){
		"Blood Moor (real, seed 17)": func() (*Field, int, int) {
			n, w := seed17World(t)
			lm := w.Current()
			f := n.fieldFor(lm)
			b := lm.BorderCells(1)
			if len(b) == 0 {
				t.Fatal("no gate cell")
			}
			return f, b[0].X, b[0].Y
		},
		"cave (synthetic)": func() (*Field, int, int) {
			g, st := caveGrid()
			return buildField(g, g.Bounds(), 3, 0.75), st.X, st.Y
		},
		"open field (synthetic)": func() (*Field, int, int) {
			g := openField(2000, 2000, 250, 250)
			return buildField(g, g.Bounds(), 3, 0.75), 2010, 2010
		},
	}
}

func TestRoamStudy(t *testing.T) {
	if os.Getenv("ROAMSTUDY") == "" {
		t.Skip("set ROAMSTUDY=1")
	}
	pairs := [][2]int{{7, 5}, {10, 7}, {14, 9}, {16, 10}, {18, 11}, {24, 14}}
	for name, mk := range studyFields(t) {
		f, x, y := mk()
		for _, p := range pairs {
			r := studyRoam(f, x, y, p[0], p[1])
			fmt.Printf("STUDY %-28s %2d/%-2d pts %5d legs %5d path %6d cells %6d in30 %5.1f%% in20 %5.1f%% in12 %5.1f%% maxgap %4.0f pockets>20 %d\n",
				name, p[0], p[1], r.pts, r.legs, r.path, r.cells, 100*r.f30, 100*r.f20, 100*r.f12, r.maxGap, r.pockets)
		}
	}
}

func TestRoamParamsClamp(t *testing.T) {
	d := env.DefaultNavTuning()
	if sp, cv := roamParams(d); sp != 16 || cv != 10 {
		t.Fatalf("defaults %d/%d", sp, cv)
	}
	d.RoamSpacing, d.RoamCover = 7, 5 // the legacy pair stays selectable
	if sp, cv := roamParams(d); sp != roamSpacing || cv != roamCover {
		t.Fatalf("legacy %d/%d", sp, cv)
	}
	d.RoamSpacing, d.RoamCover = 8, 30
	if sp, cv := roamParams(d); sp != 8 || cv != 8 {
		t.Fatalf("cover must clamp to spacing: %d/%d", sp, cv)
	}
	d.RoamSpacing, d.RoamCover = 0, 0
	if sp, cv := roamParams(d); sp != 1 || cv != 1 {
		t.Fatalf("floor: %d/%d", sp, cv)
	}
}

// The default pair leaves no pocket of walkable ground farther than 20 subtiles from the walked path, on a winding cave
// and an open field (class generic: nothing here depends on a build; docs/roamspacing.md has the real Blood Moor).
func TestRoamDefaultsLeaveNoPocket(t *testing.T) {
	sp, cv := roamParams(env.DefaultNavTuning())
	g, st := caveGrid()
	for name, f := range map[string]*Field{"cave": buildField(g, g.Bounds(), 3, 0.75), "open": buildField(openField(2000, 2000, 200, 200), d2map.Rect{X: 2000, Y: 2000, W: 200, H: 200}, 3, 0.75)} {
		r := studyRoam(f, st.X, st.Y, sp, cv)
		if r.pockets != 0 || r.maxGap > 20 {
			t.Fatalf("%s: %d pockets, max gap %.0f", name, r.pockets, r.maxGap)
		}
	}
}

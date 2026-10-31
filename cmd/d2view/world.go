package main

import "d2lod-bot/internal/status"

// grid is a rectangular walkability grid.
type grid struct {
	w, h int
	cell []bool
}

func newGrid(w, h int) *grid { return &grid{w: w, h: h, cell: make([]bool, w*h)} }
func (g *grid) at(x, y int) bool {
	return x >= 0 && y >= 0 && x < g.w && y < g.h && g.cell[y*g.w+x]
}
func (g *grid) fill(x0, y0, x1, y1 int, v bool) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if x >= 0 && y >= 0 && x < g.w && y < g.h {
				g.cell[y*g.w+x] = v
			}
		}
	}
}

type level struct {
	id    int
	g     *grid
	rev   int
	exits []status.MapExit
	spawn [2]int
}

// townFixture hand-builds a Rogue-Encampment-like town: palisade, buildings, paths.
func townFixture() *level {
	g := newGrid(120, 90)
	g.fill(8, 8, 112, 82, true) // inside the palisade
	// corner cuts so the palisade looks round-ish
	for i := 0; i < 10; i++ {
		g.fill(8, 8+i, 8+10-i, 9+i, false)
		g.fill(112-10+i, 8+i, 112, 9+i, false)
		g.fill(8, 81-i, 8+10-i, 82-i, false)
		g.fill(112-10+i, 81-i, 112, 82-i, false)
	}
	// buildings (blocked rectangles): Akara tent, Charsi forge, Gheed tent, Kashya hut, stash/waypoint area stays open
	for _, r := range [][4]int{{20, 18, 36, 30}, {80, 16, 98, 28}, {84, 56, 100, 68}, {24, 58, 40, 70}, {55, 8, 65, 16}, {52, 40, 58, 46}} {
		g.fill(r[0], r[1], r[2], r[3], false)
	}
	// east gate corridor
	g.fill(112, 38, 119, 50, true)
	return &level{id: 1, g: g, rev: 1, spawn: [2]int{60, 55}, exits: []status.MapExit{
		{X: 117, Y: 44, Kind: "exit", Name: "to Blood Moor", ToArea: 2},
		{X: 46, Y: 52, Kind: "waypoint", Name: "Waypoint"},
	}}
}

// fieldLevel builds a deterministic blobby outdoor area with rocky obstacles.
func fieldLevel() *level {
	g := newGrid(260, 200)
	h := func(x, y int) uint32 {
		v := uint32(x)*374761393 + uint32(y)*668265263
		v = (v ^ (v >> 13)) * 1274126177
		return v ^ (v >> 16)
	}
	// value noise at coarse resolution
	const c = 10
	n := func(x, y int) float64 {
		gx, gy := x/c, y/c
		fx, fy := float64(x%c)/c, float64(y%c)/c
		v := func(a, b int) float64 { return float64(h(a, b)%1000) / 1000 }
		top := v(gx, gy)*(1-fx) + v(gx+1, gy)*fx
		bot := v(gx, gy+1)*(1-fx) + v(gx+1, gy+1)*fx
		return top*(1-fy) + bot*fy
	}
	for y := 6; y < g.h-6; y++ {
		for x := 6; x < g.w-6; x++ {
			dx, dy := float64(x-g.w/2)/float64(g.w/2), float64(y-g.h/2)/float64(g.h/2)
			edge := dx*dx + dy*dy
			g.cell[y*g.w+x] = edge < 0.85 && n(x, y) > 0.28+edge*0.2
		}
	}
	g.fill(8, 92, 40, 108, true) // clearing at the town entrance
	return &level{id: 2, g: g, rev: 2, spawn: [2]int{14, 100}, exits: []status.MapExit{
		{X: 9, Y: 100, Kind: "exit", Name: "to Rogue Encampment", ToArea: 1},
	}}
}

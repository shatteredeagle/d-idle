package main

import (
	"fmt"
	"strings"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/senses"
)

// fogStage is the -fog flag: "" serves the frozen fixture, "start" or "walked" serves the real senses.Publisher over a
// synthetic cave with a roam grid on every walkable cell (docs/mapfog.md), to look at what the map shows of the fog.
var fogStage string

func fogPublisher(stage string) (*senses.Publisher, error) {
	if stage != "start" && stage != "walked" {
		return nil, fmt.Errorf("uishots: -fog %q: want start or walked", stage)
	}
	rows := make([]string, 70)
	for i := range rows {
		rows[i] = strings.Repeat("#", 240)
	}
	carve := func(x0, x1, y0, y1 int) { // a winding corridor, 6 rows high
		for y := y0; y < y1; y++ {
			rows[y] = rows[y][:x0] + strings.Repeat(".", x1-x0) + rows[y][x1:]
		}
	}
	carve(10, 100, 10, 16)
	carve(94, 100, 10, 56)
	carve(94, 170, 50, 56)
	carve(164, 170, 20, 56)
	carve(164, 230, 20, 26)
	lm := &d2map.LevelMap{No: 2, Grid: d2map.ParseASCII(0, 0, rows...)}
	p := &senses.Publisher{}
	hero := d2map.Point{X: 14, Y: 13}
	if err := p.Observe(lm, "Cave Level 1", hero); err != nil {
		return nil, err
	}
	var cs senses.Coverage
	cs = senses.Coverage{Area: 2, Cell: 8, Cover: 5}
	for x := 12; x < 232; x += 8 {
		for y := 12; y < 58; y += 4 {
			if lm.Grid.Walkable(x, y) {
				cs.Cells = append(cs.Cells, [3]int{x, y, senses.CoverUnseen})
			}
		}
	}
	if stage == "walked" {
		for x := 14; x <= 98; x += 4 { // the corridor of the first leg, then down the bend
			hero = d2map.Point{X: x, Y: 13}
			p.Observe(lm, "Cave Level 1", hero)
		}
		for y := 13; y <= 40; y += 4 {
			hero = d2map.Point{X: 97, Y: y}
			p.Observe(lm, "Cave Level 1", hero)
		}
		for i, c := range cs.Cells {
			if dx, dy := c[0]-hero.X, c[1]-hero.Y; c[0] < 100 && c[1] < 40 && dx*dx+dy*dy >= 0 {
				cs.Cells[i][2] = senses.CoverSeen
			}
		}
	}
	cs.Target = &senses.Point{X: 200, Y: 24}
	p.SetCoverage(cs)
	return p, nil
}

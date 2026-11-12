package senses

import (
	"strings"
	"testing"

	"d2lod-bot/internal/d2map"
)

// Two level shapes: an open field and a winding cave corridor (synthetic, not recordings).
func fogLevels() map[string]*d2map.LevelMap {
	field := make([]string, 60)
	for i := range field {
		field[i] = strings.Repeat(".", 240)
	}
	cave := make([]string, 60)
	for i := range cave {
		cave[i] = strings.Repeat("#", 240)
	}
	for y := 28; y < 32; y++ { // a corridor 4 rows high with a bend
		cave[y] = strings.Repeat("#", 10) + strings.Repeat(".", 200) + strings.Repeat("#", 30)
	}
	return map[string]*d2map.LevelMap{
		"field": {No: 2, Grid: d2map.ParseASCII(0, 0, field...)},
		"cave":  {No: 3, Grid: d2map.ParseASCII(0, 0, cave...)},
	}
}

func coverOver(area int, hero d2map.Point) Coverage {
	c := Coverage{Area: area, Cell: 8, Cover: 5, Target: &Point{X: 230, Y: 30}}
	for x := 4; x < 240; x += 8 {
		for y := 28; y < 60; y += 8 {
			st := CoverUnseen
			if x == 4 && y == 28 {
				st = CoverSeen
			}
			c.Cells = append(c.Cells, [3]int{x, y, st})
		}
	}
	return c
}

func TestCoverageWithheldUntilExplored(t *testing.T) {
	for name, lm := range fogLevels() {
		t.Run(name, func(t *testing.T) {
			var p Publisher
			if err := p.Observe(lm, name, d2map.Point{X: 8, Y: 30}); err != nil {
				t.Fatal(err)
			}
			p.SetCoverage(coverOver(lm.No, d2map.Point{}))
			c, ok := p.CoverageOf()
			if !ok {
				t.Fatal("no coverage")
			}
			total := len(coverOver(lm.No, d2map.Point{}).Cells)
			if c.Total != total || c.Hidden+len(c.Cells) != total || c.Drawn != len(c.Cells) {
				t.Fatalf("counts total %d hidden %d cells %d", c.Total, c.Hidden, len(c.Cells))
			}
			if len(c.Cells) == 0 || len(c.Cells) > total/4 {
				t.Fatalf("fresh start shows %d of %d squares", len(c.Cells), total)
			}
			for _, q := range c.Cells {
				if dx, dy := q[0]-8, q[1]-30; dx*dx+dy*dy > (RevealRadius+fogBlock*2)*(RevealRadius+fogBlock*2) {
					t.Fatalf("square %v outside the explored mask", q)
				}
			}
			if c.Target != nil {
				t.Fatal("the target in the fog was sent")
			}
			// walk along the corridor: squares appear only where the hero went
			for x := 8; x <= 120; x += 8 {
				p.Observe(lm, name, d2map.Point{X: x, Y: 30})
			}
			d, _ := p.CoverageOf()
			if len(d.Cells) <= len(c.Cells) || d.Hidden >= c.Hidden {
				t.Fatalf("exploring revealed nothing: %d -> %d", len(c.Cells), len(d.Cells))
			}
			for _, q := range d.Cells {
				if q[0] > 120+RevealRadius+fogBlock*2 {
					t.Fatalf("square %v beyond the explored corridor", q)
				}
			}
			for x := 120; x <= 232; x += 8 {
				p.Observe(lm, name, d2map.Point{X: x, Y: 30})
			}
			if e, _ := p.CoverageOf(); e.Target == nil {
				t.Fatal("the target was not shown once explored")
			}
		})
	}
}

func TestEventsAndHeatWithheldInFog(t *testing.T) {
	lm := fogLevels()["field"]
	var p Publisher
	p.Observe(lm, "f", d2map.Point{X: 8, Y: 30})
	p.AddEventAt("roamfail", "near", 12, 30, 1)
	p.AddEventAt("roamfail", "far", 200, 30, 1)
	if es := p.EventsOf(); len(es) != 1 || es[0].Text != "near" {
		t.Fatalf("events %+v", es)
	}
	p.heat.cell(lm.No, 200, 30).kills = 1
	for _, h := range func() []HeatPoint { h, _ := p.HeatOf(); return h }() {
		if h.X > 100 {
			t.Fatalf("heat cell %v in the fog", h)
		}
	}
}

package nav

import (
	"errors"
	"testing"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

// pocketNav: a dead-end corridor (x0..x1, y0..y1) whose only way out passes the unwanted exit ex.
func pocketNav(t *testing.T, x0, y0, w, h int, ex, start input.Point) *Navigator {
	t.Helper()
	g := openField(x0, y0, w, h)
	_, n := newKnown(t, g, start, true)
	o, err := n.observe()
	if err != nil {
		t.Fatal(err)
	}
	o.lm.Exits = []d2map.Exit{{Dest: 8, Pos: d2map.Point{X: ex.X, Y: ex.Y}}}
	n.stayLevel = o.level
	return n
}

// g2g seeds 13 and 17: the same short click refused 40 to 150 times from inside a trigger disc until the failsafe ladder
// restarted the game. Positions are from the logs (V: Burial Grounds (114,169) -> (113,168), Black Marsh (87,287) ->
// (87,295)); the exit position and the dead-end walls are assumed (U: the logs carry no map). Two shapes: east-west and
// north-south corridors.
func TestRepeatedHoldInAPocketEscalates(t *testing.T) {
	for _, c := range []struct {
		name             string
		x0, y0, w, h     int
		ex, from, target input.Point
	}{
		{"burial grounds, east-west", 96, 164, 21, 9, input.Point{X: 100, Y: 169}, input.Point{X: 114, Y: 169}, input.Point{X: 113, Y: 168}},
		{"black marsh, north-south", 82, 280, 9, 22, input.Point{X: 87, Y: 298}, input.Point{X: 87, Y: 287}, input.Point{X: 87, Y: 295}},
	} {
		n := pocketNav(t, c.x0, c.y0, c.w, c.h, c.ex, c.from)
		ctl := n.in
		var sent bool
		for i := 1; i <= 3; i++ {
			q, err := n.gateClick(ctl, c.target)
			if i < gateHoldEscalate {
				if !errors.Is(err, ErrMouthGate) {
					t.Fatalf("%s: hold %d: err %v, want ErrMouthGate", c.name, i, err)
				}
				continue
			}
			if err != nil {
				t.Fatalf("%s: hold %d still refused: %v", c.name, i, err)
			}
			sent = true
			t.Logf("%s: escalated to %v", c.name, q)
		}
		if !sent {
			t.Fatalf("%s: never escalated", c.name)
		}
		// the pocket window is open: the next hold from a new spot answers at once
		if _, err := n.gateClick(ctl, c.target); err != nil {
			t.Fatalf("%s: pocket window closed early: %v", c.name, err)
		}
	}
}

// A first refusal, and a target inside the bound with the hero outside, stay refusals however often they repeat.
func TestHoldFromOutsideStaysARefusal(t *testing.T) {
	n := pocketNav(t, 0, 0, 120, 120, input.Point{X: 30, Y: 30}, input.Point{X: 46, Y: 30}) // 16 from the exit: clip is too short
	for i := 0; i < 10; i++ {
		if _, err := n.gateClick(n.in, input.Point{X: 40, Y: 30}); !errors.Is(err, ErrMouthGate) {
			t.Fatalf("hold %d: err %v, want ErrMouthGate", i, err)
		}
	}
}

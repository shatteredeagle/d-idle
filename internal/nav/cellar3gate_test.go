package nav

import (
	"context"
	"errors"
	"strings"
	"testing"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

// Tower Cellar 3 (card cellar3gate; docs/probes/sandbox-postdeploy-audit-2026-11-27.md problem 4, V log lines only): the hero
// stood at (45,20) / (42,20) next to the arrival stairs, "gate to level 24 is on the map but not reachable (cut off, no way
// out found)", three tries. The recorded grid of the level is NOT in the repo (docs/probes/cellar3-gate-capture.md has the
// capture commands, U), so the two maps are SYNTHETIC: a corridor with the arrival stairs (dest 22) and the gate (dest 24),
// and a thin barrier of the KNOWN grid between the hero and the rest (the real game has none: the pocket rule's own premise).
// U: that this is what the real map looks like; the mechanism (the pocket click refused by the mouth gate, silently) is V in
// the code and in the test.
//
//	map A: a hall 120 x 31; the hero 16 subtiles west of the stairs (outside the mouth gate's disc), the barrier 4 east of the
//	       hero, so the nearest connected cell lies 11 from the stairs (inside the disc); cells at the north end of the barrier
//	       lie outside it; the gate is far east.
//	map B: the same mirrored in x (the gate is far west).
type cellar3Map struct {
	name         string
	stairs, gate input.Point
	start        input.Point
	barrierX     int
}

var cellar3Maps = []cellar3Map{
	{name: "A", stairs: input.Point{X: 1054, Y: 1026}, gate: input.Point{X: 1110, Y: 1015}, start: input.Point{X: 1038, Y: 1024}, barrierX: 1042},
	{name: "B", stairs: input.Point{X: 1066, Y: 1026}, gate: input.Point{X: 1010, Y: 1015}, start: input.Point{X: 1082, Y: 1024}, barrierX: 1078},
}

func (c cellar3Map) build(real bool) (*sim, *Navigator) {
	corridor := d2map.Rect{X: 1000, Y: 1000, W: 120, H: 31}
	truth, known := openField(corridor.X, corridor.Y, corridor.W, corridor.H), openField(corridor.X, corridor.Y, corridor.W, corridor.H)
	bar := d2map.Rect{X: c.barrierX, Y: corridor.Y, W: 1, H: corridor.H}
	blockRect(known, bar)
	if real {
		blockRect(truth, bar)
	}
	l := &simLevel{no: 23, rect: corridor}
	l24 := &simLevel{no: 24, rect: d2map.Rect{X: c.gate.X - 2, Y: c.gate.Y - 6, W: 5, H: 13}}
	s := newSim(truth, c.start, l24, l)
	s.knownSrc = known
	s.reveal = 1000
	s.Update()
	s.reveal = 45
	l.lm.Adjacent = nil
	l.lm.Exits = []d2map.Exit{
		{Dest: 22, Pos: d2map.Point{X: c.stairs.X, Y: c.stairs.Y}},
		{Dest: 24, Pos: d2map.Point{X: c.gate.X, Y: c.gate.Y}},
	}
	n := s.navigator()
	n.w = fkSim{s}
	n.stayLevel, n.arrival = 23, true
	n.exploreFn = func(context.Context, func() bool, *input.Point) error { return ErrExplored }
	return s, n
}

// The pocket walk toward the connected cell goes toward the arrival stairs, inside the mouth gate's disc: the gate refused the
// click and leavePocket, which dropped the error, stood still for eight rounds and reported "cut off, no way out found". Now the
// walk takes a connected cell outside the unwanted exit's disc, or crosses the disc when nothing else is left (the gate is
// only a level change there). Must fail before the fix. Both maps.
func TestExitToLeavesAPocketBesideTheArrivalStairs(t *testing.T) {
	for _, c := range cellar3Maps {
		t.Run(c.name, func(t *testing.T) {
			s, n := c.build(false)
			err := n.ExitTo(ctxT(t), 24)
			if err != nil || s.cur().no != 24 {
				t.Fatalf("ExitTo: %v, level %d at %v, clicks %d", err, s.cur().no, s.pos(), len(s.clicks))
			}
		})
	}
}

// A barrier the game has too is not walked through: the exit is refused with ErrNoPath, with the exit and the cause in the text.
func TestExitToRefusesARealBarrierBesideTheStairs(t *testing.T) {
	for _, c := range cellar3Maps {
		t.Run(c.name, func(t *testing.T) {
			s, n := c.build(true)
			err := n.ExitTo(ctxT(t), 24)
			if !errors.Is(err, ErrNoPath) || s.cur().no == 24 || !strings.Contains(err.Error(), "cause=") {
				t.Fatalf("ExitTo: %v, level %d at %v", err, s.cur().no, s.pos())
			}
		})
	}
}

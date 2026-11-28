package nav

import (
	"context"
	"errors"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/input"
)

// doorWorld: the gateWorld wall with a 14-cell border pocket at y 3050..3063 that a door object (class 24) at (3002,3057)
// shuts off from the town side. Using the door opens the cells in front of it.
func doorWorld(t *testing.T) (*sim, d2map.Rect) {
	t.Helper()
	g := openField(2800, 3000, 400, 120)
	blockRect(g, d2map.Rect{X: 2999, Y: 3000, W: 2, H: 50})
	blockRect(g, d2map.Rect{X: 2999, Y: 3064, W: 2, H: 56})
	doorCells := d2map.Rect{X: 3001, Y: 3048, W: 2, H: 18}
	blockRect(g, doorCells)
	l1 := &simLevel{no: 1, rect: d2map.Rect{X: 3000, Y: 3000, W: 200, H: 120}}
	l2 := &simLevel{no: 2, rect: d2map.Rect{X: 2800, Y: 3000, W: 200, H: 120}}
	s := newSim(g, input.Point{X: 3100, Y: 3090}, l1, l2)
	s.reveal = 300 // the whole level is in view, like a fully known d2d map
	// the map lists the door only once the hero is near it (its room is up), like d2d's objects of the rooms laid
	s.onStep = func(s *sim) {
		if math.Hypot(s.px-3002, s.py-3057) < 60 && len(s.objects) == 0 {
			s.objects = append(s.objects, d2Unit(77, 24, 3002, 3057))
			s.levels[0].lm.Presets = []d2map.Preset{{Kind: d2map.PresetObject, ID: 24, Pos: d2map.Point{X: 3002, Y: 3057}}}
		}
	}
	return s, doorCells
}

func openDoor(s *sim, cells d2map.Rect) {
	open := make([]uint16, cells.W*cells.H)
	s.truth.SetBlock(cells, open)
	for _, l := range s.levels {
		if _, ok := l.known.Flags(cells.X, cells.Y); ok {
			l.known.SetBlock(cells, open)
		}
	}
}

func TestExitToOpensTheDoorBesideACutOffGate(t *testing.T) {
	s, cells := doorWorld(t)
	opened := 0
	s.onInteract = func(s *sim, tg input.Target) error {
		if tg.ID == 77 {
			opened++
			openDoor(s, cells)
		}
		return nil
	}
	n := s.navigator()
	n.doorFn = func(class int) bool { return class == 24 }
	n.exploreFn = func(_ context.Context, _ func() bool, _ *input.Point) error { return ErrExplored }
	if err := n.ExitTo(ctxT(t), 2); err != nil {
		t.Fatalf("exit: %v", err)
	}
	if opened != 1 || s.cur().no != 2 {
		t.Fatalf("door used %d times, level %d at %v", opened, s.cur().no, s.pos())
	}
}

// Must fail without the door step: a door that is not a door class is never clicked, and the exit is refused.
func TestExitToLeavesAnObjectThatIsNoDoorAlone(t *testing.T) {
	s, cells := doorWorld(t)
	s.onInteract = func(s *sim, tg input.Target) error { openDoor(s, cells); return nil }
	n := s.navigator()
	n.doorFn = func(int) bool { return false }
	n.exploreFn = func(_ context.Context, _ func() bool, _ *input.Point) error { return ErrExplored }
	err := n.ExitTo(ctxT(t), 2)
	if err == nil || len(s.interacts) != 0 {
		t.Fatalf("err %v interacts %v", err, s.interacts)
	}
	if !errors.Is(err, ErrNoPath) && !errors.Is(err, ErrNoExit) && !errors.Is(err, ErrTimeout) {
		t.Logf("refused with %v", err)
	}
}

// The plan to the gate looked complete when it was picked (the cells behind the door were read as open) and the walk
// then found them shut: the walk ends at "the closest reachable point" and ExitTo used to return that error without
// looking for the door (V live, scenario act1.tools.malus, Monastery Gate to Outer Cloister). Must fail without
// doorAfterShortWalk. Both classes: the geometry knows no class, the navigator is built from the same world.
func TestExitToOpensTheDoorAfterAShortWalk(t *testing.T) {
	for _, cls := range []string{"Paladin", "Sorceress"} {
		t.Run(cls, func(t *testing.T) {
			s, cells := doorWorld(t)
			// the known grid shows the door cells open until the hero is near, then reads them shut like the truth
			base := s.onStep
			doorOpen := false
			s.revealRooms() // the first read already shows the cells open
			s.levels[0].known.SetBlock(cells, make([]uint16, cells.W*cells.H))
			s.onStep = func(s *sim) {
				base(s)
				near := math.Hypot(s.px-3002, s.py-3057) < 60
				l := s.levels[0]
				if _, ok := l.known.Flags(cells.X, cells.Y); !ok {
					return
				}
				if !near && !doorOpen {
					l.known.SetBlock(cells, make([]uint16, cells.W*cells.H))
				}
				if near {
					if !doorOpen {
						// a newly read cell: the planner rebuilds its field when the known count changes
						l.known.SetBlock(d2map.Rect{X: 2000, Y: 2000, W: 1, H: 1}, []uint16{0})
					}
					doorOpen = true
					// back to the truth: the cells are wall until the door is used
					l.known.SetBlock(cells, truthCells(s, cells))
				}
			}
			opened := 0
			s.onInteract = func(s *sim, tg input.Target) error {
				if tg.ID == 77 {
					opened++
					openDoor(s, cells)
				}
				return nil
			}
			n := s.navigator()
			n.doorFn = func(class int) bool { return class == 24 }
			n.exploreFn = func(_ context.Context, _ func() bool, _ *input.Point) error { return ErrExplored }
			if err := n.ExitTo(ctxT(t), 2); err != nil {
				t.Fatalf("%s: exit: %v", cls, err)
			}
			if opened != 1 || s.cur().no != 2 {
				t.Fatalf("%s: door used %d times, level %d at %v", cls, opened, s.cur().no, s.pos())
			}
		})
	}
}

func truthCells(s *sim, r d2map.Rect) []uint16 {
	out := make([]uint16, 0, r.W*r.H)
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			f, _ := s.truth.Flags(x, y)
			out = append(out, f)
		}
	}
	return out
}

// A walk along the bottom of a level keeps off the strip that touches another level (V live: it stepped onto the strip
// and ended in the neighbour). Must fail without penalizeBorders.
func TestWalkKeepsOffTheBorderStripOfAnotherLevel(t *testing.T) {
	for _, cls := range []string{"Paladin", "Sorceress"} {
		t.Run(cls, func(t *testing.T) {
			g := openField(3000, 3000, 200, 80)
			blockRect(g, d2map.Rect{X: 3080, Y: 3000, W: 40, H: 56})
			l1 := &simLevel{no: 1, rect: d2map.Rect{X: 3000, Y: 3000, W: 200, H: 60}}
			l3 := &simLevel{no: 3, rect: d2map.Rect{X: 3000, Y: 3060, W: 200, H: 20}}
			s := newSim(g, input.Point{X: 3010, Y: 3058}, l1, l3)
			s.reveal = 300
			s.revealRooms()
			n := s.navigator()
			wps, err := n.PathTo(input.Point{X: 3190, Y: 3058})
			if err != nil {
				t.Fatal(err)
			}
			for _, w := range wps {
				if w.X > 3085 && w.X < 3115 && w.Y > 3057 {
					t.Fatalf("waypoint %v hugs the border strip: %v", w, wps)
				}
			}
		})
	}
}

// The door budget is per ExitTo call: the level is read again after a town trip with its doors shut. Must fail
// without the reset in ExitTo (the third visit used to refuse the door).
func TestExitToOpensTheSameDoorOnEveryVisit(t *testing.T) {
	for _, cls := range []string{"Paladin", "Sorceress"} {
		t.Run(cls, func(t *testing.T) {
			s, cells := doorWorld(t)
			s.onInteract = func(s *sim, tg input.Target) error {
				if tg.ID == 77 {
					openDoor(s, cells)
				}
				return nil
			}
			n := s.navigator()
			n.doorFn = func(class int) bool { return class == 24 }
			n.exploreFn = func(_ context.Context, _ func() bool, _ *input.Point) error { return ErrExplored }
			n.doorTried = map[d2map.Point]int{{X: 3002, Y: 3057}: maxDoorTries} // earlier visits used it up
			if err := n.ExitTo(ctxT(t), 2); err != nil {
				t.Fatalf("exit: %v", err)
			}
		})
	}
}

// The nearest reachable cell toward a cut-off gate must not lie on the strip toward another level: walking there
// changes the level (V live, scenario act1.tools.barracks: the Monastery Gate arrival pocket, the cell chosen was on
// the south strip and the hero ended in the Tamoe Highland). Must fail without the strip filter.
func TestApproachCellKeepsOffOtherLevelsStrip(t *testing.T) {
	for _, cls := range []string{"Paladin", "Sorceress"} {
		t.Run(cls, func(t *testing.T) {
			g := openField(3000, 3000, 200, 120)
			l1 := &simLevel{no: 1, rect: d2map.Rect{X: 3000, Y: 3000, W: 200, H: 120}}
			l3 := &simLevel{no: 3, rect: d2map.Rect{X: 3000, Y: 3120, W: 200, H: 40}}
			s := newSim(g, input.Point{X: 3100, Y: 3050}, l1, l3)
			s.reveal = 300
			s.revealRooms()
			n := s.navigator()
			o, err := n.observeWait(ctxT(t))
			if err != nil {
				t.Fatal(err)
			}
			cell, ok := n.nearestReachableTo(o, input.Point{X: 3100, Y: 3140}, 2)
			if !ok {
				t.Fatal("no cell")
			}
			if cell.Y >= 3117 {
				t.Fatalf("cell %v is on or beside the strip toward level 3", cell)
			}
		})
	}
}

// A spot behind a closed door inside one level: GoTo opens the door between and walks on (V live, scenario
// act1.tools.malus: the Smith's room in the Barracks, "reached the closest reachable point"). Must fail without the
// door step in GoTo.
func TestGoToOpensTheDoorToASpotBehindIt(t *testing.T) {
	for _, cls := range []string{"Paladin", "Sorceress"} {
		t.Run(cls, func(t *testing.T) {
			g := openField(3000, 3000, 200, 100)
			blockRect(g, d2map.Rect{X: 3100, Y: 3000, W: 2, H: 100})
			cells := d2map.Rect{X: 3100, Y: 3048, W: 2, H: 18}
			l1 := &simLevel{no: 1, rect: d2map.Rect{X: 3000, Y: 3000, W: 200, H: 100}}
			s := newSim(g, input.Point{X: 3010, Y: 3057}, l1)
			s.reveal = 300
			s.onStep = func(s *sim) {
				if math.Hypot(s.px-3101, s.py-3057) < 60 && len(s.objects) == 0 {
					s.objects = append(s.objects, d2Unit(77, 24, 3101, 3057))
					s.levels[0].lm.Presets = []d2map.Preset{{Kind: d2map.PresetObject, ID: 24, Pos: d2map.Point{X: 3101, Y: 3057}}}
				}
			}
			opened := 0
			s.onInteract = func(s *sim, tg input.Target) error {
				if tg.ID == 77 {
					opened++
					openDoor(s, cells)
				}
				return nil
			}
			n := s.navigator()
			n.doorFn = func(class int) bool { return class == 24 }
			if err := n.GoTo(ctxT(t), input.Point{X: 3190, Y: 3057}); err != nil {
				t.Fatalf("goto: %v", err)
			}
			if opened != 1 || dist(s.pos(), input.Point{X: 3190, Y: 3057}) > 6 {
				t.Fatalf("door used %d times, at %v", opened, s.pos())
			}
		})
	}
}

// Two doors in a row between the hero and the spot: no single door joins both sides, GoTo opens the near one, walks, and
// opens the next (the Barracks corridor of the Smith, V lab: door (125,360) opened, then "closest reachable point"
// again). Must fail without doorOnTheWay.
func TestGoToOpensTwoDoorsInARow(t *testing.T) {
	for _, cls := range []string{"Paladin", "Sorceress"} {
		t.Run(cls, func(t *testing.T) {
			g := openField(3000, 3000, 240, 100)
			blockRect(g, d2map.Rect{X: 3080, Y: 3000, W: 2, H: 100})
			blockRect(g, d2map.Rect{X: 3160, Y: 3000, W: 2, H: 100})
			c1 := d2map.Rect{X: 3080, Y: 3048, W: 2, H: 18}
			c2 := d2map.Rect{X: 3160, Y: 3048, W: 2, H: 18}
			l1 := &simLevel{no: 1, rect: d2map.Rect{X: 3000, Y: 3000, W: 240, H: 100}}
			s := newSim(g, input.Point{X: 3010, Y: 3057}, l1)
			s.reveal = 300
			s.levels[0].lm.Presets = []d2map.Preset{
				{Kind: d2map.PresetObject, ID: 24, Pos: d2map.Point{X: 3081, Y: 3057}},
				{Kind: d2map.PresetObject, ID: 24, Pos: d2map.Point{X: 3161, Y: 3057}},
			}
			s.objects = []d2.Unit{d2Unit(77, 24, 3081, 3057), d2Unit(78, 24, 3161, 3057)}
			opened := 0
			s.onInteract = func(s *sim, tg input.Target) error {
				switch tg.ID {
				case 77:
					opened++
					openDoor(s, c1)
				case 78:
					opened++
					openDoor(s, c2)
				}
				return nil
			}
			n := s.navigator()
			n.doorFn = func(class int) bool { return class == 24 }
			if err := n.GoTo(ctxT(t), input.Point{X: 3230, Y: 3057}); err != nil {
				t.Fatalf("goto: %v", err)
			}
			if opened != 2 || dist(s.pos(), input.Point{X: 3230, Y: 3057}) > 6 {
				t.Fatalf("doors used %d, at %v", opened, s.pos())
			}
		})
	}
}

// barracksDoors are the door presets (class, x, y in subtiles) of the live Barracks capture of 2026-11-23
// (testdata/barracks28.grid, 200 by 350 subtiles, read-only `debug grid 5`; each door is a one cell gap in a wall).
var barracksDoors = [][3]int{{16, 125, 185}, {15, 100, 175}, {16, 125, 165}, {16, 120, 130}, {15, 100, 105}, {16, 120, 80},
	{16, 75, 130}, {15, 50, 105}, {16, 75, 80}, {15, 100, 35}, {16, 65, 20}, {15, 30, 90}, {16, 125, 255}, {15, 100, 245},
	{16, 125, 220}, {15, 80, 230}, {15, 165, 245}, {15, 150, 245}, {16, 175, 290}, {14, 175, 180}}

// The Smith's spawn point lay behind a chain of closed doors, the nearest of them 22 subtiles from where the walk
// ended ("reached the closest reachable point {152 267} to {73 267}", sandbox 2026-11-23): the door within 14 subtiles
// did not exist. GoTo plans through the doors, opens the first closed one on that path, and walks on. Must fail without
// doorOnPath (the old reach found no door). Both classes: the geometry knows none.
func TestGoToOpensTheDoorsOfTheLiveBarracks(t *testing.T) {
	data, err := os.ReadFile("testdata/barracks28.grid")
	if err != nil {
		t.Fatal(err)
	}
	rows := strings.Split(strings.TrimSpace(string(data)), "\n")
	const ox, oy = 1000, 1000
	for _, cls := range []string{"Paladin", "Sorceress"} {
		t.Run(cls, func(t *testing.T) {
			g := d2map.NewGrid()
			cells := make([]uint16, len(rows[0])*len(rows))
			for y, r := range rows {
				for x, c := range r {
					if c == '#' {
						cells[y*len(rows[0])+x] = 1
					}
				}
			}
			g.SetBlock(d2map.Rect{X: ox, Y: oy, W: len(rows[0]), H: len(rows)}, cells)
			var presets []d2map.Preset
			closed := map[int]bool{}
			for i, d := range barracksDoors { // every door shut: its gap becomes wall
				g.SetBlock(d2map.Rect{X: ox + d[1] - 1, Y: oy + d[2], W: 3, H: 1}, []uint16{1, 1, 1})
				presets = append(presets, d2map.Preset{Kind: d2map.PresetObject, ID: d[0], Pos: d2map.Point{X: ox + d[1], Y: oy + d[2]}})
				closed[i] = true
			}
			l1 := &simLevel{no: 28, rect: d2map.Rect{X: ox, Y: oy, W: len(rows[0]), H: len(rows)}}
			s := newSim(g, input.Point{X: ox + 152, Y: oy + 267}, l1)
			s.reveal = 400
			s.revealRooms()
			s.levels[0].lm.Presets = presets
			opened := 0
			s.onInteract = func(s *sim, tg input.Target) error {
				for i, d := range barracksDoors {
					if closed[i] && dist(tg.Pos, input.Point{X: ox + d[1], Y: oy + d[2]}) < 8 {
						closed[i] = false
						opened++
						openDoor(s, d2map.Rect{X: ox + d[1] - 1, Y: oy + d[2], W: 3, H: 1})
						return nil
					}
				}
				return nil
			}
			n := s.navigator()
			n.doorFn = func(class int) bool { return class >= 14 && class <= 16 }
			goal := input.Point{X: ox + 73, Y: oy + 267}
			if err := n.GoTo(ctxT(t), goal); err != nil {
				t.Fatalf("goto: %v (doors opened %d, at %v)", err, opened, s.pos())
			}
			if opened == 0 || dist(s.pos(), goal) > 8 {
				t.Fatalf("doors opened %d, at %v", opened, s.pos())
			}
		})
	}
}

// The real Barracks (live capture 2026-11-23): the hero arrives from the Outer Cloister at about (191,324), 25 subtiles above
// the south strip (open cells x 179..182 of row 349), and d2d never reads across that strip (pending stays 1, nothing
// proves). The way back is to cross the unproven strip (stage 1 of the Necromancer drill; run 20261123-023115-028d,
// `nav no usable gate from 28 to 27 ... borderCells 0 unproven 4 pending 1`). Both classes: the geometry knows none.
func TestExitToCrossesTheLiveBarracksSouthStrip(t *testing.T) {
	data, err := os.ReadFile("testdata/barracks28.grid")
	if err != nil {
		t.Fatal(err)
	}
	rows := strings.Split(strings.TrimSpace(string(data)), "\n")
	const ox, oy = 1000, 1000
	for _, cls := range []string{"Paladin", "Necromancer"} {
		t.Run(cls, func(t *testing.T) {
			g := d2map.NewGrid()
			cells := make([]uint16, len(rows[0])*len(rows))
			for y, r := range rows {
				for x, c := range r {
					if c == '#' {
						cells[y*len(rows[0])+x] = 1
					}
				}
			}
			g.SetBlock(d2map.Rect{X: ox, Y: oy, W: len(rows[0]), H: len(rows)}, cells)
			g.SetBlock(d2map.Rect{X: ox + 100, Y: oy + 350, W: 200, H: 120}, make([]uint16, 200*120)) // the Outer Cloister side, level 27
			l28 := &simLevel{no: 28, rect: d2map.Rect{X: ox, Y: oy, W: len(rows[0]), H: len(rows)}}
			l27 := &simLevel{no: 27, rect: d2map.Rect{X: ox + 100, Y: oy + 350, W: 200, H: 120}}
			s := newSim(g, input.Point{X: ox + 191, Y: oy + 324}, l28, l27)
			s.reveal = 400
			s.revealRooms()
			l := s.levels[0]
			adj := l.lm.Adjacent
			if len(adj) == 0 {
				t.Skip("the sim builds no adjacency for this layout")
			}
			l.lm.Adjacent, l.lm.Pending = nil, 1
			l.lm.Unproven = []d2map.Adjacent{{LevelNo: 27, Rect: adj[0].Rect, Contact: adj[0].Contact}}
			s.onStep = func(s *sim) { l.lm.Adjacent, l.lm.Pending = nil, 1 } // d2d never reads across
			n := s.navigator()
			n.exploreFn = func(context.Context, func() bool, *input.Point) error { return ErrExplored }
			if err := n.ExitTo(ctxT(t), 27); err != nil {
				t.Fatalf("%s: %v (at %v)", cls, err, s.pos())
			}
			if s.cur().no != 27 {
				t.Fatalf("%s: still in level %d at %v", cls, s.cur().no, s.pos())
			}
		})
	}
}

// The frontier of the live Barracks run (nav.barracks-return, lab night 2026-11-28) held 131 unproven cells, not 4: a
// sealed pocket cell that sorts before the real gap shifts its four cells to ranks 1..4, and the old "every fifth cell"
// sample then planned only the pocket and found no target ("nav no usable gate", 44 x "no exit to that level"). Must
// fail with the rank sample. Both classes: the geometry knows none.
func TestExitToFindsTheStripBehindASealedFrontierPocket(t *testing.T) {
	data, err := os.ReadFile("testdata/barracks28.grid")
	if err != nil {
		t.Fatal(err)
	}
	rows := strings.Split(strings.TrimSpace(string(data)), "\n")
	const ox, oy = 1000, 1000
	for _, cls := range []string{"Paladin", "Necromancer"} {
		t.Run(cls, func(t *testing.T) {
			g := d2map.NewGrid()
			cells := make([]uint16, len(rows[0])*len(rows))
			for y, r := range rows {
				for x, c := range r {
					if c == '#' {
						cells[y*len(rows[0])+x] = 1
					}
				}
			}
			// a one cell pocket in the wall beside the south row, nearer to the hero (28.6) than the strip cells (30.4 and more)
			hero := input.Point{X: 190, Y: 320}
			pocket := input.Point{X: 196, Y: 348}
			cells[pocket.Y*len(rows[0])+pocket.X] = 0
			g.SetBlock(d2map.Rect{X: ox, Y: oy, W: len(rows[0]), H: len(rows)}, cells)
			g.SetBlock(d2map.Rect{X: ox + 100, Y: oy + 350, W: 200, H: 120}, make([]uint16, 200*120))
			l28 := &simLevel{no: 28, rect: d2map.Rect{X: ox, Y: oy, W: len(rows[0]), H: len(rows)}}
			l27 := &simLevel{no: 27, rect: d2map.Rect{X: ox + 100, Y: oy + 350, W: 200, H: 120}}
			s := newSim(g, input.Point{X: ox + hero.X, Y: oy + hero.Y}, l28, l27)
			s.reveal = 400
			s.revealRooms()
			l := s.levels[0]
			adj := l.lm.Adjacent
			if len(adj) == 0 {
				t.Skip("the sim builds no adjacency for this layout")
			}
			strip := []d2map.Adjacent{{LevelNo: 27, Rect: adj[0].Rect, Contact: adj[0].Contact}}
			// the pocket sorts first, the four strip cells take ranks 1..4
			unproven := append([]d2map.Adjacent{{LevelNo: 27, Rect: adj[0].Rect, Contact: d2map.Rect{X: ox + pocket.X, Y: oy + pocket.Y, W: 1, H: 1}}}, strip...)
			l.lm.Adjacent, l.lm.Pending, l.lm.Unproven = nil, 1, unproven
			s.onStep = func(s *sim) { l.lm.Adjacent, l.lm.Pending = nil, 1 }
			n := s.navigator()
			n.exploreFn = func(context.Context, func() bool, *input.Point) error { return ErrExplored }
			if err := n.ExitTo(ctxT(t), 27); err != nil {
				t.Fatalf("%s: %v (at %v)", cls, err, s.pos())
			}
			if s.cur().no != 27 {
				t.Fatalf("%s: still in level %d at %v", cls, s.cur().no, s.pos())
			}
		})
	}
}

// A door that takes 2.5 s to open (the grid still reads it shut right after the use) is waited for, not used again: a
// second use of an open door closes it (card qdmatrix, tomb doors of Act 2). Both classes read the same geometry.
func TestExitToWaitsForADoorThatOpensSlowly(t *testing.T) {
	for _, cls := range []string{"Paladin", "Sorceress"} {
		t.Run(cls, func(t *testing.T) {
			s, cells := doorWorld(t)
			var openAt time.Time
			base := s.onStep
			s.onStep = func(s *sim) {
				base(s)
				if !openAt.IsZero() && !s.now.Before(openAt) {
					openAt = time.Time{}
					openDoor(s, cells)
				}
			}
			uses := 0
			s.onInteract = func(s *sim, tg input.Target) error {
				if tg.ID == 77 {
					uses++
					openAt = s.now.Add(2500 * time.Millisecond)
				}
				return nil
			}
			n := s.navigator()
			n.doorFn = func(class int) bool { return class == 24 }
			n.exploreFn = func(_ context.Context, _ func() bool, _ *input.Point) error { return ErrExplored }
			if err := n.ExitTo(ctxT(t), 2); err != nil {
				t.Fatalf("%s: exit: %v", cls, err)
			}
			if uses != 1 || s.cur().no != 2 {
				t.Fatalf("%s: door used %d times, level %d", cls, uses, s.cur().no)
			}
		})
	}
}

// A unit (a chest, an NPC) behind a closed door: GoToUnit opens the door between and walks on, as GoTo does for a spot (V
// drill.act2.cube: "unreachable from" at the Cube chest). Must fail without the door step in GoToUnitOf.
func TestGoToUnitOpensTheDoorToAUnitBehindIt(t *testing.T) {
	for _, cls := range []string{"Paladin", "Sorceress"} {
		t.Run(cls, func(t *testing.T) {
			g := openField(3000, 3000, 200, 100)
			blockRect(g, d2map.Rect{X: 3100, Y: 3000, W: 2, H: 100})
			cells := d2map.Rect{X: 3100, Y: 3048, W: 2, H: 18}
			l1 := &simLevel{no: 1, rect: d2map.Rect{X: 3000, Y: 3000, W: 200, H: 100}}
			s := newSim(g, input.Point{X: 3010, Y: 3057}, l1)
			s.reveal = 300
			chest := d2Unit(88, 580, 3150, 3057)
			s.objects = append(s.objects, chest)
			s.onStep = func(s *sim) {
				if math.Hypot(s.px-3101, s.py-3057) < 60 && len(s.levels[0].lm.Presets) == 0 {
					s.objects = append(s.objects, d2Unit(77, 24, 3101, 3057))
					s.levels[0].lm.Presets = []d2map.Preset{{Kind: d2map.PresetObject, ID: 24, Pos: d2map.Point{X: 3101, Y: 3057}}}
				}
			}
			opened := 0
			s.onInteract = func(s *sim, tg input.Target) error {
				if tg.ID == 77 {
					opened++
					openDoor(s, cells)
				}
				return nil
			}
			n := s.navigator()
			n.doorFn = func(class int) bool { return class == 24 }
			if err := n.GoToUnitOf(ctxT(t), d2.TypeObject, 88); err != nil {
				t.Fatalf("gotounit: %v", err)
			}
			if opened != 1 || dist(s.pos(), input.Point{X: 3150, Y: 3057}) > 6 {
				t.Fatalf("door used %d times, at %v", opened, s.pos())
			}
		})
	}
}

// Two doors in a row between the hero and a GATE: no single door joins both sides, ExitTo opens the first on the way, walks,
// and opens the next (V drill.act2.amulet: Claw Viper Temple 1, "cut-off gate, no door beside it, nearest preset 45").
// Must fail without the chain step for gates.
func TestExitToOpensTwoDoorsInARow(t *testing.T) {
	for _, cls := range []string{"Paladin", "Sorceress"} {
		t.Run(cls, func(t *testing.T) {
			g := openField(3000, 3000, 340, 100)
			blockRect(g, d2map.Rect{X: 3080, Y: 3000, W: 2, H: 100})
			blockRect(g, d2map.Rect{X: 3160, Y: 3000, W: 2, H: 100})
			c1 := d2map.Rect{X: 3080, Y: 3048, W: 2, H: 18}
			c2 := d2map.Rect{X: 3160, Y: 3048, W: 2, H: 18}
			l1 := &simLevel{no: 1, rect: d2map.Rect{X: 3000, Y: 3000, W: 240, H: 100}}
			l2 := &simLevel{no: 2, rect: d2map.Rect{X: 3240, Y: 3000, W: 100, H: 100}}
			s := newSim(g, input.Point{X: 3010, Y: 3057}, l1, l2)
			s.reveal = 400
			s.levels[0].lm.Presets = []d2map.Preset{
				{Kind: d2map.PresetObject, ID: 24, Pos: d2map.Point{X: 3081, Y: 3057}},
				{Kind: d2map.PresetObject, ID: 24, Pos: d2map.Point{X: 3161, Y: 3057}},
			}
			s.objects = []d2.Unit{d2Unit(77, 24, 3081, 3057), d2Unit(78, 24, 3161, 3057)}
			opened := 0
			s.onInteract = func(s *sim, tg input.Target) error {
				switch tg.ID {
				case 77:
					opened++
					openDoor(s, c1)
				case 78:
					opened++
					openDoor(s, c2)
				}
				return nil
			}
			n := s.navigator()
			n.doorFn = func(class int) bool { return class == 24 }
			n.exploreFn = func(_ context.Context, _ func() bool, _ *input.Point) error { return ErrExplored }
			if err := n.ExitTo(ctxT(t), 2); err != nil {
				t.Fatalf("%s: exit: %v", cls, err)
			}
			if opened != 2 || s.cur().no != 2 {
				t.Fatalf("%s: doors used %d, level %d at %v", cls, opened, s.cur().no, s.pos())
			}
		})
	}
}

// The Act 5 town gate (class 449, name "gate") opens like a door although objects.txt does not flag it IsDoor (V lab
// drill.act5.siege); the posts and flags beside it (436, 441) and an ordinary selectable object do not.
func TestHarrogathGateOpensLikeADoor(t *testing.T) {
	for _, c := range []struct {
		name string
		ob   *data.Object
		want bool
	}{
		{"door", &data.Object{ID: 24, Name: "Door", IsDoor: true}, true},
		{"town gate", &data.Object{ID: 449, Name: "gate", Selectable: true}, true},
		{"post", &data.Object{ID: 436, Name: "ettr"}, false},
		{"chest", &data.Object{ID: 580, Name: "chest", Selectable: true}, false},
		{"none", nil, false},
	} {
		if got := opensLikeADoor(c.ob); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}

// A town reached by portal from the Outer Cloister is no neighbour of it: learn() must not record the pair, or the next
// route from the camp starts with ExitTo(27) (V live 2026-11-23). The camp and the Blood Moor stay joined. Must fail
// without the town rule in learn. The graph is data, no class: both classes share the navigator.
func TestLearnIgnoresATownPortalCrossing(t *testing.T) {
	for _, cls := range []string{"Paladin", "Sorceress"} {
		t.Run(cls, func(t *testing.T) {
			g := openField(3000, 3000, 100, 100)
			l1 := &simLevel{no: 1, rect: d2map.Rect{X: 3000, Y: 3000, W: 100, H: 100}}
			s := newSim(g, input.Point{X: 3050, Y: 3050}, l1)
			n := s.navigator()
			n.learn(27, 1)
			n.learn(40, 28)
			if n.edges()[1][27] != 0 || n.edges()[27][1] != 0 {
				t.Errorf("1 and 27 became neighbours: %v", n.edges()[1])
			}
			route, err := n.Route(1, 28)
			if err != nil || len(route) == 0 || route[0] == 27 {
				t.Errorf("route %v %v", route, err)
			}
			n.learn(1, 2)
			if _, ok := n.edges()[1][2]; !ok {
				t.Error("the camp and the Blood Moor must stay joined")
			}
		})
	}
}

// V live 2026-11-24 (Andariel try-anyway, Jail Level 1): the way to the stairs to Jail Level 2 crosses eight doors one
// after the other; every door opened used one of ExitTo's eight attempts, so the walk ended "walking out to level 30 did
// not change the level" at the eighth door. Ten doors in a row must still be crossed in one call (both classes: the nav does
// not read the class). Must fail while a door opened costs an attempt.
func TestExitToCrossesTenDoorsInARow(t *testing.T) {
	const doors = 10
	for _, cls := range []string{"Paladin", "Necromancer"} {
		t.Run(cls, func(t *testing.T) {
			w := 40*doors + 160
			g := openField(3000, 3000, w+140, 100)
			var cells []d2map.Rect
			var objs []d2.Unit
			var pre []d2map.Preset
			for i := 0; i < doors; i++ {
				x := 3080 + 40*i
				blockRect(g, d2map.Rect{X: x, Y: 3000, W: 2, H: 100})
				cells = append(cells, d2map.Rect{X: x, Y: 3048, W: 2, H: 18})
				objs = append(objs, d2Unit(uint32(100+i), 24, x+1, 3057))
				pre = append(pre, d2map.Preset{Kind: d2map.PresetObject, ID: 24, Pos: d2map.Point{X: x + 1, Y: 3057}})
			}
			l1 := &simLevel{no: 1, rect: d2map.Rect{X: 3000, Y: 3000, W: w, H: 100}}
			l2 := &simLevel{no: 2, rect: d2map.Rect{X: 3000 + w, Y: 3000, W: 100, H: 100}}
			s := newSim(g, input.Point{X: 3010, Y: 3057}, l1, l2)
			s.reveal = 600
			s.levels[0].lm.Presets = pre
			s.objects = objs
			opened := 0
			s.onInteract = func(s *sim, tg input.Target) error {
				if i := int(tg.ID) - 100; i >= 0 && i < doors {
					opened++
					openDoor(s, cells[i])
				}
				return nil
			}
			n := s.navigator()
			n.doorFn = func(class int) bool { return class == 24 }
			n.exploreFn = func(_ context.Context, _ func() bool, _ *input.Point) error { return ErrExplored }
			if err := n.ExitTo(ctxT(t), 2); err != nil {
				t.Fatalf("%s: exit after %d doors: %v", cls, opened, err)
			}
			if opened != doors || s.cur().no != 2 {
				t.Fatalf("%s: doors used %d of %d, level %d at %v", cls, opened, doors, s.cur().no, s.pos())
			}
		})
	}
}

// The gates of an unproven strip at several depths (a frontier): each segment is a gate of its own, the orientation comes
// from the cells (wide along the edge), and a cell deeper than the edge pushes from the neighbour's edge so the push lands
// inside the neighbour (live 2026-11-24, Inner Cloister to Cathedral: a push of 8 from a cell 2 rows inside ended on the wall).
// Two neighbours, north and west, so neither axis is hard coded. Must fail with the push taken from the cell.
func TestGatesFromAFrontierPushFromTheNeighboursEdge(t *testing.T) {
	g := openField(3000, 3000, 200, 200)
	l1 := &simLevel{no: 1, rect: d2map.Rect{X: 3000, Y: 3000, W: 200, H: 200}}
	s := newSim(g, input.Point{X: 3100, Y: 3150}, l1)
	n := s.navigator()
	t.Run("north", func(t *testing.T) {
		nb := d2map.Rect{X: 3000, Y: 2800, W: 200, H: 200} // the neighbour above: its bottom row is our top
		adj := []d2map.Adjacent{{LevelNo: 2, Rect: nb, Contact: d2map.Rect{X: 3140, Y: 3000, W: 4, H: 1}}}
		cells := []d2map.Point{{X: 3140, Y: 3000}, {X: 3141, Y: 3000}, {X: 3142, Y: 3000}, {X: 3050, Y: 3008}, {X: 3051, Y: 3008}, {X: 3052, Y: 3008}}
		gates := n.gatesFromCells(s.levels[0].lm, 2, cells, adj)
		if len(gates) != 2 {
			t.Fatalf("%d gates, want a corridor gate and a deep gate: %+v", len(gates), gates)
		}
		if d := map[int]bool{gates[0].depth: true, gates[1].depth: true}; !d[0] || !d[8] {
			t.Errorf("gate depths %d and %d, want a notch at the edge (0) and the far side (8)", gates[0].depth, gates[1].depth)
		}
		for _, gt := range gates {
			if gt.push.Y > nb.Bottom()-1 {
				t.Errorf("gate %v pushes to %v: not inside the neighbour (its bottom row is %d)", gt.at, gt.push, nb.Bottom())
			}
			if gt.push.X != gt.at.X {
				t.Errorf("gate %v pushes sideways to %v", gt.at, gt.push)
			}
		}
	})
	t.Run("west", func(t *testing.T) {
		nb := d2map.Rect{X: 2800, Y: 3000, W: 200, H: 200} // the neighbour to the left: its right column is our left
		adj := []d2map.Adjacent{{LevelNo: 3, Rect: nb, Contact: d2map.Rect{X: 3000, Y: 3100, W: 1, H: 4}}}
		cells := []d2map.Point{{X: 3000, Y: 3100}, {X: 3000, Y: 3101}, {X: 3000, Y: 3102}, {X: 3008, Y: 3050}, {X: 3008, Y: 3051}, {X: 3008, Y: 3052}}
		gates := n.gatesFromCells(s.levels[0].lm, 3, cells, adj)
		if len(gates) != 2 {
			t.Fatalf("%d gates: %+v", len(gates), gates)
		}
		for _, gt := range gates {
			if gt.push.X > nb.Right()-1 || gt.push.Y != gt.at.Y {
				t.Errorf("gate %v pushes to %v: not inside the western neighbour (its right column is %d)", gt.at, gt.push, nb.Right())
			}
		}
	})
}

// V lab devp27 (`debug objectlist`) and live 2026-11-24: the way from the Inner Cloister to the Cathedral is a closed door in
// the notch of the north wall, and d2d does not link the neighbour (pending, an unproven strip) until it is open. The unproven
// gate is pushed at for ever without it; the nav opens the door that shuts the gate first. Both classes (the nav does not read
// the class). Must fail without openDoorAtGate: the door is never used and the exit fails.
func TestExitToOpensTheDoorThatShutsAnUnprovenBorderGate(t *testing.T) {
	for _, cls := range []string{"Paladin", "Necromancer"} {
		t.Run(cls, func(t *testing.T) {
			g := openField(2800, 3000, 400, 120)
			blockRect(g, d2map.Rect{X: 2999, Y: 3000, W: 2, H: 50})
			blockRect(g, d2map.Rect{X: 2999, Y: 3065, W: 2, H: 55})
			door := d2map.Rect{X: 2999, Y: 3050, W: 2, H: 15}
			blockRect(g, door) // the doorway is shut
			l1 := &simLevel{no: 1, rect: d2map.Rect{X: 3000, Y: 3000, W: 200, H: 120}}
			l2 := &simLevel{no: 2, rect: d2map.Rect{X: 2800, Y: 3000, W: 200, H: 120}}
			s := newSim(g, input.Point{X: 3100, Y: 3090}, l1, l2)
			s.reveal = 300
			l1.lm.Presets = []d2map.Preset{{Kind: d2map.PresetObject, ID: 24, Pos: d2map.Point{X: 3000, Y: 3057}}}
			s.objects = []d2.Unit{d2Unit(77, 24, 3000, 3057)}
			contact := d2map.Rect{X: 3002, Y: 3050, W: 1, H: 15} // the first walkable column behind the shut doorway
			l1.lm.Adjacent, l1.lm.Pending = nil, 1
			l1.lm.Unproven = []d2map.Adjacent{{LevelNo: 2, Rect: d2map.Rect{X: 2800, Y: 3000, W: 200, H: 120}, Contact: contact}}
			opened := 0
			s.onInteract = func(s *sim, tg input.Target) error {
				if tg.ID == 77 {
					opened++
					openDoor(s, door)
				}
				return nil
			}
			n := s.navigator()
			n.doorFn = func(class int) bool { return class == 24 }
			n.exploreFn = func(_ context.Context, _ func() bool, _ *input.Point) error { return ErrExplored }
			if err := n.ExitTo(ctxT(t), 2); err != nil {
				t.Fatalf("%s: exit: %v (door used %d times)", cls, err, opened)
			}
			if opened != 1 || s.cur().no != 2 {
				t.Fatalf("%s: door used %d times, level %d at %v", cls, opened, s.cur().no, s.pos())
			}
		})
	}
}

// V live 2026-11-24 (Catacombs Level 2, attempt 4): the door at (200,210) was used twice in a row (cellFlags 0, then 1): once it
// was open the planner still found the gate cut off by a barrier the map did not list as a door, picked the same door again and
// shut it. A door whose own cell already reads walkable is not used. Two layouts (the open door at the first or the second wall of
// two), both classes in the name only (the nav does not read the class). Must fail while an open door is a candidate.
func TestExitToDoesNotUseADoorThatIsAlreadyOpen(t *testing.T) {
	for _, lay := range []struct {
		name    string
		doorX   int
		solidX  int
		hero    input.Point
		classes []string
	}{
		{"open door beyond the solid wall", 3040, 3060, input.Point{X: 3100, Y: 3090}, []string{"Paladin", "Necromancer"}},
		{"open door before the solid wall", 3070, 3040, input.Point{X: 3100, Y: 3090}, []string{"Paladin", "Necromancer"}},
	} {
		for _, cls := range lay.classes {
			t.Run(lay.name+"/"+cls, func(t *testing.T) {
				g := openField(2800, 3000, 400, 120)
				blockRect(g, d2map.Rect{X: 2999, Y: 3000, W: 2, H: 50})
				blockRect(g, d2map.Rect{X: 2999, Y: 3065, W: 2, H: 55})
				blockRect(g, d2map.Rect{X: lay.solidX, Y: 3000, W: 2, H: 120}) // a barrier the map does not list as a door
				blockRect(g, d2map.Rect{X: lay.doorX, Y: 3000, W: 2, H: 50})
				blockRect(g, d2map.Rect{X: lay.doorX, Y: 3065, W: 2, H: 55}) // the open door's wall: its cells (rows 3050..3064) are open
				l1 := &simLevel{no: 1, rect: d2map.Rect{X: 3000, Y: 3000, W: 200, H: 120}}
				l2 := &simLevel{no: 2, rect: d2map.Rect{X: 2800, Y: 3000, W: 200, H: 120}}
				s := newSim(g, lay.hero, l1, l2)
				s.reveal = 300
				l1.lm.Presets = []d2map.Preset{{Kind: d2map.PresetObject, ID: 24, Pos: d2map.Point{X: lay.doorX, Y: 3057}}}
				s.objects = []d2.Unit{d2Unit(77, 24, lay.doorX, 3057)}
				used := 0
				s.onInteract = func(s *sim, tg input.Target) error {
					if tg.ID == 77 {
						used++
					}
					return nil
				}
				n := s.navigator()
				n.doorFn = func(class int) bool { return class == 24 }
				n.exploreFn = func(_ context.Context, _ func() bool, _ *input.Point) error { return ErrExplored }
				_ = n.ExitTo(ctxT(t), 2) // the barrier has no door: the exit fails, whatever the error is
				if used != 0 {
					t.Fatalf("the open door was used %d times (it would have been shut again)", used)
				}
			})
		}
	}
}

// The real Catacombs Level 2 (live capture 2026-11-24, internal/nav/testdata/catacombs35.grid, a crop of the level's `debug grid 5`
// with every door shut as a 3 by 3 block of wall): the hero arrives at (210,197) and the stairs to Catacombs Level 3 are at
// (149,278). The way is two doors in a row: the corridor door (200,210) into the room below, then the door (190,220) west
// out of it; the live run used the first door twice (it shut again) and gave up, "gate on the map but not reachable, cut off".
// Both classes: the geometry knows none. Must fail while the nav uses a door that already reads open.
func TestExitToCrossesTheLiveCatacombsTwo(t *testing.T) {
	data, err := os.ReadFile("testdata/catacombs35.grid")
	if err != nil {
		t.Fatal(err)
	}
	rows := strings.Split(strings.TrimSpace(string(data)), "\n")
	const ox, oy, cx, cy = 1000, 1000, 100, 180 // sim origin, crop origin in the level
	at := func(x, y int) input.Point { return input.Point{X: ox + x - cx, Y: oy + y - cy} }
	doors := [][3]int{{64, 230, 220}, {64, 190, 220}, {16, 200, 210}, {16, 255, 210}, {64, 130, 220}, {16, 160, 210}, {16, 140, 210}, {64, 110, 220}, {16, 100, 210}, {16, 100, 190}}
	for _, cls := range []string{"Paladin", "Necromancer"} {
		t.Run(cls, func(t *testing.T) {
			g := d2map.NewGrid()
			cells := make([]uint16, len(rows[0])*len(rows))
			for y, r := range rows {
				for x, c := range r {
					if c == '#' {
						cells[y*len(rows[0])+x] = 1
					}
				}
			}
			g.SetBlock(d2map.Rect{X: ox, Y: oy, W: len(rows[0]), H: len(rows)}, cells)
			gate := at(149, 278)
			l36 := &simLevel{no: 36, rect: d2map.Rect{X: gate.X - 3, Y: gate.Y - 2, W: 7, H: 14}}
			l35 := &simLevel{no: 35, rect: d2map.Rect{X: ox, Y: oy, W: len(rows[0]), H: len(rows)}}
			s := newSim(g, at(210, 197), l36, l35)
			s.reveal = 400
			s.revealRooms()
			var presets []d2map.Preset
			closed := map[int]bool{}
			for i, d := range doors {
				p := at(d[1], d[2])
				presets = append(presets, d2map.Preset{Kind: d2map.PresetObject, ID: d[0], Pos: d2map.Point{X: p.X, Y: p.Y}})
				closed[i] = true
			}
			l35.lm.Presets = presets
			l35.lm.Exits = []d2map.Exit{{Dest: 36, Pos: d2map.Point{X: gate.X, Y: gate.Y}}}
			toggles := map[int]int{}
			s.onInteract = func(s *sim, tg input.Target) error {
				for i, d := range doors {
					p := at(d[1], d[2])
					if dist(tg.Pos, p) < 8 {
						toggles[i]++
						r := d2map.Rect{X: p.X - 1, Y: p.Y - 1, W: 3, H: 3}
						if closed[i] {
							openDoor(s, r)
						} else {
							shut := make([]uint16, 9)
							for k := range shut {
								shut[k] = 1
							}
							s.truth.SetBlock(r, shut)
							for _, l := range s.levels {
								if _, ok := l.known.Flags(r.X, r.Y); ok {
									l.known.SetBlock(r, shut)
								}
							}
						}
						closed[i] = !closed[i]
						return nil
					}
				}
				return nil
			}
			n := s.navigator()
			n.doorFn = func(class int) bool { return class == 16 || class == 64 }
			n.exploreFn = func(context.Context, func() bool, *input.Point) error { return ErrExplored }
			err := n.ExitTo(ctxT(t), 36)
			if err != nil || s.cur().no != 36 {
				t.Fatalf("%s: exit %v, level %d at %v, door uses %v", cls, err, s.cur().no, s.pos(), toggles)
			}
			for i, c := range toggles {
				if c > 1 {
					t.Errorf("%s: door %v used %d times (the second use shuts it)", cls, doors[i], c)
				}
			}
		})
	}
}

// V live 2026-11-24 16:32 EST (attempt 6): in Catacombs 4 the hero stood at (82,112), 13 subtiles from the stairs up (95,111), and
// the way on to Andariel is the door (80,135) in the wall x 79..81. The walk to the door sent its first click east to (91,113),
// the mouth gate clipped and then held it ("inside a read trigger area", holds 19) and the hero stood there for ten minutes.
// testdata/lair37.grid is the live debug grid 5 of the level, crop x 30..109, y 100..149. Both classes (the geometry knows none).
func TestGoToThePastDoorInTheLairBesideTheStairs(t *testing.T) {
	data, err := os.ReadFile("testdata/lair37.grid")
	if err != nil {
		t.Fatal(err)
	}
	rows := strings.Split(strings.TrimSpace(string(data)), "\n")
	const ox, oy, cx, cy = 1000, 1000, 30, 100
	at := func(x, y int) input.Point { return input.Point{X: ox + x - cx, Y: oy + y - cy} }
	for _, cls := range []string{"Paladin", "Necromancer"} {
		t.Run(cls, func(t *testing.T) {
			g := d2map.NewGrid()
			cells := make([]uint16, len(rows[0])*len(rows))
			for y, r := range rows {
				for x, c := range r {
					if c == '#' || c == 'D' {
						cells[y*len(rows[0])+x] = 1
					}
				}
			}
			g.SetBlock(d2map.Rect{X: ox, Y: oy, W: len(rows[0]), H: len(rows)}, cells)
			l37 := &simLevel{no: 37, rect: d2map.Rect{X: ox, Y: oy, W: len(rows[0]), H: len(rows)}}
			s := newSim(g, at(82, 112), l37)
			s.reveal = 400
			s.revealRooms()
			door := at(80, 135)
			l37.lm.Presets = []d2map.Preset{{Kind: d2map.PresetObject, ID: 64, Pos: d2map.Point{X: door.X, Y: door.Y}}}
			l37.lm.Exits = []d2map.Exit{{Dest: 36, Pos: d2map.Point{X: at(95, 111).X, Y: at(95, 111).Y}}}
			opened := false
			s.onInteract = func(s *sim, tg input.Target) error {
				if !opened && dist(tg.Pos, door) < 8 {
					opened = true
					openDoor(s, d2map.Rect{X: door.X - 1, Y: door.Y - 1, W: 3, H: 3})
				}
				return nil
			}
			n := s.navigator()
			n.doorFn = func(class int) bool { return class == 64 }
			n.exploreFn = func(context.Context, func() bool, *input.Point) error { return ErrExplored }
			n.stayLevel, n.arrival = 37, true
			// the stalls of the earlier walks blacklisted the 2-wide corridor south of the hero (hardBlock, 3 min)
			_ = 0
			goal := at(45, 120)
			err := n.GoTo(ctxT(t), goal)
			if err != nil || dist(s.pos(), goal) > 6 {
				t.Fatalf("%s: GoTo %v, at %v (door opened %v)", cls, err, s.pos(), opened)
			}
		})
	}
}

// V live 2026-11-24 22:27Z (attempt 7): the fighter walks to a far boss in slices of 3 s. Every slice began with the walk to the closest
// point of the unreachable goal (the north end of the corridor, y 113) and then walked 20 tiles back south to the door (80,135); the
// door was never reached in four minutes. The walk after a cut-off door walk goes to that door first. Same lair grid; slices are
// cut after sliceSteps sim steps. Both classes (the geometry knows none). Must fail while every slice starts with the closest-point walk.
func TestSlicedGoToReachesTheDoorPastTheClosestPoint(t *testing.T) {
	data, err := os.ReadFile("testdata/lair37.grid")
	if err != nil {
		t.Fatal(err)
	}
	rows := strings.Split(strings.TrimSpace(string(data)), "\n")
	const ox, oy, cx, cy = 1000, 1000, 30, 100
	at := func(x, y int) input.Point { return input.Point{X: ox + x - cx, Y: oy + y - cy} }
	for _, cls := range []string{"Paladin", "Necromancer"} {
		t.Run(cls, func(t *testing.T) {
			g := d2map.NewGrid()
			cells := make([]uint16, len(rows[0])*len(rows))
			for y, r := range rows {
				for x, c := range r {
					if c == '#' || c == 'D' {
						cells[y*len(rows[0])+x] = 1
					}
				}
			}
			g.SetBlock(d2map.Rect{X: ox, Y: oy, W: len(rows[0]), H: len(rows)}, cells)
			l37 := &simLevel{no: 37, rect: d2map.Rect{X: ox, Y: oy, W: len(rows[0]), H: len(rows)}}
			s := newSim(g, at(82, 113), l37)
			s.reveal = 400
			s.revealRooms()
			door := at(80, 135)
			l37.lm.Presets = []d2map.Preset{{Kind: d2map.PresetObject, ID: 64, Pos: d2map.Point{X: door.X, Y: door.Y}}}
			l37.lm.Exits = []d2map.Exit{{Dest: 36, Pos: d2map.Point{X: at(95, 111).X, Y: at(95, 111).Y}}}
			opened := false
			s.onInteract = func(s *sim, tg input.Target) error {
				if !opened && dist(tg.Pos, door) < 8 {
					opened = true
					openDoor(s, d2map.Rect{X: door.X - 1, Y: door.Y - 1, W: 3, H: 3})
				}
				return nil
			}
			n := s.navigator()
			n.doorFn = func(class int) bool { return class == 64 }
			n.exploreFn = func(context.Context, func() bool, *input.Point) error { return ErrExplored }
			n.stayLevel = 37
			goal := at(45, 120)
			const sliceSteps = 30
			for slice := 0; slice < 30 && !opened; slice++ {
				ctx, cancel := context.WithCancel(ctxT(t))
				steps := 0
				s.onStep = func(*sim) {
					if steps++; steps >= sliceSteps {
						cancel()
					}
				}
				_ = n.GoTo(ctx, goal)
				cancel()
			}
			if !opened {
				t.Fatalf("%s: 30 slices and the door was never used (at %v)", cls, s.pos())
			}
		})
	}
}

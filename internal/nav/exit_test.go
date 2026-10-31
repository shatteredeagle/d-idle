package nav

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

// town (level 1) at x 3000..3200 next to a wide wilderness (level 2) at x 2800..3000. The wall between is at
// x=2999..3000; the real gap is only y 3050..3064, like the Rogue Encampment gate, although the wilderness is
// walkable along its whole edge.
func gateWorld() (*sim, *d2map.Grid) {
	g := openField(2800, 3000, 400, 120)
	blockRect(g, d2map.Rect{X: 2999, Y: 3000, W: 2, H: 50})
	blockRect(g, d2map.Rect{X: 2999, Y: 3065, W: 2, H: 55})
	l1 := &simLevel{no: 1, rect: d2map.Rect{X: 3000, Y: 3000, W: 200, H: 120}}
	l2 := &simLevel{no: 2, rect: d2map.Rect{X: 2800, Y: 3000, W: 200, H: 120}}
	return newSim(g, input.Point{X: 3100, Y: 3090}, l1, l2), g
}

func TestExitToThroughBorderGap(t *testing.T) {
	s, _ := gateWorld()
	n := s.navigator()
	if err := n.ExitTo(ctxT(t), 2); err != nil {
		t.Fatal(err)
	}
	if s.cur().no != 2 {
		t.Fatalf("still in level %d at %v", s.cur().no, s.pos())
	}
	// and back, from the wide side: must use the real gap, not the wall cells that also touch the town
	s.loadTicks = 0
	if err := n.ExitTo(ctxT(t), 1); err != nil {
		t.Fatal(err)
	}
	if s.cur().no != 1 {
		t.Fatalf("did not return: level %d at %v", s.cur().no, s.pos())
	}
}

func TestExitToAlreadyThere(t *testing.T) {
	s, _ := gateWorld()
	if err := s.navigator().ExitTo(ctxT(t), 1); err != nil {
		t.Fatal(err)
	}
	if len(s.clicks) != 0 {
		t.Fatal("clicked although already in the level")
	}
}

func TestExitToWaitsThroughLoadingScreen(t *testing.T) {
	s, _ := gateWorld()
	n := s.navigator()
	s.onStep = func(s *sim) {
		if s.cur().no == 2 && s.loadTicks == 0 && len(s.clicks) > 0 && !s.revealedRoom[[3]int{-1, -1, 0}] {
			s.revealedRoom[[3]int{-1, -1, 0}] = true
			s.loadTicks = 6
		}
	}
	if err := n.ExitTo(ctxT(t), 2); err != nil {
		t.Fatal(err)
	}
}

func TestExitToUnknownLevelExploresThenFails(t *testing.T) {
	g := openField(2800, 3000, 120, 120)
	s := newSim(g, input.Point{X: 2810, Y: 3010})
	n := s.navigator()
	tuningOf(n).Nav.ExploreTimeoutSec = 3000
	if err := n.ExitTo(ctxT(t), 9); !errors.Is(err, ErrNoExit) {
		t.Fatalf("err = %v, want ErrNoExit", err)
	}
}

func TestRouteHintsLearnedAndBad(t *testing.T) {
	n := &Navigator{}
	n.defaults()
	r, err := n.Route(1, 8) // Rogue Encampment -> Den of Evil
	if err != nil || len(r) != 2 || r[0] != 2 || r[1] != 8 {
		t.Fatalf("1->8: %v %v", r, err)
	}
	if r, err := n.Route(8, 1); err != nil || len(r) != 2 || r[0] != 2 || r[1] != 1 {
		t.Fatalf("8->1: %v %v", r, err)
	}
	if r, err := n.Route(1, 4); err != nil || len(r) != 3 {
		t.Fatalf("1->4: %v %v", r, err)
	}
	if _, err := n.Route(1, 40); !errors.Is(err, ErrNoExit) {
		t.Fatalf("act change should have no route: %v", err)
	}
	// a seen shortcut beats the hints, a bad guess is routed around
	n.learn(2, 4)
	if r, _ := n.Route(1, 4); len(r) != 2 {
		t.Fatalf("learned 2-4: %v", r)
	}
	n.markBad(2, 4)
	n.markBad(2, 3)
	if _, err := n.Route(1, 4); !errors.Is(err, ErrNoExit) {
		t.Fatalf("expected no route with 2-3 and 2-4 bad: %v", err)
	}
}

func TestWaypointSlots(t *testing.T) {
	if a, r, ok := waypointSlot(1); !ok || a != 0 || r != 0 {
		t.Fatal("rogue encampment is act 1 row 0")
	}
	if a, r, ok := waypointSlot(109); !ok || a != 4 || r != 0 {
		t.Fatal("harrogath is act 5 row 0")
	}
	if _, _, ok := waypointSlot(2); ok {
		t.Fatal("blood moor has no waypoint")
	}
}

func TestTakeWaypointClicksTabAndRow(t *testing.T) {
	g := openField(1000, 1000, 400, 100)
	l1 := &simLevel{no: 1, rect: d2map.Rect{X: 1000, Y: 1000, W: 200, H: 100}}
	l3 := &simLevel{no: 3, rect: d2map.Rect{X: 1200, Y: 1000, W: 200, H: 100}}
	s := newSim(g, input.Point{X: 1020, Y: 1050}, l1, l3)
	s.objects = nil
	s.units = nil
	wpU := struct{ x, y int }{1060, 1050}
	s.objects = append(s.objects, d2Unit(11, 119, wpU.x, wpU.y))
	s.onInteract = func(s *sim, t input.Target) error { s.ui[0x14] = 1; return nil }
	var gotX, gotY []int
	s.onClick = func(s *sim, x, y int) {
		gotX, gotY = append(gotX, x), append(gotY, y)
		if y == wpPanel.RowY0+wpPanel.RowDY { // row 1 = Cold Plains (level 3)
			s.px, s.py = 1300.5, 1050.5
			s.ui[0x14] = 0
		}
	}
	n := s.navigator()
	if err := n.TakeWaypoint(ctxT(t), 3); err != nil {
		t.Fatal(err)
	}
	if len(gotX) != 2 || gotX[0] != wpPanel.TabX[0] || gotY[0] != wpPanel.TabY {
		t.Fatalf("clicks %v %v", gotX, gotY)
	}
	// a waypoint that does not exist
	if err := n.TakeWaypoint(ctxT(t), 2); !errors.Is(err, ErrNoWaypoint) {
		t.Fatalf("err = %v", err)
	}
}

// The Act 1 fallback graph must follow the game's real layout (levels.txt Vis columns, VERIFIED against the
// data of the installed 1.14d): Dark Wood is reached from Stony Field only through the Underground Passage,
// the Forgotten Tower hangs off the Black Marsh, the Pit off the Tamoe Highland.
func TestRouteHintsFollowRealLayout(t *testing.T) {
	n := &Navigator{}
	n.defaults()
	for _, c := range []struct {
		from, to int
		want     []int
	}{
		{4, 5, []int{10, 5}},
		{1, 20, []int{2, 3, 4, 10, 5, 6, 20}},
		{7, 12, []int{12}},
		{12, 16, []int{16}},
		{10, 14, []int{14}},
		{3, 18, []int{17, 18}},
		{3, 19, []int{17, 19}},
		{26, 37, []int{27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37}},
	} {
		r, err := n.Route(c.from, c.to)
		if err != nil || fmt.Sprint(r) != fmt.Sprint(c.want) {
			t.Errorf("Route(%d,%d) = %v, %v; want %v", c.from, c.to, r, err, c.want)
		}
	}
}

// The waypoint unit is not in the unit tables until the character is near: the walk toward its preset must stop
// inside interact range, not on the object (its footprint blocks; the stuck ladder then clicked the waypoint and
// the next click hit the open panel: VERIFIED LIVE in Cold Plains).
func TestOpenWaypointStopsNextToPreset(t *testing.T) {
	g := openField(1000, 1000, 400, 100)
	l1 := &simLevel{no: 3, rect: d2map.Rect{X: 1000, Y: 1000, W: 400, H: 100}}
	s := newSim(g, input.Point{X: 1020, Y: 1050}, l1)
	l1.lm.Presets = []d2map.Preset{{Kind: d2map.PresetObject, ID: 119, Pos: d2map.Point{X: 1200, Y: 1050}}}
	s.onStep = func(s *sim) {
		if len(s.objects) == 0 && dist(s.pos(), input.Point{X: 1200, Y: 1050}) < 30 {
			s.objects = append(s.objects, d2Unit(11, 119, 1200, 1050))
		}
	}
	var at input.Point
	s.onInteract = func(s *sim, tg input.Target) error { at = s.pos(); s.ui[0x14] = 1; return nil }
	n := s.navigator()
	if err := n.ActivateWaypoint(ctxT(t)); err != nil {
		t.Fatal(err)
	}
	if d := dist(at, input.Point{X: 1200, Y: 1050}); d < 3 || d > 6 {
		t.Fatalf("interacted from %v subtiles away", d)
	}
}

// A waypoint that is not hovered (something stands on it, bad angle) is approached again from another side.
func TestOpenWaypointRetriesFromAnotherSide(t *testing.T) {
	g := openField(1000, 1000, 400, 100)
	l1 := &simLevel{no: 3, rect: d2map.Rect{X: 1000, Y: 1000, W: 400, H: 100}}
	s := newSim(g, input.Point{X: 1020, Y: 1050}, l1)
	s.objects = append(s.objects, d2Unit(11, 119, 1100, 1050))
	var spots []input.Point
	s.onInteract = func(s *sim, tg input.Target) error {
		spots = append(spots, s.pos())
		if len(spots) == 1 {
			return input.ErrNotHovered
		}
		s.ui[0x14] = 1
		return nil
	}
	n := s.navigator()
	if err := n.ActivateWaypoint(ctxT(t)); err != nil {
		t.Fatal(err)
	}
	if len(spots) != 2 || dist(spots[0], spots[1]) < 1 {
		t.Fatalf("interact spots %v, want two different ones", spots)
	}
}

// A walk toward the gate that ends in a third level must not be reported as "no exit from that third level":
// ToArea would mark start->target bad for good (VERIFIED LIVE: Monastery Gate, a border push fell back into the
// Tamoe Highland and every later route to the cloisters failed).
func TestExitToEndingInThirdLevelIsNotNoExit(t *testing.T) {
	g := openField(1000, 900, 200, 200)
	la := &simLevel{no: 26, rect: d2map.Rect{X: 1000, Y: 1000, W: 100, H: 100}}
	lb := &simLevel{no: 27, rect: d2map.Rect{X: 1000, Y: 900, W: 100, H: 100}}
	lc := &simLevel{no: 7, rect: d2map.Rect{X: 1100, Y: 1000, W: 100, H: 100}}
	s := newSim(g, input.Point{X: 1050, Y: 1040}, la, lb, lc)
	pushed := false
	s.onStep = func(s *sim) {
		if !pushed && s.cur() == la && s.py != 1040.5 {
			pushed = true
			s.px, s.py = 1120.5, 1040.5
		}
	}
	n := s.navigator()
	err := n.ExitTo(ctxT(t), 27)
	if !errors.Is(err, ErrLevelChanged) || errors.Is(err, ErrNoExit) {
		t.Fatalf("err = %v, want ErrLevelChanged", err)
	}
	if n.isBad(26, 27) {
		t.Fatal("edge marked bad")
	}
}

// The waypoint unit is missing from the unit table and from the presets until the character is near (Lut Gholein,
// V live): TakeWaypoint walks to the known position first, then finds it.
func TestTakeWaypointWalksToKnownPositionWhenNotStreamed(t *testing.T) {
	old := knownWaypointPos
	defer func() { knownWaypointPos = old }()
	knownWaypointPos = map[int]input.Point{1: {X: 1150, Y: 1050}}
	g := openField(1000, 1000, 400, 100)
	l1 := &simLevel{no: 1, rect: d2map.Rect{X: 1000, Y: 1000, W: 200, H: 100}}
	l3 := &simLevel{no: 3, rect: d2map.Rect{X: 1200, Y: 1000, W: 200, H: 100}}
	s := newSim(g, input.Point{X: 1020, Y: 1050}, l1, l3)
	s.objects, s.units = nil, nil
	added := false
	s.onStep = func(s *sim) {
		if !added && dist(s.pos(), input.Point{X: 1150, Y: 1050}) < 60 {
			s.objects = append(s.objects, d2Unit(11, 119, 1150, 1050))
			added = true
		}
	}
	s.onInteract = func(s *sim, t input.Target) error { s.ui[0x14] = 1; return nil }
	s.onClick = func(s *sim, x, y int) {
		if y == wpPanel.RowY0+wpPanel.RowDY {
			s.px, s.py = 1300.5, 1050.5
			s.ui[0x14] = 0
		}
	}
	if err := s.navigator().TakeWaypoint(ctxT(t), 3); err != nil {
		t.Fatal(err)
	}
	if !added {
		t.Fatal("never walked toward the known position")
	}
}

func TestAct2RouteReachesCanyonByHint(t *testing.T) {
	n := &Navigator{}
	n.defaults()
	got, err := n.Route(43, 46)
	if err != nil || len(got) == 0 || got[len(got)-1] != 46 {
		t.Fatalf("route %v err %v", got, err)
	}
}

// Exploring that reports success without moving or revealing anything must end in ErrNoExit after a bounded number
// of rounds, and a repeat call must fail at once instead of spinning again (LIVE: 6800 "exploring" lines).
func TestExitToFruitlessExploreGivesUp(t *testing.T) {
	g := openField(2800, 3000, 120, 120)
	s := newSim(g, input.Point{X: 2810, Y: 3010})
	n := s.navigator()
	calls := 0
	n.exploreFn = func(context.Context, func() bool, *input.Point) error { calls++; return nil }
	if err := n.ExitTo(ctxT(t), 9); !errors.Is(err, ErrNoExit) {
		t.Fatalf("err = %v, want ErrNoExit", err)
	}
	if calls > 3 {
		t.Fatalf("explored %d times", calls)
	}
	before := calls
	if err := n.ExitTo(ctxT(t), 9); !errors.Is(err, ErrNoExit) || calls != before {
		t.Fatalf("repeat: err = %v, explore calls %d -> %d", err, before, calls)
	}
}

// V live (act2live5): Act 2 rows 7 and 8 ignore clicks on the text and only react to the icon column; the other
// rows react to the text. Either way TakeWaypoint must end up in the level, trying the other column as fallback.
func TestTakeWaypointIconColumn(t *testing.T) {
	for _, c := range []struct {
		name         string
		level        int
		travelAtIcon bool
		wantLast     int // x of the click that travelled
	}{
		{"canyon icon", 46, true, wpPanel.IconX},
		{"palace text", 52, false, wpPanel.RowX},
		{"row 1 icon only (fallback)", 42, true, wpPanel.IconX},
	} {
		t.Run(c.name, func(t *testing.T) {
			g := openField(1000, 1000, 400, 100)
			l40 := &simLevel{no: 40, rect: d2map.Rect{X: 1000, Y: 1000, W: 200, H: 100}}
			lt := &simLevel{no: c.level, rect: d2map.Rect{X: 1200, Y: 1000, W: 200, H: 100}}
			s := newSim(g, input.Point{X: 1020, Y: 1050}, l40, lt)
			s.objects, s.units = nil, nil
			s.objects = append(s.objects, d2Unit(11, 156, 1060, 1050))
			s.onInteract = func(s *sim, t input.Target) error { s.ui[0x14] = 1; return nil }
			var xs []int
			s.onClick = func(s *sim, x, y int) {
				xs = append(xs, x)
				if (x == wpPanel.IconX) == c.travelAtIcon && y > wpPanel.RowY0-10 {
					s.px, s.py = 1300.5, 1050.5
					s.ui[0x14] = 0
				}
			}
			if err := s.navigator().TakeWaypoint(ctxT(t), c.level); err != nil {
				t.Fatal(err)
			}
			if got := xs[len(xs)-1]; got != c.wantLast {
				t.Fatalf("last click x=%d want %d (all %v)", got, c.wantLast, xs)
			}
		})
	}
}

// Interact keeps failing the hover check: plain world clicks on the object open the panel.
func TestOpenWaypointPlainClickFallback(t *testing.T) {
	g := openField(1000, 1000, 400, 100)
	l1 := &simLevel{no: 1, rect: d2map.Rect{X: 1000, Y: 1000, W: 200, H: 100}}
	l3 := &simLevel{no: 3, rect: d2map.Rect{X: 1200, Y: 1000, W: 200, H: 100}}
	s := newSim(g, input.Point{X: 1020, Y: 1050}, l1, l3)
	s.objects, s.units = nil, nil
	s.objects = append(s.objects, d2Unit(11, 119, 1060, 1050))
	s.onInteract = func(s *sim, t input.Target) error { return input.ErrNotHovered }
	s.onClickWorld = func(s *sim, p input.Point) { s.ui[0x14] = 1 }
	s.onClick = func(s *sim, x, y int) {
		if y == wpPanel.RowY0+wpPanel.RowDY {
			s.px, s.py = 1300.5, 1050.5
			s.ui[0x14] = 0
		}
	}
	if err := s.navigator().TakeWaypoint(ctxT(t), 3); err != nil {
		t.Fatal(err)
	}
}

// ActivateWaypointNear touches a known waypoint that is close and leaves a far one (and an unknown one) alone,
// without walking or exploring.
func TestActivateWaypointNear(t *testing.T) {
	for _, c := range []struct {
		name    string
		wpX     int
		preset  bool
		touched bool
	}{
		{"near preset", 1060, true, true},
		{"far preset", 1380, true, false},
		{"none known", 0, false, false},
	} {
		g := openField(1000, 1000, 400, 100)
		l1 := &simLevel{no: 4, rect: d2map.Rect{X: 1000, Y: 1000, W: 400, H: 100}}
		s := newSim(g, input.Point{X: 1020, Y: 1050}, l1)
		if c.preset {
			l1.lm.Presets = []d2map.Preset{{Kind: d2map.PresetObject, ID: 119, Pos: d2map.Point{X: c.wpX, Y: 1050}}}
			s.objects = append(s.objects, d2Unit(11, 119, c.wpX, 1050))
		}
		interacted := false
		s.onInteract = func(s *sim, tg input.Target) error { interacted = true; s.ui[0x14] = 1; return nil }
		touched, err := s.navigator().ActivateWaypointNear(ctxT(t), 60)
		if err != nil || touched != c.touched || interacted != c.touched {
			t.Errorf("%s: touched=%v interacted=%v err=%v, want touched=%v", c.name, touched, interacted, err, c.touched)
		}
	}
}

func TestExitToGateClickFallback(t *testing.T) {
	g := openField(1000, 1000, 400, 60)
	l1 := &simLevel{no: 1, rect: d2map.Rect{X: 1000, Y: 1000, W: 200, H: 60}}
	l2 := &simLevel{no: 2, rect: d2map.Rect{X: 1200, Y: 1000, W: 200, H: 60}}
	s := newSim(g, input.Point{X: 1010, Y: 1030}, l1, l2)
	exitOnLine(s, 2) // a tile exit that never fires by walking
	s.onClick = func(s *sim, x, y int) {
		if x == 340 && y == 175 {
			s.px, s.py = 1300.5, 1030.5
		}
	}
	if err := s.navigator().ExitTo(ctxT(t), 2); err != nil {
		t.Fatal(err)
	}
	if s.cur().no != 2 || len(s.rawClicks) < 2 {
		t.Fatalf("level %d, raw clicks %v", s.cur().no, s.rawClicks)
	}
}

func TestTakeWaypointWaitsThroughLongLoad(t *testing.T) {
	g := openField(1000, 1000, 400, 100)
	l1 := &simLevel{no: 40, rect: d2map.Rect{X: 1000, Y: 1000, W: 200, H: 100}}
	l2 := &simLevel{no: 46, rect: d2map.Rect{X: 1200, Y: 1000, W: 200, H: 100}}
	s := newSim(g, input.Point{X: 1020, Y: 1050}, l1, l2)
	s.objects = append(s.objects, d2Unit(11, 119, 1060, 1050))
	s.onInteract = func(s *sim, t input.Target) error { s.ui[0x14] = 1; return nil }
	s.onClick = func(s *sim, x, y int) {
		if x == wpPanel.IconX && y == wpPanel.RowY0+8*wpPanel.RowDY+wpPanel.IconDY {
			s.px, s.py = 1300.5, 1050.5
			s.loadTicks = 40 // longer than the 3 s first wait at the poll rate
			s.ui[0x14] = 0
		}
	}
	if err := s.navigator().TakeWaypoint(ctxT(t), 46); err != nil {
		t.Fatal(err)
	}
}

// KITGATE: levels named by SetAvoidLevels cost a detour, never a wall; a destination inside the set stays reachable.
func TestRouteAvoidLevels(t *testing.T) {
	n := &Navigator{}
	n.defaults()
	n.learn(3, 9) // seen crossings: Cold Plains - Cave 1 - Stony Field is two cheap edges, the hint 3-4 costs 3
	n.learn(9, 4)
	if r, _ := n.Route(3, 4); len(r) != 2 || r[0] != 9 {
		t.Fatalf("without the set the seen cave shortcut wins: %v", r)
	}
	n.SetAvoidLevels(map[int]bool{9: true, 17: true})
	if r, _ := n.Route(3, 4); len(r) != 1 || r[0] != 4 {
		t.Fatalf("avoided cave: want the direct exit, got %v", r)
	}
	if r, err := n.Route(3, 9); err != nil || len(r) != 1 || r[0] != 9 {
		t.Fatalf("an avoided destination is still routable: %v %v", r, err)
	}
	n.SetAvoidLevels(nil)
	if r, _ := n.Route(3, 4); len(r) != 2 {
		t.Fatalf("cleared set restores the old route: %v", r)
	}
}

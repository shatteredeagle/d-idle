package nav

import (
	"context"
	"errors"
	"testing"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

// A ghost obstacle (open on the map, solid in the game) defeats a short ladder; the second walk to the same
// place must remember the spot, block the cells ahead and get round it.
func TestRepeatedStallAtSameSpotRecovers(t *testing.T) {
	g := openField(1000, 1000, 200, 60)
	s := newSim(g, input.Point{X: 1010, Y: 1030})
	n := s.navigator()
	truth := g.Clone()
	blockRect(truth, d2map.Rect{X: 1080, Y: 1025, W: 4, H: 10})
	s.truth, s.knownSrc = truth, g
	tuningOf(n).Nav.MaxRecoveries = 2
	goal := input.Point{X: 1150, Y: 1030}
	if err := n.GoTo(ctxT(t), goal); !errors.Is(err, ErrStuck) {
		t.Fatalf("first walk: %v, want ErrStuck", err)
	}
	if err := n.GoTo(ctxT(t), goal); err != nil {
		t.Fatalf("second walk: %v", err)
	}
	if dist(s.pos(), goal) > 2.5 {
		t.Fatalf("at %v", s.pos())
	}
	if len(n.spots) != 1 || n.spots[0].hits < 2 || len(n.hard) == 0 || n.spots[0].method == "" {
		t.Fatalf("memory not used: %+v hard=%d", n.spots, len(n.hard))
	}
}

func TestStuckMemoryExpiresAndIgnoresOwnWalk(t *testing.T) {
	s := newSim(openField(1000, 1000, 50, 50), input.Point{X: 1010, Y: 1010})
	n := s.navigator()
	n.walkSeq = 1
	p := input.Point{X: 1020, Y: 1020}
	if _, rep := n.noteStuck(5, p); rep {
		t.Fatal("first stall is no repeat")
	}
	if _, rep := n.noteStuck(5, input.Point{X: 1024, Y: 1020}); rep {
		t.Fatal("the same walk's retry is no repeat")
	}
	n.walkSeq = 2
	if sp, rep := n.noteStuck(5, p); !rep || sp.hits != 2 {
		t.Fatalf("repeat=%v spot=%+v", rep, sp)
	}
	if _, rep := n.noteStuck(6, p); rep {
		t.Fatal("another level is another spot")
	}
	s.now = s.now.Add(stuckMemory + 1)
	n.walkSeq = 3
	if _, rep := n.noteStuck(5, p); rep {
		t.Fatal("stale memory used")
	}
}

func TestHardBlockPlansAroundAndFallsBack(t *testing.T) {
	g := openField(1000, 1000, 100, 60)
	s := newSim(g, input.Point{X: 1010, Y: 1030})
	n := s.navigator()
	o, err := n.observeWait(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	goal := input.Point{X: 1090, Y: 1030}
	n.hardBlock(input.Point{X: 1050, Y: 1030}, 2.5)
	res, err := n.plan(o.lm, o.pos, goal, 2, nil, 0)
	if err != nil || res.partial {
		t.Fatalf("plan: %v partial=%v", err, res.partial)
	}
	for _, w := range res.wps {
		if dist(w, input.Point{X: 1050, Y: 1030}) < 2.5 {
			t.Fatalf("waypoint %v inside the blocked cells", w)
		}
	}
	// a corridor whose only way is the blocked cell: the block yields rather than leaving no path
	n2 := newSim(asciiTruth(1000, 1000, "###################", "...................", "###################"), input.Point{X: 1001, Y: 1001})
	nn := n2.navigator()
	o2, err := nn.observeWait(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	nn.hardBlock(input.Point{X: 1009, Y: 1001}, 2.5)
	if _, err := nn.plan(o2.lm, o2.pos, input.Point{X: 1017, Y: 1001}, 2, nil, 0); err != nil {
		t.Fatalf("fallback plan: %v", err)
	}
}

// The middle of the gap is a ghost obstacle: the walk to it stalls, another approach cell of the same gap works.
func TestExitToStuckGateUsesOtherApproachCell(t *testing.T) {
	s, g := gateWorld()
	truth := g.Clone()
	blockRect(truth, d2map.Rect{X: 2996, Y: 3054, W: 5, H: 7})
	s.truth, s.knownSrc = truth, g
	s.px, s.py = 2900.5, 3057.5
	n := s.navigator()
	if err := n.ExitTo(ctxT(t), 1); err != nil {
		t.Fatal(err)
	}
	if s.cur().no != 1 {
		t.Fatalf("level %d at %v", s.cur().no, s.pos())
	}
}

func TestExitToStuckEscapesByHook(t *testing.T) {
	s, g := gateWorld()
	truth := g.Clone()
	blockRect(truth, d2map.Rect{X: 2995, Y: 3040, W: 12, H: 40}) // the whole gap is solid
	s.truth, s.knownSrc = truth, g
	s.px, s.py = 2900.5, 3057.5
	n := s.navigator()
	escaped := 0
	n.Escape = func(ctx context.Context) error {
		escaped++
		s.px, s.py = 3100.5, 3090.5
		return nil
	}
	err := n.ExitTo(ctxT(t), 1)
	if !errors.Is(err, ErrLevelChanged) || escaped != 1 {
		t.Fatalf("err=%v escaped=%d, want ErrLevelChanged after one escape", err, escaped)
	}
	n.Escape = nil
	s.px, s.py = 2900.5, 3057.5
	s.now = s.now.Add(stalledGateTTL + 1) // the stalled gate is tried again later
	if err := n.ExitTo(ctxT(t), 1); !errors.Is(err, ErrStuck) {
		t.Fatalf("without a hook: %v, want ErrStuck", err)
	}
}

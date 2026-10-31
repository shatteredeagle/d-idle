package nav

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

func exitOnLine(s *sim, dest int) {
	s.levels[0].lm.Exits = []d2map.Exit{{Dest: dest, Pos: d2map.Point{X: 1080, Y: 1030}}}
}

func TestGoToAvoidsExitTile(t *testing.T) {
	g := openField(1000, 1000, 200, 60)
	s := newSim(g, input.Point{X: 1010, Y: 1030})
	exitOnLine(s, 62)
	n := s.navigator()
	minD := 1e9
	s.onStep = func(s *sim) {
		if d := dist(s.pos(), input.Point{X: 1080, Y: 1030}); d < minD {
			minD = d
		}
	}
	if err := n.GoTo(ctxT(t), input.Point{X: 1150, Y: 1030}); err != nil {
		t.Fatal(err)
	}
	if minD < 3 {
		t.Fatalf("walked within %.1f of an exit tile that was not the target", minD)
	}
}

func TestPlanThroughExitWhenOnlyWay(t *testing.T) {
	g := openField(1000, 1000, 200, 60)
	blockRect(g, d2map.Rect{X: 1070, Y: 1000, W: 20, H: 26})
	blockRect(g, d2map.Rect{X: 1070, Y: 1034, W: 20, H: 26})
	s := newSim(g, input.Point{X: 1010, Y: 1030})
	exitOnLine(s, 62)
	n := s.navigator()
	if err := n.GoTo(ctxT(t), input.Point{X: 1150, Y: 1030}); err != nil {
		t.Fatalf("exit penalty must not wall off the only path: %v", err)
	}
}

func TestExitToWantedDestinationNotPenalized(t *testing.T) {
	g := openField(1000, 1000, 200, 60)
	s := newSim(g, input.Point{X: 1010, Y: 1030})
	exitOnLine(s, 62)
	n := s.navigator()
	n.wantExit = 62
	lm := s.levels[0].lm
	s.revealRooms()
	f := n.fieldFor(lm)
	f.ClearDyn()
	n.penalizeExits(f, lm, input.Point{X: 1150, Y: 1030})
	if len(f.touched) != 0 {
		t.Fatalf("exit to the wanted level was penalized (%d cells)", len(f.touched))
	}
	n.wantExit = 0
	n.penalizeExits(f, lm, input.Point{X: 1150, Y: 1030})
	if len(f.touched) == 0 {
		t.Fatal("other exit not penalized")
	}
}

func packCorridor() *d2map.Grid {
	g := openField(1000, 1000, 200, 60)
	blockRect(g, d2map.Rect{X: 1060, Y: 1000, W: 40, H: 26})
	blockRect(g, d2map.Rect{X: 1060, Y: 1034, W: 40, H: 26})
	return g
}

func packUnits() []d2.Unit {
	return []d2.Unit{
		{ID: 5, Type: d2.TypeMonster, X: 1080, Y: 1028, Mode: d2.MonModeStand},
		{ID: 6, Type: d2.TypeMonster, X: 1080, Y: 1031, Mode: d2.MonModeStand},
	}
}

func TestPackStallYieldsToCombat(t *testing.T) {
	s := newSim(packCorridor(), input.Point{X: 1010, Y: 1030})
	s.units, s.solidUnits = packUnits(), true
	n := s.navigator()
	yields := 0
	n.Yield = func(ctx context.Context, near []d2.Unit) error {
		yields++
		if len(near) == 0 {
			t.Error("yield without monsters")
		}
		s.units = nil // combat killed them
		return nil
	}
	if err := n.GoTo(ctxT(t), input.Point{X: 1150, Y: 1030}); err != nil {
		t.Fatal(err)
	}
	if yields != 1 {
		t.Errorf("yields %d", yields)
	}
}

func TestPackStallWithoutYieldRetreatsThenFails(t *testing.T) {
	s := newSim(packCorridor(), input.Point{X: 1010, Y: 1030})
	s.units, s.solidUnits = packUnits(), true
	n := s.navigator()
	if err := n.GoTo(ctxT(t), input.Point{X: 1150, Y: 1030}); err == nil {
		t.Fatal("walked through a solid pack")
	}
}

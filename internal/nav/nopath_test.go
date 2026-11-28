package nav

import (
	"errors"
	"testing"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

// The Arcane Sanctuary stall (lab night 2026-11-28, drill.act2.summoner.stepexec): a goal walled off on a fully known grid was
// searched again on every approach. The second and third asks on the same field are answered from the memo with the same
// error; a field rebuilt (a door opened, cells learned) searches again.
func TestUnreachableGoalIsSearchedOnce(t *testing.T) {
	g := asciiTruth(0, 0,
		"......#......",
		"......#......",
		"......#......",
		"......#......")
	from, to := input.Point{X: 1, Y: 1}, input.Point{X: 11, Y: 2}
	s := newSim(g, from)
	n := s.navigator()
	lm := &d2map.LevelMap{No: 74, Rect: g.Bounds(), Grid: g}
	_, err1 := n.plan(lm, from, to, n.tune().ArriveTolerance, nil, 0)
	if !errors.Is(err1, ErrNoPath) {
		res, _ := n.plan(lm, from, to, n.tune().ArriveTolerance, nil, 0)
		if !res.partial {
			t.Fatalf("the walled-off goal planned: %v %+v", err1, res)
		}
	}
	first := n.searches
	for i := 0; i < 2; i++ {
		res, err := n.plan(lm, from, to, n.tune().ArriveTolerance, nil, 0)
		if (err == nil) != (err1 == nil) || err == nil && !res.partial {
			t.Fatalf("ask %d: %v %+v, want the first answer %v", i+2, err, res, err1)
		}
	}
	if n.searches != first {
		t.Fatalf("searches %d after the first ask, %d after three: the unreachable goal was searched again", first, n.searches)
	}
	n.invalidate()
	_, _ = n.plan(lm, from, to, n.tune().ArriveTolerance, nil, 0)
	if n.searches == first {
		t.Fatal("a rebuilt field must search again")
	}
}

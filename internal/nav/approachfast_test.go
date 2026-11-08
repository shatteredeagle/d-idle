package nav

import (
	"errors"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

// countReads counts the full snapshots the navigator asks for (APPROACHFAST: every one costs about ten d2d frames,
// the position-only world update one).
type countReads struct {
	s     *sim
	snaps int
}

func (c *countReads) Snapshot() (*d2.Snapshot, error) { c.snaps++; return c.s.Snapshot() }

// approachSteps: the short one-waypoint steps of a combat approach. Two classes: a melee Paladin (walk speed, short
// steps to the target) and a ranged Sorceress (run speed, longer steps to a casting position). The walk is
// class-free; only speed and step length differ.
var approachSteps = []struct {
	name  string
	speed float64
	step  int
}{
	{"paladin-melee", 9.4, 6},
	{"sorceress-ranged", 14, 12},
}

func approachReads(t *testing.T, speed float64, step int, off bool) (snaps, updates int, took float64) {
	t.Helper()
	s := newSim(openField(1000, 1000, 400, 60), input.Point{X: 1005, Y: 1030})
	s.speed = speed
	n := s.navigator()
	cr := &countReads{s: s}
	n.st = cr
	n.NoFastPoll = off
	goal := input.Point{X: 1005 + step, Y: 1030}
	s.exact = &goal
	t0 := s.now
	if err := n.GoTo(ctxT(t), goal); err != nil {
		t.Fatalf("GoTo: %v", err)
	}
	if d := dist(s.pos(), goal); d > 3 {
		t.Fatalf("stopped %v from the goal", d)
	}
	return cr.snaps, s.updates, s.now.Sub(t0).Seconds()
}

func TestApproachStepUsesFewerFullSnapshots(t *testing.T) {
	for _, tc := range approachSteps {
		t.Run(tc.name, func(t *testing.T) {
			before, _, _ := approachReads(t, tc.speed, tc.step, true)
			after, upd, _ := approachReads(t, tc.speed, tc.step, false)
			t.Logf("full snapshots per approach step: before %d, after %d (world updates %d)", before, after, upd)
			if before < 2 {
				t.Fatalf("baseline took %d snapshots; the step is too short to compare", before)
			}
			if after >= before {
				t.Fatalf("full snapshots %d, want fewer than the %d of the old walk", after, before)
			}
		})
	}
}

// A position-only walk that does not progress still ends in ErrStuck: the detector works on the polled position.
func TestFastPollStuckDetectorStillFires(t *testing.T) {
	for _, tc := range approachSteps {
		t.Run(tc.name, func(t *testing.T) {
			g := openField(1000, 1000, 200, 60)
			s := newSim(g, input.Point{X: 1010, Y: 1030})
			s.speed = tc.speed
			truth := g.Clone()
			blockRect(truth, d2map.Rect{X: 1013, Y: 1020, W: 5, H: 20}) // a wall the map does not show
			s.truth, s.knownSrc = truth, g
			n := s.navigator()
			cr := &countReads{s: s}
			n.st = cr
			err := n.GoTo(ctxT(t), input.Point{X: 1040, Y: 1030})
			if !errors.Is(err, ErrStuck) {
				t.Fatalf("err = %v, want ErrStuck", err)
			}
		})
	}
}

// Level change, a jump of the position and the stale-monster refresh all fall back to the full read.
func TestFastPollFallsBackToFullRead(t *testing.T) {
	s := newSim(openField(1000, 1000, 400, 60), input.Point{X: 1005, Y: 1030})
	n := s.navigator()
	cr := &countReads{s: s}
	n.st = cr
	goal := input.Point{X: 1105, Y: 1030}
	s.exact = &goal
	if err := n.GoTo(ctxT(t), goal); err != nil {
		t.Fatal(err)
	}
	// 100 subtiles at speed 6 take about 17 s: at least one full read per fullPollEvery.
	if min := int(s.travelled / s.speed * 1000 / float64(fullPollEvery.Milliseconds())); cr.snaps < min {
		t.Fatalf("%d full reads over the walk, want at least %d (one per %v)", cr.snaps, min, fullPollEvery)
	}
}

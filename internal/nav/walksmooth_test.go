package nav

import (
	"fmt"
	"testing"
	"time"

	"d2lod-bot/internal/input"
)

// pauseStats walks the sim to goal and returns the pauses longer than 500 ms per 100 subtiles walked, the longest
// pause and the clicks per 100 subtiles (WALKSMOOTH: a Paladin's stop-and-go walk out of the start area).
func pauseStats(t *testing.T, s *sim, goal input.Point) (perHundred, longest, clicksPer100 float64) {
	t.Helper()
	n := s.navigator()
	s.exact = &goal
	var lastX, lastY float64 = s.px, s.py
	idle, pauses := time.Duration(0), 0
	var longestD time.Duration
	started := false
	s.onStep = func(s *sim) {
		moved := (s.px-lastX)*(s.px-lastX)+(s.py-lastY)*(s.py-lastY) > 1e-6
		lastX, lastY = s.px, s.py
		if moved {
			if started && idle > 500*time.Millisecond {
				pauses++
			}
			started = true
			idle = 0
			return
		}
		if started {
			idle += 100 * time.Millisecond
			longestD = max(longestD, idle)
		}
	}
	if err := n.GoTo(ctxT(t), goal); err != nil {
		t.Fatalf("GoTo: %v", err)
	}
	d := s.travelled
	return float64(pauses) * 100 / d, longestD.Seconds(), float64(len(s.clicks)) * 100 / d
}

// The character walks and runs (two speeds: a walking class and a running one share the nav code), d2d ends moves
// short of the target and a loaded host delays the command. None of this may make the walk stop and go.
func TestWalkIsSmoothWhenMovesEndShortOrLag(t *testing.T) {
	for _, tc := range []struct {
		name      string
		speed     float64
		stopShort float64
		delay     time.Duration
	}{
		{"walk-stopshort", 9.4, 5.5, 0},
		{"run-stopshort", 14, 5.5, 0},
		{"walk-lag", 9.4, 0, 800 * time.Millisecond},
		{"run-lag", 14, 0, 800 * time.Millisecond},
		{"run-stopshort-lag", 14, 5.5, 600 * time.Millisecond},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newSim(openField(1000, 1000, 400, 60), input.Point{X: 1005, Y: 1030})
			s.speed, s.stopShort, s.cmdDelay = tc.speed, tc.stopShort, tc.delay
			per100, longest, clicks := pauseStats(t, s, input.Point{X: 1300, Y: 1030})
			t.Logf("pauses>500ms per 100 subtiles %.2f, longest pause %.1fs, clicks per 100 %.1f", per100, longest, clicks)
			if per100 > 0.35 || longest > 1.0 {
				t.Fatalf("%s", fmt.Sprintf("stop-and-go walk: %.2f pauses over 500 ms per 100 subtiles, longest %.1f s", per100, longest))
			}
		})
	}
}

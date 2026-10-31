package nav

// Scenario corpus, nav half (docs/scenarios.md): a recorded real failure situation replayed through the public
// Navigator API as an end-state test, with a knob-off subtest that EXPECTS the old failure.

import (
	"errors"
	"testing"
	"time"
)

// Situation: the character stands in a cut-off cell of the known town grid (the level-1 exit is unreachable on the
// known grid; townPocket builds that shape on the sim) and the loop keeps asking for level 2, once a virtual minute,
// as the real loop did. Evidence (docs/probes/navsoak-compare.md, VERIFIED from the logs of two soaks): "roam leg
// failed ... {138 66} is unreachable from {145 68}" (identical coordinates in both runs), then "nav no exit known yet,
// exploring", the town to Blood Moor edge condemned for good, 0 "Entered" lines afterwards, 950 save-and-exit
// re-entries that did not clear it, 60 and 167 wall minutes lost. The pocket itself is INFERRED (a failed portal).
// End state: the character leaves the pocket into level 2, no edge is marked bad, the route onward still exists.
func TestScenario_WedgePocket(t *testing.T) {
	t.Run("fixed: leaves the pocket, no edge condemned", func(t *testing.T) {
		s, n := townPocket(false)
		var err error
		for try := 0; try < 5 && s.cur().no != 2; try++ {
			err = n.ToArea(ctxT(t), 2)
			s.Sleep(time.Minute)
		}
		if s.cur().no != 2 {
			t.Fatalf("still in level %d after 5 tries: %v", s.cur().no, err)
		}
		if n.isBad(1, 2) {
			t.Fatal("edge 1->2 marked bad")
		}
		if r, rerr := n.Route(1, 3); rerr != nil || len(r) != 2 {
			t.Fatalf("route from the town to level 3: %v %v", r, rerr)
		}
	})

	// A pocket the character really cannot leave (physical walls): a plain failure, never the wedge. The route stays
	// open across a virtual day of retries, so the character is not locked out of the whole map.
	t.Run("fixed: a real prison never condemns the route", func(t *testing.T) {
		s, n := townPocket(true)
		for try := 0; try < 20; try++ {
			if err := n.ToArea(ctxT(t), 2); err == nil || errors.Is(err, ErrNoExit) {
				t.Fatalf("try %d: err = %v, want a non-ErrNoExit failure", try, err)
			}
			s.Sleep(time.Hour)
			if _, rerr := n.Route(1, 3); rerr != nil {
				t.Fatalf("try %d: route lost: %v", try, rerr)
			}
		}
	})

	// Knob off (nav.badEdgeTTLSec 0): the old wedge reproduces and never heals.
	t.Run("knob off: old wedge reproduces", func(t *testing.T) {
		s, n := townPocket(true)
		tuningOf(n).Nav.BadEdgeTTLSec = 0
		if err := n.ToArea(ctxT(t), 2); !errors.Is(err, ErrNoExit) {
			t.Fatalf("err = %v, want ErrNoExit", err)
		}
		for try := 0; try < 24; try++ {
			s.Sleep(time.Hour)
			if _, rerr := n.Route(1, 3); rerr == nil {
				t.Fatalf("route came back after %d hours", try+1)
			}
		}
		if !n.isBad(1, 2) {
			t.Fatal("edge 1->2 not condemned")
		}
	})
}

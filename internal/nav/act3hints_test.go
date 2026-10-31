package nav

import "testing"

// reach returns the levels reachable from start over the given hint tables (undirected).
func reach(start int, tabs ...[][2]int) map[int]bool {
	adj := map[int][]int{}
	for _, t := range tabs {
		for _, h := range t {
			adj[h[0]] = append(adj[h[0]], h[1])
			adj[h[1]] = append(adj[h[1]], h[0])
		}
	}
	seen := map[int]bool{start: true}
	q := []int{start}
	for len(q) > 0 {
		n := q[0]
		q = q[1:]
		for _, m := range adj[n] {
			if !seen[m] {
				seen[m] = true
				q = append(q, m)
			}
		}
	}
	return seen
}

func TestAct345HintChains(t *testing.T) {
	for _, c := range []struct {
		name  string
		hints [][2]int
		from  int
		to    []int
	}{
		{"act3", act3Hints, 75, []int{76, 77, 78, 79, 80, 81, 82, 83}},
		{"act4", act4Hints, 103, []int{104, 105, 106}},
		{"act5", act5Hints, 109, []int{110, 111, 112, 113, 115, 117, 118}},
	} {
		r := reach(c.from, c.hints)
		for _, l := range c.to {
			if !r[l] {
				t.Errorf("%s: level %d unreachable from %d", c.name, l, c.from)
			}
		}
	}
}

func TestWaypointLevelsAreOnHintChains(t *testing.T) {
	// every act 3 waypoint except the Durance 2 one (a Vis link) must sit on the Act 3 hint chain
	r := reach(75, act3Hints)
	for _, l := range wpLevels[2] {
		if l != 101 && !r[l] {
			t.Errorf("act 3 waypoint level %d not on the hint chain", l)
		}
	}
	r = reach(103, act4Hints)
	if !r[106] {
		t.Error("City of the Damned must be on the act 4 chain")
	}
}

func TestHintsHaveNoDuplicatesAcrossActs(t *testing.T) {
	seen := map[[2]int]bool{}
	for _, h := range allHints() {
		k := [2]int{min(h[0], h[1]), max(h[0], h[1])}
		if seen[k] {
			t.Errorf("duplicate hint %v", h)
		}
		seen[k] = true
	}
}

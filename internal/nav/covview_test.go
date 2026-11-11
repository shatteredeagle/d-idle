package nav

import (
	"testing"

	"d2lod-bot/internal/input"
)

func TestCoverageView(t *testing.T) {
	n := &Navigator{}
	if _, ok := n.CoverageView(); ok {
		t.Fatal("no roam state, no view")
	}
	n.cov = &coverage{level: 2, pts: []input.Point{{X: 1, Y: 1}, {X: 9, Y: 1}, {X: 17, Y: 1}},
		seen: []bool{true, false, false}, failed: []bool{false, true, false}, legs: 4, held: 1}
	v, ok := n.CoverageView()
	if !ok || v.Explored || v.Legs != 4 || v.Held != 1 || v.Level != 2 || v.Target != nil {
		t.Fatalf("%+v", v)
	}
	if v.Pts[0].State != CoverSeen || v.Pts[1].State != CoverFailed || v.Pts[2].State != CoverUnseen {
		t.Fatalf("%+v", v.Pts)
	}
	n.cov.tgt, n.cov.hasTgt = input.Point{X: 17, Y: 1}, true
	n.cov.seen[2] = true
	v, _ = n.CoverageView()
	if !v.Explored || v.Target == nil || v.Target.X != 17 {
		t.Fatalf("%+v", v)
	}
}

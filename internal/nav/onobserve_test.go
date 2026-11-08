package nav

import (
	"testing"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

// Env.OnObserve (the senses minimap feed) gets the level map and the tracker's hero position from observe, and setting
// it changes nothing observe returns.
func TestObserveCallsHook(t *testing.T) {
	_, n := newKnown(t, openField(2000, 2000, 120, 120), input.Point{X: 2050, Y: 2060}, true)
	base, err := n.observe()
	if err != nil {
		t.Fatal(err)
	}
	var gotLM *d2map.LevelMap
	var gotPos d2map.Point
	calls := 0
	n.e.OnObserve = func(lm *d2map.LevelMap, p d2map.Point) { gotLM, gotPos, calls = lm, p, calls+1 }
	o, err := n.observe()
	if err != nil || calls != 1 || gotLM != o.lm || gotPos != o.state.Pos {
		t.Fatalf("calls %d lm %v pos %v err %v", calls, gotLM == o.lm, gotPos, err)
	}
	if o.pos != base.pos || o.level != base.level {
		t.Fatalf("hook changed the observation: %+v vs %+v", o.pos, base.pos)
	}
}

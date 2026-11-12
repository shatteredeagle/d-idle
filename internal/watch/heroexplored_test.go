package watch

import (
	"encoding/json"
	"testing"
)

// /api/hero carries the additive explored block: absent without a source, in town (no coverage) or with 0 points.
func TestHeroExploredBlock(t *testing.T) {
	for _, build := range []string{"sorceress", "paladin"} {
		r := newRig(t, build)
		var raw map[string]json.RawMessage
		r.get(HeroRoute, &raw)
		if _, ok := raw["explored"]; ok {
			t.Errorf("%s: explored without a source", build)
		}
		r.sim.Hub.SetExploredSource(func() (int, int, int, int, bool) { return 0, 0, 0, 80, false })
		r.get(HeroRoute, &raw)
		if _, ok := raw["explored"]; ok {
			t.Errorf("%s: explored in town", build)
		}
		r.sim.Hub.SetExploredSource(func() (int, int, int, int, bool) { return 0, 0, 0, 80, true })
		r.get(HeroRoute, &raw)
		if _, ok := raw["explored"]; ok {
			t.Errorf("%s: explored with 0 points", build)
		}
		r.sim.Hub.SetExploredSource(func() (int, int, int, int, bool) { return 63, 100, 63, 80, true })
		var v HeroView
		r.get(HeroRoute, &v)
		if v.Explored == nil || *v.Explored != (HeroExplored{Visited: 63, Total: 100, Pct: 63, Threshold: 80}) {
			t.Errorf("%s: explored %+v", build, v.Explored)
		}
	}
}

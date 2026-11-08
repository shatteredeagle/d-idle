package bot

import (
	"testing"
	"time"
)

// TestRiskFacts: the meter's facts come from the same guard and kit the route guard uses (fixtures only, no game).
func TestRiskFacts(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 14)
	if f := g.r.RiskFacts("Cold Plains"); f.Ok {
		t.Errorf("facts before the first snapshot: %+v", f)
	}
	g.r.poll()
	f := g.r.RiskFacts("Cold Plains")
	if !f.Ok || f.Zone != "Cold Plains" || f.GuardWhy != "" || f.Kit == "" || f.EngagePct != g.r.e.Tune.Resource.EngageLifePct || f.MinEngagePct != g.r.e.Tune.Resource.MinEngageLifePct {
		t.Errorf("Cold Plains: %+v", f)
	}
	if f.LifePct <= 0 || f.LifePct > 100 || f.Dead {
		t.Errorf("life %d dead %v", f.LifePct, f.Dead)
	}
	g.r.avoid = map[int]time.Time{3: g.r.e.Clock.Now().Add(time.Hour)}
	if f := g.r.RiskFacts("Cold Plains"); f.GuardWhy != "banned after repeated deaths" {
		t.Errorf("guard reason: %q", f.GuardWhy)
	}
	if f := g.r.RiskFacts(""); f.Zone == "" {
		t.Errorf("empty zone not filled: %+v", f)
	}
}

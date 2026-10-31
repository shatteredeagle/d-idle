package resource

import (
	"fmt"
	"testing"
)

func TestReplayBloodMoor(t *testing.T) {
	sc, tu := BloodMoorSorc(), DefaultTuning()
	res := Replay(sc, ResourcePolicy(), tu)
	leg := Replay(sc, LegacyPolicy(), tu)
	t.Logf("\n%+v\n%+v", leg, res)
	if res2 := Replay(sc, ResourcePolicy(), tu); res2 != res {
		t.Fatal("replay is not deterministic")
	}
	if leg.StarvedSec < 30 {
		t.Fatalf("scenario should starve the legacy policy, starved %.0fs", leg.StarvedSec)
	}
	if res.StarvedSec > leg.StarvedSec/4 {
		t.Fatalf("policy starved %.0fs vs legacy %.0fs", res.StarvedSec, leg.StarvedSec)
	}
	if res.Deaths > leg.Deaths || res.Deaths > 0 {
		t.Fatalf("deaths %d (legacy %d)", res.Deaths, leg.Deaths)
	}
	if res.KillsPerMin < leg.KillsPerMin*0.9 {
		t.Fatalf("policy %.2f kills/min fell below 90%% of legacy %.2f", res.KillsPerMin, leg.KillsPerMin)
	}
}

// The conclusion must not hinge on the U regen figure: halve and double it.
func TestReplayRegenSensitivity(t *testing.T) {
	for _, secs := range []float64{50, 100, 200} {
		sc := BloodMoorSorc()
		sc.ManaFullSecs = secs
		res := Replay(sc, ResourcePolicy(), DefaultTuning())
		leg := Replay(sc, LegacyPolicy(), DefaultTuning())
		t.Run(fmt.Sprint(secs), func(t *testing.T) {
			t.Logf("legacy kills=%d starved=%.0f deaths=%d | policy kills=%d starved=%.0f deaths=%d melee=%d rest=%.0f",
				leg.Kills, leg.StarvedSec, leg.Deaths, res.Kills, res.StarvedSec, res.Deaths, res.MeleeHits, res.RestSec)
			if res.StarvedSec > leg.StarvedSec+1 || res.Deaths > leg.Deaths {
				t.Fatal("policy worse than legacy")
			}
		})
	}
}

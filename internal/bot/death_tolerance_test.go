package bot

import (
	"testing"
	"time"
)

// Dying is normal: with the shipped knobs three deaths in one area while pushing a goal never ban it, the fourth does,
// and the ban is short (SOFTGATE, docs/deathspiral.md "Death tolerance").
func TestDeathToleranceThenBackOff(t *testing.T) {
	g := newRig(t)
	g.r.e.Tune.Bot.RepeatSiteBanMin = 0 // this test pins the spiral breaker alone (repeat-site: deathrule_test.go)
	now := g.r.e.Clock.Now()
	die := func(i int) {
		rec := DeathRecord{Time: now.Add(time.Duration(i) * time.Minute), Area: 2, AreaName: "Blood Moor"}
		g.r.obMu.Lock()
		g.r.dlog.add(rec)
		g.r.obMu.Unlock()
		g.r.deathPolicy(now.Add(time.Duration(i)*time.Minute), &recovery{rec: rec, area: 2})
	}
	for i := 1; i <= 3; i++ {
		die(i)
		if g.r.banned(2) {
			t.Fatalf("banned after %d deaths", i)
		}
	}
	die(4)
	g.r.obMu.Lock()
	until := g.r.avoid[2]
	g.r.obMu.Unlock()
	if until.IsZero() {
		t.Fatal("a real spiral (4 deaths in 4 minutes) must back off")
	}
	if left := until.Sub(now); left > 25*time.Minute {
		t.Fatalf("ban %v is too long", left)
	}
}

// A naked clvl 8 character with potions and gold is not gated in the default graded mode (caster: low risk; a melee
// class without a weapon would be high, see resource/gear_test.go), and the field stays open.
func TestGradedGateKeepsFieldOpenForNakedCaster(t *testing.T) {
	g, s := gearRig(t, true, 400)
	if v := g.r.gearVerdict(s); v.Gated {
		t.Fatalf("naked caster with gold gated: %+v", v)
	}
	if why := g.r.areaAllowed(s, 10); why != "" {
		t.Fatalf("area refused: %s", why)
	}
}

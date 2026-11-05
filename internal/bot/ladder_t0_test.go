package bot

import (
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/ladder"
)

// t0.area.toohard at the tower guard: untaught the ladder returns the legacy verdict; a learned "go" lifts the level
// block, a learned "skip" blocks a level the guard opened (docs/t0-slice.md).
func TestLadderTooHardTower(t *testing.T) {
	ladder.Install(nil)
	g := campaignRig(t, &fakeQuests{}, 14)
	g.r.poll()
	s := g.r.latest()
	legacy := g.r.towerGuardWhy(s)
	if !strings.Contains(legacy, "no merc, clvl 14 < 15") {
		t.Fatalf("legacy verdict %q", legacy)
	}
	l, err := ladder.Open("", "bot", &decide.ManualClock{}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	ladder.Install(l)
	if got := g.r.towerGuardWhy(s); got != legacy {
		t.Fatalf("untaught: %q, legacy %q", got, legacy)
	}
	if st := l.Stats(); st.Asked == 0 || st.Faults != 0 {
		t.Fatalf("stats %+v", st)
	}
	ladder.Install(nil)

	l, clk := teachLadder(t, ladder.PointTooHard, "go")
	for i := 0; i < 10 && l.Stats().ByRule == 0; i++ {
		g.r.towerGuardWhy(s)
		clk.Advance(5 * time.Second)
		l.Pump()
	}
	if got := g.r.towerGuardWhy(s); strings.Contains(got, "clvl") {
		t.Fatalf("learned go must lift the level block: %q (stats %+v)", got, l.Stats())
	}

	g2 := campaignRig(t, &fakeQuests{}, 20)
	g2.r.e.Tune.Route.TowerMinLife = 0 // the life rule would close it for another reason
	g2.r.poll()
	s2 := g2.r.latest()
	ladder.Install(nil)
	if g2.r.towerGuardWhy(s2) != "" && strings.Contains(g2.r.towerGuardWhy(s2), "clvl") {
		t.Skipf("clvl 20 is not open in this rig")
	}
	l, clk = teachLadder(t, ladder.PointTooHard, "skip")
	for i := 0; i < 10 && l.Stats().ByRule == 0; i++ {
		g2.r.towerGuardWhy(s2)
		clk.Advance(5 * time.Second)
		l.Pump()
	}
	if got := g2.r.towerGuardWhy(s2); !strings.HasPrefix(got, "ladder: skip") {
		t.Fatalf("learned skip must block: %q (stats %+v)", got, l.Stats())
	}
}

// t2.mercenary: untaught the hire gate decides; a learned "wait" postpones the hire.
func TestLadderMercenary(t *testing.T) {
	setup := func() (*rig, *d2.Snapshot) {
		rg := newRig(t)
		mt := &mercTownFake{fakeTown: rg.town, hireOK: true}
		rg.r.d.Town = mt
		s := townSnap(areaLutGholein, 1000)
		s.Player.Class = d2.Sorceress
		rg.r.resInput(s)
		rg.r.e.Clock.Sleep(mercGoneAfter)
		return rg, s
	}
	ladder.Install(nil)
	rg, s := setup()
	if got := rg.r.mercWanted(s); got != "hire" {
		t.Fatalf("legacy: %q", got)
	}
	l, err := ladder.Open("", "bot", &decide.ManualClock{}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	ladder.Install(l)
	rg, s = setup()
	if got := rg.r.mercWanted(s); got != "hire" || l.Stats().Asked == 0 || l.Stats().Faults != 0 {
		t.Fatalf("untaught: %q %+v", got, l.Stats())
	}
	ladder.Install(nil)

	l, clk := teachLadder(t, ladder.PointMerc, "wait")
	rg, s = setup()
	for i := 0; i < 10 && l.Stats().ByRule == 0; i++ {
		rg.r.mercWanted(s)
		clk.Advance(5 * time.Second)
		l.Pump()
	}
	if got := rg.r.mercWanted(s); got != "" {
		t.Fatalf("learned wait: %q (%+v)", got, l.Stats())
	}
}

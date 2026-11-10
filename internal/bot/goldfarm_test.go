package bot

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goldfarm"
)

// FarmRung is data driven: the zone one rung below the plan's pick (docs/goldfarm.md), for a caster and a melee class.
func TestFarmRungOneRungDown(t *testing.T) {
	p, _ := LoadPlan("")
	act1 := map[int]bool{1: true}
	safe := map[int]bool{}
	for _, s := range p.Steps["normal"] {
		if !safeTier[s.Area] {
			safe[s.Area] = true
		}
	}
	for _, c := range []struct {
		name  string
		melee bool
		level int
		ex    map[int]bool
		want  int
		state RungState
	}{
		{"caster clvl 1 farms the first field", false, 1, safe, 2, RungOK},
		{"caster clvl 10: Cave 2 is the pick, Stony Field one rung down", false, 10, nil, 4, RungOK},
		{"melee clvl 10 shares the table", true, 10, nil, 4, RungOK},
		{"caster clvl 7: Cave 1 is the pick, Cold Plains below it", false, 7, nil, 3, RungOK},
		{"melee clvl 7 (Stony Field opens at 7 for it): Cave 1 below", true, 7, nil, 9, RungOK},
		{"Stony Field skipped by the breaker: one rung further", false, 10, map[int]bool{4: true}, 9, RungOK},
		{"two rungs skipped", false, 10, map[int]bool{4: true, 9: true}, 3, RungOK},
		{"all rungs excluded", false, 10, map[int]bool{4: true, 9: true, 3: true, 8: true, 2: true}, 0, RungNone},
	} {
		st, state := p.FarmRung("normal", c.level, PickOptions{Acts: act1, Exclude: c.ex, Melee: c.melee, FullKit: true}, safe)
		if state != c.state || st.Area != c.want {
			t.Errorf("%s: area %d state %d, want %d / %d", c.name, st.Area, state, c.want, c.state)
		}
	}
}

// A level below the first zone's entry level has no safe zone at all.
func TestFarmRungTooLow(t *testing.T) {
	p, _ := LoadPlan("")
	p.Steps["normal"][0].EnterLvl = 3 // the plan's first field opens later (a data change, no code)
	for _, melee := range []bool{false, true} {
		if _, state := p.FarmRung("normal", 1, PickOptions{Melee: melee}, nil); state != RungTooLow {
			t.Errorf("melee %v: state %d, want too low", melee, state)
		}
	}
}

func farmRig(t *testing.T, level int, class d2.Class) (*rig, *goldfarm.Controller) {
	g := campaignRig(t, &fakeQuests{}, level)
	g.w.set(func(w *world) {
		w.area, w.level, w.class, w.gold0 = 3, level, class, 20
		w.items = append(w.items, d2.Unit{ID: 900, Type: d2.TypeItem, Class: 5, Item: &d2.Item{Where: d2.ContInventory}}) // a town portal scroll
	})
	g.w.belt()
	g.r.poll()
	c, _ := goldfarm.New("", nil)
	g.r.SetGoldFarm(c, func() int { return 60 })
	return g, c
}

// With the order on, the area choice is the rung down, whatever the class; off, the normal choice stands.
func TestGoldFarmTargetBothClasses(t *testing.T) {
	for _, cl := range []d2.Class{d2.Sorceress, d2.Paladin} {
		g, c := farmRig(t, 10, cl)
		s := g.r.latest()
		if c.Start(); !g.r.goldFarmOn() {
			t.Fatal("order not on")
		}
		a, ok, stop := g.r.goldFarmTarget(s)
		if !ok || stop || a != 4 {
			t.Errorf("%v: area %d ok %v stop %v, want Stony Field", cl, a, ok, stop)
		}
		v := c.View()
		if v.Area == "" || v.Want <= v.Have || v.Text == "" {
			t.Errorf("%v: status %+v", cl, v)
		}
		// the breaker skipped Stony Field: a lower zone, never the skipped one
		g.r.skipArea(4, 3*60e9, whyNavStuck)
		a, ok, _ = g.r.goldFarmTarget(s)
		if !ok || a == 4 || a == 0 {
			t.Errorf("%v: skipped zone chosen again: %d %v", cl, a, ok)
		}
	}
}

// Nothing open: no target and the order stays on (it waits); the status says why.
func TestGoldFarmWaitsWhenEverythingIsAvoided(t *testing.T) {
	g, c := farmRig(t, 10, d2.Sorceress)
	c.Start()
	for a := range safeTier {
		g.r.skipArea(a, 10*60e9, whyNavStuck)
	}
	_, ok, stop := g.r.goldFarmTarget(g.r.latest())
	if ok || !stop || !c.On() {
		t.Fatalf("ok %v stop %v on %v", ok, stop, c.On())
	}
	if v := c.View(); v.Note == "" {
		t.Errorf("no reason on screen: %+v", v)
	}
}

// The order ends itself, and says so, when the level is below every safe zone.
func TestGoldFarmHaltsWhenNoSafeZone(t *testing.T) {
	g, c := farmRig(t, 1, d2.Paladin)
	g.r.d.Plan.Steps["normal"][0].EnterLvl = 4
	c.Start()
	if _, ok, _ := g.r.goldFarmTarget(g.r.latest()); ok || c.On() {
		t.Fatalf("ok %v on %v", ok, c.On())
	}
	if v := c.View(); v.Note == "" || v.On {
		t.Errorf("reason missing: %+v", v)
	}
}

// Target0 honours the order for a leveling goal and leaves poverty's own state moving.
func TestTarget0UsesTheOrder(t *testing.T) {
	g, c := farmRig(t, 10, d2.Sorceress)
	s := g.r.latest()
	off, _ := g.r.target0(s, levelGoalControl())
	c.Start()
	a, ok := g.r.target0(s, levelGoalControl())
	if !ok || a != 4 {
		t.Fatalf("with the order: %d %v (without: %d)", a, ok, off)
	}
	if !g.r.inPoverty() {
		t.Error("poverty hysteresis stopped updating while the order runs (20 gold, no potions)")
	}
}

// A gated hero (gear verdict) only has the first field open: that is the order's zone too. Without a town portal the
// walk home works from the first field only, so the order stays there as well.
func TestGoldFarmKeepsToTheFirstFieldWhenGatedOrWithoutPortal(t *testing.T) {
	g := soakyRig(t, 3)
	c, _ := goldfarm.New("", nil)
	g.r.SetGoldFarm(c, nil)
	c.Start()
	s := g.r.latest()
	if a, ok, _ := g.r.goldFarmTarget(s); !ok || a != g.r.safeFarmArea(s) {
		t.Errorf("gated: %d %v, want %d", a, ok, g.r.safeFarmArea(s))
	}
	g2, c2 := farmRig(t, 10, d2.Paladin)
	g2.w.set(func(w *world) {
		keep := w.items[:0]
		for _, u := range w.items {
			if u.Class != 5 {
				keep = append(keep, u)
			}
		}
		w.items = keep
	})
	g2.r.loop.tpSeen = tpDebounce{} // forget the portal seen by the first poll
	g2.r.poll()
	c2.Start()
	s2 := g2.r.latest()
	if a, ok, _ := g2.r.goldFarmTarget(s2); !ok || a != 2 || g2.r.gearVerdict(s2).Gated {
		t.Errorf("no portal: %d %v, want the first field", a, ok)
	}
}

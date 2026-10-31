package bot

import (
	"errors"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goals"
)

// V: Soaky 2026-10-29 (clvl 8, gold 0, no weapon, no body armor, no potions) and its healthy twin.
func gearRig(t *testing.T, naked bool, gold int) (*rig, *d2.Snapshot) {
	g := newRig(t)
	g.w.set(func(w *world) {
		w.level, w.area, w.gold0 = 8, 1, gold
		if naked {
			w.items = nil // the rig's default character is geared
		}
	})
	s, _ := g.w.snapshot()
	return g, s
}

func TestGearVerdictNakedVersusHealthy(t *testing.T) {
	g, s := gearRig(t, true, 0)
	g.r.e.Tune.Resource.GearGateMode = 0 // these tests pin the old binary gate
	g.w.belt()                           // no potions
	s, _ = g.w.snapshot()
	if v := g.r.gearVerdict(s); !v.Gated || !v.Naked || v.CanRebuy {
		t.Fatalf("soaky: %+v", v)
	}
	g, s = gearRig(t, false, 500)
	if v := g.r.gearVerdict(s); v.Gated || v.Naked {
		t.Fatalf("healthy clvl 8 kit-full: %+v", v)
	}
	g, s = gearRig(t, true, 400)
	g.r.e.Tune.Resource.GearGateMode = 0 // these tests pin the old binary gate
	if v := g.r.gearVerdict(s); !v.Gated || !v.CanRebuy {
		t.Fatalf("naked with gold: %+v", v)
	}
}

// The walk to the Barracks passes Stony Field and Underground Passage 1 (log 12:33 to 12:35): a naked character gets
// refused on the way, a healthy one is let through exactly as before; the way out (town) is always open.
func TestRouteRefusedWhenNakedOrBanned(t *testing.T) {
	route := []int{3, 4, 10, 5}
	g, s := gearRig(t, true, 0)
	err := g.r.routeAllowed(s, 1, 17, route)
	if !errors.Is(err, errAreaGated) {
		t.Fatalf("naked: %v", err)
	}
	if err := g.r.routeAllowed(s, 10, 1, []int{4, 3, 1}); err != nil {
		t.Fatalf("walk to town refused: %v", err)
	}
	if a := g.r.safeFarmArea(s); a != 0 {
		if err := g.r.routeAllowed(s, 1, a, []int{a}); err != nil {
			t.Fatalf("safe area %d refused: %v", a, err)
		}
	}
	g, s = gearRig(t, false, 500)
	if err := g.r.routeAllowed(s, 1, 17, route); err != nil {
		t.Fatalf("healthy refused: %v", err)
	}
	g.r.avoid[10] = g.clock.Now().Add(30 * time.Minute)
	if err := g.r.routeAllowed(s, 1, 17, route); !errors.Is(err, errAreaGated) {
		t.Fatalf("banned area on the route: %v", err)
	}
	if err := g.r.routeAllowed(s, 10, 1, []int{4, 3, 1}); err != nil {
		t.Fatalf("leaving the banned area refused: %v", err)
	}
}

// Four deaths in one area inside the window ban it (dying is normal: three are tolerated).
func TestFourthDeathInAreaBans(t *testing.T) {
	g, _ := gearRig(t, true, 0)
	g.r.e.Tune.Bot.RepeatSiteBanMin = 0 // pins the spiral breaker alone
	now := g.clock.Now()
	for i, at := range []time.Duration{-10 * time.Minute, -6 * time.Minute, -3 * time.Minute, -1 * time.Minute} {
		g.r.dlog.add(DeathRecord{Time: now.Add(at), Area: 10, AreaName: "UP1", Killer: "Vile Hunter"})
		rec := &recovery{rec: *g.r.dlog.last(), area: 10}
		g.r.deathPolicy(now, rec)
		if banned := g.r.banned(10); banned != (i == 3) {
			t.Fatalf("after death %d banned=%v", i+1, banned)
		}
	}
}

func TestSixDeathsInAnHourBan(t *testing.T) {
	g, _ := gearRig(t, true, 0)
	g.r.e.Tune.Bot.RepeatSiteBanMin = 0 // pins the spiral breaker alone
	now := g.clock.Now()
	for i, at := range []time.Duration{-58 * time.Minute, -50 * time.Minute, -42 * time.Minute, -34 * time.Minute, -26 * time.Minute, -1 * time.Minute} {
		g.r.dlog.add(DeathRecord{Time: now.Add(at), Area: 10, AreaName: "UP1"})
		g.r.deathPolicy(now, &recovery{rec: *g.r.dlog.last(), area: 10})
		if banned := g.r.banned(10); banned != (i == 5) {
			t.Fatalf("after death %d banned=%v", i+1, banned)
		}
	}
}

// The newest body is empty because the character was already naked: the gear is on the older one, and that is the one
// to fetch, unless its area is banned.
func TestNakedRetrievesTheOlderGearCorpse(t *testing.T) {
	g, s := gearRig(t, true, 0)
	now := g.clock.Now()
	older := DeathRecord{Time: now.Add(-40 * time.Minute), Area: 2, AreaName: "Blood Moor", CorpseValue: 3000}
	older.Pos.X, older.Pos.Y = 111, 222
	g.r.dlog.add(older)
	g.r.dlog.add(DeathRecord{Time: now.Add(-1 * time.Minute), Area: 2, AreaName: "Blood Moor", CorpseValue: 0})
	rec := &recovery{rec: *g.r.dlog.last(), area: 2}
	ok, why := g.r.retrievalDecision(rec, s)
	if !ok || rec.area != 2 || rec.pos.X != 111 || !rec.target.Equal(older.Time) {
		t.Fatalf("decision %v %q rec %+v", ok, why, rec)
	}
	// banned: no walk into it naked
	g.r.avoid[2] = now.Add(time.Hour)
	rec = &recovery{rec: *g.r.dlog.last(), area: 2}
	if ok, why := g.r.retrievalDecision(rec, s); ok {
		t.Fatalf("banned area retrieved: %q", why)
	}
	// a healthy character with an empty newest corpse does not go looking
	g, s = gearRig(t, false, 500)
	g.r.dlog.add(older)
	g.r.dlog.add(DeathRecord{Time: now.Add(-1 * time.Minute), Area: 2, CorpseValue: 0})
	rec = &recovery{rec: *g.r.dlog.last(), area: 2}
	if ok, _ := g.r.retrievalDecision(rec, s); ok || !rec.target.IsZero() {
		t.Fatal("healthy character chased an old corpse")
	}
}

// End to end on the rig: Soaky's state (clvl 8, no gear, no potions, no gold) given a reach_level goal farms the first
// field only; a geared, kitted twin is planned exactly as before (any area but town).
func TestNakedLevelerStaysInTheFirstField(t *testing.T) {
	g := newRig(t)
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.level, w.area, w.gold0, w.items = 8, 1, 0, nil })
	g.run()
	defer g.finish()
	g.r.Start(goals.Goal{ID: "r", Type: goals.ReachLevel, Params: goals.Params{Level: 12}})
	g.waitFor("farming", func() bool { return g.w.count("explore") > 0 })
	if g.r.Status().Character.AreaID != 2 {
		t.Fatalf("area %d, want 2: %s", g.r.Status().Character.AreaID, g.w.log())
	}
	for _, bad := range []string{"exit 3", "exit 4", "exit 5", "exit 10"} {
		if g.w.count(bad) != 0 {
			t.Fatalf("naked character walked: %s", g.w.log())
		}
	}
}

// The gate follows the equipment: it lifts (and logs "gear gate: off") as soon as a weapon and body armor are worn.
func TestGateLiftsOnceDressed(t *testing.T) {
	g, naked := gearRig(t, true, 0)
	g.r.e.Tune.Resource.GearGateMode = 0 // these tests pin the old binary gate
	g.w.belt()
	naked, _ = g.w.snapshot()
	if v := g.r.gearVerdict(naked); !v.Gated {
		t.Fatalf("naked not gated: %+v", v)
	}
	if g.r.ob.gateKey == "" {
		t.Fatal("gate not noted")
	}
	_, dressed := gearRig(t, false, 0)
	if v := g.r.gearVerdict(dressed); v.Gated {
		t.Fatalf("dressed still gated: %+v", v)
	}
	if g.r.ob.gateKey != "" {
		t.Fatalf("gate state not cleared: %q", g.r.ob.gateKey)
	}
}

// A rebuy that only found gear the character cannot wear (Sorceress Str 10, Quilted needs 12) must not keep the gate
// shut; a stat point or level re-opens the question.
func TestGateSettledWhenRebuyBlocked(t *testing.T) {
	g, s := gearRig(t, true, 400)
	g.r.e.Tune.Resource.GearGateMode = 0 // these tests pin the old binary gate
	g.w.belt()
	s, _ = g.w.snapshot()
	if v := g.r.gearVerdict(s); !v.Gated {
		t.Fatalf("naked not gated: %+v", v)
	}
	g.r.noteBlocked(s.Player, []string{"weapon", "body armor"})
	if v := g.r.gearVerdict(s); v.Gated {
		t.Fatalf("still gated though the rebuy cannot help: %+v", v)
	}
	s2 := *s
	p := *s.Player
	p.Strength += 2
	s2.Player = &p
	if v := g.r.gearVerdict(&s2); !v.Gated {
		t.Fatalf("not gated again after a stat change: %+v", v)
	}
}

// Early-gear policy (docs/sme/early-gear-policy.md): a naked clvl 11 Sorceress in Normal is not gated, makes no rebuy
// trip and walks every area; an Amazon without a weapon is high risk and rebuys.
func TestEarlyGearPolicyGate(t *testing.T) {
	g, _ := gearRig(t, true, 600)
	g.r.e.Tune.Resource.EarlyGearMax = 15
	g.w.set(func(w *world) { w.level = 11 })
	g.w.belt(2)
	g.r.poll()
	s := g.r.latest()
	if v := g.r.gearVerdict(s); v.Gated || v.Naked || v.CanRebuy {
		t.Fatalf("naked Sorceress: %+v", v)
	}
	if _, ok := g.r.rebuyDue(s); ok {
		t.Fatal("rebuy trip for a Sorceress")
	}
	if why := g.r.areaAllowed(s, 2); why != "" {
		t.Fatalf("area refused for a naked Sorceress: %s", why)
	}
	ama := *s
	p := *s.Player
	p.Class = d2.Amazon
	ama.Player = &p
	if v := g.r.gearVerdict(&ama); !v.Gated || !v.Naked || !v.CanRebuy {
		t.Fatalf("Amazon without a weapon: %+v", v)
	}
	if _, ok := g.r.rebuyDue(&ama); !ok {
		t.Fatal("Amazon without a weapon gets no rebuy")
	}
}

package bot

import (
	"context"
	"errors"
	"strings"
	"testing"

	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/task"
)

func TestTowerGuardClosesAreas20To25(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 14)
	g.r.poll()
	s := g.r.latest()
	if why := g.r.towerGuardWhy(s); !strings.Contains(why, "no merc, clvl 14 < 20") {
		t.Fatalf("tower verdict %q", why)
	}
	for a := 20; a <= 25; a++ {
		if why := g.r.areaAllowed(s, a); !strings.Contains(why, "route guard") {
			t.Errorf("area %d: %q", a, why)
		}
	}
	if err := g.r.routeAllowed(s, 3, 25, []int{9, 20, 21, 22, 23, 24, 25}); !errors.Is(err, errAreaGated) {
		t.Fatalf("walk to Tower Cellar 5: %v", err)
	}
	ex := map[int]bool{}
	g.r.guardedAreas(s, ex)
	for a := 20; a <= 25; a++ {
		if !ex[a] {
			t.Errorf("area %d not excluded from the level picker", a)
		}
	}
	// a living merc and a full kit open it
	task.MercAlive = func() (bool, bool) { return true, true }
	defer func() { task.MercAlive = nil }()
	g.w.belt(101, 102, 103, 104)
	g.r.poll()
	if why := g.r.towerGuardWhy(g.r.latest()); strings.Contains(why, "no merc") {
		t.Errorf("merc alive but %q", why)
	}
	// knob 0 turns it off
	g.r.e.Tune.Route.TowerMinLevel = 0
	task.MercAlive = nil
	if why := g.r.towerGuardWhy(g.r.latest()); why != "" {
		t.Errorf("guard off: %q", why)
	}
}

func TestTowerWorldFillsSkipAndLogs(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 14)
	g.r.poll()
	w, err := taskEnv{g.r}.World(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if w.TowerMinLevel != 20 || !w.TowerNeedsMerc || !strings.HasPrefix(w.TowerBlocked, "no merc, clvl 14 < 20") {
		t.Fatalf("world %+v", w)
	}
	if task.TowerSkipWhy(w) == "" {
		t.Error("forgotten_tower must be skipped")
	}
}

func TestCorpseInTowerAbandonedAndWaypointRunRefused(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 14)
	g.r.poll()
	s := g.r.latest()
	rec := &recovery{rec: DeathRecord{Time: g.clock.Now(), Area: 25, CorpseValue: 5000}, area: 25}
	if ok, why := g.r.retrievalDecision(rec, s); ok || !strings.Contains(why, "route guard") {
		t.Fatalf("tower corpse: %v %q", ok, why)
	}
	// the WPUSE waypoint-and-walk path and the portal path both refuse the guarded area
	err := g.r.gotoDeathArea(context.Background(), 25)
	if !errors.Is(err, errAreaGated) {
		t.Fatalf("gotoDeathArea: %v", err)
	}
	if strings.Contains(g.w.log(), "wp ") || strings.Contains(g.w.log(), "exit ") || strings.Contains(g.w.log(), "back ") {
		t.Fatalf("the character moved: %s", g.w.log())
	}
}

func TestStandingInGuardedAreaLeavesFirst(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 14)
	g.w.set(func(w *world) { w.area = 21 })
	g.r.poll()
	c := control{goal: &goals.Goal{}}
	if !g.r.guardedExit(context.Background(), g.r.latest(), c) {
		t.Fatal("no exit")
	}
	if g.w.count("totown") != 1 {
		t.Fatalf("calls: %s", g.w.log())
	}
	// an allowed area (Cold Plains) and town are left alone
	for _, a := range []int{3, 1} {
		g.w.set(func(w *world) { w.area = a })
		g.r.poll()
		if g.r.guardedExit(context.Background(), g.r.latest(), c) {
			t.Errorf("area %d: must not leave", a)
		}
	}
}

func TestRecoveryAllUnreachableInGuardedAreaLeavesFirst(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 14)
	g.w.set(func(w *world) { w.area = 25 })
	g.mover.routeFn = func(from, to int) ([]int, error) { return nil, errors.New("dead end") }
	g.r.poll()
	err := (taskActs{r: g.r, c: control{goal: &goals.Goal{}}}).LevelTo(context.Background(), 15)
	if err != nil {
		t.Fatal(err)
	}
	if g.w.count("totown") != 1 {
		t.Fatalf("calls: %s", g.w.log())
	}
}

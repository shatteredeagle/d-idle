package bot

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/task"
)

func TestTowerGuardClosesAreas20To25(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 14)
	g.r.poll()
	s := g.r.latest()
	if why := g.r.towerGuardWhy(s); !strings.Contains(why, "no merc, clvl 14 < 15") {
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
	if w.TowerMinLevel != 15 || w.TowerNeedsMerc || !strings.HasPrefix(w.TowerBlocked, "no merc, clvl 14 < 15") {
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

// The SME Forgotten Tower rule through the bot: max life from the snapshot, 150 opens it at clvl 15, 149 logs the life reason,
// an unreadable (0) life falls back to 16; the mana part of the kit follows resource.ManaNeed (Barbarian none).
func TestTowerLifeRuleThroughTheBot(t *testing.T) {
	for _, c := range []struct {
		life, lvl int
		want      string // substring of the verdict, "" = open on level and life (kit may still be thin)
	}{
		{150, 15, ""}, {149, 15, "life 149 < 150"}, {0, 15, "clvl 15 < 16 (max life unreadable, fallback floor)"}, {0, 16, ""},
	} {
		g := campaignRig(t, &fakeQuests{}, c.lvl)
		g.w.set(func(w *world) { w.maxL = c.life })
		g.r.poll()
		why := g.r.towerGuardWhy(g.r.latest())
		if c.want == "" && strings.Contains(why, "clvl") || c.want != "" && !strings.Contains(why, c.want) {
			t.Errorf("max life %d clvl %d: %q, want %q", c.life, c.lvl, why, c.want)
		}
	}
}

func TestFillTowerManaFollowsTheManaColumn(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 15)
	rt := g.r.e.Tune.Route
	for _, c := range []struct {
		class d2.Class
		mana  int
	}{{d2.Sorceress, 3}, {d2.Barbarian, 0}} {
		var w task.World
		g.r.fillTower(&w, rt, c.class, &d2.Player{MaxLife: 200})
		if w.TowerKitMana != c.mana || w.MaxLife != 200 || w.TowerMinLife != 150 {
			t.Errorf("%v: %+v", c.class, w)
		}
	}
}

// TestOrderGuardWhy: the read-only fact behind the orders map. It agrees with the guards the walk uses (the tower, the
// Monastery road), is empty for open areas, and a learned or missing ladder never changes or opens a question.
func TestOrderGuardWhy(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 14)
	g.r.poll()
	if why := g.r.OrderGuardWhy(20); !strings.Contains(why, "no merc, clvl 14 < 15") {
		t.Errorf("tower: %q", why)
	}
	if why := g.r.OrderGuardWhy(25); why == "" {
		t.Errorf("tower cellar 5 open")
	}
	if why := g.r.OrderGuardWhy(3); why != "" {
		t.Errorf("Cold Plains: %q", why)
	}
	if why := g.r.OrderGuardWhy(0); why != "" {
		t.Errorf("area 0: %q", why)
	}
	g.r.avoid = map[int]time.Time{3: g.r.e.Clock.Now().Add(time.Hour)}
	if why := g.r.OrderGuardWhy(3); why != "banned after repeated deaths" {
		t.Errorf("banned area: %q", why)
	}
	g.r.e.Tune.Route.TowerMinLevel = 0
	g.r.avoid = nil
	if why := g.r.OrderGuardWhy(20); why != "" {
		t.Errorf("guard off: %q", why)
	}
}

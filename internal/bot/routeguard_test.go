package bot

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/task"
)

func TestRouteGuardBlocksTheMonasteryRoad(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 12)
	g.r.poll()
	s := g.r.latest()
	if why := g.r.routeGuardWhy(s); !strings.Contains(why, "no merc, clvl 12 < 16") {
		t.Fatalf("guard verdict %q", why)
	}
	for _, a := range []int{7, 26, 27, 28, 29, 30, 33, 37} {
		if why := g.r.areaAllowed(s, a); !strings.Contains(why, "route guard") {
			t.Errorf("area %d: %q", a, why)
		}
	}
	for _, a := range []int{2, 3, 4, 5, 6, 17} {
		if why := g.r.areaAllowed(s, a); why != "" {
			t.Errorf("area %d must stay open: %q", a, why)
		}
	}
	err := g.r.routeAllowed(s, 3, 28, []int{7, 26, 27, 28})
	if !errors.Is(err, errAreaGated) || !strings.Contains(err.Error(), "route guard") {
		t.Fatalf("walk to the Barracks: %v", err)
	}
	if err := g.r.routeAllowed(s, 3, 1, []int{2, 1}); err != nil {
		t.Fatalf("walk to town: %v", err)
	}
	// the knob 0 switches the guard off
	g.r.e.Tune.Route.MonasteryMinLevel = 0
	if why := g.r.routeGuardWhy(s); why != "" {
		t.Fatalf("guard off: %q", why)
	}
	if err := g.r.routeAllowed(s, 3, 28, []int{7, 26, 27, 28}); err != nil {
		t.Fatalf("guard off, walk: %v", err)
	}
}

func TestLevelPickerNeverEmptyWhileTheRoadIsClosed(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 12)
	g.r.poll()
	s := g.r.latest()
	ex := map[int]bool{}
	g.r.guardedAreas(s, ex)
	for _, a := range []int{7, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37} {
		if !ex[a] {
			t.Errorf("area %d not excluded", a)
		}
	}
	if ex[3] || ex[17] {
		t.Error("Cold Plains and Burial Grounds must stay pickable")
	}
	t6 := g.r.e.Tune.Bot
	for lvl := 12; lvl <= 30; lvl++ {
		st, ok := g.r.d.Plan.Pick("normal", lvl, PickOptions{Margin: t6.LevelMargin, Slack: t6.LeaveSlack, Acts: map[int]bool{1: true}, Exclude: ex, FullKit: true})
		if !ok || task.RouteGuarded(task.Area(st.Area)) {
			t.Fatalf("clvl %d: picked %+v ok=%v", lvl, st, ok)
		}
	}
}

func TestFillRouteLogsOncePerChange(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 12)
	buf := &bytes.Buffer{}
	g.r.e.Log = slog.New(slog.NewTextHandler(buf, nil))
	g.r.poll()
	for i := 0; i < 3; i++ {
		w, err := taskEnv{g.r}.World(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(w.RouteBlocked, "no merc, clvl 12 < 16") || w.RouteMinLevel != 16 || w.ImbueMinLevel != 20 || !w.ImbueNeedsMerc {
			t.Fatalf("world %+v", w)
		}
	}
	if n := strings.Count(buf.String(), "route guard: blocked monastery (no merc, clvl 12 < 16)"); n != 1 {
		t.Fatalf("%d log lines:\n%s", n, buf.String())
	}
}

func TestCorpseNotRetrievedFromClosedRoadOrCrowdSpot(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 12)
	g.r.poll()
	s := g.r.latest()
	now := g.clock.Now()
	rec := &recovery{rec: DeathRecord{Time: now, Area: 7, CorpseValue: 5000, Autopsy: &Autopsy{HostilesNear: 22}}, area: 7}
	ok, why := g.r.retrievalDecision(rec, s)
	if ok || !strings.Contains(why, "route guard") {
		t.Fatalf("guarded area corpse: %v %q", ok, why)
	}
	// guard off: a crowd death in an area that already killed once in 12 h is not walked into again
	g.r.e.Tune.Route.MonasteryMinLevel = 0
	g.r.obMu.Lock()
	g.r.dlog.add(DeathRecord{Time: now.Add(-3 * time.Hour), Area: 7})
	g.r.dlog.add(rec.rec)
	g.r.obMu.Unlock()
	ok, why = g.r.retrievalDecision(rec, s)
	if ok || !strings.Contains(why, "crowd") {
		t.Fatalf("crowd spot again: %v %q", ok, why)
	}
	// an old death (13 h) does not count
	g2 := campaignRig(t, &fakeQuests{}, 12)
	g2.r.poll()
	g2.r.e.Tune.Route.MonasteryMinLevel = 0
	now = g2.clock.Now()
	rec2 := &recovery{rec: DeathRecord{Time: now, Area: 7, CorpseValue: 5000, Autopsy: &Autopsy{HostilesNear: 22}}, area: 7}
	g2.r.obMu.Lock()
	g2.r.dlog.add(DeathRecord{Time: now.Add(-13 * time.Hour), Area: 7})
	g2.r.dlog.add(rec2.rec)
	g2.r.obMu.Unlock()
	if ok, why := g2.r.retrievalDecision(rec2, g2.r.latest()); !ok {
		t.Fatalf("old death must not block: %q", why)
	}
}

func TestDenFloorPerClassAndPickerExclusion(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 3)
	g.r.e.Tune.Campaign.DenMinLevelCaster, g.r.e.Tune.Campaign.DenMinLevelMelee = 5, 3
	g.r.poll()
	s := g.r.latest()
	for class, want := range map[d2.Class]int{d2.Sorceress: 5, d2.Necromancer: 5, d2.Druid: 5, d2.Amazon: 5, d2.Paladin: 3, d2.Barbarian: 3, d2.Assassin: 3} {
		s.Player.Class = class
		if got := g.r.denFloor(s); got != want {
			t.Errorf("%v: den floor %d, want %d", class, got, want)
		}
	}
	s.Player.Class, s.Player.Difficulty = d2.Sorceress, d2.Normal
	for lvl, excluded := range map[int]bool{3: true, 4: true, 5: false} {
		s.Player.Level = lvl
		ex := map[int]bool{}
		g.r.guardedAreas(s, ex)
		if ex[areaDenOfEvil] != excluded {
			t.Errorf("sorceress clvl %d: den excluded=%v, want %v", lvl, ex[areaDenOfEvil], excluded)
		}
	}
}

func TestEffectiveFloorLinesLoggedOncePerChange(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 12)
	buf := &bytes.Buffer{}
	g.r.e.Log = slog.New(slog.NewTextHandler(buf, nil))
	g.r.poll()
	for i := 0; i < 3; i++ {
		if _, err := (taskEnv{g.r}).World(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	out := buf.String()
	for _, want := range []string{"route guard: monastery floor=16 (merc=none)", "route guard: tower floor=16 (merc=none)", "route guard: andariel floor=17 (merc=none)"} {
		if n := strings.Count(out, want); n != 1 {
			t.Errorf("%q logged %d times:\n%s", want, n, out)
		}
	}
}

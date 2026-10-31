package bot

import (
	"context"
	"errors"
	"testing"
)

func wpRig(t *testing.T, wps map[int]bool, area int) *rig {
	q := &fakeQuests{}
	q.v.WPs = wps
	g := campaignRig(t, q, 12)
	g.w.set(func(w *world) { w.area = area })
	g.r.poll()
	return g
}

func TestLeavingTownUsesActivatedWaypoint(t *testing.T) {
	g := wpRig(t, map[int]bool{1: true, 4: true}, 1)
	if err := g.r.travelTo(context.Background(), 4); err != nil {
		t.Fatal(err)
	}
	if g.w.count("wp 4") != 1 || g.w.count("exit") != 0 {
		t.Fatalf("calls: %v", g.w.log())
	}
}

func TestUseWaypointsKnobOffWalks(t *testing.T) {
	g := wpRig(t, map[int]bool{1: true, 4: true}, 1)
	g.r.e.Tune.Nav.UseWaypoints = false
	if err := g.r.travelTo(context.Background(), 4); err != nil {
		t.Fatal(err)
	}
	if g.w.count("wp ") != 0 || g.w.count("exit 4") != 1 {
		t.Fatalf("calls: %v", g.w.log())
	}
}

// 4 -> 6 skips two hops: with wpMinDistance 3 the bot walks.
func TestWpMinDistanceWalksShortRoutes(t *testing.T) {
	g := wpRig(t, map[int]bool{4: true, 6: true}, 4)
	g.r.e.Tune.Nav.WpMinDistance = 3
	if err := g.r.travelTo(context.Background(), 6); err != nil {
		t.Fatal(err)
	}
	if g.w.count("wp ") != 0 || g.w.count("exit 6") != 1 {
		t.Fatalf("calls: %v", g.w.log())
	}
}

// The waypoint never carries the bot through a guarded area: the guard refuses before any click.
func TestWaypointDoesNotBypassRouteGuard(t *testing.T) {
	g := wpRig(t, map[int]bool{3: true, 27: true}, 3)
	g.mover.routeFn = func(from, to int) ([]int, error) { return []int{7, 26, 27}, nil }
	err := g.r.travelTo(context.Background(), 27)
	if !errors.Is(err, errAreaGated) {
		t.Fatalf("err = %v", err)
	}
	if g.w.count("wp ") != 0 {
		t.Fatalf("calls: %v", g.w.log())
	}
}

func TestCorpseRunPrefersWaypointWhenPortalLeadsElsewhere(t *testing.T) {
	g := wpRig(t, map[int]bool{1: true, 4: true, 6: true}, 1)
	g.r.ob.portalFrom = 3 // the last scroll was read in Cold Plains
	if err := g.r.gotoDeathArea(context.Background(), 6); err != nil {
		t.Fatal(err)
	}
	if g.w.count("back ") != 0 || g.w.count("wp 6") != 1 {
		t.Fatalf("calls: %v", g.w.log())
	}
}

func TestCorpseRunKeepsPortalWhenItLeadsToTheCorpse(t *testing.T) {
	g := wpRig(t, map[int]bool{1: true, 4: true, 6: true}, 1)
	g.r.ob.portalFrom = 6
	if err := g.r.gotoDeathArea(context.Background(), 6); err != nil {
		t.Fatal(err)
	}
	if g.w.count("back 6") != 1 || g.w.count("wp ") != 0 {
		t.Fatalf("calls: %v", g.w.log())
	}
}

func TestCorpseRunWaypointKnobOffUsesPortal(t *testing.T) {
	g := wpRig(t, map[int]bool{1: true, 6: true}, 1)
	g.r.e.Tune.Nav.UseWaypoints = false
	g.r.ob.portalFrom = 3
	if err := g.r.gotoDeathArea(context.Background(), 6); err != nil {
		t.Fatal(err)
	}
	if g.w.count("back 6") != 1 || g.w.count("wp ") != 0 {
		t.Fatalf("calls: %v", g.w.log())
	}
}

// A corpse in a guarded area stays refused on the waypoint path too.
func TestCorpseRunWaypointRespectsGuard(t *testing.T) {
	g := wpRig(t, map[int]bool{1: true, 27: true}, 1)
	g.mover.routeFn = func(from, to int) ([]int, error) { return []int{2, 3, 7, 26, 27}, nil }
	g.r.ob.portalFrom = 3
	if err := g.r.gotoDeathArea(context.Background(), 27); !errors.Is(err, errAreaGated) {
		t.Fatalf("err = %v", err)
	}
	if g.w.count("wp ") != 0 {
		t.Fatalf("calls: %v", g.w.log())
	}
}

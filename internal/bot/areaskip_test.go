package bot

import (
	"context"
	"d2lod-bot/internal/resource"
	"strings"
	"testing"
	"time"
)

// A nav-stuck skip is the short knob, never the old quarter of an hour.
func TestNavStuckSkipIsShort(t *testing.T) {
	g := soakyRig(t, 2)
	a := g.r.safeFarmArea(g.r.latest()) + 1
	g.r.spec = levelGoalControl().spec
	g.r.ob.targetArea = a
	g.r.skipTarget(whyNavStuck)
	left := g.r.avoid[a].Sub(g.clock.Now())
	if left <= 0 || left > 3*time.Minute || g.r.avoidWhy[a] != whyNavStuck {
		t.Fatalf("skip %v why %q", left, g.r.avoidWhy[a])
	}
}

// Every area skipped for a nav fault: the bot waits at most areaIdleMax, then farms the shortest-skip area.
func TestAllSkippedFarmsLeastBadWithinAMinute(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 8)
	g.r.poll()
	routeVia(g, 99)
	now := g.clock.Now()
	g.r.avoid[99] = now.Add(15 * time.Minute)
	g.r.avoidWhy[99] = whyNavStuck
	g.r.avoid[98] = now.Add(10 * time.Minute)
	g.r.avoidWhy[98] = whyNavStuck
	c := levelGoalControl()
	for i := 0; i < 20 && g.r.banned(98); i++ {
		g.r.noTarget(context.Background(), g.r.latest(), c)
	}
	waited := g.clock.Now().Sub(now)
	if g.r.banned(98) || !g.r.banned(99) || waited > 70*time.Second {
		t.Fatalf("98 banned %v 99 banned %v after %v", g.r.banned(98), g.r.banned(99), waited)
	}
}

// A death streak waits longer but still ends, and it frees the safe starter field before another banned area.
func TestDeathStreakPrefersStarterAndNeverIdlesLong(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 8)
	g.r.poll()
	var starter int
	for a := 1; a < 200 && starter == 0; a++ {
		if !g.r.d.Plan.IsTown(a) && g.r.isStarter(a) {
			starter = a
		}
	}
	if starter == 0 {
		t.Skip("no starter area in the plan")
	}
	other := starter + 50
	now := g.clock.Now()
	g.r.avoid[starter], g.r.avoidWhy[starter] = now.Add(20*time.Minute), whyDeath
	g.r.avoid[other], g.r.avoidWhy[other] = now.Add(5*time.Minute), whyDeath
	for i := 0; i < 80 && g.r.banned(starter); i++ {
		g.r.idleBreak(false)
		g.clock.Sleep(5 * time.Second)
	}
	waited := g.clock.Now().Sub(now)
	if g.r.banned(starter) || !g.r.banned(other) || waited > 4*time.Minute {
		t.Fatalf("starter banned %v other banned %v after %v", g.r.banned(starter), g.r.banned(other), waited)
	}
}

// A fresh ban does not wait out the cap when it is the only wait and short; the log names the producer.
func TestIdleBreakForcedByStallDetector(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 8)
	g.r.poll()
	g.r.skipArea(99, 10*time.Minute, whyNavStuck)
	if !g.r.idleBreak(true) || g.r.banned(99) {
		t.Fatal("forced break did not lift the ban")
	}
	if g.r.idleBreak(true) {
		t.Fatal("nothing left to lift")
	}
}

func TestBorderLoopSkipsTheTargetArea(t *testing.T) {
	g := soakyRig(t, 2)
	for i := 0; i < borderLoopCrossings; i++ {
		g.r.noteCrossing(3)
	}
	if _, ok := g.r.avoid[3]; !ok || g.r.avoidWhy[3] != "border-loop" {
		t.Fatalf("avoid %v why %v", g.r.avoid, g.r.avoidWhy)
	}
}

// codereview3 C2: failures of one target an hour apart do not pile up into the long skip.
func TestTargetFailsDecay(t *testing.T) {
	g := soakyRig(t, 2)
	const target = 5
	for i, want := range []time.Duration{1, 2, 4} {
		if d := g.r.targetFailSkip(target); d != want*g.r.navSkip() {
			t.Fatalf("failure %d: skip %v, want %v", i+1, d, want*g.r.navSkip())
		}
	}
	g.clock.Sleep(time.Hour)
	if d := g.r.targetFailSkip(target); d != g.r.navSkip() {
		t.Fatalf("after an hour one failure must give navSkip, got %v", d)
	}
	// a quick repeat still doubles
	if d := g.r.targetFailSkip(target); d != 2*g.r.navSkip() {
		t.Fatalf("quick repeat %v", d)
	}
	g.r.clearTargetFails(target)
	if len(g.r.targetFailLast) != 0 || len(g.r.targetFails) != 0 {
		t.Fatal("clear left state")
	}
}

// codereview3 C3: a failed way to town of a recovery kit trip counts toward KitTripCap like failed chores.
func TestFailedToTownCountsRecoveryKitTrip(t *testing.T) {
	for _, capN := range []int{2, 3} {
		g := newRig(t)
		g.e.Tune.Resource.KitTripCap = capN
		g.town.toTownErr = errBoom
		g.w.set(func(w *world) { w.area = 3; w.level = 7; w.gold0 = 40 })
		g.r.poll()
		for i := 0; i < capN; i++ {
			s := g.r.latest()
			g.r.loop.trip = &trip{plan: ChorePlan{Potions: true, Reason: resource.RecoveryKitReason}, returnTo: s.Player.Area, emergency: true}
			g.r.advanceTrip(context.Background(), s, control{})
		}
		if got := g.r.ob.recKitTrips; got < capN {
			t.Fatalf("cap %d: recKitTrips %d after %d failed trips", capN, got, capN)
		}
	}
}

// codereview3 C1: a nav-fault skip of an area on the route is no danger ban (travelTo may jump over it by waypoint);
// a death ban on the same hop still blocks, and a skip of the target itself still hides the target.
func TestNavSkipOnRouteHopDoesNotBanTheWalk(t *testing.T) {
	for _, lvl := range []int{8, 20} {
		g := campaignRig(t, &fakeQuests{}, lvl)
		g.r.poll()
		s := g.r.latest()
		routeVia(g, 99)
		until := g.clock.Now().Add(10 * time.Minute)
		for _, why := range []string{whyUnreachable, whyNavStuck} {
			g.r.avoid[99], g.r.avoidWhy[99] = until, why
			if h, _ := g.r.routeBan(s, 6); h != 0 {
				t.Fatalf("clvl %d: %s skip of hop 99 banned the route (%d)", lvl, why, h)
			}
			if err := g.r.routeAllowed(s, s.Player.Area, 6, []int{99, 6}); err != nil && strings.Contains(err.Error(), "banned") {
				t.Fatalf("clvl %d: routeAllowed refused for %s: %v", lvl, why, err)
			}
		}
		g.r.avoidWhy[99] = whyDeath
		if h, _ := g.r.routeBan(s, 6); h != 99 {
			t.Fatalf("clvl %d: a death ban must still block the route, got %d", lvl, h)
		}
		if err := g.r.routeAllowed(s, s.Player.Area, 6, []int{99, 6}); err == nil {
			t.Fatalf("clvl %d: routeAllowed must refuse a death-banned hop", lvl)
		}
		g.r.avoid[6], g.r.avoidWhy[6] = until, whyUnreachable
		delete(g.r.avoid, 99)
		if h, _ := g.r.routeBan(s, 6); h != 6 {
			t.Fatalf("clvl %d: the skipped target itself must stay banned, got %d", lvl, h)
		}
	}
}

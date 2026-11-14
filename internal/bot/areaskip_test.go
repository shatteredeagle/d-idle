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
		g.clock.Sleep(borderLoopGap) // real crossings, not one burst (staletarget)
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

// The breaker trips on a few crossings of one target, and each repeat within the memory doubles the skip up to the cap.
func TestBorderLoopSkipEscalates(t *testing.T) {
	g := soakyRig(t, 2)
	trip := func(target int) time.Duration {
		for i := 0; i < borderLoopCrossings; i++ {
			g.r.noteCrossing(target)
			if i < borderLoopCrossings-1 {
				g.clock.Sleep(borderLoopGap)
			}
		}
		return g.r.avoid[target].Sub(g.clock.Now())
	}
	want := borderLoopSkip
	for i := 0; i <= borderLoopDoublings+2; i++ {
		got := trip(9)
		if got != want {
			t.Fatalf("trip %d: skip %v, want %v", i+1, got, want)
		}
		if i < borderLoopDoublings {
			want *= 2
		}
		g.clock.Sleep(got + time.Second)
	}
	// crossings of two different targets never add up
	g2 := soakyRig(t, 2)
	for i := 0; i < borderLoopCrossings; i++ {
		g2.r.noteCrossing(3 + i%2)
		g2.clock.Sleep(borderLoopGap)
	}
	if len(g2.r.avoid) != 0 {
		t.Fatalf("mixed targets tripped: %v", g2.r.avoid)
	}
	// a long quiet spell forgets the repeats
	g.clock.Sleep(borderLoopMemory + time.Minute)
	if got := trip(9); got != borderLoopSkip {
		t.Fatalf("after the memory: %v", got)
	}
}

// Arrival bounces (docs/passagebounce.md): three stays under bounceMaxStay that go straight back skip the bounced area,
// whichever side is the walk's target; the skip escalates; a long stay or a town hop counts as nothing.
func TestArrivalBounceSkipsTheBouncedArea(t *testing.T) {
	g := soakyRig(t, 4)
	hop := func(area int, stay time.Duration) {
		g.clock.Sleep(stay)
		g.w.set(func(w *world) { w.area = area })
		g.r.poll()
	}
	bounce := func() { // Stony Field -> Underground Passage 1 -> Stony Field in 1.2 s
		hop(10, 5*time.Second)
		hop(4, 1200*time.Millisecond)
	}
	bounce()
	bounce()
	if _, ok := g.r.avoid[10]; ok {
		t.Fatal("two bounces tripped the breaker")
	}
	bounce()
	if got := g.r.avoid[10].Sub(g.clock.Now()); got <= 0 || got > borderLoopSkip {
		t.Fatalf("skip after 3 bounces: %v, want within %v", got, borderLoopSkip)
	}
	if g.r.avoidWhy[10] != "border-loop" {
		t.Fatalf("reason %q", g.r.avoidWhy[10])
	}
	// the second trip within the memory doubles it
	g.clock.Sleep(borderLoopSkip + time.Second)
	bounce()
	bounce()
	bounce()
	if got := g.r.avoid[10].Sub(g.clock.Now()); got <= borderLoopSkip || got > 2*borderLoopSkip {
		t.Fatalf("second trip: %v, want within (%v, %v]", got, borderLoopSkip, 2*borderLoopSkip)
	}
	// a long stay is not a bounce
	g2 := soakyRig(t, 4)
	for i := 0; i < 6; i++ {
		g2.clock.Sleep(10 * time.Second)
		g2.w.set(func(w *world) { w.area = 10 })
		g2.r.poll()
		g2.clock.Sleep(30 * time.Second)
		g2.w.set(func(w *world) { w.area = 4 })
		g2.r.poll()
	}
	if len(g2.r.avoid) != 0 {
		t.Fatalf("long stays tripped: %v", g2.r.avoid)
	}
}

// A bounce between a cave mouth and the transit area beside it (Den of Evil and Blood Moor, V log 2026-11-12) skips the
// cave, never the transit area every route out of the town crosses (docs/hourfix.md).
func TestBounceAtACaveMouthSkipsTheCaveNotTheTransit(t *testing.T) {
	g := soakyRig(t, 8)
	hop := func(area int, stay time.Duration) {
		g.clock.Sleep(stay)
		g.w.set(func(w *world) { w.area = area })
		g.r.poll()
	}
	if !g.r.isTransit(2) || g.r.isTransit(8) {
		t.Fatalf("transit: area 2 %v (want true), area 8 %v (want false)", g.r.isTransit(2), g.r.isTransit(8))
	}
	for i := 0; i < 3; i++ { // Den -> Blood Moor -> Den in 1.8 s
		hop(2, 5*time.Second)
		hop(8, 1800*time.Millisecond)
	}
	if g.r.banned(2) {
		t.Fatal("the transit area Blood Moor was banned")
	}
	if !g.r.banned(8) || g.r.avoidWhy[8] != "border-loop" {
		t.Fatalf("the cave was not skipped: avoid %v", g.r.avoid)
	}
	if b, _ := g.r.routeBan(g.r.latest(), 3); b != 0 {
		t.Fatalf("route to Cold Plains blocked by area %d", b)
	}
}

// bounce3 (V log 2026-11-12T19:47Z): Cold Plains and Cave Level 1 bounced both ways and the breaker skipped Cold Plains,
// the area every route past the Blood Moor needs. The pair now counts as one and the deeper side (the cave) takes the
// skip, whichever direction the bounces run and whether or not the plan makes Cold Plains a transit.
func TestBounceBetweenAPlainAndItsCaveSkipsTheCave(t *testing.T) {
	for _, start := range []int{3, 9} {
		for _, transit := range []bool{true, false} {
			g := soakyRig(t, start)
			if !transit {
				g.r.d.Plan.Steps = nil // no plan step beyond: isTransit(3) is false, as in the live log
			}
			other := 12 - start // 3 <-> 9
			hop := func(area int, stay time.Duration) {
				g.clock.Sleep(stay)
				g.w.set(func(w *world) { w.area = area })
				g.r.poll()
			}
			hop(other, 30*time.Second)
			for i := 0; i < 3; i++ {
				hop(start, 1500*time.Millisecond)
				hop(other, 700*time.Millisecond)
			}
			if g.r.banned(3) {
				t.Fatalf("start %d transit %v: Cold Plains banned (%v)", start, transit, g.r.avoidWhy)
			}
			if !g.r.banned(9) || g.r.avoidWhy[9] != "border-loop" {
				t.Fatalf("start %d transit %v: the cave was not skipped: %v", start, transit, g.r.avoid)
			}
		}
	}
}

func TestTownDepth(t *testing.T) {
	g := soakyRig(t, 1)
	for area, want := range map[int]int{1: 0, 2: 1, 3: 2, 9: 3, 8: 2, 5: 4} {
		if got := g.r.townDepth(area); got != want {
			t.Fatalf("depth of %d: %d, want %d", area, got, want)
		}
	}
}

func TestBouncesPer100(t *testing.T) {
	for _, c := range []struct {
		b, a int
		want float64
	}{{0, 0, 0}, {34, 99, 34.3}, {48, 105, 45.7}, {0, 12, 0}} {
		if got := bouncesPer100(c.b, c.a); got != c.want {
			t.Fatalf("%d/%d: %v, want %v", c.b, c.a, got, c.want)
		}
	}
}

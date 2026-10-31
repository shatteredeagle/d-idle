package bot

import (
	"context"
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

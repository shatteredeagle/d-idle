package bot

import (
	"context"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/task"
)

// routeVia makes every walk from town cross area via first.
func routeVia(g *rig, via int) {
	g.mover.routeFn = func(from, to int) ([]int, error) { return []int{via, to}, nil }
}

// A target whose route crosses a banned area is skipped, and the skip ends with the ban.
func TestBannedRouteSkipsTargetAndExpires(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 8)
	g.r.poll()
	s := g.r.latest()
	c := levelGoalControl()
	a0, ok := g.r.target(s, c)
	if !ok || a0 == 0 {
		t.Fatalf("no baseline target")
	}
	routeVia(g, 99)
	g.r.avoid[99] = g.clock.Now().Add(20 * time.Minute)
	a, ok := g.r.target(s, c)
	if ok || a != 0 {
		t.Fatalf("every route crosses the ban, got target %d ok %v", a, ok)
	}
	if !strings.Contains(g.r.banWait(), "banned for 2") {
		t.Fatalf("wait text %q", g.r.banWait())
	}
	g.clock.Sleep(21 * time.Minute)
	if g.r.banned(99) {
		t.Fatal("ban did not expire")
	}
	delete(g.r.avoid, a0) // the skip itself ends with the ban; here it is the same instant
	if a, ok := g.r.target(s, c); !ok || a == 0 {
		t.Fatalf("after expiry no target (%d %v)", a, ok)
	}
}

// With nothing allowed the bot waits in town with a clear activity: no failure count, no ladder, no error state.
func TestNothingAllowedWaitsInTown(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 8)
	g.r.poll()
	routeVia(g, 99)
	g.r.avoid[99] = g.clock.Now().Add(20 * time.Minute)
	c := levelGoalControl()
	for i := 0; i < 6; i++ {
		if g.r.noTarget(context.Background(), g.r.latest(), c) {
			t.Fatal("noTarget must not claim progress")
		}
	}
	if st := g.r.State(); st != StateInTown {
		t.Fatalf("state %v", st)
	}
	if !strings.HasPrefix(g.r.activity, "Waiting: ") {
		t.Fatalf("activity %q", g.r.activity)
	}
	if g.r.loop.failures != 0 || g.r.loop.stuckStage != 0 || strings.Contains(g.w.log(), "restart") {
		t.Fatalf("failures %d ladder %d calls %s", g.r.loop.failures, g.r.loop.stuckStage, g.w.log())
	}
}

// A gate refusal from the walk is not an action fault: five in a row never reach the ladder or a restart.
func TestGateRefusalIsNotAFault(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 8)
	g.r.poll()
	routeVia(g, 99)
	g.r.avoid[99] = g.clock.Now().Add(20 * time.Minute)
	for i := 0; i < 6; i++ {
		g.r.travel(context.Background(), g.r.latest(), levelGoalControl(), 6)
	}
	if g.r.loop.failures != 0 || g.r.loop.stuckStage != 0 || g.r.isSelfPaused() {
		t.Fatalf("failures %d ladder %d paused %v", g.r.loop.failures, g.r.loop.stuckStage, g.r.isSelfPaused())
	}
	if strings.Contains(g.w.log(), "restart") {
		t.Fatalf("calls %s", g.w.log())
	}
}

// The task layer's planning block waits visibly when every way out is banned, and never pauses for attention.
func TestPlanningBlockWithBanWaitsWithoutPause(t *testing.T) {
	g := soakyRig(t, 1)
	g.r.avoid[2] = g.clock.Now().Add(20 * time.Minute)
	res := task.Result{Outcome: task.OutBlocked, StepID: "level.8", Kind: task.KindNotFound,
		Err: task.Errorf(task.KindNotFound, "no eligible reachable leveling area")}
	for i := 0; i < planBlockMax+2; i++ {
		if err := g.r.planningBlock(context.Background(), res); err != nil {
			t.Fatal(err)
		}
	}
	if g.r.isSelfPaused() || !strings.HasPrefix(g.r.activity, "Waiting: ") {
		t.Fatalf("paused %v activity %q", g.r.isSelfPaused(), g.r.activity)
	}
}

// A long ban of a first field is shortened once the character stands in town because of it (no deadlock).
func TestStarterAreaBanIsCappedInTown(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 3)
	g.r.poll()
	if !g.r.isStarter(2) || g.r.isStarter(10) {
		t.Fatalf("starter: 2=%v 10=%v", g.r.isStarter(2), g.r.isStarter(10))
	}
	now := g.clock.Now()
	g.r.avoid[2] = now.Add(time.Hour)
	routeVia(g, 2)
	g.r.target(g.r.latest(), levelGoalControl())
	if left := g.r.avoid[2].Sub(now); left > g.r.starterBanMax() {
		t.Fatalf("ban left %v exceeds the cap", left)
	}
	g.clock.Sleep(g.r.starterBanMax() + time.Second)
	if g.r.banned(2) {
		t.Fatal("starter ban did not lift")
	}
}

// DEATHCHAIN: with every farm area banned, a recovery slice waits in town: no failure count, no ladder, no restart.
func TestRecoveryAllBannedWaitsWithoutLadder(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 11)
	g.r.poll()
	routeVia(g, 99)
	g.r.avoid[99] = g.clock.Now().Add(20 * time.Minute)
	c := levelGoalControl()
	for i := 0; i < 8; i++ {
		if g.r.recoveryStep(context.Background(), g.r.latest(), c) {
			t.Fatal("a refused slice must not claim progress")
		}
	}
	if g.r.loop.failures != 0 || g.r.loop.stuckStage != 0 || g.r.isSelfPaused() || strings.Contains(g.w.log(), "restart") {
		t.Fatalf("failures %d ladder %d calls %s", g.r.loop.failures, g.r.loop.stuckStage, g.w.log())
	}
	if st := g.r.State(); st != StateInTown || !strings.HasPrefix(g.r.activity, "Waiting: ") {
		t.Fatalf("state %v activity %q", st, g.r.activity)
	}
}

// DEATHCHAIN: a death streak with an empty purse starts recovery inside the cooldown.
func TestDeathStreakStartsRecoveryInsideCooldown(t *testing.T) {
	g := soakyRig(t, 1)
	g.r.e.Tune.Resource.RecoveryPotionFloor = 0 // the low-kit override is tested below
	now := g.clock.Now()
	g.r.loop.broke.lastEnd = now.Add(-2 * time.Minute)
	g.r.updateRecovery(g.r.latest())
	if g.r.recoveryActive() {
		t.Fatal("recovery started inside the cooldown without deaths")
	}
	for i := 0; i < 3; i++ {
		g.r.dlog.add(DeathRecord{Time: now.Add(-time.Duration(i) * time.Minute), Area: 7})
	}
	g.r.updateRecovery(g.r.latest())
	if !g.r.recoveryActive() {
		t.Fatal("3 deaths at gold 0 must start recovery despite the cooldown")
	}
}

// FIXBELT: an empty belt with an unaffordable kit starts recovery inside the cooldown (Gate 1 rerun belt window).
func TestLowKitStartsRecoveryInsideCooldown(t *testing.T) {
	g := soakyRig(t, 2)
	g.r.loop.broke.lastEnd = g.clock.Now().Add(-2 * time.Minute)
	g.r.updateRecovery(g.r.latest())
	if !g.r.recoveryActive() {
		t.Fatal("no healing potion and gold 0 must start recovery despite the cooldown")
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

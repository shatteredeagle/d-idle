package bot

import (
	"context"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/task"
)

// The classes the staletarget tests run for (build flexibility: nothing here is Sorceress-only).
var staleClasses = []struct {
	name  string
	class d2.Class
}{{"sorceress", d2.Sorceress}, {"paladin", d2.Paladin}}

// richRig is a geared, funded character standing in town (the hero14 Paladin had its kit; nothing is gated).
func richRig(t *testing.T, class d2.Class, level int) *rig {
	g := campaignRig(t, &fakeQuests{}, level)
	g.w.set(func(w *world) { w.area, w.gold0, w.class = 1, 5000, class })
	g.r.poll()
	return g
}

// hero14 log 2026-11-14: the hero waypoints from town into Dark Wood with the farm target of the last run (Blood Moor)
// still set; the guard re-planned every tick and the border-loop breaker banned Blood Moor for 48 minutes. It must drop
// the stale target: no cancel, no stamp, no ban.
func TestStaleTargetDroppedWhenArrivingFromTown(t *testing.T) {
	for _, tc := range staleClasses {
		t.Run(tc.name, func(t *testing.T) {
			g := richRig(t, tc.class, 14)
			g.r.ob.targetArea = 2
			g.w.set(func(w *world) { w.area = 5 }) // the waypoint
			g.r.poll()
			for i := 0; i < 20; i++ {
				if c := g.r.replanCause(g.r.latest(), StateClearing); c != "" {
					t.Fatalf("tick %d: re-plan %q", i, c)
				}
				g.clock.Sleep(500 * time.Millisecond)
			}
			if len(g.r.crossLog) != 0 || len(g.r.avoid) != 0 || g.r.ob.targetArea != 0 {
				t.Fatalf("crossLog %v avoid %v target %d", g.r.crossLog, g.r.avoid, g.r.ob.targetArea)
			}
		})
	}
}

// A real crossing (walked out of the target into its neighbour) still re-plans, but counts once per arrival, and the
// breaker needs five arrivals, not five guard ticks.
func TestRealCrossingCountsOncePerArrival(t *testing.T) {
	g := soakyRig(t, 2)
	g.r.ob.targetArea = 2
	g.w.set(func(w *world) { w.area = 3 })
	g.r.poll()
	for i := 0; i < 20; i++ {
		if c := g.r.replanCause(g.r.latest(), StateClearing); c == "" {
			t.Fatal("a real crossing must re-plan")
		}
		g.clock.Sleep(500 * time.Millisecond)
	}
	if len(g.r.crossLog) != 1 || len(g.r.avoid) != 0 {
		t.Fatalf("crossLog %d avoid %v", len(g.r.crossLog), g.r.avoid)
	}
}

// noteCrossing: two stamps for one target closer than borderLoopGap are one.
func TestNoteCrossingRateLimit(t *testing.T) {
	g := soakyRig(t, 2)
	for i := 0; i < 50; i++ {
		g.r.noteCrossing(3)
		g.clock.Sleep(borderLoopGap / 4)
	}
	if _, banned := g.r.avoid[3]; banned && g.clock.Now().Sub(time.Time{}) > 0 && len(g.r.crossLog) > borderLoopCrossings {
		t.Fatalf("log %d", len(g.r.crossLog))
	}
	g2 := soakyRig(t, 2)
	for i := 0; i < borderLoopCrossings*3; i++ {
		g2.r.noteCrossing(3)
	}
	if len(g2.r.avoid) != 0 {
		t.Fatalf("a burst with no clock movement tripped the breaker: %v", g2.r.avoid)
	}
}

// A border-loop ban on a field the walk passes never refuses the quest route; a death ban still does and says so; a
// non-death ban says its own reason.
func TestBorderLoopBanDoesNotRefuseTheRoute(t *testing.T) {
	for _, tc := range staleClasses {
		t.Run(tc.name, func(t *testing.T) {
			g := richRig(t, tc.class, 8)
			s := g.r.latest()
			now := g.clock.Now()
			g.r.avoid[2], g.r.avoidWhy[2] = now.Add(48*time.Minute), whyBorderLoop
			if err := g.r.routeAllowed(s, 1, 5, []int{2, 3, 4, 5}); err != nil {
				t.Fatalf("border-loop ban refused the route: %v", err)
			}
			g.r.avoid[2], g.r.avoidWhy[2] = now.Add(48*time.Minute), whyDeath
			err := g.r.routeAllowed(s, 1, 5, []int{2, 3, 4, 5})
			if err == nil || !strings.Contains(err.Error(), "repeated deaths") {
				t.Fatalf("death ban: %v", err)
			}
			// a gate ban on a hop the waypoint jumps over is no danger; with no waypoint it is named for what it is
			g.r.avoid[2], g.r.avoidWhy[2] = now.Add(48*time.Minute), whyGate
			if !g.r.d.Plan.Waypoints[1] {
				t.Skip("no waypoint in the town of the fake plan")
			}
			if err := g.r.routeAllowed(s, 1, 5, []int{2, 3, 4, 5}); err != nil {
				t.Fatalf("a hop the waypoint jumps over was refused: %v", err)
			}
			g.r.e.Tune.Nav.UseWaypoints = false
			err = g.r.routeAllowed(s, 1, 5, []int{2, 3, 4, 5})
			if err == nil || strings.Contains(err.Error(), "repeated deaths") || !strings.Contains(err.Error(), whyGate) {
				t.Fatalf("a non-death ban must say its reason: %v", err)
			}
		})
	}
}

// The quest step wins over the loop breaker: a border-loop ban on the step's own area does not stop a quest walk, but
// stops a leveling walk.
func TestQuestStepWinsOverBorderLoopBan(t *testing.T) {
	g := richRig(t, d2.Sorceress, 14)
	s := g.r.latest()
	g.r.avoid[5], g.r.avoidWhy[5] = g.clock.Now().Add(48*time.Minute), whyBorderLoop
	if err := g.r.routeAllowed(s, 1, 5, []int{2, 3, 4, 5}); err == nil {
		t.Fatal("the leveling walk must still respect the ban")
	}
	g.r.questTravel.Store(true)
	if err := g.r.routeAllowed(s, 1, 5, []int{2, 3, 4, 5}); err != nil {
		t.Fatalf("the quest walk was refused: %v", err)
	}
	g.r.questTravel.Store(false)
	// and the quest world does not report the area as banned (the Ready check of every field step)
	if w, err := (taskEnv{g.r}).World(context.Background()); err != nil || w.Banned[5] {
		t.Fatalf("world banned %v err %v", w.Banned, err)
	}
}

// The spin guard: identical refusals back off exponentially up to the cap and never spin.
func TestRefusalSpinGuardBacksOff(t *testing.T) {
	g := soakyRig(t, 1)
	start := g.clock.Now()
	for i := 0; i < 40; i++ {
		g.r.spinGuard(context.Background(), "step: area not allowed now")
	}
	if g.r.loop.task.bj.backoff != refuseSpinMax {
		t.Fatalf("backoff %v, want the cap %v", g.r.loop.task.bj.backoff, refuseSpinMax)
	}
	if waited := g.clock.Now().Sub(start); waited < 10*time.Minute {
		t.Fatalf("40 identical refusals waited only %v", waited)
	}
	g.r.spinGuard(context.Background(), "another refusal")
	if g.r.loop.task.bj.backoff != 0 {
		t.Fatal("a different refusal must reset the guard")
	}
}

// A blocked journey step: after the knob the hero farms the level plan, then retries the step, with a growing wait, and
// the status says why.
func TestBlockedJourneyFarmsThenRetries(t *testing.T) {
	for _, tc := range staleClasses {
		t.Run(tc.name, func(t *testing.T) {
			g := richRig(t, tc.class, 8)
			g.r.Start(goals.Goal{ID: "x", Type: goals.Idle})
			ctx := context.Background()
			c := g.r.control()
			c.spec = levelGoalControl().spec
			ex := task.NewExecutor(taskEnv{g.r}, taskActs{g.r, c})
			why := "area not allowed now: Blood Moor (2): skipped"
			// before the knob: waits (with the spin guard), no farming
			if !g.r.journeyBlocked(ctx, g.r.latest(), c, ex, why) || g.w.count("explore") != 0 {
				t.Fatal("the first tick must wait, not farm")
			}
			if g.r.Status().JourneyBlocked != why {
				t.Fatalf("status %q", g.r.Status().JourneyBlocked)
			}
			g.clock.Sleep(g.r.blockedFarmAfter())
			g.r.journeyBlocked(ctx, g.r.latest(), c, ex, why)
			b := g.r.loop.task.bj
			if b.retryAt.IsZero() || b.retryAt.Sub(g.clock.Now()) > g.r.blockedFarmAfter() || g.w.count("exit")+g.w.count("explore")+g.w.count("clear") == 0 {
				t.Fatalf("no farm slice: retryAt %v calls %s", b.retryAt, g.w.log())
			}
			first := b.retryAt.Sub(g.clock.Now())
			g.clock.Sleep(first + time.Second)
			if g.r.journeyBlocked(ctx, g.r.latest(), c, ex, why) {
				t.Fatal("at the retry time the step must be offered again")
			}
			if g.r.loop.task.bj.n != 1 || !g.r.loop.task.bj.retryAt.IsZero() {
				t.Fatalf("retry state %+v", g.r.loop.task.bj)
			}
			g.r.journeyBlocked(ctx, g.r.latest(), c, ex, why) // blocked again: farms for the doubled wait
			if second := g.r.loop.task.bj.retryAt.Sub(g.clock.Now()); second <= first {
				t.Fatalf("the retry wait did not grow: %v then %v", first, second)
			}
			g.r.journeyOK()
			if g.r.Status().JourneyBlocked != "" || !g.r.loop.task.bj.since.IsZero() {
				t.Fatal("a step that ran must clear the block")
			}
		})
	}
}

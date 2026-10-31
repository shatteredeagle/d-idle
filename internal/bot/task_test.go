package bot

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/task"
)

type fakeDialog struct {
	w *world
	q *fakeQuests
}

func (d fakeDialog) Talk(ctx context.Context, npc, pick string) error {
	d.w.call("talk %s/%s", npc, pick)
	if npc == task.NPCAkara && pick == "reward" {
		d.q.set(task.QDen, task.QuestState{Stage: task.StageDone, Ev: task.EvSave})
	}
	return nil
}

type fakeQuests struct {
	mu sync.Mutex
	v  QuestView
}

func (q *fakeQuests) set(id string, s task.QuestState) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.v.Quests == nil {
		q.v.Quests = map[string]task.QuestState{}
	}
	q.v.Quests[id] = s
}

func (q *fakeQuests) Read(int) (QuestView, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := QuestView{Quests: map[string]task.QuestState{}, WPs: map[int]bool{}}
	for k, v := range q.v.Quests {
		out.Quests[k] = v
	}
	for k, v := range q.v.WPs {
		out.WPs[k] = v
	}
	return out, true
}

func campaignRig(t *testing.T, q *fakeQuests, level int) *rig {
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) {})
	g.w.set(func(w *world) { w.level = level })
	g.r.d.Task = &TaskDeps{Dialog: fakeDialog{g.w, q}, Quests: q, Waypoint: func(ctx context.Context) error { g.w.call("waypoint"); return nil }}
	return g
}

func campaign() goals.Goal {
	return goals.Goal{ID: "c1", Type: goals.Campaign, Params: goals.Params{Template: goals.CampaignNormalAct1}}
}

func TestCampaignRefusedWithoutTaskDeps(t *testing.T) {
	g := newRig(t)
	if err := g.r.Start(campaign()); err == nil || !strings.Contains(err.Error(), "task layer not wired") {
		t.Fatalf("got %v", err)
	}
}

// A fresh Normal Act 1 campaign does the Den: greeting, the way to the Den, a clear, home, the reward talk; the
// reader's Done then keeps the next ticks from doing it again.
func TestCampaignPlaysDenOfEvil(t *testing.T) {
	q := &fakeQuests{}
	g := campaignRig(t, q, 3)
	g.mover.exploreFn = func(ctx context.Context, done func() bool) error { return nil }
	g.run()
	defer g.finish()
	if err := g.r.Start(campaign()); err != nil {
		t.Fatal(err)
	}
	g.waitFor("reward talk", func() bool { return g.w.count("talk Akara/reward") > 0 })
	log := g.w.log()
	i, j, k, m := strings.Index(log, "talk Akara/quest"), strings.Index(log, "exit 8"), strings.Index(log, "explore"), strings.Index(log, "talk Akara/reward")
	if !(i >= 0 && i < j && j < k && k < m) {
		t.Fatalf("Den sequence wrong: %s", log)
	}
	g.waitFor("progress", func() bool { p, ok := g.r.Progress(); return ok && p.Current >= 1 })
	p, _ := g.r.Progress()
	if p.Target != 7 || len(p.Steps) == 0 { // 6 quests plus the free Rogue claim
		t.Fatalf("progress %+v", p)
	}
	time.Sleep(50 * time.Millisecond)
	if n := g.w.count("talk Akara/reward"); n != 1 {
		t.Fatalf("reward claimed %d times", n)
	}
}

// After a restart the save says Den of Evil is done: the campaign must not talk to Akara or redo the quest (leveling
// may still visit the Den for experience).
func TestCampaignDoesNotRedoDenAfterRestart(t *testing.T) {
	q := &fakeQuests{}
	q.set(task.QDen, task.QuestState{Stage: task.StageDone, Ev: task.EvSave})
	g := campaignRig(t, q, 3)
	g.mover.exploreFn = func(ctx context.Context, done func() bool) error { return nil }
	g.run()
	defer g.finish()
	g.r.Start(campaign())
	time.Sleep(150 * time.Millisecond)
	log := g.w.log()
	if strings.Contains(log, "talk Akara") {
		t.Fatalf("Den was redone: %s", log)
	}
	p, ok := g.r.Progress()
	if !ok || p.Current < 1 || len(p.Steps) == 0 || p.Steps[0].State != goals.StateDone {
		t.Fatalf("Den not reported done: %+v", p)
	}
}

// The fighter may set monsters aside; a quest clear then walks to the leftovers instead of trusting "explored".
func TestCampaignClearHuntsLeftovers(t *testing.T) {
	q := &fakeQuests{}
	g := campaignRig(t, q, 3)
	g.mover.exploreFn = func(ctx context.Context, done func() bool) error { return nil }
	g.fight.clearNothing = true // every monster is "ignored" by the fighter
	g.w.set(func(w *world) { w.area = 8 })
	g.w.addMonster(7, 10, 30, 0)
	g.run()
	defer g.finish()
	g.r.Start(campaign())
	g.waitFor("hunt", func() bool { return g.w.count("goto") > 0 })
	if !strings.Contains(g.w.log(), "goto 130,100") {
		t.Fatalf("did not walk to the leftover: %s", g.w.log())
	}
}

// With a Boss hook the task layer's KillBoss delegates to it: a clean run counts as a kill (proxy evidence), an opened
// portal or a left game reaches r.do as the usual sentinels.
func TestKillBossDelegatesToTheBossHook(t *testing.T) {
	g := newRig(t)
	g.r.loop.brWorldFn = func(context.Context) (task.World, error) { return brWorld(30), nil }
	var ran []string
	var next error
	g.r.d.Task = &TaskDeps{Boss: func(ctx context.Context, name string) error { ran = append(ran, name); return next }}
	acts := taskActs{r: g.r}
	if err := acts.KillBoss(context.Background(), task.BossBR); err != nil || len(ran) != 1 {
		t.Fatalf("clean run: %v %v", err, ran)
	}
	if !g.r.loop.task.killed[task.BossBR] || !g.r.killedBoss(task.BossBR) {
		t.Fatal("a clean boss run must count as a kill")
	}
	next = ErrPortalOpened
	if err := acts.KillBoss(context.Background(), task.BossAndy); task.Classify(err) != task.KindFatal || g.r.loop.task.err == nil {
		t.Fatalf("portal: %v (%v)", err, g.r.loop.task.err)
	}
	if g.r.killedBoss(task.BossAndy) {
		t.Fatal("a failed run is no kill")
	}
}

// While a boss script runs, a routine trip (full pack) must not cancel it; an emergency one still may.
func TestRoutineTripWaitsForTheBossRun(t *testing.T) {
	g := newRig(t)
	g.w.set(func(w *world) { w.area = 3 })
	snap, err := g.w.snapshot()
	if err != nil {
		t.Fatal(err)
	}
	g.town.needs, g.town.needsWhy = true, "pack 80% full"
	g.r.bossRun.Store(true)
	if g.r.tripDue(snap) {
		t.Fatal("a routine trip must wait while a boss run is going")
	}
	g.r.bossRun.Store(false)
	if !g.r.tripDue(snap) {
		t.Fatal("the same trip is due once the run is over")
	}
}

// The best-by-level leveling target has no waypoint and a long route: it is refused (so the bot picks something it can
// reach instead of standing in town), while a short route or an activated waypoint on the way is fine.
func TestUnreachableLevelingTargetIsRefused(t *testing.T) {
	q := &fakeQuests{}
	g := campaignRig(t, q, 12)
	g.r.e.Tune.Route.MonasteryMinLevel = 0 // area 27 is on the guarded road: look at the hop rule alone
	g.w.set(func(w *world) { w.area = 1 })
	snap, _ := g.w.snapshot()
	g.mover.routeFn = func(from, to int) ([]int, error) {
		if to == 27 {
			return []int{2, 3, 4, 5, 6, 27}, nil
		}
		if to == 3 {
			return []int{2, 3}, nil
		}
		return nil, errors.New("no route")
	}
	if why := g.r.unreachable(context.Background(), snap, 27); why == "" {
		t.Fatal("27 over six hops with no waypoint must be refused")
	}
	if why := g.r.unreachable(context.Background(), snap, 3); why != "" {
		t.Fatalf("two hops are walkable: %s", why)
	}
	if why := g.r.unreachable(context.Background(), snap, 99); why == "" {
		t.Fatal("no route must be refused")
	}
	q.v.WPs = map[int]bool{1: true, 27: true}
	if why := g.r.unreachable(context.Background(), snap, 27); why != "" {
		t.Fatalf("activated waypoint: %s", why)
	}
}

// Every area worth farming at clvl 12 is cleared: the leveling slice starts a fresh game instead of farming Den of
// Evil or Blood Moor for next to no experience (LIVE i4 soak, 2026-10-29).
func TestLevelToStartsFreshGameWhenWorthyAreasAreCleared(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 12)
	g.r.d.Cycler = fakeCycler{g.w}
	g.w.set(func(w *world) { w.area = 1 })
	g.r.poll()
	g.r.syncGames()
	g.r.loop.cleared = map[int]bool{}
	for _, a := range []int{3, 4, 5, 6, 9, 17, 20, 27, 35, 37} {
		g.r.loop.cleared[a] = true
	}
	if err := (taskActs{r: g.r, c: control{goal: &goals.Goal{}}}).LevelTo(context.Background(), 14); err != nil {
		t.Fatal(err)
	}
	if g.w.count("newgame normal") != 1 {
		t.Fatalf("calls: %v", g.w.log())
	}
}

func TestTaskActsIsAStallRecoverer(t *testing.T) {
	g := newRig(t)
	acts := taskActs{r: g.r}
	var rec task.StallRecoverer = acts
	if err := rec.TeleportToward(context.Background(), 3); !errors.Is(err, task.ErrNoTeleport) {
		t.Fatalf("a Mover that cannot hop must give ErrNoTeleport, got %v", err)
	}
}

// Constraints.NoTownPortal (Ancients): no trip is due while the step runs, the world says so, and the fighter's
// portal advice is dropped by the guard (loop.go).
func TestNoTownPortalConstraint(t *testing.T) {
	g := newRig(t)
	g.w.set(func(w *world) { w.area = 3 })
	snap, err := g.w.snapshot()
	if err != nil {
		t.Fatal(err)
	}
	g.town.needs, g.town.needsWhy = true, "pack 80% full"
	g.r.noTP.Store(true)
	if g.r.tripDue(snap) {
		t.Fatal("a trip must not be due under NoTownPortal")
	}
	w, err := taskEnv{g.r}.World(context.Background())
	if err != nil || !w.NoTownPortal {
		t.Fatalf("world %+v %v", w, err)
	}
	g.r.noTP.Store(false)
	if !g.r.tripDue(snap) {
		t.Fatal("the trip is due again afterwards")
	}
}

// H2: a campaign in the wrong difficulty walks to town, asks for the new game, and retries with a backoff instead of
// spending one try per level for the life of the process.
func TestCampaignDifficultySwitchWalksToTownAndBacksOff(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 30)
	g.r.d.Cycler = fakeCycler{g.w}
	c := control{goal: &goals.Goal{}, spec: goalSpec{kind: goals.Campaign, difficulty: "nightmare", camp: task.Template("nightmare_all", false)}}
	g.w.set(func(w *world) { w.area = 4 })
	snap, _ := g.w.snapshot()
	if acted, hold := g.r.ensureDifficulty(context.Background(), snap, c); !hold || g.w.count("newgame nightmare") != 0 {
		t.Fatalf("out of town: acted %v hold %v, calls %v", acted, hold, g.w.log())
	}
	g.w.set(func(w *world) { w.area = 1 })
	snap, _ = g.w.snapshot()
	g.r.ensureDifficulty(context.Background(), snap, c)
	if g.w.count("newgame nightmare") != 1 {
		t.Fatalf("not asked in town: %v", g.w.log())
	}
	// the fake leaves the game in Normal: the switch did not happen. Within the backoff the campaign holds in town.
	g.w.set(func(w *world) { w.area = 1 })
	snap, _ = g.w.snapshot()
	if acted, hold := g.r.ensureDifficulty(context.Background(), snap, c); acted || !hold || g.w.count("newgame nightmare") != 1 {
		t.Fatalf("backoff: acted %v hold %v, calls %v", acted, hold, g.w.log())
	}
	g.e.Clock.Sleep(5 * time.Minute)
	g.r.ensureDifficulty(context.Background(), snap, c)
	if g.w.count("newgame nightmare") != 2 {
		t.Fatalf("no retry after the backoff: %v", g.w.log())
	}
}

// WP5 replay (Soaky soak, run 091641): 181 boss runs of about 1 s in 87 minutes. The hook below ends every run at once
// with "context canceled"; the runs must be at least bossRestartGap apart on the virtual clock.
func TestBossRunStartsAtMostOncePer30s(t *testing.T) {
	g := newRig(t)
	g.r.loop.brWorldFn = func(context.Context) (task.World, error) { return brWorld(30), nil }
	var starts []time.Time
	g.r.d.Task = &TaskDeps{Boss: func(ctx context.Context, name string) error {
		starts = append(starts, g.r.e.Clock.Now())
		return context.Canceled
	}}
	acts := taskActs{r: g.r}
	for i := 0; i < 181; i++ {
		_ = acts.KillBoss(context.Background(), task.BossBR)
		g.r.e.Clock.Sleep(480 * time.Millisecond) // the executor tick between two starts
	}
	for i := 1; i < len(starts); i++ {
		if gap := starts[i].Sub(starts[i-1]); gap < bossRestartGap {
			t.Fatalf("start %d came %v after the previous one", i, gap)
		}
	}
	if len(starts) != 181 {
		t.Fatalf("starts %d: every call is rate-limited, none dropped", len(starts))
	}
}

// A boss run owns the step: while it runs the guard neither sees a stale leveling target area nor a level change as a
// reason to re-plan; any other clearing action still does, and says why.
func TestGuardDoesNotReplanAwayFromABossRun(t *testing.T) {
	g := newRig(t)
	g.r.loop.brWorldFn = func(context.Context) (task.World, error) { return brWorld(30), nil }
	g.w.set(func(w *world) { w.area = 17 })
	snap, err := g.w.snapshot()
	if err != nil {
		t.Fatal(err)
	}
	g.r.setTargetArea(3) // left over from a Cold Plains leveling slice
	if why := g.r.replanCause(snap, StateClearing); why == "" {
		t.Fatal("outside a boss run a clearing action in a foreign area must re-plan")
	}
	g.r.bossRun.Store(true)
	if why := g.r.replanCause(snap, StateClearing); why != "" {
		t.Fatalf("a boss run was cancelled: %s", why)
	}
	g.r.bossRun.Store(false)

	// KillBoss clears the stale target itself before the script starts
	var target int
	g.r.d.Task = &TaskDeps{Boss: func(ctx context.Context, name string) error {
		g.r.obMu.Lock()
		target = g.r.ob.targetArea
		g.r.obMu.Unlock()
		return nil
	}}
	if err := (taskActs{r: g.r}).KillBoss(context.Background(), task.BossBR); err != nil || target != 0 {
		t.Fatalf("err %v target %d", err, target)
	}
}

// G1 attempts 11 and 12 (V: soak logs 22:02-22:59): every guard safety interrupt during a quest walk was swallowed.
// The task step returned nil for task.OutInterrupted, so do() saw a success and never ran handleSafety: the character
// walked on at 7% life with potions and portal scrolls in the pack until it died. A safety action must be carried out.
func TestCampaignWalkCarriesOutSafetyInterrupt(t *testing.T) {
	for _, a := range []SafetyAction{SafetyDrinkLife, SafetyTownPortal, SafetyChicken} {
		t.Run(a.String(), func(t *testing.T) {
			q := &fakeQuests{}
			g := campaignRig(t, q, 3)
			g.mover.exitBlock = true // the walk to the Den never ends by itself
			armed := true
			g.fight.safety = func(s *d2.Snapshot) SafetyAction {
				if armed && !s.Player.InTown() {
					return a
				}
				return SafetyNone
			}
			g.fight.onAct = func(SafetyAction) { armed = false }
			g.run()
			defer g.finish()
			if err := g.r.Start(campaign()); err != nil {
				t.Fatal(err)
			}
			g.waitFor("walk started", func() bool { return g.w.count("exit 2") > 0 })
			g.w.set(func(w *world) { w.area = 2 }) // out in the field
			g.waitFor("safety action", func() bool { return g.w.count("act "+a.String()) > 0 })
		})
	}
}

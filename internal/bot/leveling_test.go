package bot

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
)

type fakeLeveler struct {
	w     *world
	calls atomic.Int32
	spend bool // set the unspent points to zero
}

func (f *fakeLeveler) Spend(ctx context.Context, s *d2.Snapshot) error {
	f.calls.Add(1)
	f.w.call("spend stat=%d skill=%d", s.Player.StatPoints, s.Player.SkillPoints)
	if f.spend {
		f.w.set(func(w *world) { w.statPts, w.skillPts = 0, 0 })
	}
	return nil
}

type fakeGear struct {
	w       *world
	swaps   atomic.Int32
	shopped atomic.Int32
	budget  atomic.Int32
}

func (g *fakeGear) PendingSwaps(*d2.Snapshot) int { return int(g.swaps.Load()) }
func (g *fakeGear) Equip(ctx context.Context) (GearReport, error) {
	g.w.call("equip")
	g.swaps.Store(0)
	return GearReport{Swaps: []string{"x"}}, nil
}
func (g *fakeGear) Shop(ctx context.Context, budget int) (GearReport, error) {
	g.w.call("shop %d", budget)
	g.shopped.Add(1)
	g.budget.Store(int32(budget))
	return GearReport{}, nil
}

type fakeQuester struct {
	w   *world
	err error
}

func (q fakeQuester) Claim(ctx context.Context, quest string) error {
	q.w.call("claim %s", quest)
	return q.err
}

func reach(n int) goals.Goal {
	return goals.Goal{ID: "r", Type: goals.ReachLevel, Params: goals.Params{Level: n}}
}

func TestPointsAreSpentInTownAndNotInAFight(t *testing.T) {
	var lv *fakeLeveler
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) {})
	lv = &fakeLeveler{w: g.w, spend: true}
	g.r.d.Leveler = lv
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.level, w.statPts, w.skillPts = 2, 5, 1 })
	g.run()
	defer g.finish()
	g.r.Start(reach(10))
	g.waitFor("points spent in town", func() bool { return lv.calls.Load() == 1 })
	g.waitFor("in the field", func() bool { return g.w.count("explore") > 0 })
	// level up in the field with a hostile in range: no spending until it is dead
	g.w.set(func(w *world) { w.level, w.statPts, w.skillPts = 3, 5, 1 })
	g.w.addMonster(7, 10, 8, 0)
	g.waitFor("spent after the fight", func() bool { return lv.calls.Load() >= 2 })
	g.w.mu.Lock()
	calls := append([]string(nil), g.w.calls...)
	g.w.mu.Unlock()
	lastSpend, lastClear := -1, -1
	for i, c := range calls {
		if len(c) >= 5 && c[:5] == "spend" {
			lastSpend = i
		}
		if len(c) >= 5 && c[:5] == "clear" {
			lastClear = i
		}
	}
	if lastClear < 0 || lastSpend < lastClear {
		t.Fatalf("points were spent while a hostile was near: %v", calls)
	}
}

func TestPointsGiveUpAfterThreeFailedAttemptsPerLevel(t *testing.T) {
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { tu.Bot.PointsRetrySec = 1; c.ClearIdleSec = 100000 })
	lv := &fakeLeveler{w: g.w, spend: false} // never spends: the UI is broken
	g.r.d.Leveler = lv
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.level, w.statPts = 2, 5 })
	g.run()
	defer g.finish()
	g.r.Start(reach(10))
	g.waitFor("three attempts", func() bool { return lv.calls.Load() >= 3 })
	for i := 0; i < 200; i++ { // plenty of virtual time passes
		g.e.Clock.Sleep(time.Second)
	}
	time.Sleep(30 * time.Millisecond)
	if n := lv.calls.Load(); n != 3 {
		t.Fatalf("want exactly 3 attempts per level, got %d", n)
	}
	// the next level tries again
	g.w.set(func(w *world) { w.level = 3 })
	g.waitFor("retry at the next level", func() bool { return lv.calls.Load() >= 4 })
}

func TestGearPassEquipsAndShopsInTown(t *testing.T) {
	g := newRig(t)
	gr := &fakeGear{w: g.w}
	gr.swaps.Store(1)
	g.r.d.Gear = gr
	g.mover.exploreFn = blockExplore
	g.w.gold(1000)
	g.run()
	defer g.finish()
	g.r.Start(reach(10))
	g.waitFor("equip", func() bool { return g.w.count("equip") > 0 })
	g.waitFor("shop", func() bool { return gr.shopped.Load() > 0 })
	// (1000 - reserve 100) * 70 % = 630
	if b := gr.budget.Load(); b != 630 {
		t.Fatalf("budget %d, want 630", b)
	}
	if got := g.w.count("shop"); got != 1 {
		t.Fatalf("shopped %d times at one level", got)
	}
}

// EARLYGAME: below earlyGearMax the leveler's gear pass equips what dropped but spends no gold (potions and the scroll first).
func TestGearPassDoesNotShopBelowEarlyGearMax(t *testing.T) {
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { tu.Resource.EarlyGearMax = 15 })
	gr := &fakeGear{w: g.w}
	gr.swaps.Store(1)
	g.r.d.Gear = gr
	g.mover.exploreFn = blockExplore
	g.w.gold(1000)
	g.w.set(func(w *world) { w.level = 9 })
	g.run()
	defer g.finish()
	g.r.Start(reach(12))
	g.waitFor("equip", func() bool { return g.w.count("equip") > 0 })
	time.Sleep(300 * time.Millisecond)
	if n := gr.shopped.Load(); n != 0 {
		t.Fatalf("shopped %d times at clvl 9", n)
	}
}

func TestStarvingCharacterFarmsGoldInTheEasyArea(t *testing.T) {
	g := newRig(t)
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.level = 8; w.gold0 = 0 })
	g.w.belt() // no potions
	g.town.choresErr = errBoom
	g.run()
	defer g.finish()
	g.r.Start(reach(15))
	// level 8 would be Cold Plains or deeper; without potions and gold the plan detours to Blood Moor
	g.waitFor("farming Blood Moor", func() bool { return g.r.Status().Character.AreaID == 2 && g.w.count("explore") > 0 })
	// enough gold: back to the plan area
	g.w.set(func(w *world) { w.gold0 = 400 })
	g.waitFor("back to the plan", func() bool { a := g.r.Status().Character.AreaID; return a == 3 || a == 17 || a == 9 || a == 4 })
}

func TestNoPotionTripWithoutGold(t *testing.T) {
	g := newRig(t)
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.level = 3; w.gold0 = 5 })
	g.w.belt()
	g.run()
	defer g.finish()
	g.r.Start(reach(15))
	g.waitFor("fighting", func() bool { return g.w.count("explore") > 0 })
	time.Sleep(50 * time.Millisecond)
	if n := g.w.count("chores"); n != 0 {
		t.Fatalf("a character with 5 gold made %d potion trips", n)
	}
}

func TestDenOfEvilRunClaimsTheRewardInTown(t *testing.T) {
	g := newRig(t)
	g.r.d.Quester = fakeQuester{w: g.w}
	n := 0
	g.mover.exploreFn = func(ctx context.Context, done func() bool) error {
		n++
		if g.r.Status().Character.AreaID == areaDenOfEvil {
			return nil // nothing left: the Den is cleared
		}
		return blockExplore(ctx, done)
	}
	g.w.set(func(w *world) { w.level = 2 })
	g.run()
	defer g.finish()
	g.r.Start(reach(10))
	g.waitFor("claim", func() bool { return g.w.count("claim den_of_evil") == 1 })
	if g.w.count("totown") == 0 {
		t.Fatalf("the claim needs a town visit: %s", g.w.log())
	}
	time.Sleep(40 * time.Millisecond)
	if g.w.count("claim") != 1 {
		t.Fatalf("claimed more than once: %s", g.w.log())
	}
}

func TestRunForMapsStepsToRuns(t *testing.T) {
	g := newRig(t)
	cases := map[int]struct {
		kind  RunKind
		quest string
	}{areaDenOfEvil: {RunQuest, "den_of_evil"}, areaBurialGround: {RunBoss, ""}, 2: {RunClear, ""}}
	for area, want := range cases {
		run := g.r.runFor("normal", Step{Area: area, Name: "x"})
		if run.Kind != want.kind || run.Quest != want.quest {
			t.Errorf("area %d: %+v", area, run)
		}
	}
	if run := g.r.runFor("nightmare", Step{Area: areaDenOfEvil}); run.Kind != RunClear {
		t.Errorf("the Den is a quest only in Normal: %+v", run)
	}
}

func TestDifficultySwitchIsBannedAfterItDoesNotHappen(t *testing.T) {
	g := newRig(t)
	g.r.d.Cycler = fakeCycler{g.w} // NewGame leaves the difficulty at Normal (the character has not finished the act)
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.level = 60; w.area = 1 })
	g.run()
	defer g.finish()
	g.r.Start(reach(70))
	g.waitFor("asked for nightmare", func() bool { return g.w.count("newgame nightmare") == 1 })
	time.Sleep(50 * time.Millisecond)
	g.w.set(func(w *world) { w.level = 61 }) // next level: the ban, not diffTried, keeps it from asking again
	time.Sleep(80 * time.Millisecond)
	if n := g.w.count("newgame nightmare"); n != 1 {
		t.Fatalf("asked %d times", n)
	}
}

func TestDifficultyWishWaitsAfterRecentDeaths(t *testing.T) {
	g := newRig(t)
	g.w.set(func(w *world) { w.level = 60 })
	s, _ := g.w.snapshot()
	if got := g.r.wantedDifficulty(s); got != "nightmare" {
		t.Fatalf("level 60 wants %q", got)
	}
	g.r.obMu.Lock()
	for i := 0; i < 4; i++ {
		g.r.dlog.add(DeathRecord{Time: g.e.Clock.Now(), Area: 2})
	}
	g.r.obMu.Unlock()
	if got := g.r.wantedDifficulty(s); got != "" {
		t.Fatalf("after 4 deaths wants %q", got)
	}
}

var _ = errors.New

func TestPovertyModeHysteresisAndWalkableFarmArea(t *testing.T) {
	g := newRig(t)
	g.w.belt() // no potions; the fake pack holds no town portal
	g.w.set(func(w *world) { w.level = 12; w.gold0 = 0 })
	snap := func() *d2.Snapshot { s, _ := g.w.snapshot(); return s }
	if !g.r.updateStarvation(snap()) {
		t.Fatal("0 gold, no kit: poverty")
	}
	if a, ok := g.r.starveArea(snap()); !ok || a != 2 {
		t.Fatalf("no portal: only the walkable first field area works, got %d %v", a, ok)
	}
	g.w.set(func(w *world) { w.gold0 = 200 }) // above the kit cost (190) but under cost+margin
	if !g.r.updateStarvation(snap()) {
		t.Fatal("hysteresis: still poor at 200")
	}
	g.w.set(func(w *world) { w.gold0 = 260 })
	if g.r.updateStarvation(snap()) || g.r.inPoverty() {
		t.Fatal("260 gold covers the kit and margin: poverty ends")
	}
}

func TestStarveAreaWithoutPortalIsInTheCharactersAct(t *testing.T) {
	g := newRig(t)
	g.w.belt()
	g.w.set(func(w *world) { w.level = 30; w.gold0 = 0; w.act = 1 })
	s, _ := g.w.snapshot()
	a, ok := g.r.starveArea(s)
	if !ok || g.r.d.Plan.Act(a) != 2 {
		t.Fatalf("act 2 character without a portal: area %d (act %d) ok %v", a, g.r.d.Plan.Act(a), ok)
	}
}

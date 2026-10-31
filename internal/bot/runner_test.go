package bot

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/launch"
)

func TestRunnerImplementsContract(t *testing.T) {
	var _ goals.Runner = (*Runner)(nil)
	var _ goals.Configurer = (*Runner)(nil)
}

func TestStartValidation(t *testing.T) {
	g := newRig(t)
	for name, goal := range map[string]goals.Goal{
		"unknown area":  farm("Nowhere Land"),
		"act 2 area":    {Type: goals.FarmArea, Params: goals.Params{AreaID: 41}},
		"collect_item":  {Type: goals.CollectItem, Params: goals.Params{Count: 1}},
		"bad boss":      {Type: goals.KillBoss, Params: goals.Params{Boss: "Nobody"}},
		"bad run":       farm("run:nothing"),
		"level too big": {Type: goals.ReachLevel, Params: goals.Params{Level: 120}},
	} {
		if err := g.r.Start(goal); err == nil {
			t.Errorf("%s: Start succeeded", name)
		}
	}
	for name, goal := range map[string]goals.Goal{
		"area name":  farm("Cold Plains"),
		"area id":    {Type: goals.FarmArea, Params: goals.Params{AreaID: 3}},
		"run":        farm("run:andariel"),
		"boss":       {Type: goals.KillBoss, Params: goals.Params{Boss: "Andariel"}},
		"reach":      {Type: goals.ReachLevel, Params: goals.Params{Level: 10}},
		"idle":       {Type: goals.Idle},
		"catalog id": farm("Pit Level 1"),
	} {
		if err := g.r.Start(goal); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestFarmAreaTravelsClearsAndEmitsEvents(t *testing.T) {
	g := newRig(t)
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.level = 3 })
	g.run()
	defer g.finish()
	if err := g.r.Start(farm("Cold Plains")); err != nil {
		t.Fatal(err)
	}
	g.waitFor("arrival in Cold Plains", func() bool { return g.w.count("explore") > 0 })
	if got := g.w.log(); !strings.Contains(got, "wp 3") {
		t.Fatalf("unexpected route: %s", got)
	}
	// spawn a pack, a town NPC (never counted) and a boss; the fake fighter kills the living ones
	g.w.addMonster(7, 10, 10, 0)
	g.w.addMonster(8, 10, 12, 0)
	g.w.addMonster(9, 11, -10, 0)
	g.w.addMonster(10, 12, 15, 0)
	g.waitFor("kills", func() bool { return len(g.events(goals.EvKill)) >= 3 })
	time.Sleep(20 * time.Millisecond)
	kills := g.events(goals.EvKill)
	if len(kills) != 3 {
		t.Fatalf("want 3 kills (NPC excluded), got %d: %+v", len(kills), kills)
	}
	var boss, xp int64
	for _, k := range kills {
		if k.Boss {
			boss++
		}
		xp += k.XP
	}
	if boss != 1 || xp != 30 {
		t.Fatalf("boss=%d xp=%d, want 1 and 30", boss, xp)
	}
	var areas []string
	for _, e := range g.events(goals.EvAreaChange) {
		areas = append(areas, e.Area)
	}
	if len(areas) < 2 || areas[0] != "Rogue Encampment" || areas[len(areas)-1] != "Cold Plains" {
		t.Fatalf("area changes: %v", areas)
	}
	st := g.r.Status()
	if st.State != goals.RunnerRunning || st.GoalID != "g1" || st.Character.Name != "Testy" || st.Character.Area != "Cold Plains" {
		t.Fatalf("status %+v", st)
	}
}

func TestStatesFollowTheWork(t *testing.T) {
	g := newRig(t)
	g.mover.exploreFn = func(ctx context.Context, done func() bool) error {
		<-ctx.Done()
		return ctx.Err()
	}
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("explore", func() bool { return g.w.count("explore") > 0 })
	if g.r.State() != StateClearing {
		t.Fatalf("state %s", g.r.State())
	}
	if g.r.Status().Activity != "Fighting in Blood Moor" {
		t.Fatalf("activity %q", g.r.Status().Activity)
	}
}

func TestLevelUpDropAndPotionEvents(t *testing.T) {
	g := newRig(t)
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.level = 5 })
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("in field", func() bool { return g.w.count("explore") > 0 })
	g.w.set(func(w *world) {
		w.level = 6
		// an El rune appears in the inventory, one belt potion is used
		w.items = append(w.items, d2.Unit{ID: 500, Type: d2.TypeItem, Class: 2, Item: &d2.Item{Where: d2.ContInventory, Quality: 2}})
		var keep []d2.Unit
		for _, u := range w.items {
			if u.ID != 101 {
				keep = append(keep, u)
			}
		}
		w.items = keep
	})
	g.waitFor("events", func() bool {
		return len(g.events(goals.EvLevelUp)) > 0 && len(g.events(goals.EvDrop)) > 0 && len(g.events(goals.EvPotion)) > 0
	})
	if d := g.events(goals.EvDrop)[0]; d.Item.Name != "El Rune" || d.Item.Type != "rune" || d.Item.Quality != "normal" {
		t.Fatalf("drop %+v", d.Item)
	}
	if p := g.events(goals.EvPotion)[0]; p.Potion != "Healing Potion" {
		t.Fatalf("potion %+v", p)
	}
	if l := g.events(goals.EvLevelUp)[0]; l.Level != 6 {
		t.Fatalf("level up %+v", l)
	}
}

func TestDisableObserveAvoidsDoubleCounting(t *testing.T) {
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { c.DisableObserve = []string{"kill", "drop", "potion"} })
	g.mover.exploreFn = blockExplore
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("in field", func() bool { return g.w.count("explore") > 0 })
	g.w.addMonster(7, 10, 10, 0)
	g.waitFor("clear", func() bool { return g.w.count("clear") > 0 })
	time.Sleep(20 * time.Millisecond)
	if n := len(g.events(goals.EvKill)); n != 0 {
		t.Fatalf("kills observed despite disable: %d", n)
	}
}

func TestResupplyTripWhenPotionsLow(t *testing.T) {
	g := newRig(t)
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.level = 3 })
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("in field", func() bool { return g.w.count("explore") > 0 })
	g.w.belt(101) // one healing potion left: below the threshold of 2
	g.waitFor("trip", func() bool { return g.w.count("back") > 0 })
	log := g.w.log()
	i, j, k := strings.Index(log, "totown"), strings.Index(log, "chores"), strings.Index(log, "back 2")
	if !(i >= 0 && i < j && j < k) {
		t.Fatalf("trip order wrong: %s", log)
	}
	if !strings.Contains(log, "1 healing potions") {
		t.Fatalf("reason missing: %s", log)
	}
}

func TestNoTripWhenCatalogUnknown(t *testing.T) {
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { d.Catalog = nil })
	g.mover.exploreFn = blockExplore
	g.w.belt()
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("in field", func() bool { return g.w.count("explore") > 0 })
	if g.w.count("totown") != 0 {
		t.Fatal("went to town without potion knowledge")
	}
}

func TestResupplyThresholdKnobChangesBehavior(t *testing.T) {
	// 3 potions: fine at the default threshold of 2, a trip at the handicapped threshold of 4
	run := func(threshold int) bool {
		g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { tu.Bot.ResupplyPotions = threshold })
		g.mover.exploreFn = blockExplore
		g.w.belt(101, 102, 103)
		g.w.set(func(w *world) { w.area = 2 })
		g.run()
		defer g.finish()
		g.r.Start(farm("Blood Moor"))
		g.waitFor("started", func() bool { return g.w.count("explore") > 0 || g.w.count("totown") > 0 })
		time.Sleep(30 * time.Millisecond)
		return g.w.count("totown") > 0
	}
	if run(2) {
		t.Error("default threshold made a trip with 3 potions")
	}
	if !run(4) {
		t.Error("threshold 4 did not make a trip with 3 potions")
	}
}

type fakeCorpse struct {
	w    *world
	gear []d2.Unit
}

func (c *fakeCorpse) Retrieve(ctx context.Context, area int, at input.Point) error {
	c.w.call("corpse %d %d,%d", area, at.X, at.Y)
	c.w.set(func(w *world) { w.items = append(w.items, c.gear...) })
	return nil
}

func gearUnit() d2.Unit {
	return d2.Unit{ID: 900, Type: d2.TypeItem, Class: 4, Item: &d2.Item{Where: d2.ContEquipment, Quality: 7}}
}

func die(g *rig) {
	g.w.set(func(w *world) { w.life, w.mode = 0, combat.ModeDead })
}

// respawn puts the character back in town; gear stays on the corpse.
func respawn(g *rig) {
	g.w.set(func(w *world) {
		w.life, w.mode, w.area, w.x, w.y = 100, 1, 1, 50, 50
		var keep []d2.Unit
		for _, u := range w.items {
			if u.Item.Where != d2.ContEquipment {
				keep = append(keep, u)
			}
		}
		w.items = keep
	})
}

func TestDeathDrillDieRespawnRetrieve(t *testing.T) {
	var cp *fakeCorpse
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { cp = &fakeCorpse{gear: []d2.Unit{gearUnit()}}; d.Corpse = cp })
	cp.w = g.w
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) {
		w.level = 3
		w.area = 2
		w.x, w.y = 300, 310
		w.gold(500)
		w.items = append(w.items, gearUnit())
	})
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("in field", func() bool { return g.w.count("explore") > 0 })
	g.w.addMonster(7, 10, 5, 0)
	g.waitFor("fight", func() bool { return g.w.count("clear") > 0 })
	g.w.addMonster(8, 10, 5, 0) // the killer, alive when we die
	g.waitFor("seen", func() bool { time.Sleep(3 * time.Millisecond); return true })
	die(g)
	g.waitFor("death event", func() bool { return len(g.events(goals.EvDeath)) == 1 })
	g.waitFor("recovering state", func() bool { return g.r.State() == StateRecovering })
	ev := g.events(goals.EvDeath)[0]
	if ev.Area != "Blood Moor" || ev.AreaID != 2 {
		t.Fatalf("death event %+v", ev)
	}
	respawn(g)
	g.waitFor("corpse retrieved", func() bool { return g.w.count("corpse 2 300,310") > 0 })
	g.waitFor("work resumes", func() bool { return g.r.State() == StateClearing && g.w.count("explore") >= 2 })
	if g.w.count("chores after death") != 1 || g.w.count("corpse") != 1 {
		t.Fatalf("log: %s", g.w.log())
	}
	rep := g.r.DeathReport()
	if rep.Total != 1 || len(rep.Records) != 1 {
		t.Fatalf("report %+v", rep)
	}
	r0 := rep.Records[0]
	if !r0.Retrieved || r0.Abandoned || r0.CorpseValue != 3000*20 || r0.Pos.X != 300 || r0.AreaName != "Blood Moor" || len(r0.LifeCurve) == 0 {
		t.Fatalf("record %+v", r0)
	}
	if rep.ByArea[0].Deaths != 1 || len(rep.Heat) != 1 {
		t.Fatalf("report %+v", rep)
	}
}

// Default retrieval: VERIFIED LIVE that walking onto the corpse returns the gear and walking onto the dropped gold pile
// returns the gold; no clicks.
func TestCorpseDefaultWalksOverCorpseAndGold(t *testing.T) {
	g := newRig(t)
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) {
		w.level = 3
		w.area = 2
		w.x, w.y = 300, 310
		w.items = append(w.items, gearUnit())
		w.ground = []d2.Unit{{ID: 90, Type: d2.TypeItem, Class: goldClass, X: 302, Y: 312}, {ID: 91, Type: d2.TypeItem, Class: 2, X: 301, Y: 311}}
	})
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("in field", func() bool { return g.w.count("explore") > 0 })
	die(g)
	g.waitFor("death", func() bool { return len(g.events(goals.EvDeath)) == 1 })
	respawn(g)                                      // possibly faster than a tick: the recovery must still happen
	g.w.set(func(w *world) { w.x, w.y = 300, 310 }) // the fake mover does not walk; stand at the corpse
	g.waitFor("corpse and gold trip", func() bool { return g.w.count("back 2") > 0 && g.w.count("goto 302,312") > 0 })
	g.waitFor("gave up after one try", func() bool { return g.w.count("chores poverty rebuild") >= 1 })
	g.waitFor("work resumes", func() bool { return g.r.State() == StateClearing })
	if g.w.count("goto 300,310") != 1 || g.w.count("goto") != 2 || g.w.count("clickworld") != 0 {
		t.Fatalf("expected the corpse then the gold pile, no click: %s", g.w.log())
	}
	if r0 := g.r.DeathReport().Records[0]; !r0.Abandoned || r0.CostItems != 3 {
		t.Fatalf("record %+v", r0)
	}
}

func TestCorpseBelowValueIsAbandonedWithPovertyChores(t *testing.T) {
	g := newRig(t)
	g.mover.exploreFn = blockExplore
	cheap := gearUnit()
	cheap.Class, cheap.Item.Quality = 3, 2 // plate mail with zero cost in the fake catalog
	g.w.set(func(w *world) { w.level = 3; w.area = 2; w.items = append(w.items, cheap) })
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("in field", func() bool { return g.w.count("explore") > 0 })
	die(g)
	g.waitFor("death", func() bool { return len(g.events(goals.EvDeath)) == 1 })
	respawn(g) // possibly faster than a tick: the recovery must still happen
	g.waitFor("poverty chores", func() bool { return g.w.count("chores poverty rebuild after death") == 1 })
	g.waitFor("work resumes", func() bool { return g.r.State() == StateClearing && g.w.count("exit 2") > 0 })
	if g.w.count("goto") != 0 {
		t.Fatal("walked to a worthless corpse")
	}
	if r0 := g.r.DeathReport().Records[0]; !r0.Abandoned || !(strings.Contains(r0.Why, "threshold") || strings.Contains(r0.Why, "skipped")) {
		t.Fatalf("record %+v", r0)
	}
}

func TestSecondDeathDuringRetrievalAbortsAndNeverRetries(t *testing.T) {
	g := newRig(t)
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.level = 3; w.area = 2; w.items = append(w.items, gearUnit()) })
	g.town.backErr = func(area int) { die(g) } // dies on the way back
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("in field", func() bool { return g.w.count("explore") > 0 })
	die(g)
	g.waitFor("death 1", func() bool { return len(g.events(goals.EvDeath)) == 1 })
	g.town.armed.Store(true)
	respawn(g)
	g.waitFor("death 2", func() bool { return len(g.events(goals.EvDeath)) == 2 })
	g.town.backErr = nil
	respawn(g)
	g.waitFor("work resumes", func() bool { return g.r.State() == StateClearing && g.w.count("exit 2") > 0 })
	if n := g.w.count("back"); n != 1 {
		t.Fatalf("retrieval attempted %d times: %s", n, g.w.log())
	}
	recs := g.r.DeathReport().Records
	if len(recs) != 2 || !strings.Contains(recs[1].Why, "second death") {
		t.Fatalf("records %+v", recs)
	}
}

func TestCorpseAbandonedWhenKnobIsZero(t *testing.T) {
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { tu.Bot.CorpseRetries = 0 })
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.area = 2 })
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("in field", func() bool { return g.w.count("explore") > 0 })
	g.w.set(func(w *world) { w.life, w.mode = 0, 17 })
	g.waitFor("death", func() bool { return len(g.events(goals.EvDeath)) == 1 })
	g.w.set(func(w *world) { w.life, w.mode, w.area = 100, 1, 1 })
	g.waitFor("work resumes", func() bool { return g.w.count("exit 2") > 0 })
	if g.w.count("goto") != 0 {
		t.Fatal("tried a corpse with retries=0")
	}
}

func TestSpiralBreakerAvoidsTheArea(t *testing.T) {
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { tu.Bot.SpiralDeaths = 2; tu.Bot.CorpseRetries = 0 })
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.area = 2; w.bossImmune = true })
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	for i := 1; i <= 2; i++ {
		g.waitFor("in field", func() bool {
			return g.r.State() == StateClearing && g.r.Status().Character.AreaID == 2 && g.r.Status().Character.Life == 100 && len(g.events(goals.EvDeath)) == i-1
		})
		g.w.addMonster(uint32(50+i), 12, 4, 0) // a boss stands next to us
		seen := g.w.snaps.Load() + 3           // the killer is the nearest hostile of the last living snapshot: let the bot look first
		g.waitFor("bot sees the boss", func() bool { return g.w.snaps.Load() >= seen })
		die(g)
		g.waitFor("death", func() bool { return len(g.events(goals.EvDeath)) == i })
		g.waitFor("recovering", func() bool { return g.r.State() == StateRecovering })
		respawn(g)
	}
	// the only way on is the banned area: the bot waits in town with a visible reason (an area ban is no fault)
	g.waitFor("ban wait", func() bool { return strings.HasPrefix(g.r.Status().Activity, "Waiting: Blood Moor banned") })
	errs := g.events(goals.EvError)
	if len(errs) == 0 || errs[0].Fatal || !strings.Contains(errs[0].Message, "Death spiral") || errs[0].Monster != "Andariel" {
		t.Fatalf("want a non-fatal spiral event naming the killer, got %+v", errs)
	}
	rep := g.r.DeathReport()
	if rep.Danger["Andariel"] != 1 || rep.AvoidedAreas[2].IsZero() || rep.ByMonster["Andariel"] != 2 {
		t.Fatalf("report %+v", rep)
	}
}

func TestSpiralBreakerMovesLevelingElsewhere(t *testing.T) {
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { tu.Bot.SpiralDeaths = 2; tu.Bot.CorpseRetries = 0 })
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.area = 2; w.level = 5 })
	g.run()
	defer g.finish()
	g.r.Start(goals.Goal{ID: "r", Type: goals.ReachLevel, Params: goals.Params{Level: 12}})
	for i := 1; i <= 2; i++ {
		g.waitFor("in field", func() bool { return g.r.State() == StateClearing && g.r.Status().Character.AreaID != 1 })
		a := g.r.Status().Character.AreaID
		g.w.set(func(w *world) { w.area = a }) // keep area
		die(g)
		g.waitFor("death", func() bool { return len(g.events(goals.EvDeath)) == i })
		g.waitFor("recovering", func() bool { return g.r.State() == StateRecovering })
		respawn(g)
		if i == 1 {
			g.waitFor("back", func() bool { return g.r.State() == StateClearing })
		}
	}
	// What follows the ban (wait in town, move on, lift) depends on the fake clock, which the waits advance at full speed:
	// area_ban_test.go covers those steps deterministically. Here the bot must only survive the spiral without a fatal.
	for _, ev := range g.events(goals.EvError) {
		if ev.Fatal {
			t.Fatalf("fatal after the spiral: %+v", ev)
		}
	}
}

func TestTooManyDeathsPerHourPausesItself(t *testing.T) {
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) {
		tu.Bot.PauseDeathsPerHour = 2
		tu.Bot.SpiralDeaths = 99
		tu.Bot.CorpseRetries = 0
	})
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.area = 2 })
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	for i := 1; i <= 2; i++ {
		g.waitFor("in field", func() bool { return g.r.State() == StateClearing && g.w.count("explore") == i })
		die(g)
		g.waitFor("death", func() bool { return len(g.events(goals.EvDeath)) == i })
		g.waitFor("recovering", func() bool { return g.r.State() == StateRecovering })
		respawn(g)
	}
	g.waitFor("self pause", func() bool { return g.r.Status().State == goals.RunnerPaused })
	if m := g.r.Status().Message; !strings.Contains(m, "Paused itself") {
		t.Fatalf("message %q", m)
	}
	if !g.r.DeathReport().PausedForDeaths {
		t.Fatal("report does not show the pause")
	}
	n := g.w.count("explore")
	time.Sleep(20 * time.Millisecond)
	if g.w.count("explore") != n {
		t.Fatal("kept playing while paused")
	}
	g.r.Resume()
	g.waitFor("resumed", func() bool { return g.r.Status().State == goals.RunnerRunning && g.w.count("explore") > n })
}

func TestDeathLogPersists(t *testing.T) {
	path := t.TempDir() + "/state/deaths.json"
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { c.DeathLogPath = path })
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.area = 2 })
	g.run()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("in field", func() bool { return g.w.count("explore") > 0 })
	die(g)
	g.waitFor("death", func() bool { return len(g.events(goals.EvDeath)) == 1 })
	g.waitFor("saved", func() bool { _, err := os.Stat(path); return err == nil })
	g.finish()
	h := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { c.DeathLogPath = path })
	if rep := h.r.DeathReport(); rep.Total != 1 || rep.ByArea[0].Area != 2 {
		t.Fatalf("not restored: %+v", rep)
	}
}

func TestDiedTransition(t *testing.T) {
	alive := &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{Life: 10, MaxLife: 100, Mode: 1}}
	dead := &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{Life: 0, MaxLife: 100, Mode: 17}}
	if !diedTransition(alive, dead) || diedTransition(dead, dead) || diedTransition(dead, alive) || diedTransition(nil, alive) {
		t.Fatal("transition rule wrong")
	}
}

func TestPauseResumeStop(t *testing.T) {
	g := newRig(t)
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.area = 2 })
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("clearing", func() bool { return g.r.State() == StateClearing })

	// a hostile nearby: pause must not freeze mid-fight
	g.w.addMonster(7, 10, 5, 0)
	g.r.Pause()
	// The paused state flips at the top of the tick, before pausedFight calls Clear, so wait for the fight itself.
	g.waitFor("paused, fight done", func() bool { return g.r.Status().State == goals.RunnerPaused && g.w.count("clear") > 0 })
	n := g.w.count("explore")
	time.Sleep(20 * time.Millisecond)
	if g.w.count("explore") != n {
		t.Fatal("kept working while paused")
	}
	g.r.Resume()
	g.waitFor("resumed", func() bool { return g.r.Status().State == goals.RunnerRunning && g.w.count("explore") > n })

	g.r.Stop()
	g.waitFor("stopped in town", func() bool { return g.r.Status().State == goals.RunnerStopped && g.w.count("totown") > 0 })
	g.waitFor("stop finished", func() bool { return g.r.Status().Activity == "Stopped" })
	if err := g.r.Stop(); err != nil { // idempotent
		t.Fatal(err)
	}
	if g.r.Status().State != goals.RunnerStopped {
		t.Fatal("not stopped")
	}
}

func TestStartReplacesGoal(t *testing.T) {
	g := newRig(t)
	g.mover.exploreFn = blockExplore
	g.run()
	defer g.finish()
	g.r.Start(farm("Cold Plains"))
	g.waitFor("working in Cold Plains", func() bool { return g.w.count("explore") > 0 })
	g.r.Start(goals.Goal{ID: "g2", Type: goals.FarmArea, Params: goals.Params{Area: "Blood Moor"}})
	g.waitFor("moves to Blood Moor", func() bool {
		return g.r.Status().Character.AreaID == 2 && g.w.count("exit 2") >= 1 && g.r.State() == StateClearing
	})
	if g.r.Status().GoalID != "g2" {
		t.Fatalf("goal id %q", g.r.Status().GoalID)
	}
}

func TestIdleGoalRestsInTown(t *testing.T) {
	g := newRig(t)
	g.w.set(func(w *world) { w.area = 2 })
	g.run()
	defer g.finish()
	g.r.Start(goals.Goal{ID: "i1", Type: goals.Idle})
	g.waitFor("rests", func() bool { return g.r.State() == StateInTown })
	if g.w.count("totown") != 1 || g.w.count("explore") != 0 {
		t.Fatalf("idle goal misbehaved: %s", g.w.log())
	}
	if g.r.Status().State != goals.RunnerRunning { // idle goals accrue time as running
		t.Fatalf("status %+v", g.r.Status())
	}
}

func TestSafetyInterruptsTravelAndDrinks(t *testing.T) {
	g := newRig(t)
	g.mover.wpErr = errBoom // walk through Blood Moor instead
	g.mover.exitDelay = 20 * time.Millisecond
	drunk := false
	g.fight.safety = func(s *d2.Snapshot) SafetyAction {
		if s.Player.Area == 2 && !drunk {
			return SafetyDrinkLife
		}
		return SafetyNone
	}
	g.fight.onAct = func(a SafetyAction) { drunk = true }
	g.mover.exploreFn = blockExplore
	g.run()
	defer g.finish()
	g.r.Start(farm("Cold Plains"))
	g.waitFor("drink then continue", func() bool { return g.w.count("act drink-life") == 1 && g.w.count("explore") > 0 })
	if g.w.count("exit 3") < 2 {
		t.Fatalf("travel was not interrupted and resumed: %s", g.w.log())
	}
}

func TestTownPortalSafetyStartsATrip(t *testing.T) {
	g := newRig(t)
	armed := true
	g.fight.safety = func(s *d2.Snapshot) SafetyAction {
		if armed && s.Player.Area == 3 {
			return SafetyTownPortal
		}
		return SafetyNone
	}
	g.fight.onAct = func(a SafetyAction) {
		armed = false
		g.w.set(func(w *world) { w.area = 1 })
	}
	g.mover.exploreFn = blockExplore
	g.run()
	defer g.finish()
	g.r.Start(farm("Cold Plains"))
	g.waitFor("emergency trip", func() bool { return g.w.count("chores emergency") > 0 && g.w.count("back 3") > 0 })
}

func TestReactionTimeDelaysInterrupt(t *testing.T) {
	at := func(reaction int) time.Duration {
		g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { tu.Bot.ReactionMs = reaction })
		var mu sync.Mutex
		var firstSeen, acted time.Time
		g.fight.safety = func(s *d2.Snapshot) SafetyAction {
			if s.Player.Area == 3 {
				mu.Lock()
				if firstSeen.IsZero() {
					firstSeen = g.clock.Now()
				}
				mu.Unlock()
				return SafetyChicken
			}
			return SafetyNone
		}
		g.fight.onAct = func(a SafetyAction) {
			mu.Lock()
			acted = g.clock.Now()
			mu.Unlock()
			g.w.set(func(w *world) { w.state = d2.StateMenu })
		}
		g.mover.exploreFn = blockExplore
		g.run()
		defer g.finish()
		g.r.Start(farm("Cold Plains"))
		g.waitFor("chicken", func() bool { return g.w.count("act chicken") > 0 })
		mu.Lock()
		defer mu.Unlock()
		return acted.Sub(firstSeen)
	}
	fast, slow := at(0), at(300)
	if slow-fast < 250*time.Millisecond {
		t.Fatalf("reaction knob has no effect: fast=%s slow=%s", fast, slow)
	}
}

func TestWatchdogEscalates(t *testing.T) {
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { c.StuckAfterSec = 30 })
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.area = 2 })
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("stage 1: close panels", func() bool { return g.w.count("closeall") > 0 })
	g.waitFor("stage 2: reset in town", func() bool { return g.w.count("totown") > 0 })
	g.waitFor("last rung: paused itself, visibly", func() bool { return g.r.Status().State == goals.RunnerPaused })
	g.waitFor("the failsafe error event", func() bool { return len(g.events(goals.EvError)) > 0 })
	errs := g.events(goals.EvError)
	if len(errs) != 1 || errs[0].Fatal || !strings.Contains(errs[0].Message, "Failsafe") {
		t.Fatalf("errors %+v", errs)
	}
}

func TestMovementCountsAsProgress(t *testing.T) {
	g := newRig(t)
	g.mover.exploreFn = func(ctx context.Context, done func() bool) error {
		for ctx.Err() == nil {
			g.w.set(func(w *world) { w.x += 5 })
			time.Sleep(time.Millisecond)
		}
		return ctx.Err()
	}
	g.w.set(func(w *world) { w.area = 2 })
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("exploring", func() bool { return g.w.count("explore") > 0 })
	time.Sleep(150 * time.Millisecond) // virtual time runs far past StuckAfterSec
	if g.w.count("closeall") > 0 {
		t.Fatal("watchdog fired while the character was moving")
	}
}

func TestGameGoneIsFatal(t *testing.T) {
	g := newRig(t)
	g.mover.exploreFn = blockExplore
	g.run()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("working", func() bool { return g.w.count("explore") > 0 })
	g.health.mu.Lock()
	g.health.ev = launch.EventCrashed
	g.health.mu.Unlock()
	g.waitFor("fatal", func() bool { return g.r.Status().State == goals.RunnerError && len(g.events(goals.EvError)) > 0 })
	err := g.finish()
	if err == nil || !strings.Contains(err.Error(), "crashed") {
		t.Fatalf("Run returned %v", err)
	}
	errs := g.events(goals.EvError)
	if len(errs) == 0 || !errs[len(errs)-1].Fatal {
		t.Fatalf("no fatal error event: %+v", errs)
	}
	if g.r.Status().State != goals.RunnerError {
		t.Fatal("status not error")
	}
	if g.r.Start(farm("Blood Moor")) == nil {
		t.Fatal("Start accepted after a fatal error")
	}
}

func TestUnreadableMemoryWithoutHealthIsFatal(t *testing.T) {
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { d.Health = nil })
	g.run()
	g.r.Start(farm("Blood Moor"))
	g.w.set(func(w *world) { w.snapErr = errBoom })
	g.waitFor("fatal", func() bool { return g.r.Status().State == goals.RunnerError })
	if err := g.finish(); err == nil {
		t.Fatal("no fatal error")
	}
}

func TestMenuWithoutCyclerIsAVisibleError(t *testing.T) {
	g := newRig(t)
	g.mover.exploreFn = blockExplore
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("working", func() bool { return g.w.count("explore") > 0 })
	g.w.set(func(w *world) { w.state = d2.StateMenu })
	g.waitFor("error", func() bool { return g.r.Status().State == goals.RunnerError })
	g.w.set(func(w *world) { w.state = d2.StateInGame })
	g.waitFor("recovers by itself", func() bool { return g.r.Status().State == goals.RunnerRunning })
	if errs := g.events(goals.EvError); len(errs) == 0 || errs[0].Fatal {
		t.Fatalf("errors %+v", errs)
	}
}

type fakeCycler struct{ w *world }

func (c fakeCycler) EnterGame(ctx context.Context, d string) error {
	c.w.call("enter %s", d)
	c.w.set(func(w *world) { w.state, w.area = d2.StateInGame, 1 })
	return nil
}
func (c fakeCycler) NewGame(ctx context.Context, d string) error {
	c.w.call("newgame %s", d)
	c.w.set(func(w *world) { w.state = d2.StateMenu })
	select {
	case <-time.After(5 * time.Millisecond):
	case <-ctx.Done():
		return ctx.Err()
	}
	c.w.set(func(w *world) { w.state, w.area = d2.StateInGame, 1; w.mons = nil })
	return nil
}

func TestChickenExitReentersThroughCycler(t *testing.T) {
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { d.Cycler = fakeCycler{} })
	g.r.d.Cycler = fakeCycler{g.w}
	g.mover.exploreFn = blockExplore
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("working", func() bool { return g.w.count("explore") > 0 })
	g.w.set(func(w *world) { w.state = d2.StateMenu })
	g.waitFor("re-entered", func() bool { return g.w.count("enter") > 0 })
	g.waitFor("working again", func() bool { return g.w.count("explore") > 1 })
}

// slowCycler stays at the menus longer than the menu grace, as a loaded box does, and honours ctx like the real flow.
type slowCycler struct {
	w    *world
	wait time.Duration
}

func (c slowCycler) EnterGame(ctx context.Context, d string) error {
	select {
	case <-time.After(c.wait):
	case <-ctx.Done():
		return ctx.Err()
	}
	c.w.call("enter %s", d)
	c.w.set(func(w *world) { w.state, w.area = d2.StateInGame, 1 })
	return nil
}
func (c slowCycler) NewGame(ctx context.Context, d string) error { return c.EnterGame(ctx, d) }

func TestSlowMenuReentryIsNotCancelledByTheGuard(t *testing.T) {
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { c.MenuGraceSec = 1 })
	g.r.d.Cycler = slowCycler{g.w, 2500 * time.Millisecond}
	g.mover.exploreFn = blockExplore
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("working", func() bool { return g.w.count("explore") > 0 })
	g.w.set(func(w *world) { w.state = d2.StateMenu })
	g.waitFor("re-entered in one go", func() bool { return g.w.count("enter") > 0 })
}

func TestClearedAreaAsksForANewGame(t *testing.T) {
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) {})
	g.r.d.Cycler = fakeCycler{g.w}
	g.w.set(func(w *world) { w.area = 2 })
	n := 0
	g.mover.exploreFn = func(ctx context.Context, done func() bool) error {
		n++
		if n > 1 {
			return blockExplore(ctx, done)
		}
		return nil // nothing left to explore
	}
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("fresh game", func() bool { return g.w.count("newgame") > 0 })
	g.waitFor("farming again", func() bool { return g.w.count("explore") > 1 })
}

func TestClearedAreaWithoutCyclerWaitsVisibly(t *testing.T) {
	g := newRig(t)
	g.w.set(func(w *world) { w.area = 2 })
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("error state", func() bool { return g.r.Status().State == goals.RunnerError })
	if !strings.Contains(g.r.Status().Message, "fresh game") {
		t.Fatalf("message %q", g.r.Status().Message)
	}
	// a new game (menu, then in game) clears it
	g.w.set(func(w *world) { w.state = d2.StateMenu })
	g.waitFor("menu seen", func() bool { g.r.obMu.Lock(); defer g.r.obMu.Unlock(); return g.r.ob.state == d2.StateMenu })
	g.w.set(func(w *world) { w.state, w.area = d2.StateInGame, 2 })
	g.waitFor("works again", func() bool { return g.w.count("explore") >= 2 })
}

func TestReachLevelUsesThePlanner(t *testing.T) {
	g := newRig(t)
	g.mover.exploreFn = blockExplore
	g.run()
	defer g.finish()
	g.r.Start(goals.Goal{ID: "r", Type: goals.ReachLevel, Params: goals.Params{Level: 15}})
	g.waitFor("level 1 -> Blood Moor", func() bool { return g.r.Status().Character.Area == "Blood Moor" && g.w.count("explore") > 0 })
	g.w.set(func(w *world) { w.level = 5 })
	// level 5 -> Cold Plains/Burial/Cave; the last eligible window wins (Cave 1 is not reachable by the fake graph from 2 except via 3)
	g.waitFor("re-planned", func() bool {
		a := g.r.Status().Character.AreaID
		return a == 3 || a == 17 || a == 9
	})
}

func TestReachLevelMarginIsAHandicap(t *testing.T) {
	pick := func(margin int) int {
		p, _ := LoadPlan("")
		s, _ := p.Pick("normal", 5, PickOptions{Margin: margin, Acts: map[int]bool{1: true}})
		return s.Area
	}
	if pick(0) == 2 || pick(4) != 2 {
		t.Fatalf("margin 0 -> %d, margin 4 -> %d", pick(0), pick(4))
	}
}

func TestConfigureAppliesKnownSettingsOnly(t *testing.T) {
	g := newRig(t)
	g.r.Configure(goals.Settings{ChickenLifePct: 25, PotionLifePct: 60, Difficulty: "auto"})
	g.r.applySettings() // no combat knobs exist yet: must not fail or touch anything
	if g.r.setDirty {
		t.Fatal("settings not consumed")
	}
}

func TestTwoInstancesSideBySide(t *testing.T) {
	a, b := newRig(t), newRig(t)
	for _, g := range []*rig{a, b} {
		g.mover.exploreFn = blockExplore
	}
	a.run()
	b.run()
	defer a.finish()
	defer b.finish()
	a.r.Start(farm("Blood Moor"))
	b.r.Start(farm("Cold Plains"))
	a.waitFor("a at Blood Moor", func() bool { return a.r.Status().Character.AreaID == 2 && a.w.count("explore") > 0 })
	b.waitFor("b at Cold Plains", func() bool { return b.r.Status().Character.AreaID == 3 && b.w.count("explore") > 0 })
	if a.w.count("exit 3") != 0 || b.w.count("exit 4") != 0 {
		t.Fatal("instances influenced each other")
	}
}

func TestEventChannelDropsOldestInsteadOfBlocking(t *testing.T) {
	g := newRig(t)
	done := make(chan struct{})
	go func() {
		for i := 0; i < 5000; i++ {
			g.r.emit(goals.Event{Type: goals.EvInfo, Level: i})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("emit blocked")
	}
	if len(g.r.Events()) != cap(g.r.Events()) {
		t.Fatalf("channel holds %d", len(g.r.Events()))
	}
	first := <-g.r.Events()
	if first.Level < 5000-2048 {
		t.Fatalf("oldest was not dropped: %d", first.Level)
	}
}

func TestStatusIsFast(t *testing.T) {
	g := newRig(t)
	g.r.Start(farm("Blood Moor"))
	start := time.Now()
	for i := 0; i < 1000; i++ {
		g.r.Status()
		g.r.Pause()
		g.r.Resume()
	}
	if time.Since(start) > 3*time.Second { // catches a call that waits on the loop; wall-clock tight bounds flake under load
		t.Fatal("control methods are slow")
	}
}

// blockExplore behaves like the real Mover.Explore: it runs until the context ends or done() says there is something to fight.
func blockExplore(ctx context.Context, done func() bool) error {
	for ctx.Err() == nil {
		if done() {
			return nil
		}
		time.Sleep(time.Millisecond)
	}
	return ctx.Err()
}

func TestPortalOpenedByFighterStartsATrip(t *testing.T) {
	g := newRig(t)
	g.fight.clearErr = ErrPortalOpened
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.area = 2 })
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("in field", func() bool { return g.w.count("explore") > 0 })
	g.w.addMonster(7, 10, 5, 0)
	g.waitFor("emergency trip and return", func() bool { return g.w.count("chores emergency") > 0 && g.w.count("back 2") > 0 })
}

func TestFutileClearDoesNotBlockExploration(t *testing.T) {
	g := newRig(t)
	g.fight.clearNothing = true
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.area = 2 })
	g.w.addMonster(7, 10, 5, 0) // near, but the fighter cannot get at it
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("clear tried", func() bool { return g.w.count("clear") > 0 })
	g.waitFor("explores anyway", func() bool {
		g.w.set(func(w *world) { w.x += 50 }) // the explorer walks on
		return g.w.count("explore") > 0
	})
}

func TestDeathOverlayNeedsEscAndRespawnIsVerified(t *testing.T) {
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { tu.Bot.CorpseRetries = 0 })
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.area = 2 })
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("in field", func() bool { return g.w.count("explore") > 0 })
	if g.w.count("key esc") != 0 {
		t.Fatal("Esc pressed while alive")
	}
	g.w.set(func(w *world) { w.life, w.mode = 0, combat.ModeDying }) // Esc does nothing while dying: never pressed
	g.waitFor("death seen", func() bool { return len(g.events(goals.EvDeath)) == 1 })
	time.Sleep(20 * time.Millisecond)
	if g.w.count("key esc") != 0 {
		t.Fatal("Esc pressed while the character was still dying")
	}
	die(g)
	g.waitFor("Esc pressed on the overlay", func() bool { return g.w.count("key esc") >= 2 })
	if g.r.State() != StateRecovering {
		t.Fatalf("state %s", g.r.State())
	}
	respawn(g) // the game respawns after Esc
	g.waitFor("recovery runs after the respawn", func() bool { return g.w.count("chores") > 0 })
	n := g.w.count("key esc")
	time.Sleep(10 * time.Millisecond)
	if g.w.count("key esc") != n {
		t.Fatal("kept pressing Esc after the respawn")
	}
}

// The recycler is asked once between goals, in town, before the new goal acts; a restart is reported and the gone
// latch of the old process is dropped.
func TestRecycleAskedBetweenGoalsInTown(t *testing.T) {
	var mu sync.Mutex
	calls, did := 0, false
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) {
		d.Recycle = func(ctx context.Context) (bool, string, error) {
			mu.Lock()
			defer mu.Unlock()
			calls++
			if calls == 1 {
				did = true
				return true, "test: 1 >= 1", nil
			}
			return false, "", nil
		}
	})
	g.w.set(func(w *world) { w.area = 1 })
	g.run()
	defer g.finish()
	if err := g.r.Start(goals.Goal{Type: goals.Idle}); err != nil {
		t.Fatal(err)
	}
	g.waitFor("recycle asked twice (restart, then re-check)", func() bool { mu.Lock(); defer mu.Unlock(); return calls >= 2 })
	mu.Lock()
	defer mu.Unlock()
	if !did {
		t.Fatal("no recycle")
	}
}

func TestRecycleNotAskedInTheField(t *testing.T) {
	calls := 0
	var mu sync.Mutex
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) {
		d.Recycle = func(ctx context.Context) (bool, string, error) {
			mu.Lock()
			calls++
			mu.Unlock()
			return false, "", nil
		}
	})
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.level = 3; w.area = 3 })
	g.run()
	defer g.finish()
	g.r.Start(farm("Cold Plains"))
	time.Sleep(150 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if calls != 0 {
		t.Fatalf("recycle asked %d times in the field", calls)
	}
}

// A leveler pause shows its reason, holds the bot, and ends by itself once no points are pending.
func TestLevelerPauseShowsReasonAndResumesWhenSatisfied(t *testing.T) {
	g := newRig(t)
	g.w.set(func(w *world) { w.area = 1; w.skillPts = 3 }) // points the leveler could not place
	g.run()
	defer g.finish()
	if err := g.r.Start(goals.Goal{Type: goals.Idle}); err != nil {
		t.Fatal(err)
	}
	g.r.LevelerPause("no build for Paladin")
	h := g.r.Health()
	if !h.SelfPaused || !strings.Contains(g.r.Status().Message, "Leveler: no build") {
		t.Fatalf("not paused with a reason: %+v / %q", h, g.r.Status().Message)
	}
	time.Sleep(100 * time.Millisecond)
	if !g.r.Health().SelfPaused {
		t.Fatal("must stay paused while points are pending")
	}
	g.w.set(func(w *world) { w.skillPts = 0 })
	g.waitFor("auto resume", func() bool { return !g.r.Health().SelfPaused })
	if g.r.Status().Message != "" {
		t.Fatalf("message not cleared: %q", g.r.Status().Message)
	}
}

// A recycler that is not due leaves the displayed state alone.
func TestRecycleNotDueKeepsTheState(t *testing.T) {
	asked := 0
	var mu sync.Mutex
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) {
		d.RecycleDue = func() (bool, string) { return false, "" }
		d.Recycle = func(ctx context.Context) (bool, string, error) {
			mu.Lock()
			asked++
			mu.Unlock()
			return true, "x", nil
		}
	})
	g.w.set(func(w *world) { w.area = 1 })
	g.run()
	defer g.finish()
	g.r.Start(goals.Goal{Type: goals.Idle})
	time.Sleep(150 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if asked != 0 || strings.Contains(g.r.Status().Activity, "Restarting") {
		t.Fatalf("asked %d, activity %q", asked, g.r.Status().Activity)
	}
}

func TestBeatStampsHeartbeat(t *testing.T) {
	var r Runner
	if !r.LastStep().IsZero() {
		t.Fatal("heartbeat must start zero")
	}
	r.Beat()
	if time.Since(r.LastStep()) > time.Minute {
		t.Fatal("Beat did not stamp")
	}
}

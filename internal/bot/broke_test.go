package bot

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/boss"
	"d2lod-bot/internal/task"
)

// A broke character on a campaign pauses the quest step, farms, and resumes the quest once the gold target is met.
func TestRecoveryPausesQuestAndResumes(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 3)
	buf := &lockedBuf{}
	g.e.Log = slog.New(slog.NewTextHandler(buf, nil))
	g.e.Tune.Resource.RecoveryMaxMin = 600
	g.mover.exploreFn = blockExplore
	g.w.gold(20)
	g.w.belt()
	g.run()
	defer g.finish()
	if err := g.r.Start(campaign()); err != nil {
		t.Fatal(err)
	}
	g.waitFor("recovery start", func() bool { return strings.Contains(buf.String(), "recovery: start (gold 20 < kit 500)") })
	g.waitFor("farming", func() bool { return g.w.count("explore") > 0 })
	if g.w.count("talk Akara") > 0 {
		t.Fatalf("quest step ran while broke: %s", g.w.log())
	}
	g.w.set(func(w *world) { w.gold0 = 1000 })
	g.waitFor("recovery end", func() bool {
		return strings.Contains(buf.String(), "recovery: end (gold 1000, reason target 900 reached")
	})
	// logged right after the end line, so wait for it instead of reading in between
	g.waitFor("gold line", func() bool { return strings.Contains(buf.String(), "recovery: gold +980 in") })
}

func TestRecoveryOffKeepsOldBehaviour(t *testing.T) {
	g := campaignRig(t, &fakeQuests{}, 3)
	buf := &lockedBuf{}
	g.e.Log = slog.New(slog.NewTextHandler(buf, nil))
	g.e.Tune.Resource.RecoveryOn = 0
	g.mover.exploreFn = func(ctx context.Context, done func() bool) error { return nil }
	g.w.gold(20)
	g.w.belt()
	g.run()
	defer g.finish()
	g.r.Start(campaign())
	g.waitFor("moves", func() bool { return g.w.count("exit") > 0 })
	if strings.Contains(buf.String(), "recovery:") {
		t.Fatalf("recovery ran while off: %s", buf.String())
	}
}

// brWorld is a world that passes the Blood Raven gate at the given level (melee floor 8, caster floor 9).
func brWorld(level int) task.World {
	return task.World{Level: level, Gold: 400, Potions: 5, HasTP: true, ManaPotions: 3, Caster: true, BRHigh: true}
}

// finish an attempt the way KillBoss does: the character failed and is at level lvl.
func (g *rig) brFail(lvl int) { g.r.brEnd(ErrPortalOpened, lvl, 300) }

func TestBloodRavenAttemptBudget(t *testing.T) {
	g := newRig(t)
	buf := &lockedBuf{}
	g.e.Log = slog.New(slog.NewTextHandler(buf, nil))
	g.e.Tune.Resource.BRLevelGap = 2
	lvl := 9
	for i := 0; i < 3; i++ {
		if err := g.r.brAttempt(brWorld(lvl)); err != nil {
			t.Fatalf("attempt %d refused: %v", i+1, err)
		}
		g.brFail(lvl)
		lvl += 2 // +1 and +2 waits are both met
	}
	if !strings.Contains(buf.String(), "blood raven: attempt 3/3 start") {
		t.Fatal(buf.String())
	}
	if err := g.r.brAttempt(brWorld(lvl - 1)); err == nil {
		t.Fatal("4th attempt within the level gap must wait")
	}
	if err := g.r.brAttempt(brWorld(lvl)); err != nil {
		t.Fatalf("clvl+2 renews the budget: %v", err)
	}
	g.e.Tune.Resource.BRAttempts = 0
	for i := 0; i < 10; i++ {
		lvl += 2
		if err := g.r.brAttempt(brWorld(lvl)); err != nil {
			t.Fatal("0 = unlimited")
		}
		g.brFail(lvl)
	}
	_ = boss.NameBloodRaven
	_ = time.Second
}

func TestBloodRavenKitGate(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*task.World)
		line string
	}{
		{"short of potions", func(w *task.World) { w.Potions = 3 }, "short(3 healing potions < 4)"},
		{"gold", func(w *task.World) { w.Gold = 100 }, "short(gold 100 < 250)"},
		{"no mana potions", func(w *task.World) { w.ManaPotions = 1 }, "short(1 mana potions < 2 (caster))"},
		{"caster below 9", func(w *task.World) { w.Level = 8 }, ""},
	}
	for _, c := range cases {
		g := newRig(t)
		buf := &lockedBuf{}
		g.e.Log = slog.New(slog.NewTextHandler(buf, nil))
		w := brWorld(9)
		c.mut(&w)
		if err := g.r.brAttempt(w); !errors.Is(err, errAreaGated) {
			t.Fatalf("%s: err %v, want a planning block", c.name, err)
		}
		if c.line != "" && !strings.Contains(buf.String(), "blood raven: gate clvl=") || !strings.Contains(buf.String(), c.line) {
			t.Errorf("%s: log %s", c.name, buf.String())
		}
		if strings.Contains(buf.String(), "attempt 1/3 start") {
			t.Errorf("%s: attempt started", c.name)
		}
	}
	g := newRig(t)
	buf := &lockedBuf{}
	g.e.Log = slog.New(slog.NewTextHandler(buf, nil))
	if err := g.r.brAttempt(brWorld(9)); err != nil || !strings.Contains(buf.String(), "gate clvl=9 kit=ok") {
		t.Fatalf("full kit refused: %v %s", err, buf.String())
	}
}

// A failed attempt waits for clvl+1, the second failure for clvl+2; a same-level retry never starts, and the wait feeds
// the planner through World.BRWaitLevel.
func TestBloodRavenFailureWaitsForLevels(t *testing.T) {
	g := newRig(t)
	buf := &lockedBuf{}
	g.e.Log = slog.New(slog.NewTextHandler(buf, nil))
	if err := g.r.brAttempt(brWorld(9)); err != nil {
		t.Fatal(err)
	}
	g.brFail(9)
	if g.r.loop.br.waitLvl != 10 {
		t.Fatalf("wait level %d, want 10", g.r.loop.br.waitLvl)
	}
	if err := g.r.brAttempt(brWorld(9)); err == nil {
		t.Fatal("same-level retry started")
	}
	if err := g.r.brAttempt(brWorld(10)); err != nil {
		t.Fatalf("clvl+1 refused: %v", err)
	}
	g.brFail(10)
	if g.r.loop.br.waitLvl != 12 {
		t.Fatalf("wait level %d, want 12 after the second failure", g.r.loop.br.waitLvl)
	}
	if err := g.r.brAttempt(brWorld(11)); err == nil {
		t.Fatal("clvl+1 after the second failure started")
	}
	if !strings.Contains(buf.String(), "blood raven: attempt 2/3 end(retreat (town portal))") || !strings.Contains(buf.String(), "goldBefore=400 goldAfter=300") {
		t.Fatal(buf.String())
	}
	// a cancelled run is not a failure
	if err := g.r.brAttempt(brWorld(12)); err != nil {
		t.Fatal(err)
	}
	n := g.r.loop.br.attempts
	g.r.brEnd(context.Canceled, 12, 300)
	if g.r.loop.br.attempts != n-1 {
		t.Fatalf("cancel did not refund the attempt")
	}
	if g.r.loop.br.waitLvl != 12 {
		t.Fatalf("cancel moved the wait level to %d", g.r.loop.br.waitLvl)
	}
}

// While recovering with an empty purse a town visit buys healing potions only up to the floor (the knob), otherwise the
// normal belt plan: G1 attempt 10 spent its last 60 gold on potions.
func TestRecoveryPotionFloor(t *testing.T) {
	g := newRig(t)
	g.w.set(func(w *world) { w.gold0 = 40 })
	s, _ := g.w.snapshot()
	if f := g.r.potionFloor(s.Player); f != 0 {
		t.Fatalf("no floor outside recovery: %d", f)
	}
	g.r.loop.broke.on = true
	if f := g.r.potionFloor(s.Player); f != g.e.Tune.Resource.RecoveryPotionFloor || f == 0 {
		t.Fatalf("floor %d", f)
	}
	g.e.Tune.Resource.RecoveryPotionFloor = 0
	if f := g.r.potionFloor(s.Player); f != 0 {
		t.Fatalf("knob 0 must leave the normal plan: %d", f)
	}
	g.e.Tune.Resource.RecoveryPotionFloor = 2
	g.w.set(func(w *world) { w.gold0 = 800 })
	s, _ = g.w.snapshot()
	if f := g.r.potionFloor(s.Player); f != 0 {
		t.Fatalf("gold covers the kit, no floor: %d", f)
	}
}

// Without a merc backend the claim step logs and finishes: nothing blocks the campaign.
func TestClaimFreeMercStubSkips(t *testing.T) {
	g := newRig(t)
	buf := &lockedBuf{}
	g.e.Log = slog.New(slog.NewTextHandler(buf, nil))
	if err := (taskActs{r: g.r}).ClaimFreeMerc(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "merc: claim step skipped (no backend)") {
		t.Fatal(buf.String())
	}
}

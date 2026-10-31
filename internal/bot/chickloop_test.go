package bot

import (
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/env"
)

// Found live 2026-10-29 (i4): re-entering in town at 25% life, the trip cooldown blocked the free heal, the character
// walked out and chickened again.
func TestWoundedInTownHealsDespiteTripCooldown(t *testing.T) {
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) {})
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.level = 3; w.life = 25 })
	g.r.setLastTrip(g.e.Clock.Now())
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("heal before leaving", func() bool { return g.w.count("chores heal") > 0 })
	g.waitFor("in field", func() bool { return g.w.count("explore") > 0 })
	log := g.w.log()
	if strings.Index(log, "chores heal") > strings.Index(log, "explore") {
		t.Fatalf("left town before healing: %s", log)
	}
}

func TestWoundedInFieldWithoutRouteTownWaitsAndRetries(t *testing.T) {
	g := newRig(t)
	g.mover.exploreFn = blockExplore
	g.town.toTownErr = errBoom
	g.w.set(func(w *world) { w.level = 3; w.area = 4; w.life = 20 })
	g.w.belt()
	g.run()
	defer g.finish()
	g.r.Start(farm("Stony Field"))
	g.waitFor("waiting with a reason", func() bool {
		return strings.Contains(g.r.Status().Activity, "way to town failed")
	})
	// the wait is not a pause: after lowLifeRetry the gate clears its count and tries the way to town again
	before := g.w.count("totown")
	g.waitFor("retry after the wait", func() bool { return g.w.count("totown") > before })
	if n := g.w.count("explore"); n != 0 {
		t.Fatalf("explored %d times below the resume line", n)
	}
}

func TestFailsafePauseEndsByItself(t *testing.T) {
	g := newRig(t)
	g.r.Start(farm("Blood Moor"))
	g.r.selfPause("test failsafe")
	g.r.autoResumeIn(time.Minute)
	if p, _ := pausedWith(g.r); !p {
		t.Fatal("not paused")
	}
	if g.r.autoResumed() {
		t.Fatal("resumed early")
	}
	g.e.Clock.Sleep(2 * time.Minute)
	if !g.r.autoResumed() {
		t.Fatal("did not resume")
	}
	if p, _ := pausedWith(g.r); p {
		t.Fatal("still paused")
	}
}

func TestChickenReentriesEscalateToAPause(t *testing.T) {
	g := newRig(t)
	g.r.Start(farm("Blood Moor"))
	for i := 0; i < reentryLimit; i++ {
		g.r.noteReentry()
		g.e.Clock.Sleep(time.Second)
	}
	if p, m := pausedWith(g.r); !p || !strings.Contains(m, "re-entered") {
		t.Fatalf("not paused: %v %q", p, m)
	}
}

func pausedWith(r *Runner) (bool, string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.paused, r.message
}

// Found offline 2026-10-29 from the Soaky soak: a poor clvl 4 sorceress with no potion and no scroll fought on in the
// Den at 27/73 life for 25 minutes because the policy only saw Blood Moor as "near town". The way home from the Den
// is a walk (town.walkToTown), so the free heal trip must be due, and its plan must include the heal.
func TestPoorHurtCharacterInTheDenIsDueAFreeHealTrip(t *testing.T) {
	g := newRig(t)
	g.w.set(func(w *world) { w.level = 4; w.area = 8; w.life = 27; w.maxL = 73; w.gold0 = 0 })
	g.w.belt()
	s, _ := g.w.snapshot()
	plan, _, ok := g.r.choresDue(s)
	if !ok || !plan.Heal || !plan.Potions {
		t.Fatalf("want a heal trip, got ok=%v plan=%+v", ok, plan)
	}
	// healthy and poor: nothing to do in town, no trip
	g.w.set(func(w *world) { w.life = 73 })
	s, _ = g.w.snapshot()
	if _, _, ok := g.r.choresDue(s); ok {
		t.Fatal("healthy poor character must not make a trip")
	}
}

func TestEverySelfPauseAutoResumesWithBackoff(t *testing.T) {
	g := newRig(t)
	g.r.Start(farm("Blood Moor"))
	for i, want := range []time.Duration{5 * time.Minute, 10 * time.Minute, 20 * time.Minute} {
		g.r.selfPause("test")
		g.e.Clock.Sleep(want - time.Second)
		if g.r.autoResumed() {
			t.Fatalf("pause %d resumed before %v", i, want)
		}
		g.e.Clock.Sleep(2 * time.Second)
		if !g.r.autoResumed() {
			t.Fatalf("pause %d did not resume after %v", i, want)
		}
	}
}

package bot

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/env"
)

// TownTripTrigger says a restock trip with 45 gold is futile (TripMinGold 60): no trip, where the plain potion
// threshold alone would have gone.
func TestFutileRestockTripIsSilenced(t *testing.T) {
	g := newRig(t)
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.level = 3; w.area = 2 })
	g.w.gold(45)
	g.w.belt()
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("in field", func() bool { return g.w.count("explore") > 0 })
	time.Sleep(60 * time.Millisecond)
	if n := g.w.count("totown"); n != 0 {
		t.Fatalf("futile trip was made: %s", g.w.log())
	}
}

// With enough gold the resource trigger sends the bot home and names its reason.
func TestResourceTriggerStartsATrip(t *testing.T) {
	g := newRig(t)
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.level = 3 })
	g.w.gold(300)
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("in field", func() bool { return g.w.count("explore") > 0 })
	g.w.belt()
	g.waitFor("trip", func() bool { return g.w.count("totown") > 0 })
}

func TestPrepareRunsOnceBeforeTheGoal(t *testing.T) {
	var calls int
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) {
		d.Prepare = func(ctx context.Context) error { calls++; return nil }
	})
	g.mover.exploreFn = blockExplore
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("in field", func() bool { return g.w.count("explore") > 0 })
	if calls != 1 {
		t.Fatalf("Prepare ran %d times", calls)
	}
}

func TestPrepareFailureIsRetriedThenSkipped(t *testing.T) {
	var calls int
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) {
		d.Prepare = func(ctx context.Context) error { calls++; return errBoom }
	})
	g.mover.exploreFn = blockExplore
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("in field", func() bool { return g.w.count("explore") > 0 })
	if calls != 3 {
		t.Fatalf("Prepare ran %d times, want 3", calls)
	}
}

// Stop means "finish in town": the runner is only settled once the character stands there.
func TestStopSettlesInTown(t *testing.T) {
	g := newRig(t)
	g.mover.exploreFn = blockExplore
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("in field", func() bool { return g.w.count("explore") > 0 })
	g.r.Stop()
	g.waitFor("settled", func() bool { return g.r.Settled() })
	if s, _ := g.w.snapshot(); s.Player.Area != 1 {
		t.Fatalf("stopped in area %d, not in town", s.Player.Area)
	}
}

func TestPackFullReasonGoesQuietAfterAnUnfixedTrip(t *testing.T) {
	g := newRig(t)
	if g.r.packFullQuiet() {
		t.Fatal("quiet before any trip")
	}
	g.r.quietPackFull(time.Hour)
	if !g.r.packFullQuiet() {
		t.Fatal("not quiet after an unfixed trip")
	}
	g.r.quietPackFull(-time.Second)
	if g.r.packFullQuiet() {
		t.Fatal("still quiet after the time ran out")
	}
}

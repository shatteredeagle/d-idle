package bot

import (
	"testing"
	"time"
)

// WP4 (soak 2026-10-29 08:27): after a self pause with a monster the fighter cannot kill in reach, the status used to toggle
// paused/clearing every tick (the paused branch fell through to the goal). It must stay paused for the whole delay and then
// resume by itself.
func TestSelfPauseWithHostileNearStaysPaused(t *testing.T) {
	g := newRig(t)
	g.fight.clearNothing = true
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.area = 2 })
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("clearing", func() bool { return g.r.State() == StateClearing })
	g.w.addMonster(7, 10, 5, 0)
	g.r.selfPause("Failsafe: heal trip failed 3 times")
	g.waitFor("paused", func() bool { return g.r.State() == StatePaused })

	start := g.clock.Now()
	flaps := 0
	for g.clock.Now().Sub(start) < 2*time.Minute {
		if g.r.State() != StatePaused {
			flaps++
		}
		g.clock.Sleep(2500 * time.Millisecond)
	}
	if flaps != 0 {
		t.Fatalf("state left paused %d times in 2 minutes of fake time", flaps)
	}
	g.clock.Sleep(4 * time.Minute) // past the 5 minute selfPauseDelay
	g.waitFor("auto resume", func() bool { return !g.r.Health().SelfPaused })
}

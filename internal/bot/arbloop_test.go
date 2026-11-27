package bot

import (
	"context"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

// Featuresoak arbiter3, necromancer seed 201 (2026-11-26): the survive hook (families 3) fought where it stood during
// the trip's leg back to the field and chickened; the game re-entered in town at 9/45 life with the trip still on its
// last leg, which walked out again, chickened on arrival and came back, 169 times in a virtual hour. The leg must not
// leave town under the resume line: the hero heals first. Two builds, arbiter off, families 3 and families 255.
func TestTripLegBackDoesNotLeaveTownUnderTheResumeLine(t *testing.T) {
	arms := []struct {
		name      string
		ctl, mask int
	}{{"chain only", 0, 0}, {"families 3", 1, 3}, {"families 255", 1, 255}}
	for _, class := range []d2.Class{d2.Necromancer, d2.Paladin} {
		for _, a := range arms {
			g := newRig(t)
			g.w.set(func(w *world) { w.class, w.level, w.area, w.life, w.maxL = class, 12, 1, 20, 100 })
			g.w.belt()
			g.r.e.Tune.Bot.ArbiterShadow, g.r.e.Tune.Bot.ArbiterControl, g.r.e.Tune.Bot.ArbiterFamilies = a.ctl, a.ctl, a.mask
			g.r.poll()
			g.r.loop.trip = &trip{returnTo: 2, stage: 3, emergency: true}
			for i := 0; i < 6 && g.w.count("chores") == 0; i++ {
				g.r.runGoalShadowed(context.Background(), g.r.latest(), levelGoalControl())
				g.clock.Sleep(time.Second)
				g.r.poll()
			}
			log := g.w.log()
			heal, back := strings.Index(log, "chores"), strings.Index(log, "back ")
			if heal < 0 || (back >= 0 && back < heal) {
				t.Fatalf("class %d, %s: left town under the resume line before healing: %s", class, a.name, log)
			}
		}
	}
}

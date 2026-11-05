package bot

import (
	"context"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
)

// Rung 3 (leave and re-enter) is bounded per progress window (Gate 2 g2c seed 13: an item stuck on the cursor made the
// ladder cycle about 1,100 times in 2 h); the pause rung then backs off. escalate is driven directly with a cycler that
// always succeeds and never fixes anything.

func ladderRig(t *testing.T, class d2.Class, rung3 int) *rig {
	t.Helper()
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) {
		c.TownTimeoutSec = 20
		tu.Bot.FailsafeRung3Max = rung3
		tu.Bot.Rung3WindowMax = 0 // the burst cap has its own tests (nokill_test.go)
	})
	g.r.d.Cycler = fakeCycler{g.w}
	g.w.set(func(w *world) { w.state = d2.StateInGame; w.area = 1; w.class = class })
	g.r.Start(farm("Blood Moor"))
	return g
}

// climb runs n full ladder rounds (entries until the pause rung, then a resume) and returns the pauses seen.
func (g *rig) climb(n int) (pauses int) {
	for i := 0; i < n*8 && pauses < n; i++ {
		g.r.escalate(context.Background(), g.r.control(), "stuck", errStuck)
		if g.r.isSelfPaused() {
			pauses++
			g.resume()
		}
	}
	return pauses
}

func (g *rig) resume() {
	g.r.mu.Lock()
	g.r.selfStop, g.r.paused, g.r.message = false, false, ""
	g.r.mu.Unlock()
}

func (g *rig) realProgress(n uint64) {
	g.r.obMu.Lock()
	g.r.ob.real += n
	g.r.obMu.Unlock()
}

func TestLadderRung3Cap(t *testing.T) {
	for _, cl := range []d2.Class{d2.Sorceress, d2.Barbarian} {
		t.Run(cl.String(), func(t *testing.T) {
			g := ladderRig(t, cl, 3)
			if g.climb(6) != 6 {
				t.Fatal("did not pause 6 times")
			}
			// the 4th and later rounds skip rung 3 and pause
			if n := strings.Count(g.w.log(), "newgame"); n != 3 {
				t.Fatalf("rung 3 ran %d times after resume without progress, want 3", n)
			}
		})
	}
}

func TestLadderRung3ProgressResets(t *testing.T) {
	for _, cl := range []d2.Class{d2.Sorceress, d2.Barbarian} {
		t.Run(cl.String(), func(t *testing.T) {
			g := ladderRig(t, cl, 2)
			g.climb(4)
			g.realProgress(realResetMin)
			g.climb(4)
			if n := strings.Count(g.w.log(), "newgame"); n != 4 {
				t.Fatalf("rung 3 ran %d times, want 2 per window = 4", n)
			}
		})
	}
}

func TestLadderRung3CapOff(t *testing.T) {
	g := ladderRig(t, d2.Sorceress, 0)
	g.climb(10)
	if n := strings.Count(g.w.log(), "newgame"); n < 10 {
		t.Fatalf("only %d re-entries with the cap off", n)
	}
}

func TestPauseBackoffDoublesToCap(t *testing.T) {
	g := ladderRig(t, d2.Barbarian, 1)
	g.e.Tune.Bot.FailsafePauseMaxSec = 1200
	var got []time.Duration
	for i := 0; i < 5; i++ {
		got = append(got, g.r.pauseBackoff())
	}
	want := []time.Duration{5, 10, 20, 20, 20}
	for i := range want {
		if got[i] != want[i]*time.Minute {
			t.Fatalf("backoff %v, want %v", got, want)
		}
	}
	g.realProgress(realResetMin)
	g.r.escalate(context.Background(), g.r.control(), "stuck", errStuck)
	if g.r.loop.ladderPauses != 0 {
		t.Fatalf("progress did not reset the pause streak: %d", g.r.loop.ladderPauses)
	}
}

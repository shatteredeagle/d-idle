package bot

import (
	"context"
	"errors"
	"testing"
	"time"
)

// stallRig is a rig with a fresh-game cycler, the ladder at 30 + 30 virtual minutes and the observer's progress at "now".
func stallRig(t *testing.T) *rig {
	g := newRig(t)
	g.r.d.Cycler = fakeCycler{g.w}
	g.r.e.Tune.Bot.StallWarnMin, g.r.e.Tune.Bot.StallExitMin = 30, 30
	g.r.poll()
	g.r.campaignStall(context.Background(), g.r.latest(), true) // arms the window at now
	return g
}

func (g *rig) stallStep(d time.Duration) bool {
	g.clock.Sleep(d)
	return g.r.campaignStall(context.Background(), g.r.latest(), true)
}

func (g *rig) stalled() error {
	g.r.mu.Lock()
	defer g.r.mu.Unlock()
	return g.r.crash
}

func (g *rig) progress() {
	g.r.obMu.Lock()
	g.r.ob.campaignAt = g.r.e.Clock.Now()
	g.r.obMu.Unlock()
}

// Legitimate waits (the 3 min area ban, the longest stretch of the navsoak logs: 19 min) never escalate.
func TestStallLegitimateWaitsQuiet(t *testing.T) {
	g := stallRig(t)
	for _, d := range []time.Duration{3 * time.Minute, 16 * time.Minute, 9 * time.Minute} { // 28 min in all
		if g.stallStep(d) || g.r.loop.stallStage != 0 {
			t.Fatalf("escalated after a short wait")
		}
	}
	if g.stalled() != nil {
		t.Fatal("must not exit")
	}
}

// No progress: WARN plus a fresh game at the first window, exit at the second, progress resets.
func TestStallLadder(t *testing.T) {
	g := stallRig(t)
	g.r.loop.cleared = map[int]bool{2: true}
	if !g.stallStep(31*time.Minute) || g.r.loop.stallStage != 1 {
		t.Fatal("first window: warn and fresh game")
	}
	if len(g.r.loop.cleared) != 0 {
		t.Fatal("the fresh game must forget the per-game memory")
	}
	g.r.obMu.Lock()
	rec := g.r.ob.recycles
	g.r.obMu.Unlock()
	if rec != 1 {
		t.Fatalf("recycles %d, want 1", rec)
	}
	if g.stallStep(20*time.Minute) || g.stalled() != nil {
		t.Fatal("second window not over yet")
	}
	if !g.stallStep(11 * time.Minute) {
		t.Fatal("second window over: exit")
	}
	err := g.stalled()
	if !errors.Is(err, ErrCampaignStalled) {
		t.Fatalf("crash %v, want ErrCampaignStalled", err)
	}
}

// Progress between the windows drops back to stage 0 and the full window restarts.
func TestStallProgressResets(t *testing.T) {
	g := stallRig(t)
	g.stallStep(31 * time.Minute)
	g.clock.Sleep(10 * time.Minute)
	g.progress()
	if g.stallStep(time.Minute); g.r.loop.stallStage != 0 {
		t.Fatal("progress must reset the stage")
	}
	if g.stallStep(25 * time.Minute); g.r.loop.stallStage != 0 || g.stalled() != nil {
		t.Fatal("a fresh window after progress")
	}
	if !g.stallStep(10*time.Minute) || g.r.loop.stallStage != 1 {
		t.Fatal("warns again after a full window")
	}
}

// Paused, no goal or knob 0: nothing counts.
func TestStallOffAndPaused(t *testing.T) {
	g := stallRig(t)
	g.clock.Sleep(2 * time.Hour)
	if g.r.campaignStall(context.Background(), g.r.latest(), false) {
		t.Fatal("inactive must not escalate")
	}
	if g.stallStep(29 * time.Minute) {
		t.Fatal("the window restarts when the bot is active again")
	}
	g.r.e.Tune.Bot.StallWarnMin = 0
	if g.stallStep(5*time.Hour) || g.stalled() != nil {
		t.Fatal("0 turns the ladder off")
	}
}

// The exit status is its own code and message, restarted by the supervisor (not exitFatal).
func TestStallErrorIsDistinct(t *testing.T) {
	if ErrCampaignStalled.Error() != "campaign stalled, exiting for a restart" {
		t.Fatal(ErrCampaignStalled)
	}
}

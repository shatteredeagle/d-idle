package bot

import (
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/launch"
)

// within fails the test when f does not return in two seconds (a lock held across a callback shows up as a hang).
func within(t *testing.T, what string, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { f(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("deadlock: %s", what)
	}
}

// blockingHealth is a monitor whose Step waits for release, like a control-socket ping that gets no turn.
type blockingHealth struct{ in, release chan struct{} }

func (h *blockingHealth) Step(time.Time) launch.Event {
	h.in <- struct{}{}
	<-h.release
	return launch.EventNone
}

// TestPollNeverHoldsObMuAcrossOnSnapshot: Deps.OnSnapshot used to run under obMu, so a callback that read the runner
// (LastSnapshot takes obMu) hung the poll and everything queued behind it. docs/locks.md.
func TestPollNeverHoldsObMuAcrossOnSnapshot(t *testing.T) {
	var g *rig
	called := 0
	g = newRig(t, func(_ *Config, d *Deps, _ *env.Tuning) {
		d.OnSnapshot = func(*d2.Snapshot) {
			called++
			g.r.LastSnapshot()
		}
	})
	within(t, "OnSnapshot calling LastSnapshot", func() { g.r.poll() })
	if called == 0 {
		t.Fatal("OnSnapshot was not called")
	}
}

// TestPollNeverHoldsObMuAcrossHealthStep: the d2d monitor pings the control socket inside Step; obMu must stay free
// meanwhile so status handlers (LastSnapshot, Health) answer.
func TestPollNeverHoldsObMuAcrossHealthStep(t *testing.T) {
	bh := &blockingHealth{in: make(chan struct{}, 1), release: make(chan struct{})}
	g := newRig(t, func(_ *Config, d *Deps, _ *env.Tuning) { d.Health = bh })
	polled := make(chan struct{})
	go func() { g.r.poll(); close(polled) }()
	select {
	case <-bh.in:
	case <-time.After(2 * time.Second):
		t.Fatal("Health.Step was not reached")
	}
	within(t, "LastSnapshot while Health.Step is blocked", func() { g.r.LastSnapshot() })
	close(bh.release)
	select {
	case <-polled:
	case <-time.After(2 * time.Second):
		t.Fatal("poll did not finish")
	}
}

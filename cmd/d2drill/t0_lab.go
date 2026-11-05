package main

import (
	"context"
	"time"

	"d2lod-bot/internal/drill"
)

var _ drill.T0Lab = (*lab)(nil)

// StartRun writes the run-start marker of the CRUTCH rule (a ledger marker, not a game write): the T0 drill's setup
// cheats are before it, nothing after it.
func (l *lab) StartRun(_ context.Context, run, reason string) error {
	return l.led.MarkRun(run, "drills:T0", l.inst, reason, time.Now())
}

// WritesAfterStart is "writes after start: N" of the run record.
func (l *lab) WritesAfterStart(run string) (int, error) {
	n, _, err := l.led.WritesAfterStart(run)
	return n, err
}

// VirtualNow is the game time since the lab began: ticks of 40 ms (U: the 25 ticks a second of the lab's Pause).
func (l *lab) VirtualNow() time.Duration {
	t := l.tick()
	if t < 0 {
		return 0
	}
	return time.Duration(t) * 40 * time.Millisecond
}

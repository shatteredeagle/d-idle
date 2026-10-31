// Package telemetry turns a stream of game frames and bot events into
// statistics (XP/hour, kills, drops, ...). It is pure: no I/O, no goroutines,
// time comes from a Clock so tests are deterministic.
package telemetry

import "time"

// Clock supplies the current time.
type Clock interface{ Now() time.Time }

// RealClock is the wall clock.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }

// FakeClock is a manually advanced clock for tests and simulation.
type FakeClock struct{ T time.Time }

func (c *FakeClock) Now() time.Time          { return c.T }
func (c *FakeClock) Advance(d time.Duration) { c.T = c.T.Add(d) }

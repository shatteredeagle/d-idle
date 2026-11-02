// Package env is the per-instance dependency bundle every behavior package (nav, combat, town, loot, bot) receives,
// plus the tuning knobs they read. One Env per running game: nothing here is global, so N bots are N Envs
// (docs/net/design-now.md).
package env

import (
	"d2lod-bot/internal/gclock"
	"log/slog"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/mem"
)

// InstanceID names one running game plus its bot; it appears in logs, paths, events and API routes.
type InstanceID string

// Clock lets tests run behaviors without real sleeping.
type Clock interface {
	Now() time.Time
	Sleep(d time.Duration)
}

// RealClock is the wall clock.
type RealClock struct{}

// Now and Sleep run on game time (internal/gclock): identical to the wall clock unless D2BOT_TIMESCALE is set.
func (RealClock) Now() time.Time        { return gclock.Now() }
func (RealClock) Sleep(d time.Duration) { gclock.Sleep(d) }

// Env is everything a behavior needs. Construct it once per instance and pass it down; never reach for globals.
type Env struct {
	ID    InstanceID
	Log   *slog.Logger // already carries the instance id
	Mem   mem.Source
	Read  d2.StateReader
	World *d2map.Tracker
	In    input.Controller
	Data  *data.DB
	Tune  *Tuning
	Clock Clock
	// Emit publishes a bot event to the goal manager. Never blocks; may drop when the consumer is slow.
	Emit func(goals.Event)
}

// Snapshot is one consistent read of the whole game state for this tick.
func (e *Env) Snapshot() (*d2.Snapshot, error) { return e.Read.Snapshot() }

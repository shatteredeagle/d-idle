// Package gclock is the bot's game clock. When the original game runs under the clock hook with scale k (the game
// advances k game seconds per real second, docs/research/lockstep-p3.md), every wait the bot makes for the game to
// react must shrink by k and every game-time deadline must run k times faster. D2BOT_TIMESCALE=k sets k once at start;
// the default 1 changes nothing: Now returns time.Now, durations pass through unscaled.
//
// Use it for waits on the game (click-to-effect, panel, walking, chore and stall windows). Process management,
// backend RPC deadlines, supervisor and soak timers stay on package time.
package gclock

import (
	"context"
	"os"
	"strconv"
	"time"
)

// EnvVar names the environment variable that carries the scale.
const EnvVar = "D2BOT_TIMESCALE"

// StartNote is the log message d2bot prints once at start when the scale is not 1 ("gclock: timescale 2"), empty at 1.
// tools/baseline-run.sh (hooked lanes) checks the bot log for it, so the text is part of the interface.
func StartNote(k float64) string {
	if k == 1 {
		return ""
	}
	return "gclock: timescale " + strconv.FormatFloat(k, 'g', -1, 64)
}

// Scale bounds; a value outside them (or unparsable) falls back to 1.
const (
	MinScale = 0.25
	MaxScale = 16
)

// ParseScale reads a scale string: empty, unparsable, NaN or out of bounds gives 1.
func ParseScale(s string) float64 {
	k, err := strconv.ParseFloat(s, 64)
	if err != nil || !(k >= MinScale && k <= MaxScale) { // the negated form also rejects NaN
		return 1
	}
	return k
}

// Scaler is one clock with scale K. Tests build their own; the package functions use Default.
type Scaler struct {
	K     float64
	base  time.Time
	now   func() time.Time // nil: time.Now
	sleep func(time.Duration)
}

// New returns a Scaler with scale k (not bounds-checked; use ParseScale for input).
func New(k float64) *Scaler { return &Scaler{K: k, base: time.Now()} }

// Default is read once from D2BOT_TIMESCALE at process start.
var Default = New(ParseScale(os.Getenv(EnvVar)))

func (s *Scaler) real() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

// D converts a game duration into the real duration to wait for it.
func (s *Scaler) D(d time.Duration) time.Duration {
	if s.K == 1 {
		return d
	}
	return time.Duration(float64(d) / s.K)
}

// Now is game time: it advances K times faster than the wall clock from the moment the Scaler was made.
func (s *Scaler) Now() time.Time {
	t := s.real()
	if s.K == 1 {
		return t
	}
	return s.base.Add(time.Duration(float64(t.Sub(s.base)) * s.K))
}

// Since is the game time elapsed since t (a value from Now).
func (s *Scaler) Since(t time.Time) time.Duration { return s.Now().Sub(t) }

// Sleep waits d of game time.
func (s *Scaler) Sleep(d time.Duration) {
	if s.sleep != nil {
		s.sleep(s.D(d))
		return
	}
	time.Sleep(s.D(d))
}

// After is time.After for game time.
func (s *Scaler) After(d time.Duration) <-chan time.Time { return time.After(s.D(d)) }

// NewTimer is time.NewTimer for game time.
func (s *Scaler) NewTimer(d time.Duration) *time.Timer { return time.NewTimer(s.D(d)) }

// WithTimeout is context.WithTimeout for a game-time limit.
func (s *Scaler) WithTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, s.D(d))
}

// WithTimeoutCause is context.WithTimeoutCause for a game-time limit.
func (s *Scaler) WithTimeoutCause(ctx context.Context, d time.Duration, cause error) (context.Context, context.CancelFunc) {
	return context.WithTimeoutCause(ctx, s.D(d), cause)
}

// Clock satisfies env.Clock and input.Clock (Now, Sleep) on the Default scale.
type Clock struct{}

func (Clock) Now() time.Time        { return Now() }
func (Clock) Sleep(d time.Duration) { Sleep(d) }

func K() float64                             { return Default.K }
func D(d time.Duration) time.Duration        { return Default.D(d) }
func Now() time.Time                         { return Default.Now() }
func Since(t time.Time) time.Duration        { return Default.Since(t) }
func Sleep(d time.Duration)                  { Default.Sleep(d) }
func After(d time.Duration) <-chan time.Time { return Default.After(d) }
func NewTimer(d time.Duration) *time.Timer   { return Default.NewTimer(d) }

func WithTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return Default.WithTimeout(ctx, d)
}

func WithTimeoutCause(ctx context.Context, d time.Duration, cause error) (context.Context, context.CancelFunc) {
	return Default.WithTimeoutCause(ctx, d, cause)
}

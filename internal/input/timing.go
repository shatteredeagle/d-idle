package input

import (
	"math/rand"
	"time"
)

// Timing is the one place that decides how fast and how regular input is.
// Every primitive derives its sleeps from it; tests use Instant().
type Timing struct {
	MinPress    time.Duration // shortest mouse/key press (the game runs 25 fps; below 40 ms clicks can be lost)
	PressJitter time.Duration // extra random press time, uniform in [0, PressJitter]
	MoveSettle  time.Duration // pointer move to button down, so the game sees the hover first
	SettleJit   time.Duration
	ActionGap   time.Duration // minimum pause after any action before the next
	GapJitter   time.Duration
	Rand        *rand.Rand // nil: package-level source
}

// DefaultTiming is human-ish: ~60-100 ms presses, 50-90 ms settle, 60-140 ms between actions.
func DefaultTiming() Timing {
	return Timing{
		MinPress: 60 * time.Millisecond, PressJitter: 40 * time.Millisecond,
		MoveSettle: 50 * time.Millisecond, SettleJit: 40 * time.Millisecond,
		ActionGap: 60 * time.Millisecond, GapJitter: 80 * time.Millisecond,
	}
}

// Instant has no delays beyond the 40 ms floor of a press; for fast measurement runs.
func Instant() Timing {
	return Timing{MinPress: 40 * time.Millisecond, MoveSettle: 40 * time.Millisecond}
}

const pressFloor = 40 * time.Millisecond

func (t Timing) jit(base, j time.Duration) time.Duration {
	if j <= 0 {
		return base
	}
	var n int64
	if t.Rand != nil {
		n = t.Rand.Int63n(int64(j) + 1)
	} else {
		n = rand.Int63n(int64(j) + 1)
	}
	return base + time.Duration(n)
}

func (t Timing) Press() time.Duration {
	return max(pressFloor, t.jit(t.MinPress, t.PressJitter))
}
func (t Timing) Settle() time.Duration { return t.jit(t.MoveSettle, t.SettleJit) }
func (t Timing) Gap() time.Duration    { return t.jit(t.ActionGap, t.GapJitter) }

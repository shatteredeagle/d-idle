package d2denv

import (
	"context"
	"sync"
	"time"

	"d2lod-bot/internal/d2dctl"
)

// Beat is the liveness heartbeat of d2d: a counter that grows only while the game really advances, so a d2d that
// still answers "ping" (the channel is pumped from the main loop) but has stopped running the world looks hung to
// launch.Monitor (Heartbeat probe, HangAfter) instead of alive. Ping alone cannot tell, so it is not enough.
//
// Marker: the fork's "clock" verb gives game_ms; on a free-running d2d that is wall-clock driven and must grow in real
// time (S: game_ms is SDL_GetTicks based in the fork, docs/d2d-fork.md section 2). Under --lockstep game_ms is frozen
// between steps by design, so the loop frame counter of "info" is used (U: that it keeps counting while idle, docs say
// the loop runs at normal pace with a frozen clock). A verb that is missing (stock d2d: no clock) falls back to the
// info frame as well. Not verified live in the stuck case: nothing here reproduces a wedged d2d.
type Beat struct {
	C interface {
		Clock(context.Context) (d2dctl.ClockInfo, error)
		Info(context.Context) (d2dctl.Info, error)
	}

	mu     sync.Mutex
	marker [5]int
	seen   bool
	beats  uint64
}

// Next reads the marker once. An error (no answer at all) is returned as it is; an answer that did not advance leaves
// the counter where it was. Safe for the monitor's goroutine.
func (b *Beat) Next() (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	m, err := b.marker1(ctx)
	if err != nil {
		return 0, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.seen || m != b.marker {
		b.beats++
	}
	b.marker, b.seen = m, true
	return b.beats, nil
}

// marker1 reads every progress counter d2d has: game_ms, tick, stepped and frame of "clock" and the frame of "info".
// Any one moving is progress. The first live run flagged a healthy lockstep d2d as hung about 100 s after each start
// while it was stepping (D2DWIRE), so no single counter is trusted to move in every mode.
func (b *Beat) marker1(ctx context.Context) ([5]int, error) {
	var m [5]int
	ci, cerr := b.C.Clock(ctx)
	if cerr == nil {
		m[0], m[1], m[2], m[3] = ci.GameMS, ci.Tick, ci.Stepped, ci.Frame
	}
	in, ierr := b.C.Info(ctx)
	if ierr == nil {
		m[4] = in.Frame
	}
	if cerr != nil && ierr != nil {
		return m, ierr
	}
	return m, nil
}

// Last is the most recent counter set (game_ms, tick, stepped, clock frame, info frame) for logs.
func (b *Beat) Last() [5]int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.marker
}

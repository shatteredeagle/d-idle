package bot

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
)

// watchClock is a vclock with the lockstep Pacer's Watch: the watcher runs inside Sleep, on the sleeping goroutine, when
// the interval is crossed (the shape of d2dinput.Pacer.Watch, docs/d2d-determinism.md section 3.1).
type watchClock struct {
	*vclock
	mu    sync.Mutex
	every time.Duration
	last  time.Time
	fn    func()
	inFn  bool
	calls int
}

func (c *watchClock) Watch(every time.Duration, fn func()) func() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.every, c.fn, c.last = every, fn, c.vclock.Now()
	return func() { c.mu.Lock(); c.fn = nil; c.mu.Unlock() }
}

func (c *watchClock) Sleep(d time.Duration) {
	c.vclock.Sleep(d)
	c.mu.Lock()
	fn := c.fn
	due := fn != nil && !c.inFn && c.vclock.Now().Sub(c.last) >= c.every
	if due {
		c.last, c.inFn = c.vclock.Now(), true
		c.calls++
	}
	c.mu.Unlock()
	if due {
		fn()
		c.mu.Lock()
		c.inFn = false
		c.mu.Unlock()
	}
}

// Under a Watch clock the guard does not sleep on its own goroutine: its checks run inside the action's Sleep and still
// cancel the action (here: the character died), with the same cause as the goroutine guard.
func TestGuardRunsInsideTheActionsSleepUnderAWatchClock(t *testing.T) {
	g := newRig(t)
	wc := &watchClock{vclock: g.clock}
	g.e.Clock = wc
	g.r.e.Clock = wc
	g.w.set(func(w *world) { w.state = d2.StateInGame; w.area = 2; w.life, w.maxL = 40, 40 }) // in game, Blood Moor
	g.r.gen = 1

	var fnOnActionGoroutine bool
	sleeps := 0
	act := func(ctx context.Context) error {
		for ctx.Err() == nil {
			sleeps++
			if sleeps == 3 {
				g.w.set(func(w *world) { w.mode = combat.ModeDead })
			}
			wc.Sleep(time.Duration(g.r.e.Tune.Bot.WatchMs) * time.Millisecond) // one guard look per sleep
			if sleeps > 50 {
				return errors.New("guard never cancelled")
			}
		}
		fnOnActionGoroutine = wc.calls > 0
		return ctx.Err()
	}
	_, cause := g.r.runAction(context.Background(), 1, StateClearing, "test", 0, act)
	if !errors.Is(cause, errDied) {
		t.Fatalf("cause %v, sleeps %d, watch calls %d", cause, sleeps, wc.calls)
	}
	if !fnOnActionGoroutine || wc.calls == 0 {
		t.Fatalf("guard did not run through the watch hook (calls %d)", wc.calls)
	}
}

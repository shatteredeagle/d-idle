package d2dinput

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/d2dctl"
)

// TickDur is one world tick of d2d (25 Hz, V docs/d2d-fork.md section 2).
const TickDur = 40 * time.Millisecond

// Pacer is a clock for a lockstep d2d: Sleep does not wait, it steps the game by the equivalent number of ticks, and
// Now is the game's virtual time, so deadlines and stall timers in nav and MoveTo behave as in real time while the
// loop runs at lockstep speed. It satisfies env.Clock and input.Clock. A game that is not in lockstep mode does not
// move on "step"; use it only with the fork's --lockstep.
type Pacer struct {
	C   *d2dctl.Client
	Ctx context.Context // nil: Background
	// Chunk caps ticks per "step" call (0: 50), so one call stays well inside the client timeout.
	Chunk int
	// Base is the wall time that virtual time zero maps to (zero: the Unix epoch, which keeps logs reproducible).
	Base time.Time

	lastOK atomic.Int64 // wall unix nanos of the last successful step (read without the lock: a step can run for seconds)
	mu     sync.Mutex
	start  int // game_ms that virtual time (less carry) counts from
	carry  int // virtual ms accumulated before the game's clock last restarted
	last   int // last game_ms seen
	cur    int // virtual ms since the first use
	synced bool
	steps  int
	ticks  int
	err    error
	watch  *watcher
}

func (p *Pacer) ctx() context.Context {
	if p.Ctx != nil {
		return p.Ctx
	}
	return context.Background()
}

func (p *Pacer) sync() {
	if p.synced {
		return
	}
	p.synced = true
	if ci, err := p.C.Clock(p.ctx()); err == nil {
		p.start, p.last = ci.GameMS, ci.GameMS
	} else if p.err == nil {
		p.err = err
	}
}

// Now is Base plus the virtual time that passed since the Pacer's first use.
func (p *Pacer) Now() time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sync()
	return p.base().Add(time.Duration(p.cur) * time.Millisecond)
}

func (p *Pacer) base() time.Time {
	if p.Base.IsZero() {
		return time.Unix(0, 0) // the zero Time would read as "unset" to every IsZero check at virtual time 0
	}
	return p.Base
}

// follow maps a game_ms reading to virtual time. A d2d restarted by the failsafe ladder counts game_ms from 0 again;
// without the carry virtual time would jump back and every stall timer would go negative.
func (p *Pacer) follow(gameMS int) {
	if gameMS < p.last {
		p.carry, p.start = p.cur, 0
	}
	p.last = gameMS
	p.cur = p.carry + gameMS - p.start
}

// Sleep advances the game by ceil(d / 40 ms) ticks (at least one for d > 0). When a step fails the virtual clock
// still advances by d and d also passes in wall time (so timeouts fire, in real time, instead of looping forever or
// racing); the error is kept for Err. A watcher (Watch) runs on this goroutine after the chunk that crossed its interval.
func (p *Pacer) Sleep(d time.Duration) {
	if d <= 0 {
		return
	}
	n := int((d + TickDur - 1) / TickDur)
	chunk := p.Chunk
	if chunk <= 0 {
		chunk = 50
	}
	for n > 0 {
		k := min(n, chunk)
		p.mu.Lock()
		p.sync()
		ci, err := p.C.Step(p.ctx(), k)
		p.steps++
		if err != nil {
			if p.err == nil {
				p.err = err
			}
			lost := n * int(TickDur/time.Millisecond)
			p.carry += lost
			p.cur += lost
			p.mu.Unlock()
			// A dead channel must not race the virtual clock: live (D2DPACE) the vanished socket let 27000 virtual
			// seconds pass in 5 wall minutes, every no-progress timer fired at once and the ladder burned its rungs
			// in a second. Wait the real time too, so an outage is timed as in real time.
			time.Sleep(time.Duration(lost) * time.Millisecond)
			return
		}
		p.ticks += k
		p.lastOK.Store(time.Now().UnixNano())
		p.follow(ci.GameMS)
		fn := p.dueWatch()
		p.mu.Unlock()
		if fn != nil {
			fn()
		}
		n -= k
	}
}

// watcher is the one Watch registration: fn runs on the stepping goroutine when virtual time crosses a multiple of every.
type watcher struct {
	every int // ms
	last  int // virtual ms when it last ran (or was registered)
	fn    func()
	busy  bool // fn is running: a Sleep inside fn steps but does not re-enter it
}

// Watch registers fn to run, on the goroutine that steps the game, right after the step with which virtual time crossed
// a multiple of every (at most once per Sleep chunk). It replaces the "sleep every, then look" goroutine of a watchdog
// (the runner's guard) under lockstep: a goroutine that sleeps on this clock would step the game by itself, and the
// order in which its steps and reads interleave with the acting goroutine's is scheduling, not game state, so two
// same-seed runs diverge at the first action (docs/d2d-determinism.md). fn may call Sleep (it steps more) but is not
// re-entered. The returned stop unregisters it; only one watcher is kept (a new Watch replaces the previous one).
func (p *Pacer) Watch(every time.Duration, fn func()) (stop func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	w := &watcher{every: max(int(every/time.Millisecond), 1), last: p.cur, fn: fn}
	p.watch = w
	return func() {
		p.mu.Lock()
		if p.watch == w {
			p.watch = nil
		}
		p.mu.Unlock()
	}
}

// dueWatch (caller holds mu) returns the watcher's fn when the interval was crossed since it last ran.
func (p *Pacer) dueWatch() func() {
	w := p.watch
	if w == nil || w.busy || p.cur/w.every == w.last/w.every {
		return nil
	}
	w.last = p.cur
	w.busy = true
	return func() {
		defer func() {
			p.mu.Lock()
			w.busy = false
			p.mu.Unlock()
		}()
		w.fn()
	}
}

// Stats returns the step calls and ticks stepped so far, and the virtual time elapsed.
func (p *Pacer) Stats() (steps, ticks int, virtual time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.steps, p.ticks, time.Duration(p.cur) * time.Millisecond
}

// LastStep is the wall time the last step succeeded (zero before the first).
func (p *Pacer) LastStep() time.Time {
	n := p.lastOK.Load()
	if n == 0 {
		return time.Time{}
	}
	return time.Unix(0, n)
}

// Err is the first channel error a Sleep or Now hit (they cannot return one).
func (p *Pacer) Err() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.err
}

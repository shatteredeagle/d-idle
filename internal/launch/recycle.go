package launch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
)

// Scheduled game recycle (agent LEAKHUNT). Game.exe under Wine/ntsync gains file descriptors (anon_inode:ntsync)
// each time it loads an area for the first time in its life (VERIFIED LIVE, docs/leakhunt.md): leave/re-enter
// cycles and revisits cost nothing. The growth is bounded by the number of distinct areas and sits far below the
// 524288 fd limit, so recycling is a safety net, not a necessity. It is a knob: the bot or fleet calls
// Recycler.Maybe between goals (in town, no fight), and it restarts the game cleanly when a limit is reached.

// RecyclePolicy decides when a recycle is due. A zero field disables that trigger; the zero value never recycles.
type RecyclePolicy struct {
	MaxCycles   int           // leave/re-enter cycles (NoteCycle calls) since the last recycle
	MaxFDs      int           // open descriptors of Game.exe
	MinInterval time.Duration // never recycle twice within this time (loop guard)
}

// DefaultRecyclePolicy: generous, tuned from the measured plateau (about 57 fds in town, +70 per new area).
func DefaultRecyclePolicy() RecyclePolicy {
	return RecyclePolicy{MaxCycles: 500, MaxFDs: 4000, MinInterval: 30 * time.Minute}
}

// RecyclePolicyFromEnv overrides the default with D2BOT_RECYCLE_CYCLES, D2BOT_RECYCLE_FDS (integers, 0 = off)
// and D2BOT_RECYCLE_MIN_INTERVAL (a Go duration).
func RecyclePolicyFromEnv() RecyclePolicy {
	p := DefaultRecyclePolicy()
	if v, err := strconv.Atoi(os.Getenv("D2BOT_RECYCLE_CYCLES")); err == nil && v >= 0 {
		p.MaxCycles = v
	}
	if v, err := strconv.Atoi(os.Getenv("D2BOT_RECYCLE_FDS")); err == nil && v >= 0 {
		p.MaxFDs = v
	}
	if d, err := time.ParseDuration(os.Getenv("D2BOT_RECYCLE_MIN_INTERVAL")); err == nil {
		p.MinInterval = d
	}
	return p
}

// CountFDs returns the number of open descriptors of a process.
func CountFDs(pid int) (int, error) {
	ents, err := os.ReadDir(fmt.Sprintf("/proc/%d/fd", pid))
	return len(ents), err
}

// Recycler restarts one game instance. Every effect is a function field (tested offline with fakes).
type Recycler struct {
	Policy RecyclePolicy
	// FDs returns Game.exe's current descriptor count (nil: the fd trigger is off).
	FDs func() (int, error)
	// Leave does Save and Exit to the main menu (Flow.LeaveGame); an error is logged and the stop goes on,
	// because Stop (SIGTERM) is the fallback. V (docs/soak-analysis.md 6c): an area change does NOT write the .d2s (2 changes observed), a kill -9 loses everything since the last Save and Exit.
	Leave func(ctx context.Context) error
	Stop  func(ctx context.Context) error // Life.Stop
	Start func(ctx context.Context) error // Life.Start + helper
	Login func(ctx context.Context) error // attach memory, Flow.Start(character)
	Log   func(string)
	// Progress is called after each finished phase (leave, stop, start, login) so a long restart stays observable
	// (the d2bot watchdog heartbeat); nil is fine.
	Progress func(phase string)
	// Guard takes the cross-process instance lock for the whole recycle (nil: none). A busy lock must return an
	// error at once: another actor (sentinel, tools/live.sh) owns the game launch right now.
	Guard func() (release func(), err error)
	Now   func() time.Time // default time.Now

	mu     sync.Mutex // guards cycles, last, count (runner goroutine writes, HTTP reads)
	cycles int
	last   time.Time
	count  int
}

// NoteCycle records one leave/re-enter cycle (call it from whatever cycles the game).
func (r *Recycler) NoteCycle() {
	r.mu.Lock()
	r.cycles++
	r.mu.Unlock()
}

// Recycles is how many recycles ran.
func (r *Recycler) Recycles() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.count
}

func (r *Recycler) now() time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now()
}

// RecycleStats is everything the decision needs, so it can be made without a Recycler (tests, dashboards).
type RecycleStats struct {
	Cycles int       // leave/re-enter cycles since the last recycle
	FDs    int       // Game.exe descriptors; negative = unknown (the fd trigger is skipped)
	Last   time.Time // last recycle attempt, zero = never
	Now    time.Time
}

// Due is the pure decision: whether a recycle is due for these stats and why. No clocks, no I/O.
func (p RecyclePolicy) Due(st RecycleStats) (bool, string) {
	if !st.Last.IsZero() && p.MinInterval > 0 && st.Now.Sub(st.Last) < p.MinInterval {
		return false, ""
	}
	if p.MaxCycles > 0 && st.Cycles >= p.MaxCycles {
		return true, fmt.Sprintf("%d cycles >= %d", st.Cycles, p.MaxCycles)
	}
	if p.MaxFDs > 0 && st.FDs >= p.MaxFDs {
		return true, fmt.Sprintf("game fds %d >= %d", st.FDs, p.MaxFDs)
	}
	return false, ""
}

// Stats samples the current state (reads the game's fds through r.FDs).
func (r *Recycler) Stats() RecycleStats {
	r.mu.Lock()
	st := RecycleStats{Cycles: r.cycles, FDs: -1, Last: r.last, Now: r.now()}
	r.mu.Unlock()
	if r.FDs != nil {
		if n, err := r.FDs(); err == nil {
			st.FDs = n
		}
	}
	return st
}

// DueFor is Recycler.Due for given stats: pure apart from reading r.Policy. Tests and the dashboard use it.
func (r *Recycler) DueFor(st RecycleStats) (bool, string) { return r.Policy.Due(st) }

// Due reports whether a recycle is due now and why (Stats + DueFor).
func (r *Recycler) Due() (bool, string) { return r.DueFor(r.Stats()) }

// RecycleStatus is the dashboard view: JSON-ready, no secrets.
type RecycleStatus struct {
	Enabled      bool      `json:"enabled"`
	Recycles     int       `json:"recycles"`
	Cycles       int       `json:"cyclesSinceRecycle"`
	MaxCycles    int       `json:"maxCycles"`
	FDs          int       `json:"fds"` // -1 unknown
	MaxFDs       int       `json:"maxFds"`
	LastRecycle  time.Time `json:"lastRecycle,omitempty"`
	NextEligible time.Time `json:"nextEligible,omitempty"` // earliest time a recycle may run (zero = any time)
	Due          bool      `json:"due"`
	DueReason    string    `json:"dueReason,omitempty"`
}

// Status reports the counters for the dashboard. It reads the game's fd count, so call it at poll rate (seconds),
// not in a hot loop.
func (r *Recycler) Status() RecycleStatus {
	st := r.Stats()
	p := r.Policy
	s := RecycleStatus{Enabled: p.MaxCycles > 0 || p.MaxFDs > 0, Recycles: r.Recycles(), Cycles: st.Cycles, MaxCycles: p.MaxCycles,
		FDs: st.FDs, MaxFDs: p.MaxFDs, LastRecycle: st.Last}
	if !st.Last.IsZero() && p.MinInterval > 0 {
		s.NextEligible = st.Last.Add(p.MinInterval)
	}
	s.Due, s.DueReason = r.DueFor(st)
	return s
}

// NewRecycler builds a Recycler with the policy from the environment (RecyclePolicyFromEnv). The caller fills the
// effect functions (docs/soak.md has the wiring).
func NewRecycler() *Recycler { return &Recycler{Policy: RecyclePolicyFromEnv()} }

// FromEnv is the same as NewRecycler, as a method for a Recycler already holding its effects: it only replaces Policy.
func (r *Recycler) FromEnv() *Recycler { r.Policy = RecyclePolicyFromEnv(); return r }

// Maybe recycles when Due. It reports whether it did. Call it between goals, in town.
func (r *Recycler) Maybe(ctx context.Context) (bool, error) {
	due, why := r.Due()
	if !due {
		return false, nil
	}
	return true, r.Recycle(ctx, why)
}

// Recycle always runs: Save and Exit, stop, start, log in. The cycle counter resets only on success.
func (r *Recycler) Recycle(ctx context.Context, reason string) error {
	logf := func(f string, a ...any) {
		if r.Log != nil {
			r.Log(fmt.Sprintf(f, a...))
		}
	}
	logf("recycle: %s", reason)
	if r.Guard != nil {
		release, err := r.Guard()
		if err != nil {
			logf("recycle: instance lock busy, not restarting the game: %v", err)
			return fmt.Errorf("recycle: %w", err)
		}
		defer release()
	}
	progress := func(phase string) {
		if r.Progress != nil {
			r.Progress(phase)
		}
	}
	r.mu.Lock()
	r.last = r.now() // also on failure: MinInterval stops a retry storm
	r.mu.Unlock()
	if r.Leave != nil {
		if err := r.Leave(ctx); err != nil {
			logf("recycle: leave failed (%v), stopping anyway", err)
		}
	}
	progress("leave")
	if r.Stop == nil || r.Start == nil || r.Login == nil {
		return errors.New("recycle: Stop, Start and Login are required")
	}
	if err := r.Stop(ctx); err != nil {
		return fmt.Errorf("recycle stop: %w", err)
	}
	progress("stop")
	if err := r.Start(ctx); err != nil {
		return fmt.Errorf("recycle start: %w", err)
	}
	progress("start")
	if err := r.Login(ctx); err != nil {
		return fmt.Errorf("recycle login: %w", err)
	}
	progress("login")
	r.mu.Lock()
	r.cycles = 0
	r.count++
	r.mu.Unlock()
	logf("recycle done")
	return nil
}

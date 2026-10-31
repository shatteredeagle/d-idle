package launch

import "time"

// Event is a health transition reported by Monitor.
type Event int

const (
	EventNone    Event = iota
	EventUp            // process appeared
	EventExited        // process vanished, no crash dialog seen
	EventCrashed       // process vanished or is stuck behind Blizzard's crash reporter
	EventHung          // process alive but frozen: no heartbeat change for HangAfter
)

func (e Event) String() string {
	return [...]string{"none", "up", "exited", "crashed", "hung"}[e]
}

// Probes are the observations Monitor needs; all are injectable so the logic is testable offline.
type Probes struct {
	// Alive returns the pid of the game process.
	Alive func() (pid int, ok bool)
	// Heartbeat returns a number that keeps changing while the process is healthy. The default
	// implementation is CPUTicks (VERIFIED to advance ~11/s idle in town; UNVERIFIED in menus/loading).
	Heartbeat func(pid int) (uint64, error)
	// CrashDialog reports whether the crash reporter is running. UNVERIFIED: the install contains
	// BlizzardError.exe, which 1.14d starts on a crash/assert; confirm by name when one happens.
	CrashDialog func() bool
}

// DefaultProbes wires Probes to the real system for g.
func DefaultProbes(g *Game) Probes {
	return Probes{
		Alive: func() (int, bool) {
			p, ok := g.Find()
			return p.PID, ok
		},
		Heartbeat: CPUTicks,
		CrashDialog: func() bool {
			ps, err := ListProcs()
			return err == nil && len(FindImage(ps, "blizzarderror.exe")) > 0
		},
	}
}

// Monitor turns periodic samples into events. It is a plain state machine: call Step on a ticker
// (every 1-2 s). It reports each transition once.
type Monitor struct {
	P         Probes
	HangAfter time.Duration // heartbeat flat for this long = hung (default 45 s)

	pid        int
	lastBeat   uint64
	lastChange time.Time
	hungSent   bool
}

// Step samples once at time now.
func (m *Monitor) Step(now time.Time) Event {
	if m.HangAfter == 0 {
		m.HangAfter = 45 * time.Second
	}
	pid, ok := m.P.Alive()
	if !ok {
		was := m.pid != 0
		m.pid, m.hungSent = 0, false
		if !was {
			return EventNone
		}
		if m.P.CrashDialog != nil && m.P.CrashDialog() {
			return EventCrashed
		}
		return EventExited
	}
	if m.P.CrashDialog != nil && m.P.CrashDialog() && !m.hungSent {
		m.hungSent = true
		return EventCrashed
	}
	if pid != m.pid {
		m.pid, m.hungSent = pid, false
		m.lastChange = now
		m.lastBeat, _ = m.P.Heartbeat(pid)
		return EventUp
	}
	beat, err := m.P.Heartbeat(pid)
	if err == nil && beat != m.lastBeat {
		m.lastBeat, m.lastChange, m.hungSent = beat, now, false
		return EventNone
	}
	if !m.hungSent && now.Sub(m.lastChange) >= m.HangAfter {
		m.hungSent = true
		return EventHung
	}
	return EventNone
}

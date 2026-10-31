package launch

import (
	"testing"
	"time"
)

func TestMonitor(t *testing.T) {
	pid, beat, crash := 0, uint64(0), false
	m := &Monitor{HangAfter: 10 * time.Second, P: Probes{
		Alive:       func() (int, bool) { return pid, pid != 0 },
		Heartbeat:   func(int) (uint64, error) { return beat, nil },
		CrashDialog: func() bool { return crash },
	}}
	t0 := time.Unix(1000, 0)
	at := func(s int) time.Time { return t0.Add(time.Duration(s) * time.Second) }
	step := func(s int, want Event) {
		t.Helper()
		if got := m.Step(at(s)); got != want {
			t.Fatalf("t=%d got %v want %v", s, got, want)
		}
	}
	step(0, EventNone)
	pid = 42
	step(1, EventUp)
	beat = 5
	step(2, EventNone)
	step(11, EventNone) // flat for 9 s
	step(12, EventHung) // flat for 10 s
	step(13, EventNone) // reported once
	beat = 6
	step(14, EventNone) // recovered
	pid = 0
	step(15, EventExited)
	step(16, EventNone)
	pid, beat = 43, 1
	step(17, EventUp)
	crash = true
	step(18, EventCrashed)
	pid, crash = 0, true
	step(19, EventCrashed)
}

package goals

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCoverRunReturnsOnCancelAndSaves(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.json")
	m, f, _ := newMgr(t, path)
	mustAdd(t, m, lvl(9))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- m.Run(ctx) }()
	f.ch <- Event{Type: EvControl, Message: "ping"} // pumped, must not block the loop
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run ignores ctx cancellation")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("state not saved on shutdown: %v", err)
	}
}

func TestCoverTickDoesNotCreditAClockJump(t *testing.T) {
	m, _, c := newMgr(t, "")
	mustAdd(t, m, Goal{Type: FarmArea, Params: Params{Area: "A"}})
	m.Start()
	m.Tick(c.now())
	c.advance(time.Hour) // host suspended or clock stepped
	m.Tick(c.now())
	g, _ := m.Goal("g1")
	if g.Progress.ActiveSeconds > int64(maxCreditGap/time.Second) {
		t.Fatalf("credited %ds for a %v gap", g.Progress.ActiveSeconds, time.Hour)
	}
	c.advance(5 * time.Second)
	m.Tick(c.now())
	g, _ = m.Goal("g1")
	if g.Progress.ActiveSeconds < 5 {
		t.Fatalf("normal tick not credited: %d", g.Progress.ActiveSeconds)
	}
}

func TestCoverSaveFailureDoesNotWedgeTheScheduler(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.json")
	m, f, c := newMgr(t, path)
	mustAdd(t, m, lvl(5))
	mustAdd(t, m, lvl(7))
	// make the target a directory so the atomic rename fails
	_ = os.Remove(path)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "x"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	m.Start()
	for i := 0; i < 5; i++ {
		c.advance(time.Minute)
		m.Tick(c.now())
	}
	if f.lastStarted() == "" {
		t.Fatal("scheduling stopped after save failure")
	}
	if err := m.Close(); err == nil {
		t.Fatal("expected the save error to surface from Close")
	}
	if m.Snapshot().LastError == "" {
		t.Fatal("save failure not visible to the operator")
	}
}

func TestCoverRunnerErrorsOnEveryGoalFailQueue(t *testing.T) {
	m, f, _ := newMgr(t, "")
	f.failFor = map[Type]bool{ReachLevel: true, KillBoss: true}
	mustAdd(t, m, lvl(5))
	mustAdd(t, m, boss("Andariel", 1))
	m.Start()
	for _, g := range m.Goals() {
		if g.State != StateFailed {
			t.Fatalf("%s is %s", g.ID, g.State)
		}
	}
	// nothing runnable: Start again must not spin or panic
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
}

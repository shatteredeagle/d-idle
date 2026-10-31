package main

import (
	"d2lod-bot/internal/launch"
	"os/exec"
	"sync/atomic"
	"testing"
	"time"
)

func TestExitCode(t *testing.T) {
	if got := exitCode(exec.Command("sh", "-c", "exit 7").Run()); got != 7 {
		t.Fatalf("exit 7: %d", got)
	}
	if got := exitCode(exec.Command("sh", "-c", "kill -9 $$").Run()); got != 128+9 {
		t.Fatalf("SIGKILL: %d", got)
	}
	if got := exitCode(nil); got != 0 {
		t.Fatalf("nil: %d", got)
	}
}

// The reaper must not steal the exit status of the supervised child.
func TestReapOrphansSkipsChild(t *testing.T) {
	if err := launch.BecomeSubreaper(); err != nil {
		t.Skip(err)
	}
	cmd := exec.Command("sh", "-c", "exit 3")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var skip atomic.Int64
	skip.Store(int64(cmd.Process.Pid))
	time.Sleep(200 * time.Millisecond) // child is now a zombie of ours
	reapOrphans(&skip)
	if got := exitCode(cmd.Wait()); got != 3 {
		t.Fatalf("exit code %d, want 3 (reaper ate it)", got)
	}
}

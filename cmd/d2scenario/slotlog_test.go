package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"d2lod-bot/internal/scenario"
)

// The run writes its own start line and reads its peak from the runner's log; without the env it records 0 (unknown).
func TestSlotParallelFromRunnerLog(t *testing.T) {
	log := filepath.Join(t.TempDir(), slotLogName)
	t.Setenv(slotLogEnv, "")
	t.Setenv(slotIDEnv, "")
	if got := slotParallel(); got != 0 {
		t.Fatalf("no runner: %d, want 0", got)
	}
	if err := os.WriteFile(log, []byte(scenario.SlotLogLine("start", "other", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(slotLogEnv, log)
	t.Setenv(slotIDEnv, "me")
	logSlotStart(time.Unix(0, 2))
	if got := slotParallel(); got != 2 {
		t.Fatalf("another run busy: %d, want 2", got)
	}
}

package main

import (
	"log/slog"
	"os"
	"strconv"
	"time"
)

// Test hook for the crash drill (tools/soak/crashdrill.sh, docs/soak24.md): with D2BOT_TEST=1 and
// D2BOT_TEST_CRASH_AFTER=<seconds> the process panics in a goroutine goSafe does not wrap, which kills it with exit 2
// (or the hidden --test-crash-after=N flag, for tools/live.sh which cannot pass env) like a real unrecovered panic. The supervisor drops the variable on restarts, so the drill crashes exactly once.
const testCrashEnv = "D2BOT_TEST_CRASH_AFTER"

// testCrashDelay reports the crash delay, only when D2BOT_TEST=1.
func testCrashDelay(getenv func(string) string) (time.Duration, bool) {
	if getenv("D2BOT_TEST") != "1" {
		return 0, false
	}
	n, err := strconv.Atoi(getenv(testCrashEnv))
	if err != nil || n < 0 {
		return 0, false
	}
	return time.Duration(n) * time.Second, true
}

func armTestCrash(log *slog.Logger, flag int) {
	d, ok := testCrashDelay(os.Getenv)
	if flag >= 0 {
		d, ok = time.Duration(flag)*time.Second, true
	}
	if !ok {
		return
	}
	log.Warn("TEST CRASH armed: this process will panic", "after", d)
	go func() {
		time.Sleep(d)
		panic("D2BOT_TEST_CRASH_AFTER drill")
	}()
}

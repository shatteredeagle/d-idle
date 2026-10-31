package main

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"runtime/pprof"
	"strconv"
	"time"
)

// defaultWatchdog is how long the runner loop may go without finishing an iteration before the process is restarted.
// Longest legitimate single step is a game recycle (a few minutes), so 10 min leaves margin. D2BOT_WATCHDOG_MIN
// overrides it in minutes (0 turns the watchdog off).
const defaultWatchdog = 10 * time.Minute

func watchdogLimit() time.Duration {
	if v := os.Getenv("D2BOT_WATCHDOG_MIN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return time.Duration(n) * time.Minute
		}
	}
	return defaultWatchdog
}

// watchdog catches a runner stuck forever (a panic between a manual Lock and Unlock leaves the mutex held, a hung
// call): the process stays alive and healthy-looking, so the supervisor never sees it. It dumps every goroutine's
// stack to the log and exits with exitStorm so the supervisor (or the fleet) restarts d2bot.
type watchdog struct {
	log   *slog.Logger
	last  func() time.Time // Runner.LastStep; zero = the loop has not started
	limit time.Duration
	every time.Duration
	now   func() time.Time
	sleep func(ctx context.Context, d time.Duration)
	dump  func() string
	exit  func(code int)
}

func newWatchdog(log *slog.Logger, last func() time.Time, limit time.Duration) *watchdog {
	return &watchdog{log: log, last: last, limit: limit, every: 15 * time.Second, now: time.Now,
		sleep: func(ctx context.Context, d time.Duration) {
			select {
			case <-ctx.Done():
			case <-time.After(d):
			}
		},
		dump: allStacks,
		exit: os.Exit,
	}
}

func allStacks() string {
	var b bytes.Buffer
	_ = pprof.Lookup("goroutine").WriteTo(&b, 2)
	return b.String()
}

// check reports whether the watchdog fired.
func (w *watchdog) check() bool {
	t := w.last()
	if t.IsZero() || w.limit <= 0 {
		return false
	}
	if age := w.now().Sub(t); age >= w.limit {
		w.log.Error("watchdog: runner loop made no progress, exiting for a restart", "age", age.Round(time.Second), "limit", w.limit)
		w.log.Error("watchdog: goroutine dump", "stacks", w.dump())
		w.exit(exitStorm)
		return true
	}
	return false
}

// run checks until ctx ends or the watchdog fires.
func (w *watchdog) run(ctx context.Context) {
	for ctx.Err() == nil {
		if w.check() {
			return
		}
		w.sleep(ctx, w.every)
	}
}

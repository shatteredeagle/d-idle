package main

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestWatchdogVirtualClock(t *testing.T) {
	now := time.Unix(1000, 0)
	last := time.Time{}
	var exits []int
	w := newWatchdog(slog.Default(), func() time.Time { return last }, 10*time.Minute)
	w.now = func() time.Time { return now }
	w.dump = func() string { return "stacks" }
	w.exit = func(c int) { exits = append(exits, c) }

	if w.check() { // loop not started
		t.Fatal("fired before the loop started")
	}
	last = now
	now = now.Add(9 * time.Minute)
	if w.check() {
		t.Fatal("fired too early")
	}
	last = now // a step finished: the age restarts
	now = now.Add(9 * time.Minute)
	if w.check() {
		t.Fatal("fired after a heartbeat")
	}
	now = now.Add(time.Minute)
	if !w.check() || len(exits) != 1 || exits[0] != exitStorm {
		t.Fatalf("want one exitStorm, got %v", exits)
	}
}

func TestWatchdogOffAndRun(t *testing.T) {
	w := newWatchdog(slog.Default(), func() time.Time { return time.Unix(1, 0) }, 0)
	if w.check() {
		t.Fatal("limit 0 must disable")
	}
	w = newWatchdog(slog.Default(), func() time.Time { return time.Unix(1, 0) }, time.Minute)
	w.dump = func() string { return "" }
	fired := make(chan int, 1)
	w.exit = func(c int) { fired <- c }
	w.sleep = func(context.Context, time.Duration) {}
	w.run(context.Background())
	if <-fired != exitStorm {
		t.Fatal("run did not fire")
	}
}

func TestUnderFleet(t *testing.T) {
	t.Setenv("D2BOT_FLEET", "1")
	if !underFleet() {
		t.Fatal("D2BOT_FLEET=1 must count as under the fleet")
	}
}

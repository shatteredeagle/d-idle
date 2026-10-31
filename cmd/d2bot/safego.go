package main

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync/atomic"
	"time"
)

// A panic in a helper goroutine kills the whole process (the incident of 2026-10-29: the bot died and the game sat at the
// death screen for ten minutes). goSafe runs f and, when it panics, logs the stack and runs it again, at most
// maxGoPanics times per hour; past that it gives up on that goroutine.
const maxGoPanics = 10

var goPanics atomic.Int64 // recovered goroutine panics since start, for /status

var goBackoff = time.Second // a var for the tests

// goSafe runs f until it returns normally or ctx ends, restarting it after a recovered panic. It reports false when it
// gave up: callers whose goroutine the process cannot do without end the process then (supervisor restart).
func goSafe(ctx context.Context, log *slog.Logger, name string, f func()) bool {
	var times []time.Time
	for ctx.Err() == nil {
		v, stack := runRecovered(f)
		if v == nil {
			return true
		}
		goPanics.Add(1)
		log.Error("goroutine panic recovered", "goroutine", name, "panic", fmt.Sprint(v), "stack", string(stack))
		now := time.Now()
		keep := times[:0]
		for _, t := range times {
			if now.Sub(t) < time.Hour {
				keep = append(keep, t)
			}
		}
		times = append(keep, now)
		if len(times) > maxGoPanics {
			log.Error("goroutine panicked too often, not restarting it", "goroutine", name)
			return false
		}
		select {
		case <-ctx.Done():
			return true
		case <-time.After(goBackoff):
		}
	}
	return true
}

// runRecovered runs f and returns the panic value and stack, or nil.
func runRecovered(f func()) (v any, stack []byte) {
	defer func() {
		if v = recover(); v != nil {
			stack = debug.Stack()
		}
	}()
	f()
	return nil, nil
}

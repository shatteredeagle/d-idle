package main

import "runtime"

// osThread runs every function it is given on ONE locked OS thread.
//
// Why: under Wine each OS thread of the helper costs about 7 host fds (measured on the 2 h soak of i4: d2input
// threads 9 -> 17 and fds 74 -> 134 move together, heap and goroutine count flat; docs/leakhunt.md). The Go
// runtime starts a new thread whenever a goroutine sits in a Win32 call (EnumWindows, PostMessage, ... each
// is a wineserver round trip) long enough for the scheduler to hand its P over, and never ends idle threads.
// With all Win32 calls on one dedicated thread the number of threads stays bounded, and the calls are
// serialized as a side effect (the EnumWindows callback no longer needs its own lock).
type osThread struct {
	jobs chan func()
}

func newOSThread() *osThread {
	t := &osThread{jobs: make(chan func())}
	go func() {
		runtime.LockOSThread()
		for f := range t.jobs {
			f()
		}
	}()
	return t
}

// do runs f on the thread and returns when it is done. A panic in f is re-raised in the caller, so the
// per-command recover in safeHandle still sees it and the thread survives.
func (t *osThread) do(f func()) {
	done := make(chan any, 1)
	t.jobs <- func() {
		defer func() { done <- recover() }()
		f()
	}
	if p := <-done; p != nil {
		panic(p)
	}
}

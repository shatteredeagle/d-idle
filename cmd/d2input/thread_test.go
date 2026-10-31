//go:build linux

package main

import (
	"sync"
	"syscall"
	"testing"
)

func TestOSThreadSingleThread(t *testing.T) {
	th := newOSThread()
	seen := map[int]bool{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			th.do(func() {
				mu.Lock()
				seen[syscall.Gettid()] = true
				mu.Unlock()
			})
		}()
	}
	wg.Wait()
	if len(seen) != 1 {
		t.Fatalf("jobs ran on %d threads, want 1", len(seen))
	}
}

func TestOSThreadPanicSurvives(t *testing.T) {
	th := newOSThread()
	func() {
		defer func() {
			if recover() == nil {
				t.Error("panic not propagated")
			}
		}()
		th.do(func() { panic("boom") })
	}()
	ran := false
	th.do(func() { ran = true })
	if !ran {
		t.Fatal("thread dead after panic")
	}
}

//go:build linux

package d2dreplay

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"
)

// LabLock is the shared lock every d2d run on the worker takes (openspec/specs/lab-rules: one d2d at a time).
const LabLock = "/tmp/d2d-lab.lock"

// SlotLocks are the lock files of the 4-slot d2d pool, the same paths tools/d2dslot.sh uses (slot 0 is the legacy
// lab lock; docs/process/worker-compute.md).
func SlotLocks() []string {
	return []string{LabLock, "/tmp/d2d-lab-slot-1.lock", "/tmp/d2d-lab-slot-2.lock", "/tmp/d2d-lab-slot-3.lock"}
}

// TakeLock waits up to wait for an exclusive flock on path (like `flock -w SECONDS`). Release with the returned func.
func TakeLock(ctx context.Context, path string, wait time.Duration, logf func(string, ...any)) (func(), error) {
	_, rel, err := TakeAny(ctx, []string{path}, wait, logf)
	return rel, err
}

// TakeAny takes the first free lock of paths (re-trying all of them every 2 s until wait is over, so the context
// can cancel it) and returns its index. Held only while a d2d runs.
func TakeAny(ctx context.Context, paths []string, wait time.Duration, logf func(string, ...any)) (int, func(), error) {
	deadline := time.Now().Add(wait)
	told := false
	for {
		for i, path := range paths {
			f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o666)
			if err != nil {
				return -1, nil, err
			}
			err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
			if err == nil {
				return i, func() { syscall.Flock(int(f.Fd()), syscall.LOCK_UN); f.Close() }, nil
			}
			f.Close()
			if err != syscall.EWOULDBLOCK {
				return -1, nil, err
			}
		}
		if !told && logf != nil {
			logf("waiting for a free d2d slot (%d held)", len(paths))
			told = true
		}
		if time.Now().After(deadline) {
			return -1, nil, fmt.Errorf("d2dreplay: no free d2d slot after %s", wait)
		}
		select {
		case <-ctx.Done():
			return -1, nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

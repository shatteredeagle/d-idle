//go:build !linux

package d2dreplay

import (
	"context"
	"errors"
	"time"
)

const LabLock = "/tmp/d2d-lab.lock"

// TakeLock is Linux only (flock on the lab lock); elsewhere playback refuses to start a d2d.
func TakeLock(ctx context.Context, path string, wait time.Duration, logf func(string, ...any)) (func(), error) {
	return nil, errors.New("d2dreplay: the lab lock needs Linux")
}

func SlotLocks() []string { return []string{LabLock} }

func TakeAny(ctx context.Context, paths []string, wait time.Duration, logf func(string, ...any)) (int, func(), error) {
	return -1, nil, errors.New("d2dreplay: the lab lock needs Linux")
}

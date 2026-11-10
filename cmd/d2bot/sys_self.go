package main

import (
	"os"
	"syscall"
)

// signalSelfTerm sends SIGTERM to this process (the `docker stop` path). On Windows os.Process.Signal
// reports an error for it; the callers surface that.
func signalSelfTerm() error {
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		return err
	}
	return p.Signal(syscall.SIGTERM)
}

//go:build !unix

package main

import (
	"errors"
	"os/exec"
	"time"
)

// Process groups and SIGHUP do not exist here: the shim still runs, it just cannot sweep leftovers or reload.
func setOwnGroup(*exec.Cmd)         {}
func sweepGroup(int, time.Duration) {}
func sendHUP(int) error             { return errors.New("SIGHUP reload is not supported on this OS") }

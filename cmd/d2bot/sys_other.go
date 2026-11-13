//go:build !linux

package main

import (
	"errors"
	"os/exec"
)

func setPdeathsig(*exec.Cmd) {}

// diskFreeBytes: not probed off Linux; the updater treats an error as "not checked".
func diskFreeBytes(string) (uint64, error) { return 0, errors.New("disk probe is Linux only") }

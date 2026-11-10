//go:build linux

package main

import (
	"os/exec"
	"syscall"
)

// setPdeathsig makes the kernel SIGTERM the child when the supervising thread dies (Linux only).
func setPdeathsig(c *exec.Cmd) { c.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM} }

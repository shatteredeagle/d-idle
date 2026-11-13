//go:build linux

package main

import (
	"os/exec"
	"syscall"
)

// setPdeathsig makes the kernel SIGTERM the child when the supervising thread dies (Linux only).
func setPdeathsig(c *exec.Cmd) { c.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM} }

// diskFreeBytes is the space a normal user can still write under path.
func diskFreeBytes(path string) (uint64, error) {
	var s syscall.Statfs_t
	if err := syscall.Statfs(path, &s); err != nil {
		return 0, err
	}
	return uint64(s.Bavail) * uint64(s.Bsize), nil
}

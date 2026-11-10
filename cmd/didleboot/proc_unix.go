//go:build unix

package main

import (
	"os/exec"
	"syscall"
	"time"
)

// own process group, so the launcher's leftovers can be swept (sweepGroup)
func setOwnGroup(c *exec.Cmd) { c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }

// sweepGroup ends whatever the launcher left behind in its process group (VERIFIED LIVE, TASK-192: the setup wizard
// keeps running when the launcher shell is terminated, and then holds its port against the next start). TERM first,
// KILL after the grace. The launcher itself was terminated alone and in order (bot before tap), so this only sees
// leftovers.
func sweepGroup(pgid int, grace time.Duration) {
	alive := func() bool { return syscall.Kill(-pgid, 0) == nil }
	if !alive() {
		return
	}
	syscall.Kill(-pgid, syscall.SIGTERM)
	for end := time.Now().Add(grace); time.Now().Before(end) && alive(); {
		time.Sleep(100 * time.Millisecond)
	}
	if alive() {
		syscall.Kill(-pgid, syscall.SIGKILL)
		time.Sleep(200 * time.Millisecond)
	}
}

func sendHUP(pid int) error { return syscall.Kill(pid, syscall.SIGHUP) }

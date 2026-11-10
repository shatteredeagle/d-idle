//go:build linux

package launch

import (
	"os"
	"os/exec"
	"syscall"
)

// The Wine/Xvfb lifecycle is Linux-only (ADR: the bot drives Wine through /proc and prctl). Every
// syscall it needs lives here; os_other.go is the stub for the other GOOS values.

func sysKill(pid int, sig syscall.Signal) error { return syscall.Kill(pid, sig) }

// tryFlock takes an exclusive non-blocking flock; busy is true when another holder has it.
func tryFlock(f *os.File) (busy bool, err error) {
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	return err == syscall.EWOULDBLOCK, err
}

func unFlock(f *os.File) { syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }

func wait4NoHang(pid int) error {
	var ws syscall.WaitStatus
	_, err := syscall.Wait4(pid, &ws, syscall.WNOHANG, nil)
	return err
}

func setSession(c *exec.Cmd) { c.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }
func setGroup(c *exec.Cmd)   { c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }

func setPdeathsig(c *exec.Cmd) {
	if c.SysProcAttr == nil {
		c.SysProcAttr = &syscall.SysProcAttr{}
	}
	c.SysProcAttr.Pdeathsig = syscall.SIGTERM
}

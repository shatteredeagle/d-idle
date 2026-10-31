package launch

import (
	"bytes"
	"context"
	"net"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"d2lod-bot/internal/mem"
)

// System is everything Life needs from the operating system. OSSystem is the real one; tests use a
// fake, so the lifecycle logic (start when running, stop cleanup, mkprefix refusal, marker env) is
// checked without Wine, X or any process.
type System interface {
	Procs() ([]Proc, error)
	// Marker is the D2BOT_INSTANCE value in the environment of pid ("" if none or unreadable).
	Marker(pid int) string
	Signal(pid int, sig syscall.Signal) error
	// Run runs a command to completion and returns its combined output. env nil = inherit.
	Run(ctx context.Context, env []string, dir, name string, args ...string) ([]byte, error)
	// Spawn starts a detached process (own session, stdin /dev/null, output appended to logPath) and
	// returns once it is started. The process outlives the caller.
	Spawn(env []string, dir, logPath, name string, args ...string) error
	Exists(path string) bool
	// HelperUp reports whether something listens on 127.0.0.1:port.
	HelperUp(port int) bool
	Sleep(ctx context.Context, d time.Duration)
}

// OSSystem is the real System.
type OSSystem struct{}

func (OSSystem) Procs() ([]Proc, error)                 { return ListProcs() }
func (OSSystem) Marker(pid int) string                  { return mem.ProcInstance(pid) }
func (OSSystem) Signal(pid int, s syscall.Signal) error { return syscall.Kill(pid, s) }
func (OSSystem) Exists(path string) bool                { _, err := os.Stat(path); return err == nil }

func (OSSystem) Run(ctx context.Context, env []string, dir, name string, args ...string) ([]byte, error) {
	c := exec.CommandContext(ctx, name, args...)
	c.Env, c.Dir = env, dir
	var b bytes.Buffer
	c.Stdout, c.Stderr = &b, &b
	err := c.Run()
	return b.Bytes(), err
}

func (OSSystem) Spawn(env []string, dir, logPath, name string, args ...string) error {
	lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer lf.Close()
	c := exec.Command(name, args...)
	c.Env, c.Dir = env, dir
	c.Stdout, c.Stderr = lf, lf
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := c.Start(); err != nil {
		return err
	}
	go c.Wait() // reap the launcher if we outlive it (sentinel); the child itself keeps running
	return nil
}

func (OSSystem) HelperUp(port int) bool {
	c, err := net.DialTimeout("tcp", "127.0.0.1:"+strconv.Itoa(port), 500*time.Millisecond)
	if err != nil {
		return false
	}
	c.Close()
	return true
}

func (OSSystem) Sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

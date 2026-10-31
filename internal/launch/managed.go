package launch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"d2lod-bot/internal/mem"
)

// Managed is one Windows program run through `proton run` as a descendant of this process. The direct
// child is only the python launcher; the program itself is found afterwards by its image name.
//
// Requires BecomeSubreaper to have been called, otherwise the program is re-parented to PID 1 once the
// launcher exits and cannot be inspected (see docs/launch.md).
type Managed struct {
	Cfg   Config
	Spawn *Spawner // optional; nil starts directly
	// Exe is the host path of the Windows program, Args its arguments, Cwd its working directory.
	Exe  string
	Args []string
	Cwd  string
	// Image is the lower-case Windows image name used to find the process ("game.exe", "d2input.exe").
	Image string
	// Log is the log file name under Cfg.LogDir for the launcher's stdout/stderr.
	Log string

	mu       sync.Mutex
	launcher chan struct{} // closed when the python launcher exits
}

// Start launches the program. It returns once the launcher has been started, not when the program is up;
// use WaitUp.
func (m *Managed) Start(ctx context.Context) error {
	if _, err := os.Stat(m.Exe); err != nil {
		return err
	}
	if err := os.MkdirAll(m.Cfg.LogDir, 0o755); err != nil {
		return err
	}
	lf, err := os.OpenFile(filepath.Join(m.Cfg.LogDir, m.Log), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	// Not CommandContext-bound to ctx: cancelling ctx must not orphan half a prefix.
	cmd := m.Cfg.ProtonCmd(context.Background(), m.Cwd, "run", append([]string{m.Exe}, m.Args...)...)
	cmd.Stdout, cmd.Stderr = lf, lf
	if m.Spawn != nil {
		err = m.Spawn.Start(cmd)
	} else {
		err = cmd.Start()
	}
	if err != nil {
		lf.Close()
		return err
	}
	owned.Store(cmd.Process.Pid, true)
	done := make(chan struct{})
	m.mu.Lock()
	m.launcher = done
	m.mu.Unlock()
	go func() {
		cmd.Wait() // reaps the python launcher; re-parented Wine processes are handled by Reap
		owned.Delete(cmd.Process.Pid)
		lf.Close()
		close(done)
	}()
	return nil
}

// Find returns the running program, preferring the highest PID (Wine can show a short-lived duplicate of
// the same image while it re-execs). ok is false if it is not running.
func (m *Managed) Find() (Proc, bool) {
	ps, err := ListProcs()
	if err != nil {
		return Proc{}, false
	}
	c := FindImage(ps, m.Image)
	if m.Cfg.Instance != "" {
		var mine []Proc
		for _, p := range c {
			if got := mem.ProcInstance(p.PID); got == m.Cfg.Instance || (m.Cfg.Instance == mem.DefaultInstance && got == "") {
				mine = append(mine, p)
			}
		}
		c = mine
	}
	if len(c) == 0 {
		return Proc{}, false
	}
	return c[len(c)-1], true
}

// WaitUp polls until the program is running or the timeout/context expires.
func (m *Managed) WaitUp(ctx context.Context, timeout time.Duration) (Proc, error) {
	deadline := time.Now().Add(timeout)
	for {
		if p, ok := m.Find(); ok {
			return p, nil
		}
		if time.Now().After(deadline) {
			return Proc{}, fmt.Errorf("%s did not start within %s", m.Image, timeout)
		}
		select {
		case <-ctx.Done():
			return Proc{}, ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}
}

// Stop ends the program: SIGTERM (Wine turns it into a normal process exit), then SIGKILL after grace.
// It returns nil if the program is gone afterwards.
func (m *Managed) Stop(grace time.Duration) error {
	p, ok := m.Find()
	if !ok {
		return nil
	}
	_ = syscall.Kill(p.PID, syscall.SIGTERM)
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		if _, ok := m.Find(); !ok {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	_ = syscall.Kill(p.PID, syscall.SIGKILL)
	time.Sleep(500 * time.Millisecond)
	if _, ok := m.Find(); ok {
		return errors.New(m.Image + " survived SIGKILL")
	}
	return nil
}

// owned holds PIDs of exec.Cmd children, whose exit status belongs to Cmd.Wait and must not be reaped
// by Reap.
var owned sync.Map

// Reap collects zombie descendants that were re-parented to us. As a subreaper we become the parent of
// every orphan, and a dead orphan stays a zombie until we wait() for it. Call it from a ticker (every
// few seconds). It never touches exec.Cmd children. Zombies do not break /proc/pid/mem of live
// processes, but they pile up over days.
func Reap() {
	ps, err := ListProcs()
	if err != nil {
		return
	}
	me := os.Getpid()
	for _, p := range ps {
		if p.PPID != me || p.State != 'Z' {
			continue
		}
		if _, mine := owned.Load(p.PID); mine {
			continue
		}
		var ws syscall.WaitStatus
		syscall.Wait4(p.PID, &ws, syscall.WNOHANG, nil)
	}
}

//go:build unix

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"d2lod-bot/internal/d2dctl"
)

// Lab starts one lockstep d2d for one seed and stops it again. The real one is procLab; tests use a fake server.
type Lab interface {
	Start(ctx context.Context, seed int, logPath string) (sock string, err error)
	Stop() error
}

const lockPath = "/tmp/d2d-lab.lock"

// labLock is the cross-agent rule "one d2d at a time" (docs/d2d-fork.md): an exclusive flock held for a whole seed.
type labLock struct{ f *os.File }

func acquireLabLock() (*labLock, error) {
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o666)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("%s is held by another d2d user, try later: %w", lockPath, err)
	}
	return &labLock{f}, nil
}

func (l *labLock) release() {
	syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	l.f.Close()
}

// procLab runs the private fork binary. Flags follow the live d2d rules: niced, --no-save, autoloot and autoparty off,
// private XDG dirs and socket, the shared MPQ copy read-only.
type procLab struct {
	Bin, Data, Home, Toggles string
	Class                    int
	Name                     string
	MaxLoad                  float64
	MinFreeGB                float64

	lock *labLock
	cmd  *exec.Cmd
	sock string
}

func (p *procLab) Start(ctx context.Context, seed int, logPath string) (string, error) {
	var err error
	if p.lock, err = acquireLabLock(); err != nil {
		return "", err
	}
	fail := func(e error) (string, error) { p.lock.release(); p.lock = nil; return "", e }
	if out, _ := exec.Command("pgrep", "-x", "d2d").Output(); len(strings.TrimSpace(string(out))) > 0 {
		return fail(fmt.Errorf("a d2d is already running (pid %s): not touching it", strings.Fields(string(out))[0]))
	}
	if la := loadAvg(); p.MaxLoad > 0 && la > p.MaxLoad {
		return fail(fmt.Errorf("load average %.1f above %.0f: refusing to start (docs/d2d-fidelity.md safety rule)", la, p.MaxLoad))
	}
	if g := freeGB("/"); p.MinFreeGB > 0 && g < p.MinFreeGB {
		return fail(fmt.Errorf("only %.1f GB free on /, need %.0f", g, p.MinFreeGB))
	}
	if err := os.MkdirAll(p.Home, 0o755); err != nil {
		return fail(err)
	}
	p.sock = filepath.Join(p.Home, "d2d.sock")
	os.Remove(p.sock)
	lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fail(err)
	}
	defer lf.Close()
	args := []string{"-n", "10", p.Bin, "--data", p.Data, "--headless", "--devctl", p.sock, "--no-video", "--lockstep", "--no-save",
		"--toggle", p.Toggles, "--seed", strconv.Itoa(seed), "--start-screen", "ingame",
		"--start-class", strconv.Itoa(p.Class), "--start-name", p.Name}
	p.cmd = exec.Command("nice", args...)
	p.cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+p.Home+"/config", "XDG_DATA_HOME="+p.Home+"/data", "SDL_VIDEODRIVER=dummy")
	p.cmd.Stdout, p.cmd.Stderr = lf, lf
	p.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := p.cmd.Start(); err != nil {
		return fail(err)
	}
	for deadline := time.Now().Add(60 * time.Second); time.Now().Before(deadline); time.Sleep(500 * time.Millisecond) {
		c, err := d2dctl.Dial(ctx, p.sock, d2dctl.Options{Timeout: 3 * time.Second})
		if err == nil {
			ok := c.Ping(ctx)
			c.Close()
			if ok == nil {
				time.Sleep(1500 * time.Millisecond) // the spike needed about this long after ping before `state` was stable
				return p.sock, nil
			}
		}
		if ctx.Err() != nil {
			break
		}
	}
	p.Stop()
	return "", fmt.Errorf("d2d did not answer ping within 60 s (see %s)", logPath)
}

// Stop asks d2d to quit, then terminates its process group, and always releases the lock. It returns an error if a
// d2d of ours is still alive afterwards.
func (p *procLab) Stop() error {
	if p.cmd == nil {
		return nil
	}
	defer func() {
		if p.lock != nil {
			p.lock.release()
			p.lock = nil
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if c, err := d2dctl.Dial(ctx, p.sock, d2dctl.Options{Timeout: 3 * time.Second, NoReconnect: true}); err == nil {
		c.Quit(ctx)
		c.Close()
	}
	cancel()
	done := make(chan struct{})
	go func() { p.cmd.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(8 * time.Second):
		syscall.Kill(-p.cmd.Process.Pid, syscall.SIGTERM)
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
			<-done
		}
	}
	syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL) // stragglers of the group (nice's child), harmless if none
	pid := p.cmd.Process.Pid
	p.cmd = nil
	os.Remove(p.sock)
	if out, _ := exec.Command("pgrep", "-x", "d2d").Output(); len(strings.TrimSpace(string(out))) > 0 {
		return fmt.Errorf("a d2d is still running after stop (pgrep: %s; ours was the group of %d): check by hand", strings.Fields(string(out))[0], pid)
	}
	return nil
}

func loadAvg() float64 {
	b, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0
	}
	f, _ := strconv.ParseFloat(strings.Fields(string(b))[0], 64)
	return f
}

func freeGB(path string) float64 {
	var st syscall.Statfs_t
	if syscall.Statfs(path, &st) != nil {
		return 1e9
	}
	return float64(uint64(st.Bavail)*uint64(st.Bsize)) / (1 << 30)
}

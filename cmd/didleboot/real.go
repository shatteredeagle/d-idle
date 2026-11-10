package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type realClock struct{}

func (realClock) Now() time.Time        { return time.Now() }
func (realClock) Sleep(d time.Duration) { time.Sleep(d) }

type osChild struct {
	cmd  *exec.Cmd
	mu   sync.Mutex
	done bool
	code int
}

func (c *osChild) Exited() (int, bool) { c.mu.Lock(); defer c.mu.Unlock(); return c.code, c.done }
func (c *osChild) Terminate()          { c.cmd.Process.Signal(syscall.SIGTERM) }
func (c *osChild) Kill()               { c.cmd.Process.Kill() }

type osRunner struct {
	args []string
	env  []string
}

// Start runs the release's launcher with the release's binaries first on PATH. The launcher is what the image's
// entrypoint was before the shim existed (docs/d2d-container/README.md), so the modes and env are unchanged.
func (r osRunner) Start(rel Release) (Child, error) {
	cmd := exec.Command(rel.Entry, r.args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	path := rel.BinDir + string(os.PathListSeparator) + rel.D2DDir + string(os.PathListSeparator) + os.Getenv("PATH")
	cmd.Env = append(append([]string{}, r.env...), "PATH="+path, "DIDLE_RELEASE="+rel.Version, "DIDLE_RELEASE_DIR="+rel.BinDir)
	// a bot-only release (dev loop) carries no d2d: keep the baked one
	if _, err := os.Stat(filepath.Join(rel.D2DDir, "d2d")); err == nil {
		cmd.Env = append(cmd.Env, "D2D_BIN="+rel.D2DDir+"/d2d")
	}
	if rel.Base {
		// the baked layout: binaries in /opt/d2lod, d2d in /opt/d2d (PATH of the image already has both)
		cmd.Env = append(append([]string{}, r.env...), "DIDLE_RELEASE="+rel.Version)
	}
	setOwnGroup(cmd) // its own group, so the leftovers can be swept (proc_unix.go)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	c := &osChild{cmd: cmd}
	go func() {
		err := cmd.Wait()
		sweepGroup(cmd.Process.Pid, 5*time.Second)
		c.mu.Lock()
		c.done = true
		switch {
		case err == nil:
			c.code = 0
		default:
			if ee, ok := err.(*exec.ExitError); ok {
				c.code = ee.ExitCode()
				if c.code < 0 { // killed by a signal
					c.code = 128 + int(ee.Sys().(syscall.WaitStatus).Signal())
				}
			} else {
				c.code = 1
			}
		}
		c.mu.Unlock()
	}()
	return c, nil
}

// osProbes: the bot API on loopback, the d2d counter through the tap (d2dping), the watch screen.
type osProbes struct {
	botAddr, watchAddr string
	pingBin, sock      string
	tickVerb           string
	hc                 *http.Client
}

func newProbes(botAddr, watchAddr, pingBin, sock string) *osProbes {
	return &osProbes{botAddr: botAddr, watchAddr: watchAddr, pingBin: pingBin, sock: sock, tickVerb: "clock", hc: &http.Client{Timeout: 3 * time.Second}}
}

func (p *osProbes) http(addr, path string) bool {
	resp, err := p.hc.Get("http://" + addr + path)
	if err != nil {
		return false
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return resp.StatusCode < 500 // a 401 from the token gate still means "answering"
}

func (p *osProbes) BotAPI() bool { return p.http(p.botAddr, "/healthz") }
func (p *osProbes) Watch() bool  { return p.http(p.watchAddr, "/") }

var tickRE = regexp.MustCompile(`\b(?:tick|frame)=(\d+)`)

// Tick reads the counter with `d2dping clock`. UNVERIFIED in realtime mode: the verb is documented for the lockstep
// fork; the drill checklist (docs/self-update-drill.md) says to confirm what the realtime build answers.
func (p *osProbes) Tick() (uint64, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, p.pingBin, "-sock", p.sock+".view", p.tickVerb).Output()
	if err != nil {
		return 0, false
	}
	m := tickRE.FindSubmatch(out)
	if m == nil {
		return 0, false
	}
	n, err := strconv.ParseUint(string(m[1]), 10, 64)
	return n, err == nil
}

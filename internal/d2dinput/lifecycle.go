package d2dinput

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/input"
)

// Spec describes the character to run.
type Spec struct {
	Name string
	// Class is d2d's UI-order class index for a NEW character (V: 1 is a Necromancer, 2 a Paladin; not the d2s
	// byte). -1 loads the saved character Name from the character select screen.
	Class      int
	Hardcore   bool
	Classic    bool
	Seed       int64 // 0: not passed (random map layout)
	NoSave     bool
	Difficulty int // only 0 (Normal): d2d has no start flag, "debug difficulty" is a cheat
}

// Lifecycle starts and stops one game. Everything here is V from docs/d2d-spike.md section 3 and
// docs/d2d-probes.md unless a comment says otherwise.
type Lifecycle interface {
	// Start launches the process and waits until the control socket answers "ping".
	Start(ctx context.Context, s Spec) error
	// WaitInGame waits until "state" reports screen=ingame, then sends the priming click.
	WaitInGame(ctx context.Context, timeout time.Duration) error
	// SaveExit writes the character ("save") and ends the process ("quit").
	SaveExit(ctx context.Context) error
	// Restart is SaveExit plus Start plus WaitInGame for the same or another spec.
	Restart(ctx context.Context, s Spec, timeout time.Duration) error
	// Stop ends the process without saving.
	Stop() error
	// Controller returns the input side once started.
	Controller() *Controller
}

// ErrNoBinary: D2D_BIN is not set. The fork binary is private and never lives in our repo.
var ErrNoBinary = errors.New("d2dinput: D2D_BIN not set (path of the private d2d fork binary)")

// Process is the os/exec Lifecycle.
type Process struct {
	Bin  string // d2d binary; default $D2D_BIN
	Data string // game data (MPQ) dir passed as --data; default $D2D_DATA, omitted when empty
	Sock string // control socket path (owner-only dir recommended)
	// ConfigDir and DataDir become XDG_CONFIG_HOME / XDG_DATA_HOME: saves, config and the log live there (V),
	// which isolates an instance.
	ConfigDir, DataDir string
	Nice               int      // >0 runs under nice -n (the probes used 10)
	ExtraArgs          []string // appended verbatim
	Log                *os.File // stdout/stderr of d2d; nil: discarded

	mu   sync.Mutex
	cmd  *exec.Cmd
	done chan struct{}
	c    *d2dctl.Client
	ctl  *Controller
}

var _ Lifecycle = (*Process)(nil)

// Args builds the command line for s (exported for tests and dry runs).
func (p *Process) Args(s Spec) ([]string, error) {
	if s.Name == "" {
		return nil, errors.New("d2dinput: empty character name")
	}
	if s.Difficulty != 0 {
		return nil, fmt.Errorf("%w: starting in difficulty %d (no flag; only the cheat verb exists)", input.ErrUnsupported, s.Difficulty)
	}
	data := p.Data
	if data == "" {
		data = os.Getenv("D2D_DATA")
	}
	var a []string
	if data != "" {
		a = append(a, "--data", data)
	}
	a = append(a, "--headless", "--devctl", p.Sock, "--no-video")
	if s.Seed != 0 {
		a = append(a, "--seed", strconv.FormatInt(s.Seed, 10))
	}
	if s.NoSave {
		a = append(a, "--no-save")
	}
	if s.Class >= 0 {
		a = append(a, "--start-screen", "ingame", "--start-class", strconv.Itoa(s.Class), "--start-name", s.Name)
		if s.Hardcore {
			a = append(a, "--start-hardcore")
		}
		if s.Classic {
			a = append(a, "--start-classic")
		}
	} else {
		a = append(a, "--start-screen", "charselect", "--start-name", s.Name)
	}
	return append(a, p.ExtraArgs...), nil
}

func (p *Process) Start(ctx context.Context, s Spec) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cmd != nil {
		return errors.New("d2dinput: already started")
	}
	bin := p.Bin
	if bin == "" {
		bin = os.Getenv("D2D_BIN")
	}
	if bin == "" {
		return ErrNoBinary
	}
	if p.Sock == "" {
		return errors.New("d2dinput: Sock not set")
	}
	args, err := p.Args(s)
	if err != nil {
		return err
	}
	name := bin
	if p.Nice > 0 {
		name, args = "nice", append([]string{"-n", strconv.Itoa(p.Nice), bin}, args...)
	}
	_ = os.Remove(p.Sock)
	cmd := exec.Command(name, args...)
	cmd.Env = os.Environ()
	if p.ConfigDir != "" {
		cmd.Env = append(cmd.Env, "XDG_CONFIG_HOME="+p.ConfigDir)
	}
	if p.DataDir != "" {
		cmd.Env = append(cmd.Env, "XDG_DATA_HOME="+p.DataDir)
	}
	if p.Log != nil {
		cmd.Stdout, cmd.Stderr = p.Log, p.Log
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("d2dinput: start d2d: %w", err)
	}
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	p.cmd, p.done = cmd, done

	c := d2dctl.New(p.Sock, d2dctl.Options{Timeout: 5 * time.Second})
	for {
		select {
		case <-done:
			p.cmd = nil
			return errors.New("d2dinput: d2d exited during startup")
		case <-ctx.Done():
			p.killLocked()
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
		if _, err := os.Stat(p.Sock); err != nil {
			continue
		}
		if c.Ping(ctx) == nil {
			break
		}
	}
	p.c = c
	p.ctl = New(c)
	p.ctl.Ctx = ctx
	// A character at the select screen is loaded with Return (V, about 6 s).
	if s.Class < 0 {
		if err := c.Key(ctx, "Return"); err != nil {
			p.killLocked()
			return err
		}
	}
	return nil
}

func (p *Process) Controller() *Controller { return p.ctl }

// WaitInGame polls "state" until screen=ingame, then Kick (V: needed after a save load).
func (p *Process) WaitInGame(ctx context.Context, timeout time.Duration) error {
	p.mu.Lock()
	c := p.c
	p.mu.Unlock()
	if c == nil {
		return errors.New("d2dinput: not started")
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		st, _, err := c.State(ctx)
		if err == nil && st.Screen == "ingame" {
			return c.Kick(ctx)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("d2dinput: not in game after %v: %w", timeout, ctx.Err())
		case <-p.done:
			return errors.New("d2dinput: d2d exited")
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// SaveExit sends "save" then "quit" and waits for the process to go (SIGTERM after 10 s).
func (p *Process) SaveExit(ctx context.Context) error {
	p.mu.Lock()
	c, done := p.c, p.done
	p.mu.Unlock()
	if c == nil {
		return errors.New("d2dinput: not started")
	}
	serr := c.Save(ctx)
	qerr := c.Quit(ctx)
	if err := p.wait(done, 10*time.Second); err != nil && qerr == nil {
		qerr = err
	}
	return errors.Join(serr, qerr)
}

func (p *Process) Restart(ctx context.Context, s Spec, timeout time.Duration) error {
	if p.c != nil {
		if err := p.SaveExit(ctx); err != nil {
			return err
		}
	}
	if err := p.Start(ctx, s); err != nil {
		return err
	}
	return p.WaitInGame(ctx, timeout)
}

func (p *Process) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cmd == nil {
		return nil
	}
	p.killLocked()
	return nil
}

// wait expects the process to leave by itself, else SIGTERM, then clears state.
func (p *Process) wait(done chan struct{}, d time.Duration) error {
	var err error
	select {
	case <-done:
	case <-time.After(d):
		p.mu.Lock()
		if p.cmd != nil {
			_ = p.cmd.Process.Signal(os.Interrupt)
		}
		p.mu.Unlock()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
		err = errors.New("d2dinput: d2d did not quit by itself")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.reset()
	return err
}

func (p *Process) killLocked() {
	if p.cmd != nil {
		_ = p.cmd.Process.Kill()
		<-p.done
	}
	p.reset()
}

func (p *Process) reset() {
	if p.c != nil {
		p.c.Close()
	}
	p.cmd, p.c, p.ctl = nil, nil, nil
	if p.Sock != "" {
		_ = os.Remove(filepath.Clean(p.Sock))
	}
}

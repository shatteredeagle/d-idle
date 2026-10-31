package d2denv

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dinput"
	"d2lod-bot/internal/input"
)

// Proc starts, watches and stops one d2d process behind a STABLE d2dctl.Client: the client dials lazily and redials
// after a transport error, so the Reader, the Controller and the world source built on it survive a restart. (This is
// why it does not use d2dinput.Process, which makes a new client per start; only its Args builder is reused.)
//
// Attach mode (Bin == ""): the d2d belongs to somebody else. Start only waits for the socket, SaveExit only sends "save"
// and nothing is ever killed. Tags: the flags are V (docs/d2d-spike.md section 3), the restart flow is V offline only.
type Proc struct {
	C         *d2dctl.Client
	Bin       string // d2d binary; "" = attach
	Data      string // --data (MPQ dir), "" = d2d's default
	Sock      string
	Home      string // XDG_CONFIG_HOME and XDG_DATA_HOME become <Home>/config and <Home>/data (own saves and log)
	Nice      int
	NoSave    bool     // d2d runs with --no-save: "save" is refused there (V live: err no character store), so it is not sent
	ExtraArgs []string // appended verbatim: --toggle, --lockstep, ...
	Log       *os.File
	// LogPath, when set, is where d2d's output goes: every start renames the previous file to <path>.<mtime> (the
	// newest keepLogs are kept) and writes a fresh one, so the log of a d2d that died is still there after its
	// restart (it used to be appended to one file, D2DPACE could not tell the instances apart).
	LogPath string
	// Logf receives the one-line diagnosis of a channel that stopped answering (nil: silent).
	Logf     func(string, ...any)
	OnInGame func() // called when a (re)started game is in game: the run toggle starts at walking (d2dinput.ResetRun)
	lastDiag string

	mu   sync.Mutex
	exit string // how the last d2d ended ("signal: segmentation fault", "exit status 1")
	cmd  *exec.Cmd
	done chan struct{}
}

// Managed reports whether this process starts d2d itself.
func (p *Proc) Managed() bool { return p.Bin != "" }

// PID is the d2d process id, 0 when unknown (attach mode or not running).
func (p *Proc) PID() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Pid
	}
	return 0
}

// Alive answers "does the channel respond", the health probe for both modes.
func (p *Proc) Alive(ctx context.Context) bool {
	// A request in flight (a long lockstep step) holds the single command stream: that d2d is working, and queueing a
	// ping behind it only slows the bot.
	if p.C.Busy() {
		return true
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	err := p.C.Ping(ctx)
	if err != nil {
		p.diagnose(err)
	} else {
		p.mu.Lock()
		p.lastDiag = ""
		p.mu.Unlock()
	}
	return err == nil
}

// Diagnosis says WHY the channel does not answer, from what the OS knows: did the process we started exit (and how),
// is the socket file gone, is the process alive but silent. It is what D2DPACE lacked when the socket "vanished"
// (U cause): the exit state and the tail of d2d.out are in the log line.
func (p *Proc) Diagnosis(err error) string {
	var parts []string
	if !p.Managed() {
		parts = append(parts, "attached d2d")
	} else if p.Gone() {
		parts = append(parts, "d2d process exited: "+p.ExitState())
	} else if pid := p.PID(); pid != 0 {
		parts = append(parts, fmt.Sprintf("d2d process pid %d still running", pid))
	}
	if _, serr := os.Stat(p.Sock); serr != nil {
		parts = append(parts, "socket file is gone ("+serr.Error()+")")
	}
	parts = append(parts, "ping: "+err.Error())
	if t := p.logTail(600); t != "" {
		parts = append(parts, "d2d.out tail: "+t)
	}
	return strings.Join(parts, "; ")
}

func (p *Proc) diagnose(err error) {
	if p.Logf == nil {
		return
	}
	d := p.Diagnosis(err)
	p.mu.Lock()
	same := p.lastDiag == d
	p.lastDiag = d
	p.mu.Unlock()
	if !same {
		p.Logf("d2d channel not answering: %s", d)
	}
}

func (p *Proc) logTail(n int) string {
	if p.LogPath == "" {
		return ""
	}
	f, err := os.Open(p.LogPath)
	if err != nil {
		return ""
	}
	defer f.Close()
	if st, err := f.Stat(); err == nil && st.Size() > int64(n) {
		_, _ = f.Seek(-int64(n), io.SeekEnd)
	}
	b, _ := io.ReadAll(f)
	return strings.Join(strings.Fields(strings.ReplaceAll(string(b), "\n", " | ")), " ")
}

const keepLogs = 24 // a restart storm (the ladder restarts every 2 s) must not evict the log of the first instance

// rotateLog moves the current LogPath aside and opens a fresh one.
func (p *Proc) rotateLog() {
	if p.LogPath == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(p.LogPath), 0o700)
	if st, err := os.Stat(p.LogPath); err == nil && st.Size() > 2000 { // a d2d that died at once leaves nothing worth keeping
		_ = os.Rename(p.LogPath, p.LogPath+"."+st.ModTime().Format("20060102-150405"))
	}
	if old, _ := filepath.Glob(p.LogPath + ".*"); len(old) > keepLogs { // names sort by time
		sort.Strings(old)
		for _, f := range old[:len(old)-keepLogs] {
			_ = os.Remove(f)
		}
	}
	if p.Log != nil {
		_ = p.Log.Close()
		p.Log = nil
	}
	if f, err := os.OpenFile(p.LogPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600); err == nil {
		p.Log = f
	}
}

// Start launches d2d for s (managed) and waits until ping answers.
func (p *Proc) Start(ctx context.Context, s d2dinput.Spec) error {
	if p.Managed() {
		// A restart already launched d2d (LeaveGame with Reenter) while the runner saw the menus and entered again: wait for
		// that process like EnterGame promises instead of failing five times. U: inferred from the log and the code, no live run.
		if err := p.launch(s); err != nil && !errors.Is(err, errAlreadyRunning) {
			return err
		}
	}
	for {
		if p.Alive(ctx) {
			break
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("d2denv: no d2d on %s: %w", p.Sock, ctx.Err())
		case <-p.exited():
			return errors.New("d2denv: d2d exited during startup")
		case <-time.After(250 * time.Millisecond):
		}
	}
	// A saved character waits at the select screen (V): Return loads it, about 6 s.
	if p.Managed() && s.Class < 0 {
		return p.C.Key(ctx, "Return")
	}
	return nil
}

func (p *Proc) exited() <-chan struct{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.done == nil {
		return nil // never fires
	}
	return p.done
}

// ExitState is how the last d2d this Proc started ended ("" while running or never started).
func (p *Proc) ExitState() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exit
}

// Gone reports that a d2d this Proc started has exited (never true in attach mode or before a start).
func (p *Proc) Gone() bool {
	select {
	case <-p.exited():
		return true
	default:
		return false
	}
}

var errAlreadyRunning = errors.New("d2denv: d2d already running")

func (p *Proc) launch(s d2dinput.Spec) error {
	args, err := (&d2dinput.Process{Data: p.Data, Sock: p.Sock, ExtraArgs: p.ExtraArgs}).Args(s)
	if err != nil {
		return err
	}
	name := p.Bin
	if p.Nice > 0 {
		name, args = "nice", append([]string{"-n", strconv.Itoa(p.Nice), p.Bin}, args...)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cmd != nil {
		return errAlreadyRunning
	}
	_ = os.Remove(p.Sock)
	p.rotateLog()
	cmd := exec.Command(name, args...)
	cmd.Env = os.Environ()
	if p.Home != "" {
		for _, d := range []string{"config", "data"} {
			if err := os.MkdirAll(p.Home+"/"+d, 0o700); err != nil {
				return err
			}
		}
		cmd.Env = append(cmd.Env, "XDG_CONFIG_HOME="+p.Home+"/config", "XDG_DATA_HOME="+p.Home+"/data")
	}
	if p.Log != nil {
		cmd.Stdout, cmd.Stderr = p.Log, p.Log
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("d2denv: start d2d: %w", err)
	}
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		p.mu.Lock()
		p.exit = cmd.ProcessState.String()
		p.mu.Unlock()
		close(done)
	}()
	p.cmd, p.done = cmd, done
	return nil
}

// WaitInGame polls "state" until screen=ingame, then sends the priming click (V: needed after a load).
func (p *Proc) WaitInGame(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		if st, _, err := p.C.State(ctx); err == nil && st.Screen == "ingame" {
			if p.OnInGame != nil {
				p.OnInGame()
			}
			return p.C.Kick(ctx)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("d2denv: not in game after %v: %w", timeout, ctx.Err())
		case <-p.exited():
			return errors.New("d2denv: d2d exited")
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func (p *Proc) logf(f string, a ...any) {
	if p.Logf != nil {
		p.Logf(f, a...)
	}
}

// SaveExit is the d2d "Save and Exit": "save", then (managed) "quit" and wait for the process to go (interrupt after
// 10 s, kill after 3 more). Attach mode only saves.
func (p *Proc) SaveExit(ctx context.Context) error {
	var serr error
	if !p.NoSave {
		serr = p.C.Save(ctx)
	}
	if !p.Managed() {
		return serr
	}
	// This is the "socket vanished" of D2DPACE (V live, D2DSPEED): combat's Chicken calls Leave = LeaveGame, which quits
	// a managed d2d; the socket file goes with the process and nothing restarts it until the runner notices "exited".
	if p.Logf != nil {
		p.Logf("d2d: Save and Exit requested (chicken / leave game): the process quits and its socket goes away")
	}
	qerr := p.C.Quit(ctx)
	return errors.Join(serr, p.reap(10*time.Second, qerr == nil))
}

// Stop ends a managed d2d without saving; attach mode does nothing.
func (p *Proc) Stop() error {
	if !p.Managed() {
		return nil
	}
	p.mu.Lock()
	cmd, done := p.cmd, p.done
	p.mu.Unlock()
	if cmd == nil {
		return nil
	}
	if p.Logf != nil {
		p.Logf("d2d: stopped by the bot (kill)")
	}
	_ = cmd.Process.Kill()
	<-done
	p.clear()
	return nil
}

func (p *Proc) reap(wait time.Duration, complain bool) error {
	p.mu.Lock()
	cmd, done := p.cmd, p.done
	p.mu.Unlock()
	if cmd == nil {
		return nil
	}
	var err error
	select {
	case <-done:
	case <-time.After(wait):
		_ = cmd.Process.Signal(os.Interrupt)
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
		err = errors.New("d2denv: d2d did not quit by itself")
	}
	p.clear()
	if !complain {
		return nil
	}
	return err
}

func (p *Proc) clear() {
	p.mu.Lock()
	p.cmd, p.done = nil, nil
	p.mu.Unlock()
	_ = os.Remove(p.Sock)
}

// Cycler is the bot.Cycler of the d2d backend and the combat Leave hook: leaving a game is "Save and Exit", entering
// one is a start of the process (a character on disk is then loaded from the select screen, V).
type Cycler struct {
	P *Proc
	// Spec is the first start; once a game ran with saving on, later starts load the saved character (Class -1).
	Spec    d2dinput.Spec
	Timeout time.Duration // WaitInGame (0 = 90 s)
	OnCycle func()        // told about every successful (re-)entry
	// Reenter starts a managed d2d again right after LeaveGame (a chicken exit must not leave the bot without a game).
	Reenter bool

	mu      sync.Mutex
	started bool

	leaveMu sync.Mutex
	leaves  atomic.Int64
}

func (c *Cycler) timeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return 90 * time.Second
}

func (c *Cycler) spec() d2dinput.Spec {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.Spec
	if c.started && !s.NoSave {
		s.Class = -1
	}
	return s
}

func okDifficulty(diff string) error {
	if diff == "" || diff == "normal" {
		return nil
	}
	return fmt.Errorf("%w: d2d has no start flag for difficulty %q", input.ErrUnsupported, diff)
}

// EnterGame starts d2d (or waits for the one that runs) and returns once the character stands in a game.
func (c *Cycler) EnterGame(ctx context.Context, diff string) error {
	if err := okDifficulty(diff); err != nil {
		return err
	}
	if st, _, err := c.P.C.State(ctx); err == nil && st.Screen == "ingame" {
		return nil
	}
	s := c.spec()
	if err := c.P.Start(ctx, s); err != nil {
		return err
	}
	if err := c.P.WaitInGame(ctx, c.timeout()); err != nil {
		return err
	}
	boost(ctx, c.P.C)
	c.mu.Lock()
	c.started = true
	c.mu.Unlock()
	if c.OnCycle != nil {
		c.OnCycle()
	}
	return nil
}

// LeaveGame is Save and Exit.
//
// With Reenter set (the backend does) a managed d2d is started again at once: combat's Chicken calls this, and leaving
// the d2d process dead made the runner notice "exited" only about 90 s later (D2DSPEED, V live: socket file gone).
func (c *Cycler) LeaveGame(ctx context.Context) error {
	gen := c.leaves.Load()
	c.leaveMu.Lock()
	defer c.leaveMu.Unlock()
	if c.leaves.Load() != gen {
		// A second leave (chicken and safety both fired) that waited behind the first: that one already restarted the game.
		// Without this the two SaveExit/EnterGame pairs raced, the new d2d was quit again and the bot halted ("restart failed
		// twice", D2DLOOP r5, V live).
		return nil
	}
	defer c.leaves.Add(1)
	reenter := c.Reenter && c.P.Managed()
	if reenter {
		// The restart must run to the end: combat's chicken passes the fight's context, which the runner cancels as soon as
		// the state changes. A restart cut off after the process was launched left a d2d without a socket, the next
		// leave failed on its dead socket and the bot halted ("restart failed twice", D2DLOOP r23, V live log).
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.WithoutCancel(ctx), 3*c.timeout())
		defer cancel()
	}
	if err := c.P.SaveExit(ctx); err != nil {
		c.P.logf("d2d: leave: save and exit failed: %v", err)
		if !reenter { // SaveExit reaps the process even when save or quit failed, so a restart can go on
			return err
		}
	}
	if reenter {
		err := c.EnterGame(ctx, "")
		c.P.logf("d2d: leave: re-entered the game (err %v)", err)
		return err
	}
	return nil
}

// NewGame leaves and enters a fresh game. Attach mode cannot restart somebody else's d2d: it only reports that.
func (c *Cycler) NewGame(ctx context.Context, diff string) error {
	if err := okDifficulty(diff); err != nil {
		return err
	}
	if !c.P.Managed() {
		return fmt.Errorf("%w: attached d2d cannot be restarted", input.ErrUnsupported)
	}
	if err := c.LeaveGame(ctx); err != nil {
		return err
	}
	return c.EnterGame(ctx, diff)
}

// boost applies $D2D_BOOST, a list "stat=value,..." of "debug stat" settings, to a fresh character: the lab's way to get
// a clvl 25 character with a big life pool (stats 6 and 7 are life and max life as 8.8 fixed point, 12 is the level)
// so speed sessions are not cut short by a clvl 1 death. A private-fork, throwaway-character tool; unset = nothing.
func boost(ctx context.Context, c *d2dctl.Client) {
	for _, kv := range strings.Split(os.Getenv("D2D_BOOST"), ",") {
		a, b, ok := strings.Cut(kv, "=")
		id, e1 := strconv.Atoi(strings.TrimSpace(a))
		v, e2 := strconv.Atoi(strings.TrimSpace(b))
		if ok && e1 == nil && e2 == nil {
			_ = c.SetStat(ctx, id, v)
		}
	}
}

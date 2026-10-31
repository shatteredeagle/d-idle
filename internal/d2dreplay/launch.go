package d2dreplay

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Launch starts a hidden d2d from a replay header's recipe (flags V in docs/d2d-spike.md section 3 and
// internal/d2dinput/lifecycle.go Args). Saves and config go to a private home so nothing of the user's is touched.
type Launch struct {
	Bin    string // d2d binary (the lab build)
	Data   string // --data MPQ dir
	Sock   string // --devctl socket, private dir
	Home   string // XDG_CONFIG_HOME and XDG_DATA_HOME become <Home>/config and <Home>/data
	Nice   int
	Build  Build   // non-path args from the header
	Recipe *Recipe // fresh character
	Log    io.Writer
	Wait   time.Duration // time to wait for "ping" (0: 60 s)
}

// Args builds the command line.
func (l *Launch) Args() ([]string, error) {
	if l.Recipe == nil {
		return nil, errors.New("d2dreplay: no start recipe in the replay (save-based replays need --save, not implemented)")
	}
	if err := l.Recipe.validate(); err != nil {
		return nil, err
	}
	extra, err := SafeArgs(l.Build.Args)
	if err != nil {
		return nil, err
	}
	a := []string{}
	if l.Data != "" {
		a = append(a, "--data", l.Data)
	}
	a = append(a, "--headless", "--devctl", l.Sock, "--no-video", "--seed", strconv.FormatInt(l.Recipe.Seed, 10))
	screen := l.Recipe.Screen
	if screen == "" {
		screen = "ingame"
	}
	a = append(a, "--start-screen", screen, "--start-class", strconv.Itoa(l.Recipe.Class), "--start-name", l.Recipe.Name)
	if l.Recipe.Hardcore {
		a = append(a, "--start-hardcore")
	}
	if l.Recipe.Classic {
		a = append(a, "--start-classic")
	}
	a = append(a, extra...)
	return a, nil
}

var (
	nameRE   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{1,15}$`)
	toggleRE = regexp.MustCompile(`^[a-z][a-z0-9_]*=(on|off)(,[a-z][a-z0-9_]*=(on|off))*$`)
)

func (r *Recipe) validate() error {
	if !nameRE.MatchString(r.Name) {
		return fmt.Errorf("d2dreplay: recipe name %q is not 2-16 letters/digits/underscore", r.Name)
	}
	if r.Class < 0 || r.Class > 7 {
		return fmt.Errorf("d2dreplay: recipe class %d out of range", r.Class)
	}
	if r.Screen != "" && r.Screen != "ingame" {
		return fmt.Errorf("d2dreplay: recipe screen %q not supported", r.Screen)
	}
	return nil
}

// SafeArgs allow-lists the d2d arguments of a replay header (untrusted: it must never carry --devctl, --data or
// anything else that reaches the shared fork): --lockstep, --no-save and --toggle name=on|off,... only.
func SafeArgs(args []string) ([]string, error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--lockstep", "--no-save":
		case "--toggle":
			if i+1 >= len(args) || !toggleRE.MatchString(args[i+1]) {
				return nil, errors.New("d2dreplay: header d2d args: bad --toggle value")
			}
			i++
		default:
			return nil, fmt.Errorf("d2dreplay: header d2d arg %q is not allowed (only --lockstep, --no-save, --toggle)", args[i])
		}
	}
	return args, nil
}

// Proc is a running d2d.
type Proc struct {
	Cmd  *exec.Cmd
	Conn *Conn
	done chan error
}

// Start launches d2d and waits until the socket answers ping.
func (l *Launch) Start(ctx context.Context) (*Proc, error) {
	args, err := l.Args()
	if err != nil {
		return nil, err
	}
	if l.Home != "" {
		for _, d := range []string{"config", "data"} {
			if err := os.MkdirAll(filepath.Join(l.Home, d), 0o700); err != nil {
				return nil, err
			}
		}
	}
	os.Remove(l.Sock)
	var cmd *exec.Cmd
	if l.Nice > 0 {
		cmd = exec.Command("nice", append([]string{"-n", strconv.Itoa(l.Nice), l.Bin}, args...)...)
	} else {
		cmd = exec.Command(l.Bin, args...)
	}
	cmd.Env = os.Environ()
	if l.Home != "" {
		cmd.Env = append(cmd.Env, "XDG_CONFIG_HOME="+filepath.Join(l.Home, "config"), "XDG_DATA_HOME="+filepath.Join(l.Home, "data"))
	}
	cmd.Stdout, cmd.Stderr = l.Log, l.Log
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	p := &Proc{Cmd: cmd, done: make(chan error, 1)}
	go func() { p.done <- cmd.Wait() }()
	wait := l.Wait
	if wait == 0 {
		wait = 60 * time.Second
	}
	wctx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	c, err := DialWait(wctx, l.Sock, 120*time.Second)
	if err != nil {
		p.Kill()
		return nil, fmt.Errorf("d2dreplay: d2d did not come up: %w", err)
	}
	p.Conn = c
	return p, nil
}

// Stop asks d2d to quit and kills it after 5 s; it reports whether d2d is gone (the caller must not release the
// slot while it is still running).
func (p *Proc) Stop() bool {
	if p.Conn != nil {
		p.Conn.Timeout = 5 * time.Second
		p.Conn.Do("quit")
		p.Conn.Close()
	}
	select {
	case err := <-p.done:
		p.done <- err
		return true
	case <-time.After(5 * time.Second):
		return p.Kill()
	}
}

// Kill terminates d2d at once and waits for it to exit (up to 10 s); it reports whether it is gone.
func (p *Proc) Kill() bool {
	if p.Cmd.Process != nil {
		p.Cmd.Process.Kill()
	}
	select {
	case err := <-p.done:
		p.done <- err // keep the result for later waiters
		return true
	case <-time.After(10 * time.Second):
		return false
	}
}

// WaitReady polls "state" until the screen is in-game or the timeout passes: with --start-screen ingame the
// level still loads on real time (docs/d2d-fork.md section 3), so the first verbs must not race it.
func WaitReady(ctx context.Context, d Doer, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	last := ""
	for time.Now().Before(deadline) {
		lines, status, err := d.Do("state")
		if err != nil {
			return last, err
		}
		if !IsErr(status) {
			last = status
			if len(lines) > 0 {
				last = lines[0]
			}
			for _, kv := range strings.Fields(last) {
				if kv == "screen=ingame" || kv == "screen=game" {
					return last, nil
				}
			}
		}
		select {
		case <-ctx.Done():
			return last, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	return last, nil
}

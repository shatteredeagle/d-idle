package d2dreplay

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// Tap is the passive recorder: it owns the only connection to d2d and serves one downstream client (the bot or a
// script) on its own socket. Every line is forwarded unchanged and answered with d2d's reply; mutating verbs are
// written to the replay with the tick before them; after a "step" that crosses a checkpoint boundary the tap reads
// the fingerprint itself (read verbs only, between two client commands). Nothing is injected into the client's
// stream and no line is rewritten (spec: Passive recording).
type Tap struct {
	Upstream string // d2d's socket
	Listen   string // our socket for the client
	W        *Writer
	Every    int           // checkpoint interval in ticks
	Idle     time.Duration // stop when no client has been connected for this long (0: never)
	Timeout  time.Duration // one upstream round trip (steps can be long)
	Logf     func(string, ...any)
	// Redial bounds how long the tap retries the upstream after it dropped (a managed bot restarts d2d on the
	// same socket within its link wait, docs/d2d-view.md).
	Redial time.Duration

	mu      sync.Mutex
	up      *Conn
	tick    int
	nextCP  int
	verbs   int
	cps     int
	restart int
	last    time.Time
	start   time.Time
	dropped bool
}

// Stats is a snapshot for the CLI.
type Stats struct {
	Ticks, Verbs, Checkpoints, Restarts int
	Wall                                time.Duration
}

func (t *Tap) logf(f string, a ...any) {
	if t.Logf != nil {
		t.Logf(f, a...)
	}
}

// Stats returns the counters so far.
func (t *Tap) Stats() Stats {
	t.mu.Lock()
	defer t.mu.Unlock()
	return Stats{t.tick, t.verbs, t.cps, t.restart, time.Since(t.start)}
}

// upstream returns the live upstream connection, redialing after a drop (and recording a restart marker).
func (t *Tap) upstream() (*Conn, error) {
	if t.up != nil {
		return t.up, nil
	}
	wait := t.Redial
	if wait == 0 {
		wait = 90 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	c, err := DialWait(ctx, t.Upstream, t.Timeout)
	if err != nil {
		return nil, err
	}
	t.up = c
	if t.dropped {
		t.restart++
		t.W.Add(Entry{R: true, T: t.tick})
		t.W.Flush()
		t.logf("tap: upstream d2d replaced at tick %d (restart %d)", t.tick, t.restart)
	}
	return c, nil
}

// exec forwards one client line and records it. Caller holds t.mu.
func (t *Tap) exec(line string) (lines []string, status string) {
	up, err := t.upstream()
	if err != nil {
		return nil, "err replay: upstream d2d not reachable"
	}
	now := time.Now()
	w := int(now.Sub(t.last) / time.Millisecond)
	t.last = now
	lines, status, err = up.Do(line)
	if err != nil {
		t.logf("tap: upstream error on %q: %v", line, err)
		up.Close()
		t.up, t.dropped = nil, true
		return nil, "err replay: upstream d2d not reachable"
	}
	if !Mutating(line) {
		return lines, status
	}
	e := Entry{T: t.tick, Verb: line, W: w}
	if IsErr(status) {
		e.Err = status
	}
	t.W.Add(e)
	t.verbs++
	if n, ok := StepCount(line); ok && !IsErr(status) {
		t.tick += n
		if t.Every > 0 && t.tick >= t.nextCP {
			fp, err := Checkpoint(up)
			if err != nil {
				t.logf("tap: checkpoint at tick %d failed: %v", t.tick, err)
			} else {
				t.W.Add(Entry{C: t.tick, H: fp.Hash, Lv: fp.Lv, Area: fp.Area, Life: fp.Life})
				t.cps++
			}
			t.W.Flush()
			t.nextCP = (t.tick/t.Every + 1) * t.Every
		}
	}
	return lines, status
}

// Serve runs the tap until ctx ends or Idle elapses with no client (a "quit" from the client is a d2d restart, not
// the end: managed bots quit and relaunch d2d at chickens and save points). The file is closed with a trailer.
func (t *Tap) Serve(ctx context.Context) error {
	if t.Timeout == 0 {
		t.Timeout = 120 * time.Second
	}
	if t.Every <= 0 {
		t.Every = 250
	}
	t.nextCP = t.Every
	if t.Listen == t.Upstream {
		return errors.New("d2dreplay: -listen is the upstream socket")
	}
	// Only a stale socket is removed, never another kind of file.
	if st, err := os.Lstat(t.Listen); err == nil {
		if st.Mode()&os.ModeSocket == 0 {
			return fmt.Errorf("d2dreplay: %s exists and is not a socket", t.Listen)
		}
		os.Remove(t.Listen)
	}
	ln, err := net.Listen("unix", t.Listen)
	if err != nil {
		return err
	}
	os.Chmod(t.Listen, 0o600)
	defer os.Remove(t.Listen)
	// Dial upstream now (a managed bot may still be starting its d2d: exec retries later), so the first entry's
	// wall gap counts from a live d2d.
	t.mu.Lock()
	if _, err := t.upstream(); err != nil {
		t.logf("tap: upstream not up yet (%v), will retry on the first command", err)
	}
	t.start = time.Now()
	t.last = t.start
	t.mu.Unlock()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var (
		cmu        sync.Mutex // guards active, connected, lastClient
		active     net.Conn
		connected  bool
		lastClient time.Time
		wg         sync.WaitGroup
	)
	go func() {
		<-ctx.Done()
		ln.Close()
		cmu.Lock() // a client that stays attached must not keep the tap alive
		if active != nil {
			active.Close()
		}
		cmu.Unlock()
	}()
	if t.Idle > 0 {
		go func() {
			tk := time.NewTicker(time.Second)
			defer tk.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-tk.C:
					cmu.Lock()
					idle := !connected && !lastClient.IsZero() && time.Since(lastClient) > t.Idle
					cmu.Unlock()
					if idle {
						t.logf("tap: no client for %s, stopping", t.Idle)
						cancel()
						return
					}
				}
			}
		}()
	}
	var accErr error
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() == nil {
				accErr = err
			}
			break
		}
		cmu.Lock()
		if connected {
			cmu.Unlock()
			io.WriteString(conn, "err busy\n")
			conn.Close()
			continue
		}
		if ctx.Err() != nil {
			cmu.Unlock()
			conn.Close()
			break
		}
		connected, active = true, conn
		cmu.Unlock()
		wg.Add(1)
		go func() {
			defer wg.Done()
			t.serve(conn)
			cmu.Lock()
			connected, active, lastClient = false, nil, time.Now()
			cmu.Unlock()
		}()
	}
	cancel()
	wg.Wait()
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.up != nil {
		t.up.Close()
	}
	err = t.W.Close(&Entry{Ticks: t.tick, Verbs: t.verbs, WallMS: int(time.Since(t.start) / time.Millisecond)})
	if accErr != nil {
		return accErr
	}
	return err
}

// serve handles one client connection until it hangs up.
func (t *Tap) serve(conn net.Conn) {
	defer conn.Close()
	rd := bufio.NewReaderSize(conn, 1<<16)
	for {
		l, err := rd.ReadString('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) {
				t.logf("tap: client: %v", err)
			}
			return
		}
		line := strings.TrimRight(l, "\r\n")
		var lines []string
		var status string
		if err := CleanLine(line); err != nil {
			status = "err replay: " + strings.ReplaceAll(err.Error(), "\n", " ")
		} else {
			t.mu.Lock()
			lines, status = t.exec(line)
			t.mu.Unlock()
		}
		var b strings.Builder
		for _, x := range lines {
			b.WriteString(x)
			b.WriteByte('\n')
		}
		b.WriteString(status)
		b.WriteByte('\n')
		if _, err := io.WriteString(conn, b.String()); err != nil {
			return
		}
		if strings.TrimSpace(line) == "quit" && !IsErr(status) {
			// A managed bot quits d2d at a chicken or save point and starts a new one on the same socket: the
			// recording goes on (restart marker at the redial). The tap only stops on idle, -max or a signal.
			t.mu.Lock()
			if t.up != nil {
				t.up.Close()
				t.up, t.dropped = nil, true
			}
			t.mu.Unlock()
		}
	}
}

// String summarizes the counters.
func (s Stats) String() string {
	return fmt.Sprintf("ticks %d (virtual %s) verbs %d checkpoints %d restarts %d wall %s",
		s.Ticks, (time.Duration(s.Ticks) * TickMS * time.Millisecond).Round(time.Second), s.Verbs, s.Checkpoints, s.Restarts, s.Wall.Round(time.Second))
}

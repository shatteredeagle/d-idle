package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// d2d's control socket takes ONE client (docs/d2dctl.md), so a viewer cannot attach next to the bot. The Mux sits
// between them: it holds the only upstream connection and serves two private downstream sockets, one for the bot
// (everything passes) and one for the viewer (read verbs only, see readOnly). Each command is one locked round trip,
// so replies never interleave; the viewer waits on the lock, the bot never waits on the viewer for longer than one
// read verb (about 50 us on lab5, V docs/d2d-pace.md).
type Mux struct {
	upstream string
	// Pace and RenderEvery switch the watch mode (see Mode): they only touch "step" lines of the bot.
	mode atomic.Pointer[Mode]

	mu   sync.Mutex // one round trip at a time
	conn net.Conn
	rd   *bufio.Reader

	lastStep atomic.Int64 // wall unix nanos when the bot's last step reply came back
	ticks    atomic.Int64 // ticks the bot has stepped through the tap (count of the n of "step n")
	steps    atomic.Int64
	botLines atomic.Int64
	viewOps  atomic.Int64
	upErr    atomic.Value // string: last upstream problem

	botDown func() // called when a bot connection ends (tests)
}

// Mode is the pixel path mode. Fast: the bot's steps pass untouched, frames are sampled screenshots. Watch: the
// tap rewrites the bot's "step n" to "step n render" (d2d draws every stepped frame) and, when Speed > 0, sleeps after
// each step so virtual time runs at most Speed times real time.
type Mode struct {
	Watch bool
	Speed float64 // watch mode only: max virtual seconds per wall second (0: no limit, only render)
}

func NewMux(upstream string) *Mux {
	m := &Mux{upstream: upstream}
	m.mode.Store(&Mode{})
	return m
}

func (m *Mux) Mode() Mode     { return *m.mode.Load() }
func (m *Mux) SetMode(x Mode) { m.mode.Store(&x) }

// readOnly says whether a line may come from the viewer. Everything that moves, clicks, steps or edits is refused;
// "screenshot" is allowed only with a path under dir.
func readOnly(line, shotDir string) bool {
	if strings.ContainsFunc(line, func(r rune) bool { return r < 0x20 && r != '\t' && r != '\r' && r != '\n' || r == 0x7f }) {
		return false
	}
	f := strings.Fields(line)
	if len(f) == 0 {
		return false
	}
	switch f[0] {
	case "state", "monsters", "npcs", "ground", "info", "ping":
		return len(f) == 1
	case "clock":
		return len(f) == 1
	case "dump":
		return len(f) == 1 || (len(f) == 2 && (f[1] == "items" || f[1] == "player" || f[1] == "ui"))
	case "screenshot":
		return len(f) == 2 && strings.HasPrefix(f[1], shotDir+"/") && !strings.Contains(f[1], "..")
	case "debug":
		if len(f) < 2 {
			return false
		}
		switch f[1] {
		case "players", "level", "warps", "portals", "presets", "objects", "corpses", "layout", "nearby":
			return len(f) == 2
		case "grid":
			return len(f) == 3
		}
	}
	return false
}

var errUpstream = errors.New("tap: upstream d2d not reachable")

// roundTrip sends one line upstream and returns the payload lines and the status line. Caller holds m.mu.
func (m *Mux) roundTrip(line string) (lines []string, status string, err error) {
	if m.conn == nil {
		c, e := net.DialTimeout("unix", m.upstream, 2*time.Second)
		if e != nil {
			m.upErr.Store(e.Error())
			return nil, "", errUpstream
		}
		m.conn, m.rd = c, bufio.NewReaderSize(c, 1<<20)
	}
	if _, err = fmt.Fprintf(m.conn, "%s\n", line); err == nil {
		for {
			var l string
			l, err = m.rd.ReadString('\n')
			if err != nil {
				break
			}
			l = strings.TrimRight(l, "\r\n")
			if l == "ok" || l == "err" || strings.HasPrefix(l, "ok ") || strings.HasPrefix(l, "err ") {
				return lines, l, nil
			}
			lines = append(lines, l)
		}
	}
	m.upErr.Store(err.Error())
	m.conn.Close()
	m.conn, m.rd = nil, nil
	return nil, "", errUpstream
}

// Exec runs one line from the bot (bot true) or the viewer.
func (m *Mux) Exec(line string, bot bool) ([]string, string) {
	m.mu.Lock()
	if bot {
		m.botLines.Add(1)
		line = m.rewriteStep(line)
	} else {
		m.viewOps.Add(1)
	}
	lines, st, err := m.roundTrip(line)
	if err != nil {
		m.mu.Unlock()
		return nil, "err " + err.Error()
	}
	n := 0
	if bot && strings.HasPrefix(line, "step ") {
		n = m.afterStep(line)
	}
	m.mu.Unlock()
	// The pacing sleep is outside the lock (a slow watch speed must not freeze the viewer or other readers); the
	// bot is still paced because its reply is delayed.
	m.pace(n)
	return lines, st
}

func (m *Mux) rewriteStep(line string) string {
	md := m.mode.Load()
	f := strings.Fields(line)
	if md.Watch && len(f) == 2 && f[0] == "step" {
		return line + " render"
	}
	return line
}

// Watch speed bounds: a speed under minWatchSpeed is raised to it (0 stays "no limit"), and one step never sleeps
// longer than maxPaceSleep.
const (
	minWatchSpeed = 0.5
	maxWatchSpeed = 1000
	maxPaceSleep  = 2 * time.Second
)

// afterStep books a finished step and returns its tick count. Caller holds m.mu.
func (m *Mux) afterStep(line string) int {
	var n int
	fmt.Sscanf(strings.TrimPrefix(line, "step "), "%d", &n)
	m.steps.Add(1)
	m.ticks.Add(int64(n))
	m.lastStep.Store(time.Now().UnixNano())
	return n
}

// pace sleeps so n ticks take at least n*40ms/Speed of wall time in watch mode, in 100 ms slices that stop as soon
// as the mode changes.
func (m *Mux) pace(n int) {
	md := m.mode.Load()
	if !md.Watch || md.Speed <= 0 || n <= 0 {
		return
	}
	d := time.Duration(float64(n) * 40 * float64(time.Millisecond) / md.Speed)
	if d > maxPaceSleep || d < 0 {
		d = maxPaceSleep
	}
	for end := time.Now().Add(d); ; {
		left := time.Until(end)
		if left <= 0 || m.mode.Load() != md {
			return
		}
		time.Sleep(min(left, 100*time.Millisecond))
	}
}

// Serve accepts clients on path until ln closes. bot selects the bot rules (no filter).
func (m *Mux) Serve(path string, bot bool, shotDir string) (net.Listener, error) {
	_ = os.Remove(path)
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	_ = os.Chmod(path, 0o600)
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go m.client(c, bot, shotDir)
		}
	}()
	return ln, nil
}

func (m *Mux) client(c net.Conn, bot bool, shotDir string) {
	defer c.Close()
	if bot && m.botDown != nil {
		defer m.botDown()
	}
	rd := bufio.NewReaderSize(c, 70<<10)
	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		var out strings.Builder
		if !bot {
			// forward exactly what readOnly judged, not the raw bytes
			line = strings.Join(strings.Fields(line), " ")
		}
		if !bot && !readOnly(line, shotDir) {
			out.WriteString("err view: read-only viewer socket refuses '" + line + "'\n")
		} else {
			lines, st := m.Exec(line, bot)
			for _, l := range lines {
				out.WriteString(l + "\n")
			}
			out.WriteString(st + "\n")
		}
		if _, err := c.Write([]byte(out.String())); err != nil {
			return
		}
	}
}

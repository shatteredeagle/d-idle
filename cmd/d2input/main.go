// d2input: tiny helper that runs INSIDE the game's Wine prefix and posts mouse/key window
// messages straight to the Diablo II window. It never moves the host cursor and does not need focus.
// A native Linux process (internal/input) talks to it over loopback TCP, one text command per line,
// one reply line per command: "ok [data]" or "err CODE message".
//
//	list                              -> ok hwnd=0x.. pid=N client=WxH visible=1 ; hwnd=... (one line, windows joined by " ; ")
//	ping                              -> ok pong
//	info                              -> ok hwnd=0x.. client=WxH visible=1
//	move X Y                          (client-area pixels)
//	down|up l|r X Y [shift|mkshift|ctrl|mkctrl ...]
//	click l|r X Y [hold=MS] [shift|mkshift|ctrl|mkctrl ...]
//	key down|up|press NAME [hold=MS]  (a-z, 0-9, f1-f12, esc, space, enter, backspace, tab, shift, ctrl, alt)
//	char X                            (WM_CHAR with one character, for edit boxes; X may be a literal char or a name like space)
//	hung                              -> ok alive | ok hung   (IsHungAppWindow)
//	sleep MS
//	seq CMD ; CMD ; ...               run several commands back to back, one reply at the end
//	quit
//
// Targeting: any window command may be prefixed with "@<hwnd>" (decimal or 0x hex, as printed by list),
// e.g. "@0x30042 seq move 1 2 ; sleep 5". The prefix applies to a whole seq. Untargeted commands go to
// the only "Diablo II" window; with several windows they fail with "err ambiguous". A target that is not
// (or no longer) a Diablo II window gives "err nowindow no such window". list reports Windows pids, which
// are NOT Linux pids: bind a hwnd to a game process by posting a mouse move and reading MouseX/Y.
//
// Modifier flags: "shift"/"ctrl" post WM_KEYDOWN of the key around the mouse message and put MK_SHIFT/
// MK_CONTROL in wParam; "mkshift"/"mkctrl" only set the wParam bit (use with a key you hold via "key down").
// Error codes: usage, badbutton, badkey, nowindow, unknown, busy.
package main

import (
	"bufio"
	"d2lod-bot/internal/buildinfo"
	"fmt"
	"net"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

const defaultAddr = "127.0.0.1:47012"

// listenAddr is "-addr HOST:PORT" or $D2INPUT_ADDR, else the default. Every Wine prefix (= every game
// instance, see docs/multi.md) runs its own helper on its own port.
func listenAddr() string {
	for i, a := range os.Args[1:] {
		if a == "-addr" && i+2 < len(os.Args) {
			return os.Args[i+2]
		}
	}
	if v := os.Getenv("D2INPUT_ADDR"); v != "" {
		return v
	}
	return defaultAddr
}

// sys is the thin Windows layer, replaced by a fake in tests.
type sys interface {
	list() []winInfo
	valid(h uintptr) bool
	post(h uintptr, msg uint32, w, l uintptr) bool
	scan(vk uint32) uintptr
	info(h uintptr) string
	hung(h uintptr) bool
}

// winInfo is one "Diablo II" top-level window.
type winInfo struct {
	h       uintptr
	pid     uint32 // Windows pid
	w, ht   int    // client size
	visible bool
}

func (w winInfo) String() string {
	v := 0
	if w.visible {
		v = 1
	}
	return fmt.Sprintf("hwnd=0x%x pid=%d client=%dx%d visible=%d", w.h, w.pid, w.w, w.ht, v)
}

const (
	wmKeyDown     = 0x0100
	wmKeyUp       = 0x0101
	wmChar        = 0x0102
	wmMouseMove   = 0x0200
	wmLButtonDown = 0x0201
	wmLButtonUp   = 0x0202
	wmRButtonDown = 0x0204
	wmRButtonUp   = 0x0205

	mkLButton = 0x01
	mkRButton = 0x02
	mkShift   = 0x04
	mkControl = 0x08

	defaultHold = 60
	maxSleep    = 10000
)

func lparam(x, y int) uintptr {
	return uintptr(uint32(uint16(int16(y)))<<16 | uint32(uint16(int16(x))))
}

func vk(name string) (uint32, bool) {
	name = strings.ToLower(name)
	switch {
	case len(name) == 1 && name[0] >= 'a' && name[0] <= 'z':
		return uint32(name[0] - 'a' + 'A'), true
	case len(name) == 1 && name[0] >= '0' && name[0] <= '9':
		return uint32(name[0]), true
	case strings.HasPrefix(name, "f") && len(name) <= 3:
		if n, err := strconv.Atoi(name[1:]); err == nil && n >= 1 && n <= 12 {
			return uint32(0x70 + n - 1), true
		}
	}
	switch name {
	case "esc":
		return 0x1B, true
	case "space":
		return 0x20, true
	case "enter":
		return 0x0D, true
	case "backspace":
		return 0x08, true
	case "tab":
		return 0x09, true
	case "shift":
		return 0x10, true
	case "ctrl":
		return 0x11, true
	case "alt":
		return 0x12, true
	}
	return 0, false
}

type server struct {
	s sys
}

func (v *server) keyMsg(h uintptr, down bool, code uint32) {
	l := uintptr(1) | v.s.scan(code)<<16
	if down {
		v.s.post(h, wmKeyDown, uintptr(code), l)
		return
	}
	v.s.post(h, wmKeyUp, uintptr(code), l|1<<30|1<<31)
}

func sleepMS(ms int) { time.Sleep(time.Duration(ms) * time.Millisecond) }

type cmdErr struct{ code, msg string }

func (e *cmdErr) Error() string { return e.code + " " + e.msg }

func errf(code, f string, a ...any) *cmdErr { return &cmdErr{code, fmt.Sprintf(f, a...)} }

// args holds the tokens of one command with key=value options and modifier flags split off.
type args struct {
	pos   []string
	hold  int
	shift bool // post the key
	ctrl  bool
	mk    uintptr // extra wParam bits
}

func parseArgs(f []string) (args, *cmdErr) {
	a := args{hold: defaultHold}
	for _, t := range f {
		switch {
		case t == "shift":
			a.shift, a.mk = true, a.mk|mkShift
		case t == "ctrl":
			a.ctrl, a.mk = true, a.mk|mkControl
		case t == "mkshift":
			a.mk |= mkShift
		case t == "mkctrl":
			a.mk |= mkControl
		case strings.HasPrefix(t, "hold="):
			n, err := strconv.Atoi(t[5:])
			if err != nil || n < 0 || n > maxSleep {
				return a, errf("usage", "bad hold %q", t)
			}
			a.hold = n
		default:
			a.pos = append(a.pos, t)
		}
	}
	return a, nil
}

func (a args) xy(i int) (int, int, *cmdErr) {
	if len(a.pos) < i+2 {
		return 0, 0, errf("usage", "missing X Y")
	}
	x, e1 := strconv.Atoi(a.pos[i])
	y, e2 := strconv.Atoi(a.pos[i+1])
	if e1 != nil || e2 != nil {
		return 0, 0, errf("usage", "X Y must be integers")
	}
	return x, y, nil
}

func button(s string) (down, up uint32, mk uintptr, ok bool) {
	switch s {
	case "l":
		return wmLButtonDown, wmLButtonUp, mkLButton, true
	case "r":
		return wmRButtonDown, wmRButtonUp, mkRButton, true
	}
	return 0, 0, 0, false
}

// resolve picks the window a command acts on: the explicit target, else the only window.
func (v *server) resolve(target uintptr) (uintptr, *cmdErr) {
	if target != 0 {
		if !v.s.valid(target) {
			return 0, errf("nowindow", "no such window")
		}
		return target, nil
	}
	ws := v.s.list()
	switch len(ws) {
	case 0:
		return 0, errf("nowindow", "no Diablo II window")
	case 1:
		return ws[0].h, nil
	}
	return 0, errf("ambiguous", "%d Diablo II windows, target one with @hwnd", len(ws))
}

// exec runs one non-seq command against target (0 = the only window) and returns the reply data.
func (v *server) exec(line string, target uintptr) (string, *cmdErr) {
	f := strings.Fields(line)
	if len(f) == 0 {
		return "", errf("usage", "empty command")
	}
	cmd := f[0]
	switch cmd {
	case "ping":
		return "pong", nil
	case "sleep":
		if len(f) != 2 {
			return "", errf("usage", "sleep MS")
		}
		n, err := strconv.Atoi(f[1])
		if err != nil || n < 0 || n > maxSleep {
			return "", errf("usage", "bad sleep %q", f[1])
		}
		sleepMS(n)
		return "", nil
	case "list":
		var out []string
		for _, w := range v.s.list() {
			out = append(out, w.String())
		}
		return strings.Join(out, " ; "), nil
	case "info", "move", "down", "up", "click", "key", "char", "hung":
	default:
		return "", errf("unknown", "unknown command %s", cmd)
	}
	a, perr := parseArgs(f[1:])
	if perr != nil {
		return "", perr
	}
	h, rerr := v.resolve(target)
	if rerr != nil {
		return "", rerr
	}
	switch cmd {
	case "info":
		return v.s.info(h), nil
	case "hung":
		if v.s.hung(h) {
			return "hung", nil
		}
		return "alive", nil
	case "char":
		if len(a.pos) != 1 {
			return "", errf("usage", "char X")
		}
		r := []rune(a.pos[0])
		if a.pos[0] == "space" {
			r = []rune{' '}
		}
		if len(r) != 1 || r[0] > 0xFFFF {
			return "", errf("usage", "char takes one character")
		}
		v.s.post(h, wmChar, uintptr(r[0]), 1)
	case "move":
		x, y, e := a.xy(0)
		if e != nil {
			return "", e
		}
		v.s.post(h, wmMouseMove, a.mk, lparam(x, y))
	case "down", "up", "click":
		if len(a.pos) < 1 {
			return "", errf("usage", "%s l|r X Y", cmd)
		}
		d, u, mk, ok := button(a.pos[0])
		if !ok {
			return "", errf("badbutton", "button must be l or r")
		}
		x, y, e := a.xy(1)
		if e != nil {
			return "", e
		}
		l := lparam(x, y)
		if a.shift {
			v.keyMsg(h, true, 0x10)
		}
		if a.ctrl {
			v.keyMsg(h, true, 0x11)
		}
		if a.shift || a.ctrl {
			sleepMS(20)
		}
		switch cmd {
		case "down":
			v.s.post(h, d, mk|a.mk, l)
		case "up":
			v.s.post(h, u, a.mk, l)
		case "click":
			v.s.post(h, wmMouseMove, a.mk, l)
			sleepMS(40)
			v.s.post(h, d, mk|a.mk, l)
			sleepMS(a.hold)
			v.s.post(h, u, a.mk, l)
		}
		if a.shift || a.ctrl {
			sleepMS(20)
		}
		if a.ctrl {
			v.keyMsg(h, false, 0x11)
		}
		if a.shift {
			v.keyMsg(h, false, 0x10)
		}
	case "key":
		if len(a.pos) < 2 {
			return "", errf("usage", "key down|up|press NAME")
		}
		code, ok := vk(a.pos[1])
		if !ok {
			return "", errf("badkey", "unknown key %s", a.pos[1])
		}
		switch a.pos[0] {
		case "down":
			v.keyMsg(h, true, code)
		case "up":
			v.keyMsg(h, false, code)
		case "press":
			v.keyMsg(h, true, code)
			sleepMS(a.hold)
			v.keyMsg(h, false, code)
		default:
			return "", errf("usage", "key down|up|press NAME")
		}
	}
	return "", nil
}

// handle runs a line (possibly a seq) and formats the single reply line.
func (v *server) handle(line string) string {
	line = strings.TrimSpace(line)
	var target uintptr
	if strings.HasPrefix(line, "@") {
		tok, rest, _ := strings.Cut(line, " ")
		n, err := strconv.ParseUint(tok[1:], 0, 64)
		if err != nil || n == 0 {
			return "err usage bad target " + tok
		}
		target, line = uintptr(n), strings.TrimSpace(rest)
	}
	if rest, ok := strings.CutPrefix(line, "seq "); ok {
		n := 0
		for _, part := range strings.Split(rest, ";") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if strings.HasPrefix(part, "seq") {
				return "err usage nested seq"
			}
			if _, e := v.exec(part, target); e != nil {
				return fmt.Sprintf("err %s step %d (%s): %s", e.code, n+1, part, e.msg)
			}
			n++
		}
		return fmt.Sprintf("ok %d", n)
	}
	data, e := v.exec(line, target)
	if e != nil {
		return "err " + e.code + " " + e.msg
	}
	if data == "" {
		return "ok"
	}
	return "ok " + data
}

func main() {
	buildinfo.Exit("d2input")
	limitProcs()
	addr := listenAddr()
	initLog(addr)
	watchSignals()
	// The listener is retried: a transient bind failure (port in TIME_WAIT after a quick restart) must not
	// make a supervisor see a dead helper.
	var ln net.Listener
	var err error
	for i := 0; i < 20; i++ {
		if ln, err = net.Listen("tcp", addr); err == nil {
			break
		}
		logf("listen %s: %v (retry %d)", addr, err, i)
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		exitf(1, "listen: %v", err)
	}
	fmt.Println("d2input listening on", addr)
	logf("listening on %s", addr)
	v := &server{s: newSys()}
	go heartbeat(v.s)
	for {
		c, err := ln.Accept()
		if err != nil {
			nErrs.Add(1)
			logf("accept: %v", err)
			time.Sleep(50 * time.Millisecond)
			continue
		}
		go serve(v, c)
	}
}

// limitProcs sets GOMAXPROCS to 1 (or $D2INPUT_PROCS). The Go runtime keeps one idle/spinning OS thread per
// P it ever used and never ends them; the helper sees every host core (16 on the worker) and under Wine each
// thread costs ~7 host fds. Verified on a Linux stand-in (osThread + 6 clients + blocking syscalls):
// threadcreate 12 / 9 / 7 for GOMAXPROCS 8 / 4 / 1. The helper is I/O bound (a few hundred commands a
// minute), so one P is plenty.
func limitProcs() {
	n := 1
	if v, err := strconv.Atoi(os.Getenv("D2INPUT_PROCS")); err == nil && v > 0 {
		n = v
	}
	runtime.GOMAXPROCS(n)
}

func serve(v *server, c net.Conn) {
	nConns.Add(1)
	defer nConns.Add(-1)
	defer c.Close()
	defer func() {
		if r := recover(); r != nil {
			nPanics.Add(1)
			logf("PANIC in connection %v: %v\n%s", c.RemoteAddr(), r, debug.Stack())
		}
	}()
	sc := bufio.NewScanner(c)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "quit" {
			fmt.Fprintln(c, "ok bye")
			exitf(0, "quit command from %v", c.RemoteAddr())
		}
		nCmds.Add(1)
		if _, err := fmt.Fprintln(c, v.safeHandle(line)); err != nil {
			logf("write to %v: %v", c.RemoteAddr(), err)
			return
		}
	}
	if err := sc.Err(); err != nil {
		logf("read from %v: %v", c.RemoteAddr(), err)
	}
}

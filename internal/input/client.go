package input

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/mem"
)

// DefaultAddr is where cmd/d2input listens (inside the Wine prefix, reachable on the host loopback).
const DefaultAddr = "127.0.0.1:47012"

var (
	// ErrNoWindow means the helper is up but the "Diablo II" window does not exist (game closed or restarting).
	ErrNoWindow = errors.New("input: no Diablo II window")
	// ErrHelperDown means the helper itself cannot be reached.
	ErrHelperDown = errors.New("input: d2input helper unreachable")
	// ErrTimeout means the helper did not reply in time. The command may or may not have run.
	ErrTimeout = errors.New("input: helper reply timed out")
	// ErrAmbiguous means several Diablo II windows exist and the client has no target.
	ErrAmbiguous = errors.New("input: several Diablo II windows, no target")
	// ErrBind means a window could not be tied to a game process (posted move not seen in its memory).
	ErrBind = errors.New("input: window not bound to game process")
)

// HwndEnv names the window handle a client should target (as printed by the helper's list, 0x.. or decimal).
const HwndEnv = "D2BOT_HWND"

// CmdError is an "err CODE message" reply from the helper.
type CmdError struct{ Code, Msg string }

func (e *CmdError) Error() string { return "input: helper: " + e.Code + ": " + e.Msg }

func (e *CmdError) Is(target error) bool {
	return (target == ErrNoWindow && e.Code == "nowindow") || (target == ErrAmbiguous && e.Code == "ambiguous")
}

// Client talks to the d2input helper. It is safe for concurrent use; commands are serialized
// (the helper is a single input device, interleaved clicks would be wrong anyway).
type Client struct {
	Addr        string
	DialTimeout time.Duration // default 2s
	BaseTimeout time.Duration // slack added to every reply deadline, default 2s

	// Hwnd is the window every command is sent to (0: untargeted, the helper's only window).
	Hwnd uint64
	// Instance, when Hwnd is 0, makes the client find its window itself on first use: the game
	// process of that instance (mem.FindGameFor) is bound to one of the helper's windows with Bind.
	Instance string

	// Restart, when set, brings a dead helper back (default: `d2game helper iN` on the machine that runs the
	// games, see restartHelper). Called at most once per HealCooldown after every dial attempt of a call failed.
	Restart      func() error
	HealCooldown time.Duration // default 15s
	healed       time.Time

	tmu  sync.Mutex // guards Hwnd/Instance resolution
	mu   sync.Mutex
	conn net.Conn
	rd   *bufio.Reader
}

// NewClient returns a client for the helper at addr. Targeting comes from the environment: $D2BOT_HWND
// pins a window, else $D2BOT_INSTANCE makes the client discover and bind its window lazily; with
// neither it is untargeted (the old one-window behavior).
func NewClient(addr string) *Client {
	c := newClient(addr)
	if v := os.Getenv(HwndEnv); v != "" {
		if n, err := strconv.ParseUint(v, 0, 64); err == nil {
			c.Hwnd = n
			return c
		}
	}
	c.Instance = os.Getenv(mem.InstanceEnv)
	if addr == "" {
		c.Addr = AddrFor(c.Instance)
	}
	return c
}

// NewClientFor returns a client pinned to one window.
func NewClientFor(addr string, hwnd uint64) *Client {
	c := newClient(addr)
	c.Hwnd = hwnd
	return c
}

// NewClientInstance returns a client that finds the window of the named game instance by itself.
func NewClientInstance(addr, instance string) *Client {
	if addr == "" {
		addr = AddrFor(instance)
	}
	c := newClient(addr)
	c.Instance = instance
	return c
}

// AddrFor is the helper address of an instance: every game instance lives in its own Wine prefix with
// its own d2input helper on port 47012+N for "iN" ("" and "i0" use DefaultAddr). $D2INPUT_ADDR overrides.
func AddrFor(instance string) string {
	if v := os.Getenv("D2INPUT_ADDR"); v != "" {
		return v
	}
	if n, err := strconv.Atoi(strings.TrimPrefix(instance, "i")); err == nil && n > 0 && strings.HasPrefix(instance, "i") {
		return "127.0.0.1:" + strconv.Itoa(47012+n)
	}
	return DefaultAddr
}

func newClient(addr string) *Client {
	if addr == "" {
		addr = DefaultAddr
	}
	return &Client{Addr: addr, DialTimeout: 2 * time.Second, BaseTimeout: 2 * time.Second}
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.dropLocked()
}

func (c *Client) dropLocked() error {
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn, c.rd = nil, nil
	return err
}

// dialBackoff are the waits between the dial attempts of one call: a helper that is being restarted
// (by the sentinel, by us) is back within a second or two, a dead one is declared down after 3 refusals.
// errDial marks a failure before anything was sent (safe to retry after a helper restart).
var errDial = errors.New("dial failed")

var dialBackoff = []time.Duration{150 * time.Millisecond, 500 * time.Millisecond}

func (c *Client) dialLocked() error {
	if c.conn != nil {
		return nil
	}
	var err error
	for i := 0; ; i++ {
		var conn net.Conn
		if conn, err = net.DialTimeout("tcp", c.Addr, c.DialTimeout); err == nil {
			c.conn, c.rd = conn, bufio.NewReader(conn)
			return nil
		}
		if i >= len(dialBackoff) {
			return fmt.Errorf("%w: %w: %v", ErrHelperDown, errDial, err)
		}
		time.Sleep(dialBackoff[i])
	}
}

// heal tries to restart a dead helper (3 dial attempts already failed). It reports whether a restart ran
// and the helper answers again. Rate limited so a bot loop cannot spawn restarts in a tight loop.
func (c *Client) heal() bool {
	c.mu.Lock()
	cd := c.HealCooldown
	if cd == 0 {
		cd = 15 * time.Second
	}
	if time.Since(c.healed) < cd {
		c.mu.Unlock()
		return false
	}
	c.healed = time.Now()
	c.mu.Unlock()
	restart := c.Restart
	if restart == nil {
		restart = func() error { return restartHelper(c.Addr) }
	}
	if err := restart(); err != nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.dialLocked() == nil
}

// restartHelper runs `d2game helper iN` for the instance whose helper port is addr. It only works on the
// machine that runs the games (the worker); elsewhere there is no d2game and the call fails harmlessly.
// $D2GAME_BIN overrides the binary. The helper command is idempotent and does not take the game lock.
func restartHelper(addr string) error {
	_, ps, ok := strings.Cut(addr, ":")
	port, err := strconv.Atoi(ps)
	if !ok || err != nil || port < 47012 || port > 47021 {
		return fmt.Errorf("input: no instance for helper %s", addr)
	}
	bin := os.Getenv("D2GAME_BIN")
	if bin == "" {
		home, _ := os.UserHomeDir()
		bin = home + "/d2lod-live/d2game/d2game"
	}
	if _, err := os.Stat(bin); err != nil {
		return err
	}
	if port == 47012 {
		return fmt.Errorf("input: i0's helper has no automatic restart (tools/helper.sh restart)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "helper", "i"+strconv.Itoa(port-47012))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("input: d2game helper: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Do sends one command line and returns the data after "ok". work is how long the helper is
// expected to spend executing it (sleeps, holds); the reply deadline is work + BaseTimeout.
// A dead connection is redialed once before anything was sent; a reply timeout is never retried
// (the command might already have run) and drops the connection so a late reply cannot desync it.
func (c *Client) Do(cmd string, work time.Duration) (string, error) {
	if cmd == "ping" || cmd == "list" {
		return c.doRaw(cmd, work)
	}
	h, err := c.window()
	if err != nil {
		return "", err
	}
	if h == 0 {
		return c.doRaw(cmd, work)
	}
	r, err := c.doRaw(fmt.Sprintf("@0x%x %s", h, cmd), work)
	var ce *CmdError
	if err != nil && errors.As(err, &ce) && ce.Code == "nowindow" && c.Instance != "" && os.Getenv(HwndEnv) == "" {
		// The game was restarted: its window is gone. Forget the binding and find the new one once.
		c.tmu.Lock()
		if c.Hwnd == h {
			c.Hwnd = 0
		}
		c.tmu.Unlock()
		if h, err = c.window(); err != nil {
			return "", err
		}
		if h != 0 {
			return c.doRaw(fmt.Sprintf("@0x%x %s", h, cmd), work)
		}
	}
	return r, err
}

// window returns the target hwnd, resolving it from Instance on first use (0 = untargeted).
func (c *Client) window() (uint64, error) {
	c.tmu.Lock()
	defer c.tmu.Unlock()
	if c.Hwnd != 0 || c.Instance == "" {
		return c.Hwnd, nil
	}
	h, err := ResolveInstance(c, c.Instance)
	if err != nil {
		return 0, err
	}
	c.Hwnd = h
	return h, nil
}

func (c *Client) doRaw(cmd string, work time.Duration) (string, error) {
	r, err := c.doRaw1(cmd, work)
	if err != nil && errors.Is(err, errDial) && c.heal() {
		return c.doRaw1(cmd, work) // the helper was restarted; the failed command never ran
	}
	return r, err
}

func (c *Client) doRaw1(cmd string, work time.Duration) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for attempt := 0; ; attempt++ {
		if err := c.dialLocked(); err != nil {
			return "", err
		}
		c.conn.SetWriteDeadline(time.Now().Add(c.BaseTimeout))
		if _, err := fmt.Fprintf(c.conn, "%s\n", cmd); err != nil {
			c.dropLocked()
			if attempt == 0 {
				continue // stale connection, nothing was executed
			}
			return "", fmt.Errorf("%w: %v", ErrHelperDown, err)
		}
		c.conn.SetReadDeadline(time.Now().Add(work + c.BaseTimeout))
		line, err := c.rd.ReadString('\n')
		if err != nil {
			c.dropLocked()
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				return "", ErrTimeout
			}
			if attempt == 0 && line == "" {
				continue // peer closed while idle (EOF, no reply bytes): the write hit a dead socket, so resend once
			}
			return "", fmt.Errorf("%w: %v", ErrHelperDown, err)
		}
		return parseReply(strings.TrimSpace(line))
	}
}

func parseReply(line string) (string, error) {
	switch {
	case line == "ok":
		return "", nil
	case strings.HasPrefix(line, "ok "):
		return line[3:], nil
	case strings.HasPrefix(line, "err "):
		code, msg, _ := strings.Cut(line[4:], " ")
		return "", &CmdError{Code: code, Msg: msg}
	}
	return "", fmt.Errorf("input: bad helper reply %q", line)
}

// Ping checks the helper is alive (not that the game window exists; use Info for that).
func (c *Client) Ping() error {
	_, err := c.Do("ping", 0)
	return err
}

// WindowInfo is the helper's view of the game window.
type WindowInfo struct {
	Hwnd          string
	H             uint64 // Hwnd as a number
	Pid           int    // WINDOWS pid (list only); not a Linux pid
	Width, Height int
	Visible       bool
}

func parseWindow(r string) WindowInfo {
	var w WindowInfo
	for _, f := range strings.Fields(r) {
		k, v, _ := strings.Cut(f, "=")
		switch k {
		case "hwnd":
			w.Hwnd = v
			w.H, _ = strconv.ParseUint(v, 0, 64)
		case "pid":
			w.Pid, _ = strconv.Atoi(v)
		case "client":
			ws, hs, _ := strings.Cut(v, "x")
			w.Width, _ = strconv.Atoi(ws)
			w.Height, _ = strconv.Atoi(hs)
		case "visible":
			w.Visible = v == "1"
		}
	}
	return w
}

// Info returns the client's window (the target, or the only one), or ErrNoWindow.
func (c *Client) Info() (WindowInfo, error) {
	r, err := c.Do("info", 0)
	if err != nil {
		return WindowInfo{}, err
	}
	return parseWindow(r), nil
}

// Discover lists every Diablo II window the helper sees (never targeted). Pids are Windows pids.
func Discover(c *Client) ([]WindowInfo, error) {
	r, err := c.doRaw("list", 0)
	if err != nil {
		return nil, err
	}
	var out []WindowInfo
	for _, part := range strings.Split(r, ";") {
		if strings.TrimSpace(part) != "" {
			out = append(out, parseWindow(part))
		}
	}
	return out, nil
}

// Bind proves that ctrl (aimed at one window) drives the process behind src: it posts a mouse move to
// a random point through ctrl and checks the game's own MouseX/MouseY in src become exactly that point.
// Windows pids differ from Linux pids, so this handshake is the only link between hwnd and process.
func Bind(ctrl Controller, src mem.Source) error {
	r := Reader{Src: src}
	cx, cy, err := r.Mouse()
	if err != nil {
		return err
	}
	x, y := 40+rand.Intn(720), 40+rand.Intn(520)
	if x == cx && y == cy {
		x++
	}
	if err := ctrl.MoveMouse(x, y); err != nil {
		return err
	}
	for i := 0; i < 8; i++ {
		time.Sleep(40 * time.Millisecond)
		if mx, my, err := r.Mouse(); err == nil && mx == x && my == y {
			return nil
		}
	}
	return fmt.Errorf("%w (posted %d,%d)", ErrBind, x, y)
}

// ResolveInstance finds the helper window that belongs to the game process of instance and returns its
// hwnd. A helper that sees exactly one window (the normal case: one helper per prefix) needs no
// handshake; with several windows each is tried with Bind.
func ResolveInstance(c *Client, instance string) (uint64, error) {
	ws, err := Discover(c)
	if err != nil {
		return 0, err
	}
	if len(ws) == 0 {
		return 0, ErrNoWindow
	}
	if len(ws) == 1 {
		return ws[0].H, nil // per-prefix helper: it only sees its own game's window
	}
	pid, err := mem.FindGameFor(instance)
	if err != nil {
		return 0, err
	}
	src, err := mem.Open(pid)
	if err != nil {
		return 0, err
	}
	defer src.Close()
	if len(ws) == 1 {
		return ws[0].H, nil // per-prefix helper: it only sees its own game's window
	}
	var last error
	for _, w := range ws {
		g := &Game{C: NewClientFor(c.Addr, w.H), T: DefaultTiming(), held: map[string]bool{}}
		err := Bind(g, src)
		g.C.Close()
		if err == nil {
			return w.H, nil
		}
		last = err
	}
	return 0, last
}

// Seq runs several commands in one round trip. Their combined sleeps/holds set the reply deadline.
func (c *Client) Seq(steps ...string) error {
	if len(steps) == 0 {
		return nil
	}
	_, err := c.Do("seq "+strings.Join(steps, " ; "), seqWork(steps))
	return err
}

func seqWork(steps []string) time.Duration {
	var ms int
	for _, s := range steps {
		f := strings.Fields(s)
		if len(f) == 2 && f[0] == "sleep" {
			n, _ := strconv.Atoi(f[1])
			ms += n
		}
		for _, t := range f {
			if v, ok := strings.CutPrefix(t, "hold="); ok {
				n, _ := strconv.Atoi(v)
				ms += n
			}
		}
		ms += 150 // click/shift implicit sleeps
	}
	return time.Duration(ms) * time.Millisecond
}

// Alive reports whether the game window answers (false when hung). ErrNoWindow if it is gone.
func (c *Client) Alive() (bool, error) {
	r, err := c.Do("hung", 0)
	return r == "alive", err
}

// TypeChar posts WM_CHAR for one character (edit boxes such as the character name).
func (c *Client) TypeChar(ch rune) error {
	return c.Seq("char " + string(ch))
}

// OOG adapts a Client to the small interface internal/launch uses for out-of-game menus
// (Click(x, y), Key(name)); no game memory is needed, so it works on the menu screens.
type OOG struct {
	C *Client
	T Timing
}

func NewOOG(c *Client) *OOG { return &OOG{C: c, T: DefaultTiming()} }

func (o *OOG) Click(x, y int) error {
	err := o.C.Seq(
		fmt.Sprintf("move %d %d", x, y),
		"sleep "+ms(o.T.Settle()),
		fmt.Sprintf("down l %d %d", x, y),
		"sleep "+ms(o.T.Press()),
		fmt.Sprintf("up l %d %d", x, y),
	)
	time.Sleep(o.T.Gap())
	return err
}

func (o *OOG) Key(name string) error {
	err := o.C.Seq(fmt.Sprintf("key press %s hold=%s", name, ms(o.T.Press())))
	time.Sleep(o.T.Gap())
	return err
}

// TypeChar lets OOG satisfy launch.Typer (character names are typed with WM_CHAR).
func (o *OOG) TypeChar(r rune) error {
	err := o.C.TypeChar(r)
	time.Sleep(o.T.Gap())
	return err
}

// IsHelperDown reports whether err means the helper process itself is unreachable.
func IsHelperDown(err error) bool { return errors.Is(err, ErrHelperDown) }

// IsNoWindow reports whether err means the helper is fine but the game window is missing.
func IsNoWindow(err error) bool { return errors.Is(err, ErrNoWindow) }

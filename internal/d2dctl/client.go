// Package d2dctl is a client for the d2d dev control channel: a line protocol over an AF_UNIX stream socket.
//
// Protocol (our summary; facts from docs/d2d-spike.md, observed live, and the upstream channel notes):
// one command per LF-terminated line, a reply is zero or more payload lines then a final "ok[ rest]" or
// "err <message>" line. Only one client may be connected: a second one gets "err busy" and is dropped.
// Lines longer than 64 KiB drop the client with "err line too long".
//
// The channel has no authentication and can write files (screenshot), so this package only ever dials
// "unix" sockets.
package d2dctl

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// MaxLine is the channel's line limit in bytes (V: documented upstream, not stress-tested by us).
const MaxLine = 64 << 10

// DefaultTimeout bounds one command when neither the context nor Options.Timeout says otherwise.
const DefaultTimeout = 5 * time.Second

var (
	// ErrBusy: another client already holds the channel.
	ErrBusy = errors.New("d2dctl: channel busy (another client is connected)")
	// ErrLineTooLong: a line exceeded MaxLine (ours when sending, the peer's when it reports it).
	ErrLineTooLong = errors.New("d2dctl: line too long")
	// ErrClosed: the client was closed, or the connection broke and reconnecting is off.
	ErrClosed = errors.New("d2dctl: client closed")
)

// CmdError is an "err <message>" reply. Lines holds any payload that came before it.
type CmdError struct {
	Msg   string
	Reply *Reply
}

func (e *CmdError) Error() string { return "d2d: " + e.Msg }

// Reply is one command's answer.
type Reply struct {
	Cmd   string   // the line sent
	Lines []string // payload lines before the status line
	Rest  string   // text after "ok" on the status line (may be empty)
	// Bad lists payload lines a typed method could not parse (they are still in Lines).
	Bad []string
}

// Text returns payload lines and the status rest joined with newlines, for regex assertions and logs.
func (r *Reply) Text() string {
	parts := append([]string(nil), r.Lines...)
	if r.Rest != "" {
		parts = append(parts, r.Rest)
	}
	return strings.Join(parts, "\n")
}

// Options tunes a Client. The zero value is sensible.
type Options struct {
	Timeout     time.Duration // per command; 0 = DefaultTimeout. A shorter context deadline wins.
	NoReconnect bool          // do not redial after a broken connection
	NoAutoKick  bool          // do not send the priming click before cmd move/interact (see Kick)
}

// Client talks to one d2d control socket. It is safe for concurrent use: calls are serialised on the
// single command stream, so a reply always belongs to the command that was sent.
type Client struct {
	path string
	opts Options

	mu     sync.Mutex
	conn   net.Conn
	rd     *bufio.Reader
	closed bool
	lastEr error // why the connection was dropped, reported when reconnect is off

	kickPending atomic.Bool
	tr          tracer
	invHint     atomic.Bool           // the inventory panel is believed open (see InventoryHint)
	charHint    atomic.Bool           // the character panel is believed open (see CharacterHint)
	pre         map[string]prefetched // replies fetched ahead by Prefetch, consumed by the matching Do
	jr          *Journal              // command journal for the determinism finder (OpenJournal), nil: off
}

type prefetched struct {
	rep *Reply
	err error
}

// New returns a client that connects lazily on the first command.
func New(path string, opts Options) *Client { return &Client{path: path, opts: opts} }

// Dial is New plus an immediate connect, so a missing socket fails fast.
func Dial(ctx context.Context, path string, opts Options) (*Client, error) {
	c := New(path, opts)
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.connect(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

// Path is the socket path.
func (c *Client) Path() string { return c.path }

func (c *Client) connect(ctx context.Context) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", c.path)
	if err != nil {
		return fmt.Errorf("d2dctl: dial %s: %w", c.path, err)
	}
	c.conn = conn
	c.rd = bufio.NewReaderSize(conn, MaxLine+2)
	return nil
}

func (c *Client) drop(err error) {
	if c.conn != nil {
		c.conn.Close()
		c.conn, c.rd = nil, nil
	}
	c.lastEr = err
}

// Close closes the connection; later calls return ErrClosed.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	if c.conn != nil {
		err := c.conn.Close()
		c.conn, c.rd = nil, nil
		return err
	}
	return nil
}

// Connected reports whether a connection is currently open (it may still be dead; the next call finds out).
func (c *Client) Connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

// Do sends one command line and returns its reply. An "err" reply comes back as (*Reply, *CmdError);
// "err busy" as ErrBusy. After any transport error the connection is dropped (the stream may be out of
// step); the next call redials unless NoReconnect. A command is never resent automatically, because
// verbs like click are not idempotent.
func (c *Client) Do(ctx context.Context, line string) (*Reply, error) {
	if strings.ContainsAny(line, "\r\n") {
		return nil, errors.New("d2dctl: command contains a newline")
	}
	if len(line) >= MaxLine {
		return nil, ErrLineTooLong
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if p, ok := c.pre[line]; ok {
		delete(c.pre, line)
		return p.rep, p.err
	}
	if err := c.ready(ctx); err != nil {
		return nil, err
	}
	timeout := c.opts.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	conn := c.conn // not c.conn: drop() below nils it while the callback may still be queued (nil deref at shutdown)
	stop := context.AfterFunc(ctx, func() { conn.SetDeadline(time.Unix(1, 0)) })
	defer stop()

	t0 := time.Now()
	rep, err := c.exchange(line)
	if err == nil {
		c.noteKey(line)
	}
	c.tr.add(line, time.Since(t0), rep)
	if err != nil {
		var ce *CmdError
		if !errors.As(err, &ce) || errors.Is(err, ErrBusy) || errors.Is(err, ErrLineTooLong) {
			if ctx.Err() != nil {
				err = fmt.Errorf("%w (%v)", ctx.Err(), err)
			}
			c.drop(err)
		}
	}
	c.journal(line, rep, err)
	return rep, err
}

// ready makes sure there is a connection (caller holds mu).
func (c *Client) ready(ctx context.Context) error {
	if c.closed {
		return ErrClosed
	}
	if c.conn == nil {
		if c.lastEr != nil && c.opts.NoReconnect {
			return fmt.Errorf("%w: %v", ErrClosed, c.lastEr)
		}
		return c.connect(ctx)
	}
	return nil
}

// Prefetch sends the lines in ONE write and stores their replies; each later Do with the same line returns its stored
// reply instead of a round trip, once. d2d dispatches every complete line of its receive buffer in one frame (V by
// reading devctl.cpp Channel::pump, and live: docs/d2d-pace.md), so n verbs cost one frame instead of n. Use it for
// reads only, and call DropPrefetch when the observation is done so a stale reply is never served later. A transport
// error drops the connection and the unread lines (nothing is stored for them).
func (c *Client) Prefetch(ctx context.Context, lines ...string) error {
	for _, l := range lines {
		if strings.ContainsAny(l, "\r\n") || len(l) >= MaxLine {
			return errors.New("d2dctl: bad prefetch line")
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pre = nil
	if err := c.ready(ctx); err != nil {
		return err
	}
	timeout := c.opts.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	conn := c.conn
	stop := context.AfterFunc(ctx, func() { conn.SetDeadline(time.Unix(1, 0)) })
	defer stop()

	t0 := time.Now()
	if _, err := conn.Write([]byte(strings.Join(lines, "\n") + "\n")); err != nil {
		c.drop(err)
		return err
	}
	pre := make(map[string]prefetched, len(lines))
	for _, l := range lines {
		rep, err := c.readReply(l)
		if err != nil {
			var ce *CmdError
			if !errors.As(err, &ce) || errors.Is(err, ErrBusy) || errors.Is(err, ErrLineTooLong) {
				if ctx.Err() != nil {
					err = fmt.Errorf("%w (%v)", ctx.Err(), err)
				}
				c.drop(err)
				return err
			}
		}
		pre[l] = prefetched{rep, err}
		c.journal(l, rep, err)
		// one round trip shared by the batch: charge it to the first line, the rest count as free
		c.tr.add(l, time.Since(t0), rep)
		t0 = time.Now()
	}
	c.pre = pre
	return nil
}

// DropPrefetch discards replies that were not consumed.
func (c *Client) DropPrefetch() {
	c.mu.Lock()
	c.pre = nil
	c.mu.Unlock()
}

func (c *Client) exchange(line string) (*Reply, error) {
	if _, err := c.conn.Write([]byte(line + "\n")); err != nil {
		rep := &Reply{Cmd: line}
		// A busy server answers "err busy" and hangs up right away, so our write can fail
		// with EPIPE while its reply is still waiting to be read.
		if l, rerr := c.readLine(); rerr == nil && l == "err busy" {
			return rep, fmt.Errorf("%w: %w", ErrBusy, &CmdError{Msg: "busy", Reply: rep})
		}
		return nil, err
	}
	return c.readReply(line)
}

// readReply reads one command's reply from the stream.
func (c *Client) readReply(line string) (*Reply, error) {
	rep := &Reply{Cmd: line}
	for {
		l, err := c.readLine()
		if err != nil {
			return rep, err
		}
		switch {
		case l == "ok":
			return rep, nil
		case strings.HasPrefix(l, "ok "):
			rep.Rest = strings.TrimSpace(l[3:])
			return rep, nil
		case l == "err" || strings.HasPrefix(l, "err "):
			msg := strings.TrimSpace(strings.TrimPrefix(l, "err"))
			ce := &CmdError{Msg: msg, Reply: rep}
			switch msg {
			case "busy":
				return rep, fmt.Errorf("%w: %w", ErrBusy, ce)
			case "line too long":
				return rep, fmt.Errorf("%w: %w", ErrLineTooLong, ce)
			}
			return rep, ce
		}
		rep.Lines = append(rep.Lines, l)
	}
}

func (c *Client) readLine() (string, error) {
	b, err := c.rd.ReadSlice('\n')
	switch {
	case errors.Is(err, bufio.ErrBufferFull):
		return "", ErrLineTooLong
	case err != nil:
		if len(b) > 0 {
			err = fmt.Errorf("d2dctl: connection closed mid-line: %w", err)
		}
		return "", err
	}
	return strings.TrimRight(string(b), "\r\n"), nil
}

// noteKey keeps the inventory hint: d2d's "dump ui" of lab2 has no inventory flag, so the client remembers the hotkey.
// "key i" toggles it, "key Escape" closes the topmost panel (the game menu opens only when none is), a level change or
// game start has no panel. A lab3 or newer d2d reports ui.rects.inv_grid and the hint is then ignored.
func (c *Client) noteKey(line string) {
	switch line {
	case "key i", "key I":
		c.invHint.Store(!c.invHint.Load())
	case "key c", "key C":
		c.charHint.Store(!c.charHint.Load())
	case "key Escape":
		c.invHint.Store(false)
		c.charHint.Store(false)
	}
}

// CharacterHint is the same guess for the character panel (stat buttons), which "dump ui" does not report at all
// (D2DLOOP r4, V live: the leveler's "panel did not open (flag 0x2)" with five stat points unspent).
func (c *Client) CharacterHint() bool { return c.charHint.Load() }

// InventoryHint is the hotkey-tracked guess whether the inventory panel is open (U: only right while nothing else
// opens or closes it; an NPC menu, a level change or a death can close it without a key).
func (c *Client) InventoryHint() bool { return c.invHint.Load() }

// Busy reports whether a command is in flight (the stream is held), without waiting for it.
func (c *Client) Busy() bool {
	if c.mu.TryLock() {
		c.mu.Unlock()
		return false
	}
	return true
}

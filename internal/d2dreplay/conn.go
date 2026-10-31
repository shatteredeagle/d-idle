package d2dreplay

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

// Conn is one raw control-channel connection (docs/d2dctl.md protocol): a line in, payload lines and a status
// line out. The tap and the player use it instead of d2dctl.Client so nothing (auto kick, prefetch, tracing)
// is added to the stream between the client and d2d.
type Conn struct {
	path string
	c    net.Conn
	rd   *bufio.Reader
	// Timeout bounds one round trip (a long "step" needs more; 0 = none).
	Timeout time.Duration
}

// Dial connects to a control socket.
func Dial(path string, timeout time.Duration) (*Conn, error) {
	c, err := net.DialTimeout("unix", path, 3*time.Second)
	if err != nil {
		return nil, err
	}
	return &Conn{path: path, c: c, rd: bufio.NewReaderSize(c, 1<<20), Timeout: timeout}, nil
}

// DialWait keeps dialing until the socket answers "ping" or the context ends.
func DialWait(ctx context.Context, path string, timeout time.Duration) (*Conn, error) {
	for {
		c, err := Dial(path, timeout)
		if err == nil {
			if _, st, e := c.Do("ping"); e == nil && st == "ok" {
				return c, nil
			}
			c.Close()
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("d2dreplay: %s not answering: %w", path, ctx.Err())
		case <-time.After(250 * time.Millisecond):
		}
	}
}

// ErrBusy mirrors d2d's "err busy" (a second client).
var ErrBusy = errors.New("d2dreplay: control socket busy")

// Do sends one line and returns the payload lines and the status line ("ok", "ok <rest>" or "err <msg>").
// A transport error leaves the connection unusable (the stream may be out of step).
func (c *Conn) Do(line string) (lines []string, status string, err error) {
	if c.Timeout > 0 {
		c.c.SetDeadline(time.Now().Add(c.Timeout))
	} else {
		c.c.SetDeadline(time.Time{})
	}
	if _, err = io.WriteString(c.c, line+"\n"); err != nil {
		return nil, "", err
	}
	for {
		l, err := c.rd.ReadString('\n')
		if err != nil {
			if len(lines) == 0 && strings.HasPrefix(strings.TrimSpace(l), "err busy") {
				return nil, "", ErrBusy
			}
			return nil, "", err
		}
		l = strings.TrimRight(l, "\r\n")
		if l == "ok" || l == "err" || strings.HasPrefix(l, "ok ") || strings.HasPrefix(l, "err ") {
			if l == "err busy" && len(lines) == 0 {
				return nil, "", ErrBusy
			}
			return lines, l, nil
		}
		lines = append(lines, l)
	}
}

// Close closes the socket.
func (c *Conn) Close() error {
	if c == nil || c.c == nil {
		return nil
	}
	return c.c.Close()
}

// IsErr says whether a status line is an "err" reply.
func IsErr(status string) bool { return status == "err" || strings.HasPrefix(status, "err ") }

// Doer is what the fingerprint and the player need from a connection.
type Doer interface {
	Do(line string) (lines []string, status string, err error)
}

// Fingerprint is a checkpoint: the hash and a few readable fields for scrubbing.
type Fingerprint struct {
	Hash string
	Lv   int
	Area int
	Life string
}

// CheckpointVerbs are the reads whose replies make up the state hash, in hash order. They are DETLOOP's journal
// checkpoint verbs (agent/detloop internal/d2dctl/journal.go CheckpointVerbs, docs/d2d-determinism.md): "dump"
// without its time section (game_ms and wall fields differ by construction), the monster list and the ground items.
// Tests replace them (the d2dctl fake has no dump verb).
var CheckpointVerbs = []string{"dump player,stats,items,quests,ui,merc", "monsters", "ground"}

// Checkpoint computes HashV1 at the current tick: sha256 over, for each verb of CheckpointVerbs in order, every
// payload line followed by '\n', then the "ok" rest (empty for a bare ok), then one zero byte. That is byte for
// byte the detloop journal's checkpoint, so the first 16 hex digits equal its "C <tick> <hash>" lines and either
// tool can confirm the other. A separate "state" read fills the scrub fields (not part of the hash).
func Checkpoint(d Doer) (Fingerprint, error) {
	h := sha256.New()
	var fp Fingerprint
	for _, v := range CheckpointVerbs {
		lines, status, err := d.Do(v)
		if err != nil {
			return fp, err
		}
		if IsErr(status) {
			return fp, fmt.Errorf("%s: %s", v, status)
		}
		for _, l := range lines {
			h.Write([]byte(l))
			h.Write([]byte{'\n'})
		}
		h.Write([]byte(strings.TrimPrefix(strings.TrimPrefix(status, "ok"), " ")))
		h.Write([]byte{0})
	}
	fp.Hash = hex.EncodeToString(h.Sum(nil))
	st, status, err := d.Do("state")
	if err != nil {
		return fp, err
	}
	if !IsErr(status) {
		stateLine := strings.Join(append(st, strings.TrimPrefix(status, "ok ")), " ")
		for _, kv := range strings.Fields(stateLine) {
			k, v, _ := strings.Cut(kv, "=")
			switch k {
			case "level":
				fmt.Sscan(v, &fp.Lv)
			case "life":
				fp.Life = v
			case "area", "levelid":
				fmt.Sscan(v, &fp.Area)
			}
		}
	}
	if fp.Area == 0 {
		if lv, status, err := d.Do("debug level"); err == nil && !IsErr(status) {
			s := strings.TrimPrefix(status, "ok ")
			if len(lv) > 0 {
				s = lv[0]
			}
			fmt.Sscan(s, &fp.Area)
		}
	}
	return fp, nil
}

// FileSHA256 hashes a file (the d2d binary, a save).
func FileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

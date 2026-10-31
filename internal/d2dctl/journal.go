package d2dctl

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

// site names the first caller outside d2dctl, as "pkg.Func:line".
func site() string {
	var pcs [16]uintptr
	n := runtime.Callers(3, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	for {
		f, more := frames.Next()
		if !strings.Contains(f.Function, "/d2dctl.") && f.Function != "" {
			fn := f.Function
			if i := strings.LastIndex(fn, "/"); i >= 0 {
				fn = fn[i+1:]
			}
			return fmt.Sprintf("%s:%d", fn, f.Line)
		}
		if !more {
			return "?"
		}
	}
}

// Journal records every command the client sends, in order, with the lockstep tick it was sent at and a hash of the
// reply, plus a hash of the game state every Every ticks. Two journals of two same-seed runs are compared by
// cmd/d2ddet: the first line that differs is the first observable divergence (docs/d2d-determinism.md).
//
// Line format (tab separated, one record per line, "# ..." comments):
//
//	>	<tick>	<command line>	<reply>	<site>
//	C	<tick>	<state hash>
//
// <site> is the first caller outside this package (function name and line), so the comparer can say which code path
// sent a command the other run did not. It is not part of the comparison.
//
// <reply> is the first 12 hex digits of sha256 over the reply payload and rest, "err:<message>" for an err reply,
// "tick=<n>" for step and clock (their frame counter depends on the start-up frames, so the raw text is not hashed).
// Verbs in Skip (ping, info, clock, screenshot) are not recorded: the health monitor sends them on a wall timer, they
// read nothing the world depends on, and their position in the stream is noise.
//
// A checkpoint is the sha256 of the replies to CheckpointVerbs, taken right after the step that crossed a multiple of
// Every ticks; it is sent on the same connection (the caller holds the client's lock) and therefore appears in BOTH
// runs at the same tick, so the extra verbs do not themselves make the runs differ.
type Journal struct {
	Every int             // ticks between checkpoints (0: no checkpoints)
	Skip  map[string]bool // verbs (first word) not recorded

	// Raw, when non-nil, gets the full text of every reply to the verbs in RawVerbs (first word), one block per
	// exchange: "@ tick line" then the payload lines. For finding WHICH field of a reply differs between two runs
	// (D2DCTL_JOURNAL_RAW=state,...; side file <journal>.raw, not part of any comparison).
	Raw      io.Writer
	RawVerbs map[string]bool

	mu   sync.Mutex
	w    *bufio.Writer
	f    *os.File
	tick int
	n    int
}

// CheckpointVerbs are the reads whose replies make up a state hash. "dump" without the time section: game_ms and
// wall-time fields would differ by construction. Order matters (it is the hash input).
var CheckpointVerbs = []string{"dump player,stats,items,quests,ui,merc", "monsters", "ground"}

// DefaultSkip are the verbs the journal leaves out.
var DefaultSkip = map[string]bool{"ping": true, "info": true, "clock": true, "screenshot": true}

// OpenJournal starts recording to path (truncated). every is the checkpoint interval in ticks.
func (c *Client) OpenJournal(path string, every int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	j := &Journal{Every: every, Skip: DefaultSkip, f: f, w: bufio.NewWriter(f)}
	fmt.Fprintf(j.w, "# d2dctl journal v1 every=%d verbs=%s\n", every, strings.Join(CheckpointVerbs, ";"))
	c.mu.Lock()
	c.jr = j
	c.mu.Unlock()
	return nil
}

// RawReplies makes the open journal also write the full replies of the given verbs to w (see Journal.Raw).
func (c *Client) RawReplies(w io.Writer, verbs ...string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.jr == nil {
		return
	}
	c.jr.Raw, c.jr.RawVerbs = w, map[string]bool{}
	for _, v := range verbs {
		c.jr.RawVerbs[v] = true
	}
}

// CloseJournal flushes and closes the journal (no-op when none is open).
func (c *Client) CloseJournal() error {
	c.mu.Lock()
	j := c.jr
	c.jr = nil
	c.mu.Unlock()
	if j == nil {
		return nil
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	j.w.Flush()
	return j.f.Close()
}

// Tick is the last lockstep tick the journal saw (0 before the first step or clock reply).
func (j *Journal) Tick() int {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.tick
}

// ReplyHash is the journal's digest of a reply: 12 hex digits of sha256 over the payload lines and the ok rest. The
// "frame" field of a dump's time section is left out: it counts the process's frames, including the ones before the
// bot connected and every idle frame, so it differs between two otherwise identical runs (V: the first pair).
func ReplyHash(r *Reply) string {
	h := sha256.New()
	for _, l := range r.Lines {
		h.Write([]byte(camRe.ReplaceAllString(frameRe.ReplaceAllString(l, ""), "cam=")))
		h.Write([]byte{'\n'})
	}
	h.Write([]byte(r.Rest))
	return hex.EncodeToString(h.Sum(nil))[:12]
}

// frameRe and camRe: the process frame counter of dump's time section and the camera position of "state" (it follows the
// player with a smoothing that runs in wall time: V p4, "state" differed at tick 8 while dump, monsters and ground were
// equal; nothing in d2dsrc reads cam=). Both are masked so a divergence is a game-state difference, not a render one.
var (
	frameRe = regexp.MustCompile(`"frame":[0-9]+,?`)
	camRe   = regexp.MustCompile(`cam=[-0-9.,]+`)
)

// clockVerb reports whether the reply carries the lockstep counters (step, clock), whose frame field is not hashed.
func clockVerb(line string) bool {
	f := strings.Fields(line)
	return len(f) > 0 && (f[0] == "step" || f[0] == "clock")
}

// note records one exchange; returns true when a checkpoint is due (the caller then sends the checkpoint verbs on
// the connection it holds and passes the replies to checkpoint).
func (j *Journal) note(line string, rep *Reply, err error) (checkpoint bool) {
	f := strings.Fields(line)
	if len(f) == 0 {
		return false
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if clockVerb(line) && rep != nil && err == nil {
		prev := j.tick
		if t, ok := parseClock(rep).KV.Int("tick"); ok {
			j.tick = t
		}
		if j.Every > 0 && f[0] == "step" && j.tick/j.Every != prev/j.Every {
			checkpoint = true
		}
	}
	if j.Skip[f[0]] {
		return checkpoint
	}
	var r string
	switch {
	case err != nil:
		var ce *CmdError
		if errors.As(err, &ce) {
			r = "err:" + ce.Msg
		} else {
			r = "fail:" + strings.ReplaceAll(err.Error(), "\t", " ")
		}
	case clockVerb(line):
		r = fmt.Sprintf("tick=%d", j.tick)
	default:
		r = ReplyHash(rep)
	}
	if j.Raw != nil && j.RawVerbs[f[0]] && rep != nil {
		fmt.Fprintf(j.Raw, "@ %d %s\n%s\n%s\n", j.tick, line, strings.Join(rep.Lines, "\n"), rep.Rest)
	}
	j.n++
	fmt.Fprintf(j.w, ">\t%d\t%s\t%s\t%s\n", j.tick, line, r, site())
	if j.n%64 == 0 {
		j.w.Flush()
	}
	return checkpoint
}

// checkpoint writes the state hash of the replies to CheckpointVerbs.
func (j *Journal) checkpoint(reps []*Reply, err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if err != nil {
		fmt.Fprintf(j.w, "C\t%d\tfail:%s\n", j.tick, strings.ReplaceAll(err.Error(), "\t", " "))
		j.w.Flush()
		return
	}
	h := sha256.New()
	for _, r := range reps {
		for _, l := range r.Lines {
			h.Write([]byte(l))
			h.Write([]byte{'\n'})
		}
		h.Write([]byte(r.Rest))
		h.Write([]byte{0})
	}
	fmt.Fprintf(j.w, "C\t%d\t%s\n", j.tick, hex.EncodeToString(h.Sum(nil))[:16])
	j.w.Flush()
}

// journal is the hook called by Do and Prefetch with the client's lock held: it records the exchange and takes a
// checkpoint when one is due, on the same connection, so the extra reads land between the same two commands in
// every run.
func (c *Client) journal(line string, rep *Reply, err error) {
	j := c.jr
	if j == nil {
		return
	}
	if !j.note(line, rep, err) || c.conn == nil {
		return
	}
	reps := make([]*Reply, 0, len(CheckpointVerbs))
	for _, v := range CheckpointVerbs {
		r, cerr := c.exchange(v)
		if cerr != nil {
			var ce *CmdError
			if !errors.As(cerr, &ce) {
				c.drop(cerr)
			}
			j.checkpoint(nil, cerr)
			return
		}
		reps = append(reps, r)
	}
	j.checkpoint(reps, nil)
}

package d2dinput

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/input"
)

// The hold verb of fork branch didle/forkcaps-20261127 (docs/design/input-cast-verbs.md section 2a; V source read,
// apps/d2d/devctl/verbs.hpp): "cmd skillhold <id> <x> <y> <unit> <frames> <left 0|1>" answers at once
// "ok hold=<frames> skill=<id> left|right" and the fork then sends one held frame per 40 ms world tick until frames
// are sent or the hero dies, changes level or leaves the game; "cmd skillhold status" answers
// "held=<sent> casts=<n> frames=<n> state=running|done|dead|level|left". The hand flag is parsed with atoi, so it is
// 0 or 1, not the word "left"; no unit is -1.

// HoldTick is one held frame: a world tick of the fork.
const HoldTick = 40 * time.Millisecond

// holdMargin bounds the status polling beyond the hold itself (a devctl command goes out at the next world tick).
const holdMargin = 500 * time.Millisecond

// HoldResult is the end of the last hold as the fork reported it.
type HoldResult struct {
	Frames, Held, Casts int
	// State is done, dead, level or left; "running" when the polling bound ran out first; "" when status failed.
	State string
	Err   error // the status read that failed, if any
}

// LastHold returns the end of the last CastHold.
func (c *Controller) LastHold() HoldResult {
	c.holdMu.Lock()
	defer c.holdMu.Unlock()
	return c.lastHold
}

// HoldFrames is the number of held frames for a press of d: whole world ticks, rounded up, 1..2500 (the fork's limits).
func HoldFrames(d time.Duration) int {
	n := int((d + HoldTick - 1) / HoldTick)
	return min(max(n, 1), 2500)
}

// CastHold holds skill on the hand at the unit (0: the point at) for d and returns when the fork reports the hold
// over. Only a refused start is an error: once the fork accepted the hold the cast is sent, so a failed status read or
// a hold still running at the bound is recorded in LastHold and the Trace, never returned (an error would make the
// caller cast a second time by another path). d2d has no release message; the hold ends by itself.
func (c *Controller) CastHold(skill int, left bool, at input.Point, unit uint32, d time.Duration) error {
	x, y := c.cell(at)
	u, l := -1, 0
	if unit != 0 {
		u = int(unit)
	}
	if left {
		l = 1
	}
	frames := HoldFrames(d)
	line := fmt.Sprintf("cmd skillhold %d %s %s %d %d %d", skill, fcell(x), fcell(y), u, frames, l)
	if _, err := c.C.Do(c.ctx(), line); err != nil {
		return err
	}
	r := c.waitHold(frames, time.Duration(frames)*HoldTick+holdMargin)
	c.holdMu.Lock()
	c.lastHold = r
	c.holdMu.Unlock()
	if c.Trace != nil && (r.State != "done" || r.Err != nil) {
		c.Trace("input hold end", "skill", skill, "state", r.State, "held", r.Held, "frames", r.Frames, "err", fmt.Sprint(r.Err))
	}
	return nil
}

// waitHold polls the status on the controller's clock until the hold is not running or the bound runs out.
func (c *Controller) waitHold(frames int, bound time.Duration) HoldResult {
	clk := c.clk()
	end := clk.Now().Add(bound)
	for {
		rep, err := c.C.Do(c.ctx(), "cmd skillhold status")
		if err != nil {
			return HoldResult{Frames: frames, Err: err}
		}
		r := parseHold(rep)
		if r.State != "running" || !clk.Now().Before(end) {
			return r
		}
		clk.Sleep(min(c.poll(), HoldTick*time.Duration(max(frames-r.Held, 1))))
	}
}

// parseHold reads the key=value pairs of a status reply.
func parseHold(rep *d2dctl.Reply) HoldResult {
	var r HoldResult
	for _, kv := range strings.Fields(rep.Text()) {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		n, _ := strconv.Atoi(v)
		switch k {
		case "held":
			r.Held = n
		case "casts":
			r.Casts = n
		case "frames":
			r.Frames = n
		case "state":
			r.State = v
		}
	}
	return r
}

func fcell(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

var _ input.HoldCaster = (*Controller)(nil)

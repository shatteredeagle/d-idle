package d2dreplay

import (
	"context"
	"fmt"
	"time"
)

// Player re-runs a replay against a d2d of the same recipe in lockstep and verifies the checkpoints.
type Player struct {
	Conn Doer
	// GapMin: recorded wall gaps at least this long are reproduced by sleeping (level loads and menus run on real
	// time in the fork, docs/d2d-fork.md section 3); 0 never sleeps. GapMax caps one sleep (0: 10 s).
	GapMin, GapMax time.Duration
	Until          int                  // stop after the first step that reaches this tick (0: play everything)
	AfterStep      func(tick int) error // render hook, called after every step
	Logf           func(string, ...any)
}

// Divergence is the first checkpoint that did not match.
type Divergence struct {
	Index    int // checkpoint number (0-based)
	Tick     int
	Want     string
	Got      string
	Lv, Area int
}

// Result of a playback.
type Result struct {
	OK          bool
	Checkpoints int // verified (matching) checkpoints
	Ticks       int
	Verbs       int
	ReplyDiffs  int // verbs whose ok/err outcome differed from the recording (a hint, not a verdict)
	Diverged    *Divergence
	Stopped     string // why playback ended early ("restart marker at tick N", "until tick N"), "" when complete
}

func (r Result) String() string {
	switch {
	case r.Diverged != nil:
		d := r.Diverged
		return fmt.Sprintf("DIVERGED at tick %d (checkpoint %d, virtual %s): want %s got %s (clvl %d area %d); %d checkpoints matched before",
			d.Tick, d.Index, (time.Duration(d.Tick) * TickMS * time.Millisecond).Round(time.Second), short(d.Want), short(d.Got), d.Lv, d.Area, r.Checkpoints)
	case r.Stopped != "":
		return fmt.Sprintf("OK so far: %d checkpoints matched, %d ticks, %d verbs; stopped: %s", r.Checkpoints, r.Ticks, r.Verbs, r.Stopped)
	}
	return fmt.Sprintf("OK: %d checkpoints matched, %d ticks (virtual %s), %d verbs, %d reply differences",
		r.Checkpoints, r.Ticks, (time.Duration(r.Ticks) * TickMS * time.Millisecond).Round(time.Second), r.Verbs, r.ReplyDiffs)
}

func short(h string) string {
	if len(h) > 16 {
		return h[:16] + ".."
	}
	return h
}

func (p *Player) logf(f string, a ...any) {
	if p.Logf != nil {
		p.Logf(f, a...)
	}
}

// Run plays rp. The returned error is a transport or protocol failure; a divergence is in Result, not an error.
func (p *Player) Run(ctx context.Context, rp *Replay) (Result, error) {
	var res Result
	if rp.Header.Hash != HashV1 {
		return res, fmt.Errorf("d2dreplay: checkpoint hash %q not implemented (this tool computes %s)", rp.Header.Hash, HashV1)
	}
	gapMax := p.GapMax
	if gapMax == 0 {
		gapMax = 10 * time.Second
	}
	tick, cpIndex := 0, 0
	for _, e := range rp.Entries {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		switch e.Kind() {
		case "verb":
			if e.Verb == "quit" {
				continue
			}
			if err := ValidVerb(e.Verb); err != nil {
				return res, fmt.Errorf("d2dreplay: refusing to send: %w", err)
			}
			if e.T < 0 || e.T > maxTick {
				return res, fmt.Errorf("d2dreplay: tick %d out of range", e.T)
			}
			if p.GapMin > 0 && time.Duration(e.W)*time.Millisecond >= p.GapMin {
				d := min(time.Duration(e.W)*time.Millisecond, gapMax)
				select {
				case <-ctx.Done():
					return res, ctx.Err()
				case <-time.After(d):
				}
			}
			if e.T != tick {
				return res, fmt.Errorf("d2dreplay: file says tick %d before %q, player is at %d", e.T, e.Verb, tick)
			}
			_, status, err := p.Conn.Do(e.Verb)
			if err != nil {
				return res, fmt.Errorf("d2dreplay: %q at tick %d: %w", e.Verb, tick, err)
			}
			res.Verbs++
			if IsErr(status) != (e.Err != "") {
				res.ReplyDiffs++
				p.logf("play: reply differs at tick %d for %q: recorded %q, now %q", tick, e.Verb, e.Err, status)
			}
			if n, ok := StepCount(e.Verb); ok && !IsErr(status) {
				tick += n
				res.Ticks = tick
				if p.AfterStep != nil {
					if err := p.AfterStep(tick); err != nil {
						return res, err
					}
				}
				if p.Until > 0 && tick >= p.Until {
					res.Stopped = fmt.Sprintf("until tick %d", p.Until)
					return res, nil
				}
			}
		case "checkpoint":
			if e.C != tick {
				return res, fmt.Errorf("d2dreplay: checkpoint %d is at tick %d but the player is at %d", cpIndex, e.C, tick)
			}
			fp, err := Checkpoint(p.Conn)
			if err != nil {
				return res, fmt.Errorf("d2dreplay: checkpoint at tick %d: %w", tick, err)
			}
			if fp.Hash != e.H {
				res.Diverged = &Divergence{Index: cpIndex, Tick: tick, Want: e.H, Got: fp.Hash, Lv: fp.Lv, Area: fp.Area}
				return res, nil
			}
			res.Checkpoints++
			cpIndex++
		case "restart":
			res.Stopped = fmt.Sprintf("restart marker at tick %d (a d2d restart is not replayed)", e.T)
			return res, nil
		}
	}
	res.OK = true
	return res, nil
}

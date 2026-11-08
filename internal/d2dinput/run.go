package d2dinput

import (
	"os"
	"strconv"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/data"
)

// RunPolicy turns d2d's run toggle on and off by stamina, the way the original character runs while stamina
// lasts (docs/d2d-run.md). d2d has no "cmd run" verb on the channel: the toggle is the key "r" (V: the fork's
// kToggleRun is 'R', apps/d2d/town.cpp), and `running` is not reported (V live: debug players pmode is -1 both
// walking and running). The controller therefore keeps a belief; a new game always starts walking, so Reset
// clears it.
//
// d2d forces a walk at stamina 0 but leaves the toggle on (S, world.cpp stamina_frame) and a walk at less than
// 1.0 stamina never regenerates, so the policy switches the run off BEFORE it hits zero (Off) and on again once
// the bar has refilled (On, standing or walking regenerates).
type RunPolicy struct {
	// On: resume running at or above this fraction of max stamina. Off: stop running below it.
	On, Off float64
}

// DefaultRunPolicy: defaults from the drain/regen numbers in docs/d2d-run.md.
func DefaultRunPolicy() *RunPolicy { return &RunPolicy{On: 0.25, Off: 0.08} }

// RunPolicyFromEnv returns DefaultRunPolicy, nil (never run, the old behaviour) when D2D_RUN is "off" or "0",
// and honours D2D_RUN_ON / D2D_RUN_OFF (fractions) for experiments.
func RunPolicyFromEnv() *RunPolicy {
	switch os.Getenv("D2D_RUN") {
	case "off", "0":
		return nil
	}
	p := DefaultRunPolicy()
	if v, err := strconv.ParseFloat(os.Getenv("D2D_RUN_ON"), 64); err == nil {
		p.On = v
	}
	if v, err := strconv.ParseFloat(os.Getenv("D2D_RUN_OFF"), 64); err == nil {
		p.Off = v
	}
	return p
}

// want returns the run state the policy wants given the stamina and the current state.
func (p *RunPolicy) want(cur, max int, running bool) bool {
	if max <= 0 {
		return running
	}
	f := float64(cur) / float64(max)
	switch {
	case running && f < p.Off:
		return false
	case !running && f >= p.On:
		return true
	}
	return running
}

// stamina parses "stamina=cur/max" from a state reply (raw 8.8 fixed point, only the ratio matters).
func stamina(s d2dctl.State) (cur, max int, ok bool) {
	p, ok := s.Pair("stamina")
	return p.Cur, p.Max, ok
}

// ResetRun forgets the belief about the toggle (a new game process starts walking: UNVERIFIED live for the
// first move after a restart, the proof recipe is in docs/fixrun.md). Callers: d2denv.Proc.WaitInGame (OnInGame,
// the backend's every fresh game / chicken re-entry / restart) and d2dinput.Process.WaitInGame.
func (c *Controller) ResetRun() {
	c.runMu.Lock()
	c.running = false
	c.runHeals, c.chainStart = 0, time.Time{}
	c.runResets++
	c.runMu.Unlock()
}

const (
	// driftChain: moves closer together than this belong to one continuous walk.
	driftChain = 1500 * time.Millisecond
	// driftAfter: running drains stamina within a second or two (docs/d2d-run.md: 3.9 per second of 74), so a walk
	// this long at full stamina is not running.
	driftAfter = 4 * time.Second
	// driftMaxHeals bounds the corrections per controller: the belief cannot be read, so never keep toggling.
	driftMaxHeals = 2
)

// noDrain tracks a chain of moves (called before each move with the stamina just read) and reports true once, after
// driftAfter of continuous moves outside town at full stamina while the run toggle is believed on.
func (c *Controller) noDrain(cur, max int) bool {
	now := c.clk().Now()
	c.runMu.Lock()
	defer c.runMu.Unlock()
	if c.chainStart.IsZero() || now.Sub(c.lastMoveAt) > driftChain || cur < max || data.IsTown(c.lastLevel) {
		c.chainStart = now
	}
	c.lastMoveAt = now
	if cur < max || data.IsTown(c.lastLevel) || c.runHeals >= driftMaxHeals || now.Sub(c.chainStart) < driftAfter {
		return false
	}
	c.chainStart = now
	return true
}

// RunInfo is the observable run state for the minute log line (change fixrun). Believed is OUR belief of the
// toggle; the true toggle is not readable on d2d (V live per docs/d2d-run.md: pmode -1 both ways, `state` has no
// run field). Walking is the raw `walking=` value of the last state read, logged only so a live proof can
// correlate it with Believed (meaning UNVERIFIED). Cur/Max are the raw stamina pair of the last read (0/0 before
// the first move). Presses and Resets count run-key presses and belief resets since the controller was built.
type RunInfo struct {
	Believed        bool
	Cur, Max        int
	Presses, Resets int
	Walking         string
}

// RunInfo returns the run observables; safe to call from another goroutine.
func (c *Controller) RunInfo() RunInfo {
	c.runMu.Lock()
	defer c.runMu.Unlock()
	return RunInfo{Believed: c.running, Cur: c.stCur, Max: c.stMax, Presses: c.runPresses, Resets: c.runResets, Walking: c.walkRaw}
}

// LogKV is RunInfo as slog key/value pairs for the bot's minute line (the bot asks its input for this through an
// optional interface, so it need not import d2dinput). runShare is not here: d2dsoak derives it from
// runBelieved over the minute lines.
func (c *Controller) LogKV() []any {
	i := c.RunInfo()
	return []any{"runBelieved", i.Believed, "stam", i.Cur, "stamMax", i.Max, "runKeys", i.Presses, "runResets", i.Resets, "walkingRaw", i.Walking}
}

// manageRun is called before every move command. Errors are not fatal: a failed read leaves the run as it is.
//
// No self-heal by keypress: a wrong press while the character really runs would switch the run OFF, and the
// stamina signal cannot tell "walking" from "running in town" (no drain in town, docs/d2d-run.md), so the
// belief is corrected only by ResetRun at a game start and the divergence is made observable (RunInfo).
func (c *Controller) manageRun() {
	p := c.Run
	if p == nil {
		return
	}
	st, _, err := c.C.State(c.ctx())
	if err != nil {
		return
	}
	cur, max, ok := stamina(st)
	if !ok {
		return
	}
	c.runMu.Lock()
	c.stCur, c.stMax = cur, max
	c.walkRaw, _ = st.Get("walking")
	running := c.running
	c.runMu.Unlock()
	if running && c.noDrain(cur, max) {
		// WALKSMOOTH: believed running, moving for seconds outside town, and stamina never left the maximum: the
		// character is walking (V live: a Paladin session logged stam == stamMax in every minute line while it
		// travelled; the belief starts wrong if the game starts with run on). The key toggles, so press it once.
		if c.C.Key(c.ctx(), "r") == nil {
			c.runMu.Lock()
			c.runPresses++
			c.runHeals++
			c.runMu.Unlock()
		}
		return
	}
	if want := p.want(cur, max, running); want != running {
		if c.C.Key(c.ctx(), "r") == nil {
			c.runMu.Lock()
			c.running = want
			c.runPresses++
			c.runMu.Unlock()
		}
	}
}

package d2dinput

import (
	"os"
	"strconv"

	"d2lod-bot/internal/d2dctl"
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

// ResetRun forgets the belief about the toggle (a new game process starts walking).
func (c *Controller) ResetRun() { c.running = false }

// manageRun is called before every move command. Errors are not fatal: a failed read leaves the run as it is.
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
	if want := p.want(cur, max, c.running); want != c.running {
		if c.C.Key(c.ctx(), "r") == nil {
			c.running = want
		}
	}
}

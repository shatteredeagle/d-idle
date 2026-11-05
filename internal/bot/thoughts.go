package bot

import (
	"fmt"
	"time"

	"d2lod-bot/internal/ladder"
)

// EnableLadder installs the decision ladder when the ladder.enabled knob is on and returns it; with the knob off it
// does nothing and returns nil, so every wired call site keeps its legacy path (docs/t0-slice.md). The thought sink is
// the bot log, the fade clock is the bot's own clock counted from this call, and a thought opened while the runner
// clears (a fight may be on) waits to be asked until the runner is in another state. answers may be nil (no player:
// thoughts open and fade; the defaults act). stateDir "" keeps the answers in memory.
func (r *Runner) EnableLadder(stateDir, profile string, answers ladder.AnswerSource) (*ladder.Ladder, error) {
	if r.e.Tune == nil || r.e.Tune.Ladder.Enabled == 0 {
		return nil, nil
	}
	t0 := r.e.Clock.Now()
	clk := ladder.ClockFunc(func() time.Duration { return r.e.Clock.Now().Sub(t0) })
	sink := ladder.LogSink{Info: func(msg string, kv ...any) { r.e.Log.Info(msg, kv...) }}
	l, err := ladder.Open(stateDir, profile, clk, sink, answers, func() bool { return r.State() == StateClearing })
	if err != nil {
		return nil, fmt.Errorf("bot: ladder: %w", err)
	}
	ladder.Install(l)
	return l, nil
}

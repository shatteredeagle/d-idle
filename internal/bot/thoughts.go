package bot

import (
	"fmt"
	"time"

	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/ladder"
)

// EnableLadder installs the decision ladder when the ladder.enabled knob is on and returns it; with the knob off it
// does nothing and returns nil, so every wired call site keeps its legacy path (docs/t0-slice.md). The thought sink is
// the bot log, the fade clock is the bot's own clock counted from this call, and a thought opened while the runner
// clears (a fight may be on) waits to be asked until the runner is in another state. answers may be nil (no player:
// thoughts open and fade; the defaults act). stateDir "" keeps the answers in memory.
func (r *Runner) EnableLadder(stateDir, profile string, answers ladder.AnswerSource) (*ladder.Ladder, error) {
	return r.EnableLadderWith(stateDir, profile, func(*decide.Store, *decide.Catalogue, decide.Clock) (ladder.Sink, ladder.AnswerSource, error) {
		return nil, answers, nil
	})
}

// LadderWire connects the ladder to what shows its thoughts and answers them. It gets the store, the catalogue and the
// fade clock the ladder will use (a watch screen needs the same three) and returns the sink and the answer source; a nil
// sink is the bot log only, a nil source means no scripted player.
type LadderWire func(st *decide.Store, cat *decide.Catalogue, clk decide.Clock) (ladder.Sink, ladder.AnswerSource, error)

// EnableLadderWith is EnableLadder with the sink and answer source built by wire after the store exists. Knob off: it
// installs nothing and does not call wire.
func (r *Runner) EnableLadderWith(stateDir, profile string, wire LadderWire) (*ladder.Ladder, error) {
	if r.e.Tune == nil || r.e.Tune.Ladder.Enabled == 0 {
		return nil, nil
	}
	t0 := r.e.Clock.Now()
	clk := ladder.ClockFunc(func() time.Duration { return r.e.Clock.Now().Sub(t0) })
	cat, err := decide.Shipped()
	if err != nil {
		return nil, fmt.Errorf("bot: ladder: %w", err)
	}
	var st *decide.Store
	if stateDir == "" {
		st, err = decide.New(cat, profile, clk)
	} else {
		st, err = decide.Open(stateDir, profile, cat, clk)
	}
	if err != nil {
		return nil, fmt.Errorf("bot: ladder: %w", err)
	}
	logSink := ladder.LogSink{Info: func(msg string, kv ...any) { r.e.Log.Info(msg, kv...) }}
	sink, answers, err := wire(st, cat, clk)
	if err != nil {
		return nil, fmt.Errorf("bot: ladder: %w", err)
	}
	var s ladder.Sink = logSink
	if sink != nil {
		s = ladder.Tee{logSink, sink}
	}
	l, err := ladder.New(ladder.Config{Cat: cat, Store: st, Clock: clk, Sink: s, Answers: answers, InFight: r.engaged})
	if err != nil {
		return nil, fmt.Errorf("bot: ladder: %w", err)
	}
	ladder.Install(l)
	return l, nil
}

// engaged reports an actual engagement for the ladder: a living hostile within the fight radius of the last snapshot
// (the same radius and hostile test the safety rules use). Walking through a cleared field is not a fight, so a
// question reaches the player there (ADR 0016: the hero never stops).
func (r *Runner) engaged() bool {
	s := r.latest()
	return s != nil && r.hostileNear(s, r.e.Tune.Combat.FightRadius) > 0
}

// pumpLadder runs once per tick (Runner.step): it puts queued thoughts to the player and delivers answers that are
// due, so a question opened at one decision point does not wait for the next Decide call. Ladder.Pump holds no game
// access and holds nothing while engaged (Config.InFight). No ladder installed: nothing runs.
func (r *Runner) pumpLadder() {
	if l := ladder.Active(); l != nil {
		l.Pump()
	}
}

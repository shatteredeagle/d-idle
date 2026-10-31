package cheat

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrCheatsDisabled is returned by a Gate that was not switched on.
var ErrCheatsDisabled = errors.New("cheat: cheats are disabled (start the runner with --allow-cheats, test mode only)")

// Gate is the API sketch for the bot runner (NOT wired into cmd/d2bot yet). The runner builds one Gate from
// its flag: `--allow-cheats` defaults to false in every mode and must never be set by a production profile.
// The bot and its tests call Gate.Do instead of cheat.Do, so a missing flag turns every cheat into an error
// and nothing reaches the game or the ledger's "success" lines.
//
//	gate := cheat.Gate{Allow: *allowCheats, Env: cheat.DefaultEnv(), Actor: "bot"}
//	if _, err := gate.Do(ctx, cheat.KindLevel, map[string]any{"value": 25}, "test: reach level 25 fast"); err != nil { ... }
//	stop := gate.God(ctx, "test: survive the act 1 run")   // runs until stop() or ctx ends
//	defer stop()
//
// The statistics side asks cheat.Cheated(runStart) (or Ledger.Cheated(runStart, character)) and flags or
// excludes the run.
type Gate struct {
	Allow bool
	Env   *Env
	Actor string
}

func (g Gate) Do(ctx context.Context, kind Kind, params map[string]any, reason string) (*Result, error) {
	if !g.Allow {
		return nil, ErrCheatsDisabled
	}
	return g.Env.Do(ctx, Request{Kind: kind, Params: params, Reason: reason, Actor: g.Actor})
}

// God starts invincibility in the background and returns the function that ends it (and restores max life).
// The goroutine ends when stop is called or ctx is cancelled; errors are returned by stop (safe to call
// more than once). Max life is restored on both; a killed process is repaired by the next start (godstate.go).
func (g Gate) God(ctx context.Context, reason string) (stop func() error) {
	if !g.Allow {
		return func() error { return ErrCheatsDisabled }
	}
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() {
		_, err := g.Env.Do(ctx, Request{Kind: KindGod, Reason: reason, Actor: g.Actor,
			Params: map[string]any{"on": true, "for": (24 * time.Hour).String()}})
		if errors.Is(err, context.Canceled) {
			err = nil
		}
		done <- err
	}()
	var once sync.Once
	var res error
	return func() error {
		once.Do(func() { cancel(); res = <-done })
		return res
	}
}

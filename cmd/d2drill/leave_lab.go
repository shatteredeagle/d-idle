package main

import (
	"context"
	"fmt"
)

// gameCycler is the part of d2denv.Cycler the chicken hook needs.
type gameCycler interface {
	LeaveGame(ctx context.Context) error
	EnterGame(ctx context.Context, diff string) error
}

// leaveHook is Fighter.Leave for the rig: the real Save and Exit (the d2d process quits, its socket goes away) and a
// rejoin, so the run goes on in a fresh process. EnterGame returns at once when LeaveGame already re-entered (Reenter).
// done is told after the exit, before the rejoin: after the rejoin the snapshot shows InGame again, so the lab cannot
// see the exit any other way.
func leaveHook(c gameCycler, done func()) func(context.Context) error {
	return func(ctx context.Context) error {
		if err := c.LeaveGame(ctx); err != nil {
			return fmt.Errorf("save and exit: %w", err)
		}
		done()
		if err := c.EnterGame(ctx, ""); err != nil {
			return fmt.Errorf("rejoin: %w", err)
		}
		return nil
	}
}

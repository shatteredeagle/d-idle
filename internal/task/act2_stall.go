package task

import (
	"context"
	"errors"
)

// StallRecoverer is the optional part of the bot's Actions for walks that stall inside monster packs (V live,
// docs/act2.md nav3: plain nav sat at one subtile for 5+ minutes in the Maggot Lair and the Far Oasis chain; a combat
// Clear freed it, and for a Sorceress teleport-to-adjacent-then-ToArea crossed the chain in seconds).
type StallRecoverer interface {
	// ClearAround runs a bounded combat.Clear around the character (nav.Yield does the same inside one walk).
	ClearAround(ctx context.Context) error
	// TeleportToward teleports to the known exit or Adjacent contact into area a and then calls ToArea (Sorceress,
	// V live). It returns ErrNoTeleport when the character cannot teleport or no exit is known.
	TeleportToward(ctx context.Context, a Area) error
}

// ErrNoTeleport: the character has no Teleport or no hop target is known; the caller falls back to Clear + retry.
var ErrNoTeleport = errors.New("task: no teleport hop available")

// stallRounds bounds the recover-and-retry loop of one travel step.
const stallRounds = 2

// StepTravelA2 is StepTravel with a stall recovery for Act 2: when Travel fails with a retryable error (stuck,
// timeout, no path) and the bot can recover, try the teleport hop first (documented option, Sorceress), else clear the
// pack in the way, then travel again. Prereq, resource, danger, death and fatal faults go straight back to the executor.
func StepTravelA2(id, title string, a Area, done Pred) Step {
	s := StepTravel(id, title, a, done)
	plain := s.Do
	s.Do = func(ctx context.Context, x Actions) error {
		err := plain(ctx, x)
		rec, ok := x.(StallRecoverer)
		for round := 0; err != nil && ok && round < stallRounds && stalled(err); round++ {
			if ctx.Err() != nil {
				return err
			}
			if terr := rec.TeleportToward(ctx, a); terr == nil {
				return nil
			} else if !errors.Is(terr, ErrNoTeleport) && !stalled(terr) {
				return terr
			}
			if cerr := rec.ClearAround(ctx); cerr != nil && Classify(cerr) != KindUI {
				return cerr
			}
			err = plain(ctx, x)
		}
		return err
	}
	return s
}

func stalled(err error) bool {
	switch Classify(err) {
	case KindUI, KindTimeout, KindNotFound:
		return true
	}
	return false
}

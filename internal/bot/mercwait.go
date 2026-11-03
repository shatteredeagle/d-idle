package bot

import "context"

// MercWait is set by the merc wiring (cmd/d2bot/mercwire.go): after a town portal or a waypoint the merc arrives a
// moment after the player, so the bot waits for her (bounded, one log line). Nil leaves the bot as it was.
var MercWait func(ctx context.Context, why string)

func (r *Runner) waitForMerc(ctx context.Context, why string) {
	if f := MercWait; f != nil && ctx.Err() == nil {
		f(ctx, why)
	}
}

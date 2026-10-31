package launch

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Keeper restarts the game after a crash, exit or hang and brings the character back into the game.
// Every side effect is a function field so the logic is tested offline; Wire fills them from a Config.
type Keeper struct {
	Mon *Monitor
	// Kill ends the old game process and any crash reporter (no-op when already gone).
	Kill func() error
	// Launch starts the game and returns when Game.exe exists.
	Launch func(ctx context.Context) error
	// Ready waits until the client can take input and its memory is readable (window up, attach),
	// and runs the mute step; returns the Flow to drive the menus with.
	Ready func(ctx context.Context) (*Flow, error)
	Char  CharSpec
	Log   func(string)
	Sleep func(time.Duration)

	// MaxRestarts within Window is the crash-loop guard (defaults 3 in 10 min).
	MaxRestarts int
	Window      time.Duration
	starts      []time.Time
}

// ErrCrashLoop means the game keeps dying; the caller should alert instead of restarting again.
var ErrCrashLoop = errors.New("game restarted too often")

func (k *Keeper) logf(f string, a ...any) {
	if k.Log != nil {
		k.Log(fmt.Sprintf(f, a...))
	}
}

// Step samples the Monitor once. On Crashed, Exited or Hung it runs Restart. It returns the event.
func (k *Keeper) Step(ctx context.Context, now time.Time) (Event, error) {
	ev := k.Mon.Step(now)
	switch ev {
	case EventCrashed, EventExited, EventHung:
		k.logf("monitor: %s, restarting", ev)
		return ev, k.Restart(ctx, now)
	}
	return ev, nil
}

// Restart kills what is left, launches the game and runs Flow.Start for the character. Menu state after
// a crash is unknown, which Start handles from any screen. A character that was mid-game is restored from
// its last save (Save and Exit or the autosave on area change; progress since is lost, by design of D2).
func (k *Keeper) Restart(ctx context.Context, now time.Time) error {
	max, win := k.MaxRestarts, k.Window
	if max == 0 {
		max = 3
	}
	if win == 0 {
		win = 10 * time.Minute
	}
	kept := k.starts[:0]
	for _, t := range k.starts {
		if now.Sub(t) < win {
			kept = append(kept, t)
		}
	}
	k.starts = kept
	if len(k.starts) >= max {
		return fmt.Errorf("%w: %d in %s", ErrCrashLoop, len(k.starts), win)
	}
	k.starts = append(k.starts, now)
	if err := k.Kill(); err != nil {
		return fmt.Errorf("kill: %w", err)
	}
	if k.Sleep != nil {
		k.Sleep(2 * time.Second) // let wineserver release the window and save files
	} else {
		time.Sleep(2 * time.Second)
	}
	if err := k.Launch(ctx); err != nil {
		return fmt.Errorf("launch: %w", err)
	}
	f, err := k.Ready(ctx)
	if err != nil {
		return fmt.Errorf("ready: %w", err)
	}
	if err := f.Start(ctx, k.Char); err != nil {
		return fmt.Errorf("start %q: %w", k.Char.Name, err)
	}
	k.logf("back in game as %q", k.Char.Name)
	return nil
}

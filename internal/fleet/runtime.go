package fleet

import (
	"context"
	"errors"
	"io"
	"time"

	"d2lod-bot/internal/launch"
)

// Runtime is everything the fleet needs from the game side of one instance. LifeRuntime is the real one.
type Runtime interface {
	Start(ctx context.Context, a Alloc, s Spec) error // prefix and Xvfb on demand, Game.exe, the d2input helper
	Login(ctx context.Context, a Alloc, s Spec) error // OOG: main menu to in-game as s.Character
	Alive(ctx context.Context, a Alloc) (bool, error) // the game process exists
	LeaveGame(ctx context.Context, a Alloc) error     // OOG: save and exit to the main menu
	Stop(ctx context.Context, a Alloc) error          // kill game, wineserver and Xvfb of the instance
}

// Preparer is the Runtime of a fleet whose bot runner owns the game (GameOwner): Prepare creates the prefix,
// the slot's Xvfb and the d2input helper but starts no Game.exe, and the fleet does no OOG login or leave.
type Preparer interface {
	Prepare(ctx context.Context, a Alloc, s Spec) error
}

// GameStopper is optionally implemented by a Runtime that can end only the game process (Game.exe and its
// wineserver) and leave the Xvfb and helper of the slot up.
type GameStopper interface {
	StopGame(ctx context.Context, a Alloc) error
}

// GameOwner is implemented by a BotRunner whose d2bot launches Game.exe itself (d2bot --managed as the
// game's parent and subreaper, the only model that reads memory under Yama ptrace_scope=1 without caps).
type GameOwner interface{ OwnsGame() bool }

// LifeRuntime implements Runtime over launch.Life. Flow builds the OOG flow of an instance (memory source and
// input client of that instance, e.g. mem.OpenInstance(id) and input.NewClient with D2BOT_INSTANCE set);
// the fleet cannot import those without choosing a process model, so the caller supplies it.
//
// UNVERIFIED live: launch.Life itself is verified offline only (docs/lifecycle.md), and the helper is
// started here with EnsureHelper after the game as docs/multi.md describes.
type LifeRuntime struct {
	Life         *launch.Life
	Flow         func(id string) (*launch.Flow, io.Closer, error) // the closer (memory handle) may be nil
	LeaveTimeout time.Duration                                    // default 20 s
}

func (r LifeRuntime) Start(ctx context.Context, a Alloc, _ Spec) error {
	if err := r.Life.Start(ctx, a.ID); err != nil {
		return err
	}
	return r.Life.EnsureHelper(ctx, a.ID)
}

// Prepare: EnsureHelper creates the prefix and the Xvfb display on demand, then starts the helper.
func (r LifeRuntime) Prepare(ctx context.Context, a Alloc, _ Spec) error {
	return r.Life.EnsureHelper(ctx, a.ID)
}

func (r LifeRuntime) Login(ctx context.Context, a Alloc, s Spec) error {
	f, done, err := r.flow(a)
	if err != nil {
		return err
	}
	defer done()
	return f.Start(ctx, launch.CharSpec{Name: s.Character, Class: s.Class, Create: s.Class != "", Hardcore: s.Hardcore, Difficulty: s.Difficulty})
}

func (r LifeRuntime) Alive(_ context.Context, a Alloc) (bool, error) {
	return len(r.Life.GamePIDs(a.ID)) > 0, nil
}

func (r LifeRuntime) LeaveGame(ctx context.Context, a Alloc) error {
	f, done, err := r.flow(a)
	if err != nil {
		return err
	}
	defer done()
	t := r.LeaveTimeout
	if t == 0 {
		t = 20 * time.Second
	}
	return f.LeaveGame(ctx, t)
}

func (r LifeRuntime) Stop(ctx context.Context, a Alloc) error { return r.Life.Stop(ctx, a.ID) }

func (r LifeRuntime) StopGame(ctx context.Context, a Alloc) error { return r.Life.StopGame(ctx, a.ID) }

func (r LifeRuntime) flow(a Alloc) (*launch.Flow, func(), error) {
	if r.Flow == nil {
		return nil, nil, errors.New("fleet: LifeRuntime.Flow not set")
	}
	f, c, err := r.Flow(a.ID)
	if err != nil {
		return nil, nil, err
	}
	return f, func() {
		if c != nil {
			_ = c.Close()
		}
	}, nil
}

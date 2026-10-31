package boss

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// FarmOps is what the per-run loop needs besides the boss runner. The bot adapter maps it to the OOG/new-game
// tooling, the town chores (heal, repair, sell, restock; internal/town) and the character reads.
type FarmOps interface {
	// Char reads the character for the scheduler. Called before every run (level, difficulty and flags change).
	Char(ctx context.Context) (FarmChar, error)
	// NewGame leaves the current game and creates a fresh one, arriving in town (monsters and bosses respawn).
	NewGame(ctx context.Context) error
	// Chores heals, repairs, sells, restocks potions and tomes, and revives the merc in town.
	Chores(ctx context.Context) error
}

// BossKiller is the Runner/Act45Runner/FarmRunner method set the loop uses.
type BossKiller interface {
	KillBoss(ctx context.Context, name string) error
	LastResult() Result
}

// FarmOpts configures FarmLoop.
type FarmOpts struct {
	Goal      string // xp (default), runes, uniques
	Fixed     string // run the named run every time instead of ranking ("" = scheduler)
	Limits    Limits
	FirstHere bool // the first run starts in the current game (no NewGame before it)
	Log       func(format string, args ...any)
	// OnRun is called after every counted run (not after a skipped too-weak one); killed is true when every step killed.
	OnRun func(run string, killed bool)
}

// RunStat is the tally of one run name.
type RunStat struct {
	Runs, Kills, Fails, Loot int
	Time                     time.Duration
}

// FarmReport says what a loop did.
type FarmReport struct {
	Runs    int // runs started
	Kills   int // runs in which every step killed its boss
	Deaths  int
	Loot    int
	Elapsed time.Duration
	Stop    string // why the loop ended: runs, time, deaths, fails, none (no run allowed), error, ctx
	ByRun   map[string]*RunStat
}

// FarmLoop plays runs one after another: new game, town chores, the run's bosses, loot (the boss scripts loot and
// leave for town themselves), then the next pick. It is meant for the 'runs:NAME:N' goal. now/sleep come from w (a
// World) so the loop is testable on the fake clock.
func FarmLoop(ctx context.Context, w World, ops FarmOps, k BossKiller, tab FarmTable, o FarmOpts) (FarmReport, error) {
	rep := FarmReport{ByRun: map[string]*RunStat{}}
	t0 := w.Now()
	logf := o.Log
	if logf == nil {
		logf = func(string, ...any) {}
	}
	skip := map[string]bool{}
	failsOf := map[string]int{}
	consec := 0
	first := o.FirstHere
	done := func(why string, err error) (FarmReport, error) {
		rep.Stop, rep.Elapsed = why, w.Now().Sub(t0)
		return rep, err
	}
	for {
		if err := ctx.Err(); err != nil {
			return done("ctx", err)
		}
		switch l := o.Limits; {
		case l.Runs > 0 && rep.Runs >= l.Runs:
			return done("runs", nil)
		case l.Duration > 0 && w.Now().Sub(t0) >= l.Duration:
			return done("time", nil)
		case l.Deaths > 0 && rep.Deaths >= l.Deaths:
			return done("deaths", nil)
		case consec >= l.consec():
			return done("fails", nil)
		}
		c, err := ops.Char(ctx)
		if err != nil {
			return done("error", fmt.Errorf("farm: character: %w", err))
		}
		run, ok := pickRun(tab, c, o, skip)
		if !ok {
			if o.Fixed != "" {
				if r, found := tab.Find(o.Fixed); !found {
					return done("none", fmt.Errorf("%w: no run named %q", ErrRunLocked, o.Fixed))
				} else if why := lockedWhy(r, c); why != "" {
					return done("locked", fmt.Errorf("%w: %s: %s", ErrRunLocked, r.Name, why))
				}
			}
			return done("none", nil)
		}
		if !first {
			if err := ops.NewGame(ctx); err != nil {
				return done("error", fmt.Errorf("farm: new game: %w", err))
			}
		}
		first = false
		if err := ops.Chores(ctx); err != nil {
			if ctx.Err() != nil {
				return done("ctx", ctx.Err())
			}
			logf("farm: chores: %v", err) // a failed chore is not a failed run; the run's own potion floor decides
		}
		rep.Runs++
		st := rep.ByRun[run.Name]
		if st == nil {
			st = &RunStat{}
			rep.ByRun[run.Name] = st
		}
		st.Runs++
		t1 := w.Now()
		killed, err := playRun(ctx, k, run, &rep, st)
		st.Time += w.Now().Sub(t1)
		switch {
		case errors.Is(err, ErrTooWeak):
			skip[run.Name] = true // the floor in the script is stricter than the table's: try another run
			rep.Runs--
			st.Runs--
			logf("farm: %s too weak, skipping it", run.Name)
			continue
		case ctx.Err() != nil:
			return done("ctx", ctx.Err())
		case errors.Is(err, ErrDead):
			rep.Deaths++
		}
		if err != nil {
			logf("farm: %s: %v", run.Name, err)
		}
		if o.OnRun != nil {
			o.OnRun(run.Name, killed)
		}
		if killed {
			rep.Kills++
			st.Kills++
			consec, failsOf[run.Name] = 0, 0
			continue
		}
		st.Fails++
		consec++
		if failsOf[run.Name]++; failsOf[run.Name] >= 2 && o.Fixed == "" {
			skip[run.Name] = true
		}
	}
}

// ErrRunLocked: a named run is not allowed for this character (unlock flag missing or clvl under the table's
// floor). Retrying cannot help, so callers should hold the goal instead of retrying it.
var ErrRunLocked = errors.New("boss: run not unlocked")

// lockedWhy says why run r is closed to c, or "" when it is open. Same rules as FarmTable.Rank.
func lockedWhy(r FarmRun, c FarmChar) string {
	if f := r.Floor(c.Difficulty); c.Level < f {
		return fmt.Sprintf("needs clvl %d, have %d", f, c.Level)
	}
	for _, n := range r.Needs {
		if !c.Has[n] {
			return "needs " + n
		}
	}
	return ""
}

func pickRun(tab FarmTable, c FarmChar, o FarmOpts, skip map[string]bool) (FarmRun, bool) {
	if o.Fixed != "" {
		r, ok := tab.Find(o.Fixed)
		return r, ok && !skip[r.Name] && lockedWhy(r, c) == ""
	}
	return tab.Pick(c, o.Goal, skip)
}

// playRun runs the steps in order. Every step must kill its boss for the run to count; the first error ends the run
// (later steps are skipped). Loot is tallied per step, also for a step that failed.
func playRun(ctx context.Context, k BossKiller, run FarmRun, rep *FarmReport, st *RunStat) (bool, error) {
	for _, step := range run.Steps {
		err := k.KillBoss(ctx, step)
		n := k.LastResult().Loot
		rep.Loot += n
		st.Loot += n
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

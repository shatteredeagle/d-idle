package bot

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"d2lod-bot/internal/boss"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/task"
)

// The runs goal (docs/farm.md): boss.FarmLoop plays the runs, this file is its FarmOps (character read, new game
// through the Cycler, town chores) and turns every completed run into one boss-kill event, which is the goal's progress.

// FarmCall is one runs goal handed to TaskDeps.Farm.
type FarmCall struct {
	Run      string // run name, or "auto" for the scheduler
	Runs     int
	Duration time.Duration
	Ops      boss.FarmOps
	// FirstHere is true when the first run may start in the current game (a fresh start or an interrupted run); false
	// when the last run ended in this game, so a new game is due.
	FirstHere bool
	// OnRun must be passed on as boss.FarmOpts.OnRun.
	OnRun func(run string, killed bool)
}

// farmProgress is what a runs goal has done since Runner.Start, kept across farmStep restarts (a death or a level-up
// interrupt ends the step and the manager starts it again) because r.goal.Progress is only the Start-time copy.
type farmProgress struct {
	gen      uint64
	kills    int
	spent    time.Duration
	ended    bool // the last run ended in the current game
	started  time.Time
	baseKill int
	baseSecs int64
}

// farmRemaining returns the runs and duration still to do. Count and UntilMins of 0 mean unlimited and stay 0; an
// exhausted limit becomes the smallest positive value so the loop ends at once instead of running unlimited.
func farmRemaining(p goals.Params, killed int, secs int64) (int, time.Duration) {
	runs, dur := 0, time.Duration(0)
	if p.Count > 0 {
		runs = max(p.Count-killed, 1)
	}
	if p.UntilMins > 0 {
		dur = max(time.Duration(p.UntilMins)*time.Minute-time.Duration(secs)*time.Second, time.Second)
	}
	return runs, dur
}

// farmOps implements boss.FarmOps over the runner.
type farmOps struct {
	r *Runner
	c control
}

func (o farmOps) Char(ctx context.Context) (boss.FarmChar, error) {
	s, err := o.r.e.Snapshot()
	if err != nil || s == nil || s.State != d2.StateInGame || s.Player == nil {
		if s = o.r.latest(); s == nil {
			return boss.FarmChar{}, errors.New("no character snapshot")
		}
	}
	p := s.Player
	return boss.FarmChar{Level: p.Level, Class: p.Class.String(), Difficulty: int(p.Difficulty), Has: o.unlocks(int(p.Difficulty))}, nil
}

// unlocks maps the quest reader to the farm table's unlock flags. Anya's red portal needs Prison of Ice done in the
// character's difficulty (S: docs/progression; U live). Without a reader nothing is unlocked, the safe direction.
func (o farmOps) unlocks(diff int) map[string]bool {
	has := map[string]bool{}
	if td := o.r.d.Task; td != nil && td.Quests != nil {
		if v, ok := td.Quests.Read(diff); ok {
			for q, flag := range questUnlocks {
				has[flag] = has[flag] || v.Quests[q].Stage >= task.StageDone
			}
		}
	}
	return has
}

// questUnlocks maps a quest id to the farm-table unlock flag its completion grants.
var questUnlocks = map[string]string{task.QPrison: "anya_portal"}

// lockedNeed returns the unlock flag a "run locked" error waits for ("needs anya_portal"), or "" (a clvl lock).
func lockedNeed(err error) string {
	m := needsFlag.FindStringSubmatch(err.Error())
	if m == nil {
		return ""
	}
	return m[1]
}

var needsFlag = regexp.MustCompile(`needs ([a-z_]+)$`)

func (o farmOps) NewGame(ctx context.Context) error {
	if o.r.d.Cycler == nil {
		return errors.New("no Cycler: cannot start a new game")
	}
	diff := o.c.spec.difficulty
	if diff == "" {
		if s := o.r.latest(); s != nil {
			diff = difficultyName(s.Player.Difficulty)
		}
	}
	o.r.farmMu.Lock()
	o.r.farm.ended = false
	o.r.farmMu.Unlock()
	err := o.r.oogRun(func(ctx context.Context) error { return o.r.d.Cycler.NewGame(ctx, diff) })(ctx)
	if err == nil {
		o.r.syncTaskGame()
	}
	return err
}

func (o farmOps) Chores(ctx context.Context) error {
	if o.r.d.Town == nil {
		return nil
	}
	if s := o.r.latest(); s != nil && !s.Player.InTown() {
		if err := o.r.d.Town.ToTown(ctx); err != nil {
			return err
		}
	}
	_, err := o.r.d.Town.Chores(ctx, o.r.fullChores("farm run"))
	return err
}

// farmStep runs the whole loop as one action; its end (limits, no run allowed, error) ends the action.
func (r *Runner) farmStep(ctx context.Context, snap *d2.Snapshot, c control) bool {
	p := c.goal.Params
	r.farmMu.Lock()
	if r.farm.gen != c.gen {
		r.farm = farmProgress{gen: c.gen, baseKill: c.goal.Progress.BossKills, baseSecs: c.goal.Progress.ActiveSeconds}
	}
	r.farm.started = time.Now()
	runs, dur := farmRemaining(p, r.farm.baseKill+r.farm.kills, r.farm.baseSecs+int64(r.farm.spent.Seconds()))
	call := FarmCall{Run: p.Area, Runs: runs, Duration: dur, FirstHere: !r.farm.ended, Ops: farmOps{r: r, c: c}}
	r.farmMu.Unlock()
	defer func() {
		r.farmMu.Lock()
		r.farm.spent += time.Since(r.farm.started)
		r.farmMu.Unlock()
	}()
	call.OnRun = func(run string, killed bool) {
		r.farmMu.Lock()
		r.farm.ended = true
		if killed {
			r.farm.kills++
		}
		r.farmMu.Unlock()
		if killed {
			r.emit(goals.Event{Type: goals.EvKill, Monster: run, Boss: true})
		}
	}
	return r.do(ctx, c, StateClearing, "Farm runs: "+p.Area, 0, "farm runs", func(ctx context.Context) error {
		r.bossRun.Store(true)
		defer r.bossRun.Store(false)
		rep, err := r.d.Task.Farm(ctx, call)
		r.e.Log.Info("farm loop ended", "stop", rep.Stop, "runs", rep.Runs, "kills", rep.Kills, "deaths", rep.Deaths, "loot", rep.Loot, "elapsed", rep.Elapsed.Round(time.Second).String())
		if errors.Is(err, boss.ErrRunLocked) {
			// not retryable: hold the goal (the manager resumes it on the next level up) instead of the retry path
			r.emit(goals.Event{Type: goals.EvError, Hold: true, Level: snap.Player.Level, Unlock: lockedNeed(err), Diff: int(snap.Player.Difficulty), Message: "run locked: " + err.Error()})
			return nil
		}
		if err != nil {
			return err
		}
		switch rep.Stop {
		case "fails", "none", "deaths", "error":
			return fmt.Errorf("farm runs stopped: %s", rep.Stop)
		}
		return nil
	})
}

// compileRuns validates a runs goal without the game and returns its label.
func (r *Runner) compileRuns(p goals.Params) (string, error) {
	if r.d.Task == nil || r.d.Task.Farm == nil {
		return "", errors.New("farm loop not wired (Deps.Task.Farm is nil)")
	}
	if strings.TrimSpace(p.Area) == "" {
		return "", errors.New("runs goal needs a run name or auto")
	}
	return fmt.Sprintf("farm runs: %s x%d", p.Area, p.Count), nil
}

package bot

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/event"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/task"
)

// Deps are the collaborators of a Runner. Mover, Fighter and Town are required; the rest are optional and each
// missing one has a documented fallback (docs/bot.md).
type Deps struct {
	Mover   Mover
	Fighter Fighter
	Town    Townsman
	Loot    Looter // nil: no pickup, no verdicts
	Catalog Catalog
	Plan    *Plan
	Health  Health    // nil: game loss is inferred from unreadable memory
	Cycler  Cycler    // nil: no fresh games, no re-entry after a chicken exit
	Corpse  Corpse    // nil: click-the-spot default
	Leveler Leveler   // nil: points are not spent
	Gear    Gear      // nil: no equip, no gear shopping
	Quester Quester   // nil: quest rewards are not claimed
	Task    *TaskDeps // nil: campaign goals are refused at Start
	// Prepare runs once, in a game with nothing hostile near, before the first goal action (hotkey binding). A failure is
	// retried up to three times, then the bot carries on without.
	Prepare func(ctx context.Context) error
	// Snapshot reads the game state; default env.Env.Snapshot.
	Snapshot func() (*d2.Snapshot, error)
	// Restart, if set, restarts the game process (ladder step 4). Managed mode can implement it; nil skips the step.
	Restart func(ctx context.Context) error
	// SavedExp, if set, reads the experience stored in the character's .d2s (ok false: unreadable). Without it the
	// exp at game entry stands in for it (xp at risk logging).
	SavedExp func() (uint32, bool)
	// Recycle, if set, is asked between goals (a new goal about to start, or idle), in town with nothing hostile near: it
	// restarts the game when the recycler's policy says so (docs/soak.md). did reports a restart that happened, reason says
	// why; an error means it half-ran (the game may be down) and is reported as a recycle_failed event.
	Recycle func(ctx context.Context) (did bool, reason string, err error)
	// RecycleDue, if set, is a cheap pre-check: the runner shows "Restarting the game" only when it says a recycle is due.
	RecycleDue func() (bool, string)
	// OnSnapshot, if set, is called after every successful in-game poll (display and telemetry feeds). It must be quick and
	// must not call back into the Runner.
	OnSnapshot func(*d2.Snapshot)
	// Bus, if set, receives every event the bot emits (internal/event/MIGRATION.md steps 1 and 2). The Runner then
	// feeds Events() from its own bus subscription through event.ToGoals, so the goal manager keeps working unchanged.
	Bus *event.Bus
}

// goalSpec is a goal compiled at Start (names resolved, nothing left to fail later except travel).
type goalSpec struct {
	kind       goals.Type
	area       int // fixed target area (farm_area, kill_boss); 0 for reach_level and idle
	level      int
	difficulty string // "", normal, nightmare, hell
	label      string
	camp       *task.Campaign // campaign goals
}

type control struct {
	goal     *goals.Goal
	spec     goalSpec
	gen      uint64
	paused   bool
	stopping bool
	settings goals.Settings
}

// Runner is the real goals.Runner: a state machine driven by a tick loop on d2.Snapshot. Create it with New, start
// its goroutine with Run, and hand it to goals.New. See docs/bot.md.
type Runner struct {
	e   *env.Env
	d   Deps
	cfg Config
	ch  chan goals.Event
	bus *event.Bus
	sub *event.Sub // bridge subscription (bus -> ch), nil without a bus

	lastStep atomic.Int64 // wall-clock unix nanos of the last finished loop iteration (heartbeat for the d2bot watchdog)

	mu          sync.Mutex // control and status
	goal        *goals.Goal
	spec        goalSpec
	gen         uint64
	paused      bool
	stopping    bool
	stopped     bool // a Stop happened (Status reports stopped rather than idle)
	settings    goals.Settings
	setDirty    bool
	cancelAct   context.CancelCauseFunc
	st          State
	activity    string
	message     string
	char        goals.Character
	act0        int // 0-based act of the player at the last snapshot
	updated     time.Time
	stateSince  time.Time
	activeSince time.Time
	fatal       error
	errorUntil  time.Time

	killMu      sync.Mutex      // killedNames
	killedNames map[string]bool // normalized names of monsters killed in this game (campaign proxies)

	obMu          sync.Mutex
	ob            observer
	dlog          *deathLog         // guarded by obMu
	avoid         map[int]time.Time // areas out of play (death spiral, nav faults), guarded by obMu
	decMu         sync.Mutex        // decisions
	decisions     []decisionNote    // last planner/runner decisions for the planner dump (deathrule.go)
	avoidWhy      map[int]string    // producer of each avoid entry (areaskip.go), guarded by obMu
	crossLog      []time.Time       // recent "walked over an area border" re-plans (noteCrossing), guarded by obMu
	hmu           sync.Mutex        // lastError and ladderStep
	lastError     string
	ladder        int
	selfStop      bool                 // paused by the death-rate rule, guarded by mu
	merc          mercState            // sticky merc knowledge and chore backoff (resource.go)
	drank         atomic.Int64         // potions drunk, counted from every EvPotion event (combat emits them live; the observer's own diff is off there)
	noTP          atomic.Bool          // the running task step has Constraints.NoTownPortal: no safety portal, no trip
	farmMu        sync.Mutex           // guards farm
	farm          farmProgress         // runs goal progress across farmStep restarts (farm.go)
	bossRun       atomic.Bool          // a boss script is running (Deps.Task.Boss): routine town trips wait, the guard's re-plan cancels are off
	bossStart     map[string]time.Time // start of the last unfinished boss run by name (taskActs.KillBoss, loop goroutine only)
	levelerPaused bool                 // selfStop came from the leveler (LevelerPause), guarded by mu
	autoResumeAt  time.Time            // a failsafe self pause ends by itself at this time (zero: waits for Resume), guarded by mu
	lastSelfPause time.Time            // for the self pause backoff, guarded by mu
	pauseStreak   int
	panicTimes    []time.Time // recovered panics inside PanicWindow, guarded by mu
	panics        int         // all recovered panics, guarded by mu
	lastOverrun   time.Time   // the last time the panic budget was exceeded, guarded by mu
	crash         error       // set when the panic budget cannot be handled by pausing: Run returns it, guarded by mu

	// Main-goroutine-only state.
	loop loopState
}

// loopState belongs to the Run goroutine alone.
type loopState struct {
	gen              uint64 // generation the fields below belong to
	failures         int    // consecutive failed actions
	stuckStage       int
	inLadder         bool // escalate is running: a stall inside a rung is that rung failing, not a new ladder
	ladderProgress   uint64
	restarts         int // failsafe game restarts since the last real progress
	goneRestarts     int // restarts after an unplanned game exit, same reset rule
	goneProgress     uint64
	restartTimes     []time.Time // unplanned restarts in the last hour
	trip             *trip
	stuckAt          time.Time
	cleared          map[int]bool // areas fully explored in this game
	sweepKills       map[int]int  // kills per area in this game, for the sweep log line
	clearedGen       int          // ob.newGames value cleared belongs to
	stepArea         int          // planner choice (reach_level), kept until level or availability changes
	stepLevel        int
	needBuff         bool
	corpseTry        int
	waitNote         bool
	waitSince        time.Time // start of the current wait in town for area bans (areaskip.go)
	waitLast         time.Time
	diffTries        int       // failed or pending difficulty switches in a row (backoff exponent)
	diffNext         time.Time // no new switch attempt before this
	pendingErr       string
	rec              *recovery
	errGames         int
	stopTries        int
	lastRecycleCheck time.Time // idle (no goal) recycle polling
	saveNext         time.Time // earliest next save point after a failed or planned one
	lastPoints       int
	pointsTryAt      time.Time
	pointsFails      int
	pointsFailLvl    int
	gearAt           time.Time
	shopLevel        int
	shopGold         int
	starved          bool
	broke            brokeState                                // recovery phase (broke.go)
	br               brState                                   // Blood Raven attempt budget (broke.go)
	brWorldFn        func(context.Context) (task.World, error) // tests: replaces the world the Blood Raven gate reads
	starveNote       bool
	rebuyBar         int    // gold needed before the next gear rebuy trip
	gearNote         bool   // the gear gate was logged for this spell
	routeNote        string // last route guard / imbue verdict logged (one line per change)
	run              *LevelRun
	questPending     map[string]bool
	questDone        map[string]bool
	questTries       map[string]int // sweeps that ended with the quest objective still open
	hold             areaHold       // hysteresis of the planner's area choice
	claimSkipUntil   time.Time      // no claim trip before this (the last one failed)
	beltFillNext     time.Time      // no belt fill attempt before this
	diffBan          map[string]time.Time
	forceTrip        bool
	healFails        int         // failed heal attempts while below the resume line; reset when life is back above it
	lowLifeUntil     time.Time   // end of the current low-life wait (lowLifeWait); zero when not waiting
	reentries        []time.Time // recent re-entries after a chicken (notInGame), pruned to reentryWindow
	futile           bool
	futilePos        input.Point
	futileUntil      time.Time
	// empty-area leave (emptyArea): the area watched, its kill count and the time it last changed
	emptyArea  int
	emptyKills int
	emptySince time.Time
	stallStage int       // campaign stall ladder (stall.go): 0 quiet, 1 warned and fresh game tried
	stallAt    time.Time // when stage 1 was reached
	emptyLast  time.Time // last call: a longer pause (town trip, new game) restarts the count
	task       taskState
	wp         wpState
	prepared   bool
	prepTries  int
}

// New builds a Runner. It installs e.Emit (if unset) so behaviors publishing through the Env feed the same event channel.
func New(e *env.Env, d Deps, cfg Config) (*Runner, error) {
	if e == nil {
		return nil, errors.New("bot: nil env")
	}
	if d.Mover == nil || d.Fighter == nil || d.Town == nil {
		return nil, errors.New("bot: Mover, Fighter and Town are required")
	}
	if d.Plan == nil {
		p, err := LoadPlan(cfg.PlanDir)
		if err != nil {
			return nil, err
		}
		d.Plan = p
	}
	if d.Snapshot == nil {
		if e.Read == nil {
			return nil, errors.New("bot: no snapshot source (Deps.Snapshot or Env.Read)")
		}
		d.Snapshot = e.Snapshot
	}
	if e.Clock == nil {
		e.Clock = env.RealClock{}
	}
	if e.Tune == nil {
		t := env.DefaultTuning()
		e.Tune = &t
	}
	if e.Log == nil {
		e.Log = discardLogger()
	}
	r := &Runner{e: e, d: d, cfg: cfg.WithDefaults(), ch: make(chan goals.Event, 2048), st: StateIdle, activity: "Idle",
		settings: goals.DefaultSettings()}
	r.loop.cleared = map[int]bool{}
	r.avoid = map[int]time.Time{}
	r.avoidWhy = map[int]string{}
	dl, err := loadDeathLog(r.cfg.DeathLogPath)
	if err != nil {
		e.Log.Warn("death log not loaded, starting empty", "err", err)
	}
	r.dlog = dl
	if d.Bus != nil {
		r.bus = d.Bus
		r.sub = d.Bus.Subscribe("bot-manager-bridge", 2048)
		go r.bridge(r.sub)
	}
	if e.Emit == nil {
		e.Emit = r.emit
	}
	return r, nil
}

// emit publishes an event without blocking: when the channel is full the oldest event is dropped.
func (r *Runner) emit(ev goals.Event) {
	if ev.Type == goals.EvError {
		r.hmu.Lock()
		r.lastError = ev.Message
		r.hmu.Unlock()
	}
	if ev.Type == goals.EvKill {
		r.noteKill(ev.Monster)
	}
	if ev.Type == goals.EvPotion {
		r.drank.Add(1)
	}
	// The bus event types cannot carry Hold or the quest fields, so these go to the manager directly (the manager
	// republishes quest events; a hold is mirrored as info).
	if ev.Type == goals.EvQuest || ev.Hold {
		if ev.Hold && r.bus != nil {
			r.bus.Publish(event.FromGoals(goals.Event{Type: goals.EvInfo, Message: ev.Message}, event.SrcObserver))
		}
		r.push(ev)
		return
	}
	if r.bus != nil {
		r.bus.Publish(event.FromGoals(ev, event.SrcObserver))
		return
	}
	r.push(ev)
}

// bridge copies bus events that goals can express into the channel the manager reads (drop-oldest, never blocks).
func (r *Runner) bridge(sub *event.Sub) {
	for !r.bridgeLoop(sub) {
	}
}

// bridgeLoop forwards bus events until the subscription closes (true); a recovered panic returns false to go on.
func (r *Runner) bridgeLoop(sub *event.Sub) (closed bool) {
	defer func() {
		if v := recover(); v != nil {
			r.notePanic(v, debug.Stack())
		}
	}()
	for e := range sub.C() {
		if g, ok := event.ToGoals(e); ok && !isManagerKind(g.Type) {
			r.push(g)
		}
	}
	return true
}

// isManagerKind reports types the goal manager itself produces; they may be on the bus (for SSE) but must not loop back.
func isManagerKind(t goals.EventType) bool {
	switch t {
	case goals.EvKill, goals.EvDrop, goals.EvDeath, goals.EvLevelUp, goals.EvAreaChange, goals.EvPotion, goals.EvError:
		return false
	}
	return true
}

// Close ends the bus bridge. The runner needs no other cleanup.
func (r *Runner) Close() {
	if r.sub != nil {
		r.sub.Close()
	}
}

func (r *Runner) push(ev goals.Event) {
	for {
		select {
		case r.ch <- ev:
			return
		default:
		}
		select {
		case <-r.ch:
		default:
		}
	}
}

// Events implements goals.Runner.
func (r *Runner) Events() <-chan goals.Event { return r.ch }

// Configure implements goals.Configurer; the settings are applied between actions.
func (r *Runner) Configure(s goals.Settings) {
	r.mu.Lock()
	r.settings, r.setDirty = s, true
	r.mu.Unlock()
	r.wakeUp()
}

func (r *Runner) wakeUp() {} // the loop polls at TickMs; kept as the hook for a future wake channel

// Start implements goals.Runner: compile the goal and switch the loop over to it.
func (r *Runner) Start(g goals.Goal) error {
	spec, err := r.compile(g)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fatal != nil {
		return fmt.Errorf("bot halted after a fatal error: %w", r.fatal)
	}
	gc := g
	r.goal, r.spec = &gc, spec
	r.gen++
	r.paused, r.stopping, r.stopped = false, false, false
	r.errorUntil = time.Time{}
	r.message = ""
	r.setLocked(StateIdle, "Starting: "+spec.label)
	r.cancelLocked(errNewGoal)
	return nil
}

// Pause implements goals.Runner. The loop freezes at the next safe moment: immediately in town, in the field once
// no hostile unit is within the clear radius.
func (r *Runner) Pause() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.goal != nil {
		r.paused = true
		if r.st != StatePaused {
			r.activity = "Pausing"
		}
	}
	return nil
}

// Resume implements goals.Runner.
func (r *Runner) Resume() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.paused = false
	if r.selfStop {
		r.selfStop, r.message = false, ""
	}
	r.autoResumeAt = time.Time{}
	return nil
}

// Settled reports whether a Stop has finished: the bot is in town (or gave up trying) and idle. A runner that was never
// stopped is settled too.
func (r *Runner) Settled() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return !r.stopping
}

// Stop implements goals.Runner: abandon the goal and finish in town when possible.
func (r *Runner) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.goal == nil && !r.stopping {
		r.stopped = true
		return nil
	}
	r.goal = nil
	r.gen++
	r.paused, r.stopping, r.stopped = false, true, true
	r.errorUntil = time.Time{}
	r.message = ""
	r.setLocked(StateIdle, "Stopping")
	r.cancelLocked(errStop)
	return nil
}

// Status implements goals.Runner. It returns the cached view and never touches the game.
func (r *Runner) Status() goals.Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := goals.Status{Activity: r.activity, Message: r.message, Character: r.char, Updated: r.updated}
	if r.goal != nil {
		s.GoalID = r.goal.ID
	}
	switch {
	case r.selfStop && r.fatal == nil:
		s.State = goals.RunnerPaused
	case r.fatal != nil || r.st == StateError:
		s.State = goals.RunnerError
		if r.fatal != nil {
			s.Message = r.fatal.Error()
		}
	case r.goal == nil && (r.stopped || r.stopping):
		s.State = goals.RunnerStopped
	case r.goal == nil:
		s.State = goals.RunnerIdle
	case r.st == StatePaused:
		s.State = goals.RunnerPaused
	default:
		s.State = goals.RunnerRunning
	}
	return s
}

// State returns the internal state (for tests, logs and the debug endpoint).
func (r *Runner) State() State {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.st
}

func (r *Runner) setLocked(st State, activity string) {
	now := r.e.Clock.Now()
	if st != r.st || r.stateSince.IsZero() {
		r.stateSince = now
		idle := r.goal != nil && r.goal.Type == goals.Idle
		if skippedState(r.st, false) && !skippedState(st, idle) {
			r.activeSince = now
		}
	}
	if activity != r.activity {
		r.noteDecision(string(st) + ": " + activity)
	}
	r.st, r.activity = st, activity
	r.updated = now
}

func (r *Runner) setState(st State, activity string) {
	r.mu.Lock()
	if r.st != st {
		r.e.Log.Info("state", "from", r.st, "to", st, "activity", activity)
	}
	r.setLocked(st, activity)
	r.mu.Unlock()
}

func (r *Runner) cancelLocked(cause error) {
	if r.cancelAct != nil {
		r.cancelAct(cause)
	}
}

func (r *Runner) control() control {
	r.mu.Lock()
	defer r.mu.Unlock()
	return control{goal: r.goal, spec: r.spec, gen: r.gen, paused: r.paused, stopping: r.stopping, settings: r.settings}
}

// compile resolves a goal's names without touching the game. It is the only place Start can fail.
func (r *Runner) compile(g goals.Goal) (goalSpec, error) {
	p := g.Params
	spec := goalSpec{kind: g.Type, difficulty: p.Difficulty}
	switch g.Type {
	case goals.Idle:
		spec.label = "rest in town"
	case goals.ReachLevel:
		if p.Level < 1 || p.Level > 99 {
			return spec, fmt.Errorf("level %d out of range", p.Level)
		}
		if len(r.d.Plan.Steps["normal"]) == 0 {
			return spec, errors.New("no progression data loaded")
		}
		spec.level, spec.label = p.Level, fmt.Sprintf("reach level %d", p.Level)
	case goals.FarmArea, goals.KillBoss:
		id, err := r.resolveArea(g)
		if err != nil {
			return spec, err
		}
		if act := r.d.Plan.Act(id); act > 0 && !r.actAllowed(act) {
			return spec, fmt.Errorf("%s is in act %d; the bot cannot change acts yet", r.areaName(id), act)
		}
		spec.area, spec.label = id, "farm "+r.areaName(id)
	case goals.Campaign:
		if r.d.Task == nil {
			return spec, errors.New("task layer not wired (Deps.Task is nil)")
		}
		spec.camp = task.Template(p.Template, p.Cain).WithRogueClaim()
		if spec.camp == nil {
			return spec, fmt.Errorf("unknown campaign template %q", p.Template)
		}
		first, last, label := campaignActs(spec.camp.Name)
		spec.label = label
		for act := first; act <= last; act++ {
			if !r.actAllowed(act) {
				return spec, fmt.Errorf("the %s campaign needs act %d to be allowed", label, act)
			}
		}
	case goals.Runs:
		label, err := r.compileRuns(p)
		if err != nil {
			return spec, err
		}
		spec.label = label
	default:
		return spec, fmt.Errorf("goal type %q is not supported yet", g.Type)
	}
	return spec, nil
}

// campaignActs gives the first and last act a template covers and its dashboard label.
func campaignActs(name string) (first, last int, label string) {
	switch name {
	case goals.CampaignNormalAct2:
		return 2, 2, "Normal Act 2"
	case goals.CampaignNormalAct3:
		return 3, 3, "Normal Act 3"
	case goals.CampaignNormalAct4:
		return 4, 4, "Normal Act 4"
	case goals.CampaignNormalAct5:
		return 5, 5, "Normal Act 5"
	case goals.CampaignNormalAll:
		return 1, 5, "Normal Acts 1-5"
	case goals.CampaignNightmareAll:
		return 1, 5, "Nightmare Acts 1-5"
	case goals.CampaignHellAll:
		return 1, 5, "Hell Acts 1-5"
	}
	return 1, 1, "Normal Act 1"
}

func (r *Runner) resolveArea(g goals.Goal) (int, error) {
	p := g.Params
	if g.Type == goals.KillBoss {
		want := normName(p.Boss)
		for _, run := range r.d.Plan.Runs {
			for _, k := range run.Kill {
				if normName(k) == want {
					return run.FinalArea, nil
				}
			}
		}
		return 0, fmt.Errorf("no run known for boss %q", p.Boss)
	}
	return r.d.Plan.Resolve(p.Area, p.AreaID, r.d.Catalog, func(name string) int {
		for _, a := range goals.Catalog().Areas {
			if a.ID > 0 && strings.EqualFold(a.Name, name) {
				return a.ID
			}
		}
		return 0
	})
}

// actAllowed: configured acts plus the act the character currently stands in.
func (r *Runner) actAllowed(act int) bool {
	for _, a := range r.cfg.Acts {
		if a == act {
			return true
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.act0+1 == act
}

// Beat stamps the heartbeat from inside a long step. Call it only after an observable sub-step finished (a restart
// phase), never from a timer, so the d2bot watchdog still catches a step that hangs between two sub-steps.
func (r *Runner) Beat() { r.lastStep.Store(time.Now().UnixNano()) }

// LastStep is the wall-clock time of the last finished Run loop iteration (zero before Run started). The loop keeps
// iterating while paused, self-paused or in an error wait, so a stale value means a step is stuck or the loop died.
func (r *Runner) LastStep() time.Time {
	n := r.lastStep.Load()
	if n == 0 {
		return time.Time{}
	}
	return time.Unix(0, n)
}

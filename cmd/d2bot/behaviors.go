package main

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/cheat"
	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/leveler/townvendor"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/nav"
	"d2lod-bot/internal/town"
)

var errNeedTables = errors.New("live play needs the game tables (data.Open failed): set --gamedir / D2BOT_GAMEDIR")

// runnerRef lets the behaviors (built before the Runner) call into it: the leveler's Pause hook.
type runnerRef struct{ p atomic.Pointer[bot.Runner] }

// beat stamps the runner's watchdog heartbeat (a finished sub-step of a long restart); no-op before the runner exists.
func (r *runnerRef) beat() {
	if r != nil {
		if run := r.p.Load(); run != nil {
			run.Beat()
		}
	}
}

func (r *runnerRef) pause(reason string) {
	if r != nil {
		if run := r.p.Load(); run != nil {
			run.LevelerPause(reason)
		}
	}
}

// behaviors is everything the runner is built from for one instance.
type behaviors struct {
	Mover   bot.Mover
	Fighter bot.Fighter
	Town    bot.Townsman
	Loot    bot.Looter
	Cycler  bot.Cycler
	Leveler bot.Leveler
	Gear    bot.Gear
	Quester bot.Quester
	Task    *bot.TaskDeps
	combat  *combat.Fighter // for hotkey binding
	// start runs before the hotkeys are bound: class-aware setup from the logged-in character (class.go).
	start func(ctx context.Context)
}

// prepare binds the skill hotkeys once the runner stands in a game (the game may still be at the menus when d2bot starts,
// for example in managed mode right after the launch). Nil without a fighter.
func (b *behaviors) prepare(ctx context.Context) error {
	if b.combat == nil {
		return nil
	}
	if b.start != nil {
		b.start(ctx)
	}
	return b.combat.EnsureHotkeys(ctx)
}

// buildBehaviors constructs nav, combat, town and loot for one instance and installs the hooks.
// Hooks: combat.Leave = launch.Flow.LeaveGame (Save and Exit through the menus). combat.Portal is left at combat's
// default (right-click the scroll in the pack, UNVERIFIED geometry): the Townsman opens its own portals in ToTown.
func buildBehaviors(ctx context.Context, e *env.Env, o options, gameDir string, db *data.DB, charName func() string, log *slog.Logger, cfg bot.Config, ref *runnerRef) (*behaviors, error) {
	if o.stubs {
		return &behaviors{Mover: nav.New(e), Fighter: stubFighter{}, Town: stubTown{}}, nil
	}
	if db == nil {
		return nil, errNeedTables
	}
	nv := nav.New(e)
	cb := combat.New(e, nv)
	nv.Yield = packYield(e, cb)
	if rows, err := combat.LoadListRows(gameDir, db); err != nil {
		log.Warn("skill list rows unavailable: hotkeys cannot be bound", "err", err)
	} else {
		cb.SetListRows(rows)
	}
	lt := loot.New(e, nv)
	tw := town.New(e, nv)
	nv.Escape = tw.ToTown           // a gate the walk keeps stalling on (docs/nav.md, stuck-spot memory): leave by portal and re-plan from town
	nv.ClearCursor = tw.ClearCursor // a waypoint panel does not open with an item on the cursor (docs/waypoints.md)
	tw.SetJudge(lt)

	cli := input.NewClient(o.helper)
	flow := &launch.Flow{Mem: e.Mem, In: typer{input.NewOOG(cli), cli}, Log: func(s string) { log.Info("oog", "msg", s) }}
	if cheat.CheckCharacter(charName()) == nil { // a real character never had god mode: no ledger noise on every save
		flow.BeforeSave = guardSave(cheat.Gate{Allow: true, Env: cheat.DefaultEnv(), Actor: "d2bot"}, log)
	}
	cb.Leave = func(ctx context.Context) error { return flow.LeaveGame(ctx, 20*time.Second) }
	bh := &behaviors{
		Mover: moverAdapter{nv}, Fighter: fighterAdapter{cb}, Town: townAdapter{t: tw, e: e, mv: nv}, Loot: looterAdapter{lt},
		Cycler: cycler{flow: flow, name: charName, world: e.World}, combat: cb,
	}
	bh.start = func(ctx context.Context) { applyClass(ctx, e.Snapshot, cb, tw, lt, cfg.LevelerBuild, e.Data, log) }
	bh.Task = newTaskDeps(ctx, o, tw, nv, e, charName)
	ta := townAdapter{t: tw, e: e, mv: nv}
	bg := newBossGame(e, db, cb, nv, lt, ta.ToTown)
	br := newBossRunner(bg)
	bh.Task.Boss = bossKill(br, e.Log.Info)
	bh.Task.Farm = farmLoop(bg, br, e.Log.Info)
	if !cfg.NoLeveler {
		// New attack skills need a hotkey: the fighter binds what it knows after every spend that learned a skill.
		lv := leveler.New(e, leveler.Options{Build: cfg.LevelerBuild, PlanDir: cfg.PlanDir, Pause: ref.pause, Rebind: func(ctx context.Context) error {
			bctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()
			return cb.EnsureHotkeys(bctx)
		}})
		bh.Leveler = lv
		bh.Gear = gearAdapter{l: lv, v: townvendor.Vendor{T: tw}, e: e}
		bh.Quester = questAdapter{t: tw, e: e}
	}
	return bh, nil
}

// saveGuard is the part of cheat.Gate that guardSave needs.
type saveGuard interface {
	GuardSave(ctx context.Context, reason string) (string, error)
}

// guardSave is launch.Flow.BeforeSave: restore an inflated max life (god mode marker) before the game writes the
// character. A character that is not on the cheat allow-list or a missing marker is not an error worth stopping a
// save for; the message is only logged.
func guardSave(g saveGuard, log *slog.Logger) func(context.Context, string) error {
	return func(ctx context.Context, reason string) error {
		msg, err := g.GuardSave(ctx, reason)
		if err != nil {
			log.Warn("save guard", "err", err)
			return err
		}
		if msg != "" {
			log.Info("save guard", "msg", msg)
		}
		return nil
	}
}

// Stubs: every action fails, Safety never asks for anything. Used with --stub-behaviors to exercise the rest of the
// wiring (API, goals, runner, watchdog) against a live game without any input being sent.
type stubFighter struct{}

var errStub = errors.New("stub behavior: nothing is done")

func (stubFighter) Clear(context.Context, bot.ClearOpts) (bot.ClearResult, error) {
	return bot.ClearResult{}, errStub
}
func (stubFighter) Kill(context.Context, uint32) error          { return errStub }
func (stubFighter) Buff(context.Context) error                  { return errStub }
func (stubFighter) Safety(*d2.Snapshot) bot.SafetyAction        { return bot.SafetyNone }
func (stubFighter) Act(context.Context, bot.SafetyAction) error { return errStub }

type stubTown struct{}

func (stubTown) ToTown(context.Context) error { return errStub }
func (stubTown) Chores(context.Context, bot.ChorePlan) (bot.ChoreReport, error) {
	return bot.ChoreReport{}, errStub
}
func (stubTown) BackToField(context.Context, int) error { return errStub }
func (stubTown) Rebuy(context.Context, int) (bot.RebuyReport, error) {
	return bot.RebuyReport{}, errStub
}
func (stubTown) NeedsTrip(*d2.Snapshot) (bool, string) { return false, "" }

var (
	_ bot.Mover    = moverAdapter{}
	_ bot.Fighter  = fighterAdapter{}
	_ bot.Townsman = townAdapter{}
	_ bot.Looter   = looterAdapter{}
	_ bot.Cycler   = cycler{}
)

// packYield is nav's pack-stall hook: a bounded clear of the pack that blocks the walk (never in town, never
// against friendlies).
func packYield(e *env.Env, cb *combat.Fighter) func(context.Context, []d2.Unit) error {
	cat := combat.NewCatalog(e.Data)
	var pin pinState // belt and braces next to combat's own pinned-evade guard (unpinned, boss.go)
	return bot.PackYield(bot.YieldDeps{
		Timeout:  4 * time.Second, // short calls so unpinned can see a stuck character
		Snapshot: e.Snapshot,
		Friendly: func(s *d2.Snapshot, u d2.Unit) bool { return combat.IsFriendly(s, u, cat) },
		Clear: func(ctx context.Context, radius int, timeout time.Duration) error {
			return unpinned(e, &pin, func() error {
				_, err := cb.Clear(ctx, combat.ClearOpts{Radius: radius, Timeout: timeout, Yield: true})
				return mapCombatErr(err)
			})
		},
	})
}

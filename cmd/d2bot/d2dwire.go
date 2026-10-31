package main

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dcombat"
	"d2lod-bot/internal/d2denv"
	"d2lod-bot/internal/d2dmerc"
	"d2lod-bot/internal/d2dsrc"
	"d2lod-bot/internal/d2dtown"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/leveler/townvendor"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/merc"
	"d2lod-bot/internal/nav"
	"d2lod-bot/internal/town"
)

// d2dStack is the real combat, town, leveler and task layers of the Game.exe path (behaviors.go buildBehaviors) built
// over a d2d backend: the same packages, with d2d's reader/input seams instead of memory and pixels
// (docs/d2d-wire.md). Without game tables nothing can be built and the caller keeps the "not yet" stubs.
type d2dStack struct {
	rig     *d2dcombat.Rig
	cb      fighterAdapter
	tw      *town.Town
	lt      *loot.Looter
	town    townAdapter
	leveler *leveler.Leveler
	gear    bot.Gear
	quester bot.Quester
	task    *bot.TaskDeps
	prepare func(ctx context.Context) error
}

// buildD2DStack wires the stack. nv is the backend navigator (be.Navigator()); ref is the runner back-reference the
// leveler's Pause hook needs.
func buildD2DStack(ctx context.Context, o options, be *d2denv.Backend, nv *nav.Navigator, db *data.DB, cfg bot.Config, ref *runnerRef, log *slog.Logger) *d2dStack {
	e := be.Env
	rig := d2dcombat.Attach(ctx, e, nv, be.Ctl)
	nv.SetMouthGuard(cfg.MouthGuard) // config key mouthGuard, off by default: -27 % border crossings but no kill gain (docs/killfix.md)
	cb := rig.Fighter
	cb.Leave = be.Cycler.LeaveGame
	cb.Portal = d2dPortal(e, be, combat.NewCatalog(db))

	// Town clicks need the hover-then-click step (every dialog and vendor click, docs/d2d-town.md break 4).
	te := *e
	te.In = d2dtown.HoverFirstBelt(be.Ctl, be.Client)
	tw := town.New(&te, nv)
	ad := d2dtown.New(be.Client, be.Source, d2dsrc.NewDBTables(db))
	tw.UseSource(ad.Source())
	nv.Escape = tw.ToTown
	nv.ClearCursor = tw.ClearCursor
	lt := loot.New(rig.Env, nv)
	tw.SetJudge(lt)

	s := &d2dStack{rig: rig, cb: fighterAdapter{cb}, tw: tw, lt: lt, town: townAdapter{t: tw, e: e, mv: nv}}
	name := o.char
	if name == "" {
		name = "D2dbot"
	}
	charName := func() string { return name }
	oo := o
	oo.managed = false // saveDirOf would load the Game.exe managed config; there is no save dir to find for d2d
	s.task = newTaskDeps(ctx, oo, tw, nv, e, charName)
	s.task.ClaimMerc = claimDep(d2dmerc.Claimer{C: be.Client}, &d2dmerc.Reader{Read: be.Reader.Read, LevelEstimate: levelEstimate},
		func() merc.Difficulty { return merc.Normal }, log) // d2d starts Normal only (Spec.Difficulty)
	ta := s.town
	bg := newBossGame(e, db, cb, nv, lt, ta.ToTown)
	br := newBossRunner(bg)
	s.task.Boss = bossKill(br, log.Info)
	s.task.Farm = farmLoop(bg, br, log.Info)
	if !cfg.NoLeveler {
		s.leveler = leveler.New(e, leveler.Options{Build: cfg.LevelerBuild, PlanDir: cfg.PlanDir, Pause: ref.pause, Rebind: func(context.Context) error {
			_, err := rig.BindSkills()
			return err
		}})
		s.gear = gearAdapter{l: s.leveler, v: townvendor.Vendor{T: tw}, e: e}
		s.quester = questAdapter{t: tw, e: e}
	}
	s.prepare = func(ctx context.Context) error {
		applyClass(ctx, e.Snapshot, cb, tw, lt, cfg.LevelerBuild, db, log)
		_, err := rig.BindSkills()
		return err
	}
	return s
}

// seams are the d2denv seams of this stack (looting stays off: the pixel pickup path is not verified on d2d and the
// fork's autoloot is toggled off for these runs, docs/d2d-wire.md).
func (s *d2dStack) seams() d2denv.Seams {
	return d2denv.Seams{
		Fighter: func(*env.Env, *nav.Navigator) bot.Fighter { return s.cb },
		Town: func(*env.Env, *nav.Navigator) bot.Townsman {
			return d2dTown{townAdapter: s.town, portal: s.rig.Fighter.Portal}
		},
		Loot: s.lootSeam,
	}
}

// d2dTown is the Townsman for d2d: ToTown first opens a portal with combat's d2dPortal (the town package's own opening
// waits for the inventory panel flag that d2d never reports: "hand: panel did not open: ui flag 0x1 never became true"
// x5 in a row, then the failsafe ladder restarted the game; V live D2DPACE), after which the adapter finds the open portal
// next to the player and walks in. Without a scroll it falls through to the adapter's walk-back.
type d2dTown struct {
	townAdapter
	portal func(context.Context) error
}

func (t d2dTown) ToTown(ctx context.Context) error {
	if s, err := t.e.Snapshot(); err == nil && s.Player != nil && !s.Player.InTown() && t.portal != nil {
		open := false
		for _, o := range s.Objects {
			open = open || (o.Class == town.ObjPortal && max(abs(o.X-s.Player.X), abs(o.Y-s.Player.Y)) <= 25)
		}
		if !open {
			if err := t.portal(ctx); err != nil && !errors.Is(err, combat.ErrNoPortal) {
				t.e.Log.Info("d2d town: no portal opened, falling back", "err", err)
			}
		}
	}
	return t.townAdapter.ToTown(ctx)
}

// d2dHealth maps the channel's liveness to launch.Monitor events. Alive = the channel answers. Heartbeat = d2denv.Beat,
// the game's own frame or clock counter: a d2d that still answers ping but has stopped running the world is EventHung
// after the monitor's HangAfter and goes down the failsafe ladder like a frozen Game.exe (U live: nothing reproduced a
// wedged d2d, see internal/d2denv/beat.go).
func d2dHealth(p *d2denv.Proc, beat *d2denv.Beat, lockstep func() bool, lastStep func() time.Time, log *slog.Logger) *launch.Monitor {
	return d2dHealthWith(p, beat, lockstep, lastStep, p.Alive, pingGrace, log)
}

// pingGrace is how long pings may fail while the process still exists before the d2d counts as gone. Under lockstep a
// long "step" holds the one control client for seconds and a loaded box makes a 3 s ping miss; the live run read that as
// "exited" and restarted a healthy game three times (D2DWIRE). A d2d that really ended is seen at once through Proc.Gone.
const pingGrace = 45 * time.Second

// lastStep is the wall time of the bot's last completed lockstep step (nil or zero: none). The control socket takes one
// client: a bot that steps back to back (waiting for mana at clvl 1 is minutes of virtual time) starves the monitor's
// ping, and 50 s of failed pings then read as "exited" and restarted a healthy d2d, losing xp (V live D2DPACE). A step
// that finished within the grace is proof of life.
func d2dHealthWith(p *d2denv.Proc, beat *d2denv.Beat, lockstep func() bool, lastStep func() time.Time, ping func(context.Context) bool, grace time.Duration, log *slog.Logger) *launch.Monitor {
	var mu sync.Mutex
	lastOK := time.Now()
	var lastBeat uint64
	flat := 0
	return &launch.Monitor{P: launch.Probes{
		Alive: func() (int, bool) {
			mu.Lock()
			defer mu.Unlock()
			if p.Gone() {
				log.Warn("d2d process exited", "pid", p.PID(), "state", p.ExitState())
				return 0, false
			}
			if ping(context.Background()) {
				lastOK = time.Now()
			} else if lastStep != nil && time.Since(lastStep()) < grace {
				lastOK = time.Now() // the game is answering the bot's steps, only the ping cannot get a turn
			} else if time.Since(lastOK) > grace {
				log.Warn("d2d stopped answering pings", "silentFor", time.Since(lastOK).Round(time.Second))
				return 0, false
			}
			pid := p.PID()
			if pid == 0 {
				pid = 1 // attach mode: the pid is not known, only that it answers
			}
			return pid, true
		},
		Heartbeat: func(int) (uint64, error) {
			if lockstep != nil && lockstep() {
				// Under lockstep the game only moves when the bot steps it and the monitor's time is the bot's virtual
				// time, so "no progress for HangAfter" is meaningless: the live runs restarted a healthy, answering d2d
				// (frames still counting) right after a town portal that way. A dead d2d is still seen by Alive.
				return uint64(time.Now().UnixNano()), nil
			}
			n, err := beat.Next()
			if err != nil || n == lastBeat {
				if flat++; flat%30 == 0 {
					log.Warn("d2d heartbeat flat", "polls", flat, "counters(ms,tick,stepped,clockframe,infoframe)", beat.Last(), "err", err)
				}
			} else {
				flat = 0
			}
			lastBeat = n
			return n, err
		},
	}, HangAfter: 90 * time.Second}
}

// d2dPortal opens a town portal from the pack: combat's default waits for the inventory panel flag, which d2d does not
// report (the live runs said "inventory panel did not open" three times and then chickened out of the game). The panel is
// opened with "i" and given a fixed time, the scroll's pixel comes from "debug itemat" (V verb, docs/d2d-town.md) and
// the right click casts it. U that "i" opens the panel on d2d; success is judged by the portal object or the scroll
// leaving the pack, as in combat.
func d2dPortal(e *env.Env, be *d2denv.Backend, cat combat.Catalog) func(context.Context) error {
	const townPortalObject = 59
	pick := func(s *d2.Snapshot) *d2.Unit {
		if s.Inventory == nil {
			return nil
		}
		for i := range s.Inventory.Items {
			u := &s.Inventory.Items[i]
			if u.Item != nil && u.Item.Where == d2.ContInventory && cat.Item(u.Class).Portal {
				return u
			}
		}
		return nil
	}
	return func(ctx context.Context) error {
		var last error
		for try := 0; try < 3; try++ {
			s, err := e.Snapshot()
			if err != nil || s.Player == nil {
				return errors.New("d2d portal: no snapshot")
			}
			it := pick(s)
			if it == nil {
				return combat.ErrNoPortal
			}
			last = func() error {
				if err := e.In.Key("i"); err != nil {
					return err
				}
				defer e.In.Key("i")
				e.Clock.Sleep(700 * time.Millisecond)
				x, y, err := be.Client.ItemAt(ctx, int(it.ID))
				if err != nil {
					return err
				}
				if err := e.In.MoveMouse(x, y); err != nil {
					return err
				}
				e.Clock.Sleep(150 * time.Millisecond)
				if err := e.In.Click(input.Right, x, y); err != nil {
					return err
				}
				for i := 0; i < 8; i++ {
					e.Clock.Sleep(200 * time.Millisecond)
					ns, err := e.Snapshot()
					if err != nil || ns.Player == nil {
						continue
					}
					for _, o := range ns.Objects {
						if o.Class == townPortalObject && math.Hypot(float64(o.X-ns.Player.X), float64(o.Y-ns.Player.Y)) < 25 {
							return nil
						}
					}
					if ns.Inventory != nil {
						gone := true
						for _, u := range ns.Inventory.Items {
							gone = gone && u.ID != it.ID
						}
						if gone {
							return nil
						}
					}
				}
				return errors.New("d2d portal: no portal after the click")
			}()
			if last == nil {
				return nil
			}
			e.Log.Info("d2d town portal attempt failed", "try", try+1, "err", last)
			e.Clock.Sleep(300 * time.Millisecond)
		}
		return last
	}
}

// lateEmit is an env.Emit that can be handed out before the runner exists; events before bind are dropped.
type lateEmit struct {
	f atomic.Pointer[func(goals.Event)]
}

func (l *lateEmit) bind(f func(goals.Event)) { l.f.Store(&f) }

func (l *lateEmit) send(ev goals.Event) {
	if f := l.f.Load(); f != nil && *f != nil {
		(*f)(ev)
	}
}

// Package d2dcombat runs the bot's real combat layer (internal/combat) on the d2d backend: a StateReader from d2dsrc,
// input over d2dinput with the lockstep Pacer as the clock, hotkeys replaced by skill ids, potions by belt columns,
// and nav's Yield hook wired to Fighter.Clear. No behaviour of combat or nav changes; see docs/d2d-combat.md.
package d2dcombat

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dinput"
	"d2lod-bot/internal/d2dsrc"
	"d2lod-bot/internal/d2dworld"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/nav"
)

// Config is what New needs. Ctl is required and shared by every part (one client per socket).
type Config struct {
	Ctl    *d2dctl.Client
	Ctx    context.Context // bounds every channel call and the pacer; nil: Background
	Tables *data.DB        // skills, monsters, items; combat cannot run without it
	Log    *slog.Logger    // nil: discard
	Chunk  int             // max ticks per step call (0: Pacer default)
	// LinkWait is the world's wait (wall time) for neighbour levels on arrival; 0 disables it.
	LinkWait time.Duration
	Seed     int64 // handicap dice seed (0: combat's default)
}

// Rig is the wired bot side for one d2d instance.
type Rig struct {
	Ctl     *d2dctl.Client
	Src     *d2dsrc.Source
	Pace    *d2dinput.Pacer
	Meter   *Meter // the env clock: Pace plus damage/mana/death totals
	In      *Input
	World   *d2dworld.World
	Env     *env.Env
	Nav     *nav.Navigator
	Fighter *combat.Fighter
}

// New builds the rig. nav.Yield is set to a bounded Fighter.Clear (the same hook cmd/d2bot installs for Game.exe).
func New(cfg Config) (*Rig, error) {
	if cfg.Ctl == nil || cfg.Tables == nil {
		return nil, errors.New("d2dcombat: Ctl and Tables are required")
	}
	log := cfg.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	ctx := cfg.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	r := &Rig{Ctl: cfg.Ctl}
	r.Pace = &d2dinput.Pacer{C: cfg.Ctl, Ctx: ctx, Chunk: cfg.Chunk}
	r.Src = d2dsrc.New(cfg.Ctl, d2dsrc.Options{Tables: d2dsrc.NewDBTables(cfg.Tables), Logf: func(string, ...any) {}})
	r.Meter = &Meter{P: r.Pace, C: cfg.Ctl, Ctx: ctx}
	ctl := d2dinput.New(cfg.Ctl)
	ctl.Ctx, ctl.Clock = ctx, r.Meter
	ctl.Skills = map[string]int{}
	r.In = &Input{Controller: ctl, ctx: ctx}
	r.World = d2dworld.New(d2dworld.Client{Client: cfg.Ctl}, d2dworld.Options{Timeout: cfg.LinkWait + 30*time.Second, LinkWait: cfg.LinkWait,
		Sleep: r.Pace.Sleep, Now: r.Pace.Now}) // stepping the game is the wait; the old extra 50 ms wall sleep only slowed lockstep
	tune := env.DefaultTuning()
	r.Env = &env.Env{ID: "d2dcombat", Log: log, Read: viewReader{r.Src}, In: r.In, Data: cfg.Tables, Clock: r.Meter, Tune: &tune}
	r.Nav = nav.New(r.Env)
	r.Nav.UseWorld(r.World)
	r.Fighter = combat.New(r.Env, r.Nav)
	if cfg.Seed != 0 {
		r.Fighter.SetSeed(cfg.Seed)
	}
	r.Nav.Yield = r.yield()
	return r, nil
}

// Attach builds the combat side over an existing backend (d2denv.Backend) instead of a second reader and pacer: e is
// the backend env, nv its navigator (nv.Yield is set here), ctl the shared input controller (its Skills map is
// created). The Rig's Env is a copy of e with the viewport reader and the combat input; the clock is e.Clock, so under
// lockstep every combat sleep steps the game. Src, Pace and Meter stay nil.
func Attach(ctx context.Context, e *env.Env, nv *nav.Navigator, ctl *d2dinput.Controller) *Rig {
	if ctx == nil {
		ctx = context.Background()
	}
	ctl.Skills = map[string]int{}
	in := &Input{Controller: ctl, ctx: ctx}
	ce := *e
	ce.Read, ce.In = viewReader{e.Read}, in
	r := &Rig{Ctl: ctl.C, In: in, Env: &ce, Nav: nv}
	r.Fighter = combat.New(r.Env, nv)
	nv.Yield = r.yield()
	return r
}

// yield is the nav pack-stall hook: bot.PackYield around Fighter.Clear, as in cmd/d2bot/behaviors.go packYield
// (without its pinned-evade guard, which needs d2bot internals).
func (r *Rig) yield() func(context.Context, []d2.Unit) error {
	cat := combat.NewCatalog(r.Env.Data)
	return bot.PackYield(bot.YieldDeps{
		Timeout:  4 * time.Second,
		Snapshot: r.Env.Snapshot,
		Friendly: func(s *d2.Snapshot, u d2.Unit) bool { return combat.IsFriendly(s, u, cat) },
		Clear: func(ctx context.Context, radius int, timeout time.Duration) error {
			_, err := r.Fighter.Clear(ctx, combat.ClearOpts{Radius: radius, Timeout: timeout, Yield: true})
			return err
		},
	})
}

// BindSkills gives every skill of the character that combat's catalog knows a pseudo hotkey "sk<id>" and installs the
// mapping in both the input (CastAt becomes "cmd skill <id>") and the Fighter (it treats the skill as bound), which
// replaces EnsureHotkeys and its popup clicking. Call it again after skills change (level up, points spent).
func (r *Rig) BindSkills() (int, error) {
	snap, err := r.Env.Snapshot()
	if err != nil {
		return 0, err
	}
	if snap.Player == nil {
		return 0, errors.New("d2dcombat: no player in the snapshot")
	}
	cat := combat.NewCatalog(r.Env.Data)
	keys := map[int]string{}
	for _, sk := range snap.Player.Skills {
		if cat.Skill(sk.ID) == nil {
			continue
		}
		k := fmt.Sprintf("sk%d", sk.ID)
		keys[sk.ID] = k
		r.In.Skills[k] = sk.ID
	}
	r.Fighter.SetHotkeys(keys)
	return len(keys), nil
}

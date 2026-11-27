package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/points"
	"d2lod-bot/internal/power"
	"d2lod-bot/internal/watch"
)

// The points desk on the real bot (docs/points.md): the sheet, the explicit spend by verb, the per-hero autopilot switch
// and the gate that makes the bot bank points by default. Spending goes through the d2d `cmd stat` and `cmd skillpt`
// verbs; the Game.exe path keeps the leveler's clicks and the API answers "no verb" there.

// pointVerbs sends the verbs. The first answer that looks like "no such verb" switches it off with a reason, so a build
// without the verb is reported as such and never retried.
type pointVerbs struct {
	c     *d2dctl.Client
	db    *data.DB
	class func() string // the hero's class name
	off   atomic.Pointer[string]
}

func (v *pointVerbs) caps() points.Verbs {
	switch {
	case v == nil || v.c == nil:
		return points.Verbs{Via: "clicks (Game.exe path)", Why: "this backend spends by clicking the panels, which only the leveler does; use the in-game panels"}
	case v.off.Load() != nil:
		return points.Verbs{Via: "d2d verbs", Why: *v.off.Load()}
	}
	return points.Verbs{Stat: true, Skill: true, Via: "d2d verbs (cmd stat, cmd skillpt)"}
}

func (v *pointVerbs) note(err error) error {
	if err != nil {
		if m := strings.ToLower(err.Error()); strings.Contains(m, "unknown") || strings.Contains(m, "usage") {
			why := "this d2d build answers \"" + err.Error() + "\"; see docs/process/points-verbs-devp14.md"
			v.off.Store(&why)
		}
	}
	return err
}

func (v *pointVerbs) spendStat(ctx context.Context, statID int) error {
	return v.note(v.c.CmdStat(ctx, statID, 1))
}

func (v *pointVerbs) spendSkillIndex(ctx context.Context, idx int) error {
	return v.note(v.c.CmdSkillPt(ctx, idx))
}

// StatPoint and SkillPoint are leveler.PointVerbs: the leveler's attribute order is the sheet's.
func (v *pointVerbs) StatPoint(ctx context.Context, attr int) error {
	return v.spendStat(ctx, points.Stats[attr].ID)
}

func (v *pointVerbs) SkillPoint(ctx context.Context, skillID int) error {
	t := points.Tables{Skills: v.db.Skills, Classes: v.db.Classes}
	c := t.Class(v.class())
	if c == nil {
		return os.ErrInvalid
	}
	idx, ok := points.ClassSkillIndex(t, c.Code, skillID)
	if !ok {
		return os.ErrInvalid
	}
	return v.spendSkillIndex(ctx, idx)
}

var _ leveler.PointVerbs = (*pointVerbs)(nil)

// newPointVerbs is nil without a client (the leveler then clicks).
func newPointVerbs(c *d2dctl.Client, db *data.DB, class func() string) *pointVerbs {
	if c == nil || db == nil {
		return nil
	}
	return &pointVerbs{c: c, db: db, class: class}
}

// pointsGearSlots are the worn slots the sheet lists, in the order a person reads them.
var pointsGearSlots = []struct {
	body int
	name string
}{{d2.BodyHead, "head"}, {d2.BodyTorso, "body"}, {d2.BodyRHand, "weapon"}, {d2.BodyLHand, "shield"}, {d2.BodyGloves, "gloves"}, {d2.BodyFeet, "boots"}, {d2.BodyBelt, "belt"}}

// wornRequirements lists what the equipped items ask of the hero; items without a base row or without a requirement are left out.
func wornRequirements(s *d2.Snapshot, db *data.DB) []points.GearReq {
	if s == nil || s.Inventory == nil || db == nil {
		return nil
	}
	var out []points.GearReq
	for _, sl := range pointsGearSlots {
		u := s.Inventory.Equipped(sl.body)
		if u == nil || u.Class < 0 || u.Class >= len(db.Items) {
			continue
		}
		b := db.Items[u.Class]
		if b.ReqStr == 0 && b.ReqDex == 0 {
			continue
		}
		name := b.Display
		if name == "" {
			name = b.Name
		}
		out = append(out, points.GearReq{Slot: sl.name, Item: name, Str: b.ReqStr, Dex: b.ReqDex})
	}
	return out
}

// attackKinds is the set of skill kinds the power model resolves an attack skill with (profiles.json skillKinds).
func attackKinds() func(string) bool {
	p, err := power.Load()
	if err != nil {
		return nil
	}
	return func(n string) bool { _, ok := p.SkillKinds[n]; return ok }
}

// wirePoints builds the desk of the d2d backend, closes the bot's own spending behind the player's choice and hands the
// desk to the ladder wiring. c may be nil (a backend without verbs). Returns nil without game tables.
func wirePoints(stateDir, hero string, ref *runnerRef, r *bot.Runner, e *env.Env, c *d2dctl.Client, db *data.DB, pv *pointVerbs) *points.Desk {
	if ref == nil || r == nil || e == nil || db == nil || len(db.Skills) == 0 {
		return nil
	}
	dir := ""
	if stateDir != "" {
		dir = questStoreDir(stateDir)
	}
	res := func() env.ResourceTuning {
		if e.Tune == nil {
			return env.DefaultResourceTuning()
		}
		return e.Tune.Resource
	}
	cfg := points.Config{
		Hero: hero, T: points.Tables{Skills: db.Skills, Classes: db.Classes}, Dir: dir,
		Enabled: func() bool { return res().PointsApi != 0 },
		Player: func() *d2.Player {
			if s := r.LastSnapshot(); s != nil {
				return s.Player
			}
			return nil
		},
		Fresh: func(context.Context) (*d2.Player, error) {
			s, err := e.Snapshot()
			if err != nil {
				return nil, err
			}
			return s.Player, nil
		},
		Gear: func() []points.GearReq { return wornRequirements(r.LastSnapshot(), db) },
		Busy: r.FightOn,
		Verbs: func() points.Verbs {
			if pv == nil {
				return (*pointVerbs)(nil).caps()
			}
			return pv.caps()
		},
		Playable: combat.Playable,
		Attack:   attackKinds(),
	}
	if c != nil {
		cfg.Base = func(ctx context.Context) (*[4]int, bool) {
			a, ok, err := c.BaseAttrs(ctx)
			if err != nil || !ok {
				return nil, false
			}
			return &a, true
		}
	}
	if pv != nil {
		cfg.SpendStat, cfg.SpendSkill = pv.spendStat, pv.spendSkillIndex
	}
	fillAutoAlloc(&cfg, db, e, e.Log)
	d := points.NewDesk(cfg)
	// A locked build with the allocator on owns the points: the leveler's own spending stays closed so they never double-spend.
	r.SetPointsGate(func() bool { return (res().BotSpendsPoints() || d.Autopilot()) && !d.Locked() })
	ref.pd = d
	go runAutoAlloc(context.Background(), d, 3*time.Second)
	return d
}

// bindPoints mounts the points routes on the watch screen.
func bindPoints(f *watch.Fun, d *points.Desk) {
	if f == nil || d == nil {
		return
	}
	api := watch.NewPointsAPI()
	api.Bind(d)
	f.Points = api
	if f.Skills != nil && d.CanSpendSkills() {
		f.Skills.EnableAllocate(d) // the skill page then shows its + control
	}
}

// pointsHint is the second line of the points row: what lets the hero spend them (Questy bed 2026-11-25: a clvl 8 Paladin with
// 69 max life, every stat point banked, nobody had locked a build or set the knob).
const pointsHint = "Spend them on the Points tile, lock a build there (the hero then spends from it), turn autopilot on for this hero, or set resource.pointsDecisions to 1 to let the bot spend them from its plan table."

// pointsNeeds is the Needs-you row of banked points: one row while any stat or skill point waits.
func pointsNeeds(d *points.Desk) []watch.GlanceNeed {
	if d == nil {
		return nil
	}
	text, _, _, ok := d.Waiting()
	if !ok || d.Autopilot() {
		return nil
	}
	sub := pointsHint
	return []watch.GlanceNeed{{ID: "points", Kind: "points", Sev: "info", Text: text, Sub: sub, Actions: []watch.GlanceVerb{}}}
}

func init() {
	glanceExtra = append(glanceExtra, func(src *watch.GlanceSources, d glanceDeps) {
		if d.Points == nil {
			return
		}
		prev := src.Needs
		src.Needs = func() []watch.GlanceNeed {
			var out []watch.GlanceNeed
			if prev != nil {
				out = prev()
			}
			return append(out, pointsNeeds(d.Points())...)
		}
	})
}

// pointsStateName is where a hero's points files live, for the docs and the tests.
func pointsStateName(stateDir, hero string) string {
	return filepath.Join(questStoreDir(stateDir), hero+".points.json")
}

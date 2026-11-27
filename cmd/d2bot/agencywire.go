package main

import (
	"log/slog"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/agency"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/plandata"
	"d2lod-bot/internal/power"
	"d2lod-bot/internal/watch"
)

// The skill planner in shadow (docs/agency.md, knob agency.planner): from the skills the hero actually has it derives the
// rotation, the aura and a viability line, logs `planner shadow:` once a minute next to what the combat profile picks, and
// serves GET /api/hero/plan. Nothing reads its result: the hero plays as before.

// agencyWire is the planner on the real bot.
type agencyWire struct {
	pl    *agency.Planner
	db    *data.DB
	tune  func() env.AgencyTuning
	hero  func() string
	log   *slog.Logger
	lim   *agency.Limiter
	now   func() time.Time
	combo sync.Map                        // table name -> combat line
	owned func(*d2.Player) map[string]int // hard points by skill name; the leveler's reading of the skill list

	drvLim *agency.Limiter // mode 2: the plan is re-read at most every ten seconds
	mu     sync.Mutex
	drive  agency.Drive // mode 2: the last decision, read by the fighter
	snap   *d2.Snapshot
	view   *agency.View
	viewT  time.Time
}

// wireAgency returns nil without game tables or when the capability table or the model cannot load (one warning).
func wireAgency(db *data.DB, e *env.Env, hero func() string, now func() time.Time, log *slog.Logger) *agencyWire {
	if db == nil || len(db.Skills) == 0 || e == nil {
		return nil
	}
	aenv, err := agency.NewEnv()
	if err != nil {
		log.Warn("agency: no planner", "err", err)
		return nil
	}
	if miss := aenv.Caps.Overlay(db.SkillByName); len(miss) > 0 {
		log.Info("agency: capability rows without a skills.txt row", "skills", strings.Join(miss, ","))
	}
	p := &agency.Planner{Env: aenv}
	if tab := dangerTable(db); tab != nil {
		if m, err := power.NewModel(tab.Params); err == nil {
			p.Model, p.Tab = m, tab
		} else {
			log.Warn("agency: no survival model, the viability line is off", "err", err)
		}
	}
	if now == nil {
		now = time.Now
	}
	w := &agencyWire{pl: p, db: db, hero: hero, log: log, lim: agency.NewLimiter(time.Minute), drvLim: agency.NewLimiter(10 * time.Second), now: now,
		tune: func() env.AgencyTuning {
			if e.Tune == nil {
				return env.DefaultAgencyTuning()
			}
			return e.Tune.Agency
		}}
	w.owned = func(pl *d2.Player) map[string]int { return leveler.OwnedSkills(db, pl) }
	return w
}

// chainSnap runs the hooks in order on every snapshot.
func chainSnap(fns ...func(*d2.Snapshot)) func(*d2.Snapshot) {
	return func(s *d2.Snapshot) {
		for _, f := range fns {
			if f != nil {
				f(s)
			}
		}
	}
}

// OnSnapshot keeps the last snapshot and, when the knob says shadow, writes the minute's line. Quick: the planner runs at most
// once a minute here.
func (w *agencyWire) OnSnapshot(s *d2.Snapshot) {
	if w == nil || s == nil || s.Player == nil {
		return
	}
	w.mu.Lock()
	w.snap = s
	w.mu.Unlock()
	mode := w.tune().Planner
	if mode == agency.ModeDrive && w.drvLim.Allow(w.now()) {
		w.driveStep(s)
	}
	if mode != 1 || !w.lim.Allow(w.now()) {
		return
	}
	v, pl, err := w.eval(s)
	if err != nil {
		w.log.Warn("planner shadow: no plan", "err", err)
		return
	}
	w.store(v)
	w.log.Info(agency.ShadowLine(v, pl))
}

// driveStep (mode 2) re-reads the plan and keeps the decision the fighter asks for; a failed plan keeps the profile.
func (w *agencyWire) driveStep(s *d2.Snapshot) {
	v, pl, err := w.eval(s)
	d := agency.Drive{Why: "keep profile: no plan"}
	if err != nil {
		w.log.Warn("planner drive: no plan", "err", err)
	} else {
		v.Mode, v.ShadowOnly = "drive", false
		w.store(v)
		d = agency.DriveOf(pl)
	}
	w.mu.Lock()
	changed := w.drive.Attack != d.Attack || w.drive.Aura != d.Aura
	w.drive = d
	w.mu.Unlock()
	if changed {
		w.log.Info("planner drive: " + d.Why)
	}
}

// Drive is the fighter's seam (combat.Fighter.Drive): the planner's attack and aura at knob mode 2, "" otherwise (the
// profile plays, mode 0 and 1 are as before).
func (w *agencyWire) Drive() (attack, aura string) {
	if w == nil || w.tune().Planner != agency.ModeDrive {
		return "", ""
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.drive.Attack, w.drive.Aura
}

func (w *agencyWire) store(v agency.View) {
	w.mu.Lock()
	w.view, w.viewT = &v, w.now()
	w.mu.Unlock()
}

// View is the API's source: a fresh evaluation of the last snapshot at most every ten seconds; the "off" view with the
// reason when the knob is 0.
func (w *agencyWire) View() any {
	if w == nil {
		return agency.View{Schema: "didle.plan/1", Mode: "off", Note: "no game tables", Rotation: []agency.ViewSkill{}, Warnings: []agency.ViewWarn{}, ShadowOnly: true}
	}
	if w.tune().Planner == 0 {
		return agency.View{Schema: "didle.plan/1", Mode: "off", Note: "the planner is off (knob agency.planner 0)", Rotation: []agency.ViewSkill{}, Warnings: []agency.ViewWarn{}, ShadowOnly: true}
	}
	w.mu.Lock()
	s, v, at := w.snap, w.view, w.viewT
	w.mu.Unlock()
	if s == nil {
		return agency.View{Schema: "didle.plan/1", Enabled: true, Mode: w.modeName(), Note: "no snapshot yet", Rotation: []agency.ViewSkill{}, Warnings: []agency.ViewWarn{}, ShadowOnly: true}
	}
	if v != nil && w.now().Sub(at) < 10*time.Second {
		return *v
	}
	nv, _, err := w.eval(s)
	if err != nil {
		return agency.View{Schema: "didle.plan/1", Enabled: true, Mode: w.modeName(), Note: "no plan: " + err.Error(), Rotation: []agency.ViewSkill{}, Warnings: []agency.ViewWarn{}, ShadowOnly: true}
	}
	if w.tune().Planner == agency.ModeDrive {
		nv.Mode, nv.ShadowOnly = "drive", false
	}
	w.store(nv)
	return nv
}

func (w *agencyWire) modeName() string {
	if w.tune().Planner == agency.ModeDrive {
		return "drive"
	}
	return "shadow"
}

// eval reads the hero out of the snapshot and runs the planner.
func (w *agencyWire) eval(s *d2.Snapshot) (agency.View, agency.Plan, error) {
	p := s.Player
	class := strings.ToLower(p.Class.String())
	owned := w.owned(p)
	a := agency.Allocation{Class: class, Clvl: p.Level, Skills: owned}
	if int(p.Class) < len(w.db.Classes) {
		a.Ene = max(0, p.Energy-w.db.Classes[p.Class].Int)
	}
	hero := power.HeroFromPlayer(p, s.Merc, -1)
	in := agency.Input{Hero: w.hero(), Alloc: a, Combat: w.combatLine(class, p, owned), Power: hero, Diff: agencyDiff(p.Difficulty), Act: data.ActOfArea(p.Area)}
	if hero.DmgMax > 0 {
		in.Weapon.Avg = float64(hero.DmgMin+hero.DmgMax) / 2
	}
	in.FCR, in.IAS, in.GearRead = hero.Attack.FCR, hero.Attack.IAS, hero.Attack.FCRKnown && hero.Attack.IASKnown
	return w.pl.Evaluate(in)
}

func agencyDiff(d d2.Difficulty) string {
	return [...]string{"normal", "nightmare", "hell"}[min(max(int(d), 0), 2)]
}

// combatLine is the combat line of the plan table the leveler would pick for the hero (the profile the bot fights by).
func (w *agencyWire) combatLine(class string, p *d2.Player, owned map[string]int) string {
	name := leveler.SelectBuild(class, p.Level, owned)
	if c, ok := w.combo.Load(name); ok {
		return c.(string)
	}
	line := ""
	if raw, err := plandata.FS.ReadFile("build-" + name + ".json"); err == nil {
		if b, err := leveler.ParseBuild(raw); err == nil {
			line = b.Combat
		}
	}
	w.combo.Store(name, line)
	return line
}

// bindPlan mounts GET /api/hero/plan on the watch screen.
func bindPlan(f *watch.Fun, w *agencyWire) {
	if f == nil || w == nil {
		return
	}
	api := watch.NewPlanAPI()
	api.Bind(w.View)
	f.Plan = api
}

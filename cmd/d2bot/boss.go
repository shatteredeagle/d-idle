package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"d2lod-bot/internal/boss"
	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/nav"
	"d2lod-bot/internal/task"
)

// The boss.World / boss.Actions adapter (docs/boss.md section 3): nav, combat, loot and town behind the boss scripts.

// poisonCloudMissiles are the missiles.txt names treated as a standing poison hazard (Danger kind "poison-cloud").
// U: which of them Andariel really leaves is a docs/boss.md live check (6); the list is data, not logic.
var poisonCloudMissiles = []string{"poisonexplosioncloud", "poisonjavcloud", "queenpoisoncloud", "corpsepoisoncloud", "primepoisoncloud", "andypoisonbolt"}

type bossGame struct {
	e   *env.Env
	db  *data.DB
	cat combat.Catalog
	f   *combat.Fighter
	nv  *nav.Navigator
	lt  *loot.Looter
	// toTown is the Townsman's ToTown (reuses an open portal); loot walks use the navigator.
	toTown func(ctx context.Context) error

	pinned     pinState // consecutive Attack slices that ended near one spot
	cloudIDs   map[int]bool
	logPresets map[int]int
	seenLog    map[uint32]time.Time // unique units already logged (rate limit), docs/boss.md live checks 2, 3, 5, 6
}

func newBossGame(e *env.Env, db *data.DB, f *combat.Fighter, nv *nav.Navigator, lt *loot.Looter, toTown func(context.Context) error) *bossGame {
	g := &bossGame{e: e, db: db, cat: combat.NewCatalog(db), f: f, nv: nv, lt: lt, toTown: toTown, cloudIDs: map[int]bool{}, seenLog: map[uint32]time.Time{}, logPresets: map[int]int{}}
	if db != nil {
		for _, n := range poisonCloudMissiles {
			if m := db.MissileByName(n); m != nil {
				g.cloudIDs[m.ID] = true
			}
		}
	}
	return g
}

// ---- World ----

func (g *bossGame) Now() time.Time { return g.e.Clock.Now() }

func (g *bossGame) Sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	g.e.Clock.Sleep(d)
	return ctx.Err()
}

// View turns one snapshot into the script's view (docs/boss.md section 3 table).
func (g *bossGame) View(ctx context.Context) (boss.View, error) {
	if err := ctx.Err(); err != nil {
		return boss.View{}, err
	}
	s, err := g.e.Snapshot()
	if err != nil {
		return boss.View{}, err
	}
	if s.State != d2.StateInGame || s.Player == nil {
		return boss.View{}, errors.New("boss: not in a game")
	}
	v := g.viewOf(s)
	g.logUniques(s, v)
	return v, nil
}

// logUniques writes one line per unique/boss unit (flags, class, unique number, name, position, life) at most every 3 s:
// the data docs/boss.md section 5 asks to record (class ids, UniqueNo/Name offsets, spawn spots).
func (g *bossGame) logUniques(s *d2.Snapshot, v boss.View) {
	now := g.e.Clock.Now()
	for i := range s.Monsters {
		u := &s.Monsters[i]
		if u.Monster == nil || !(u.Monster.IsUnique() || u.Monster.IsChampion()) {
			continue
		}
		if t, ok := g.seenLog[u.ID]; ok && now.Sub(t) < 3*time.Second {
			continue
		}
		g.seenLog[u.ID] = now
		g.e.Log.Info("boss view: unique", "id", u.ID, "class", u.Class, "flags", u.Monster.Flags, "uniqueNo", u.Monster.UniqueNo,
			"name", u.Monster.Name, "x", u.X, "y", u.Y, "mode", u.Mode, "lifePct", u.Monster.LifePct, "area", v.Area,
			"playerX", v.Pos.X, "playerY", v.Pos.Y, "poisoned", v.Poisoned)
	}
}

func (g *bossGame) viewOf(s *d2.Snapshot) boss.View {
	p := s.Player
	belt := combat.BeltOf(s.Inventory, g.cat)
	v := boss.View{
		Area: p.Area, Difficulty: int(p.Difficulty), Level: p.Level, Pos: boss.Pt{X: p.X, Y: p.Y},
		Life: p.Life, Max: p.MaxLife, Mana: p.Mana, MaxMana: p.MaxMana, Dead: combat.IsDead(p),
		Poisoned: p.Poisoned(), PoisonRes: p.Resist.Poison,
		Antidotes: belt.Count(combat.PotionAntidote), Potions: belt.Count(combat.PotionHealing) + belt.Count(combat.PotionRejuv),
	}
	if g.e != nil {
		v.BeltSlots = combat.BeltCapacity(s.Inventory, g.e.Data)
	}
	if s.Inventory != nil {
		for _, u := range s.Inventory.Items {
			if u.Item != nil && g.cat.Item(u.Class).Portal {
				v.HasTP = true
				break
			}
		}
	}
	hostile := map[uint32]bool{}
	for _, t := range combat.Hostiles(s, combat.HostileCtx{Cat: g.cat, Diff: p.Difficulty, ClassLvl: p.Level}) {
		hostile[t.Unit.ID] = true
	}
	for i := range s.Monsters {
		u := &s.Monsters[i]
		m := boss.Mon{ID: u.ID, Class: u.Class, UniqueNo: -1, Pos: boss.Pt{X: u.X, Y: u.Y}, Mode: u.Mode, LifePct: 100, Hostile: hostile[u.ID]}
		if mi := u.Monster; mi != nil {
			if mi.IsUnique() {
				// 0 is also what every random unique carries; only the name says it is Bishibosh then
				if mi.UniqueNo != 0 || strings.EqualFold(mi.Name, "Bishibosh") {
					m.UniqueNo = int(mi.UniqueNo)
				}
				m.Name = mi.Name
				if m.Name == "" {
					m.Name = g.cat.MonsterName(g.cat.Monster(u.Class))
				}
			}
			m.Minion = mi.IsMinion()
			m.MaxLife = int(mi.MaxLife)
			if mi.LifePct > 0 {
				m.LifePct = min(mi.LifePct, 100)
			}
		}
		v.Monsters = append(v.Monsters, m)
	}
	for i := range s.Missiles {
		if u := &s.Missiles[i]; g.cloudIDs[u.Class] {
			v.Dangers = append(v.Dangers, boss.Danger{Kind: "poison-cloud", Pos: boss.Pt{X: u.X, Y: u.Y}})
		}
	}
	return v
}

// Presets lists the static units of the current level from the map tracker.
func (g *bossGame) Presets(ctx context.Context) ([]boss.Preset, error) {
	if g.e.World == nil {
		return nil, nil
	}
	if _, err := g.e.World.Update(); err != nil && !errors.Is(err, d2map.ErrNotInGame) {
		return nil, err
	}
	lm := g.e.World.Current()
	if lm == nil {
		return nil, nil
	}
	out := make([]boss.Preset, 0, len(lm.Presets))
	var npcs []string
	for _, p := range lm.Presets {
		out = append(out, boss.Preset{Kind: p.Kind, ID: p.ID, Pos: boss.Pt{X: p.Pos.X, Y: p.Pos.Y}})
		if p.Kind == d2map.PresetNPC && g.logPresets[lm.No] != len(lm.Presets) {
			npcs = append(npcs, fmt.Sprintf("%d@%d,%d", p.ID, p.Pos.X, p.Pos.Y))
		}
	}
	if g.logPresets[lm.No] != len(lm.Presets) { // monster presets of the level, once per change (docs/boss.md live checks 2 to 6)
		g.logPresets[lm.No] = len(lm.Presets)
		g.e.Log.Info("boss presets", "level", lm.No, "presets", len(lm.Presets), "monsterPresets", npcs)
	}
	return out, nil
}

// ---- Actions ----

type bossActs struct{ g *bossGame }

func (a bossActs) Travel(ctx context.Context, area int) error { return a.g.nv.ToArea(ctx, area) }
func (a bossActs) ExitTo(ctx context.Context, area int) error { return a.g.nv.ExitTo(ctx, area) }
func (a bossActs) TeleportTo(ctx context.Context, p boss.Pt) error {
	return a.g.nv.TeleportTo(ctx, p.X, p.Y)
}

func (a bossActs) GoTo(ctx context.Context, p boss.Pt) error {
	return a.g.nv.GoTo(ctx, input.Point{X: p.X, Y: p.Y})
}

// Explore reveals the level for at most budget and stops as soon as stop() is true. Nothing left to explore and the
// budget running out are both a normal end.
func (a bossActs) Explore(ctx context.Context, budget time.Duration, stop func() bool) error {
	ectx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	err := a.g.nv.Explore(ectx, stop)
	if errors.Is(err, nav.ErrExplored) || ctx.Err() == nil && errors.Is(err, context.DeadlineExceeded) {
		return nil
	}
	return err
}

// Attack is Fighter.Kill for one slice. Combat's escapes become the package's errors; the slice ending is not one.
func (a bossActs) Attack(ctx context.Context, id uint32, slice time.Duration) error {
	sctx, cancel := context.WithTimeout(ctx, slice)
	defer cancel()
	// V live (i2, Combi, 2026-10-29): pinned in the escort crowd, combat's kite and missile dodge moved nowhere for 10 minutes
	// and never cast (Countess stayed at 75%); with both off she died in 10 s. See unpinned.
	err := unpinned(a.g.e, &a.g.pinned, func() error { return a.g.f.Kill(sctx, id) })
	return mapKillErr(ctx, sctx, err)
}

func mapKillErr(parent, slice context.Context, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, combat.ErrTownPortal):
		return fmt.Errorf("%w: %v", boss.ErrEscaped, err)
	case errors.Is(err, combat.ErrChicken):
		return fmt.Errorf("%w: %v", boss.ErrChicken, err)
	case errors.Is(err, combat.ErrDead):
		return fmt.Errorf("%w: %v", boss.ErrDead, err)
	case parent.Err() != nil:
		return parent.Err()
	case slice.Err() != nil:
		return nil // the slice ran out: the script polls and attacks again
	}
	return err
}

func (a bossActs) ClearAround(ctx context.Context, p boss.Pt, radius int, budget time.Duration) error {
	origin := input.Point{X: p.X, Y: p.Y}
	sctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	_, err := a.g.f.Clear(sctx, combat.ClearOpts{Radius: radius, Origin: &origin, Timeout: budget})
	return mapKillErr(ctx, sctx, err)
}

func (a bossActs) Antidote(ctx context.Context) error { return a.g.f.Antidote(ctx) }

func (a bossActs) DrinkHealing(ctx context.Context) error {
	err := a.g.f.Act(ctx, combat.DrinkLife)
	if errors.Is(err, combat.ErrNoPotion) {
		err = a.g.f.Act(ctx, combat.DrinkRejuv)
	}
	return err
}

// Loot picks up ground items around the spec's center: quest items by name whatever the pickit says, the rest by the
// pickit ("any" = the loosest strictness). Only items on the ground are ever clicked, never a chest.
func (a bossActs) Loot(ctx context.Context, spec boss.LootSpec) (int, error) {
	c := input.Point{X: spec.Center.X, Y: spec.Center.Y}
	o := loot.PickOpts{Radius: spec.Radius, Names: spec.Quest}
	if spec.Center != (boss.Pt{}) {
		o.Center = &c
	}
	for _, k := range spec.Kinds {
		if k == "any" {
			o.StrictSet, o.Strictness = true, 0
		}
	}
	rep, err := a.g.lt.PickUp(ctx, o)
	return len(rep.Picked), err
}

func (a bossActs) ToTown(ctx context.Context) error { return a.g.toTown(ctx) }
func (a bossActs) Chicken(ctx context.Context) error {
	return a.g.f.Act(ctx, combat.Chicken)
}

// ---- runner ----

// superUniqueScripts are the scripts whose boss is a super unique matched by its number. V: the number is resolved
// from SuperUniques.txt by name. Council members (classes 345..347), Mephisto, Duriel and the other Act 2/3 bosses
// match by class, which is unique to them (docs/act3.md section 6), so they need no number.
// Hephasto, Shenk and Nihlathak carry their number from the 1.14d table in the script (V, docs/act4.md and act5.md); the
// lookup replaces it when the install says otherwise.
var superUniqueScripts = []string{boss.NameCountess, boss.NameHephasto, boss.NameShenk, boss.NameNihlathak}

// newBossRunner builds the runner over every act's scripts with the super unique numbers resolved from the tables.
func newBossRunner(g *bossGame) *actRunner {
	sc := boss.AllActs()
	if g.db != nil {
		for _, n := range superUniqueScripts {
			if su := g.db.SuperUniqueByName(n); su != nil && sc[n] != nil {
				sc[n] = sc[n].WithUniqueNo(su.ID)
			}
		}
	}
	return &actRunner{FarmRunner: boss.FarmRunner{Act45Runner: boss.Act45Runner{
		Runner: boss.Runner{W: g, A: bossActs{g}, Scripts: sc, Opts: boss.Opts{CanTeleport: g.nv != nil && g.nv.TeleportKey != "", AndarielMinLevel: andarielFloor(g.e), MaxRetreats: brMaxRetreats(g.e), Hooks: boss.Hooks{
			BeforeAttack: func(ctx context.Context, _ boss.View, _ boss.Mon) error { return g.f.Buff(ctx) },
		}}},
		Ops: objectAdapter{e: g.e, nv: g.nv},
	}}}
}

// actRunner is boss.Act45Runner (Diablo with seals, the Ancients, Baal with waves) plus the High Council: the task layer
// asks for one boss, the game has three members, so the name runs boss.RunCouncil (first member required, the other two
// best effort).
type actRunner struct{ boss.FarmRunner }

func (r *actRunner) KillBoss(ctx context.Context, name string) error {
	if r.Scripts[name] == nil && name != boss.NameAncients && name != boss.NamePindleskin && name != boss.NameEldritch { // the Ancients are three scripts run by RunAncients; // e.g. Fangskin, an optional guard: not found lets the optional step skip
		return task.Errorf(task.KindNotFound, "boss: no script for %q", name)
	}
	if name != boss.NameCouncil {
		return r.FarmRunner.KillBoss(ctx, name)
	}
	res, err := boss.RunCouncil(ctx, r.W, r.A, r.Opts)
	r.Last = res
	return err
}

// bossRunner is what bossKill needs: both runners provide it.
type bossRunner interface {
	KillBoss(ctx context.Context, name string) error
	LastResult() boss.Result
}

func (r *actRunner) LastResult() boss.Result { return r.Last }

// bossKill is bot.TaskDeps.Boss: run the script and map its errors to the bot's sentinels and the task kinds
// (docs/boss.md section 3).
func bossKill(r bossRunner, log func(msg string, args ...any)) func(ctx context.Context, name string) error {
	return func(ctx context.Context, name string) error {
		err := r.KillBoss(ctx, name)
		res := r.LastResult()
		log("boss run", "boss", name, "killed", res.Killed, "loot", res.Loot, "phases", strings.Join(res.Phases, ","),
			"notes", strings.Join(res.Notes, "; "), "elapsed", res.Elapsed.Round(time.Second).String(), "bossMaxLife", res.BossMaxLife, "err", err)
		return mapBossErr(err)
	}
}

func mapBossErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, boss.ErrEscaped):
		return fmt.Errorf("%w: %v", bot.ErrPortalOpened, err)
	case errors.Is(err, boss.ErrChicken):
		return fmt.Errorf("%w: %v", bot.ErrLeftGame, err)
	case errors.Is(err, boss.ErrTooWeak), errors.Is(err, boss.ErrAborted), errors.Is(err, boss.ErrLeftArena):
		return task.Wrap(task.KindDanger, err)
	case errors.Is(err, boss.ErrBossNotFound), errors.Is(err, boss.ErrBossLost):
		return task.Wrap(task.KindNotFound, err)
	case errors.Is(err, boss.ErrTimeout):
		return task.Wrap(task.KindTimeout, err)
	case errors.Is(err, boss.ErrDead):
		return task.Wrap(task.KindDeath, err)
	}
	return err
}

var (
	_ boss.World   = (*bossGame)(nil)
	_ boss.Actions = bossActs{}
)

// pinState tracks where consecutive unpinned calls ended.
type pinState struct {
	x, y, n int
}

// unpinned runs fn; after three consecutive calls that all ended within 6 subtiles of the first (the character is pinned
// in a crowd and combat's kite and dodge moves go nowhere instead of casting), the next calls run with kite and missile
// dodge off until the character gets away. The tuning is restored afterwards. V live (i2, Combi, 2026-10-29): the strict
// "same tile" variant never fired in the Barracks (the dodge nudges the character a tile or two each time).
func unpinned(e *env.Env, st *pinState, fn func() error) error {
	tn := &e.Tune.Combat
	saveK, saveD := tn.KiteEnabled, tn.DodgeRadius
	if st.n >= 3 {
		tn.KiteEnabled, tn.DodgeRadius = 0, 0
	}
	err := fn()
	tn.KiteEnabled, tn.DodgeRadius = saveK, saveD
	if after, e2 := e.Snapshot(); e2 == nil && after != nil && after.Player != nil {
		x, y := after.Player.X, after.Player.Y
		if st.n > 0 && abs(x-st.x) <= 6 && abs(y-st.y) <= 6 {
			st.n++
		} else {
			st.x, st.y, st.n = x, y, 1
		}
	}
	return err
}

// farmLoop is TaskDeps.Farm: boss.FarmLoop over the boss runner and this game, with the table from the plan data.
// Teleport (a FarmChar field the bot package cannot see) is filled from the navigator.
func farmLoop(g *bossGame, r *actRunner, log func(msg string, args ...any)) func(context.Context, bot.FarmCall) (boss.FarmReport, error) {
	return func(ctx context.Context, c bot.FarmCall) (boss.FarmReport, error) {
		tab, err := boss.DefaultFarmTable()
		if err != nil {
			return boss.FarmReport{}, err
		}
		fixed := c.Run
		if strings.EqualFold(fixed, "auto") {
			fixed = ""
		}
		tele := g.nv != nil && g.nv.TeleportKey != ""
		return boss.FarmLoop(ctx, g, teleOps{c.Ops, tele}, r, tab, boss.FarmOpts{
			Fixed: fixed, FirstHere: c.FirstHere, Limits: boss.Limits{Runs: c.Runs, Duration: c.Duration}, OnRun: c.OnRun,
			Log: func(f string, a ...any) { log("farm", "msg", fmt.Sprintf(f, a...)) },
		})
	}
}

type teleOps struct {
	boss.FarmOps
	tele bool
}

func (t teleOps) Char(ctx context.Context) (boss.FarmChar, error) {
	c, err := t.FarmOps.Char(ctx)
	c.Teleport = t.tele
	return c, err
}

// andarielFloor is the Andariel clvl knob (0 = the script's own floor when there is no env).
func andarielFloor(e *env.Env) int {
	if e == nil {
		return 0
	}
	return e.Tune.Boss.AndarielMinLevel
}

// brMaxRetreats is the boss.brMaxRetreats knob (0 = the script's own number, also without an env).
func brMaxRetreats(e *env.Env) int {
	if e == nil || e.Tune == nil {
		return 0
	}
	return e.Tune.Boss.BRMaxRetreats
}

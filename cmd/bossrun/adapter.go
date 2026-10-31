package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"d2lod-bot/internal/boss"
	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/nav"
)

// COPY of cmd/d2bot/boss.go (package main cannot be imported), minus bossKill/mapBossErr. When the d2bot adapter changes,
// re-copy it here (diff the two files); the bossrun-only parts are in main.go and town.go.

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
	var npcs, objs []string
	for _, p := range lm.Presets {
		out = append(out, boss.Preset{Kind: p.Kind, ID: p.ID, Pos: boss.Pt{X: p.Pos.X, Y: p.Pos.Y}})
		if p.Kind == d2map.PresetNPC && g.logPresets[lm.No] != len(lm.Presets) {
			npcs = append(npcs, fmt.Sprintf("%d@%d,%d", p.ID, p.Pos.X, p.Pos.Y))
		}
		if p.Kind == 2 && g.logPresets[lm.No] != len(lm.Presets) {
			objs = append(objs, fmt.Sprintf("%d@%d,%d", p.ID, p.Pos.X, p.Pos.Y))
		}
	}
	if g.logPresets[lm.No] != len(lm.Presets) { // monster presets of the level, once per change (docs/boss.md live checks 2 to 6)
		g.logPresets[lm.No] = len(lm.Presets)
		g.e.Log.Info("boss presets", "level", lm.No, "presets", len(lm.Presets), "monsterPresets", npcs, "objectPresets", objs)
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

// newBossRunner builds the Runner with the Countess' super unique number resolved from the tables.
func newBossRunner(g *bossGame) *boss.Runner {
	sc := boss.Act1()
	if g.db != nil {
		if su := g.db.SuperUniqueByName(boss.NameCountess); su != nil {
			sc[boss.NameCountess] = sc[boss.NameCountess].WithUniqueNo(su.ID)
		}
	}
	return &boss.Runner{W: g, A: bossActs{g}, Scripts: sc, Opts: boss.Opts{AndarielMinLevel: andarielFloor(g.e), Hooks: boss.Hooks{
		BeforeAttack: func(ctx context.Context, _ boss.View, _ boss.Mon) error { return g.f.Buff(ctx) },
	}}}
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

// andarielFloor is the Andariel clvl knob (0 = the script's own floor when there is no env).
func andarielFloor(e *env.Env) int {
	if e == nil {
		return 0
	}
	return e.Tune.Boss.AndarielMinLevel
}

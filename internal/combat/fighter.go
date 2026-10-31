package combat

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/gear"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/resource"
	"d2lod-bot/internal/town/panel"
)

// Fighter kills things and keeps the character alive. It is used by one goroutine at a time.
type Fighter struct {
	Build Build
	// GearBuild is the leveler build name (gear.Builds); with an Amazon its main_attack picks the first skill.
	GearBuild string
	// Leave exits the game (Save and Exit). The default presses Esc and clicks the placeholder button; the bot
	// should install launch.Flow.LeaveGame here.
	Leave func(ctx context.Context) error
	// Portal opens a town portal. The default right-clicks the scroll or tome in the inventory panel
	// (UNVERIFIED geometry); the town agent can install a better one.
	Portal func(ctx context.Context) error

	e   *env.Env
	mv  Mover
	cat Catalog

	povMu sync.Mutex
	pov   poverty

	snapFn  func() (*d2.Snapshot, error)
	gridFn  func(*d2.Snapshot) (panel.Grid, error) // tests; nil = the data tables
	sight   func(a, b input.Point) bool            // tests; nil = Sighter or the collision grid
	ignores map[uint32]ignoreRec
	rng     *rand.Rand

	keys          map[int]string // skill id -> verified F-key
	lastDrink     map[PotionKind]time.Time
	manaChickenAt time.Time // last chicken caused by the mana-burn watch (hard.go)
	lastCast      time.Time
	lastCastSkill map[int]time.Time
	buffAt        map[int]time.Time
	lastDodge     time.Time
	diagMu        sync.Mutex // guards portalCast and casts (read from the bot's autopsy)
	portalCast    PortalCast
	casts         []castRec
	walking       atomic.Bool // the bot is on a travel or quest walk (SetWalking)
	pinned        int         // consecutive evasive moves that went nowhere (evade)
	noEvadeUntil  time.Time   // pinned: no kite or dodge before this
	resCache      map[uint32]resEntry
	health        healthWindow
	curve         []LifeSample
	weights       DeathWeights
	listRows      map[int]int // skill id -> skilldesc ListRow (popup rows)

	leftCell    map[int]input.Screen // skill id -> left popup cell that selected it (bindleft.go)
	leftFail    map[int]time.Time    // skill id -> do not try to select it as left skill before this time
	auraFn      func(*d2.Player) (int, bool)
	auraChecked time.Time
	auraArea    int
	auraDirty   bool // death or area change: the aura must be applied again
	necro       necroState
	trap        trapMem

	eq            *gear.Equipped // last noted gear, nil when unknown
	weaponProblem string
	weaponLogged  string
	statsFn       func(uint32) (d2.Stats, error) // test seam; nil = e.Read.UnitStats
	skipWhy       string                         // last skipped-drink reason logged (noteDrinkSkip)
	skipAt        time.Time
	portalBackoff time.Time // no TownPortal safety action before this (diagMu)
}

type resEntry struct {
	at  time.Time
	res [6]int
	ok  bool
}

// New builds a Fighter. mover may be nil for callers that only use Safety / Act / Buff.
func New(e *env.Env, mover Mover) *Fighter {
	f := &Fighter{
		e: e, mv: mover, cat: NewCatalog(e.Data),
		rng:           rand.New(rand.NewSource(1)),
		keys:          map[int]string{},
		lastDrink:     map[PotionKind]time.Time{},
		lastCastSkill: map[int]time.Time{},
		buffAt:        map[int]time.Time{},
		leftCell:      map[int]input.Screen{},
		leftFail:      map[int]time.Time{},
		auraDirty:     true,
		resCache:      map[uint32]resEntry{},
	}
	f.snapFn = e.Snapshot
	if e.Data != nil {
		SetTables(e.Data)
	}
	return f
}

// SetCatalog replaces the game data lookup (tests; running without extracted data).
func (f *Fighter) SetCatalog(c Catalog) { f.cat = c }

// SetSeed seeds the handicap dice so runs are reproducible.
func (f *Fighter) SetSeed(seed int64) { f.rng = rand.New(rand.NewSource(seed)) }

// SetHotkeys installs a known skill-id to key mapping (e.g. after EnsureHotkeys of an earlier session).
func (f *Fighter) SetHotkeys(m map[int]string) {
	f.keys = map[int]string{}
	for k, v := range m {
		f.keys[k] = v
	}
}

// Hotkeys returns the verified skill-id to key mapping.
func (f *Fighter) Hotkeys() map[int]string {
	out := map[int]string{}
	for k, v := range f.keys {
		out[k] = v
	}
	return out
}

func (f *Fighter) tune() env.CombatTuning { return f.e.Tune.Combat }
func (f *Fighter) now() time.Time         { return f.e.Clock.Now() }
func (f *Fighter) log() *slog.Logger {
	if f.e.Log != nil {
		return f.e.Log
	}
	return slog.Default()
}

func (f *Fighter) sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d > 0 {
		f.e.Clock.Sleep(d)
	}
	return ctx.Err()
}

func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

func (f *Fighter) emit(ev goals.Event) {
	if f.e.Emit == nil {
		return
	}
	if ev.Time.IsZero() {
		ev.Time = f.now()
	}
	f.e.Emit(ev)
}

// read returns the next in-game snapshot, waiting through loading screens.
func (f *Fighter) read(ctx context.Context) (*d2.Snapshot, error) {
	limit := f.now().Add(15 * time.Second)
	for {
		s, err := f.snapFn()
		if err != nil {
			return nil, err
		}
		if s.State == d2.StateInGame && s.Player != nil {
			return s, nil
		}
		if f.now().After(limit) {
			return nil, ErrNotInGame
		}
		if err := f.sleep(ctx, ms(f.tune().PollMs)); err != nil {
			return nil, err
		}
	}
}

func isDead(p *d2.Player) bool { return IsDead(p) }

func (f *Fighter) hostileCtx(p *d2.Player) HostileCtx {
	return HostileCtx{Cat: f.cat, Diff: p.Difficulty, ClassLvl: p.Level, Weights: f.weights}
}

// ---- survival -------------------------------------------------------------------------------------------

// Safety is the pure survival decision for one snapshot (cooldowns come from the Fighter's own drink times).
func (f *Fighter) Safety(snap *d2.Snapshot) SafetyAction {
	if snap == nil || snap.Player == nil || snap.State != d2.StateInGame {
		return SafetyNone
	}
	p, t := snap.Player, f.tune()
	now := f.now()
	ready := func(k PotionKind, gap int) bool {
		at, ok := f.lastDrink[k]
		return !ok || now.Sub(at) >= ms(gap)
	}
	in := SafetyInput{
		InTown: p.InTown(), Life: p.Life, Max: p.MaxLife, Mana: p.Mana, MMax: p.MaxMana, Diff: p.Difficulty,
		Belt:       BeltOf(snap.Inventory, f.cat),
		HealReady:  ready(PotionHealing, t.PotionGapMs),
		ManaReady:  ready(PotionMana, t.PotionGapMs),
		RejuvReady: ready(PotionRejuv, t.RejuvGapMs),
		Tune:       t,
	}
	in.Tune.DrinkManaPct = 0 // the mana potion is the resource policy's call (below), not Decide's
	hs := Hostiles(snap, f.hostileCtx(p))
	for _, h := range hs {
		if h.Dist <= float64(t.FightRadius) {
			in.InFight = true
			break
		}
	}
	in.Walking = f.walking.Load()
	if t.CrowdN > 0 {
		if in.Crowd, in.Adjacent = Crowd(p, hs); in.Crowd >= t.CrowdN {
			in.CanRetreat = f.canRetreat(snap, hs)
		}
	}
	if snap.Inventory != nil && !f.portalBackedOff() {
		in.HasPortal = f.portalItem(snap) != nil
	}
	a := Decide(in)
	if why := drinkSkipReason(in, a); why != "" {
		f.noteDrinkSkip(why, in)
	}
	if a != SafetyNone {
		return a
	}
	// Amplify Damage / Iron Maiden on us: drink early, or leave when nothing is left to drink
	if in.InFight && CurseRetreat(p, t.CurseRetreatPct) {
		switch {
		case in.Belt.Count(PotionHealing) > 0 && in.HealReady:
			return DrinkLife
		case in.Belt.Count(PotionRejuv) > 0 && in.RejuvReady:
			return DrinkRejuv
		case in.Belt.Count(PotionHealing)+in.Belt.Count(PotionRejuv) == 0 && in.HasPortal:
			return TownPortal
		}
	}
	// mana potion: PotionPolicy (poor characters, melee fallback, cast budget; docs/resource.md)
	if p.MaxMana > 0 && in.ManaReady {
		ri := f.resInputSnap(snap)
		d := resource.PotionPolicy(ri, f.rtune())
		if p.Difficulty != d2.Normal {
			d = resource.PotionPolicyHell(ri, f.rtune(), f.hellCtx(snap, Hostiles(snap, f.hostileCtx(p))))
		}
		if d.Drink == resource.PotionMana {
			return DrinkMana
		}
	}
	return SafetyNone
}

// Act carries a survival action out. Drinking presses the belt key of a column holding that potion; TownPortal
// right-clicks the scroll or tome in the inventory; Chicken leaves the game.
func (f *Fighter) Act(ctx context.Context, a SafetyAction) error {
	switch a {
	case SafetyNone:
		return nil
	case DrinkLife, DrinkMana, DrinkRejuv:
		return f.drink(ctx, a)
	case TownPortal:
		if f.Portal != nil {
			return f.Portal(ctx)
		}
		return f.openPortal(ctx)
	case Chicken:
		return f.leave(ctx)
	case CrowdRetreat, CrowdAbandon:
		return f.crowdAct(ctx, a)
	}
	return nil
}

func (f *Fighter) drink(ctx context.Context, a SafetyAction) error {
	snap, err := f.read(ctx)
	if err != nil {
		return err
	}
	kind := map[SafetyAction]PotionKind{DrinkLife: PotionHealing, DrinkMana: PotionMana, DrinkRejuv: PotionRejuv}[a]
	belt := BeltOf(snap.Inventory, f.cat)
	// an emergency takes the strongest potion, a scratch the weakest
	best := a == DrinkRejuv || pct(snap.Player.Life, snap.Player.MaxLife) < f.tune().DrinkRejuvPct
	col, ok := belt.ColumnFor(kind, best)
	if !ok {
		return ErrNoPotion
	}
	if err := f.e.In.Key(strconv.Itoa(col + 1)); err != nil {
		return err
	}
	f.lastDrink[kind] = f.now()
	f.info("potion", "kind", kind.String(), "col", col+1, "life", snap.Player.Life, "mana", snap.Player.Mana)
	f.emit(goals.Event{Type: goals.EvPotion, Potion: kind.String(),
		Message: "Drank a " + kind.String() + " potion at " + strconv.Itoa(pct(snap.Player.Life, snap.Player.MaxLife)) + "% life"})
	return nil
}

// Antidotes is the number of antidote potions in the belt.
func (f *Fighter) Antidotes(snap *d2.Snapshot) int {
	return BeltOf(snap.Inventory, f.cat).Count(PotionAntidote)
}

// Antidote drinks one antidote potion from the belt (ErrNoPotion when there is none). Poison is cured by the state
// ending, so the caller checks Player.Poisoned afterwards.
func (f *Fighter) Antidote(ctx context.Context) error {
	snap, err := f.read(ctx)
	if err != nil {
		return err
	}
	col, ok := BeltOf(snap.Inventory, f.cat).ColumnFor(PotionAntidote, false)
	if !ok {
		return ErrNoPotion
	}
	if err := f.e.In.Key(strconv.Itoa(col + 1)); err != nil {
		return err
	}
	f.lastDrink[PotionAntidote] = f.now()
	f.info("potion", "kind", PotionAntidote.String(), "col", col+1, "poisoned", snap.Player.Poisoned())
	f.emit(goals.Event{Type: goals.EvPotion, Potion: PotionAntidote.String(), Message: "Drank an antidote potion"})
	return nil
}

func (f *Fighter) info(msg string, args ...any) { f.log().Info("combat "+msg, args...) }

func (f *Fighter) leave(ctx context.Context) error {
	if f.Leave != nil {
		return f.Leave(ctx)
	}
	// UNVERIFIED placeholder: Esc menu, then the Save and Exit row (internal/launch/oog_table.go BtnSaveAndExit).
	if err := f.e.In.Key("esc"); err != nil {
		return err
	}
	if err := f.sleep(ctx, 500*time.Millisecond); err != nil {
		return err
	}
	return f.e.In.Click(input.Left, 400, 315)
}

// objTownPortal is the town portal object class (objects.txt "Portal"; id U, as in internal/town/acts.go).
const objTownPortal = 59

const portalAttempts = 3

// openPortal casts Town Portal from the scroll or tome in the inventory panel. Each attempt rereads the inventory,
// makes sure the panel is open (the "i" key toggles, so it is only pressed when the UI flag is off), clicks the
// item centre computed with the same screen-offset layout the town agent uses, closes the panels and checks that
// the cast happened (portal object next to the player, or the scroll gone).
//
// History: BOSSLIVE (2026-10-29) failed 5 times with "pointer is not over the portal scroll". Cause (V by code
// reading, not re-run live): the old click point used the raw 640x480 inventory.txt cell without the (screen-640,
// screen-480)/2 offset of an 800x600 window, so it never hit the item, and the check compared s.Hovered.ID (the
// world selection, which is not set by inventory items, U) with the item id, which can never match. The hover
// check is gone; success is verified from the game state instead.
func (f *Fighter) openPortal(ctx context.Context) error {
	start := f.now()
	f.diagMu.Lock()
	f.portalCast = PortalCast{Started: true, Start: start}
	f.diagMu.Unlock()
	err := f.openPortalTries(ctx)
	f.diagMu.Lock()
	f.portalCast.Dur, f.portalCast.OK = f.now().Sub(start), err == nil
	f.diagMu.Unlock()
	return err
}

func (f *Fighter) openPortalTries(ctx context.Context) error {
	var last error
	for try := 0; try < portalAttempts; try++ {
		if try > 0 {
			f.e.In.CloseAll()
			if err := f.sleep(ctx, 300*time.Millisecond); err != nil {
				return err
			}
		}
		f.drinkIfDue(ctx)
		before := f.portalScrolls()
		done, err := f.portalAttempt(ctx)
		if done {
			return nil
		}
		if errors.Is(err, ErrNoPortal) || ctx.Err() != nil {
			return err
		}
		last = err
		f.log().Info(fmt.Sprintf("portal: attempt %d failed because %s scrolls=%d->%d", try+1,
			strings.TrimPrefix(err.Error(), "combat: "), before, f.portalScrolls()))
	}
	// An emergency re-fires every tick; three clicks every few seconds only burn scrolls and time.
	f.diagMu.Lock()
	f.portalBackoff = f.now().Add(portalBackoff)
	f.diagMu.Unlock()
	return fmt.Errorf("combat: town portal not opened after %d tries: %w", portalAttempts, last)
}

// drinkIfDue drinks a healing potion before a portal try when life is under the drink line, a hostile is in reach and
// the potion is ready. A cast is a blocking sequence of about 2 s per try (6 s for three failed tries) that Safety
// cannot interrupt; Stony Field i4 and Dark Wood i5 (2026-10-30) died inside it with potions in the belt.
func (f *Fighter) drinkIfDue(ctx context.Context) {
	snap, err := f.snapFn()
	if err != nil || snap == nil || snap.Player == nil || snap.State != d2.StateInGame || snap.Player.InTown() {
		return
	}
	p, t := snap.Player, f.tune()
	if pct(p.Life, p.MaxLife) >= t.DrinkLifePct || BeltOf(snap.Inventory, f.cat).Count(PotionHealing) == 0 {
		return
	}
	if at, ok := f.lastDrink[PotionHealing]; ok && f.now().Sub(at) < ms(t.PotionGapMs) {
		return
	}
	for _, h := range Hostiles(snap, f.hostileCtx(p)) {
		if h.Dist <= float64(t.FightRadius) {
			if err := f.drink(ctx, DrinkLife); err != nil {
				f.info("potion: drink during the portal cast failed", "err", err)
			}
			return
		}
	}
}

// portalBackoff is how long a failed cast (all tries) silences the TownPortal safety action, so the fight falls
// back to potions, retreat and the chicken rule instead of retrying the same clicks.
const portalBackoff = 30 * time.Second

func (f *Fighter) portalBackedOff() bool {
	f.diagMu.Lock()
	defer f.diagMu.Unlock()
	return f.now().Before(f.portalBackoff)
}

// portalScrolls is the number of casts the pack holds: scrolls plus tome quantity (-1 when unreadable).
func (f *Fighter) portalScrolls() int {
	s, err := f.snapFn()
	if err != nil || s == nil || s.Inventory == nil {
		return -1
	}
	n := 0
	for i := range s.Inventory.Items {
		u := &s.Inventory.Items[i]
		if u.Item == nil || u.Item.Where != d2.ContInventory || !f.cat.Item(u.Class).Portal {
			continue
		}
		n += f.portalQty(u)
	}
	return n
}

// portalQty is the casts left in one portal item. A scroll is one cast. A tome's count is its quantity stat, and
// an EMPTY tome has no stat list at all (V live on i4 per internal/town/shop.go qty; Fresha i5 2026-10-30 clicked an
// empty tome 159 times). Without a stats source the tome counts as usable, like the town code.
func (f *Fighter) portalQty(u *d2.Unit) int {
	if !f.cat.Item(u.Class).Tome {
		return 1
	}
	var st d2.Stats
	var err error
	switch {
	case f.statsFn != nil:
		st, err = f.statsFn(u.Addr)
	case f.e.Read != nil:
		st, err = f.e.Read.UnitStats(u.Addr)
	default:
		return 1
	}
	if err != nil || !st.Has(statQuantity) {
		return 0
	}
	return int(st.Get(statQuantity))
}

func (f *Fighter) portalAttempt(ctx context.Context) (bool, error) {
	snap, err := f.read(ctx)
	if err != nil {
		return false, err
	}
	it := f.portalItem(snap)
	if it == nil {
		return false, ErrNoPortal
	}
	grid, err := f.inventoryGrid(snap)
	if err != nil {
		return false, err
	}
	if !snap.UI.Open(d2.UIInventory) {
		if err := f.e.In.Key("i"); err != nil {
			return false, err
		}
		if !f.waitUI(ctx, d2.UIInventory) {
			return false, errors.New("combat: inventory panel did not open")
		}
	}
	defer f.e.In.CloseAll()
	w, h := f.itemSize(it.Class)
	c := grid.ItemCenter(it.Item.GX, it.Item.GY, w, h)
	if err := f.e.In.MoveMouse(c.X, c.Y); err != nil {
		return false, err
	}
	if err := f.sleep(ctx, 100*time.Millisecond); err != nil {
		return false, err
	}
	if err := f.e.In.Click(input.Right, c.X, c.Y); err != nil {
		return false, err
	}
	for i := 0; i < 6; i++ {
		if err := f.sleep(ctx, 150*time.Millisecond); err != nil {
			return false, err
		}
		s, err := f.snapFn()
		if err != nil || s.Player == nil {
			continue
		}
		if portalCast(s, it) {
			f.info("town portal cast", "item", it.ID, "gx", it.Item.GX, "gy", it.Item.GY, "at", c)
			return true, nil
		}
	}
	return false, errors.New("combat: no portal appeared after the click")
}

// portalItem is the first scroll or non-empty tome in the main pack.
func (f *Fighter) portalItem(snap *d2.Snapshot) *d2.Unit {
	if snap.Inventory == nil {
		return nil
	}
	for i := range snap.Inventory.Items {
		u := &snap.Inventory.Items[i]
		if u.Item == nil || u.Item.Where != d2.ContInventory || !f.cat.Item(u.Class).Portal || f.portalQty(u) <= 0 {
			continue
		}
		return u
	}
	return nil
}

// portalCast is true when the cast is visible: a portal object within 25 subtiles of the player, or the scroll's
// unit no longer in the pack. A tome only shows through the object (its quantity is not read here).
func portalCast(s *d2.Snapshot, it *d2.Unit) bool {
	for _, o := range s.Objects {
		if o.Class == objTownPortal && math.Hypot(float64(o.X-s.Player.X), float64(o.Y-s.Player.Y)) < 25 {
			return true
		}
	}
	if s.Inventory != nil {
		for _, u := range s.Inventory.Items {
			if u.ID == it.ID {
				return false
			}
		}
		return true
	}
	return false
}

func (f *Fighter) waitUI(ctx context.Context, idx int) bool {
	for i := 0; i < 8; i++ {
		if f.sleep(ctx, 150*time.Millisecond) != nil {
			return false
		}
		if s, err := f.snapFn(); err == nil && s.UI.Open(idx) {
			return true
		}
	}
	return false
}

func (f *Fighter) itemSize(class int) (w, h int) {
	if db := f.e.Data; db != nil && class >= 0 && class < len(db.Items) && db.Items[class].Width > 0 {
		return db.Items[class].Width, db.Items[class].Height
	}
	return 1, 1
}

// inventoryGrid is the backpack grid of the player's class in the current window size (800x600 when unread).
func (f *Fighter) inventoryGrid(s *d2.Snapshot) (panel.Grid, error) {
	if f.gridFn != nil {
		return f.gridFn(s)
	}
	if f.e.Data == nil {
		return panel.Grid{}, errors.New("combat: no game data for the inventory layout")
	}
	scr := panel.Pt{X: s.Screen.X, Y: s.Screen.Y}
	if scr.X <= 0 || scr.Y <= 0 {
		scr = panel.Pt{X: 800, Y: 600}
	}
	l, err := panel.NewLayout(f.e.Data, s.Player.Class, scr, true)
	if err != nil {
		return panel.Grid{}, err
	}
	return l.Inventory, nil
}

// ---- buffs ----------------------------------------------------------------------------------------------

// Buff casts the armors and shields that are missing. The buff state is not readable yet (UNVERIFIED offset), so
// "missing" means not cast within BuffRecastSec.
func (f *Fighter) Buff(ctx context.Context) error {
	snap, err := f.read(ctx)
	if err != nil {
		return err
	}
	p := snap.Player
	if _, err := f.KeepAura(ctx, snap, nil, true); err != nil {
		return err
	}
	known := KnownSkills(p, f.cat, f.keys)
	mana := p.Mana
	for _, group := range f.profileFor(p).Buffs {
		for _, name := range group {
			sp, ok := known[name]
			if !ok || curseNames[name] || (sp.Key == "" && p.RightSkill != sp.ID) {
				continue
			}
			if !f.buffDue(p, sp) {
				break
			}
			if sp.Cost > mana {
				break
			}
			if err := f.cast(snap, sp, input.Point{X: p.X, Y: p.Y}); err != nil {
				return err
			}
			f.buffAt[sp.ID] = f.now()
			mana -= sp.Cost
			f.info("buff", "skill", sp.Name)
			if err := f.sleep(ctx, ms(f.tune().CastGapMs)+ms(400)); err != nil {
				return err
			}
			break
		}
	}
	return nil
}

// ---- casting --------------------------------------------------------------------------------------------

// holdClicker is implemented by input.Game: press the button, keep it down for d, release.
type holdClicker interface {
	HoldClick(b input.Button, x, y int, d time.Duration) error
}

func (f *Fighter) cast(snap *d2.Snapshot, sp SkillSpec, at input.Point) error {
	var err error
	left := sp.Hand == HandLeft || sp.ID == meleeSpec().ID
	switch {
	case left:
		s := screenOf(snap, at)
		if hc, ok := f.e.In.(holdClicker); ok && sp.Hold {
			err = hc.HoldClick(input.Left, s.X, s.Y, f.holdFor(snap.Player))
		} else {
			err = f.e.In.Click(input.Left, s.X, s.Y)
		}
	case sp.Key != "":
		err = f.e.In.CastAt(sp.Key, at)
	case snap.Player.RightSkill == sp.ID:
		s := screenOf(snap, at)
		err = f.e.In.Click(input.Right, s.X, s.Y)
	default:
		return ErrNoHotkey
	}
	f.lastCast = f.now()
	f.lastCastSkill[sp.ID] = f.lastCast
	f.noteCast(sp.ID, f.lastCast)
	return err
}

// holdFor is how long a hold skill keeps the left button down per tick.
func (f *Fighter) holdFor(p *d2.Player) time.Duration {
	if m := f.profileFor(p).Melee; m != nil && m.HoldMs > 0 {
		return ms(m.HoldMs)
	}
	return 600 * time.Millisecond
}

// profileFor is the combat profile of the character (classdata.go).
func (f *Fighter) profileFor(p *d2.Player) *Profile { return SelectProfile(p, f.cat, f.Build) }

// attackSkills lists the usable attack skills in preference order. A right-hand skill must be bound to a key or
// already selected; a left-hand skill needs the left popup (SetListRows) or to be selected already, and a left skill
// that is not selected costs the profile's mana reserve on top, so the choice does not flip at the cost line.
func (f *Fighter) attackSkills(p *d2.Player) []SkillSpec {
	known := KnownSkills(p, f.cat, f.keys)
	pr := f.profileFor(p)
	reserve := 0
	if pr.Melee != nil {
		reserve = pr.Melee.ManaReserve
	}
	var out []SkillSpec
	for _, name := range pr.AttackNames(p.Level, func(n string) bool { _, ok := known[n]; return ok }) {
		sp, ok := known[name]
		if !ok {
			continue
		}
		if sp.Hand == HandLeft {
			if p.LeftSkill != sp.ID {
				if len(f.listRows) == 0 || f.now().Before(f.leftFail[sp.ID]) {
					continue
				}
				if sp.Cost > 0 {
					sp.Cost += reserve
				}
			}
		} else if sp.Key == "" && p.RightSkill != sp.ID {
			continue
		}
		out = append(out, sp)
	}
	return f.gearFilter(p, out)
}

func (f *Fighter) skillReady(sp SkillSpec) bool {
	if sp.Delay == 0 {
		return true
	}
	at, ok := f.lastCastSkill[sp.ID]
	return !ok || f.now().Sub(at) >= sp.Delay
}

// refineResists replaces the monstats resistances of a target with the live stat list (curses, difficulty).
func (f *Fighter) refineResists(t *Target, diff d2.Difficulty) {
	now := f.now()
	if c, ok := f.resCache[t.Unit.ID]; ok && now.Sub(c.at) < 1500*time.Millisecond {
		if c.ok {
			t.Res, t.HasRes = c.res, true
		}
		return
	}
	e := resEntry{at: now}
	if f.e.Mem != nil && f.e.Read != nil {
		if sl, err := readU32(f.e, t.Unit.Addr+0x5C); err == nil && sl != 0 {
			if st, err := f.e.Read.ReadStatList(sl); err == nil && len(st) > 0 {
				e.res, e.ok = ResistsFromStats(st, t.Mon, diff), true
			}
		}
	}
	f.resCache[t.Unit.ID] = e
	if e.ok {
		t.Res, t.HasRes = e.res, true
	}
}

func readU32(e *env.Env, addr uint32) (uint32, error) {
	var b [4]byte
	if err := e.Mem.ReadAt(b[:], addr); err != nil {
		return 0, err
	}
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24, nil
}

func staticFloor(t env.CombatTuning, d d2.Difficulty) int {
	switch d {
	case d2.Hell:
		return t.StaticFloorHell
	case d2.Nightmare:
		return t.StaticFloorNM
	}
	return t.StaticFloorNorm
}

// drinkSkipReason says why no healing potion is being drunk although life is under the drink line in a fight ("" when
// one is, or when there is nothing to explain). The potion bug triage (docs/potion-bug.md) needs it: the soak log had
// "none drunk" deaths with no hint of which rule held the drink back.
func drinkSkipReason(in SafetyInput, a SafetyAction) string {
	if in.InTown || !in.InFight || in.Max <= 0 || in.Life <= 0 || pct(in.Life, in.Max) >= in.Tune.DrinkLifePct {
		return ""
	}
	switch a {
	case DrinkLife, DrinkRejuv:
		return ""
	case SafetyNone:
		switch {
		case in.Belt.Count(PotionHealing)+in.Belt.Count(PotionRejuv) == 0:
			return "no healing potion in the belt"
		case !in.HealReady && !in.RejuvReady:
			return "potion cooldown"
		}
		return "no rule asks for a drink"
	}
	return "chose " + a.String() + " first"
}

func (b Belt) String() string {
	return fmt.Sprintf("hp=%d mp=%d rv=%d", b.Count(PotionHealing), b.Count(PotionMana), b.Count(PotionRejuv))
}

// noteDrinkSkip logs a skipped drink, at most once per reason and 2 s.
func (f *Fighter) noteDrinkSkip(why string, in SafetyInput) {
	now := f.now()
	f.diagMu.Lock()
	if why == f.skipWhy && now.Sub(f.skipAt) < 2*time.Second {
		f.diagMu.Unlock()
		return
	}
	f.skipWhy, f.skipAt = why, now
	f.diagMu.Unlock()
	f.log().Info(fmt.Sprintf("potion: not drinking because %s life=%d%% belt=%s", why, pct(in.Life, in.Max), in.Belt))
}

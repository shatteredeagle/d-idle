package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2denv"
	"d2lod-bot/internal/d2dinput"
	"d2lod-bot/internal/d2dmerc"
	"d2lod-bot/internal/d2dsrc"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/merc"
	"d2lod-bot/internal/nav"
	"d2lod-bot/internal/task"
)

// Merc wiring for the d2d bot loop (docs/merc-wire.md). Everything here is behind the knob merc.enabled (default 0):
// with it off no seam is wrapped, task.MercAlive stays nil and the bot behaves byte for byte as before.
//
// How it hooks in without touching d2dbackend.go: init() wraps d2dSeams.Town, so the wrapper sees whatever town
// package is installed there (by an init() in an earlier-sorted file or the var initialiser; a later-sorted init()
// that overwrites d2dSeams.Town would drop the wrapper: call installMercSeam(&d2dSeams) after it then).

const (
	mercStateTTL  = 5 * time.Second // how long a read of the merc is reused (each read is a full d2d snapshot)
	mercLineEvery = time.Minute
)

// Claimer takes the free Rogue from Kashya. d2dmerc.Claimer implements it (V-d2d). granted false with a nil error:
// a merc was already owned, the reward is gone and nothing more is to be done.
type Claimer interface {
	ClaimFreeRogue(ctx context.Context) (granted bool, err error)
}

// Reviver revives the dead merc at the hireling NPC and returns the gold paid. d2dmerc.Reviver implements it (U).
type Reviver interface {
	Revive(ctx context.Context) (paid int, err error)
}

// Hirer drives the Act 2 hire window. No d2d implementation exists yet (the rows are only readable on screen, docs/d2d-probes.md
// section 4), so production wiring leaves it nil and the hire step does nothing; tests use a fake.
type Hirer interface {
	// Window opens the hireling's window and reads its rows (name, level, aura, price). The window stays open.
	Window(ctx context.Context) ([]merc.HireOffer, error)
	// Hire clicks a row and returns the gold actually paid.
	Hire(ctx context.Context, row int) (paid int, err error)
	Unequip(ctx context.Context, slot merc.Slot) error
	Close(ctx context.Context) error
}

const (
	areaLutGholein  = 40  // levels.txt id (V)
	hireQuoteNormal = 400 // Normal Desert Mercenary quote used only to size the gold farm (V docs/town.md: 360 at merc level 9)
)

// latestMercState is combat.Fighter.MercState: the state of the running wiring, an error while there is none.
func latestMercState() (merc.MercState, error) {
	if m := latestMerc.Load(); m != nil {
		return m.State()
	}
	return merc.MercState{}, errors.New("merc wiring not running")
}

// latestMerc is the wiring of the running bot, for the API mount. nil: merc policy not running.
var latestMerc atomic.Pointer[mercWire]

// mercMount is the handler for inst.Mounts["merc"] (/api/v1/instances/{id}/merc/): the merc state as JSON.
// {"enabled":false} when the knob is off. Add `"merc": mercMount()` to the Mounts map of runD2D.
func mercMount() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if m := latestMerc.Load(); m != nil {
			json.NewEncoder(w).Encode(m.Status())
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"enabled": false})
	})
}

// installMercSeam wraps s.Town. The wrapper decides at construction (knob read from the env) whether to do anything.
func installMercSeam(s *d2denv.Seams) {
	if f := s.Fighter; f != nil {
		s.Fighter = func(e *env.Env, nv *nav.Navigator) bot.Fighter {
			ft := f(e, nv)
			if a, ok := ft.(fighterAdapter); ok && a.f != nil && e.Tune != nil && e.Tune.Merc.Enabled != 0 {
				a.f.MercState = latestMercState // reads the wiring the Town seam builds, whichever seam runs first
			}
			return ft
		}
	}
	inner := s.Town
	s.Town = func(e *env.Env, nv *nav.Navigator) bot.Townsman {
		var t bot.Townsman
		if inner != nil {
			t = inner(e, nv)
		} else {
			t = d2denv.NotYetTown(e.Log)
		}
		if e.Tune == nil || e.Tune.Merc.Enabled == 0 {
			return t
		}
		w := newMercWireD2D(e)
		if w == nil {
			return t
		}
		return &mercTown{Townsman: t, w: w}
	}
}

// newMercWireD2D builds the wiring over a d2d env; nil when e is not a d2d env.
func newMercWireD2D(e *env.Env) *mercWire {
	src, ok := e.Read.(interface {
		Read(context.Context) (*d2dsrc.Result, error)
	})
	if !ok {
		return nil
	}
	rd := &d2dmerc.Reader{Read: src.Read, LevelEstimate: levelEstimate}
	var rev Reviver
	if c, ok := e.In.(*d2dinput.Controller); ok && c.C != nil {
		rev = &d2dmerc.Reviver{C: c.C}
	}
	var cat bot.Catalog
	if e.Data != nil {
		cat = bot.NewDataCatalog(e.Data)
	}
	w := newMercWire(e, rd, rev, cat)
	task.MercAlive = w.AliveHook()
	bot.MercGoldFarm = w.GoldFarm
	bot.MercWait = w.WaitForMerc
	latestMerc.Store(w)
	e.Log.Info("merc policy on (merc.enabled=1)", "reviver", rev != nil)
	return w
}

// levelEstimate: the d2d dump has no merc level. U: take the player's level, an overestimate for a merc hired at
// clvl-5+rand(5) that gained little XP; it only makes the revive gold bar higher (level^2/2 x 15).
func levelEstimate(s *d2.Snapshot) int {
	if s == nil || s.Player == nil {
		return 0
	}
	return max(s.Player.Level, 1)
}

// mercWire is the policy state of one bot.
type mercWire struct {
	e     *env.Env
	log   interface{ Info(string, ...any) }
	knobs merc.Knobs
	rd    merc.Reader
	rev   Reviver
	hirer Hirer
	cat   bot.Catalog
	now   func() time.Time

	mu       sync.Mutex
	cur      merc.MercState
	curErr   error
	curAt    time.Time
	seen     bool // a merc was owned at some read
	wasAlive bool
	deaths   []time.Time
	revives  []time.Time
	lastLine time.Time
	lastDec  string

	lutDone   map[lutKey]bool          // the swap decision of the first Lut Gholein visit is final (hired, or declined for good)
	statePath string                   // where lutDone is kept across restarts ("" = memory only); mercStatePath at construction
	loaded    map[string]bool          // characters whose entry was read from statePath
	winLogged map[merc.Difficulty]bool // `merc: hire window` is written once per difficulty
	farmOn    bool
}

func newMercWire(e *env.Env, rd merc.Reader, rev Reviver, cat bot.Catalog) *mercWire {
	return &mercWire{e: e, log: e.Log, knobs: merc.KnobsFrom(e.Tune.Merc), rd: rd, rev: rev, cat: cat, now: time.Now,
		lutDone: map[lutKey]bool{}, loaded: map[string]bool{}, statePath: mercStatePath, winLogged: map[merc.Difficulty]bool{}}
}

// State is the merc state of at most mercStateTTL ago. A death while the player is away from town is logged once.
func (w *mercWire) State() (merc.MercState, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.curAt.IsZero() && w.now().Sub(w.curAt) < mercStateTTL {
		return w.cur, w.curErr
	}
	st, err := w.rd.State()
	w.cur, w.curErr, w.curAt = st, err, w.now()
	if err == nil && st.Known {
		w.observe(st)
	}
	return st, err
}

// observe tracks alive -> dead transitions for the deaths-per-hour input. Caller holds mu.
func (w *mercWire) observe(st merc.MercState) {
	switch {
	case st.Owned && st.Alive:
		w.wasAlive, w.seen = true, true
	case st.Owned && !st.Alive:
		if w.wasAlive {
			w.deaths = append(w.deaths, w.now())
			w.log.Info(merc.Prefix + "died")
		}
		w.wasAlive, w.seen = false, true
	default:
		w.wasAlive = false
	}
}

// AliveHook is task.MercAlive: guards read it; an unemployed merc counts as unknown (merc.AliveHook).
func (w *mercWire) AliveHook() func() (bool, bool) {
	return merc.AliveHook(readerFunc(w.State))
}

type readerFunc func() (merc.MercState, error)

func (f readerFunc) State() (merc.MercState, error)    { return f() }
func (f readerFunc) Offers() ([]merc.HireOffer, error) { return nil, d2dmerc.ErrNoOffers }

func within(ts []time.Time, now time.Time) (kept []time.Time) {
	for _, t := range ts {
		if now.Sub(t) < time.Hour {
			kept = append(kept, t)
		}
	}
	return kept
}

// Status is the dashboard/API body.
func (w *mercWire) Status() map[string]any {
	st, err := w.State()
	w.mu.Lock()
	defer w.mu.Unlock()
	now := w.now()
	w.deaths, w.revives = within(w.deaths, now), within(w.revives, now)
	out := map[string]any{"enabled": true, "known": st.Known, "owned": st.Owned, "alive": st.Alive, "class": st.Class,
		"lifePct": st.LifePct, "level": st.Level, "deathsLastHour": len(w.deaths), "revivesLastHour": len(w.revives),
		"lastDecision": w.lastDec}
	if err != nil {
		out["error"] = err.Error()
	}
	return out
}

// Tick writes the `merc: seen ...` line once a minute while a merc is owned.
func (w *mercWire) Tick() {
	st, err := w.State()
	if err != nil || !st.Known || !st.Owned {
		return
	}
	w.mu.Lock()
	due := w.now().Sub(w.lastLine) >= mercLineEvery
	if due {
		w.lastLine = w.now()
	}
	w.mu.Unlock()
	if due {
		w.log.Info(merc.LogSeen(st))
	}
}

// kit reports a town portal and a healing or rejuvenation potion in belt or pack. Without a catalog it reports
// neither (fail closed: the revive then waits for "kit").
func (w *mercWire) kit(s *d2.Snapshot) (tp, potions bool) {
	if w.cat == nil || s == nil || s.Inventory == nil {
		return false, false
	}
	for i := range s.Inventory.Items {
		u := &s.Inventory.Items[i]
		if u.Item == nil || (u.Item.Where != d2.ContBelt && u.Item.Where != d2.ContInventory) {
			continue
		}
		m, ok := w.cat.Item(u.Class)
		if !ok {
			continue
		}
		if strings.Contains(m.Name, "Town Portal") {
			tp = true
		}
		if m.Potion == bot.PotionHealing || m.Potion == bot.PotionRejuv {
			potions = true
		}
	}
	return tp, potions
}

func mercCtx(w *mercWire, s *d2.Snapshot) merc.Ctx {
	now := w.now()
	w.mu.Lock()
	w.deaths, w.revives = within(w.deaths, now), within(w.revives, now)
	c := merc.Ctx{RevivesLastHour: len(w.revives), DeathsLastHour: len(w.deaths), Knobs: w.knobs}
	w.mu.Unlock()
	c.Gold, c.Difficulty = s.Player.Gold+s.Player.StashGold, merc.Difficulty(s.Player.Difficulty)
	c.Act = s.Player.Act + 1
	c.PlayerClass = s.Player.Class.String()
	c.HaveTP, c.HavePotions = w.kit(s)
	c.Kit = kitRefill(w.e)
	return c
}

// kitRefill is the full kit refill in gold from the resource knobs (merc.KitRefill).
func kitRefill(e *env.Env) int {
	rt := e.Tune.Resource
	return merc.KitRefill(rt.KitTPGold, rt.KitPotionGold, rt.KitMinHP, rt.KitMinMP)
}

// GoldFarm is bot.MercGoldFarm: true while the merc is dead or missing, gold cannot pay the next merc action and the
// character is below merc.goldFarmMaxLevel. The price is the formula estimate (pre-check only) or the Normal hire quote.
// Logs `merc: gold farm on|off` once per change.
func (w *mercWire) GoldFarm(s *d2.Snapshot) bool {
	if s == nil || s.Player == nil {
		return false
	}
	st, err := w.State()
	if err != nil || !st.Known {
		return false
	}
	c := mercCtx(w, s)
	missing, price := !st.Owned, hireQuoteNormal
	if st.Owned && !st.Alive {
		missing, price = true, merc.ReviveCost(max(st.Level, 1))
	}
	need := merc.GoldNeed(c, price)
	on := missing && merc.GoldFarm(true, c.Gold, need, s.Player.Level, w.knobs)
	w.mu.Lock()
	changed := on != w.farmOn
	w.farmOn = on
	w.mu.Unlock()
	if changed {
		w.log.Info(merc.LogGoldFarm(on, need, c.Gold, s.Player.Level))
	}
	return on
}

// lutKey is one first-visit decision: per character and difficulty.
type lutKey struct {
	name string
	d    merc.Difficulty
}

// mercStatePath is the file of the first-visit flags, next to the other bot state of the instance (the counters file
// follows the same one-small-JSON-per-instance pattern, internal/bot/counters.go). Set by runD2D; "" keeps them in memory.
var mercStatePath string

// mercFile maps a character name to the difficulties whose Lut Gholein decision is final.
type mercFile struct {
	LutDone map[string][]int `json:"lutDone"`
}

func (w *mercWire) readFile() mercFile {
	var f mercFile
	if b, err := os.ReadFile(w.statePath); err == nil {
		json.Unmarshal(b, &f)
	}
	if f.LutDone == nil {
		f.LutDone = map[string][]int{}
	}
	return f
}

// lutIsDone reports the first-visit decision for the character, reading the file once per character per run.
func (w *mercWire) lutIsDone(name string, d merc.Difficulty) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.statePath != "" && name != "" && !w.loaded[name] {
		w.loaded[name] = true
		for _, di := range w.readFile().LutDone[name] {
			w.lutDone[lutKey{name, merc.Difficulty(di)}] = true
		}
	}
	return w.lutDone[lutKey{name, d}]
}

// lutMark makes the decision final and writes it (a failed write is logged and costs only a repeat visit after a restart).
func (w *mercWire) lutMark(name string, d merc.Difficulty) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.lutDone[lutKey{name, d}] = true
	if w.statePath == "" || name == "" {
		return
	}
	f := w.readFile()
	for _, di := range f.LutDone[name] {
		if merc.Difficulty(di) == d {
			return
		}
	}
	f.LutDone[name] = append(f.LutDone[name], int(d))
	b, _ := json.Marshal(f)
	tmp := w.statePath + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err == nil {
		err = os.Rename(tmp, w.statePath)
		if err != nil {
			w.log.Info(merc.Prefix + "first-visit flag not saved (" + err.Error() + ")")
		}
	} else {
		w.log.Info(merc.Prefix + "first-visit flag not saved (" + err.Error() + ")")
	}
}

// HireStep is the Act 2 town step: on the first Lut Gholein visit per difficulty read the hire window (one
// `merc: hire window` line per row), and swap to the best Desert Mercenary the policy allows: strip the Rogue's gear
// first (the hire destroys it), pay the read price. A decline for gold or an unreadable window keeps the visit open;
// any other decision is final. "" = nothing to do (no hirer, not in Lut Gholein, merc state unknown).
func (w *mercWire) HireStep(ctx context.Context, s *d2.Snapshot) string {
	if w.hirer == nil || s == nil || s.Player == nil || !s.Player.InTown() || s.Player.Area != areaLutGholein {
		return ""
	}
	st, err := w.State()
	if err != nil || !st.Known {
		return ""
	}
	c := mercCtx(w, s)
	name := s.Player.Name
	if w.lutIsDone(name, c.Difficulty) {
		return ""
	}
	c.FirstLutVisit = true
	offers, err := w.hirer.Window(ctx)
	if err != nil {
		w.log.Info(merc.Prefix + "hire window unreadable (" + err.Error() + ")")
		return "merc hire skipped: window unreadable"
	}
	defer w.hirer.Close(ctx)
	w.mu.Lock()
	first := !w.winLogged[c.Difficulty]
	w.winLogged[c.Difficulty] = true
	w.mu.Unlock()
	if first {
		for _, o := range offers {
			w.log.Info(merc.LogHireWindow(o, c.Gold, s.Player.Level))
		}
	}
	prefs := merc.AuraPrefs(c.PlayerClass, c.Difficulty, c.BowOrJavelin)
	// Wait for the best aura in the window: a cheaper acceptable aura is not taken while the better one is unaffordable.
	top := 1 << 30
	for _, o := range offers {
		if o.Class == merc.ClassDesert {
			if r := merc.AuraRank(prefs, o.Aura); r >= 0 && r < top {
				top = r
			}
		}
	}
	best, bestRank, why := -1, 1<<30, ""
	var bestDec merc.Decision
	for i, o := range offers {
		var dec merc.Decision
		if st.Owned {
			dec = merc.ShouldReplace(st, o, c)
		} else {
			dec = merc.ShouldHire(o, st, c)
		}
		if dec.Do && merc.AuraRank(prefs, o.Aura) > top {
			dec = merc.Decision{Reason: merc.ReasonGold} // the better aura is in the window: wait for the gold
		}
		if !dec.Do {
			if dec.Reason == merc.ReasonGold || why == "" {
				why = dec.Reason
			}
			continue
		}
		if r := merc.AuraRank(prefs, o.Aura); r < bestRank {
			best, bestRank, bestDec = i, r, dec
		}
	}
	if why == "" {
		why = merc.ReasonNoBetter
	}
	if best < 0 {
		w.log.Info(merc.Prefix + "hire skipped (" + why + ")")
		if why != merc.ReasonGold {
			w.lutMark(name, c.Difficulty)
		}
		return "merc hire skipped: " + why
	}
	if bestDec.StripFirst {
		plan := merc.PlanStrip(st, st.FreeCells)
		if plan.StashCells > 0 {
			w.log.Info(merc.Prefix + "hire skipped (strip needs stash space)")
			return "merc hire skipped: pack full"
		}
		for _, step := range plan.Steps {
			if err := w.hirer.Unequip(ctx, step.Slot); err != nil {
				w.log.Info(merc.Prefix + "hire skipped (strip failed: " + err.Error() + ")")
				return "merc hire failed"
			}
			w.log.Info(merc.LogStrip(step.Slot, step.Item.Name, step.Item.Cells()))
		}
	}
	o := offers[best]
	paid, err := w.hirer.Hire(ctx, o.Row)
	if err != nil {
		w.log.Info(merc.Prefix + "hire failed (" + err.Error() + ")")
		return "merc hire failed"
	}
	w.lutMark(name, c.Difficulty)
	w.mu.Lock()
	w.curAt = time.Time{}
	w.mu.Unlock()
	w.log.Info(merc.LogHired(o.Class, o.Aura, paid, c.Gold, c.Gold-paid))
	return "merc hired"
}

// Waiting for the merc after a portal or waypoint (merc.ShouldWaitForMerc).
const (
	mercWaitMax  = 6 * time.Second // the longest wait for her to catch up
	mercWaitStep = 500 * time.Millisecond
	mercNearDist = 12 // subtiles: nearer than this counts as arrived (U: judgment; the unit table lists her within the view)
)

// WaitForMerc is bot.MercWait: after a town portal (why "portal") or a waypoint, hold still until a living merc is near
// or mercWaitMax has passed, then go on. Bounded; nothing for a missing, dead or unknown merc. Writes
// `merc: waiting (<why>)` when it waits at all and `merc: wait over ...` when it ends.
func (w *mercWire) WaitForMerc(ctx context.Context, why string) {
	waited := 0
	maxMs := int(mercWaitMax / time.Millisecond)
	logged := false
	for {
		st, err := w.rd.State() // fresh: the cached State could be 5 s old
		if err != nil {
			return
		}
		s, serr := w.e.Snapshot()
		if serr != nil || s == nil || s.Player == nil || s.Player.InTown() {
			return
		}
		m := d2.FindMerc(s)
		near := m.Alive && max(abs(m.X-s.Player.X), abs(m.Y-s.Player.Y)) <= mercNearDist
		if !merc.ShouldWaitForMerc(st, near, waited, maxMs) {
			if logged {
				w.log.Info(merc.Prefix + "wait over waited_ms=" + strconv.Itoa(waited) + " near=" + strconv.FormatBool(near))
			}
			return
		}
		if !logged {
			logged = true
			w.log.Info(merc.Prefix + "waiting for the merc why=" + why)
		}
		w.e.Clock.Sleep(mercWaitStep)
		waited += int(mercWaitStep / time.Millisecond)
		if ctx.Err() != nil {
			return
		}
	}
}

// ReviveStep is the town-visit step: if the merc is known dead and the policy says so, revive. Returns what happened
// as one short string for the chore report ("" nothing to do). Revive is U on every game (docs/merc-wire.md): a
// failed attempt is logged and costs nothing but time; the revive cap counts attempts that paid.
func (w *mercWire) ReviveStep(ctx context.Context, s *d2.Snapshot) string {
	if s == nil || s.Player == nil || !s.Player.InTown() {
		return ""
	}
	st, err := w.State()
	if err != nil || !st.Known || !st.Owned || st.Alive {
		return ""
	}
	c := mercCtx(w, s)
	dec, price := merc.ShouldRevive(st, c, 0)
	w.mu.Lock()
	w.lastDec = dec.Reason
	w.mu.Unlock()
	if !dec.Do {
		w.log.Info(merc.LogReviveSkipped(dec.Reason, price, c.Gold))
		return "merc revive skipped: " + dec.Reason
	}
	if w.rev == nil {
		w.log.Info(merc.LogReviveSkipped("no reviver", price, c.Gold))
		return "merc revive skipped: no reviver"
	}
	est := price
	paid, err := w.rev.Revive(ctx)
	if err != nil {
		w.log.Info(merc.LogReviveSkipped("failed: "+err.Error(), price, c.Gold))
		return "merc revive failed"
	}
	w.mu.Lock()
	w.revives = append(w.revives, w.now())
	w.curAt = time.Time{} // re-read: the merc should be alive now
	w.mu.Unlock()
	if q, ok := w.rev.(interface{ Quote() int }); ok && q.Quote() > 0 {
		est = q.Quote() // the exact quoted price replaces the level estimate in the log
	}
	w.log.Info(merc.LogRevived(paid, st.Level, est))
	return "merc revived"
}

// mercTown adds the merc step to the Townsman's chores; everything else passes through.
type mercTown struct {
	bot.Townsman
	w *mercWire
}

func (t *mercTown) Chores(ctx context.Context, plan bot.ChorePlan) (bot.ChoreReport, error) {
	rep, err := t.Townsman.Chores(ctx, plan)
	if s, serr := t.w.e.Snapshot(); serr == nil {
		if what := t.w.HireStep(ctx, s); what != "" {
			rep.Done = append(rep.Done, what)
		}
		if what := t.w.ReviveStep(ctx, s); what != "" {
			rep.Done = append(rep.Done, what)
		}
	}
	return rep, err
}

// NeedsTrip is polled by the runner; it doubles as the cheap tick for the once-a-minute merc line.
func (t *mercTown) NeedsTrip(s *d2.Snapshot) (bool, string) {
	t.w.Tick()
	return t.Townsman.NeedsTrip(s)
}

// claimVerify bounds the read-back after a claim: claimTries reads, claimPause apart. A var so tests do not sleep.
var (
	claimTries = 4
	claimPause = 500 * time.Millisecond
)

// claimFailLimit is how many claims may end unseen before the step stops retrying (the reward is spent at the first
// grant: a Rogue that died at once cannot be claimed again, only revived or hired).
const claimFailLimit = 3

// claimDep builds task deps ClaimMerc over a Claimer: the policy check, the claim, the read-back, the log line. A nil
// claimer answers task.ErrClaimUnsupported (the step logs and moves on). The claim step already requires Blood Raven
// dead and Normal difficulty (task.ClaimNeeded); ShouldClaimFreeRogue re-checks them from the reader. After a grant
// the merc is read back from the dump (merc.present): `merc: claim verified` or `merc: claim not seen after N tries`
// (an error, so the step retries, at most claimFailLimit times; Gate 2 logged "claimed" and the merc was gone 5 s later).
func claimDep(c Claimer, rd merc.Reader, diff func() merc.Difficulty, log interface{ Info(string, ...any) }) func(context.Context) error {
	unseen := 0
	return func(ctx context.Context) error {
		if c == nil {
			return task.ErrClaimUnsupported
		}
		var st merc.MercState
		if rd != nil {
			st, _ = rd.State()
		}
		d := merc.Normal
		if diff != nil {
			d = diff()
		}
		if dec := merc.ShouldClaimFreeRogue(st, merc.FreeRogueCtx{AtKashya: true, Difficulty: d, BloodRavenDone: true}); !dec.Do {
			log.Info(merc.Prefix + "claim skipped (" + dec.Reason + ")")
			return nil
		}
		granted, err := c.ClaimFreeRogue(ctx)
		if err != nil {
			return err
		}
		if !granted {
			log.Info(merc.Prefix + "claim: nothing granted (merc already owned)")
			return nil
		}
		log.Info(merc.Prefix + "claimed free rogue")
		if rd == nil {
			return nil
		}
		for i := 0; i < claimTries; i++ {
			if i > 0 {
				if err := sleepCtx(ctx, claimPause); err != nil {
					return err
				}
			}
			if s, err := rd.State(); err == nil && s.Known && s.Owned {
				log.Info(merc.Prefix + "claim verified")
				return nil
			}
		}
		log.Info(merc.Prefix + "claim not seen after " + strconv.Itoa(claimTries) + " tries")
		if unseen++; unseen >= claimFailLimit {
			return nil
		}
		return errClaimUnseen
	}
}

var errClaimUnseen = errors.New("merc: claimed but not seen in the dump")

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// guardMercHook is task.MercAlive for runs with the merc policy off: the route guards then read the dump (merc.present),
// not only the unit table, where the Rogue shows up only while it is near the player. Root cause of Gate 2: with
// merc.enabled=0 the hook stayed nil, d2.FindMerc found the Rogue once in town and never again (5 s later the character
// stood in the Burial Grounds), so every guard read "merc=none" for 8 h. A read error or a missing dump section is
// "unknown" and leaves the unit table as the only source. Reads are reused for mercStateTTL (each is a full snapshot).
func guardMercHook(rd merc.Reader, now func() time.Time) func() (bool, bool) {
	var mu sync.Mutex
	var at time.Time
	var alive, known bool
	return func() (bool, bool) {
		mu.Lock()
		defer mu.Unlock()
		if !at.IsZero() && now().Sub(at) < mercStateTTL {
			return alive, known
		}
		alive, known = merc.AliveHook(rd)()
		at = now()
		return alive, known
	}
}

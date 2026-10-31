package main

import (
	"context"
	"encoding/json"
	"net/http"
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
}

func newMercWire(e *env.Env, rd merc.Reader, rev Reviver, cat bot.Catalog) *mercWire {
	return &mercWire{e: e, log: e.Log, knobs: merc.KnobsFrom(e.Tune.Merc), rd: rd, rev: rev, cat: cat, now: time.Now}
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
	return c
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
	paid, err := w.rev.Revive(ctx)
	if err != nil {
		w.log.Info(merc.LogReviveSkipped("failed: "+err.Error(), price, c.Gold))
		return "merc revive failed"
	}
	w.mu.Lock()
	w.revives = append(w.revives, w.now())
	w.curAt = time.Time{} // re-read: the merc should be alive now
	w.mu.Unlock()
	w.log.Info(merc.LogRevived(paid, st.Level))
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

// claimDep builds task deps ClaimMerc over a Claimer: the policy check, the claim, the log line. A nil claimer
// answers task.ErrClaimUnsupported (the step logs and moves on). The claim step already requires Blood Raven dead and
// Normal difficulty (task.ClaimNeeded); ShouldClaimFreeRogue re-checks them from the reader.
func claimDep(c Claimer, rd merc.Reader, diff func() merc.Difficulty, log interface{ Info(string, ...any) }) func(context.Context) error {
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
		if granted {
			log.Info(merc.Prefix + "claimed free rogue")
		} else {
			log.Info(merc.Prefix + "claim: nothing granted (merc already owned)")
		}
		return nil
	}
}

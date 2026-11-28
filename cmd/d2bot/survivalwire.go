package main

import (
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/danger"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/power"
	"d2lod-bot/internal/questboard"
	"d2lod-bot/internal/survival"
)

// Survival slice S-C (docs/survival/ledger.md): the per-zone ledger learns from the run, the controller names the zone
// it would farm and why, and a words-only block is served for the UI. SHADOW ONLY: nothing here changes what the bot does.
const (
	shadowTravelMin   = 2.0 // U: minutes of travel assumed for every candidate zone but the current one
	shadowKitPotions  = 4   // the kit rule's healing potions (docs/bot kit), U for this use
	shadowReviveQuote = 400 // U: the real quote is read in town; swap7 paid 360 in Normal
)

// survFight is the hook the fighter's OnSummary chain calls (senseshistory.go); nil until the shadow is wired.
var survFight atomic.Pointer[func(combat.FightSummary)]

// survGlance is what /api/survival serves; nil until the first decision.
var survGlance atomic.Pointer[survival.Glance]

// survivalGlance is the watch.SurvivalSource of the bot's hub.
func survivalGlance() any {
	if g := survGlance.Load(); g != nil {
		return g
	}
	return nil
}

type predKey = survival.ZoneKey

type shadowPred struct{ takenPerKill, secPerKill float64 }

type survivalShadow struct {
	mu       sync.Mutex
	db       *data.DB
	m        *power.Model
	tab      *danger.Table
	p        *survival.Params
	led      *survival.Ledger
	path     string
	power    float64
	lastDiff string
	pred     map[predKey]shadowPred
	warned   map[string]bool
	now      func() time.Time // the bot clock: virtual (Unix epoch plus game time) under lockstep, wall otherwise
}

// wireSurvival returns the ZoneChoice and ZoneObserve hooks; both nil without game tables or a loadable model. A ledger
// file that cannot be read starts empty (and a file from a newer build is left alone: saves are skipped).
func wireSurvival(db *data.DB, ledgerPath string, now func() time.Time) (func(*d2.Snapshot, int, string, int, []bot.ZoneCand) string, func(bot.ZoneEvent)) {
	w := newSurvivalShadow(db, ledgerPath, now)
	if w == nil {
		return nil, nil
	}
	return w.choice, w.observe
}

// wireSurvivalModel is wireSurvival plus the S-D hooks: the zone decision (bot.zoneModel 1) and the boss margin. All nil
// without game tables or a loadable model.
func wireSurvivalModel(db *data.DB, ledgerPath string, now func() time.Time) (choice func(*d2.Snapshot, int, string, int, []bot.ZoneCand) string, observe func(bot.ZoneEvent),
	decide func(*d2.Snapshot, int, string, int, string, []bot.ZoneCand) (bot.ZoneVerdict, error), boss func(*d2.Snapshot, string, string, string) (string, bool)) {
	w := newSurvivalShadow(db, ledgerPath, now)
	if w == nil {
		return nil, nil, nil, nil
	}
	return w.choice, w.observe, w.decide, w.bossMargin
}

// newSurvivalShadow: now must be the clock the runner stamps its events with (Env.Clock.Now). Mixing it with time.Now made
// every lockstep event 56 years old to the ledger (the Pacer counts from the Unix epoch), so chickens, deaths and merc
// deaths never counted (docs/survival/ab-screen-analysis-2026-11-16.md). nil means the wall clock.
func newSurvivalShadow(db *data.DB, ledgerPath string, now func() time.Time) *survivalShadow {
	if now == nil {
		now = time.Now
	}
	tab := dangerTable(db)
	if tab == nil {
		return nil
	}
	m, err := power.NewModel(tab.Params)
	if err != nil {
		slog.Warn("survival: no power model", "err", err)
		return nil
	}
	p, err := survival.LoadParams()
	if err != nil {
		slog.Warn("survival: no params", "err", err)
		return nil
	}
	w := &survivalShadow{db: db, m: m, tab: tab, p: p, path: ledgerPath, pred: map[predKey]shadowPred{}, warned: map[string]bool{}, now: now}
	led, _, err := survival.LoadLedger(ledgerPath, p)
	if err != nil {
		slog.Warn("survival: ledger not loaded, starting empty and not saving", "err", err)
		w.path = ""
	}
	w.led = led
	f := w.fight
	survFight.Store(&f)
	survShadow.Store(w)
	return w
}

// chainSurvFight puts the ledger's fight hook on the fighter's OnSummary. It must run before useSensesHistory, which
// chains whatever is there but returns early without a senses rig: the d2d harness has none, so fights never reached
// the ledger and kills and seconds stayed 0 in every zone (rescreen 2026-11-16).
func chainSurvFight(f *combat.Fighter) {
	if f == nil {
		return
	}
	prev := f.OnSummary
	f.OnSummary = func(s combat.FightSummary) {
		if prev != nil {
			prev(s)
		}
		if sf := survFight.Load(); sf != nil {
			(*sf)(s)
		}
	}
}

func (w *survivalShadow) warn(err error) string {
	if k := err.Error(); !w.warned[k] {
		w.warned[k] = true
		slog.Warn("survival: no choice", "err", err)
	}
	return ""
}

func (w *survivalShadow) save() {
	if w.path == "" {
		return
	}
	if err := w.led.Save(w.path); err != nil && !w.warned["save"] {
		w.warned["save"] = true
		slog.Warn("survival: ledger not saved", "err", err)
	}
}

// choice is the bot.Deps.ZoneChoice hook: the shadow choice at the default dial, as log lines. Shadow only.
func (w *survivalShadow) choice(s *d2.Snapshot, area int, diff string, potions int, cands []bot.ZoneCand) string {
	w.mu.Lock()
	defer w.mu.Unlock()
	v, err := w.judge(s, area, diff, potions, "", cands, false)
	if err != nil {
		return w.warn(err)
	}
	return v.Line
}

// decide is the bot.Deps.ZoneDecide hook (slice S-D): the same judgement at the knob's dial, returned to the planner.
// The caller falls back to the plan tables on an error.
func (w *survivalShadow) decide(s *d2.Snapshot, area int, diff string, potions int, dial string, cands []bot.ZoneCand) (bot.ZoneVerdict, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.judge(s, area, diff, potions, dial, cands, true)
}

// bossMargin is the bot.Deps.BossMargin hook: the word of the hero's margin against a boss, ok false without data.
func (w *survivalShadow) bossMargin(s *d2.Snapshot, key, diff, dial string) (string, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if s == nil || s.Player == nil {
		return "", false
	}
	b, err := w.tab.FindBoss(key, diff)
	if err != nil {
		return "", false
	}
	hero, a, err := heroOf(w.m, w.db, s, -1, w.warn)
	if err != nil {
		return "", false
	}
	word, _, err := w.m.BossWord(hero, a, b, dial)
	if err != nil {
		w.warn(err)
		return "", false
	}
	return word, true
}

// judge is one controller decision. dial "" is the profile default (the shadow); model true publishes the glance as the
// deciding block.
func (w *survivalShadow) judge(s *d2.Snapshot, area int, diff string, potions int, dialID string, cands []bot.ZoneCand, model bool) (bot.ZoneVerdict, error) {
	p := s.Player
	hero, a, err := heroOf(w.m, w.db, s, potions, w.warn)
	if err != nil {
		return bot.ZoneVerdict{}, err
	}
	if dialID == "" {
		dialID = w.m.P.DefaultDial
	}
	dial, err := w.m.P.Dial(dialID)
	if err != nil {
		return bot.ZoneVerdict{}, err
	}
	build := func(h power.Hero) ([]survival.Candidate, error) {
		var out []survival.Candidate
		for _, c := range cands {
			z, err := w.tab.Find(c.Area, diff)
			if err != nil {
				continue // an area the profile does not cover is not a candidate
			}
			zm, err := w.m.Zone(h, a, z, dial.ID)
			if err != nil {
				return nil, err
			}
			k := survival.ZoneKey{Area: c.Area, Diff: diff}
			if n := float64(z.Encounters.Typical.N); n > 0 {
				w.pred[k] = shadowPred{zm.Typical.Terms.Taken / n, zm.Typical.Terms.PerKill}
			}
			tm := shadowTravelMin
			if c.Area == area {
				tm = 0
			}
			out = append(out, survival.Candidate{Key: k, Name: z.Name, ZM: zm, Eco: survival.Economy(z, zm, h.Level, w.p), TravelMin: tm, Depleted: c.Depleted, MLvl: z.MLvl, Units: z.Density.Units})
		}
		return out, nil
	}
	cs, err := build(hero)
	if err != nil {
		return bot.ZoneVerdict{}, err
	}
	need := 0
	if potions >= 0 && potions < shadowKitPotions {
		need = (shadowKitPotions - potions) * int(w.p.PotionPrice)
	}
	mercDead := hero.Merc.Hired && !hero.Merc.Alive
	if mercDead {
		need += shadowReviveQuote
	}
	w.power, w.lastDiff = survival.PowerOf(hero), diff
	in := survival.Input{Now: w.now(), Dial: dial, Current: survival.ZoneKey{Area: area, Diff: diff}, Power: w.power,
		Purse: survival.Purse{Gold: p.Gold, Need: need}, Cands: cs, FreshAllowed: true, OffenceUnread: w.m.OffenceUnread(hero),
		Active: model, Archetype: a.ID, MercDead: mercDead, Level: p.Level}
	d, err := w.led.Choose(in)
	if err != nil {
		return bot.ZoneVerdict{}, err
	}
	lines := d.Line()
	v := bot.ZoneVerdict{Area: d.Area, Fresh: d.Fresh, Hole: d.Hole, Hold: d.Hold, Margin: d.Margin, Why: d.Why.Text, K: d.CurK}
	var merc *survival.ReviveDecision
	if mercDead {
		alive := hero
		alive.Merc.Alive = true
		with, err1 := build(alive)
		if err1 == nil {
			rd := w.led.Revive(survival.ReviveInput{Base: in, With: with, Without: cs, Quote: shadowReviveQuote, Reserve: shadowKitPotions * int(w.p.PotionPrice), Gold: p.Gold})
			merc = &rd
			v.MercDead, v.MercRevive, v.MercWhy = true, rd.Revive, rd.Why.Text
			v.MercGain, v.MercCost, v.MercZone = rd.Gain, rd.Cost, rd.Zone
			lines += fmt.Sprintf("\nzone revive: %s: %s (zone %s, gain %.0f, cost %.0f)", rd.Why.Code, rd.Why.Text, rd.Zone, rd.Gain, rd.Cost)
		}
	}
	g := survival.GlanceOf(d, dial.ID, merc)
	g.Model = model
	survGlance.Store(g)
	v.Line = lines
	return v, nil
}

// observe is the bot.Deps.ZoneObserve hook.
func (w *survivalShadow) observe(ev bot.ZoneEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()
	k := survival.ZoneKey{Area: ev.Area, Diff: ev.Diff}
	now := ev.At
	if now.IsZero() {
		now = w.now()
	}
	switch ev.Kind {
	case bot.ZoneEvChicken:
		w.led.Observe(k, now, survival.Event{Kind: survival.EvChicken}, w.power)
	case bot.ZoneEvDeath:
		w.led.Observe(k, now, survival.Event{Kind: survival.EvDeath}, w.power)
	case bot.ZoneEvMercDied:
		w.led.Observe(k, now, survival.Event{Kind: survival.EvMercDeath}, w.power)
	case bot.ZoneEvArea:
		w.led.Observe(k, now, survival.Event{Kind: survival.EvXP, XP: ev.XP, Seconds: ev.Seconds}, w.power)
		if ev.Seconds > 0 {
			w.led.Observe(k, now, survival.Event{Kind: survival.EvIntake, LifeLost: float64(ev.LifeLost), Seconds: ev.Seconds}, w.power)
		}
	default:
		return
	}
	w.save()
}

// fight feeds one finished fight (the fighter's OnSummary) with the model's prediction for the same kills, which is
// what k_zone and k_hero compare against. Without a prediction for the zone the fight adds kills and seconds only.
func (w *survivalShadow) fight(fs combat.FightSummary) {
	w.mu.Lock()
	defer w.mu.Unlock()
	pr := w.pred[survival.ZoneKey{Area: fs.Area, Diff: w.lastDiff}] // zero when the model made no prediction: kills and time still count
	if fs.Kills <= 0 || w.lastDiff == "" {
		return
	}
	n := float64(fs.Kills)
	w.led.Observe(survival.ZoneKey{Area: fs.Area, Diff: w.lastDiff}, w.now(), survival.Event{Kind: survival.EvFight, Kills: n, Seconds: fs.Sec,
		Taken: float64(fs.Damage), PredTaken: pr.takenPerKill * n, PredKillSec: pr.secPerKill * n, Potions: float64(fs.Potions), Starved: float64(fs.Starved)}, w.power)
}

// questMargin answers the quest board's margin query (slice S4): the hero as readiness sees it and, for a boss key, the
// margin against it divided by the ledger's hero correction (a boss has no zone key, so the zone part is 1; U: a boss
// that killed the hero before is not yet remembered per boss). boss "" asks for the hero only. Read only.
func (w *survivalShadow) questMargin(s *d2.Snapshot, boss, diff string, potions int) (questboard.Hero, *questboard.Encounter, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if s == nil || s.Player == nil {
		return questboard.Hero{}, nil, false
	}
	hero, a, err := heroOf(w.m, w.db, s, potions, w.warn)
	if err != nil {
		return questboard.Hero{}, nil, false
	}
	qh := questboard.HeroFrom(hero)
	if boss == "" {
		return qh, nil, true
	}
	b, err := w.tab.FindBoss(boss, diff)
	if err != nil {
		return qh, nil, true
	}
	r, err := w.m.Margin(hero, a, power.BossPack(b))
	if err != nil {
		return qh, nil, true
	}
	k := w.led.KHeroFor(w.m.OffenceUnread(hero))
	enc := questboard.EncounterFrom(b.Name, "boss", r, k)
	return qh, &enc, true
}

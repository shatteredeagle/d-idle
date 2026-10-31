package bot

import (
	"context"
	"fmt"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/resource"
)

// Death handling (PLAN.md "Death is normal"). Death is a first-class state: the observer detects the transition
// (player dead mode or life 0) and records a DeathRecord; the loop then runs Recovering: wait for the respawn, apply
// the death policy (spiral breaker, pause rule), decide on the corpse, rebuild.
//
// diedTransition is the pure Died(prev, cur) rule. COMBAT is to expose its own; when it does, call that here instead.
func diedTransition(prev, cur *d2.Snapshot) bool {
	if cur == nil || cur.Player == nil || !isDead(cur.Player) {
		return false
	}
	return prev == nil || prev.Player == nil || !isDead(prev.Player)
}

const (
	whyRetrievalFailed = "retrieval failed, abandoned"
	whySecondDeath     = "second death during retrieval"
	// gearCorpseAge is how far back an older corpse is still believed to hold the gear (U).
	gearCorpseAge = 2 * time.Hour
)

type recovery struct {
	rec         DeathRecord
	target      time.Time // Time of the record whose corpse is retrieved when it is not rec (the older gear corpse)
	area        int
	pos         input.Point
	at          time.Time
	tries       int
	attempting  bool
	respawned   bool
	policyDone  bool
	decided     bool
	retrieve    bool
	why         string
	goldBefore  int
	equipBefore int
	warned      bool
	escs        int
	lastEsc     time.Time
	nextSample  time.Duration
}

// deathWait runs while the character is dead: it opens the recovery once per death and waits for the respawn.
func (r *Runner) deathWait(c control) bool {
	l := &r.loop
	now := r.e.Clock.Now()
	if l.rec == nil {
		r.openRecovery()
	}
	r.setState(StateRecovering, "Recovering after death")
	if !l.rec.policyDone {
		l.rec.policyDone = true
		r.deathPolicy(now, l.rec)
	}
	r.pressContinue(now, l.rec)
	if now.Sub(l.rec.at) > sec(r.cfg.RecoverTimeoutSec) && !l.rec.warned {
		l.rec.warned = true
		r.setError("The character did not respawn; check the game window", sec(r.cfg.ErrorRetrySec))
	}
	return false
}

// Death overlay (VERIFIED LIVE, docs/combat.md "Death"): Esc is ignored while the player mode is 0 (dying) and respawns the
// character from mode 17 on, so it is pressed only then, once per combat.EscRetryEvery. It is never pressed while the
// character is alive (that opens the game menu). Dead-state samples are logged for fixtures.
func (r *Runner) pressContinue(now time.Time, rec *recovery) {
	since := now.Sub(rec.at)
	if since >= rec.nextSample {
		rec.nextSample += time.Second
		if rec.nextSample > 20*time.Second {
			rec.nextSample = since + 10*time.Second
		}
		if s := r.LastSnapshot(); s != nil {
			var open []int
			for i, v := range s.UI {
				if v != 0 {
					open = append(open, i)
				}
			}
			r.e.Log.Info("dead", "sinceMs", since.Milliseconds(), "mode", s.Player.Mode, "life", s.Player.Life, "area", s.Player.Area,
				"x", s.Player.X, "y", s.Player.Y, "uiFlags", open, "escPresses", rec.escs)
		}
	}
	if s := r.LastSnapshot(); s == nil || !combat.CanRespawn(s.Player) || now.Sub(rec.lastEsc) < combat.EscRetryEvery {
		return
	}
	rec.lastEsc = now
	rec.escs++
	if err := r.e.In.Key("esc"); err != nil {
		r.e.Log.Warn("death overlay: Esc failed", "err", err)
		return
	}
	r.e.Log.Info("death overlay: pressed Esc to continue", "n", rec.escs, "sinceMs", since.Milliseconds())
}

// openRecovery starts the recovery record for the newest death.
func (r *Runner) openRecovery() {
	l := &r.loop
	now := r.e.Clock.Now()
	{
		prevAttempt := l.rec != nil && l.rec.attempting
		r.obMu.Lock()
		var last DeathRecord
		if lr := r.dlog.last(); lr != nil {
			last = *lr
		}
		gold, equip := r.ob.lastDeath.gold, r.ob.lastDeath.equip
		r.obMu.Unlock()
		prevTarget := time.Time{}
		if l.rec != nil {
			prevTarget = l.rec.target
		}
		l.rec = &recovery{rec: last, area: last.Area, pos: last.Pos, at: now, goldBefore: gold, equipBefore: equip}
		if prevAttempt {
			if !prevTarget.IsZero() {
				r.obMu.Lock()
				r.markCorpse(prevTarget, false, whySecondDeath)
				r.obMu.Unlock()
			}
			l.rec.tries = 1 << 20 // a second death on the way to the corpse: abort, never try again
			l.rec.why = whySecondDeath
		}
		l.trip = nil
	}
	if !l.rec.policyDone {
		l.rec.policyDone = true
		r.deathPolicy(now, l.rec)
	}
}

// deathPolicy applies the spiral breaker and the deaths-per-hour pause rule after a death.
func (r *Runner) deathPolicy(now time.Time, rec *recovery) {
	t := r.e.Tune.Bot
	unfavorable := r.deathUnfavorable(rec.area)
	r.obMu.Lock()
	inArea := r.dlog.since(now.Add(-time.Duration(t.SpiralMinutes)*time.Minute), rec.area)
	inAreaHour := r.dlog.since(now.Add(-time.Hour), rec.area)
	lastHour := r.dlog.since(now.Add(-time.Hour), 0)
	// Dying is normal and a few deaths are how a hard goal gets done: the area is only avoided once SpiralDeaths
	// (default 4) deaths fall in the window, one fewer when the area is clearly too high for the level or the
	// character died with nothing to recover with. SpiralDeaths above the default turns the breaker down on purpose.
	fast := t.SpiralDeaths
	if unfavorable {
		fast--
	}
	slow := t.SpiralDeaths * 3 / 2
	spiral := (inArea >= fast || inAreaHour >= slow) && !r.d.Plan.IsTown(rec.area)
	if spiral {
		r.skipAreaLocked(rec.area, r.banLength(), whyDeath)
		r.setReaction(rec.rec.Time, fmt.Sprintf("area banned for %d min (death spiral)", int(r.banLength().Minutes())))
		if rec.rec.Killer != "" && !rec.rec.KillerChampion { // a champion is not its base monster: no weight for the plain kind
			r.dlog.Danger[rec.rec.Killer]++
		}
		if err := r.dlog.save(r.cfg.DeathLogPath); err != nil {
			r.e.Log.Warn("death log not saved", "err", err)
		}
	}
	r.obMu.Unlock()
	if spiral {
		msg := fmt.Sprintf("Death spiral: %d deaths in %d minutes in %s (last killer: %s). Avoiding it for %d minutes.",
			inArea, t.SpiralMinutes, rec.rec.AreaName, orUnknown(rec.rec.Killer), int(r.banLength().Minutes()))
		r.e.Log.Error("death spiral", "area", rec.area, "deaths", inArea, "killer", rec.rec.Killer)
		r.emit(goals.Event{Type: goals.EvError, Message: msg, Monster: rec.rec.Killer, Area: rec.rec.AreaName, AreaID: rec.area})
		r.loop.stepArea = 0
	}
	r.applyRepeatSite(rec)
	if lastHour >= t.PauseDeathsPerHour {
		r.obMu.Lock()
		r.setReaction(rec.rec.Time, "bot paused itself (too many deaths in the hour)")
		r.obMu.Unlock()
		msg := fmt.Sprintf("Paused itself after %d deaths in the last hour. Resume when you have looked at the death log.", lastHour)
		r.mu.Lock()
		if r.goal != nil && !r.selfStop {
			r.paused, r.selfStop, r.message = true, true, msg
			r.mu.Unlock()
			r.e.Log.Error("paused after too many deaths", "lastHour", lastHour)
			r.emit(goals.Event{Type: goals.EvError, Message: msg})
		} else {
			r.mu.Unlock()
		}
	}
}

// markCorpse records what became of the corpse stamped at. Caller holds obMu.
func (r *Runner) markCorpse(at time.Time, retrieved bool, why string) {
	for i := range r.dlog.Records {
		if rc := &r.dlog.Records[i]; rc.Time.Equal(at) {
			rc.Retrieved, rc.Abandoned, rc.Why = retrieved, !retrieved, why
			return
		}
	}
}

// banLength is how long a death spiral keeps the character out of an area.
func (r *Runner) banLength() time.Duration {
	if r.cfg.DangerBanSec > 0 {
		return sec(r.cfg.DangerBanSec)
	}
	return time.Duration(max(r.e.Tune.Bot.DangerBanMin, 1)) * time.Minute
}

// deathUnfavorable says the area was a bad bet when the character died: monster level well above clvl, or no
// potions and no gold to recover with (the latest snapshot is the respawned character, so potions and gold are what
// it has now).
func (r *Runner) deathUnfavorable(area int) bool {
	s := r.latest()
	if s == nil || s.Player == nil {
		return false
	}
	if m := r.areaMLvl(area, s.Player.Difficulty); m > 0 && m-s.Player.Level >= 3 {
		return true
	}
	healing, _, known := r.beltAndInv(s)
	return known && healing == 0 && s.Player.Gold == 0 && s.Player.Level >= resource.GearGateLevel
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}

// crowdSpotAgain: the death was in a crowd (hostiles near at least combat.crowdN) and the same area already took another
// death within 12 h. A second corpse run into the same crowd is how Soaky lost the same gear three times.
func (r *Runner) crowdSpotAgain(src DeathRecord, at time.Time) bool {
	n := r.e.Tune.Combat.CrowdN
	if n <= 0 || src.Autopsy == nil || src.Autopsy.HostilesNear < n {
		return false
	}
	r.obMu.Lock()
	defer r.obMu.Unlock()
	for i := range r.dlog.Records {
		if rc := &r.dlog.Records[i]; rc.Area == src.Area && !rc.Time.Equal(at) && rc.Time.Before(at) && at.Sub(rc.Time) <= repeatEscalateWindow {
			return true
		}
	}
	return false
}

// retrievalDecision is the corpse policy: retrieve only when the gear is worth the trip and the area is judged
// survivable, at most CorpseRetries attempts per death.
func (r *Runner) retrievalDecision(rec *recovery, snap *d2.Snapshot) (bool, string) {
	t := r.e.Tune.Bot
	now := r.e.Clock.Now()
	src := rec.rec
	worth := corpseWorth(src) // worn items plus the expected gold pile at the death spot
	if worth < t.CorpseValueMin && r.gearVerdict(snap).Naked {
		// This body holds nothing (the character was already naked): the gear lies on an older one. Retrieve that first.
		r.obMu.Lock()
		g := r.dlog.gearCorpse(t.CorpseValueMin, now.Add(-gearCorpseAge), rec.rec.Time)
		if g != nil {
			src = *g
		}
		r.obMu.Unlock()
		if g != nil {
			rec.target, rec.area, rec.pos = g.Time, g.Area, g.Pos
			worth = g.CorpseValue // the older body's gold was dropped long ago
			r.e.Log.Info("corpse: the gear is on an older body", "area", g.Area, "x", g.Pos.X, "y", g.Pos.Y, "value", g.CorpseValue, "ageMin", int(now.Sub(g.Time).Minutes()))
		}
	}
	if !r.cfg.Hardcore && rec.why == "" && rec.area != 0 && !r.d.Plan.IsTown(rec.area) && worth < t.CorpseMinValue {
		r.e.Log.Info(fmt.Sprintf("corpse: skipped (value %d < %d)", worth, t.CorpseMinValue), "area", rec.area)
		return false, fmt.Sprintf("corpse skipped (value %d < %d)", worth, t.CorpseMinValue)
	}
	switch {
	case r.cfg.Hardcore:
		return false, "hardcore: no corpse policy yet"
	case rec.why != "":
		return false, rec.why
	case t.CorpseRetries <= rec.tries:
		return false, "no retrieval attempts left"
	case rec.area == 0 || r.d.Plan.IsTown(rec.area):
		return false, "no corpse in the field"
	case worth < t.CorpseValueMin:
		return false, fmt.Sprintf("corpse worth about %d gold, threshold %d", worth, t.CorpseValueMin)
	case src.KillerBoss:
		return false, "killed by a boss"
	case r.banned(rec.area):
		return false, "area avoided after a death spiral"
	case r.areaGuardWhy(snap, rec.area) != "":
		r.e.Log.Info(fmt.Sprintf("corpse: skipped (guarded area %d)", rec.area), "area", rec.area, "why", r.areaGuardWhy(snap, rec.area))
		return false, "route guard closed (" + r.areaGuardWhy(snap, rec.area) + ")"
	case r.crowdSpotAgain(src, rec.rec.Time):
		return false, "a crowd killed it here and this area took a death in the last 12 h: not walking into it again"
	case !rec.target.IsZero() && r.areaMLvl(rec.area, snap.Player.Difficulty) > snap.Player.Level-2:
		return false, fmt.Sprintf("a naked character must not walk into monster level %d at clvl %d", r.areaMLvl(rec.area, snap.Player.Difficulty), snap.Player.Level)
	}
	for _, s := range r.d.Plan.Steps[difficultyName(snap.Player.Difficulty)] {
		if s.Area == rec.area && snap.Player.Level < s.EnterLvl {
			return false, fmt.Sprintf("area needs level %d, character is %d", s.EnterLvl, snap.Player.Level)
		}
	}
	r.obMu.Lock()
	recent := r.dlog.since(now.Add(-time.Duration(t.SpiralMinutes)*time.Minute), rec.area)
	r.obMu.Unlock()
	if recent >= t.SpiralDeaths-1 {
		return false, fmt.Sprintf("%d recent deaths in this area", recent)
	}
	return true, fmt.Sprintf("corpse worth about %d gold, area survivable", worth)
}

// finishRecovery runs once the character is alive again: decide on the corpse, restock (poverty mode when the gear
// is gone for good), retrieve at most once, then record what the death cost.
func (r *Runner) finishRecovery(ctx context.Context, snap *d2.Snapshot, c control) bool {
	l := &r.loop
	rec := l.rec
	rec.respawned = true
	if rec.warned {
		r.mu.Lock()
		r.errorUntil = time.Time{}
		r.mu.Unlock()
	}
	if !rec.decided {
		rec.decided = true
		rec.retrieve, rec.why = r.retrievalDecision(rec, snap)
		if c.goal == nil {
			rec.retrieve, rec.why = false, "no goal"
		}
		r.e.Log.Info("corpse decision", "retrieve", rec.retrieve, "why", rec.why, "area", rec.area, "x", rec.pos.X, "y", rec.pos.Y,
			"value", rec.rec.CorpseValue, "naked", r.gearVerdict(snap).Naked, "older", !rec.target.IsZero())
	}
	lost := equipCount(snap) < rec.equipBefore
	if snap.Player.InTown() {
		plan := r.fullChores("after death")
		if !rec.retrieve && lost {
			plan.Poverty, plan.GearTier = true, r.e.Tune.Bot.PovertyTier
			plan.Reason = "poverty rebuild after death"
		}
		if !r.do(ctx, c, StateRecovering, "Restocking after death", sec(r.cfg.TownTimeoutSec), "post-death chores", func(ctx context.Context) error {
			_, err := r.d.Town.Chores(ctx, plan)
			return err
		}) {
			return true
		}
	}
	retrieved := false
	if rec.retrieve {
		rec.tries++
		rec.attempting = true
		before := equipCount(snap)
		ok := r.do(ctx, c, StateRecovering, "Retrieving the corpse", sec(r.cfg.TravelTimeoutSec), "corpse", func(ctx context.Context) error {
			return r.retrieveCorpse(ctx, rec.area, rec.pos)
		})
		if s := r.poll(); s != nil && isDead(s.Player) {
			return true // died on the way: the new recovery sees attempting and never retries
		}
		rec.attempting = false
		if s := r.latest(); s != nil {
			retrieved = equipCount(s) > before
			r.e.Log.Info("corpse attempt", "ok", ok, "equipBefore", before, "equipAfter", equipCount(s))
		}
		if !retrieved {
			// one attempt only: the gear stays where it fell and the rebuild starts
			rec.why = whyRetrievalFailed
			if s := r.latest(); s != nil && equipCount(s) < rec.equipBefore {
				plan := r.fullChores("poverty rebuild after failed retrieval")
				plan.Poverty, plan.GearTier = true, r.e.Tune.Bot.PovertyTier
				l.trip = &trip{plan: plan, returnTo: fieldArea(s.Player), emergency: true}
				r.setLastTrip(r.e.Clock.Now())
			}
		}
	}
	r.sweepDeathGold(ctx, c, rec)
	now := r.e.Clock.Now()
	r.obMu.Lock()
	var gold, equip int
	if s := r.ob.lastSnap; s != nil && s.Player != nil && s.State == d2.StateInGame {
		gold = s.Player.Gold
		equip = equipCount(s)
		r.baselineItems(s) // recovered or rebought gear is not a drop
	}
	if !rec.target.IsZero() {
		why := rec.why
		if retrieved {
			why = "retrieved after a later death"
		}
		r.markCorpse(rec.target, retrieved, why)
	}
	if lr := r.dlog.last(); lr != nil && lr.Time.Equal(rec.rec.Time) {
		lr.Retrieved, lr.Abandoned = retrieved && rec.target.IsZero(), !retrieved || !rec.target.IsZero()
		lr.Why = rec.why
		lr.CostGold = max(rec.goldBefore-gold, 0)
		lr.CostItems = max(rec.equipBefore-equip, 0)
		lr.CostSeconds = int(now.Sub(rec.at) / time.Second)
		if err := r.dlog.save(r.cfg.DeathLogPath); err != nil {
			r.e.Log.Warn("death log not saved", "err", err)
		}
	}
	r.obMu.Unlock()
	l.rec = nil
	l.needBuff = true
	r.setLastTrip(now)
	l.pendingErr = ""
	return true
}

// sweepDeathGold takes the gold pile the death left behind (see goldsweep.go). A corpse run that reached the spot
// already walked onto the nearest pile and only needs the leftovers; otherwise the bot goes back to the death spot from
// town when sweepDecision allows it. A carried purse above the stash keep line then sends the bot to the stash.
func (r *Runner) sweepDeathGold(ctx context.Context, c control, rec *recovery) {
	s := r.latest()
	if s == nil || s.Player == nil || c.goal == nil || isDead(s.Player) {
		return
	}
	d := rec.rec
	there := rec.retrieve && rec.target.IsZero() && rec.why != whyRetrievalFailed
	if rec.retrieve && !there {
		return // an older body or a failed corpse run: not worth a second try
	}
	if !there {
		if ok, why := r.sweepDecision(rec, s); !ok {
			r.e.Log.Info("death: gold sweep skipped", "why", why, "area", d.Area)
			return
		} else if !s.Player.InTown() {
			return
		}
	}
	before := s.Player.Gold
	r.do(ctx, c, StateRecovering, "Sweeping the death spot for gold", sec(r.cfg.TravelTimeoutSec), "gold sweep", func(ctx context.Context) error {
		return r.goldSweep(ctx, d.Area, d.Pos, there)
	})
	after := before
	if n := r.poll(); n != nil && n.Player != nil {
		after = n.Player.Gold
	}
	picked := max(after-before, 0)
	r.e.Log.Info(fmt.Sprintf("death: gold sweep area=%d picked=+%d gold (carried %d -> %d)", d.Area, picked, before, after), "expected", expectedPile(d))
	r.obMu.Lock()
	r.noteGoldRecovered(d.Time, picked)
	r.obMu.Unlock()
	if p := r.latest(); p != nil && p.Player != nil && !isDead(p.Player) {
		keep := r.e.Tune.Town.StashGoldBase + p.Player.Level*r.e.Tune.Town.StashGoldPerLvl
		if excess := p.Player.Gold - keep; excess >= max(keep, 500) && r.loop.trip == nil && !p.Player.InTown() {
			r.loop.trip = &trip{plan: r.fullChores("bank swept gold"), returnTo: fieldArea(p.Player)}
			r.setLastTrip(r.e.Clock.Now())
		}
	}
}

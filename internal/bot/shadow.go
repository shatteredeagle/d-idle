package bot

import (
	"context"
	"maps"
	"sync"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/arbiter"
	"d2lod-bot/internal/d2"
)

// The intent arbiter in shadow mode (docs/arbiter.md, knob bot.arbiterShadow). Beside runGoal, the arbiter is asked what
// it WOULD choose from the same world facts; the answer is only logged. runGoal and every hook in it are unchanged.

type arbShadow struct {
	mu sync.Mutex
	sh *arbiter.Shadow
	t0 time.Time
	// the standing typed refusal of a step (arbRefused): its need and the wake facts at that moment, and the family the
	// arbiter chose this tick (set by runGoalControlled while the chain runs; the quest hook reads it)
	refNeed, refFacts, chose string
	ctl                      atomic.Int64 // ticks the arbiter acted for since the last minute line (arbiter_control.go)
	tickAt                   time.Time    // last "tick" line (ticklog.go)
}

// minuteKV is the "arb" attribute of the minute line: per hook family diverged/ticks since the last minute line.
func (a *arbShadow) minuteKV() []any {
	a.mu.Lock()
	sh := a.sh
	a.mu.Unlock()
	if sh == nil {
		return nil
	}
	kv := []any{"arb", sh.Drain()}
	if n := a.ctl.Swap(0); n > 0 {
		kv = append(kv, "arbctl", n)
	}
	return kv
}

// Arbiter is the last shadow decision for the watch API and the debug page (read only; false while the knob is off).
func (r *Runner) Arbiter() (arbiter.Decision, bool) {
	r.arb.mu.Lock()
	sh := r.arb.sh
	r.arb.mu.Unlock()
	if sh == nil {
		return arbiter.Decision{}, false
	}
	return sh.Last()
}

// runGoalShadowed runs the chain and, with the knob on, asks the shadow. The shadow never blocks, never acts and its
// panics are recovered inside; the chain's result is returned untouched.
func (r *Runner) runGoalShadowed(ctx context.Context, snap *d2.Snapshot, c control) bool {
	acted := r.runGoalArb(ctx, snap, c)
	r.explainTick(snap, acted)
	return acted
}

func (r *Runner) runGoalArb(ctx context.Context, snap *d2.Snapshot, c control) bool {
	t := r.e.Tune.Bot
	if t.ArbiterShadow == 0 {
		acted := r.runGoal(ctx, snap, c)
		if t.TickLogSec > 0 {
			if v, ok := r.arbView(snap, c); ok {
				r.tickLog(snap, v, acted)
			}
		}
		return acted
	}
	v, ok := r.shadowView(snap, c)
	if ok && t.ArbiterControl != 0 {
		acted := r.runGoalControlled(ctx, snap, c, v)
		r.tickLog(snap, v, acted)
		return acted
	}
	acted := r.runGoal(ctx, snap, c)
	if ok {
		r.shadowTick(v, acted)
		r.tickLog(snap, v, acted)
	}
	return acted
}

func (r *Runner) shadowView(snap *d2.Snapshot, c control) (arbiter.View, bool) {
	v, ok := r.arbView(snap, c)
	if ok {
		v.Now = r.shadowNow()
	}
	return v, ok
}

// arbView collects the arbiter's world facts without touching the shadow (View.Now stays 0).
func (r *Runner) arbView(snap *d2.Snapshot, c control) (v arbiter.View, ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	p := snap.Player
	t := r.e.Tune.Bot
	l := &r.loop
	b := func(x bool) int {
		if x {
			return 1
		}
		return 0
	}
	inTown := p.InTown()
	area := "town"
	if !inTown {
		area = r.areaName(p.Area)
	}
	kit := r.kitOf(snap)
	r.obMu.Lock()
	real := r.ob.real
	r.obMu.Unlock()
	pg := r.d.PlayerGear
	f := map[string]int{
		arbiter.FactInTown:       b(inTown),
		arbiter.FactLifePct:      p.LifePct(),
		arbiter.FactHostile:      r.hostileNear(snap, t.ClearRadius),
		arbiter.FactPlayerOrder:  b(pg != nil && pg.Pending() > 0),
		arbiter.FactTrip:         b(l.trip != nil),
		arbiter.FactTripEmerg:    b(l.trip != nil && l.trip.emergency),
		arbiter.FactClaim:        b(!inTown && c.spec.camp == nil && len(l.questPending) > 0 && r.d.Quester != nil && !r.e.Clock.Now().Before(l.claimSkipUntil)),
		arbiter.FactPotions:      kit.BeltHP,
		arbiter.FactPotionsMin:   t.ResupplyPotions,
		arbiter.FactPackGold:     p.Gold,
		arbiter.FactStashGold:    p.StashGold,
		arbiter.FactRecovery:     b(r.recoveryActive()),
		arbiter.FactGoldFarm:     b(r.goldFarmOn()),
		arbiter.FactObjective:    b(c.spec.camp != nil),
		arbiter.FactQuestRefused: b(r.questRefusalStands(snap)),
		arbiter.FactSafety:       b(r.inDanger(snap)),
		arbiter.FactSellable:     r.resInput(snap).Sellable,
		arbiter.FactAreaDry:      b(len(r.dryAreas()) > 0),
	}
	if af, ok := r.d.Fighter.(interface {
		ArmyFacts(*d2.Snapshot) map[string]int
	}); ok {
		maps.Copy(f, af.ArmyFacts(snap)) // army facts (card dlv-stance-army); none for a hero without an army
	}
	if r.upkeepOn() {
		maps.Copy(f, r.planFacts(snap))
	}
	obj := ""
	if l.stepArea > 0 && c.spec.camp != nil {
		obj = r.areaName(l.stepArea) // U: the area of the quest step the planner picked last
	}
	return arbiter.View{Area: area, Progress: real, Facts: f, ObjectiveArea: obj}, true
}

// inDanger is the arbiter's danger fact: the fighter's safety verdict asks for a life action (a drink, a portal, a
// chicken, a crowd retreat). The verdict is the one danger rule; the arbiter holds no life line of its own.
func (r *Runner) inDanger(snap *d2.Snapshot) bool {
	if r.d.Fighter == nil {
		return false
	}
	a := r.d.Fighter.Safety(snap)
	return a != SafetyNone && a != SafetyDrinkMana
}

func (r *Runner) shadowNow() float64 {
	a := &r.arb
	a.mu.Lock()
	defer a.mu.Unlock()
	now := r.e.Clock.Now()
	if a.sh == nil {
		t := r.e.Tune.Bot
		cfg := arbiter.DefaultConfig()
		cfg.MinDwell, cfg.TravelWindow = float64(t.ArbiterDwellSec), float64(t.ArbiterTravelSec)
		a.sh = arbiter.NewShadow(cfg, arbiter.ShadowConfig{LogGapSec: float64(t.ArbiterLogGapSec)}, r.arbSources()...)
		a.t0 = now
	}
	return now.Sub(a.t0).Seconds()
}

func (r *Runner) shadowTick(v arbiter.View, acted bool) {
	defer func() { _ = recover() }()
	a := &r.arb
	a.mu.Lock()
	sh := a.sh
	a.mu.Unlock()
	chain := ""
	if acted {
		chain = chainFamily(r.currentWhy())
	}
	if line := sh.Tick(v, chain); line != "" {
		r.e.Log.Info("arbiter shadow", "diverge", line)
	}
	if d, ok := sh.Last(); ok {
		for _, in := range d.Starved {
			r.inv.noteStarved(r.starvedCause(d, in))
		}
	}
}

// arbRefused hands a typed refusal to the arbiter: a need with a row in arbiter.RefusalIntents becomes a temporary
// intent. Idle while the arbiter knobs are off.
func (r *Runner) arbRefused(need, wake string) {
	defer func() { _ = recover() }()
	if r.e.Tune.Bot.ArbiterShadow == 0 {
		return
	}
	now := r.shadowNow()
	facts := r.wakeFacts(r.latest())
	r.arb.mu.Lock()
	sh := r.arb.sh
	r.arb.refNeed, r.arb.refFacts = need, facts
	r.arb.mu.Unlock()
	sh.Refused(need, wake, now)
}

// questRefusalStands reads the typed refusal arbRefused recorded: true while the wake facts (level, kit, gold, area) are
// the ones the refusal was logged with. The quest source stays quiet meanwhile instead of asking again.
func (r *Runner) questRefusalStands(snap *d2.Snapshot) bool {
	r.arb.mu.Lock()
	need, facts := r.arb.refNeed, r.arb.refFacts
	r.arb.mu.Unlock()
	return need != "" && r.wakeFacts(snap) == facts
}

func (r *Runner) currentWhy() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.why.Code
}

// chainFamily maps the whyline code of the activity the chain just started or continued to the arbiter's hook family.
// U: the code can lag one activity behind when the chain acted without announcing a reason; unknown codes count as no
// family (the tick is not compared).
func chainFamily(code string) string {
	switch code {
	case WhyTownTrip, WhyTownChores, WhyHealTrip, WhyWaitHeal, WhyToTown, WhyBackToField, WhyRepairTrip, WhyRepairWait,
		WhyAfterDeath, WhyDeathCorpse, WhyPlayerReset:
		return "chores"
	case WhyClaimTrip:
		return "claim"
	case WhyQuestStep, WhyJourneyBlocked:
		return "quest"
	case WhyFarmUser:
		return "goldfarm"
	case WhySavingFor, WhyDeathGold, WhyFarmGold:
		return "recovery"
	case WhyPlayerItems:
		return "player"
	case WhyGuardExit:
		return "survive"
	case WhyWalk, WhyFight, WhyAreaDone, WhyFarmRuns, WhyFarmReset, WhyRepairFarm:
		return "level"
	}
	return ""
}

// upkeepOn: the arbiter controls the chores family, so the one upkeep source (supply plan, town10) replaces the trip and
// resupply sources. Off by default: bot.arbiterControl 0, or bit 4 of bot.arbiterFamilies unset (the default 3).
// A town without a planner keeps the old sources: the upkeep source sees the potion deficit only through the plan.
func (r *Runner) upkeepOn() bool {
	t := r.e.Tune.Bot
	if _, ok := r.d.Town.(supplyPlanner); !ok {
		return false
	}
	return t.ArbiterShadow != 0 && t.ArbiterControl != 0 && arbiter.Controlled(t.ArbiterFamilies, "chores")
}

// arbSources is the source set of the shadow: StdSources, or UpkeepSources when the chores family is controlled.
func (r *Runner) arbSources() []arbiter.Source {
	if r.upkeepOn() {
		return arbiter.UpkeepSources()
	}
	return arbiter.StdSources()
}

// planFacts is the supply plan's facts for the upkeep source (an excursion into the field, the default profile); none
// when the backend has no planner or it cannot answer.
func (r *Runner) planFacts(snap *d2.Snapshot) map[string]int {
	sp, ok := r.d.Town.(supplyPlanner)
	if !ok {
		return nil
	}
	pl, err := sp.SupplyPlan(snap, "", 0)
	if err != nil {
		r.planWarn.Do(func() { r.e.Log.Warn("town plan: " + err.Error()) })
		return nil
	}
	return arbiter.PlanFacts(pl)
}

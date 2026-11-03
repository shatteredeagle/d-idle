package combat

import (
	"fmt"
	"sync"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/merc"
)

// Field rules of the merc policy (card mercwire, docs/merc-wire.md). Both are off while Fighter.MercState is nil, which
// is the case unless the merc wiring (cmd/d2bot/mercwire.go, knob merc.enabled) installs it.

const (
	mercPotionGap  = 4 * time.Second // between two potions for the merc (the belt verb round trip plus the heal)
	mercLogEvery   = time.Minute
	mercTrivialMax = 1 // fewer than 2 hostiles in the fight radius is a trivial fight (U: judgment)
)

// mercFieldState is the throttle state of the two rules.
type mercFieldState struct {
	mu                          sync.Mutex // Safety is also called from the bot's guard goroutine
	potionAt, noVerbAt, leaveAt time.Time
}

// mercSnapshot reads the merc state through the wiring (cached there). ok is false when the rules are off or unknown.
func (f *Fighter) mercSnapshot() (merc.MercState, bool) {
	if f.MercState == nil {
		return merc.MercState{}, false
	}
	st, err := f.MercState()
	return st, err == nil && st.Known
}

// mercDownLeave is the crowd rule with the merc down (merc.LeaveWithMercDown): the pack that killed her is still here,
// so leave instead of fighting on alone. A town portal when the cast can succeed, else a retreat step; never a retreat
// while a runner is close (the RunnerGuard knob: walking away from a runner only costs hits, so the fight goes on and
// the ordinary crowd rule decides). SafetyNone: the rule has nothing to say.
func (f *Fighter) mercDownLeave(snap *d2.Snapshot, in SafetyInput) SafetyAction {
	t := in.Tune
	if in.InTown || !in.InFight || t.CrowdN <= 0 {
		return SafetyNone
	}
	st, ok := f.mercSnapshot()
	if !ok || !merc.LeaveWithMercDown(st, in.Crowd >= t.CrowdN) {
		return SafetyNone
	}
	var a SafetyAction
	switch {
	case in.HasPortal && in.Adjacent < crowdAdjacentMax:
		a = TownPortal
	case in.CanRetreat && !f.runnerClose(snap, snap.Player, float64(t.KiteDistance)):
		a = CrowdRetreat
	default:
		return SafetyNone
	}
	f.mercSt.mu.Lock()
	due := f.now().Sub(f.mercSt.leaveAt) >= mercLogEvery
	if due {
		f.mercSt.leaveAt = f.now()
	}
	f.mercSt.mu.Unlock()
	if due {
		f.log().Info(fmt.Sprintf("%sdown in a crowd, leaving action=%s crowd=%d", merc.Prefix, a, in.Crowd))
	}
	return a
}

// mercPotion gives a healing potion from the belt to the merc below merc.potionLifePct in a non-trivial fight
// (merc.ShouldGiveHealing). The verb is the semantic input's Belt(col, merc=true), "cmd belt N merc" on d2d (V: the
// controller sends it, internal/d2dinput test; U: that the fork heals the merc with it, drill D8.m was blocked). An
// input without it (Game.exe) logs `merc: potion wanted (no verb)` at most once a minute. Reports whether a potion went.
func (f *Fighter) mercPotion(snap *d2.Snapshot, hs []Target) bool {
	st, ok := f.mercSnapshot()
	if !ok || snap.Player.InTown() {
		return false
	}
	t := f.tune()
	near := 0
	for _, h := range hs {
		if h.Dist <= float64(t.FightRadius) {
			near++
		}
	}
	belt := BeltOf(snap.Inventory, f.cat)
	pc := merc.PotionCtx{FightTrivial: near <= mercTrivialMax, HealingPotions: belt.Count(PotionHealing)}
	if !merc.ShouldGiveHealing(st, pc, merc.KnobsFrom(f.e.Tune.Merc)) {
		return false
	}
	now := f.now()
	f.mercSt.mu.Lock()
	gapOK := now.Sub(f.mercSt.potionAt) >= mercPotionGap
	f.mercSt.mu.Unlock()
	if !gapOK {
		return false
	}
	sem, ok := input.AsSemantic(f.e.In)
	if !ok {
		f.mercSt.mu.Lock()
		due := now.Sub(f.mercSt.noVerbAt) >= mercLogEvery
		if due {
			f.mercSt.noVerbAt = now
		}
		f.mercSt.mu.Unlock()
		if due {
			f.log().Info(merc.Prefix + "potion wanted (no verb)")
		}
		return false
	}
	col, ok := belt.ColumnFor(PotionHealing, false)
	if !ok {
		return false
	}
	f.mercSt.mu.Lock()
	f.mercSt.potionAt = now // also after a failure: no hammering
	f.mercSt.mu.Unlock()
	if err := sem.Belt(col, true); err != nil {
		f.log().Info(fmt.Sprintf("%spotion failed col=%d err=%v", merc.Prefix, col+1, err))
		return false
	}
	f.log().Info(fmt.Sprintf("%sgave healing potion col=%d merc_life=%d", merc.Prefix, col+1, st.LifePct))
	return true
}

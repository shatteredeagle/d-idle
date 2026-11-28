package bot

import (
	"context"

	"d2lod-bot/internal/arbiter"
	"d2lod-bot/internal/d2"
)

// The arbiter's control path (docs/arbiter.md, knobs bot.arbiterControl and bot.arbiterFamilies, ledger row in
// data/darklaunch.json). Default off. With it on, the arbiter's decision is taken BEFORE the chain; when it chose a
// family that is under control, that family's hook runs first and the chain's lower hooks do not start. When the arbiter
// chose nothing, panicked, or the controlled hook found nothing to do, the old chain runs unchanged: the fallback.
// Wired so far: survive (a reflex: no chore, claim or quest step starts while the hero is under the drink line with
// hostiles near; it fights where it stands and the guard's reflex drink does the drinking) and player (item actions from
// the watch page run first), chores (a trip in progress or due) and claim (the claim trip), goldfarm (the player's gold farm order) and recovery (the purse recovery farm; a running gold farm order keeps its place, as in the chain). The other family bits are accepted and ignored. U: not run live.

// runGoalControlled is runGoalArb for a hero under control. It returns what the chain or the controlled hook did.
func (r *Runner) runGoalControlled(ctx context.Context, snap *d2.Snapshot, c control, v arbiter.View) bool {
	t := r.e.Tune.Bot
	r.shadowNow() // creates the shadow
	r.arb.mu.Lock()
	sh := r.arb.sh
	r.arb.mu.Unlock()
	d, ok := sh.Decide(v)
	acted, took, declined := false, false, false
	if ok && d.Chosen != nil {
		acted, took = r.arbAct(ctx, snap, c, d, t.ArbiterFamilies)
		if arbHooked(t.ArbiterFamilies, arbiter.Family(d.Chosen.Source)) {
			// Close the loop (review part 7, A3): a hook that did nothing backs its intent off (a decline, capped at
			// arbiter.DeclineMax), one that acted clears it.
			if took {
				sh.Report(*d.Chosen, v.Now, true)
			} else {
				sh.Decline(*d.Chosen, v.Now)
			}
			declined = !took
		}
	}
	if !took {
		r.arb.mu.Lock()
		r.arb.chose = ""
		if ok && d.Chosen != nil && !declined { // a family whose own hook just declined holds nothing back
			r.arb.chose = arbiter.Family(d.Chosen.Source)
		}
		r.arb.mu.Unlock()
		acted = r.runGoal(ctx, snap, c)
		r.arb.mu.Lock()
		r.arb.chose = ""
		r.arb.mu.Unlock()
	}
	r.arbCompare(sh, d, v.Now, acted, took)
	return acted
}

// arbHooked says the family is under control and has a hook arbAct runs (quest and level only veto, see arbHoldsQuest).
func arbHooked(mask int, fam string) bool {
	return fam != arbiter.SrcQuest && fam != arbiter.SrcLevel && arbiter.Controlled(mask, fam)
}

// arbAct runs the controlled hook for the family the arbiter chose. took is false when the family is not under control
// or its hook did nothing, so the chain must run.
func (r *Runner) arbAct(ctx context.Context, snap *d2.Snapshot, c control, d arbiter.Decision, mask int) (acted, took bool) {
	defer func() {
		if recover() != nil {
			acted, took = false, false
		}
	}()
	fam := arbiter.Family(d.Chosen.Source)
	if !arbiter.Controlled(mask, fam) {
		return false, false
	}
	switch fam {
	case arbiter.SrcSurvive:
		p := snap.Player
		if p == nil || p.InTown() {
			return false, false
		}
		r.setWhy(WhyFight, "area", r.areaName(p.Area), "life", itoa(p.LifePct()))
		acted = r.sweep(ctx, snap, c, p.Area)
	case arbiter.SrcPlayer:
		acted = r.playerActionStep(ctx, snap, c)
	case "chores":
		acted = r.arbChores(ctx, snap, c)
	case "claim":
		acted = r.claimTripStep(ctx, snap, c)
	case arbiter.SrcGoldFarm:
		if c.spec.camp != nil && r.goldFarmOn() {
			// the chain's prefix stands above both farm hooks (goal.go): a running trip goes on and due chores start
			// whatever the arbiter backed off (review 11, N1); a hurt hero with no potion heals first
			acted = r.arbChores(ctx, snap, c) || r.lowLifeGate(ctx, snap, c) || r.goldFarmStep(ctx, snap, c)
		}
	case arbiter.SrcRecovery:
		if c.spec.camp != nil && !r.goldFarmOn() && r.recoveryDue(snap) {
			acted = r.arbChores(ctx, snap, c) || r.lowLifeGate(ctx, snap, c) || r.recoveryStep(ctx, snap, c)
		}
	default:
		return false, false
	}
	if acted {
		r.arb.ctl.Add(1)
	}
	return acted, acted
}

// arbChores runs the chores family: a trip in progress goes on, else one starts when the chores are due. Under
// control the trip is not held back by the hooks the chain puts first (repair trip, belt fill, flail hold), which is the
// point: the arbiter judged chores the right thing now. False when nothing is due: the chain runs.
func (r *Runner) arbChores(ctx context.Context, snap *d2.Snapshot, c control) bool {
	if r.loop.trip != nil {
		return r.advanceTrip(ctx, snap, c)
	}
	return r.choresStep(ctx, snap, c)
}

// arbCompare is the second half of shadowTick for a decision taken before the chain acted.
func (r *Runner) arbCompare(sh *arbiter.Shadow, d arbiter.Decision, now float64, acted, took bool) {
	defer func() { _ = recover() }()
	chain := ""
	switch {
	case took:
		chain = arbiter.Family(d.Chosen.Source) // the arbiter acted: agreement by construction
	case acted:
		chain = chainFamily(r.currentWhy())
	}
	if line := sh.Compare(d, now, chain); line != "" {
		r.e.Log.Info("arbiter shadow", "diverge", line)
	}
	r.noteStarvedAll(d)
}

// arbHoldsQuest is the quest and level families' hook under control: the quest step defers to the arbiter's decision.
// When the arbiter chose another family (a refill it wants, a recovery farm, a player order) and that family's own hook
// found nothing to do this tick (quest bit), or chose the filler because the quest source stayed quiet behind a standing
// refusal (level bit), the quest step does not start; the hero runs a leveling slice in a safe area instead (the quest
// walk with an empty belt, the refusal storm). False when the knobs are off, the bit is not set, or the arbiter chose
// the quest step or nothing.
func (r *Runner) arbHoldsQuest() bool {
	t := r.e.Tune.Bot
	if t.ArbiterShadow == 0 || t.ArbiterControl == 0 {
		return false
	}
	r.arb.mu.Lock()
	fam := r.arb.chose
	r.arb.mu.Unlock()
	bit := arbiter.SrcQuest
	if fam == arbiter.SrcLevel {
		bit = arbiter.SrcLevel
	}
	if fam == "" || fam == arbiter.SrcQuest || !arbiter.Controlled(t.ArbiterFamilies, bit) {
		return false
	}
	// The chain asked goldFarmOn and recoveryDue just above this call and both said no (the gear gate, or a board chunk
	// that needs no gold): its rule stands, the arbiter's coarser recovery fact does not undo it.
	if fam == arbiter.SrcGoldFarm || fam == arbiter.SrcRecovery {
		return false
	}
	r.arb.ctl.Add(1)
	return true
}

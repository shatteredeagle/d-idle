package bot

import (
	"context"
	"fmt"
	"math"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/repair"
)

// The repair trip (docs/repair.md, user decision 2026-11-11): the player's "Repair now" makes a trip at the next safe
// moment instead of waiting for a town visit, and "Farm until repair" first earns the gold. The request lives in
// repair.Controller; this file chooses the moment and starts the trip, advanceTrip (goal.go) runs it with the
// repair phases, and the town does the shopping. All numbers are the "trip" block of internal/repair/modes.json.

// SetRepairTrip connects the repair controller. Without one (drills, named runs) the request never starts a trip.
func (r *Runner) SetRepairTrip(c *repair.Controller) { r.rep = c }

// safeAct is the verdict of repairSafeMoment.
type safeAct int

const (
	safeGo      safeAct = iota // leave now
	safeWait                   // keep playing, ask again
	safeRetreat                // step away from the pack first
)

// safeIn is what the decision reads (a pure function of it, so both classes test it with plain numbers).
type safeIn struct {
	Fight    int  // hostile within the clear radius: a pack is being fought
	Close    int  // hostile within PortalClearTiles, the fighter's ignore list not applied
	Elite    bool // a boss, unique or champion is among the pack
	Potions  int  // healing potions in the belt
	Escape   bool // a town portal source is carried
	LifePct  int
	WaitedS  int // seconds since the request began waiting for this moment
	NoBreak  bool
	Reason   string
	Retreats int // retreat steps taken in this wait
}

// repairSafeMoment is the rule: never leave in the middle of a fight. A pack in reach is finished first; a boss or
// elite is finished too when life is fine and healing potions and a way home exist (the exception), however long it
// takes up to three waits; otherwise after SafeWaitSecs the hero steps away from whatever is within PortalClearTiles.
// The cast itself needs no hostile within PortalClearTiles.
func repairSafeMoment(in safeIn, t repair.Trip) (safeAct, string) {
	if in.NoBreak {
		return safeWait, in.Reason
	}
	if in.Fight > 0 {
		if in.Elite && in.Potions >= t.ElitePotionMin && in.Escape && in.LifePct >= 50 && in.WaitedS < 3*t.SafeWaitSecs {
			return safeWait, "finishing the boss or elite first"
		}
		if in.WaitedS >= t.SafeWaitSecs && in.Close > 0 && in.Retreats < repairRetreatMax {
			return safeRetreat, "stepping away from the pack"
		}
		return safeWait, "finishing the pack first"
	}
	if in.Close > 0 {
		if in.WaitedS >= t.SafeWaitSecs && in.Retreats < repairRetreatMax {
			return safeRetreat, "stepping away from what is close"
		}
		return safeWait, "monsters are close"
	}
	return safeGo, ""
}

// repairRetreatMax bounds the retreat steps of one wait; after them the attempt is dropped for a while.
const repairRetreatMax = 6

// repairRetryGap is the pause after a wait that never found a safe moment.
const repairRetryGap = 90 * time.Second

// hostileWithin counts living hostile monsters within radius, the fighter's ignore list NOT applied: a monster it gave
// up on is still a monster next to the portal.
func (r *Runner) hostileWithin(s *d2.Snapshot, radius int) int {
	if s == nil || s.Player == nil || s.Player.InTown() {
		return 0
	}
	n := 0
	for i := range s.Monsters {
		u := &s.Monsters[i]
		if r.hostile(s, u) && math.Hypot(float64(u.X-s.Player.X), float64(u.Y-s.Player.Y)) <= float64(radius) {
			n++
		}
	}
	return n
}

func (r *Runner) packElite(s *d2.Snapshot) bool {
	for i := range s.Monsters {
		u := &s.Monsters[i]
		if u.Monster != nil && (u.Monster.IsUnique() || u.Monster.IsChampion()) && r.hostile(s, u) &&
			math.Hypot(float64(u.X-s.Player.X), float64(u.Y-s.Player.Y)) <= float64(r.e.Tune.Bot.ClearRadius) {
			return true
		}
	}
	return false
}

// repairFarmFigures is the farming leg's counter: gold on hand (pack plus stash, the town withdraws what the smith
// needs) and the target repairNeed x goldMult + kit + margin (goldFarmFigures with the repair multiplier).
func (r *Runner) repairFarmFigures(s *d2.Snapshot) (have, want int) {
	have, want, _ = r.goldFarmFigures(s)
	return have, want
}

// repairWhy is the Why line while a repair request waits or farms; the sweep asks it before its own lines.
func (r *Runner) repairWhy(s *d2.Snapshot) (code string, kv []string, ok bool) {
	if r.rep == nil || s == nil || s.Player == nil {
		return "", nil, false
	}
	q, open := r.rep.Active()
	if !open {
		return "", nil, false
	}
	switch q.Phase {
	case repair.PhaseFarming:
		have, want := r.repairFarmFigures(s)
		return WhyRepairFarm, []string{"area", r.areaName(s.Player.Area), "have", itoa(have), "want", itoa(want)}, true
	case repair.PhaseWaiting:
		return WhyRepairWait, []string{"why", q.Note}, true
	}
	return "", nil, false
}

// repairTripStep looks at the open repair request once per goal step and, at a safe moment, starts the trip. It
// reports whether it acted (a loot pass, a retreat step, the first trip step); false lets the normal play go on.
func (r *Runner) repairTripStep(ctx context.Context, snap *d2.Snapshot, c control) bool {
	rc := r.rep
	if rc == nil || snap == nil || snap.Player == nil {
		return false
	}
	l := &r.loop
	p := snap.Player
	now := r.e.Clock.Now()
	q, open := rc.Active()
	if !open {
		l.repairWaitSince, l.repairRetreats, l.repairLooted = time.Time{}, 0, false
		r.stopRepairFarm()
		return false
	}
	if l.trip != nil {
		return false // a trip (this one, or another) owns the hero
	}
	switch q.Phase {
	case repair.PhaseReturning: // paid, and the trip that was walking back is gone: the repair is done
		rc.EndTrip(now, true, false, "")
		return false
	case repair.PhaseGoing, repair.PhaseRepairing:
		// the trip that set this phase is gone (an emergency trip replaced it): start over, it did not count
		rc.AbortTrip("interrupted")
		q, _ = rc.Active()
	}
	if q.Kind == repair.KindFarm {
		have, want := r.repairFarmFigures(snap)
		if r.gf != nil {
			_ = r.gf.StartForRepair()
		}
		rc.FarmProgress(have, want, r.farmAreaName(snap))
		if have < want {
			return false // the Farm gold order plays on; the area choice is goldFarmTarget
		}
		rc.FarmReached()
		r.stopRepairFarm()
		r.e.Log.Info("repair: farm target reached, going to repair", "have", have, "want", want)
		q, _ = rc.Active()
	}
	r.stopRepairFarm()
	if !l.repairRetryAt.IsZero() && now.Before(l.repairRetryAt) {
		return false
	}
	if rc.TripWait(now) > 0 {
		return false // the last trip failed: its reason stays on the page until the cooldown is over
	}
	tcfg := rc.TripCfg()
	if p.InTown() { // no portal: straight to the smith
		return r.startRepairTrip(ctx, snap, c, 0)
	}
	in := safeIn{
		Fight: r.hostileNear(snap, r.e.Tune.Bot.ClearRadius), Close: r.hostileWithin(snap, tcfg.PortalClearTiles),
		Elite: r.packElite(snap), Escape: r.resInput(snap).HasTP, LifePct: p.LifePct(),
		Retreats: l.repairRetreats,
	}
	if h, _, known := r.beltAndInv(snap); known {
		in.Potions = h
	}
	if r.noTP.Load() {
		in.NoBreak, in.Reason = true, "this quest step cannot be interrupted"
	} else if r.bossRun.Load() {
		in.NoBreak, in.Reason = true, "a boss fight is on"
	}
	if l.repairWaitSince.IsZero() {
		l.repairWaitSince = now
	}
	in.WaitedS = int(now.Sub(l.repairWaitSince) / time.Second)
	switch act, why := repairSafeMoment(in, tcfg); act {
	case safeWait:
		rc.SetPhase(repair.PhaseWaiting, fmt.Sprintf("%s, %d s of %d", why, in.WaitedS, tcfg.SafeWaitSecs))
		if in.WaitedS > 4*tcfg.SafeWaitSecs+tcfg.SafeWaitSecs/2 { // never found one: drop this attempt for a while
			l.repairWaitSince, l.repairRetreats = time.Time{}, 0
			l.repairRetryAt = now.Add(repairRetryGap)
			rc.SetPhase(repair.PhaseQueued, "no safe moment yet, trying again soon")
		}
		return false
	case safeRetreat:
		rc.SetPhase(repair.PhaseWaiting, why)
		l.repairRetreats++
		r.setWhy(WhyRepairWait, "why", why)
		r.do(ctx, c, StateClearing, "Stepping away before the repair trip", sec(r.cfg.TownTimeoutSec), "repair retreat", func(ctx context.Context) error {
			return r.d.Fighter.Act(ctx, SafetyCrowdRetreat)
		})
		return true
	}
	if !l.repairLooted && r.d.Loot != nil { // the pack is dead: pick up what is in view before leaving
		l.repairLooted = true
		r.setWhy(WhyRepairWait, "why", "picking up loot first")
		r.do(ctx, c, StateClearing, "Picking up loot before the repair trip", sec(r.cfg.SweepTimeoutSec), "repair loot", func(ctx context.Context) error {
			_, err := r.d.Loot.PickUp(ctx, PickOpts{Radius: r.e.Tune.Bot.ClearRadius})
			return err
		})
		return true
	}
	return r.startRepairTrip(ctx, snap, c, fieldArea(p))
}

// startRepairTrip hands the hero to the trip machine (advanceTrip begins the request's trip at the cast point).
func (r *Runner) startRepairTrip(ctx context.Context, snap *d2.Snapshot, c control, returnTo int) bool {
	l := &r.loop
	l.repairWaitSince, l.repairRetreats, l.repairLooted = time.Time{}, 0, false
	plan := r.fullChores("repair trip")
	l.trip = &trip{plan: plan, returnTo: returnTo, repair: true}
	r.e.Log.Info("town trip", "reason", plan.Reason, "inTown", snap.Player.InTown(), "from", snap.Player.Area)
	return r.advanceTrip(ctx, snap, c)
}

// stopRepairFarm ends a Farm gold order a repair request owned.
func (r *Runner) stopRepairFarm() {
	if r.gf != nil && r.gf.ForRepair() {
		if q, open := r.rep.Active(); !open || q.Kind != repair.KindFarm {
			_ = r.gf.StopForRepair()
		}
	}
}

func (r *Runner) farmAreaName(s *d2.Snapshot) string {
	if r.gf == nil {
		return ""
	}
	return r.gf.View().Area
}

// repairBegin is called by advanceTrip where the trip really starts (the cast, the walk, or the first chore in town).
// false: do not go (a hostile is next to the portal spot, the trip budget or cooldown refuses); the trip is dropped and
// the request stays queued.
func (r *Runner) repairBegin(snap *d2.Snapshot, t *trip) bool {
	if !t.repair || t.begun {
		return true
	}
	rc := r.rep
	if rc == nil {
		return false
	}
	if !snap.Player.InTown() {
		if n := r.hostileWithin(snap, rc.TripCfg().PortalClearTiles); n > 0 {
			r.e.Log.Info("repair trip: not leaving, hostile too close", "count", n)
			rc.SetPhase(repair.PhaseWaiting, "monsters are close")
			r.loop.trip = nil
			return false
		}
	}
	if err := rc.BeginTrip(r.e.Clock.Now()); err != nil {
		r.e.Log.Info("repair trip refused", "why", err.Error())
		r.loop.trip = nil
		return false
	}
	t.begun = true
	r.setLastTrip(r.e.Clock.Now())
	return true
}

// repairTripEnd closes the request's trip when the machine drops it. repaired: the smith was paid.
func (r *Runner) repairTripEnd(t *trip, reason string) {
	if t == nil || !t.repair || !t.begun || r.rep == nil {
		return
	}
	t.begun = false
	q, open := r.rep.Active()
	repaired := open && q.Phase == repair.PhaseReturning
	short := open && !repaired && q.Short > 0
	if !open {
		return
	}
	if reason == "" && !repaired {
		reason = q.Note
	}
	r.rep.EndTrip(r.e.Clock.Now(), repaired, short, reason)
}

// repairRetry decides after a chores round of a repair trip: the repair was paid (go on), or the smith failed and one
// more round is made in this visit. Short of gold is not a failure to retry: the request moves on at the trip's end.
func (r *Runner) repairRetry(t *trip) bool {
	q, open := r.rep.Active()
	if !open || q.Phase == repair.PhaseReturning || q.Short > 0 || t.repairTry >= 1 {
		return false
	}
	t.repairTry++
	r.e.Log.Warn("repair trip: the smith did not repair, trying once more", "note", q.Note)
	return true
}

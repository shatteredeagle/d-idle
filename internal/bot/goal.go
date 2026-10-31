package bot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/resource"
)

// runGoal makes one decision for the active goal and runs at most one action.
func (r *Runner) runGoal(ctx context.Context, snap *d2.Snapshot, c control) bool {
	l := &r.loop
	p := snap.Player
	inTown := p.InTown()
	r.syncGames()

	if r.d.Prepare != nil && !l.prepared && l.prepTries < 3 && r.hostileNear(snap, r.e.Tune.Bot.ClearRadius) == 0 {
		l.prepTries++
		if r.do(ctx, c, StateInTown, "Binding hotkeys", 4*time.Minute, "prepare", r.d.Prepare) {
			l.prepared = true
		} else {
			r.e.Log.Warn("preparation failed; the fighter uses what is bound", "try", l.prepTries)
		}
		return true
	}
	if r.pointsStep(ctx, snap, c) {
		return true
	}
	if c.spec.kind == goals.Idle {
		return r.rest(ctx, snap, c)
	}
	if recoveryGoal(c) {
		r.updateRecovery(snap)
	}
	if l.trip != nil {
		return r.advanceTrip(ctx, snap, c)
	}
	if inTown && r.savePointStep(ctx, snap, c) {
		return true
	}
	if r.beltFillStep(ctx, snap) {
		return true
	}
	if plan, emergency, ok := r.needChores(snap); ok {
		now := r.e.Clock.Now()
		gap := sec(r.cfg.ResupplyCooldownSec)
		if emergency {
			gap /= 4
		}
		if now.Sub(r.getLastTrip()) >= gap {
			l.trip = &trip{plan: plan, returnTo: fieldArea(p), emergency: emergency}
			r.noteRebuyTrip(plan, p.Gold)
			r.setLastTrip(now) // also bounds retries when the chores keep failing
			r.e.Log.Info("town trip", "reason", plan.Reason, "emergency", emergency)
			return r.advanceTrip(ctx, snap, c)
		}
		if inTown && plan.Heal && r.belowResume(p) {
			// The cooldown paces the shopping trips, it must not keep a wounded character walking back out: the heal
			// is free and takes seconds (found live 2026-10-29: Booti re-entered at 25% life, the cooldown from the
			// failed field trip blocked the heal, she walked out and chickened again, about every 20 s).
			l.trip = &trip{plan: ChorePlan{Heal: true, Reason: "heal below the resume line", GoldFloor: r.e.Tune.Bot.GoldFloor}}
			r.e.Log.Info("town trip", "reason", l.trip.plan.Reason, "life", p.LifePct())
			return r.advanceTrip(ctx, snap, c)
		}
	}
	if r.lowLifeGate(ctx, snap, c) {
		return true
	}
	if inTown && (r.gearStep(ctx, snap, c) || (c.spec.camp == nil && r.claimStep(ctx, snap, c))) {
		return true
	}
	if !inTown && c.spec.camp == nil && len(l.questPending) > 0 && r.d.Quester != nil && !r.e.Clock.Now().Before(l.claimSkipUntil) && r.hostileNear(snap, r.e.Tune.Bot.ClearRadius) == 0 {
		if ok := r.do(ctx, c, StateResupplying, "Going to town to claim a quest reward", sec(r.cfg.TownTimeoutSec), "claim trip", r.d.Town.ToTown); ok {
			return true
		}
		// ToTown walks home when it has no scroll; a failure here means the way is blocked: stop retrying for a while
		l.claimSkipUntil = r.e.Clock.Now().Add(claimBackoff)
		r.e.Log.Warn("claim trip failed; no new claim trip for a while", "minutes", int(claimBackoff/time.Minute))
		return true
	}
	if acted, hold := r.ensureDifficulty(ctx, snap, c); acted {
		return true
	} else if hold {
		return false
	}
	if c.spec.kind == goals.Runs {
		return r.farmStep(ctx, snap, c)
	}
	if c.spec.camp != nil {
		if r.recoveryActive() && !r.gearVerdict(snap).Gated {
			return r.recoveryStep(ctx, snap, c) // the quest step waits until the purse is filled again
		}
		return r.taskStep(ctx, snap, c)
	}
	target, ok := r.target(snap, c)
	if !ok {
		return r.noTarget(ctx, snap, c)
	}
	if r.d.Plan.IsTown(target) {
		return r.rest(ctx, snap, c)
	}
	if p.Area != target {
		return r.travel(ctx, snap, c, target)
	}
	return r.sweep(ctx, snap, c, target)
}

func fieldArea(p *d2.Player) int {
	if p.InTown() {
		return 0
	}
	return p.Area
}

// syncGames forgets per-game memory (cleared areas) when a new game started.
func (r *Runner) syncGames() {
	r.obMu.Lock()
	n := r.ob.newGames
	r.obMu.Unlock()
	if n != r.loop.clearedGen {
		r.loop.clearedGen = n
		r.loop.cleared = map[int]bool{}
		r.loop.sweepKills = nil
		r.loop.needBuff = true
		r.loop.stepArea = 0
		r.loop.waitNote = false
	}
}

// ---- town trips ----------------------------------------------------------------------------------

// trip is one resumable town visit: portal home, chores, settle, portal back.
type trip struct {
	plan      ChorePlan
	returnTo  int
	stage     int // 0 go home, 1 chores, 2 settle, 3 back to the field
	emergency bool
	startGold int // gold when the trip was first advanced (in the field for most trips), for the trip log line
	restocks  int // extra shop rounds the field gate asked for in this town visit (max gateRestocks)
}

// antidotes counts antidote potions in belt and pack (0 without a catalog).
func (r *Runner) antidotes(s *d2.Snapshot) int {
	if r.d.Catalog == nil || s.Inventory == nil {
		return 0
	}
	n := 0
	for i := range s.Inventory.Items {
		u := &s.Inventory.Items[i]
		if u.Item == nil || (u.Item.Where != d2.ContBelt && u.Item.Where != d2.ContInventory) {
			continue
		}
		if m, ok := r.d.Catalog.Item(u.Class); ok && m.Antidote {
			n++
		}
	}
	return n
}

func (r *Runner) beltAndInv(s *d2.Snapshot) (healing int, fullPct int, known bool) {
	if r.d.Catalog == nil || s.Inventory == nil {
		return 0, 0, false
	}
	cells := 0
	for i := range s.Inventory.Items {
		u := &s.Inventory.Items[i]
		if u.Item == nil {
			continue
		}
		m, ok := r.d.Catalog.Item(u.Class)
		if !ok {
			continue
		}
		switch u.Item.Where {
		case d2.ContBelt:
			if m.Potion == PotionHealing || m.Potion == PotionRejuv {
				healing++
			}
		case d2.ContInventory:
			cells += max(m.W, 1) * max(m.H, 1)
		}
	}
	return healing, cells * 100 / invCells, true
}

// denYieldGap is how many levels below its Den floor a character may be when the floor yields.
const denYieldGap = 1

// fullKitPotions is the healing potion count that counts as a full belt for the planner.
const fullKitPotions = 4

// fullKit reports whether the character has a full belt and a town portal, the condition for planner steps that carry
// a KitLvl. Unknown belt contents (no catalog) count as a kit, so a bot without a reader keeps its old behaviour.
func (r *Runner) fullKit(s *d2.Snapshot) bool {
	healing, _, known := r.beltAndInv(s)
	if !known {
		return true
	}
	return healing >= fullKitPotions && r.resInput(s).HasTP
}

// leftBehinder is the optional part of a Looter that reports valuables left for lack of room (loot.Looter).
type leftBehinder interface {
	LeftBehind() int
	ClearLeftBehind()
}

// leftBehindPackPct is the pack fill from which left-behind valuables raise a trip.
const leftBehindPackPct = 80

// invCells is the character inventory grid (10 x 4). UNVERIFIED against data.InventoryLayout, which the TOWN agent owns.
const invCells = 40

// needChores decides whether a town visit is due. In the field the trigger is low potions or a full inventory; in
// town (goal start, or arriving by any route) it also covers healing.
func (r *Runner) needChores(s *d2.Snapshot) (plan ChorePlan, emergency, ok bool) {
	if r.loop.forceTrip {
		r.loop.forceTrip = false
		return r.fullChores("forced"), true, true
	}
	if plan, ok := r.rebuyDue(s); ok {
		return plan, false, true
	}
	return r.choresDue(s)
}

// rebuyDue sends a naked character that has earned the gold (resource.GearGate CanRebuy) to town for the poverty rebuy:
// weapon and body armor first. It asks again only once the gold has grown by half, so a vendor without a viable
// cheap item cannot make a trip loop. The bar is raised only when the trip really starts (startTrip): raising it here
// let a trip cooldown swallow the rebuy and push the bar above the gold for good (G1 attempt 6: 3779 gold, never rebuilt).
func (r *Runner) rebuyDue(s *d2.Snapshot) (ChorePlan, bool) {
	l := &r.loop
	v := r.gearVerdict(s)
	if !v.CanRebuy {
		if !v.Naked {
			l.rebuyBar = 0
		}
		return ChorePlan{}, false
	}
	if s.Player.Gold < l.rebuyBar {
		return ChorePlan{}, false
	}
	plan := r.fullChores("gear rebuy: " + v.Reason)
	plan.Poverty, plan.GearTier = true, r.e.Tune.Bot.PovertyTier
	return plan, true
}

// noteRebuyTrip raises the rebuy bar once a gear rebuy trip has really begun.
func (r *Runner) noteRebuyTrip(plan ChorePlan, gold int) {
	if plan.Poverty {
		r.loop.rebuyBar = gold * 3 / 2
		r.e.Log.Info("gear rebuy trip", "gold", gold, "reason", plan.Reason)
	}
}

// choresDue is the pure part of needChores (the guard calls it from its own goroutine).
func (r *Runner) choresDue(s *d2.Snapshot) (plan ChorePlan, emergency, ok bool) {
	p := s.Player
	t := r.e.Tune.Bot
	healing, full, known := r.beltAndInv(s)
	var why []string
	trip := resource.Trip{}
	if known && !p.InTown() {
		trip = r.tripVerdict(s)
	}
	broke := r.recoveryActive() && p.Gold < r.e.Tune.Resource.RecoveryKitGold
	if broke {
		trip = resource.Trip{} // recovery: a potion trip with an empty purse buys nothing (G1 attempt 9: 90 min of walking)
	}
	if trip.Go {
		plan.Potions = true
		plan.Heal = true // the policy's reasons include the free heal, which only runs when asked for
		why = append(why, "resource: "+trip.Reason)
		emergency = r.resInput(s).Belt.HP == 0
	}
	if known {
		threshold := t.ResupplyPotions
		if p.InTown() {
			threshold = t.ResupplyPotions * 2
		}
		if healing < threshold && p.Gold >= minPotionPrice && !trip.Futile && !broke {
			plan.Potions = true
			why = append(why, fmt.Sprintf("%d healing potions", healing))
			emergency = healing == 0 && !p.InTown()
		}
		if full >= t.InvFullPct && !r.packFullQuiet() {
			plan.Sell, plan.Stash, plan.Identify = true, true, true
			why = append(why, fmt.Sprintf("inventory %d%% full", full))
		}
	}
	// valuables were left on the ground for lack of room: make room (sell, stash, identify) and come back for them.
	// Pack >= 80 percent only, so a lone 2x3 item that does not fit a nearly empty pack cannot loop trips; the trip
	// rides the same gates as every routine trip (boss run, no-portal step, cooldown) in tripDue.
	if lb, ok := r.d.Loot.(leftBehinder); ok && known && !p.InTown() && full >= leftBehindPackPct && !r.packFullQuiet() {
		if n := lb.LeftBehind(); n > 0 {
			plan.Sell, plan.Stash, plan.Identify = true, true, true
			why = append(why, fmt.Sprintf("loot: pack full, %d valuable item(s) left on the ground", n))
		}
	}
	// Without a town portal scroll the trip is a walk of about a minute each way: one junk item is not worth it (D2DLOOP r22,
	// V live: a trip every 6 virtual minutes, about 40 % of the farming time, for 1 to 2 items and 2 to 30 gold).
	sellMin := r.e.Tune.Resource.PovertySellMin // XPGAP knob, 0 = the built-in minimums
	if sell, left := r.sellable(s), r.povertySellLeft(); !p.InTown() && r.inPoverty() && sell > left &&
		povertySellOK(sell, r.resInput(s).HasTP, sellMin) {
		// poor: junk is the quickest gold, sell it before anything else (one item is enough, the pack need not be full)
		plan.Sell, plan.Potions = true, p.Gold >= minPotionPrice
		why = append(why, fmt.Sprintf("poverty: %d items to sell, %d gold", sell, p.Gold))
	} else if !p.InTown() && p.Gold < t.GearMinGold && r.sellable(s) >= max(t.SellMinItems, sellMin) {
		plan.Sell, plan.Potions = true, p.Gold >= minPotionPrice
		why = append(why, fmt.Sprintf("%d items to sell, %d gold", r.sellable(s), p.Gold))
	}
	if p.InTown() && (p.LifePct() < 100 || p.ManaPct() < 100) {
		plan.Heal = true
		why = append(why, "heal")
	}
	if r.d.Town != nil && s.Inventory != nil && !r.cfg.IgnoreTownAdvice {
		if bad, reason := r.d.Town.NeedsTrip(s); bad {
			plan.Potions, plan.Sell, plan.Stash, plan.Identify = true, true, true, true
			why = append(why, reason)
		}
	}
	if w := r.mercWanted(s); w != "" {
		why = append(why, "merc: "+w)
	}
	if len(why) == 0 {
		return plan, false, false
	}
	plan.Repair = plan.Potions || plan.Sell
	plan.Reason = fmt.Sprint(why)
	return plan, emergency, true
}

func (r *Runner) advanceTrip(ctx context.Context, snap *d2.Snapshot, c control) bool {
	l := &r.loop
	t := l.trip
	p := snap.Player
	town := sec(r.cfg.TownTimeoutSec)
	t.plan.GoldFloor = r.e.Tune.Bot.GoldFloor
	t.plan.PotionFloor = r.potionFloor(p)
	if t.startGold == 0 {
		t.startGold = p.Gold
	}
	if !p.InTown() && t.stage < 3 {
		if t.stage > 0 { // fell out of town mid-trip (portal taken early): start over from the field
			t.stage = 0
		}
		ok := r.do(ctx, c, StateResupplying, "Going to town ("+t.plan.Reason+")", town, "to town", r.d.Town.ToTown)
		if !ok && r.State() == StateResupplying {
			// no portal, no scroll, no luck: give the trip up (the cooldown keeps it from being retried at once)
			if l.failures >= 1 {
				r.e.Log.Warn("town trip abandoned", "reason", t.plan.Reason)
				l.trip, l.failures = nil, 0
			}
		}
		return ok
	}
	switch t.stage {
	case 0, 1:
		t.stage = 1
		ok := r.do(ctx, c, StateResupplying, "Chores in town ("+t.plan.Reason+")", town, "chores", func(ctx context.Context) error {
			rep, err := r.d.Town.Chores(ctx, t.plan)
			if err == nil {
				r.e.Log.Info("chores done", "done", rep.Done, "skipped", rep.Skipped)
				if s := r.latest(); s != nil && len(rep.Blocked) > 0 {
					r.noteBlocked(s.Player, rep.Blocked)
					if n, ok := r.d.Leveler.(StatNeeder); ok && (rep.NeedStr > 0 || rep.NeedDex > 0) {
						n.NeedStats(rep.NeedStr, rep.NeedDex)
					}
				}
			}
			return err
		})
		if ok {
			t.stage = 2
			if s := r.latest(); s != nil {
				r.mercChore(ctx, s)
			}
			r.setLastTrip(r.e.Clock.Now())
			if s := r.latest(); s != nil {
				r.noteTripGold(s.Player.Gold)
				r.logTrip(t, s.Player.Gold)
				if lb, ok := r.d.Loot.(leftBehinder); ok {
					lb.ClearLeftBehind()
				}
				if r.inPoverty() {
					r.setSellLeft(r.sellable(s)) // what the vendor would not take stays, it must not trigger the next trip
				}
				// Selling and stashing did not make room (stash full, nothing sellable): do not make the same trip again.
				if _, full, known := r.beltAndInv(s); known && full >= r.e.Tune.Bot.InvFullPct {
					r.quietPackFull(time.Duration(r.e.Tune.Town.FutileMinutes) * time.Minute)
				}
			}
		}
		return true
	case 2:
		r.setState(StateResupplying, "Settling in town")
		r.pause(ctx, time.Duration(r.e.Tune.Bot.PortalSettleMs)*time.Millisecond)
		if t.returnTo != 0 {
			t.stage = 3
		} else {
			l.trip = nil
			l.needBuff = true
		}
		return true
	default:
		if r.gateField(snap, t) {
			return true
		}
		ok := r.do(ctx, c, StateResupplying, "Returning to the field", sec(r.cfg.TravelTimeoutSec), "back to field", func(ctx context.Context) error {
			return r.d.Town.BackToField(ctx, t.returnTo)
		})
		if ok || l.failures > 1 {
			l.trip = nil // on repeated failure travel() walks back by itself
			l.needBuff = true
		}
		return true
	}
}

// rest keeps the character safely in town (the idle goal, or a farm target that is a town).
func (r *Runner) rest(ctx context.Context, snap *d2.Snapshot, c control) bool {
	if !snap.Player.InTown() {
		return r.do(ctx, c, StateResupplying, "Returning to town", sec(r.cfg.TownTimeoutSec), "to town", r.d.Town.ToTown)
	}
	if plan, _, ok := r.needChores(snap); ok && r.e.Clock.Now().Sub(r.getLastTrip()) >= sec(r.cfg.ResupplyCooldownSec) {
		r.loop.trip = &trip{plan: plan}
		r.noteRebuyTrip(plan, snap.Player.Gold)
		r.setLastTrip(r.e.Clock.Now())
		return r.advanceTrip(ctx, snap, c)
	}
	r.setState(StateInTown, "Resting in town")
	return false
}

func (r *Runner) povertySellLeft() int {
	r.obMu.Lock()
	defer r.obMu.Unlock()
	return r.ob.sellLeft
}

func (r *Runner) setSellLeft(n int) {
	r.obMu.Lock()
	r.ob.sellLeft = n
	r.obMu.Unlock()
}

// povertyWalkMinItems is the pack junk a poor character wants before it walks to town without a portal scroll.
const povertyWalkMinItems = 4

// povertyPortalMinItems is the same with a portal scroll: the trip burns the scroll (100 gold, KitTPGold) and junk sells for
// about 25 gold an item, so fewer items lose money (D2DECON, V d2d g11: 8 of 14 trips sold one item with a scroll and the
// next shop visit bought a new scroll, gold stayed under 100 all run; the kit was never complete).
const povertyPortalMinItems = 6

// povertySellWorth says whether a poor character's pack junk justifies a sell trip.
func povertySellWorth(sell int, hasTP bool) bool {
	if hasTP {
		return sell >= povertyPortalMinItems
	}
	return sell >= povertyWalkMinItems
}

// povertySellOK is povertySellWorth with the povertySellMin knob (XPGAP) on top: min 0 keeps the built-in rule.
func povertySellOK(sell int, hasTP bool, min int) bool {
	return povertySellWorth(sell, hasTP) && sell >= min
}

// ---- target selection ----------------------------------------------------------------------------

func (r *Runner) allowedActs(s *d2.Snapshot) map[int]bool {
	m := map[int]bool{}
	for _, a := range r.cfg.Acts {
		m[a] = true
	}
	if s != nil && s.Player != nil {
		m[s.Player.Act+1] = true
	}
	return m
}

// banned reports whether the death-spiral breaker currently keeps the bot out of area.
func (r *Runner) banned(area int) bool {
	r.obMu.Lock()
	defer r.obMu.Unlock()
	until, ok := r.avoid[area]
	if ok && !r.e.Clock.Now().Before(until) {
		delete(r.avoid, area)
		r.e.Log.Info("area ban lifted", "area", area, "name", r.areaName(area))
		return false
	}
	return ok
}

// starterBanMax caps what is left of the ban of a first field (the way out of a town) once the character stands in
// town because of it: staying there gains no gear or level, so a long ban would only deadlock it (LIVE Tcont
// 2026-10-29: Blood Moor banned, nothing reachable). The cap applies when the planner finds the ban in its way.
func (r *Runner) starterBanMax() time.Duration {
	return time.Duration(max(r.e.Tune.Bot.StarterBanMin, 1)) * time.Minute
}

// isStarter says area is the first field of an act in the plan, the only way out of that act's town.
func (r *Runner) isStarter(area int) bool {
	seen := map[int]bool{}
	for _, st := range r.d.Plan.Steps["normal"] {
		if r.d.Plan.IsTown(st.Area) {
			continue
		}
		act := r.d.Plan.Act(st.Area)
		if !seen[act] {
			seen[act] = true
			if st.Area == area {
				return true
			}
		}
	}
	return false
}

// routeBan is the first banned area on the walk from the current area to target (hops excludes the start), with the
// time its ban ends. 0 when the walk is free or the route is unknown.
func (r *Runner) routeBan(s *d2.Snapshot, target int) (int, time.Time) {
	if s == nil || s.Player == nil || r.d.Mover == nil || r.d.Plan.IsTown(target) || s.Player.Area == target {
		return 0, time.Time{}
	}
	hops, err := r.d.Mover.Route(s.Player.Area, target)
	if err != nil {
		return 0, time.Time{}
	}
	for _, h := range append(hops, target) {
		if h == s.Player.Area || r.d.Plan.IsTown(h) || !r.banned(h) {
			continue
		}
		r.obMu.Lock()
		until := r.avoid[h]
		if now := r.e.Clock.Now(); s.Player.InTown() && until.Sub(now) > r.starterBanMax() && r.isStarter(h) {
			until = now.Add(r.starterBanMax())
			r.avoid[h] = until
			r.e.Log.Warn("starter area ban shortened: the character is stuck in town", "area", h, "name", r.areaName(h), "minutes", int(r.starterBanMax().Minutes()))
		}
		r.obMu.Unlock()
		return h, until
	}
	return 0, time.Time{}
}

// banWait is the dashboard text while every way out is banned: the longest-lasting ban in force, "" when none.
func (r *Runner) banWait() string {
	now := r.e.Clock.Now()
	best, bu := 0, time.Time{}
	r.obMu.Lock()
	for a, until := range r.avoid {
		if until.After(now) && until.After(bu) {
			best, bu = a, until
		}
	}
	r.obMu.Unlock()
	if best == 0 {
		return ""
	}
	return fmt.Sprintf("Waiting: %s banned for %d more min", r.areaName(best), int(bu.Sub(now).Minutes())+1)
}

func difficultyName(d d2.Difficulty) string {
	return [...]string{"normal", "nightmare", "hell"}[min(max(int(d), 0), 2)]
}

// target returns the area to work in (and publishes it for the guard).
func (r *Runner) target(s *d2.Snapshot, c control) (int, bool) {
	a, ok := r.target0(s, c)
	// A target whose walk crosses a banned area is no target: skip it until that ban ends and pick the next best one.
	for try := 0; ok && try < 8; try++ {
		b, until := r.routeBan(s, a)
		if b == 0 {
			break
		}
		r.obMu.Lock()
		r.avoid[a] = until
		r.avoidWhy[a] = r.avoidWhy[b]
		r.obMu.Unlock()
		r.loop.stepArea = 0
		r.e.Log.Info("target skipped: the route crosses a banned area", "area", a, "name", r.areaName(a), "banned", r.areaName(b))
		if try == 7 {
			ok = false
			break
		}
		a, ok = r.target0(s, c)
	}
	if !ok {
		a = 0
	}
	if ok && c.spec.kind == goals.ReachLevel {
		a = r.holdTarget(s, a)
	}
	r.obMu.Lock()
	r.ob.targetArea = a
	r.obMu.Unlock()
	return a, ok
}

// Hysteresis of the area choice (found live: the Paladin flipped between Blood Moor and Cold Plains at the border,
// poverty mode and the plan taking turns, each flip a route through a waypoint that was not usable).
const (
	holdMin       = 3 * time.Minute  // a chosen area is kept at least this long at the same level
	holdMinFlappy = 10 * time.Minute // ... when it flipped holdFlips times within holdWindow
	holdFlips     = 3
	holdWindow    = 10 * time.Minute
)

type areaHold struct {
	area, level int
	since       time.Time
	flips       []time.Time
}

// holdTarget returns the previously chosen area instead of the new choice a while after the switch, unless the level
// changed or the old area is cleared, avoided or a town.
func (r *Runner) holdTarget(s *d2.Snapshot, a int) int {
	h := &r.loop.hold
	now := r.e.Clock.Now()
	lvl := s.Player.Level
	if h.area == 0 || h.area == a || h.level != lvl || r.loop.cleared[h.area] || r.banned(h.area) || r.d.Plan.IsTown(h.area) || r.d.Plan.IsTown(a) {
		if h.area != a {
			h.area, h.level, h.since = a, lvl, now
		}
		return a
	}
	need := holdMin
	live := h.flips[:0]
	for _, t := range h.flips {
		if now.Sub(t) < holdWindow {
			live = append(live, t)
		}
	}
	h.flips = live
	if len(h.flips) >= holdFlips {
		need = holdMinFlappy
	}
	if now.Sub(h.since) < need {
		return h.area
	}
	h.flips = append(h.flips, now)
	h.area, h.level, h.since = a, lvl, now
	return a
}

// worthFloor is how many levels below the character a farm area's leave level may be when better areas are only
// excluded (cleared, unreachable): beyond it the experience penalty makes the area worthless (docs/leveler.md).
const worthFloor = 4

// target0 is the choice itself. ok is false when nothing is available (cleared in this game, or too dangerous).
func (r *Runner) target0(s *d2.Snapshot, c control) (int, bool) {
	l := &r.loop
	p := s.Player
	switch c.spec.kind {
	case goals.ReachLevel:
		if r.recoveryActive() && !r.gearVerdict(s).Gated {
			r.updateStarvation(s)
			r.pushPoverty(true, recoveryPackDanger) // after updateStarvation, which pushes the lean 0
			if a, ok := r.recoveryArea(s); ok {
				l.stepArea = 0
				return a, true
			}
		}
		if p.InTown() {
			if v := r.fieldVerdict(s); v.BloodMoor {
				if a, ok := r.starveStep(s, difficultyName(p.Difficulty)); ok {
					return a, true
				}
			}
		}
		if g := r.gearVerdict(s); g.Gated {
			a := r.safeFarmArea(s)
			if a == 0 || r.banned(a) {
				return 0, false // nothing safe is open: wait in town rather than walk a naked character into the field
			}
			if l.cleared[a] {
				// Nothing left to kill here in this game: no target, so the leveling slice walks home and starts a fresh game
				// (monsters respawn). Returning the cleared area made the sweep spin without progress, which fed the failsafe
				// ladder (LIVE G1 attempt 4: 64 kills, then 'area cleared' 50 ms apart for 60 s).
				return 0, false
			}
			if !l.gearNote {
				l.gearNote = true
				r.e.Log.Warn("gear gate: farming the first field", "reason", g.Reason, "area", a, "name", r.areaName(a), "gold", p.Gold)
			}
			l.stepArea = 0
			return a, true
		} else {
			l.gearNote = false
		}
		if r.updateStarvation(s) {
			if a, ok := r.starveArea(s); ok {
				if !l.starveNote {
					l.starveNote = true
					r.e.Log.Info("poverty mode: farming", "area", a, "name", r.areaName(a), "gold", p.Gold)
				}
				return a, true
			}
		}
		if l.stepArea != 0 && l.stepLevel == p.Level && !l.cleared[l.stepArea] && !r.banned(l.stepArea) {
			return l.stepArea, true
		}
		ex, cleared := map[int]bool{}, map[int]bool{}
		for a := range l.cleared {
			ex[a], cleared[a] = true, true
		}
		r.obMu.Lock()
		for a, until := range r.avoid {
			if until.After(r.e.Clock.Now()) {
				ex[a] = true
			}
		}
		r.obMu.Unlock()
		r.guardedAreas(s, ex)
		t := r.e.Tune.Bot
		diff := difficultyName(p.Difficulty)
		if healing, _, known := r.beltAndInv(s); known && healing < 2 && diff == "normal" {
			ex[areaDenOfEvil] = true // a dungeon quest needs a potion buffer; farm the open field first
		}
		pick := func() (Step, bool) {
			return r.d.Plan.Pick(diff, p.Level, PickOptions{Margin: t.LevelMargin, Slack: t.LeaveSlack, Acts: r.allowedActs(s), Exclude: ex, Cleared: cleared, Floor: worthFloor, FullKit: r.fullKit(s), Melee: MeleeTank(p.Class)})
		}
		step, ok := pick()
		if !ok && r.denGated(s) && p.Level >= r.denFloor(s)-denYieldGap && !cleared[areaDenOfEvil] && !r.banned(areaDenOfEvil) {
			// Blood Moor is cleared in this game and nothing else fits: waiting in town gains nothing, so the Den floor
			// yields (the potion buffer rule above still holds). Only one level below the floor: a clvl 1 caster in the
			// Den drinks 4 potions in 12 s (D2DPACE live); it waits for a fresh game instead.
			if healing, _, known := r.beltAndInv(s); !known || healing >= 2 {
				delete(ex, areaDenOfEvil)
				r.e.Log.Info("den: floor waived, nothing else to level in", "level", p.Level)
				step, ok = pick()
			}
		}
		for ok {
			d := r.d.Plan.EntryAllowed(diff, p.Level, step.Area, nil)
			if d.OK {
				break
			}
			r.e.Log.Info("planner: entry gate refused area", "area", step.Area, "why", d.Why)
			ex[step.Area] = true
			step, ok = pick()
		}
		if !ok {
			return 0, false
		}
		if step.Area != l.stepArea {
			run := r.runFor(diff, step)
			l.run = &run
			r.e.Log.Info("planner", "level", p.Level, "difficulty", diff, "area", step.Area, "name", step.Name, "run", run.Key, "kind", run.Kind, "why", step.Purpose)
		}
		r.e.Log.Info(levelPlanLine(p.Level, step, t.LevelMargin, t.LeaveSlack))
		l.stepArea, l.stepLevel = step.Area, p.Level
		r.obMu.Lock()
		r.ob.stepLevel = p.Level
		r.obMu.Unlock()
		return step.Area, true
	default:
		a := c.spec.area
		return a, !l.cleared[a] && !r.banned(a)
	}
}

// ensureDifficulty asks the out-of-game flow for a new game in the wanted difficulty. Without a Cycler the wish is
// logged and the bot carries on in the current difficulty (hold is false). A campaign cannot carry on in another
// difficulty: it walks to town, tries the switch and, while it is in backoff or failed, holds in town (hold true, the
// caller idles) instead of going Stuck into a permanent pause. Tries are spaced by diffBackoff, which doubles per
// failure up to 30 minutes and resets once the difficulty matches.
func (r *Runner) ensureDifficulty(ctx context.Context, s *d2.Snapshot, c control) (acted, hold bool) {
	p := s.Player
	l := &r.loop
	want := c.spec.difficulty
	if want == "" && c.settings.Difficulty != "" && c.settings.Difficulty != "auto" {
		want = c.settings.Difficulty
	}
	if want == "" && c.spec.kind == goals.ReachLevel {
		want = r.wantedDifficulty(s) // never goes down, honors the ban after a failed switch and the death rate
	}
	if want == "" || rank(want) == int(p.Difficulty) {
		l.diffTries, l.diffNext = 0, time.Time{}
		return false, false
	}
	camp := c.spec.camp != nil
	now := r.e.Clock.Now()
	if now.Before(l.diffNext) {
		return false, camp
	}
	if !p.InTown() {
		if !camp {
			r.e.Log.Info("difficulty wish not met", "want", want, "have", difficultyName(p.Difficulty), "inTown", false)
			l.diffNext = now.Add(time.Minute)
			return false, false
		}
		return r.rest(ctx, s, c), true // walk to town first; the switch is tried from there
	}
	if r.d.Cycler == nil {
		r.e.Log.Info("difficulty wish not met", "want", want, "have", difficultyName(p.Difficulty), "cycler", false)
		l.diffNext = now.Add(10 * time.Minute)
		return false, false
	}
	l.diffTries++
	l.diffNext = now.Add(min(time.Duration(1<<min(l.diffTries, 5))*time.Minute, 30*time.Minute))
	ok := r.do(ctx, c, StateInTown, "Starting a "+want+" game", sec(r.cfg.TownTimeoutSec), "new game", r.oogRun(func(ctx context.Context) error {
		return r.d.Cycler.NewGame(ctx, want)
	}))
	if ok {
		r.verifyDifficulty(want)
	} else if l.diffBan == nil || l.diffBan[want].IsZero() {
		if l.diffBan == nil {
			l.diffBan = map[string]time.Time{}
		}
		l.diffBan[want] = now.Add(time.Duration(r.e.Tune.Bot.DifficultyBanMin) * time.Minute)
	}
	return ok, false
}

func rank(d string) int {
	switch d {
	case "nightmare":
		return 1
	case "hell":
		return 2
	}
	return 0
}

// noTarget handles "everything useful in this game is done": go home and ask for a fresh game, or wait visibly.
func (r *Runner) noTarget(ctx context.Context, s *d2.Snapshot, c control) bool {
	l := &r.loop
	if !s.Player.InTown() {
		return r.do(ctx, c, StateResupplying, "Area done, returning to town", sec(r.cfg.TownTimeoutSec), "to town", r.d.Town.ToTown)
	}
	why := "the target area is cleared or too dangerous"
	if c.spec.kind == goals.ReachLevel {
		why = "no eligible area is left in this game"
	}
	if r.d.Cycler != nil && len(l.cleared) > 0 {
		diff := c.spec.difficulty
		return r.do(ctx, c, StateInTown, "Starting a fresh game", sec(r.cfg.TownTimeoutSec), "new game", r.oogRun(func(ctx context.Context) error {
			return r.d.Cycler.NewGame(ctx, diff)
		}))
	}
	if w := r.banWait(); w != "" {
		if r.idleBreak(false) {
			return false
		}
		r.setState(StateInTown, w)
		r.pause(ctx, 5*time.Second)
		return false
	}
	reason := "no out-of-game flow is wired"
	if r.d.Cycler != nil {
		reason = "nothing was cleared in this game (areas avoided for a while or held back), so a new game would not help"
	}
	r.setError("Waiting for a fresh game: "+why+" and "+reason, sec(r.cfg.ErrorRetrySec))
	return false
}

// ---- traveling -----------------------------------------------------------------------------------

func (r *Runner) travel(ctx context.Context, s *d2.Snapshot, c control, target int) bool {
	name := r.areaName(target)
	return r.do(ctx, c, StateTraveling, "Walking to "+name, sec(r.cfg.TravelTimeoutSec), "travel to "+name, func(ctx context.Context) error {
		return r.travelTo(ctx, target)
	})
}

func (r *Runner) curArea() int {
	if s := r.latest(); s != nil {
		return s.Player.Area
	}
	return 0
}

// waitArea polls until the player is in area, or fails after d. Loading screens read as "not in game" and are waited out.
func (r *Runner) waitArea(ctx context.Context, area int, d time.Duration) error {
	deadline := r.e.Clock.Now().Add(d)
	for {
		if err := ctx.Err(); err != nil {
			return context.Cause(ctx)
		}
		if s := r.poll(); s != nil && s.Player.Area == area && !isDead(s.Player) {
			return nil
		}
		if r.e.Clock.Now().After(deadline) {
			return fmt.Errorf("did not arrive in %s (area %d) within %s", r.areaName(area), area, d)
		}
		r.e.Clock.Sleep(time.Duration(r.e.Tune.Bot.WatchMs) * time.Millisecond)
	}
}

// travelTo walks from the current area to target: waypoint to the furthest waypoint area on the route when one works
// (from an area that has a waypoint), otherwise exit by exit. Every hop is verified by the area id.
func (r *Runner) travelTo(ctx context.Context, target int) error {
	for hops := 0; hops < 40; hops++ {
		cur := r.curArea()
		if cur == 0 {
			r.e.Clock.Sleep(time.Duration(r.e.Tune.Bot.WatchMs) * time.Millisecond)
			if err := ctx.Err(); err != nil {
				return context.Cause(ctx)
			}
			continue
		}
		if cur == target {
			return nil
		}
		hops, err := r.d.Mover.Route(cur, target) // nav owns the exit graph, with its own seen-exit and hint fallbacks
		if err != nil || len(hops) == 0 {
			return fmt.Errorf("no route from %s (%d) to %s (%d): %v", r.areaName(cur), cur, r.areaName(target), target, err)
		}
		if err := r.routeAllowed(r.latest(), cur, target, hops); err != nil {
			return err
		}
		path := append([]int{cur}, hops...)
		if r.d.Plan.Waypoints[cur] && r.e.Tune.Nav.UseWaypoints {
			moved := false
			snap := r.latest()
			active, known := r.activeWaypoints(snap)
			// Only a trip that skips at least WpMinDistance level hops beats walking (the panel round trip costs
			// the 10 s cooldown plus the clicks); the route guard above already vetted every area on the path.
			for i := len(path) - 1; i >= max(2, int(r.e.Tune.Nav.WpMinDistance)) && !moved; i-- {
				w := path[i]
				if !r.d.Plan.Waypoints[w] || !r.wpTripWorthTrying(snap, w, active, known) {
					continue
				}
				if err := r.d.Mover.TakeWaypoint(ctx, w); err != nil {
					if ctx.Err() != nil {
						return context.Cause(ctx)
					}
					r.e.Log.Info("wp: not usable", "from", cur, "to", w, "err", err)
					r.loop.wp.failed[wpKey{diffOf(snap), w}] = r.e.Clock.Now().Add(wpFailedFor)
					continue
				}
				if err := r.waitArea(ctx, w, 30*time.Second); err != nil {
					return err
				}
				r.markTouched(snap, cur, w)
				moved = true
			}
			if moved {
				continue
			}
		}
		r.touchWaypointHere(ctx)
		next := path[1]
		if err := r.d.Mover.ExitTo(ctx, next); err != nil {
			return fmt.Errorf("exit to %s: %w", r.areaName(next), err)
		}
		if err := r.waitArea(ctx, next, 30*time.Second); err != nil {
			return err
		}
	}
	return errors.New("travel: too many hops")
}

// ---- clearing -----------------------------------------------------------------------------------

// Futile-clear handling: how far the bot must move, and for how long the rule holds.
const (
	futileMoveAway = 40
	futileFor      = 90 * time.Second
)

// sweep is one pass in the target area: buff, fight what is near, pick up, explore toward more. When exploration runs
// out of unexplored ground with nothing hostile left, the area counts as cleared for this game.
func (r *Runner) sweep(ctx context.Context, s *d2.Snapshot, c control, area int) bool {
	name := r.areaName(area)
	return r.do(ctx, c, StateClearing, "Fighting in "+name, sec(r.cfg.SweepTimeoutSec), "sweep "+name, func(ctx context.Context) error {
		return r.sweepPass(ctx, area)
	})
}

// sweepPass is the action inside sweep (also the Clear action of campaign steps): one pass of buff, fight, pick up,
// explore in area.
func (r *Runner) sweepPass(ctx context.Context, area int) error {
	l := &r.loop
	radius := r.e.Tune.Bot.ClearRadius
	name := r.areaName(area)
	if l.needBuff {
		l.needBuff = false
		if err := r.d.Fighter.Buff(ctx); err != nil {
			r.e.Log.Warn("buff failed", "err", err)
		}
	}
	// A Clear that kills nothing means what is near cannot be fought from here (unreachable, avoided, immune): the
	// bot walks on and only reacts to hostiles again once it moved FutileMoveAway subtiles from that spot.
	hostile := func() bool {
		s := r.latest()
		if s == nil || r.hostileNear(s, radius) == 0 {
			return false
		}
		if l.futile && r.e.Clock.Now().Before(l.futileUntil) && max(abs(s.Player.X-l.futilePos.X), abs(s.Player.Y-l.futilePos.Y)) < futileMoveAway {
			return false
		}
		return true
	}
	if !hostile() {
		r.touchWaypointHere(ctx)
	}
	if hostile() {
		res, err := r.d.Fighter.Clear(ctx, ClearOpts{Radius: radius})
		if err != nil {
			return err
		}
		if res.Killed > 0 {
			if l.sweepKills == nil {
				l.sweepKills = map[int]int{}
			}
			l.sweepKills[area] += res.Killed
		}
		if res.Killed == 0 {
			if s := r.latest(); s != nil {
				l.futile, l.futilePos, l.futileUntil = true, input.Point{X: s.Player.X, Y: s.Player.Y}, r.e.Clock.Now().Add(futileFor)
				r.e.Log.Info("clear killed nothing; moving on", "remaining", res.Remaining, "x", s.Player.X, "y", s.Player.Y)
			}
			r.pause(ctx, time.Duration(r.e.Tune.Bot.TickMs)*time.Millisecond)
		} else {
			l.futile = false
		}
	}
	if r.d.Loot != nil {
		if _, err := r.d.Loot.PickUp(ctx, PickOpts{Radius: radius}); err != nil {
			return fmt.Errorf("pick up: %w", err)
		}
	}
	if !hostile() {
		if r.emptyAreaLeave(area) {
			return nil
		}
		// Explore (a roam) runs for minutes until a hostile shows: the empty-area check must run inside it.
		if err := r.d.Mover.Explore(ctx, func() bool { return hostile() || r.emptyAreaLeave(area) }); err != nil {
			return fmt.Errorf("explore: %w", err)
		}
		if r.loop.cleared[area] && r.loop.emptyArea == 0 {
			return nil // left as empty (emptyAreaLeave logged it)
		}
		if !hostile() {
			finish := l.run != nil && l.run.Area == area
			complete := true
			if finish {
				var ended bool
				if ended, complete = r.sweepEndsQuest(ctx, l.run); !ended {
					return nil
				}
			}
			l.cleared[area] = true
			r.e.Log.Info("area cleared", "area", area, "name", name)
			r.logSweep(area, true)
			if finish && complete {
				r.runFinished(l.run)
			}
		}
	}
	return nil
}

// emptyAreaLeave marks area cleared once the sweep went Config.EmptyAreaSec without a kill (KILLFIX: on d2d 7 of the 20
// longest kill gaps were 200-500 s of roaming an emptied area). Quest areas (the step's run is in this area) keep
// roaming: the quest needs every target. The planner then takes the next area or the fresh-game path.
func (r *Runner) emptyAreaLeave(area int) bool {
	l := &r.loop
	if r.cfg.EmptyAreaSec <= 0 || (l.run != nil && l.run.Area == area) {
		return false
	}
	now := r.e.Clock.Now()
	r.obMu.Lock()
	kills := r.ob.kills
	r.obMu.Unlock()
	stale := now.Sub(l.emptyLast) > 90*time.Second
	l.emptyLast = now
	if l.emptyArea != area || l.emptyKills != kills || l.emptySince.IsZero() || stale {
		l.emptyArea, l.emptyKills, l.emptySince = area, kills, now
		return false
	}
	if now.Sub(l.emptySince) < sec(r.cfg.EmptyAreaSec) {
		return false
	}
	l.cleared[area] = true
	l.emptyArea = 0
	r.e.Log.Info("area abandoned: no kill for a while", "area", area, "name", r.areaName(area), "seconds", int(now.Sub(l.emptySince)/time.Second))
	r.logSweep(area, true)
	return true
}

// logSweep writes the sweep summary; coverage comes from a Mover that roams (fully known maps), else 0/0.
func (r *Runner) logSweep(area int, cleared bool) {
	var v, m int
	if c, ok := r.d.Mover.(interface{ Coverage() (int, int) }); ok {
		v, m = c.Coverage()
	}
	r.e.Log.Info(fmt.Sprintf("sweep: coverage %d/%d points, kills %d, cleared=%t", v, m, r.loop.sweepKills[area], cleared), "area", area)
}

func (r *Runner) getLastTrip() time.Time {
	r.obMu.Lock()
	defer r.obMu.Unlock()
	return r.ob.lastTrip
}

func (r *Runner) setLastTrip(t time.Time) {
	r.obMu.Lock()
	r.ob.lastTrip = t
	r.obMu.Unlock()
}

func (r *Runner) quietPackFull(d time.Duration) {
	r.obMu.Lock()
	r.ob.packFullUntil = r.e.Clock.Now().Add(d)
	r.obMu.Unlock()
}

func (r *Runner) packFullQuiet() bool {
	r.obMu.Lock()
	defer r.obMu.Unlock()
	return r.e.Clock.Now().Before(r.ob.packFullUntil)
}

// tripDue reports whether a town trip is due now, cooldown included.
func (r *Runner) tripDue(s *d2.Snapshot) bool {
	_, emergency, ok := r.choresDue(s)
	if !ok {
		return false
	}
	// A routine trip (full pack, worn gear) waits until the boss run is over: it cancels the fight and the boss heals while
	// the character shops (V live: Blood Raven went back to 100 percent three times). The boss script has its own escape.
	if r.bossRun.Load() && !emergency {
		return false
	}
	if r.noTP.Load() { // Constraints.NoTownPortal (Ancients): the step must not be interrupted by a trip
		return false
	}
	gap := sec(r.cfg.ResupplyCooldownSec)
	if emergency {
		gap /= 4
	}
	return r.e.Clock.Now().Sub(r.getLastTrip()) >= gap
}

// ---- low life gate -------------------------------------------------------------------------------

const (
	healFailLimit = 3
	reentryLimit  = 4 // chicken re-entries within reentryWindow before the goal is paused
	reentryWindow = 15 * time.Minute
	lowLifeRetry  = 2 * time.Minute // wait between heal-trip rounds when the way to town keeps failing
)

// resumeLifePct is the life the character needs to go (back) into the field. Life does not regenerate by itself in
// D2 (S: general game knowledge, not checked live; resource.Input.LifeRegen is 0 unless gear says otherwise), so
// resting in the field cannot reach it: only town (healer, potions) can.
func (r *Runner) resumeLifePct() int { return resource.FromEnv(*r.e.Tune).EngageLifePct }

func (r *Runner) belowResume(p *d2.Player) bool {
	return p.MaxLife > 0 && p.LifePct() < r.resumeLifePct()
}

// lowLifeGate stops the goal from engaging anything below the resume line. In the field with nothing hostile near and
// nothing to drink it goes to town (portal, walk, or waypoint: Town.ToTown); in town, when the heal did not lift life
// above the line, it holds. Repeated failures wait lowLifeRetry (lowLifeWait) and try again instead of pausing.
func (r *Runner) lowLifeGate(ctx context.Context, snap *d2.Snapshot, c control) bool {
	p := snap.Player
	l := &r.loop
	if !r.belowResume(p) {
		l.healFails, l.lowLifeUntil = 0, time.Time{}
		return false
	}
	if p.InTown() {
		if l.trip != nil {
			return false
		}
		// the heal ran (or was impossible) and life is still low: do not walk out and chicken again
		l.healFails++
		if l.healFails >= healFailLimit {
			if r.lowLifeWait(fmt.Sprintf("Life %d%% in town and healing did not help (%d tries)", p.LifePct(), l.healFails)) {
				return true
			}
		}
		l.trip = &trip{plan: ChorePlan{Heal: true, Reason: "life below the resume line", GoldFloor: r.e.Tune.Bot.GoldFloor}}
		return r.advanceTrip(ctx, snap, c)
	}
	if r.hostileNear(snap, r.e.Tune.Bot.ClearRadius) > 0 {
		return false // the fighter and the safety guard handle a fight
	}
	if in := r.resInput(snap); in.Belt.HP+in.Belt.RV > 0 {
		return false // a potion is the guard's business
	}
	if l.healFails >= healFailLimit {
		if r.lowLifeWait(fmt.Sprintf("Life %d%% in area %d, no potion, and the way to town failed %d times", p.LifePct(), p.Area, l.healFails)) {
			return true
		}
	}
	ok := r.do(ctx, c, StateResupplying, fmt.Sprintf("Going to town to heal (life %d%%)", p.LifePct()), sec(r.cfg.TownTimeoutSec), "heal trip", r.d.Town.ToTown)
	if !ok {
		l.healFails++
	}
	return true
}

// lowLifeWait is the last resort of the low life gate. It used to selfPause, which an unattended soak never leaves
// (live 2026-10-29: Soaky idled 20+ minutes in Cave Level 1 at 19% life). Now it holds for lowLifeRetry in an
// interruptible wait (a hostile near or a potion ends the gate's say, see lowLifeGate), then clears the failure count
// so the heal trip is tried again. The reason stays visible as the activity.
func (r *Runner) lowLifeWait(why string) bool {
	l := &r.loop
	now := r.e.Clock.Now()
	if l.lowLifeUntil.IsZero() {
		l.lowLifeUntil = now.Add(lowLifeRetry)
		r.e.Log.Warn("low life and no way to heal: waiting before retrying", "why", why, "retry", lowLifeRetry)
	}
	if !now.Before(l.lowLifeUntil) {
		l.lowLifeUntil, l.healFails = time.Time{}, 0
		return false
	}
	r.setState(StateResupplying, why+"; retrying in "+l.lowLifeUntil.Sub(now).Round(time.Second).String())
	r.e.Clock.Sleep(time.Duration(r.e.Tune.Bot.WatchMs) * time.Millisecond)
	return true
}

// noteReentry records a rejoin after the game was left (a chicken or a crash). Too many in a short time means the
// recovery does not work: pause with the reason rather than chicken forever.
func (r *Runner) noteReentry() {
	l := &r.loop
	now := r.e.Clock.Now()
	keep := l.reentries[:0]
	for _, t := range l.reentries {
		if now.Sub(t) < reentryWindow {
			keep = append(keep, t)
		}
	}
	l.reentries = append(keep, now)
	if len(l.reentries) >= reentryLimit {
		msg := fmt.Sprintf("The game was left and re-entered %d times in %v; paused. Look at the character (life, potions), then Resume.", len(l.reentries), reentryWindow)
		l.reentries = nil
		r.selfPause(msg)
	}
}

// gateRestocks bounds the shop rounds the field gate may add to one town visit; after that the character goes anyway.
const gateRestocks = 2

func (r *Runner) fieldVerdict(s *d2.Snapshot) resource.FieldVerdict {
	if _, _, known := r.beltAndInv(s); !known {
		return resource.FieldVerdict{Go: true, Reason: "belt unknown"}
	}
	return resource.FieldGate(r.resInput(s), resource.FromEnv(*r.e.Tune))
}

// gateField applies resource.FieldGate before the trip returns to the field. It reports true when it changed the plan:
// Restock runs the chores once more (bounded), BloodMoor redirects a poor character to its first field, where the walk
// home works. A poor character is never sent shopping, so the two cannot loop.
func (r *Runner) gateField(s *d2.Snapshot, t *trip) bool {
	if g := r.gearVerdict(s); g.Gated {
		if a := r.safeFarmArea(s); a != 0 && t.returnTo != a {
			r.e.Log.Info("field gate: gear gate, first field only", "reason", g.Reason, "area", a)
			t.returnTo = a
		}
		return false
	}
	v := r.fieldVerdict(s)
	switch {
	case v.Go:
		return false
	case v.BloodMoor:
		if a, ok := r.starveStep(s, difficultyName(s.Player.Difficulty)); ok && t.returnTo != a {
			r.e.Log.Info("field gate: first field only", "reason", v.Reason, "area", a)
			t.returnTo = a
		}
	case v.Restock && t.restocks < gateRestocks:
		t.restocks++
		r.e.Log.Info("field gate: restock", "reason", v.Reason, "attempt", t.restocks)
		t.plan.Potions, t.plan.Repair = true, true
		t.stage = 1
		return true
	default:
		r.e.Log.Info("field gate: restock attempts used, going anyway", "reason", v.Reason)
	}
	return false
}

// levelPlanLine is the "level plan" log line: the area the picker chose for the level and why (window, past the last
// window, or below the first entry level).
func levelPlanLine(level int, st Step, margin, slack int) string {
	why := fmt.Sprintf("in window %d-%d", st.EnterLvl, st.LeaveLvl)
	switch {
	case level > st.LeaveLvl+slack:
		why = fmt.Sprintf("past every window, last eligible (leaves at %d)", st.LeaveLvl)
	case level < st.EnterLvl+margin:
		why = fmt.Sprintf("below every entry level, first eligible (enters at %d)", st.EnterLvl)
	}
	return fmt.Sprintf("level plan: clvl %d -> area %d %s (%s)", level, st.Area, st.Name, why)
}

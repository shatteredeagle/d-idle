package bot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/ladder"
	"d2lod-bot/internal/repair"
	"d2lod-bot/internal/resource"
	"d2lod-bot/internal/town"
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
	if r.playerActionStep(ctx, snap, c) || r.errandTripStep(ctx, snap, c) {
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
	if r.repairTripStep(ctx, snap, c) {
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
			if inTown {
				// A visit in town costs no scroll and lastTrip was stamped by the failed field attempt seconds ago: the
				// full gap sent the character back out with no potion six times (Gate 1 20261102, BELT A). Only a
				// short pause keeps a shop that cannot sell from being retried every tick.
				gap = townShopGap
			}
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
	// After the town chores, not before: a save point that fired first re-entered the game with the purse unspent and the
	// tome empty (Gate 2 seed 13, docs/gate2loop.md).
	if inTown && r.savePointStep(ctx, snap, c) {
		return true
	}
	if !inTown && c.spec.camp == nil && len(l.questPending) > 0 && r.d.Quester != nil && !r.e.Clock.Now().Before(l.claimSkipUntil) && r.hostileNear(snap, r.e.Tune.Bot.ClearRadius) == 0 {
		r.setWhy(WhyClaimTrip)
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
		if r.goldFarmOn() {
			return r.goldFarmStep(ctx, snap, c) // the player's order: farm gold until told to stop (docs/goldfarm.md)
		}
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
		r.forgetGame()
	}
}

// forgetGame drops the loop's per-game memory (cleared areas, sweep kills, buff and step state). One helper for both
// ways a new game is noticed: the observer's menu poll (syncGames) and a bot-started fresh game on a backend with no
// menu poll (freshGameToLevel, d2d).
func (r *Runner) forgetGame() {
	l := &r.loop
	l.cleared = map[int]bool{}
	l.sweepKills = nil
	l.needBuff = true
	l.stepArea = 0
	l.waitNote = false
	// F1: a belt refusal from the last game (sticky belt-full mark) does not hold in the new one
	if bf, ok := r.d.Town.(interface{ ForgetBelt() }); ok {
		bf.ForgetBelt()
	}
}

// ---- town trips ----------------------------------------------------------------------------------

// trip is one resumable town visit: portal home, chores, settle, portal back.
type trip struct {
	plan       ChorePlan
	returnTo   int
	stage      int // 0 go home, 1 chores, 2 settle, 3 back to the field
	emergency  bool
	startGold  int  // gold when the trip was first advanced (in the field for most trips), for the trip log line
	startBasis int  // the policy's gold (stash included while no healing potion) at that moment, for the in-town latch
	restocks   int  // extra shop rounds the field gate asked for in this town visit (max gateRestocks)
	repair     bool // the player's Repair now trip (repairtrip.go): phases, guards and the log of durability before and after
	begun      bool // the request's trip was started (repairBegin)
	repairTry  int  // chore rounds at the smith in this trip
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

// denPotionEntry is the belt healing count the Den needs to be entered; denPotionStay is the count below which a hero
// already inside leaves it (docs/denshuttle.md: drinking the second to last potion in the Den flipped the farm target to
// Blood Moor mid fight, and every crossing of that mouth is a d2d arrival bounce).
const (
	denPotionEntry = 2
	denPotionStay  = 1
)

// denKitShort reports that the potion buffer rule keeps the hero out of the Den now. It has a gap between entering and
// staying so one drink inside the Den does not turn the target around.
func (r *Runner) denKitShort(s *d2.Snapshot) bool {
	healing, _, known := r.beltAndInv(s)
	if !known {
		return false
	}
	if s.Player.Area == areaDenOfEvil {
		return healing < denPotionStay
	}
	return healing < denPotionEntry
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
	plan, emergency, ok = r.choresDue(s)
	if ok && emergency && r.townLatched(s) {
		return plan, emergency, false
	}
	return plan, emergency, ok
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

// restockReason says a Townsman trip reason is a plain potion or portal scroll restock (the trip floor applies to it).
func restockReason(reason string) bool {
	for _, k := range []string{"healing potions", "mana potions", "belt columns", "no town portal scroll"} {
		if strings.Contains(reason, k) {
			return true
		}
	}
	return false
}

// spendPlanner is the optional Townsman half of the trip floor: what the shop would spend on a visit now.
type spendPlanner interface {
	PlannedSpend(s *d2.Snapshot) int
}

// tripWorthIt is the trip floor (goldfloor, docs/goldfloor.md finding 2: 83 percent of the trips spent under 30 gold, each
// burning a portal scroll): a pure restock trip with a healing potion still in the belt needs a planned purchase of
// Town.TripMinSpend gold. Without a planner, or with the knob at 0, every trip is worth it (the old behaviour).
func (r *Runner) tripWorthIt(s *d2.Snapshot) bool {
	floor := r.e.Tune.Town.TripMinSpend
	sp, ok := r.d.Town.(spendPlanner)
	if floor <= 0 || !ok {
		return true
	}
	spend := sp.PlannedSpend(s)
	if spend >= floor {
		return true
	}
	r.e.Log.Debug("trip floor: purchase too small, no trip", "planned", spend, "floor", floor, "gold", s.Player.Gold)
	return false
}

// townShopGap is the pause between two shopping rounds in town while the character has no healing potion.
const townShopGap = 5 * time.Second

// choresDue is the pure part of needChores (the guard calls it from its own goroutine).
func (r *Runner) choresDue(s *d2.Snapshot) (plan ChorePlan, emergency, ok bool) {
	p := s.Player
	t := r.e.Tune.Bot
	healing, full, known := r.beltAndInv(s)
	var why []string
	// hard marks a reason that is no potion/scroll restock (survival trip, sell, pack, merc, ...): the trip floor leaves it alone.
	hard := false
	trip := resource.Trip{}
	if known && !p.InTown() {
		trip = r.tripVerdict(s)
	}
	// BELTFIX5: banked gold counts when no healing potion is left; the town visit withdraws it (Town.withdraw)
	kitGold := p.Gold
	if healing == 0 {
		kitGold += p.StashGold
	}
	broke := r.recoveryActive() && kitGold < r.e.Tune.Resource.RecoveryKitGold
	if broke {
		trip = resource.Trip{} // recovery: a potion trip with an empty purse buys nothing (G1 attempt 9: 90 min of walking)
	}
	if known && !p.InTown() {
		// recovery: the purse reached one potion with none left; buy the minimal kit now, farm on for the rest of the target
		r.obMu.Lock()
		if healing > 0 {
			r.ob.recKitTrips = 0
		}
		capped := r.ob.recKitTrips >= r.e.Tune.Resource.KitTripCap // failing chores must not repeat the trip
		r.obMu.Unlock()
		if go2, why2 := resource.RecoveryKitTrip(resource.RecoveryKitInput{
			Recovering: r.recoveryActive(), InTown: false, Healing: healing, Gold: kitGold, Capped: capped,
			WayHome: r.resInput(s).HasTP || resource.ChainHomeArea(p.Area) || r.wpActive(s),
		}, resource.FromEnv(*r.e.Tune)); go2 {
			hard, emergency = true, true
			plan.Potions = true
			why = append(why, why2)
		}
	}
	if trip.Go {
		plan.Potions = true
		plan.Heal = true                                  // the policy's reasons include the free heal, which only runs when asked for
		hard = strings.Contains(trip.Reason, "free heal") // a potion-only resource trip is floorable
		why = append(why, "resource: "+trip.Reason)
		emergency = r.resInput(s).Belt.HP == 0
	}
	if why2, ok := r.kitTrip(s); ok && known {
		hard = true // not floorable: the kit is the survival need (KITGATELEVEL)
		plan.Potions = true
		why = append(why, why2)
		// no healing potion anywhere is the Gate 1 BELT window: the short emergency cooldown applies (tripDue)
		if k, kn := r.kitRead(s); kn && k.HP == 0 {
			emergency = true
		}
	}
	if known {
		threshold := t.ResupplyPotions
		if p.InTown() {
			threshold = t.ResupplyPotions * 2
		}
		// Recovery suppresses potion trips from the field (an empty purse buys nothing), not the visit already under way:
		// in town with no healing potion and gold for one the shop opens (potionFloor caps it, the kit gold stays).
		// Gate 1 20261103: left Rogue Encampment with 185 gold and 0 potions, 3 virtual minutes outside town unhealed.
		if healing < threshold && kitGold >= minPotionPrice && !trip.Futile && (!broke || (p.InTown() && healing == 0)) {
			plan.Potions = true
			why = append(why, fmt.Sprintf("%d healing potions", healing))
			emergency = healing == 0
		}
		if full >= t.InvFullPct && !r.packFullQuiet() {
			hard = true
			plan.Sell, plan.Stash, plan.Identify = true, true, true
			why = append(why, fmt.Sprintf("inventory %d%% full", full))
		}
	}
	// valuables were left on the ground for lack of room: make room (sell, stash, identify) and come back for them.
	// Pack >= 80 percent only, so a lone 2x3 item that does not fit a nearly empty pack cannot loop trips; the trip
	// rides the same gates as every routine trip (boss run, no-portal step, cooldown) in tripDue.
	if lb, ok := r.d.Loot.(leftBehinder); ok && known && !p.InTown() && full >= leftBehindPackPct && !r.packFullQuiet() {
		if n := lb.LeftBehind(); n > 0 {
			hard = true
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
		hard = true
		plan.Sell, plan.Potions = true, p.Gold >= minPotionPrice
		why = append(why, fmt.Sprintf("poverty: %d items to sell, %d gold", sell, p.Gold))
	} else if !p.InTown() && p.Gold < t.GearMinGold && lowGoldSellOK(r.sellable(s), r.resInput(s).HasTP, t.SellMinItems, sellMin) {
		hard = true
		plan.Sell, plan.Potions = true, p.Gold >= minPotionPrice
		why = append(why, fmt.Sprintf("%d items to sell, %d gold", r.sellable(s), p.Gold))
	}
	if p.InTown() && (p.LifePct() < 100 || p.ManaPct() < 100) {
		hard = true
		plan.Heal = true
		why = append(why, "heal")
	}
	if r.d.Town != nil && s.Inventory != nil && !r.cfg.IgnoreTownAdvice {
		if bad, reason := r.d.Town.NeedsTrip(s); bad {
			plan.Potions, plan.Sell, plan.Stash, plan.Identify = true, true, true, true
			why = append(why, reason)
			hard = hard || !restockReason(reason)
		}
	}
	if w := r.mercWanted(s); w != "" {
		hard = true
		why = append(why, "merc: "+w)
	}
	if len(why) == 0 {
		return plan, false, false
	}
	if !hard && !emergency && !p.InTown() && healing > 0 && !r.tripWorthIt(s) {
		return ChorePlan{}, false, false
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
		t.startBasis = r.resInput(snap).Gold
	}
	if t.repair && !t.begun {
		if !r.repairBegin(snap, t) {
			return false // not now: the request stays queued and the normal play goes on
		}
		r.rep.SetPhase(repair.PhaseGoing, "")
	}
	if !p.InTown() && t.stage < 3 {
		if t.stage > 0 { // fell out of town mid-trip (portal taken early): start over from the field
			t.stage = 0
		}
		if t.repair {
			r.setWhy(WhyRepairTrip, "step", "going home")
		} else {
			r.setWhy(WhyTownTrip, "why", t.plan.Reason)
		}
		ok := r.do(ctx, c, StateResupplying, "Going to town ("+t.plan.Reason+")", town, "to town", r.d.Town.ToTown)
		if !ok && strings.Contains(t.plan.Reason, resource.RecoveryKitReason) && r.State() == StateResupplying {
			// a failed way home counts like failed chores: without it the same trip is due again on the next tick
			r.obMu.Lock()
			r.ob.recKitTrips++
			r.obMu.Unlock()
		}
		if !ok && r.State() == StateResupplying {
			// no portal, no scroll, no luck: give the trip up (the cooldown keeps it from being retried at once)
			if l.failures >= 1 {
				r.e.Log.Warn("town trip abandoned", "reason", t.plan.Reason)
				r.repairTripEnd(t, "could not get to town")
				l.trip, l.failures = nil, 0
			}
		}
		return ok
	}
	switch t.stage {
	case 0, 1:
		t.stage = 1
		if t.repair {
			r.rep.SetPhase(repair.PhaseRepairing, "")
			r.setWhy(WhyRepairTrip, "step", "repairing in town")
		} else {
			r.setWhy(WhyTownChores, "why", t.plan.Reason)
		}
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
		if !ok && strings.Contains(t.plan.Reason, resource.RecoveryKitReason) && r.State() == StateResupplying {
			r.obMu.Lock()
			r.ob.recKitTrips++
			r.obMu.Unlock()
		}
		if !ok && strings.Contains(t.plan.Reason, "kit: ") && r.State() == StateResupplying {
			// a kit trip whose chores keep failing counts like a finished one, else it repeats until the failsafe ladder
			// skips the field and pauses the character far from the safe area (KITGATELEVEL)
			r.obMu.Lock()
			r.ob.kitTrips++
			capped := r.ob.kitTrips >= r.e.Tune.Resource.KitTripCap
			r.obMu.Unlock()
			if capped {
				r.e.Log.Warn("town trip abandoned", "reason", t.plan.Reason)
				l.trip, l.failures = nil, 0
			}
		}
		if ok && t.repair && r.repairRetry(t) {
			return true // the smith did not work this round: one more, then the request keeps its state
		}
		if ok {
			t.stage = 2
			if s := r.latest(); s != nil {
				r.mercChore(ctx, s)
			}
			r.setLastTrip(r.e.Clock.Now())
			if s := r.latest(); s != nil {
				r.noteTripEnd(s)
				r.noteTownFutile(t, s)
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
			r.repairTripEnd(t, "")
			l.trip = nil
			l.needBuff = true
		}
		return true
	default:
		if r.gateField(snap, t) {
			return true
		}
		if t.repair {
			r.setWhy(WhyRepairTrip, "step", "returning to the field")
		} else {
			r.setWhy(WhyBackToField)
		}
		ok := r.do(ctx, c, StateResupplying, "Returning to the field", sec(r.cfg.TravelTimeoutSec), "back to field", func(ctx context.Context) error {
			err := r.d.Town.BackToField(ctx, t.returnTo)
			if err == nil {
				r.waitForMerc(ctx, "portal")
			}
			return err
		})
		if ok || l.failures > 1 {
			r.repairTripEnd(t, "")
			l.trip = nil // on repeated failure travel() walks back by itself
			l.needBuff = true
		}
		return true
	}
}

// rest keeps the character safely in town (the idle goal, or a farm target that is a town).
func (r *Runner) rest(ctx context.Context, snap *d2.Snapshot, c control) bool {
	if !snap.Player.InTown() {
		r.setWhy(WhyToTown)
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

// lowGoldSellOK is the second sell-trip door (gold under GearMinGold): the item minimum, and with the povertySellMin
// knob set the poor-character rule of povertySellOK too. Knob 0 keeps the old rule.
func lowGoldSellOK(sell int, hasTP bool, minItems, min int) bool {
	return sell >= max(minItems, min) && (min == 0 || povertySellOK(sell, hasTP, min))
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

// routeBanned says h blocks a walk to target: banned, and not merely a nav-fault skip of another area. Such a skip
// (whyUnreachable, whyNavStuck) is no danger, and travelTo may jump over the area by waypoint (codereview3 C1).
func (r *Runner) routeBanned(h, target int) bool {
	if !r.banned(h) {
		return false
	}
	if h == target {
		return true
	}
	r.obMu.Lock()
	why := r.avoidWhy[h]
	r.obMu.Unlock()
	return why != whyUnreachable && why != whyNavStuck
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
		if h == s.Player.Area || r.d.Plan.IsTown(h) || !r.routeBanned(h, target) {
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
		if r.goldFarmOn() {
			if a, ok, stop := r.goldFarmTarget(s); stop {
				return 0, false
			} else if ok {
				l.stepArea = 0
				return a, true
			}
		}
		if r.recoveryActive() && !r.gearVerdict(s).Gated {
			r.updateStarvation(s)
			r.pushPoverty(true, recoveryPackDanger) // after updateStarvation, which pushes the lean 0
			if a, ok := r.recoveryArea(s); ok {
				l.stepArea = 0
				return a, true
			}
			if r.kitGateActive(s) {
				return 0, false
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
		if l.stepArea == areaDenOfEvil && p.Difficulty == d2.Normal && r.questClaimed("den_of_evil") {
			l.stepArea = 0 // the cached Den pick outlived its claim: 2026-11-12 17:34 walked back into the Den after the reward (hourfix2)
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
		farmed := r.farmExclude(s, ex)
		r.guardedAreasNoFarm(s, ex)
		t := r.e.Tune.Bot
		diff := difficultyName(p.Difficulty)
		if diff == "normal" && r.questClaimed("den_of_evil") {
			ex[areaDenOfEvil] = true // the reward is taken: the Den is no quest run any more (hourfix)
		}
		if r.denKitShort(s) && diff == "normal" {
			ex[areaDenOfEvil] = true // a dungeon quest needs a potion buffer; farm the open field first
		}
		pick := func() (Step, bool) {
			opts := PickOptions{Margin: t.LevelMargin, Slack: t.LeaveSlack, Acts: r.allowedActs(s), Exclude: ex, Cleared: cleared, Floor: worthFloor, FullKit: r.fullKit(s), Melee: MeleeTank(p.Class), XP: r.stepXP(p.Level, p.Difficulty)}
			st, ok := r.d.Plan.Pick(diff, p.Level, opts)
			if !ok && len(farmed) > 0 {
				// the farm is a preference: every farm area is banned or excluded, so pick among all areas again
				r.e.Log.Info("farm: restricted pick empty, falling back")
				for _, a := range farmed {
					delete(ex, a)
				}
				farmed = nil
				return r.d.Plan.Pick(diff, p.Level, opts)
			}
			return st, ok
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
		if ok && step.Area != areaDenOfEvil && diff == "normal" && !ex[areaDenOfEvil] && !r.questClaimed("den_of_evil") {
			// The Den is always taken before the fields behind it (a melee class enters Cold Plains at clvl 4, the Den
			// leaves at 6): data order alone would pick the later step in the overlap (hourfix).
			for _, ds := range r.d.Plan.Steps[diff] {
				if ds.Area == areaDenOfEvil && ds.enter(MeleeTank(p.Class))+t.LevelMargin <= p.Level && p.Level <= ds.LeaveLvl+t.LeaveSlack {
					step = ds
				}
			}
		}
		for ok {
			d := r.d.Plan.EntryAllowed(diff, p.Level, step.Area, nil)
			if d.OK {
				if why := levelGapRefusal(diff, p.Level, r.areaMLvl(step.Area, d2.Difficulty(rank(diff)))); why != "" {
					d = combat.EntryDecision{Why: why}
				}
			}
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
	ok := r.do(ctx, c, StateInTown, "Starting a "+want+" game", sec(r.cfg.TownTimeoutSec), "new game", r.newGameRun(func(ctx context.Context) error {
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
		r.setWhy(WhyAreaDone)
		return r.do(ctx, c, StateResupplying, "Area done, returning to town", sec(r.cfg.TownTimeoutSec), "to town", r.d.Town.ToTown)
	}
	why := "the target area is cleared or too dangerous"
	if c.spec.kind == goals.ReachLevel {
		why = "no eligible area is left in this game"
	}
	if r.d.Cycler != nil && len(l.cleared) > 0 {
		if hold := r.freshHold(s.Player); hold > 0 {
			r.setState(StateInTown, "Fresh game held back after a recovery ("+hold.Round(time.Second).String()+")")
			r.pause(ctx, 5*time.Second)
			return false
		}
		diff := c.spec.difficulty
		return r.do(ctx, c, StateInTown, "Starting a fresh game", sec(r.cfg.TownTimeoutSec), "new game", r.newGameRun(func(ctx context.Context) error {
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
	r.setWait("Waiting for a fresh game: "+why+" and "+reason, sec(r.cfg.ErrorRetrySec), goals.EvInfo)
	return false
}

// ---- traveling -----------------------------------------------------------------------------------

func (r *Runner) travel(ctx context.Context, s *d2.Snapshot, c control, target int) bool {
	name := r.areaName(target)
	r.setWhy(WhyWalk, "area", name, "level", itoa(s.Player.Level))
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
				r.waitForMerc(ctx, "waypoint")
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
	// futileGoneFor is the window when the Clear found nothing to set aside (Remaining 0): the hostile that made the
	// sweep call Clear died or left in between, so nothing was unreachable and a long window would only blind the
	// sweep to the next pack, a shaman's revive or a returning fleer.
	futileGoneFor = 3 * time.Second
)

// sweep is one pass in the target area: buff, fight what is near, pick up, explore toward more. When exploration runs
// out of unexplored ground with nothing hostile left, the area counts as cleared for this game.
func (r *Runner) sweep(ctx context.Context, s *d2.Snapshot, c control, area int) bool {
	name := r.areaName(area)
	if code, kv, ok := r.repairWhy(s); ok {
		r.setWhy(code, kv...)
	} else if have, want, on := r.goldFarmNumbers(s); on {
		r.setWhy(WhyFarmUser, "area", name, "have", itoa(have), "want", itoa(want))
	} else if sv := r.savingFor(s); sv.Item != "" {
		r.setWhy(WhySavingFor, "area", name, "item", sv.Item, "need", itoa(sv.Need))
	} else {
		r.setWhy(WhyFight, "area", name, "life", itoa(s.Player.LifePct()))
	}
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
	r.noteArea(area)
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
		res, err := r.d.Fighter.Clear(ctx, ClearOpts{Radius: radius, Interlude: r.goldGrab})
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
				window := futileFor
				if res.Remaining == 0 {
					window = futileGoneFor
				}
				l.futile, l.futilePos, l.futileUntil = true, input.Point{X: s.Player.X, Y: s.Player.Y}, r.e.Clock.Now().Add(window)
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
		r.walkBackLoot(ctx, area, hostile)
		if r.emptyAreaLeave(area) {
			r.goldBeforeLeaving(ctx, area, radius, hostile)
			return nil
		}
		if r.goldInReach(radius) {
			l.goldIdleUntil = r.e.Clock.Now().Add(goldIdleFor) // the pickup above took nothing it could reach; roam on
		}
		// Explore (a roam) runs for minutes until a hostile shows: the empty-area check must run inside it. A pile that
		// shows up on the way ends the leg too (goldprio): the next pass picks it up before the roam goes on.
		goldStop := false
		if err := r.d.Mover.Explore(ctx, func() bool {
			if hostile() || r.emptyAreaLeave(area) || r.exploredEnough(area) {
				return true
			}
			goldStop = r.goldInReach(radius)
			return goldStop
		}); err != nil {
			if r.exploreGaveUp(ctx, area, err) {
				r.goldBeforeLeaving(ctx, area, radius, hostile)
				return nil
			}
			return fmt.Errorf("explore: %w", err)
		}
		if goldStop && !hostile() {
			return nil // the sweep is called again and its pickup takes the pile
		}
		if r.loop.cleared[area] && r.loop.emptyArea == 0 {
			r.goldBeforeLeaving(ctx, area, radius, hostile)
			return nil // left as empty (emptyAreaLeave logged it)
		}
		if !hostile() {
			r.walkBackLoot(ctx, area, hostile) // end of the roam leg, before the area counts as cleared
			r.goldBeforeLeaving(ctx, area, radius, hostile)
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

// exploreGiveUp is how many explore failures in a row, in one area with no kill in between, make it count as emptied.
const exploreGiveUp = 2

// exploreGaveUp handles a failed explore (roam timeout, stuck, no path) in a quiet area. emptyAreaLeave cannot catch
// this: it is off by default, and its timer restarts on every area change and every pause over 90 s, which the
// Blood Moor <-> Cold Plains ping-pong of the g1 soak did every few minutes (docs/soakstall.md). The failures are
// counted per area instead, and a kill restarts the count. It marks the area cleared (the planner then takes the
// next area or the fresh-game path) and returns true. Quest and boss runs keep roaming. The planner sets a run (Area
// included) for every leveling step, so the guard is the run kind, not the area (Gate 1: 20 of 22 timeouts uncounted).
func (r *Runner) exploreGaveUp(ctx context.Context, area int, err error) bool {
	l := &r.loop
	if ctx.Err() != nil || l.keepRoaming(area) {
		return false
	}
	r.obMu.Lock()
	kills := r.ob.kills
	r.obMu.Unlock()
	if l.exploreFails == nil {
		l.exploreFails = map[int]exploreFail{}
	}
	f := l.exploreFails[area]
	if f.kills != kills {
		f = exploreFail{kills: kills}
	}
	f.n++
	l.exploreFails[area] = f
	if f.n < exploreGiveUp {
		return false
	}
	delete(l.exploreFails, area)
	l.cleared[area] = true
	r.e.Log.Info("area abandoned: exploring failed with no kill", "area", area, "name", r.areaName(area), "failures", f.n, "err", err)
	r.e.Log.Info("explore-giveup", "area", area, "failures", f.n, "kills", kills)
	r.logSweep(area, true)
	return true
}

// keepRoaming is true when the current run needs every target of area: a quest or boss run, not a plain clear run.
func (l *loopState) keepRoaming(area int) bool {
	return l.run != nil && l.run.Area == area && l.run.Kind != RunClear
}

// Ping-pong guard (docs/soakstall.md, the 17:27 window of Gate 1: 61 virtual minutes alternating Blood Moor and Cold
// Plains with no kill and no explore timeout, so exploreGaveUp never counted).
const (
	pingPongAlt  = 4               // area switches between the same two areas with no kill
	pingPongSpan = 5 * time.Minute // ... that took at least this long (virtual)
)

// A kill only restarts the record when it is progress: the experience gained since the record began must reach
// pingPongXPPerMin * clvl per virtual minute of the span (U: the clvl scales the floor with the XP a kill is worth
// in the areas of that level). Cluster C of gate1r2-20261101: 26 kills, 18 XP in 52 minutes at clvl 14, a floor of
// 14 per minute; a real farm gives tens of XP per kill.
const pingPongXPPerMin = 1.0

type pingPong struct {
	a, b  int // the two areas; a is the current one
	alt   int
	kills int
	xp    uint32 // experience when the record began
	since time.Time
}

// noteArea records the area of a sweep pass. After pingPongAlt switches between the same two areas with no kill, both
// count as emptied and the planner takes the fresh-game path. Quest and boss areas are never marked.
func (r *Runner) noteArea(area int) {
	l := &r.loop
	now := r.e.Clock.Now()
	r.obMu.Lock()
	kills, xp, level := r.ob.kills, r.ob.exp, r.ob.level
	r.obMu.Unlock()
	pp := &l.pp
	fresh := func(a int) { *pp = pingPong{a: a, kills: kills, xp: xp, since: now} }
	if pp.a == 0 {
		fresh(area)
		return
	}
	if pp.kills != kills {
		mins := now.Sub(pp.since).Minutes()
		if xp < pp.xp || float64(xp-pp.xp) >= pingPongXPPerMin*float64(max(level, 1))*max(mins, 1) {
			fresh(area)
			return
		}
		pp.kills = kills // a trickle of kills: the alternation count and span continue
	}
	if area == pp.a {
		return
	}
	if pp.alt > 0 && area != pp.b { // a third area: not a ping-pong
		fresh(area)
		return
	}
	pp.a, pp.b = area, pp.a
	pp.alt++
	if pp.alt < pingPongAlt || now.Sub(pp.since) < pingPongSpan || l.keepRoaming(pp.a) || l.keepRoaming(pp.b) {
		return
	}
	if l.cleared == nil {
		l.cleared = map[int]bool{}
	}
	l.cleared[pp.a], l.cleared[pp.b] = true, true
	r.e.Log.Info("area abandoned: ping-pong without a kill", "areas", []int{pp.a, pp.b}, "switches", pp.alt, "seconds", int(now.Sub(pp.since)/time.Second))
	r.logSweep(pp.a, true)
	*pp = pingPong{}
}

// emptyAreaLeave marks area cleared once the sweep went Config.EmptyAreaSec without a kill (KILLFIX: on d2d 7 of the 20
// longest kill gaps were 200-500 s of roaming an emptied area). Quest areas (the step's run is in this area) keep
// roaming: the quest needs every target. The planner then takes the next area or the fresh-game path.
func (r *Runner) emptyAreaLeave(area int) bool {
	l := &r.loop
	limit := r.cfg.EmptyAreaSec
	if limit == 0 && l.levelSlice {
		limit = r.e.Tune.Bot.ClearIdleMin * 60
	}
	if l.run != nil && l.run.Area == area {
		// A quest area keeps roaming for its targets, but a sweep that sees no hostile at all for bot.emptyFieldSec is
		// an emptied field, not a quest in progress (g2d seed 17: 12 minutes in Stony Field after a resupply trip).
		limit = r.e.Tune.Bot.EmptyFieldSec
		if s := r.latest(); limit <= 0 || s == nil || len(r.aliveHostiles(s)) > 0 {
			l.emptySince = time.Time{}
			return false
		}
	}
	if limit <= 0 {
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
	if now.Sub(l.emptySince) < sec(limit) {
		return false
	}
	l.cleared[area] = true
	l.emptyArea = 0
	r.noteNokill("emptyfield")
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
	r.e.Log.Info(fmt.Sprintf("sweep: coverage %d/%d points, kills %d, cleared=%t", v, m, r.loop.sweepKills[area], cleared), "area", area, "exploredPct", ExploredPercent(v, m))
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
		if r.noPortalLeave(ctx, p, c) {
			return true
		}
		if r.lowLifeWait(fmt.Sprintf("Life %d%% in area %d, no potion, and the way to town failed %d times", p.LifePct(), p.Area, l.healFails)) {
			return true
		}
	}
	var terr error
	r.setWhy(WhyHealTrip, "life", itoa(p.LifePct()), "line", itoa(r.resumeLifePct()))
	ok := r.do(ctx, c, StateResupplying, fmt.Sprintf("Going to town to heal (life %d%%)", p.LifePct()), sec(r.cfg.TownTimeoutSec), "heal trip", func(ctx context.Context) error {
		terr = r.d.Town.ToTown(ctx)
		return terr
	})
	if ok {
		l.noPortalSince = time.Time{}
	} else {
		l.healFails++
		if errors.Is(terr, town.ErrNoPortal) && l.noPortalSince.IsZero() {
			l.noPortalSince = r.e.Clock.Now()
		}
	}
	return true
}

// noPortalLeave is g2e F2 (docs/research/g2e-diagnosis.md section 2): in the field with no potion, low life and no
// way to town (no scroll, no walk fallback for this area) the hero used to wait lowLifeRetry five times and left only
// through the 240 s failsafe (seed 11, 11 belt minutes). A new game starts in town, which heals. It leaves once the
// way to town failed because no portal exists and bot.noPortalWaitSec passed since the first such failure. It
// reports whether it acted.
func (r *Runner) noPortalLeave(ctx context.Context, p *d2.Player, c control) bool {
	l := &r.loop
	wait := time.Duration(r.e.Tune.Bot.NoPortalWaitSec) * time.Second
	if l.noPortalSince.IsZero() || r.d.Cycler == nil || wait >= 240*time.Second || r.e.Clock.Now().Sub(l.noPortalSince) < wait {
		return false
	}
	r.e.Log.Warn("low life, no potion and no town portal: leaving the game", "life", p.LifePct(), "area", p.Area, "name", r.areaName(p.Area), "waited", r.e.Clock.Now().Sub(l.noPortalSince).Round(time.Second))
	l.noPortalSince, l.healFails, l.lowLifeUntil = time.Time{}, 0, time.Time{}
	if !r.do(ctx, c, StateRecovering, "No way to town: leaving and re-entering the game", sec(r.cfg.TownTimeoutSec), "new game", r.newGameRun(func(ctx context.Context) error {
		return r.d.Cycler.NewGame(ctx, difficultyName(p.Difficulty))
	})) {
		return false
	}
	r.forgetGame()
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
	r.setWhy(WhyWaitHeal, "why", why)
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
	in, rt := r.resInput(s), resource.FromEnv(*r.e.Tune)
	return r.askReturn(s, in, resource.FieldGate(in, rt))
}

// askReturn is point t1.return (docs/t0-slice.md): the field gate's Go is the answer until the ladder (when on) learns a
// rule for the context. A learned "stay in town" turns a Go into a restock round, which gateRestocks bounds; a learned
// "return" never lifts a refusal the gate gave (a poor character is still sent to its first field only).
func (r *Runner) askReturn(s *d2.Snapshot, in resource.Input, v resource.FieldVerdict) resource.FieldVerdict {
	l := ladder.Active()
	if l == nil {
		return v
	}
	kit := ladder.KitFor(in.HasTP, in.Belt.HP+in.Belt.RV+in.Belt.PackHP+in.Belt.PackRV)
	go2, err := l.ReturnWanted(ladder.HeroFrom(s.Player).With("have", v.HaveHP, "want", v.WantHP), kit, v.Go)
	if err != nil {
		l.Fault(err)
		return v
	}
	if v.Go && !go2 {
		return resource.FieldVerdict{Restock: true, HaveHP: v.HaveHP, WantHP: v.WantHP, Reason: "ladder: stay in town (" + v.Reason + ")"}
	}
	return v
}

// gateField applies resource.FieldGate before the trip returns to the field. It reports true when it changed the plan:
// Restock runs the chores once more (bounded), BloodMoor redirects a poor character to its first field, where the walk
// home works. A poor character is never sent shopping, so the two cannot loop.
func (r *Runner) gateField(s *d2.Snapshot, t *trip) bool {
	r.redirectStalePack(s, t)
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

// lootMemory is the optional part of a Looter that remembers drops beyond the pickup radius (loot.Looter,
// docs/lootmemory.md).
type lootMemory interface {
	ObserveGround(s *d2.Snapshot)
	ResetMemory()
	WalkBack(ctx context.Context, abort func() bool, why func(n, gold int, first string)) (int, error)
}

// LootWalking reports whether the runner is walking back for remembered drops now. Safe from other goroutines.
func (r *Runner) LootWalking() bool { return r.lootWalk.Load() }

// walkBackLoot fetches remembered drops at a quiet moment (nothing hostile near). A hostile showing up aborts it.
func (r *Runner) walkBackLoot(ctx context.Context, area int, hostile func() bool) {
	lm, ok := r.d.Loot.(lootMemory)
	if !ok || ctx.Err() != nil {
		return
	}
	r.lootWalk.Store(true)
	defer r.lootWalk.Store(false)
	_, err := lm.WalkBack(ctx, hostile, func(n, gold int, first string) {
		what := first
		if gold > 0 {
			what = itoa(gold) + " gold / " + first
		}
		r.e.Log.Info("loot_memory: going back for "+what, "drops", n, "area", r.areaName(area))
	})
	if err != nil && ctx.Err() == nil {
		r.e.Log.Warn("loot_memory: walk back failed", "err", err)
	}
}

// levelGapRefusal is combat.LevelGapOK (docs/difficulty.md item 23) for the planner's pick: Nightmare and Hell refuse
// an area whose monster level is too far above the character; Normal is never asked (its margin and the death
// spiral breaker are the older rules), so a Normal run is byte-for-byte the same. "" means allowed.
func levelGapRefusal(diff string, level, mlvl int) string {
	if diff == "normal" {
		return ""
	}
	if combat.LevelGapOK(d2.Difficulty(rank(diff)), level, mlvl) {
		return ""
	}
	return fmt.Sprintf("clvl %d too far below monster level %d", level, mlvl)
}

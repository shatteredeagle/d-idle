package bot

import (
	"context"
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/resource"
	"d2lod-bot/internal/task"
)

// Leveling pieces of the loop (docs/bot.md "Leveling"): spend points, wear better gear, farm gold when poor,
// the Run abstraction the planner works with, and the difficulty switch rules.

// minPotionPrice is the cheapest potion a vendor sells in Normal (Minor Healing Potion, VERIFIED LIVE: 30 gold).
// With less gold a potion trip cannot buy anything.
const minPotionPrice = 30

// ---- points --------------------------------------------------------------------------------------

func pointsPending(s *d2.Snapshot) bool {
	return s != nil && s.Player != nil && s.Player.StatPoints+s.Player.SkillPoints > 0
}

// pointsDue is the guard's check: points are unspent, nothing hostile is near and the retry gap has passed. It reads
// loop state that only the main goroutine writes, so it only looks at values that are stable while an action runs.
func (r *Runner) pointsDue(s *d2.Snapshot, now time.Time) bool {
	if r.d.Leveler == nil || !pointsPending(s) || r.loop.pointsFails >= 3 && r.loop.pointsFailLvl == s.Player.Level {
		return false
	}
	if t := r.loop.pointsTryAt; !t.IsZero() && now.Sub(t) < time.Duration(r.e.Tune.Bot.PointsRetrySec)*time.Second {
		return false
	}
	return r.hostileNear(s, r.e.Tune.Bot.ClearRadius) == 0
}

// pointsStep spends unspent stat and skill points through the Leveler: in town, or in the field while nothing hostile is
// near. It retries after PointsRetrySec when points are still unspent and gives up for the level after 3 attempts that
// spent nothing (the next level, or a new game, tries again). It reports whether it acted.
func (r *Runner) pointsStep(ctx context.Context, snap *d2.Snapshot, c control) bool {
	l := &r.loop
	p := snap.Player
	if r.d.Leveler == nil || !pointsPending(snap) {
		l.pointsFails = 0
		return false
	}
	if l.pointsFails >= 3 && l.pointsFailLvl == p.Level {
		return false
	}
	now := r.e.Clock.Now()
	if !l.pointsTryAt.IsZero() && now.Sub(l.pointsTryAt) < time.Duration(r.e.Tune.Bot.PointsRetrySec)*time.Second {
		return false
	}
	if !p.InTown() && r.hostileNear(snap, r.e.Tune.Bot.ClearRadius) > 0 {
		return false // never while a fight is on
	}
	l.pointsTryAt = now
	before := p.StatPoints + p.SkillPoints
	r.do(ctx, c, StateInTown, "Spending stat and skill points", sec(r.cfg.TownTimeoutSec), "level up", func(ctx context.Context) error {
		return r.d.Leveler.Spend(ctx, snap)
	})
	after := before
	if s := r.poll(); s != nil && s.Player != nil {
		after = s.Player.StatPoints + s.Player.SkillPoints
	}
	switch {
	case after == 0:
		l.pointsFails, l.pointsTryAt = 0, time.Time{}
	case after < before: // partial: try the rest soon
		l.pointsFails, l.pointsTryAt = 0, time.Time{}
	default:
		l.pointsFails++
		l.pointsFailLvl = p.Level
		r.e.Log.Warn("points not spent", "attempt", l.pointsFails, "stat", p.StatPoints, "skill", p.SkillPoints)
	}
	return true
}

// ---- gear ----------------------------------------------------------------------------------------

// gearStep runs in town, at most every GearCheckSec: equip better items from the pack, and shop when there is gold
// (once per character level or when the gold doubled since the last pass). It reports whether it acted.
func (r *Runner) gearStep(ctx context.Context, snap *d2.Snapshot, c control) bool {
	l := &r.loop
	p := snap.Player
	if r.d.Gear == nil || !p.InTown() {
		return false
	}
	t := r.e.Tune.Bot
	now := r.e.Clock.Now()
	if !l.gearAt.IsZero() && now.Sub(l.gearAt) < time.Duration(t.GearCheckSec)*time.Second {
		return false
	}
	swaps := r.d.Gear.PendingSwaps(snap)
	budget := 0
	// Early-gear policy (docs/sme/early-gear-policy.md): below earlyGearMax gold goes to the scroll and potions (the chores
	// and the weapon rebuy), never to vendor armor or magic items; wearing what dropped (swaps) stays free.
	early := resource.Early(p.Level, r.e.Tune.Resource.EarlyGearMax)
	if !early && p.Gold >= t.GearMinGold && (p.Level != l.shopLevel || p.Gold >= 2*l.shopGold+t.GearMinGold) {
		budget = (p.Gold - t.GearReserve) * t.GearBudgetPct / 100
	}
	if swaps == 0 && budget < 50 {
		return false
	}
	l.gearAt = now
	if budget >= 50 {
		l.shopLevel, l.shopGold = p.Level, p.Gold
	}
	r.do(ctx, c, StateInTown, "Gear: equip and shop", sec(r.cfg.TownTimeoutSec), "gear", func(ctx context.Context) error {
		// Failures here are logged, never escalated: gear is an upgrade, not a requirement.
		if budget >= 50 {
			rep, err := r.d.Gear.Shop(ctx, budget)
			r.e.Log.Info("gear shop", "budget", budget, "bought", rep.Bought, "spent", rep.Spent, "err", err)
		}
		rep, err := r.d.Gear.Equip(ctx)
		r.e.Log.Info("gear equip", "swaps", rep.Swaps, "err", err)
		return ctx.Err()
	})
	return true
}

// ---- poverty -------------------------------------------------------------------------------------

// updateStarvation is the poverty hysteresis (resource.PovertyCheck): a character without the minimal kit (a town
// portal and HPFirst healing potions) and with less gold than it costs is in poverty mode until the gold covers it.
// It logs one "poverty mode" line per state change, with the reason.
func (r *Runner) updateStarvation(s *d2.Snapshot) bool {
	l := &r.loop
	if _, _, known := r.beltAndInv(s); !known {
		return false
	}
	in := r.resInput(s)
	v := resource.PovertyCheck(in, resource.FromEnv(*r.e.Tune), l.starved)
	adv := resource.PovertyAdvice(in, v, r.e.Tune.Bot.SellMinItems)
	switch {
	case v.On && !l.starved:
		l.starveNote = false
		r.e.Log.Warn("poverty mode: on", "reason", v.Reason, "gold", s.Player.Gold, "missing", v.Missing)
	case !v.On && l.starved:
		r.e.Log.Info("poverty mode: off", "reason", v.Reason, "gold", s.Player.Gold)
	}
	l.starved = v.On
	r.obMu.Lock()
	r.ob.poor = v.On
	if adv.Action != r.ob.povertyPlan {
		r.e.Log.Info("poverty advice", "action", adv.Action.String(), "reason", adv.Reason)
	}
	r.ob.povertyPlan, r.ob.maxPackDanger = adv.Action, adv.MaxPackDanger
	if !v.On {
		r.ob.sellLeft = 0
	}
	r.obMu.Unlock()
	r.pushPoverty(v.On, adv.MaxPackDanger)
	return v.On
}

func (r *Runner) inPoverty() bool {
	r.obMu.Lock()
	defer r.obMu.Unlock()
	return r.ob.poor || r.ob.recovering
}

// starveArea is the area a poor character farms. Without a town portal the way back to town is a walk, which only
// works from the first field area of an act (town.townNeighbour), so that area is the only choice; with a portal it is
// the plan area for a character PovertyLevels lower, which the planner already keeps clear of dangerous ground. ok is
// false when nothing is open (cleared in this game or avoided).
func (r *Runner) starveArea(s *d2.Snapshot) (int, bool) {
	diff := difficultyName(s.Player.Difficulty)
	if r.resInput(s).HasTP {
		ex := map[int]bool{}
		for a := range r.loop.cleared {
			ex[a] = true
		}
		for a := range r.avoidedNow() {
			ex[a] = true
		}
		r.guardedAreas(s, ex)
		t := r.e.Tune.Bot
		if st, ok := r.d.Plan.Pick(diff, s.Player.Level-r.e.Tune.Resource.PovertyLevels, PickOptions{Margin: t.LevelMargin, Slack: t.LeaveSlack, Acts: r.allowedActs(s), Exclude: ex, FullKit: r.fullKit(s), Melee: MeleeTank(s.Player.Class)}); ok {
			return st.Area, true
		}
	}
	return r.starveStep(s, diff)
}

// avoidedNow lists the areas the death-spiral breaker currently bans.
func (r *Runner) avoidedNow() map[int]bool {
	m := map[int]bool{}
	r.obMu.Lock()
	defer r.obMu.Unlock()
	for a, until := range r.avoid {
		if until.After(r.e.Clock.Now()) {
			m[a] = true
		}
	}
	return m
}

// sellable counts pack items the Looter would sell (a starving character carries loot it could turn into gold).
func (r *Runner) sellable(s *d2.Snapshot) int {
	if r.d.Loot == nil || s.Inventory == nil {
		return 0
	}
	n := 0
	ue, _ := r.d.Loot.(UnitEvaluator)
	for i := range s.Inventory.Items {
		u := &s.Inventory.Items[i]
		if u.Item == nil || u.Item.Where != d2.ContInventory {
			continue
		}
		v := r.d.Loot.Evaluate(*u.Item)
		if ue != nil {
			v = ue.EvaluateUnit(*u, s.Player.Level, s.Player.Gold)
		}
		if v == VerdictSell {
			n++
		}
	}
	return n
}

// ---- runs ----------------------------------------------------------------------------------------

// RunKind says what a Run asks of the bot.
type RunKind string

const (
	RunClear RunKind = "clear" // explore and kill everything in the area
	RunQuest RunKind = "quest" // clear a quest area, then claim the reward in town
	RunBoss  RunKind = "boss"  // clear the area on the way to a named boss (the boss itself is killed by the sweep, UNVERIFIED)
)

// LevelRun is one unit of leveling work the planner picks: a place, what to do there, and what follows. The planner
// (Plan.Pick) chooses a Step by character level; runFor turns the Step into a Run.
type LevelRun struct {
	Key   string
	Name  string
	Kind  RunKind
	Area  int
	Quest string // RunQuest: quest id of data/quest-points.json
	Boss  string // RunBoss
}

// Special areas of the Normal plan. Ids are levels.txt ids (Plan data); the quest and boss names come from
// quest-points.json and the Act I plan.
const (
	areaDenOfEvil    = 8
	areaBurialGround = 17
)

func (r *Runner) runFor(diff string, st Step) LevelRun {
	run := LevelRun{Key: fmt.Sprintf("clear:%d", st.Area), Name: st.Name, Kind: RunClear, Area: st.Area}
	switch {
	case diff == "normal" && st.Area == areaDenOfEvil:
		run.Key, run.Kind, run.Quest = "den_of_evil", RunQuest, "den_of_evil"
	case diff == "normal" && st.Area == areaBurialGround:
		run.Key, run.Kind, run.Boss = "blood_raven", RunBoss, "Blood Raven"
	}
	return run
}

// Quest objective gating (found live on the Paladin, Assassin and Druid: every Den run was declared cleared by the sweep
// while monsters were left, and the reward claim then failed).
const (
	questSweepRetries = 3               // sweeps that may end with the objective open before the area is given up
	claimBackoff      = 5 * time.Minute // after a failed claim trip or claim, no new attempt for this long
)

// questObjectiveMet reads the live quest word (the reader must see in-session changes). known is false without such a
// reader, then the sweep result is all there is. A quest the reader lists as Cleared or Done counts as met.
func (r *Runner) questObjectiveMet(q string) (met, known bool) {
	td := r.d.Task
	if td == nil || td.Quests == nil || !td.QuestsLive {
		return false, false
	}
	s := r.latest()
	if s == nil || s.Player == nil {
		return false, false
	}
	v, ok := td.Quests.Read(int(s.Player.Difficulty))
	if !ok {
		return false, false
	}
	st, has := v.Quests[q]
	return has && st.Stage >= task.StageCleared, true
}

// sweepEndsQuest decides whether an exhausted sweep of the run's area really completes the run. A quest run needs the
// quest word (Den of Evil: "all monsters dead") when a live reader exists: otherwise the bot walks to a leftover monster
// (the fighter sets some aside) a few times and then gives the area up without claiming (ended true, complete false).
func (r *Runner) sweepEndsQuest(ctx context.Context, run *LevelRun) (ended, complete bool) {
	if run == nil || run.Kind != RunQuest {
		return true, true
	}
	met, known := r.questObjectiveMet(run.Quest)
	if !known || met {
		return true, true
	}
	l := &r.loop
	if l.questTries == nil {
		l.questTries = map[string]int{}
	}
	l.questTries[run.Quest]++
	if l.questTries[run.Quest] > questSweepRetries {
		r.e.Log.Warn("quest area explored but the quest is not complete; giving it up", "quest", run.Quest, "area", run.Area)
		return true, false
	}
	if s := r.latest(); s != nil {
		if left := r.aliveHostiles(s); len(left) > 0 {
			u := left[0]
			r.e.Log.Info("quest objective open: hunting a leftover", "quest", run.Quest, "left", len(left), "x", u.X, "y", u.Y)
			l.futile = false
			_ = r.d.Mover.GoTo(ctx, input.Point{X: u.X, Y: u.Y})
		}
	}
	return false, false
}

// runFinished is called when the sweep declared the run's area cleared in this game.
func (r *Runner) runFinished(run *LevelRun) {
	if run == nil {
		return
	}
	l := &r.loop
	r.e.Log.Info("run finished", "run", run.Key, "kind", run.Kind, "area", run.Area)
	if run.Kind == RunQuest && !l.questDone[run.Quest] {
		if l.questPending == nil {
			l.questPending = map[string]bool{}
		}
		l.questPending[run.Quest] = true
	}
}

// claimStep claims pending quest rewards in town. Without a Quester the reward is logged as unclaimed (stub).
func (r *Runner) claimStep(ctx context.Context, snap *d2.Snapshot, c control) bool {
	l := &r.loop
	if len(l.questPending) == 0 || !snap.Player.InTown() {
		return false
	}
	for q := range l.questPending {
		if met, known := r.questObjectiveMet(q); known && !met {
			r.e.Log.Info("quest reward not claimed: the quest word says the objective is open", "quest", q)
			delete(l.questPending, q)
			continue
		}
		if r.d.Quester == nil {
			r.e.Log.Warn("quest reward not claimed: no quester wired (stub, UNVERIFIED)", "quest", q)
			delete(l.questPending, q)
			return false
		}
		ok := r.do(ctx, c, StateInTown, "Claiming the "+q+" reward", sec(r.cfg.TownTimeoutSec), "claim "+q, func(ctx context.Context) error {
			return r.d.Quester.Claim(ctx, q)
		})
		if ok {
			if l.questDone == nil {
				l.questDone = map[string]bool{}
			}
			l.questDone[q] = true
			delete(l.questPending, q)
			r.e.Log.Info("quest claimed", "quest", q)
		} else {
			delete(l.questPending, q) // one attempt per clear; the next fresh game's run tries again
			l.claimSkipUntil = r.e.Clock.Now().Add(claimBackoff)
		}
		return true
	}
	return false
}

// ---- difficulty ----------------------------------------------------------------------------------

// wantedDifficulty is the auto rule for reach_level: go up to the highest difficulty whose first plan step the
// character qualifies for (LevelMargin extra levels), unless the bot recently failed to get there or died too often.
// It never asks to go down. "" means stay.
//
// The game only offers Nightmare to a character that finished Normal and Hell to one that finished Nightmare; the bot
// cannot read that (UNVERIFIED quest flags), so the switch is an attempt: after the new game it checks the real
// difficulty and, if it did not change, bans the wish for DifficultyBanMin minutes.
func (r *Runner) wantedDifficulty(s *d2.Snapshot) string {
	p := s.Player
	raw := leveler.RawResists(p)
	want := r.d.Plan.DifficultyFor(p.Level, r.e.Tune.Bot.LevelMargin, &raw) // a failed resist gate keeps farming here
	if rank(want) <= int(p.Difficulty) {
		return ""
	}
	if until, ok := r.loop.diffBan[want]; ok && r.e.Clock.Now().Before(until) {
		return ""
	}
	r.obMu.Lock()
	deaths := r.dlog.since(r.e.Clock.Now().Add(-time.Hour), 0)
	r.obMu.Unlock()
	if deaths >= max(1, r.e.Tune.Bot.PauseDeathsPerHour/2) {
		return "" // too shaky to take on a harder difficulty
	}
	return want
}

// verifyDifficulty runs after a NewGame asked for want: it bans the wish when the game stayed in the old difficulty.
func (r *Runner) verifyDifficulty(want string) {
	deadline := r.e.Clock.Now().Add(20 * time.Second)
	for r.e.Clock.Now().Before(deadline) {
		if s := r.poll(); s != nil && s.Player != nil {
			if rank(want) == int(s.Player.Difficulty) {
				r.e.Log.Info("difficulty switched", "to", want)
				return
			}
			break
		}
		r.e.Clock.Sleep(300 * time.Millisecond)
	}
	if r.loop.diffBan == nil {
		r.loop.diffBan = map[string]time.Time{}
	}
	r.loop.diffBan[want] = r.e.Clock.Now().Add(time.Duration(r.e.Tune.Bot.DifficultyBanMin) * time.Minute)
	r.e.Log.Warn("difficulty switch did not happen; staying put", "want", want, "banMin", r.e.Tune.Bot.DifficultyBanMin)
}

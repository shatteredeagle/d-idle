package bot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/resource"
	"d2lod-bot/internal/task"
)

// The recovery phase (docs/recovery.md): running out of gold is not a failure. A broke character pauses its quest step,
// farms gold and sellable loot in a safe area, restocks and returns. The state machine is resource.RecoveryCheck; this
// file keeps its bookkeeping, picks the farm area and runs the farming slice in place of the campaign's quest step.

// brokeState is the recovery bookkeeping of the Run goroutine.
type brokeState struct {
	on        bool
	since     time.Time
	lastEnd   time.Time
	startGold int
	startXP   int
	lastLog   time.Time
	loot      [3]int // looter totals at the last progress line
	cycles    int
	areaNote  int
}

// safeTier is the Normal act 1 farm list of a broke character (SME docs/sme/play-economy.md): Blood Moor, Cold Plains,
// Stony Field, Den of Evil, Cave 1 and 2. Burial Grounds, the Crypt and Tristram wait until the kit is bought.
var safeTier = map[int]bool{2: true, 3: true, 4: true, areaDenOfEvil: true, 9: true, 13: true}

// recoveryPackDanger is the pack danger a recovering character still engages without potions: trash only. Poverty mode's
// 0 refused every pack of Cave Level 1 (G1 attempt 9: kills=0, gold 37 for 90 minutes), so no gold ever came in.
const recoveryPackDanger = 1

// recoveryLooter is the part of the Looter the recovery phase drives (loot.Looter.SetRecovering, RecoveryTotals).
type recoveryLooter interface {
	SetRecovering(on bool)
	RecoveryTotals() (picked, gold, full int)
}

func (r *Runner) lootTotals() (t [3]int) {
	if rl, ok := r.d.Loot.(recoveryLooter); ok {
		t[0], t[1], t[2] = rl.RecoveryTotals()
	}
	return t
}

// updateRecovery advances the state machine once per goal tick. It logs the edges and a progress line per minute.
func (r *Runner) updateRecovery(s *d2.Snapshot) bool {
	b := &r.loop.broke
	if s == nil || s.Player == nil {
		return b.on
	}
	if _, _, known := r.beltAndInv(s); !known {
		return b.on
	}
	in := r.resInput(s)
	t := resource.FromEnv(*r.e.Tune)
	healing, mana := in.Belt.HP+in.Belt.RV, in.Belt.MP
	r.obMu.Lock()
	streak := r.dlog.since(r.e.Clock.Now().Add(-resource.RecoveryForceMin*time.Minute), 0)
	r.obMu.Unlock()
	v := resource.RecoveryCheck(resource.RecoveryInput{
		Gold: s.Player.Gold, Shortfall: resource.KitShortfall(in.HasTP, healing, mana, t), Sellable: in.Sellable,
		SellMin: r.e.Tune.Bot.SellMinItems, Now: r.e.Clock.Now(), Since: b.since, LastEnd: b.lastEnd, Was: b.on,
		Force: streak >= resource.RecoveryForceDeaths,
	}, t)
	now := r.e.Clock.Now()
	switch {
	case v.Start:
		*b = brokeState{on: true, since: now, lastEnd: b.lastEnd, startGold: s.Player.Gold, startXP: int(s.Player.Exp), lastLog: now, cycles: b.cycles + 1, loot: r.lootTotals()}
		r.e.Log.Warn(fmt.Sprintf("recovery: start (gold %d < kit %d)", s.Player.Gold, t.RecoveryKitGold), "level", s.Player.Level, "cycle", b.cycles)
	case v.End:
		mins := max(int(now.Sub(b.since).Minutes()), 1)
		gain := s.Player.Gold - b.startGold
		r.e.Log.Info(fmt.Sprintf("recovery: end (gold %d, reason %s)", s.Player.Gold, v.Reason), "level", s.Player.Level)
		r.e.Log.Info(fmt.Sprintf("recovery: gold %+d in %d min", gain, mins), "goldPerHour", gain*60/mins, "xpGained", int(s.Player.Exp)-b.startXP)
		b.on, b.lastEnd = false, now
		r.loop.stepArea = 0 // the quest planner picks again
	case b.on && now.Sub(b.lastLog) >= time.Minute:
		b.lastLog = now
		lt := r.lootTotals()
		r.e.Log.Info(fmt.Sprintf("recovery: loot +%d items, +%d gold in piles, %d left for lack of room", lt[0]-b.loot[0], lt[1]-b.loot[1], lt[2]-b.loot[2]),
			"inventoryPct", r.packPct(s))
		b.loot = lt
		mins := max(int(now.Sub(b.since).Minutes()), 1)
		gain := s.Player.Gold - b.startGold
		r.e.Log.Info(fmt.Sprintf("recovery: gold %+d in %d min", gain, mins), "gold", s.Player.Gold, "goldPerHour", gain*60/mins, "sellable", in.Sellable, "area", s.Player.Area)
	}
	if rl, ok := r.d.Loot.(recoveryLooter); ok {
		rl.SetRecovering(b.on)
	}
	r.obMu.Lock()
	r.ob.recovering = b.on
	r.obMu.Unlock()
	if b.on {
		r.pushPoverty(true, recoveryPackDanger)
	}
	return b.on
}

// packPct is the share of backpack cells in use (0 when unknown).
func (r *Runner) packPct(s *d2.Snapshot) int {
	_, full, _ := r.beltAndInv(s)
	return full
}

// potionFloor is the healing potion cap of a town visit: while recovering with the kit unaffordable the last gold is for
// the kit (portal first), not a full belt. 0 = the normal belt plan.
func (r *Runner) potionFloor(p *d2.Player) int {
	if r.recoveryActive() && p.Gold < r.e.Tune.Resource.RecoveryKitGold {
		return max(r.e.Tune.Resource.RecoveryPotionFloor, 0)
	}
	return 0
}

func (r *Runner) recoveryActive() bool { return r.loop.broke.on }

// recoveryArea is the farm area of a recovering character: the plan area of a character PovertyLevels lower, restricted
// to the safe tier in Normal act 1. Without a town portal only the first field qualifies (the walk home works there).
func (r *Runner) recoveryArea(s *d2.Snapshot) (int, bool) {
	diff := difficultyName(s.Player.Difficulty)
	if !r.resInput(s).HasTP {
		return r.starveStep(s, diff)
	}
	ex := map[int]bool{}
	for a := range r.loop.cleared {
		ex[a] = true
	}
	for a := range r.avoidedNow() {
		ex[a] = true
	}
	r.guardedAreas(s, ex)
	if diff == "normal" && s.Player.Act == 0 {
		for _, st := range r.d.Plan.Steps[diff] {
			if !safeTier[st.Area] {
				ex[st.Area] = true
			}
		}
	}
	if healing, _, known := r.beltAndInv(s); known && healing < 2 {
		ex[areaDenOfEvil] = true // a dungeon needs a potion buffer
	}
	if diff == "normal" && s.Player.Act == 0 && r.e.Tune.Resource.RecoveryXPPick != 0 {
		if a, ok := r.recoveryXPArea(s, ex); ok {
			return a, true
		}
	}
	t := r.e.Tune.Bot
	st, ok := r.d.Plan.Pick(diff, s.Player.Level-r.e.Tune.Resource.PovertyLevels, PickOptions{Margin: t.LevelMargin, Slack: t.LeaveSlack, Acts: r.allowedActs(s), Exclude: ex})
	if !ok {
		return r.starveStep(s, diff)
	}
	if r.loop.broke.areaNote != st.Area {
		r.loop.broke.areaNote = st.Area
		r.e.Log.Info("recovery: farming", "area", st.Area, "name", r.areaName(st.Area), "gold", s.Player.Gold)
	}
	return st.Area, true
}

// recoveryStep replaces the campaign's quest step while recovery runs: one leveling slice in the recovery area.
func (r *Runner) recoveryStep(ctx context.Context, snap *d2.Snapshot, c control) bool {
	n := snap.Player.Level + 1
	return r.do(ctx, c, StateClearing, "Recovery: farming gold", 40*time.Minute, "recovery", func(ctx context.Context) error {
		err := taskActs{r, c}.LevelTo(ctx, n)
		if err != nil && strings.Contains(err.Error(), noAreaMsg) {
			// no farm area is eligible (bans after deaths): that is a planning block, not a fault. Wait at home until a ban ends.
			if s := r.latest(); s != nil && !s.Player.InTown() && r.d.Town != nil {
				if terr := r.d.Town.ToTown(ctx); terr != nil {
					return terr
				}
			}
			return fmt.Errorf("%w: %v", errAreaGated, err)
		}
		return err
	})
}

// recoveryGoal reports whether the goal is one recovery applies to: campaigns and leveling. Boss-run farms make their
// own gold and are left alone.
func recoveryGoal(c control) bool {
	return c.spec.camp != nil || c.spec.kind == goals.ReachLevel
}

// ---- Blood Raven budget ----------------------------------------------------------------------------

// brState tracks the Blood Raven attempts. A caster at clvl 9 portalled out of her fight 14 times for 2160 gold (V, G1
// attempt 8 log). Rules (plan-v2 10.1, SME): every attempt needs the kit and the gold (task.BRLine), at most BRAttempts
// attempts, and a failed attempt waits for clvl+1 (+2 after the second failure) before the next one, never a same-level retry.
type brState struct {
	attempts int
	lastLvl  int
	fails    int // failed attempts since the last kill or renewal
	waitLvl  int // clvl a failed attempt waits for; feeds World.BRWaitLevel so the planner levels meanwhile
	noted    bool
	gateNote string
	start    time.Time
	startGld int
	active   bool
}

// brAttempt is called at the start of every Blood Raven fight with the world it starts in. A non-nil error refuses it: the
// quest step is blocked (a danger fault) and the planner levels or restocks meanwhile.
func (r *Runner) brAttempt(w task.World) error {
	t := r.e.Tune.Resource
	b := &r.loop.br
	level := w.Level
	if line := task.BRLine(w); line != b.gateNote {
		b.gateNote = line
		r.e.Log.Info(line, "level", level, "gold", w.Gold)
	}
	if level < b.waitLvl {
		return fmt.Errorf("blood raven: last attempt failed, wait for clvl %d: %w", b.waitLvl, errAreaGated)
	}
	if g, ok := task.GateFor(task.QBurial); ok {
		if why := g.NotReady(w); why != "" {
			return fmt.Errorf("blood raven: not ready (%s): %w", why, errAreaGated)
		}
	}
	if t.BRAttempts > 0 && b.attempts >= t.BRAttempts {
		if level < b.lastLvl+t.BRLevelGap {
			if !b.noted {
				b.noted = true
				r.e.Log.Info(fmt.Sprintf("blood raven: budget spent (%d/%d), waiting for clvl %d", b.attempts, t.BRAttempts, b.lastLvl+t.BRLevelGap), "level", level)
			}
			return fmt.Errorf("blood raven: %d attempts spent, wait for clvl %d: %w", b.attempts, b.lastLvl+t.BRLevelGap, errAreaGated)
		}
		b.attempts, b.fails, b.noted = 0, 0, false
		r.e.Log.Info("blood raven: budget renewed", "level", level)
	}
	b.attempts++
	b.lastLvl = level
	b.start, b.startGld, b.active = r.e.Clock.Now(), w.Gold, true
	r.e.Log.Info(fmt.Sprintf("blood raven: attempt %d/%d start", b.attempts, t.BRAttempts), "level", level, "gold", w.Gold)
	return nil
}

// brEnd closes the attempt brAttempt opened: log reason, time and gold, and on a failure set the level to wait for.
// A cancelled run (pause, shutdown, re-plan) is not the character's failure: no wait and the attempt is refunded.
func (r *Runner) brEnd(err error, level, gold int) {
	b := &r.loop.br
	if !b.active {
		return
	}
	b.active = false
	reason, secs := "killed", int(r.e.Clock.Now().Sub(b.start).Seconds())
	if err != nil {
		reason = err.Error()
		switch {
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			reason = "cancelled"
			b.attempts-- // not the character's failure: a pause or re-plan must not spend the budget
		case errors.Is(err, ErrPortalOpened):
			reason = "retreat (town portal)"
			fallthrough
		default:
			b.fails++
			b.waitLvl = level + min(b.fails, 2)
		}
	} else {
		b.fails, b.waitLvl = 0, 0
	}
	n := b.attempts
	if len(reason) > 80 {
		reason = reason[:80]
	}
	r.e.Log.Info(fmt.Sprintf("blood raven: attempt %d/%d end(%s)", n, r.e.Tune.Resource.BRAttempts, reason),
		"secs", secs, "goldBefore", b.startGld, "goldAfter", gold, "level", level, "waitLevel", b.waitLvl)
}

// brFinish ends the open attempt with the level and gold the character has now.
func (r *Runner) brFinish(err error) {
	level, gold := 0, 0
	if s := r.latest(); s != nil && s.Player != nil {
		level, gold = s.Player.Level, s.Player.Gold
	}
	r.brEnd(err, level, gold)
}

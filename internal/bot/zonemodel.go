package bot

import (
	"context"
	"fmt"
	"sync"
	"time"

	"d2lod-bot/internal/d2"
)

// Survival slice S-D (docs/survival/zonemodel.md): with bot.zoneModel 1 the survival controller chooses the leveling
// zone among the plan's zones. The plan tables stay the candidate list and the fallback.

// zoneDialIDs maps bot.zoneDial to the dial ids of internal/power/profiles.json.
var zoneDialIDs = []string{"cautious", "balanced", "bold"}

// zoneRecheck is the dwell: a chosen zone is kept this long before the model is asked again (design 11: minimum dwell).
const zoneRecheck = 3 * time.Minute

// ZoneVerdict is the controller's decision as the bot needs it. Margin is a word (safe, risky, too hard).
type ZoneVerdict struct {
	Area       int
	Fresh      bool   // the best zone is cleared and a fresh game beats walking to a poorer one
	Hole       bool   // nothing is safe: the easiest field
	Hold       bool   // the hole's zone has no margin at all: never walked to
	Margin     string // word
	Line       string // the `zone choice:` log line
	Why        string // sentence for the Status line
	MercDead   bool   // the merc is dead and the controller judged the revive
	MercRevive bool   // reviving pays; false with MercDead: leave him dead
	MercWhy    string
	MercGain   float64 // value the revive adds until he dies again (same units as MercCost)
	MercCost   float64
	MercZone   string  // the zone the gain is measured in
	K          float64 // the ledger correction (k_zone x k_hero) of the zone the hero is in, for the `zone reask` line; 0 = unknown
}

// zoneState is the last verdict, for the merc policy and the Status word.
type zoneState struct {
	mu     sync.Mutex
	v      ZoneVerdict
	at     time.Time
	have   bool
	noted  string // merc note already logged
	failed string // last error logged
	reask  zoneReask
	burn   map[int]zoneBurn     // zones that just cost the hero something (zonerecover)
	recov  map[int]zoneHold     // answers of the recovery/return questions, kept for recoverHold
	noted1 map[string]time.Time // once-per-hour notes (dropped unreachable area, hole fallback, empty field), by key
}

// Cost-event re-ask (card survreask). A chicken, a death, a merc death or a potion crisis feeds the ledger at once, but
// the cached pick used to survive until the dwell ended (seed 149: six chickens in 35 s in one zone). Such an event
// now makes the next leveling pick ask the model again, at most reaskMax times per reaskWindow and not closer than
// reaskGap; the model's own hysteresis (switch margin, step-down rows) still decides whether to leave.
const (
	reaskGap     = 5 * time.Second
	reaskWindow  = 10 * time.Minute
	reaskMax     = 6
	crisisN      = 2 // emergency (rejuvenation) drinks within crisisWindow count as one cost event
	crisisWindow = time.Minute
)

type zoneReask struct {
	pending string      // the reason of the oldest unanswered cost event
	stamps  []time.Time // re-asks made, pruned to reaskWindow
	pots    []time.Time
}

// allowed prunes the window and says whether a re-ask may run now.
func (q *zoneReask) allowed(now time.Time) bool {
	keep := q.stamps[:0]
	for _, t := range q.stamps {
		if now.Sub(t) < reaskWindow {
			keep = append(keep, t)
		}
	}
	q.stamps = keep
	if n := len(keep); n > 0 && now.Sub(keep[n-1]) < reaskGap {
		return false
	}
	return len(keep) < reaskMax
}

// zoneCost records a cost event; the next leveling pick re-asks the model. No effect with bot.zoneModel 0.
func (r *Runner) zoneCost(reason string) {
	if r.e == nil || !r.zoneModelOn() {
		return
	}
	z := &r.zst
	z.mu.Lock()
	defer z.mu.Unlock()
	if z.reask.pending == "" {
		z.reask.pending = reason
	}
}

// zonePotion counts an emergency potion; crisisN of them within crisisWindow are a cost event.
func (r *Runner) zonePotion() {
	if r.e == nil || !r.zoneModelOn() {
		return
	}
	now := r.e.Clock.Now()
	z := &r.zst
	z.mu.Lock()
	keep := z.reask.pots[:0]
	for _, t := range z.reask.pots {
		if now.Sub(t) < crisisWindow {
			keep = append(keep, t)
		}
	}
	keep = append(keep, now)
	crisis := len(keep) >= crisisN
	if crisis {
		keep = keep[:0]
	}
	z.reask.pots = keep
	z.mu.Unlock()
	if crisis {
		area := 0
		if ls := r.latest(); ls != nil && ls.Player != nil {
			area = ls.Player.Area
		}
		r.zoneCostAt("potion_crisis", area)
	}
}

// zoneDial is the dial id of the knob; an out-of-range value is the default (balanced).
func (r *Runner) zoneDial() string {
	i := r.e.Tune.Bot.ZoneDial
	if i < 0 || i >= len(zoneDialIDs) {
		i = 1
	}
	return zoneDialIDs[i]
}

func (r *Runner) zoneModelOn() bool { return r.e.Tune.Bot.ZoneModel == 1 && r.d.ZoneDecide != nil }

// zoneRecheckDue is true when the cached pick is old enough to ask the model again.
func (r *Runner) zoneRecheckDue() bool {
	if !r.zoneModelOn() {
		return false
	}
	z := &r.zst
	z.mu.Lock()
	defer z.mu.Unlock()
	now := r.e.Clock.Now()
	if z.reask.pending != "" && z.reask.allowed(now) {
		return true
	}
	return z.have && now.Sub(z.at) >= zoneRecheck
}

// zoneCands lists the plan's zones of this difficulty: the eligible steps, and the cleared zones as depleted candidates.
func (r *Runner) zoneCands(eligible []Step, diff string, cleared map[int]bool) []ZoneCand {
	seen := map[int]bool{}
	var out []ZoneCand
	for _, st := range eligible {
		if !seen[st.Area] {
			seen[st.Area] = true
			out = append(out, ZoneCand{Area: st.Area})
		}
	}
	for _, st := range r.d.Plan.Steps[diff] {
		if st.Area != 0 && cleared[st.Area] && !seen[st.Area] && !r.d.Plan.IsTown(st.Area) {
			seen[st.Area] = true
			out = append(out, ZoneCand{Area: st.Area, Depleted: true})
		}
	}
	return out
}

// zoneNoteHour logs msg at most once an hour per key.
func (r *Runner) zoneNoteHour(key, msg string) {
	z, now := &r.zst, r.e.Clock.Now()
	z.mu.Lock()
	if at, ok := z.noted1[key]; ok && now.Sub(at) < time.Hour {
		z.mu.Unlock()
		return
	}
	if z.noted1 == nil {
		z.noted1 = map[string]time.Time{}
	}
	z.noted1[key] = now
	z.mu.Unlock()
	r.e.Log.Info(msg)
}

// zoneReachableCands drops the zones the hero cannot get to now (no route, a guarded area on the way, a long walk with
// no activated waypoint): the controller used to rank them, pick a hole among them and the walk then failed
// ("area skipped unreachable" 14 times in 26 minutes, audit postdeploy2).
func (r *Runner) zoneReachableCands(s *d2.Snapshot, cands []ZoneCand) []ZoneCand {
	out := cands[:0:0]
	for _, c := range cands {
		if c.Depleted {
			out = append(out, c) // a cleared zone is only a depleted yardstick, never walked to by this pick
			continue
		}
		if why := r.unreachable(context.Background(), s, c.Area); why != "" {
			r.zoneNoteHour(fmt.Sprintf("drop:%d", c.Area), fmt.Sprintf("zone candidate dropped: %s not reachable now (%s)", r.areaName(c.Area), why))
			continue
		}
		out = append(out, c)
	}
	return out
}

// zoneModelPick is PickOptions.Model for the goal's leveling pick; nil while the model is off.
func (r *Runner) zoneModelPick(s *d2.Snapshot, diff string, cleared map[int]bool) func([]Step) (int, bool, bool) {
	if !r.zoneModelOn() || s == nil || s.Player == nil {
		return nil
	}
	return func(eligible []Step) (int, bool, bool) {
		potions := -1
		if healing, _, known := r.beltAndInv(s); known {
			potions = healing
		}
		cands := r.zoneReachableCands(s, r.zoneCands(eligible, diff, cleared))
		if len(cands) == 0 {
			r.zoneNoteHour("empty", "zone choice: no eligible candidate (every zone of the plan is unreachable now), the plan tables decide")
			return 0, false, false
		}
		v, err := r.d.ZoneDecide(s, s.Player.Area, diff, potions, r.zoneDial(), cands)
		z := &r.zst
		if err != nil {
			z.mu.Lock()
			first := z.failed != err.Error()
			z.failed, z.have = err.Error(), false
			z.mu.Unlock()
			if first { // one line per distinct error, never a line per tick
				r.e.Log.Warn("zone model: error, the plan tables decide", "err", err)
			}
			return 0, false, false
		}
		now := r.e.Clock.Now()
		z.mu.Lock()
		prev, hadPrev, reason := z.v, z.have, z.reask.pending
		z.v, z.at, z.have, z.failed = v, now, true, ""
		if reason != "" {
			z.reask.pending = ""
			z.reask.stamps = append(z.reask.stamps, now)
		}
		z.mu.Unlock()
		if reason != "" {
			kBefore := 0.0
			if hadPrev {
				kBefore = prev.K
			}
			act := "stay"
			if v.Fresh {
				act = "leave (fresh game)"
			} else if v.Area != s.Player.Area {
				act = "leave"
			}
			r.e.Log.Info(fmt.Sprintf("zone reask: reason=%s k %.2f -> %.2f choice %d (%s)", reason, kBefore, v.K, v.Area, act))
		}
		if v.Hole && v.Area != s.Player.Area && v.Area != r.safeFarmArea(s) {
			// A hole names the best of what is LEFT. Left after bans for the bot's own faults (or with no margin at all)
			// that is the hardest zone, not the easiest field: Tower Cellar Level 4 after one cut-off cell banned ten
			// zones (live 2026-11-28, L2). The hero takes the act's first field if the plan offers it, else nothing is
			// picked (the caller waits in town; fault bans are short).
			if why := r.holeHoldWhy(v, diff); why != "" {
				r.zoneNoteHour(fmt.Sprintf("holehold:%d", v.Area), fmt.Sprintf("zone choice: hole fallback to area %d refused, %s; %s", v.Area, why, v.Line))
				safe := r.safeFarmArea(s)
				for _, st := range eligible {
					if st.Area == safe && r.fullKit(s) {
						return safe, false, true
					}
				}
				return 0, true, false
			}
		}
		if v.Hole && v.Area != s.Player.Area && !r.fullKit(s) {
			// the first-field fallback walks a hero who is not safe here into a zone below the margin: only with the kit
			// the plan itself asks for (potions and a town portal); otherwise the plan tables decide
			r.zoneNoteHour(fmt.Sprintf("holekit:%d", v.Area), fmt.Sprintf("zone choice: hole fallback to area %d refused, the kit is short; %s", v.Area, v.Line))
			return 0, false, false
		}
		for _, line := range splitLines(v.Line) {
			if v.Hole {
				r.zoneNoteHour(fmt.Sprintf("hole:%d", v.Area), line) // the same hole every dwell is one line an hour
				continue
			}
			r.e.Log.Info(line)
		}
		if v.MercDead { // every revive decision, with the numbers (the plain `zone revive:` line has the words)
			r.e.Log.Info("merc revive: decision", "revive", v.MercRevive, "zone", v.MercZone, "gain", int(v.MercGain), "cost", int(v.MercCost), "why", v.MercWhy)
		}
		if v.Fresh {
			if r.d.Cycler == nil || len(cleared) == 0 {
				return 0, false, false // no way to take a fresh game: the plan rule decides
			}
			return 0, true, false
		}
		return v.Area, false, v.Area != 0
	}
}

// holeHoldWhy says why a hole verdict must not be walked to ("" = it may): the zone has no margin at all, or a zone of the
// plan is out of play only for a fault of the bot, so "nothing is safe enough" is not known to be true.
func (r *Runner) holeHoldWhy(v ZoneVerdict, diff string) string {
	if v.Hold {
		return "the zone has no margin"
	}
	now := r.e.Clock.Now()
	r.obMu.Lock()
	defer r.obMu.Unlock()
	for _, st := range r.d.Plan.Steps[diff] {
		if until, ok := r.avoid[st.Area]; ok && until.After(now) && r.avoidWhy[st.Area] != whyDeath && st.Area != v.Area {
			return fmt.Sprintf("%s is only skipped (%s)", r.areaName(st.Area), r.avoidWhy[st.Area])
		}
	}
	return ""
}

// zoneLeaveMercDead is true while the model says the dead merc stays dead (the zone kills him, or he adds too little).
// It only ever suppresses the revive chore; hiring and the legacy gold rules are untouched.
func (r *Runner) zoneLeaveMercDead() bool {
	if !r.zoneModelOn() {
		return false
	}
	z := &r.zst
	z.mu.Lock()
	defer z.mu.Unlock()
	now := r.e.Clock.Now()
	area := 0 // the zone the model plans (0: any closed zone counts)
	if z.have {
		area = z.v.Area
	}
	if r.mercLoop().BlockedArea(area, now) {
		if m := &r.zmerc; m.blockNoted.CompareAndSwap(false, true) {
			r.e.Log.Info("merc: left dead by the revive loop breaker", "area", area, "limit", r.e.Tune.Bot.MercReviveLoopN)
		}
		return true
	}
	if !z.have || !z.v.MercDead || z.v.MercRevive || r.e.Clock.Now().Sub(z.at) > 4*zoneRecheck {
		return false
	}
	if z.noted != z.v.MercWhy {
		z.noted = z.v.MercWhy
		r.e.Log.Info("merc: left dead by the zone model", "why", z.v.MercWhy)
	}
	return true
}

// ZoneMargin is the word of the margin of the zone the model last chose ("" while the model is off or silent), for the
// Status sentence: words only, never a number (user decision 2026-11-14).
func (r *Runner) ZoneMargin() (word, why string) {
	z := &r.zst
	z.mu.Lock()
	defer z.mu.Unlock()
	if !z.have {
		return "", ""
	}
	return z.v.Margin, z.v.Why
}

func splitLines(s string) []string {
	var out []string
	for len(s) > 0 {
		i := 0
		for i < len(s) && s[i] != '\n' {
			i++
		}
		if i > 0 {
			out = append(out, s[:i])
		}
		if i == len(s) {
			break
		}
		s = s[i+1:]
	}
	return out
}

// Recovery and return decisions (card survrecover). A chicken, death or merc death sends the bot through town and, with
// little gold, through the gold-farm recovery; neither path reaches Plan.Pick, so the model never saw the zone that had
// just hurt the hero (holdout seed 149: the return portal took him straight back to Dark Wood). The two places that
// choose where to go back to (the trip's return portal and the recovery farm area) now ask zoneRecover when the zone
// they would use cost the hero something within burnWindow. The model's answer stands for recoverHold per zone, so two
// asks in a row never flap; any error or silence leaves the legacy choice.
const (
	burnWindow  = 15 * time.Minute
	recoverHold = 5 * time.Minute
)

type zoneBurn struct {
	at     time.Time
	reason string
}

type zoneHold struct {
	until  time.Time
	choice int
}

// zoneCostAt is zoneCost that also remembers the zone the event happened in.
func (r *Runner) zoneCostAt(reason string, area int) {
	if r.e == nil || !r.zoneModelOn() {
		return
	}
	if area != 0 && !r.d.Plan.IsTown(area) {
		z := &r.zst
		z.mu.Lock()
		if z.burn == nil {
			z.burn = map[int]zoneBurn{}
		}
		z.burn[area] = zoneBurn{at: r.e.Clock.Now(), reason: reason}
		z.mu.Unlock()
	}
	r.zoneCost(reason)
}

// zoneBurned is the reason of a cost event in area within burnWindow ("" when none or the model is off).
func (r *Runner) zoneBurned(area int) string {
	if !r.zoneModelOn() {
		return ""
	}
	z := &r.zst
	z.mu.Lock()
	defer z.mu.Unlock()
	if b, ok := z.burn[area]; ok && r.e.Clock.Now().Sub(b.at) < burnWindow {
		return b.reason
	}
	return ""
}

// zoneRecover answers "go back to want?" among cands (area ids, want included): want when the zone is clean, the model
// is off or fails, else the model's choice (which may still be want). Read-only: the ledger already holds the event.
func (r *Runner) zoneRecover(s *d2.Snapshot, want int, cands []int, via string) int {
	reason := r.zoneBurned(want)
	if reason == "" || s == nil || s.Player == nil {
		return want
	}
	now := r.e.Clock.Now()
	z := &r.zst
	z.mu.Lock()
	if h, ok := z.recov[want]; ok && now.Before(h.until) {
		z.mu.Unlock()
		return h.choice
	}
	z.mu.Unlock()
	diff := difficultyName(s.Player.Difficulty)
	potions := -1
	if healing, _, known := r.beltAndInv(s); known {
		potions = healing
	}
	zc := make([]ZoneCand, 0, len(cands))
	for _, a := range cands {
		zc = append(zc, ZoneCand{Area: a})
	}
	v, err := r.d.ZoneDecide(s, s.Player.Area, diff, potions, r.zoneDial(), zc)
	if err != nil {
		z.mu.Lock()
		first := z.failed != err.Error()
		z.failed = err.Error()
		z.mu.Unlock()
		if first {
			r.e.Log.Warn("zone recover: error, the legacy choice stands", "err", err)
		}
		return want
	}
	choice := want
	for _, a := range cands {
		if a == v.Area && !v.Fresh {
			choice = a
		}
	}
	z.mu.Lock()
	if z.recov == nil {
		z.recov = map[int]zoneHold{}
	}
	z.recov[want] = zoneHold{until: now.Add(recoverHold), choice: choice}
	z.mu.Unlock()
	act := "return"
	if choice != want {
		act = "alternative"
	}
	r.e.Log.Info(fmt.Sprintf("zone recover: reason=%s via=%s area %d margin=%s k %.2f choice %d (%s)", reason, via, want, v.Margin, v.K, choice, act))
	return choice
}

// zoneRecoverReturn is zoneRecover for a trip's return portal: the candidates are the plan's zones this hero may enter.
func (r *Runner) zoneRecoverReturn(s *d2.Snapshot, want int) int {
	if want == 0 || r.zoneBurned(want) == "" || s == nil || s.Player == nil || r.gearVerdict(s).Gated || r.kitGateActive(s) {
		return want // a gated hero has only the safe field: the gates decide
	}
	p := s.Player
	t := r.e.Tune.Bot
	ex := map[int]bool{}
	for a := range r.avoidedNow() {
		ex[a] = true
	}
	r.guardedAreas(s, ex)
	o := PickOptions{Margin: t.LevelMargin, Slack: t.LeaveSlack, Acts: r.allowedActs(s), Exclude: ex, FullKit: r.fullKit(s), Melee: MeleeTank(p.Class), Class: planClass(p.Class), Merc: false}
	cands := []int{want}
	seen := map[int]bool{want: true}
	for _, st := range r.d.Plan.eligible(difficultyName(p.Difficulty), p.Level, o, false) {
		if !seen[st.Area] && st.entry(o)+o.Margin <= p.Level {
			seen[st.Area] = true
			cands = append(cands, st.Area)
		}
	}
	return r.zoneRecover(s, want, cands, "return")
}

// portalLandsBurned: the town portal leads to where the last scroll was read, not to the zone the trip returns to. When
// the two differ and the landing zone just cost the hero (chicken, death, burn) or is not allowed, the portal would drop
// it into the zone it is leaving ("wanted 6, in 9", holdout seeds 149 and 163).
func (r *Runner) portalLandsBurned(level int, redirected bool) bool {
	r.obMu.Lock()
	from := r.ob.portalFrom
	r.obMu.Unlock()
	if from != 0 && !r.d.Plan.IsTown(from) && r.stalePackArea(r.latest()) == from {
		// the pack the hero fled from still stands where the portal drops it (Questy bed 2026-11-26: four deaths at one spot in the
		// Dark Wood, each after a portal return to the Treehead pack): walk in from the other side
		return true
	}
	if level == 0 || from == 0 || from == level || r.d.Plan.IsTown(from) {
		return false
	}
	// redirected: the field gate sent the trip to another zone than the one it left (stale pack, kit, gear); the portal
	// drops the hero where it left, next to the pack that made it leave (V live 2026-11-24 20:32Z: Andariel's lair, 18
	// hostiles, dead in 11 s with the gate's "first field" 23 as the target)
	return redirected || r.zoneBurned(from) != "" || (r.latest() != nil && r.areaAllowed(r.latest(), from) != "")
}

// returnToField is the trip's way back: BackToField, except that a portal that lands in a burned or barred zone is not
// taken; the hero walks from town to the wanted zone (waypoints and exits, like the corpse run) and only falls back to the
// portal when that walk fails.
func (r *Runner) returnToField(ctx context.Context, level int, redirected bool) error {
	if r.portalLandsBurned(level, redirected) {
		r.e.Log.Info("back to field: portal lands in a burned zone, walking instead", "area", level, "name", r.areaName(level))
		err := r.travelTo(ctx, level)
		if err == nil || ctx.Err() != nil {
			return err
		}
		r.e.Log.Warn("back to field: walk failed, taking the portal", "err", err)
	}
	return r.d.Town.BackToField(ctx, level)
}

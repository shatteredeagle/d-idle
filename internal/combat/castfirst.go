package combat

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// The cast-first loop (docs/combat.md "Cast first"). G1 attempt 13 (V, soak log and the user watching Soaky): full mana,
// a pack in reach and no attack for minutes, because the movement branches of engage (dodge, kite, approach, rest,
// back-off) each took the tick. The rule now: a step that is not a cast may not replace a cast unless it is the single
// melee step-back; a watchdog forces a cast when none happened for watchdogSec while a hostile is in range.
const (
	watchdogSec   = 6               // no cast for this long with a hostile in range and mana: force one
	evadeGap      = 2 * time.Second // a dodge/sidestep/back-off is allowed only this soon after a cast
	novaCrowd     = 3               // hostiles within novaRadius that make Frost Nova worth a cast
	novaRadius    = 4               // subtiles
	novaGap       = 2500 * time.Millisecond
	stepBackDist  = 3  // single melee monster this close ...
	stepBackLife  = 60 // ... and life below this percent and falling
	stepBackSteps = 2  // steps per burst, then a cast or a drink
	healthEvery   = time.Minute
)

// noteCast records an attack cast of the fight.
func (st *fightState) noteCast(now time.Time) {
	st.casts++
	st.lastCastAt = now
	if st.casts == 1 && !st.t0.IsZero() {
		st.firstCast = now.Sub(st.t0)
	}
}

// evadeOK says whether a non-cast movement (dodge, sidestep, back-off, keep-range) may take this tick: only when the
// fight cast recently, so such steps can never replace the casts for more than evadeGap.
func (f *Fighter) evadeOK(st *fightState) bool {
	return f.now().Sub(st.lastCastAt) < evadeGap
}

// decision is the debug line for a tick that did not cast, and remembers why for the watchdog.
func (f *Fighter) decision(st *fightState, act, why string, args ...any) {
	st.lastNoCast = act + ": " + why
	f.log().Debug("combat decision", append([]any{"act", act, "why", why}, args...)...)
}

// castRange is the distance within which a hostile counts as in reach of the character's attack.
func (f *Fighter) castRange(p *d2.Player) int {
	r := 0
	for _, sp := range f.attackSkills(p) {
		if sp.Kind != KindSelf && sp.Kind != KindBuff && sp.Range > r {
			r = sp.Range
		}
	}
	if r == 0 {
		r = meleeRange
	}
	return r
}

// castFirst runs after survival and before the movement branches. handled is true when it spent the tick.
func (f *Fighter) castFirst(ctx context.Context, snap *d2.Snapshot, cand []Target, st *fightState) (handled bool, err error) {
	p := snap.Player
	now := f.now()
	if len(cand) == 0 || p.InTown() {
		st.nearSince = time.Time{}
		return false, nil
	}
	rng := float64(f.castRange(p))
	if cand[0].Dist > rng { // cand follows the distance order of Hostiles
		st.nearSince = time.Time{}
		return false, nil
	}
	if st.nearSince.IsZero() {
		st.nearSince = now
	}
	if !f.castReady() {
		return false, nil
	}
	if ok, err := f.novaTick(snap, st); ok || err != nil {
		return ok, err
	}
	idle := now.Sub(st.lastCastAt)
	if since := now.Sub(st.nearSince); since < idle {
		idle = since
	}
	if idle < watchdogSec*time.Second {
		return false, nil
	}
	return f.forceCast(snap, cand, st, idle)
}

// novaTick casts Frost Nova once when a crowd stands within novaRadius (a bound Frost Nova is needed; the plan gives it
// a key in the Sorceress layouts). The bolts follow on the next ticks.
func (f *Fighter) novaTick(snap *d2.Snapshot, st *fightState) (bool, error) {
	p := snap.Player
	now := f.now()
	if now.Sub(st.lastNova) < novaGap {
		return false, nil
	}
	nova, ok := KnownSkills(p, f.cat, f.keys)["Frost Nova"]
	if !ok || (nova.Key == "" && p.RightSkill != nova.ID) || nova.Cost > p.Mana || p.Mana < f.tune().NovaManaFloor {
		return false, nil
	}
	n := 0
	for _, h := range Hostiles(snap, f.hostileCtx(p)) {
		if h.Dist > novaRadius {
			break
		}
		if !h.Immune(nova.Elem) {
			n++
		}
	}
	if n < novaCrowd {
		return false, nil
	}
	st.lastNova = now
	st.novas++
	st.noteCast(now)
	f.log().Debug("combat decision", "act", "nova", "why", "crowd", "n", n)
	f.castLogTargets(n)
	return true, f.cast(snap, nova, input.Point{X: p.X, Y: p.Y})
}

// forceCast is the watchdog: cast the first usable attack at the nearest hostile that is on screen.
func (f *Fighter) forceCast(snap *d2.Snapshot, cand []Target, st *fightState, idle time.Duration) (bool, error) {
	p := snap.Player
	now := f.now()
	why := st.lastNoCast
	if why == "" {
		why = "no decision recorded"
	}
	logOnce := func(msg string) {
		if now.Sub(st.wdLogAt) >= 5*time.Second {
			st.wdLogAt = now
			f.log().Info("combat: watchdog cannot cast ("+msg+")", "idle", idle.Round(time.Second), "why", why, "mana", p.Mana)
		}
	}
	var tg *Target
	for i := range cand {
		if onScreen(snap, input.Point{X: cand[i].Unit.X, Y: cand[i].Unit.Y}) {
			tg = &cand[i]
			break
		}
	}
	if tg == nil {
		logOnce("no hostile on screen")
		return false, nil
	}
	f.refineResists(tg, p.Difficulty)
	for _, sp := range f.attackSkills(p) {
		if sp.Kind == KindSelf || sp.Kind == KindBuff || sp.Cost > p.Mana || tg.Immune(sp.Elem) || !f.skillReady(sp) {
			continue
		}
		if sp.Kind != KindMelee && tg.Dist > float64(sp.Range) {
			continue
		}
		if sp.Hand == HandLeft && p.LeftSkill != sp.ID {
			continue // selecting it costs a tick of its own; engage does that
		}
		tr := st.tracks[tg.Unit.ID]
		if tr != nil {
			tr.casts++
		}
		st.forced++
		st.noteCast(now)
		st.book.Add(f.now().Sub(st.t0).Seconds(), sp.Cost)
		f.log().Info(fmt.Sprintf("combat: forced cast (no cast for %ds, %s)", int(idle.Seconds()), why),
			"skill", sp.Name, "target", tg.Name, "dist", int(tg.Dist))
		return true, f.cast(snap, sp, input.Point{X: tg.Unit.X, Y: tg.Unit.Y})
	}
	logOnce("no affordable, ready, unblocked attack skill")
	return false, nil
}

// healthWindow accumulates the per-minute health line for the soak harness.
type healthWindow struct {
	since                           time.Time
	kills, casts, kites, novas, dmg int
	forced                          int
}

// addFight folds a finished fight in and logs the line once a minute has passed.
func (f *Fighter) addFight(ft *FightTelemetry, st *fightState, res ClearResult) {
	now := f.now()
	h := &f.health
	if h.since.IsZero() {
		h.since = now.Add(-res.Elapsed)
	}
	h.kills += res.Kills
	h.casts += st.casts
	h.kites += st.kitesAll
	h.novas += st.novas
	h.forced += st.forced
	h.dmg += ft.Damage
	el := now.Sub(h.since)
	if el < healthEvery {
		return
	}
	min := el.Minutes()
	per := 0
	if h.kills > 0 {
		per = h.dmg / h.kills
	}
	f.info(fmt.Sprintf("health killsPerMin=%.1f castsPerMin=%.1f kitesPerMin=%.1f novas=%d forced=%d damagePerKill=%d windowSec=%d",
		float64(h.kills)/min, float64(h.casts)/min, float64(h.kites)/min, h.novas, h.forced, per, int(el.Seconds())))
	*h = healthWindow{since: now}
}

// skillNames lists the names of known skills for a log line.
func skillNames(m map[string]SkillSpec) []string {
	var out []string
	for n := range m {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// noTarget logs why a fight with hostiles around found nothing to engage (the nav-yield loop: nav yields, combat
// returns at once, nav resumes), so the next soak shows the reason instead of an idle walk.
func (f *Fighter) noTarget(hs []Target, st *fightState, sp fightSpec) {
	near, memo, black, avoid := 0, 0, 0, 0
	for _, h := range hs {
		if h.Dist > engageDist {
			continue
		}
		near++
		switch {
		case h.Avoid:
			avoid++
		case st.blacklist[h.Unit.ID]:
			black++
		case f.ignoredNow(h.Unit.ID, h.Dist):
			memo++
		}
	}
	if near > 0 {
		if sp.yield {
			f.note(Note{Kind: "yield", Text: "Nothing to engage in scope: navigation yields"})
		}
		f.info("no target in scope", "near", near, "avoid", avoid, "blacklisted", black, "ignoredEarlier", memo, "yield", sp.yield)
	}
}

// fightTick logs the one compact per-fight summary that tells the attack fraction: engage ticks, attacks issued,
// moves (approach, kite, dodge, back-off), seconds to the first attack, kills per minute and the latest retreat reason.
func (f *Fighter) fightTick(ft *FightTelemetry, st *fightState, start time.Time, res ClearResult) {
	el := f.now().Sub(start)
	first := -1.0
	if st.casts > 0 {
		first = st.firstCast.Seconds()
	}
	moves := st.approach + st.kitesAll + st.dodges + st.mdodges + st.backoffsAll
	f.info("fight.tick", "ticks", st.ticks, "attacks", st.casts, "moves", moves, "backoffs", st.backoffsAll,
		"approaches", st.approach, "kites", st.kitesAll, "dodges", st.dodges+st.mdodges, "maxHostiles", st.maxHostiles,
		"firstAttackSec", first, "sec", int(el.Seconds()), "kills", res.Kills, "killsPerMin", perMin(res.Kills, el),
		"retreat", st.retreatWhy, "lastNoCast", st.lastNoCast, "end", ft.End, "remaining", ft.Remaining)
}

func perMin(n int, d time.Duration) float64 {
	if d <= 0 {
		return 0
	}
	return float64(n) / d.Minutes()
}

// noteEnd records how a fight ended and what was left: end is cleared (nothing alive in scope), the loop exit that
// stopped it with hostiles left (timeout, noTargets = all ignored or avoided, rankEmpty, targetGone), or the
// survival result (dead, townPortal, chicken, error). remaining counts hostiles in scope on the last look, ignored
// ones included: remaining above 0 with end=noTargets is the "left the rest alive" case.
func (f *Fighter) noteEnd(ft *FightTelemetry, st *fightState, sp fightSpec, s *d2.Snapshot, err error) {
	if s != nil && s.Player != nil {
		for _, h := range Hostiles(s, f.hostileCtx(s.Player)) {
			if f.inRange(h, sp, s.Player) {
				ft.Remaining++
			}
		}
		ft.LifePct, ft.ManaPct = pct(s.Player.Life, s.Player.MaxLife), pct(s.Player.Mana, s.Player.MaxMana)
	}
	switch {
	case errors.Is(err, ErrDead):
		ft.End = "dead"
	case errors.Is(err, ErrTownPortal):
		ft.End = "townPortal"
	case errors.Is(err, ErrChicken):
		ft.End = "chicken"
	case err != nil:
		ft.End = "error"
	case st.endWhy == "noTargets" && ft.Remaining == 0, st.endWhy == "" && ft.Remaining == 0:
		ft.End = "cleared"
	case st.endWhy == "":
		ft.End = "unknown"
	default:
		ft.End = st.endWhy
	}
}

package combat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/resource"
)

// engageDist: a monster this close to the player is fought even when it is outside the clear radius.
const engageDist = 18

type fightSpec struct {
	only     uint32 // 0 = clear everything, else just this unit
	origin   input.Point
	radius   int
	deadline time.Time
	perUnit  time.Duration
	yield    bool // called by a stalled walk: fight what blocks it, whatever was ignored earlier
}

// track is the per-target book keeping. Times accumulate only while the unit is the active target, so a monster
// that was set aside for others is not given up on for the time it spent waiting.
type track struct {
	last            time.Time
	engaged, noDrop time.Duration
	dt              time.Duration // time since the previous engage, charged to noDrop only when a cast follows
	bestLife        int
	casts           int
	approaches      int
	inShot          bool // was in range with a clear line at least once
	assessAt        time.Time
	plan            reachPlan
	firePos         input.Point
	statics         int
}

type seenUnit struct {
	name string
	boss bool
}

type fightState struct {
	tracks    map[uint32]*track
	seen      map[uint32]seenUnit
	blacklist map[uint32]bool
	avoided   map[uint32]bool
	stalled   bool // the no-progress watchdog already logged in this fight
	ignored   []Ignored
	lastExp   uint32
	haveExp   bool
	pendingXP int64
	dodges    int
	mdodges   int // missile sidesteps (knob combat.missileDodge)
	kites     int // retreat steps in the current burst; resets after 10 quiet seconds
	lastKite  time.Time
	casts     int // attack casts of this fight (telemetry)
	kitesAll  int // kite steps of this fight (telemetry)
	approach  int // approach/firing-position moves of this fight (telemetry)
	novas     int
	forced    int

	// cast-first loop (castfirst.go)
	lastCastAt time.Time // last attack cast of this fight (start of the fight until the first one)
	nearSince  time.Time // a hostile has been in cast range since then, zero when none is
	lastNova   time.Time
	lastNoCast string // reason of the latest tick that did not cast
	wdLogAt    time.Time

	// life reference for "life is dropping" (kite rule): the life a couple of seconds ago
	lifeRef   int
	lifeRefAt time.Time
	everSeen  bool

	// resource policy (resource.go)
	skipPre   time.Time // no PreEngage before this time (it timed out or failed)
	restFrom  time.Time // when the current rest before a pack began
	retreated bool      // already walked out of sight for this rest
	waitFrom  time.Time // when the current out-of-mana wait began
	backoffs  int       // consecutive out-of-mana back-off steps; reset by a cast or a free skill
	fleeFrom  time.Time // start of the current break-away burst
	fleeLast  time.Time

	// Nightmare/Hell (hard.go)
	mb    resource.ManaBurnTracker
	spent int // mana spent by our own casts since the last mana-burn sample
	t0    time.Time
}

func newFightState() *fightState {
	return &fightState{tracks: map[uint32]*track{}, seen: map[uint32]seenUnit{}, blacklist: map[uint32]bool{}, avoided: map[uint32]bool{}}
}

// Clear kills every hostile monster within o.Radius of the origin (and anything within engageDist of the player)
// until none is left, the deadline passes or the character has to leave: ErrTownPortal (a portal was opened),
// ErrChicken (the game was left) or ErrDead are returned with the partial result.
func (f *Fighter) Clear(ctx context.Context, o ClearOpts) (ClearResult, error) {
	t := f.tune()
	snap, err := f.read(ctx)
	if err != nil {
		return ClearResult{}, err
	}
	spec := fightSpec{radius: o.Radius, perUnit: time.Duration(t.MaxTargetSec) * time.Second, yield: o.Yield}
	if spec.radius <= 0 {
		spec.radius = t.ClearRadius
	}
	spec.origin = input.Point{X: snap.Player.X, Y: snap.Player.Y}
	if o.Origin != nil {
		spec.origin = *o.Origin
	}
	to := o.Timeout
	if to <= 0 {
		to = time.Duration(t.ClearMaxSec) * time.Second
	}
	spec.deadline = f.now().Add(to)
	res, err := f.fight(ctx, spec, snap)
	if err == nil {
		f.classUpkeep(ctx) // raise, summon or re-cast what is missing; failures are not a Clear failure
	}
	return res, err
}

// Kill fights one unit (a boss or a named monster) until it dies. The unit must be on the map within a couple of
// seconds, else ErrTargetGone. The time limit is the Clear limit, not the per-target limit.
func (f *Fighter) Kill(ctx context.Context, unitID uint32) error {
	t := f.tune()
	snap, err := f.read(ctx)
	if err != nil {
		return err
	}
	spec := fightSpec{only: unitID, radius: 1 << 20, perUnit: time.Duration(t.ClearMaxSec) * time.Second}
	spec.origin = input.Point{X: snap.Player.X, Y: snap.Player.Y}
	spec.deadline = f.now().Add(spec.perUnit)
	res, err := f.fight(ctx, spec, snap)
	if err != nil {
		return err
	}
	if res.Skipped > 0 {
		return ErrTargetGaveUp
	}
	if res.TimedOut {
		return ErrTargetGaveUp
	}
	if res.Kills == 0 {
		return ErrTargetGone
	}
	return nil
}

// noProgressMax is how many fight-loop iterations may pass without the clock moving before the loop forces a sleep.
const noProgressMax = 40

func (f *Fighter) fight(ctx context.Context, sp fightSpec, snap *d2.Snapshot) (res ClearResult, err error) {
	start := f.now()
	var ft FightTelemetry
	var st *fightState
	defer func() { // runs after the Elapsed defer below
		if ft.Area != 0 && st != nil {
			ft.Dodges = st.dodges + st.mdodges
			ft.Casts, ft.Kites, ft.Approaches = st.casts, st.kitesAll, st.approach
			ft.Novas, ft.Forced = st.novas, st.forced
			f.info(ft.Finish(res))
			f.addFight(&ft, st, res)
		}
	}()
	defer func() { res.Elapsed = f.now().Sub(start) }()
	st = newFightState()
	st.t0, st.lastCastAt = start, start
	t := f.tune()
	var lastTick time.Time
	still := 0
	for {
		if err = ctx.Err(); err != nil {
			return
		}
		// watchdog: under a lockstep clock time only moves on sleeps and steps, so an iteration chain that never
		// sleeps (failed walks, instant actions) would loop forever; force one poll sleep and say so once
		if n := f.now(); n.Equal(lastTick) {
			if still++; still >= noProgressMax {
				if !st.stalled {
					st.stalled = true
					f.info("no progress, holding", "iterations", still)
				}
				if err = f.sleep(ctx, ms(t.PollMs)); err != nil {
					return
				}
				still = 0
				snap = nil
			}
		} else {
			lastTick, still = n, 0
		}
		if f.now().After(sp.deadline) {
			res.TimedOut = true
			break
		}
		if snap == nil {
			if snap, err = f.read(ctx); err != nil {
				return
			}
		}
		p := snap.Player
		if isDead(p) {
			f.buffAt = map[int]time.Time{}
			f.auraDirty = true
			return res, ErrDead
		}
		f.sampleLife(p)
		ft.Sample(p)
		hs := Hostiles(snap, f.hostileCtx(p))
		ft.Area, ft.Diff = p.Area, int(p.Difficulty)
		ft.Danger = max(ft.Danger, f.packDanger(p, hs))
		f.noteKills(snap, hs, st, &res)

		// 1. survival
		if a := f.Safety(snap); a != SafetyNone {
			// Reaction time is a handicap for routine potions; a character about to die reacts at once.
			if a != Chicken && pct(p.Life, p.MaxLife) > t.ChickenPct+10 {
				if err = f.sleep(ctx, ms(t.ReactionMs)); err != nil {
					return
				}
			}
			aerr := f.Act(ctx, a)
			switch {
			case (a == TownPortal || a == CrowdAbandon) && aerr == nil:
				return res, ErrTownPortal
			case a == Chicken && aerr == nil:
				return res, ErrChicken
			case aerr == nil:
				res.Potions++
			case errors.Is(aerr, ErrNoPotion), errors.Is(aerr, ErrNoPortal):
				f.info("survival action unavailable", "action", a.String(), "err", aerr)
			case (a == TownPortal || a == CrowdAbandon) && ctx.Err() == nil:
				// the portal could not be opened (BOSSLIVE died at 85% boss after 5 failed casts): leave the game
				f.info("town portal failed, leaving the game", "err", aerr)
				if lerr := f.Act(ctx, Chicken); lerr != nil {
					return res, fmt.Errorf("%w (leave failed: %v)", aerr, lerr)
				}
				return res, ErrChicken
			default:
				return res, aerr
			}
			snap = nil
			continue
		}

		// 1a. Nightmare/Hell rules (antidote, mana burn); after survival so a flee never skips a potion
		if herr := f.hardTick(ctx, snap, hs, st); herr != nil {
			if herr == errHardActed {
				snap = nil
				continue
			}
			return res, herr
		}

		// 1b. a lost fight (no potion, no mana, no way to attack, dead merc): portal or leave
		if handled, aerr := f.abortFight(ctx, snap, hs, st); handled {
			if aerr != nil {
				return res, aerr
			}
			snap = nil
			continue
		}

		// 2. the targets of this fight
		cand := f.inScope(hs, sp, p, st)
		if sp.only != 0 {
			for _, h := range hs {
				if h.Unit.ID == sp.only {
					st.everSeen = true
				}
			}
			if len(cand) == 0 {
				if st.blacklist[sp.only] || st.everSeen || f.now().Sub(start) > 2*time.Second {
					break
				}
				if err = f.sleep(ctx, ms(t.PollMs)); err != nil {
					return
				}
				snap = nil
				continue
			}
		}
		for _, h := range hs {
			if h.Avoid && !st.avoided[h.Unit.ID] && f.inRange(h, sp, p) {
				st.avoided[h.Unit.ID] = true
				f.ignore(st, h, IgnoreAvoid, fmt.Sprintf("danger %.0f", h.Danger))
			}
		}
		if sp.only == 0 {
			for _, h := range f.packSkips(p, cand) {
				f.ignore(st, h, IgnoreImmune, immuneDetail(h))
			}
			cand = f.inScope(hs, sp, p, st)
		}
		if len(cand) == 0 {
			f.noTarget(hs, st, sp)
			break
		}
		if handled, cerr := f.castFirst(ctx, snap, cand, st); handled || cerr != nil {
			if cerr != nil {
				return res, cerr
			}
			if err = f.sleep(ctx, ms(t.PollMs)); err != nil {
				return
			}
			snap = nil
			continue
		}
		if handled, perr := f.preEngage(ctx, snap, hs, cand, st, sp.only != 0); handled || perr != nil {
			if perr != nil {
				return res, perr
			}
			snap = nil
			continue
		}
		ranked := Rank(cand, t.TargetQuality, f.rng.Float64())
		if len(ranked) == 0 {
			break
		}
		// a careless fighter loses ticks
		if t.TargetQuality < 100 && f.rng.Float64()*200 < float64(100-t.TargetQuality) {
			if err = f.sleep(ctx, ms(t.CastGapMs)); err != nil { // hesitation costs about a cast
				return
			}
			snap = nil
			continue
		}
		tg := ranked[0]
		f.refineResists(&tg, p.Difficulty)
		if err = f.engage(ctx, snap, tg, sp, st); err != nil {
			return
		}
		if err = f.sleep(ctx, ms(t.PollMs)); err != nil {
			return
		}
		snap = nil
	}
	res.Skipped = len(st.blacklist)
	res.Avoided = len(st.avoided)
	res.Ignored = f.ignoredReport(snap, st, sp)
	// one last look so the final kill is counted
	if s, e := f.read(ctx); e == nil {
		f.noteKills(s, Hostiles(s, f.hostileCtx(s.Player)), st, &res)
	}
	return res, nil
}

func (f *Fighter) inRange(h Target, sp fightSpec, p *d2.Player) bool {
	return dist(h.Unit.X, h.Unit.Y, sp.origin.X, sp.origin.Y) <= float64(sp.radius) || h.Dist <= engageDist
}

func (f *Fighter) inScope(hs []Target, sp fightSpec, p *d2.Player, st *fightState) []Target {
	var out []Target
	for _, h := range hs {
		if st.blacklist[h.Unit.ID] || h.Avoid || sp.only == 0 && !sp.yield && f.ignoredNow(h.Unit.ID, h.Dist) {
			continue
		}
		if sp.only != 0 {
			if h.Unit.ID == sp.only {
				out = append(out, h)
			}
			continue
		}
		if f.inRange(h, sp, p) {
			out = append(out, h)
		}
	}
	return out
}

// ignoredReport is the Ignored list of a finished fight: this fight's entries plus remembered units still on the map in scope.
func (f *Fighter) ignoredReport(snap *d2.Snapshot, st *fightState, sp fightSpec) []Ignored {
	out := append([]Ignored(nil), st.ignored...)
	have := map[uint32]bool{}
	for _, i := range out {
		have[i.ID] = true
	}
	if snap == nil || snap.Player == nil {
		return out
	}
	for _, h := range Hostiles(snap, f.hostileCtx(snap.Player)) {
		if r, ok := f.ignores[h.Unit.ID]; ok && !have[h.Unit.ID] && f.now().Before(r.until) && f.inRange(h, sp, snap.Player) {
			out = append(out, r.Ignored)
		}
	}
	return out
}

// noteKills counts monsters seen alive earlier that are now dying or dead, emits Kill events and attributes the
// experience gained since the last tick.
func (f *Fighter) noteKills(snap *d2.Snapshot, alive []Target, st *fightState, res *ClearResult) {
	p := snap.Player
	if st.haveExp && p.Exp > st.lastExp {
		st.pendingXP += int64(p.Exp - st.lastExp)
	}
	st.lastExp, st.haveExp = p.Exp, true
	byID := map[uint32]*d2.Unit{}
	for i := range snap.Monsters {
		byID[snap.Monsters[i].ID] = &snap.Monsters[i]
	}
	var died []seenUnit
	for id, su := range st.seen {
		if u, ok := byID[id]; ok && !u.Alive() {
			died = append(died, su)
			delete(st.seen, id)
		} else if !ok {
			delete(st.seen, id) // out of the table: not counted as a kill (UNVERIFIED how long corpses stay)
		}
	}
	for _, a := range alive {
		if _, ok := st.seen[a.Unit.ID]; !ok {
			st.seen[a.Unit.ID] = seenUnit{name: a.Name, boss: a.Boss}
		}
	}
	if len(died) == 0 {
		return
	}
	share := st.pendingXP / int64(len(died))
	st.pendingXP -= share * int64(len(died)) // the remainder (tiny kills give 1-2 xp) goes to the next kill, not the floor
	for _, d := range died {
		res.Kills++
		if d.boss {
			res.Bosses++
		}
		res.XP += share
		f.emit(goals.Event{Type: goals.EvKill, Monster: d.name, Boss: d.boss, XP: share})
		f.info("kill", "monster", d.name, "boss", d.boss, "xp", share)
	}
}

// engage takes one step against the chosen target: dodge, kite, approach, or cast.
func (f *Fighter) engage(ctx context.Context, snap *d2.Snapshot, tg Target, sp fightSpec, st *fightState) error {
	t := f.tune()
	p := snap.Player
	now := f.now()
	me := input.Point{X: p.X, Y: p.Y}
	tr := st.tracks[tg.Unit.ID]
	if tr == nil {
		tr = &track{last: now, bestLife: tg.LifePct}
		st.tracks[tg.Unit.ID] = tr
	}
	if dt := now.Sub(tr.last); dt > 0 {
		dt = min(dt, 2*time.Second)
		tr.engaged += dt
		tr.dt = dt
	} else {
		tr.dt = 0
	}
	tr.last = now
	if tg.LifePct < tr.bestLife {
		tr.bestLife, tr.noDrop = tg.LifePct, 0
	}
	if tr.engaged > sp.perUnit || (tr.casts > 0 && tr.noDrop+tr.dt > time.Duration(t.NoProgressSec)*time.Second && sp.only == 0) {
		why := IgnoreNoProgress
		if tr.engaged > sp.perUnit {
			why = IgnoreTime
		}
		f.ignore(st, tg, why, fmt.Sprintf("casts %d life %d%% engaged %v", tr.casts, tg.LifePct, tr.engaged.Round(time.Second)))
		return nil
	}
	if !tr.inShot && sp.only == 0 && tr.engaged > time.Duration(t.ReachSec)*time.Second {
		f.ignore(st, tg, IgnoreNoShot, fmt.Sprintf("never in range with a clear line in %v, %d approaches", tr.engaged.Round(time.Second), tr.approaches))
		return nil
	}

	// missiles on top of us
	if t.DodgeRadius > 0 && f.mv != nil && f.mayEvade() && f.evadeOK(st) && now.Sub(f.lastDodge) > 1200*time.Millisecond && st.dodges < 40 {
		var near []input.Point
		for _, m := range hostileMissiles(snap) {
			if pdist(me, m) < float64(t.DodgeRadius) {
				near = append(near, m)
			}
		}
		if pt, ok := awayPoint(me, near, nil, t.DodgeRadius+4, f.valid); ok {
			f.lastDodge = now
			st.dodges++
			f.decision(st, "dodge", "missile on top of us")
			f.info("dodging", "to", pt)
			return f.evade(ctx, me, pt)
		}
	}

	if f.evadeOK(st) {
		if moved, err := f.sidestepMissiles(ctx, snap, &st.mdodges); moved || err != nil {
			f.decision(st, "sidestep", "dangerous missile")
			return err
		}
	}

	hard := p.Difficulty != d2.Normal
	var plan Plan
	if hard {
		if plan = PlanForTarget(tg, p.Class, p.Difficulty, p.Level); plan.Skip {
			f.ignore(st, tg, IgnoreAvoid, "plan: "+plan.Notes)
			return nil
		}
	}
	pr := f.profileFor(p)
	if _, err := f.KeepAura(ctx, snap, &tg, false); err != nil {
		return err
	}
	if acted, err := f.classTick(ctx, snap); acted || err != nil {
		f.decision(st, "class", "upkeep (summon, raise, trap)")
		return err
	}
	f.noteGear(snap)
	order := f.attackSkills(p)
	pick := PickAttack
	if hard {
		pick = PickByResist
	}
	spec, why := pick(order, &tg, p.Mana, f.skillReady)
	if why != PickOK {
		if fb, ok := f.fallbackSkill(p, &tg); ok && (why == PickImmune || why == PickNoSkill) {
			spec, why = fb, PickOK
		}
	}
	switch why {
	case PickImmune:
		if hard {
			w := ImmuneWorkaround(p.Class, &tg)
			f.info("immune target", "name", tg.Name, "workaround", w.Skills, "merc", w.Merc, "skip", w.Skip)
		}
		f.ignore(st, tg, IgnoreImmune, fmt.Sprintf("%v res %v", why, tg.Res))
		return nil
	case PickNoSkill:
		// Never idle: with the basic attack on the left button a click on the target is still an attack.
		if p.LeftSkill == 0 {
			spec, why = meleeSpec(), PickOK
			f.info("no bound attack skill, swinging the basic attack", "known", len(KnownSkills(p, f.cat, f.keys)), "keys", len(f.keys))
			break
		}
		f.info("no attack skill bound", "known", skillNames(KnownSkills(p, f.cat, f.keys)), "keys", f.keys, "left", p.LeftSkill, "right", p.RightSkill)
		f.ignore(st, tg, IgnoreNoSkill, "no attack skill known and usable for this class")
		return nil
	case PickCooling:
		f.decision(st, "wait", "attack skills cooling down")
		return nil
	}
	// Low mana (nothing affordable, or under the economy line): the policy's ladder picks a cheaper spell, melee,
	// a wait or a retreat (docs/resource.md, AttackChoice).
	// A melee profile ends its ladder in the free basic attack, so it never waits for mana.
	if why == PickNeedMana || (!pr.MeleeClass() && pct(p.Mana, p.MaxMana) < f.rtune().EconomyManaPct) {
		use, ok, lerr := f.lowMana(ctx, snap, tg, order, Hostiles(snap, f.hostileCtx(p)), tr, st)
		if !ok {
			return lerr
		}
		spec = use
	} else {
		st.waitFrom = time.Time{}
	}

	target := input.Point{X: tg.Unit.X, Y: tg.Unit.Y}

	if hard && plan.NoMelee && spec.Kind == KindMelee {
		f.ignore(st, tg, IgnoreAvoid, "plan: no melee ("+plan.Notes+")")
		return nil
	}
	if hard && f.mv != nil && f.mayEvade() && f.evadeOK(st) && plan.KeepRange > 0 && tg.Dist < float64(plan.KeepRange) && spec.Kind != KindMelee && spec.Kind != KindSelf && st.kites < 2*t.KiteMaxSteps {
		if pt, ok := awayPoint(me, []input.Point{target}, hostileMissiles(snap), t.KiteRetreat, f.valid); ok {
			st.kites++
			st.lastKite = now
			f.decision(st, "keep-range", "plan keep range", "from", tg.Name)
			f.info("keeping range", "from", tg.Name, "range", plan.KeepRange, "to", pt)
			return f.evade(ctx, me, pt)
		}
	}

	if pr.MeleeClass() && spec.Kind == KindMelee && f.mv != nil {
		if handled, err := f.meleeGuard(ctx, snap, pr.Melee, st); handled || err != nil {
			return err
		}
	}

	// Cursed (Amplify Damage / Iron Maiden) and below the retreat line: break contact instead of trading blows; Safety
	// drinks meanwhile. Bounded like a kite burst, and skipped when pinned (evade).
	if f.mv != nil && f.mayEvade() && f.evadeOK(st) && CurseRetreat(p, t.CurseRetreatPct) && st.kites < 2*t.KiteMaxSteps {
		var pts []input.Point
		for _, h := range Hostiles(snap, f.hostileCtx(p)) {
			if h.Dist <= float64(t.KiteDistance) {
				pts = append(pts, input.Point{X: h.Unit.X, Y: h.Unit.Y})
			}
		}
		if len(pts) > 0 {
			if pt, ok := awayPoint(me, pts, hostileMissiles(snap), t.KiteRetreat, f.valid); ok {
				st.kites++
				st.lastKite = now
				f.decision(st, "curse-retreat", "amplify/iron maiden and low life")
				f.info("cursed: backing off", "life", pct(p.Life, p.MaxLife), "to", pt)
				return f.evade(ctx, me, pt)
			}
		}
	}

	// Step back from ONE melee monster that is hurting us, never from a pack (G1 attempt 13, V: at 52-67% life with 15
	// Fallen around, every tick below KiteLifePct was a retreat step: 55 s and no kill; at 85% life it killed 11 in 2 s).
	// SME rule: single melee within stepBackDist, life below stepBackLife and falling, at most stepBackSteps in a burst,
	// then a cast or a drink. Anything else is a cast.
	if now.Sub(st.lastKite) > 10*time.Second {
		st.kites = 0
	}
	if t.KiteEnabled != 0 && f.mv != nil && f.mayEvade() && st.kites < stepBackSteps && (!hard || plan.Kite) && spec.Kind != KindSelf && spec.Kind != KindMelee {
		if th := f.kiteThreats(snap, p, stepBackDist+1); len(th) == 1 && th[0].Dist <= stepBackDist &&
			pct(p.Life, p.MaxLife) < stepBackLife && st.lifeDropping(now, p.Life) {
			pts := []input.Point{{X: th[0].Unit.X, Y: th[0].Unit.Y}}
			if pt, ok := awayPoint(me, pts, hostileMissiles(snap), t.KiteRetreat, f.valid); ok {
				st.kites++
				st.kitesAll++
				st.lastKite = now
				f.decision(st, "kite", "single melee within 3, life < 60% and falling", "life", pct(p.Life, p.MaxLife))
				f.info("kiting", "from", pts[0], "to", pt)
				return f.evade(ctx, me, pt)
			}
		}
	}

	// get into range, view and line of sight; decide per target whether it can be reached at all
	reach := spec.Range - t.RangeMargin
	if pr.MeleeClass() && spec.Kind == KindMelee && pr.Melee.Reach > 0 {
		reach = pr.Melee.Reach
	}
	if reach < 2 {
		reach = 2
	}
	// A melee click on a monster in sight walks the character to it by itself (the game paths around obstacles), so a
	// melee profile only walks when the target is beyond its sight range or off screen (live: zombies 4 subtiles away
	// behind a failing line-of-sight test were dropped after 12 approach steps).
	meleeGo := pr.MeleeClass() && spec.Kind == KindMelee && tg.Dist <= float64(spec.Range) && onScreen(snap, target)
	inReach := tg.Dist <= float64(reach)
	if spec.Kind != KindMelee && spec.Kind != KindSelf {
		inReach = tg.Dist <= float64(spec.Range) // the margin is for walking, a target inside the range is shot at
	}
	if !meleeGo && (!inReach || !onScreen(snap, target) || !f.los(me, target)) {
		if f.mv == nil {
			return nil
		}
		plan, fp := f.assess(tr, me, tg, reach)
		if plan == planNone {
			f.ignore(st, tg, IgnoreUnreachable, "no path and no firing position with a clear line of sight")
			return nil
		}
		if tr.approaches >= t.MaxApproaches {
			f.ignore(st, tg, IgnoreApproaches, fmt.Sprintf("%d approaches, %.0f subtiles away", tr.approaches, tg.Dist))
			return nil
		}
		tr.approaches++
		st.approach++
		f.decision(st, "approach", "no shot", "dist", int(tg.Dist), "plan", int(plan))
		if plan == planFire {
			return f.move(ctx, fp)
		}
		want := float64(reach - 1)
		if want > tg.Dist*0.75 {
			want = tg.Dist * 0.75
		}
		return f.move(ctx, approachPoint(me, target, want))
	}

	tr.inShot = true

	if spec.Hand == HandLeft && p.LeftSkill != spec.ID {
		if err := f.SetLeft(ctx, spec.ID); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			f.leftFail[spec.ID] = f.now().Add(time.Duration(t.LeftFailSec) * time.Second)
			f.info("left skill not selectable", "skill", spec.Name, "err", err)
			return nil
		}
	}

	// static field opener on elites and bosses
	if sf, ok := f.staticOpener(p, &tg, tr); ok {
		if !f.castReady() {
			f.decision(st, "wait", "cast gap")
			return nil
		}
		tr.statics++
		tr.casts++
		st.noteCast(now)
		tr.noDrop += tr.dt
		return f.cast(snap, sf, target)
	}

	if !f.castReady() {
		f.decision(st, "wait", "cast gap")
		return nil
	}
	tr.casts++
	st.noteCast(now)
	tr.noDrop += tr.dt
	st.spent += spec.Cost
	f.log().Debug("combat decision", "act", "cast", "skill", spec.Name, "target", tg.Name, "dist", int(tg.Dist), "hostiles", len(snap.Monsters))
	return f.cast(snap, spec, target)
}

// castReady keeps the minimum gap between two casts without blocking: the fight loop ticks in between, so
// survival checks never wait for a cast gap.
func (f *Fighter) castReady() bool {
	return f.now().Sub(f.lastCast) >= ms(f.tune().CastGapMs)
}

func (f *Fighter) staticOpener(p *d2.Player, tg *Target, tr *track) (SkillSpec, bool) {
	t := f.tune()
	if !(tg.Elite || tg.Boss) || tr.statics >= t.StaticCasts {
		return SkillSpec{}, false
	}
	if tg.Mon != nil && strings.Contains(strings.ToLower(tg.Mon.Key), "bloodraven") && tr.statics >= t.StaticCastsBR {
		return SkillSpec{}, false
	}
	sf, ok := KnownSkills(p, f.cat, f.keys)["Static Field"]
	if !ok || (sf.Key == "" && p.RightSkill != sf.ID) || sf.Cost > p.Mana || tg.Immune(sf.Elem) {
		return SkillSpec{}, false
	}
	if tg.LifePct <= staticFloor(t, p.Difficulty) || tg.Dist > float64(sf.Range) {
		return SkillSpec{}, false
	}
	return sf, true
}

// fallbackSkill: when every attack is blocked, Static Field still works on cold immunes (plan section 2).
func (f *Fighter) fallbackSkill(p *d2.Player, tg *Target) (SkillSpec, bool) {
	sf, ok := KnownSkills(p, f.cat, f.keys)["Static Field"]
	if !ok || (sf.Key == "" && p.RightSkill != sf.ID) || sf.Cost > p.Mana || tg.Immune(sf.Elem) {
		return SkillSpec{}, false
	}
	return sf, true
}

// kiteThreats are living melee monsters closer than dist.
func (f *Fighter) kiteThreats(snap *d2.Snapshot, p *d2.Player, d int) []Target {
	var out []Target
	for _, h := range Hostiles(snap, f.hostileCtx(p)) {
		if h.Dist >= float64(d) {
			break
		}
		if h.Mon != nil && h.Mon.Ranged {
			continue
		}
		out = append(out, h)
	}
	return out
}

func anyElite(ts []Target) bool {
	for _, t := range ts {
		if t.Elite || t.Boss {
			return true
		}
	}
	return false
}

// valid asks the mover whether a point is reachable.
func (f *Fighter) valid(p input.Point) bool {
	if f.mv == nil {
		return true
	}
	_, err := f.mv.PathTo(p)
	return err == nil
}

func anyDanger(ts []Target) bool {
	for _, t := range ts {
		if t.Danger >= 1 {
			return true
		}
	}
	return false
}

// move walks through the Mover. A failed walk (no path, stuck) is not fatal to the fight: the approach counter
// gives up on the target after a few of them.
func (f *Fighter) move(ctx context.Context, p input.Point) error {
	if g, ok := f.mv.(interface{ NearMouth(input.Point) bool }); ok && g.NearMouth(p) {
		f.info("move skipped: the point is in a cave mouth", "to", p) // the fight goes on from here; the monsters come
		return nil
	}
	err := f.mv.GoTo(ctx, p)
	if err != nil && ctx.Err() == nil {
		f.info("move failed", "to", p, "err", err)
		// a failed walk must still cost a tick: under a lockstep clock time only moves on sleeps and steps, and
		// "no path: empty path" returns at once (live: soakfull seed 11 and runab seed 42 spun here with time frozen)
		return f.sleep(ctx, ms(f.tune().PollMs))
	}
	return err
}

// pinRadius: an evasive move that ends within this many subtiles of where it started went nowhere.
const (
	pinRadius = 2
	pinMoves  = 2               // consecutive no-op evasive moves before giving up on evading
	pinHold   = 8 * time.Second // how long the fighter then stands and fights before trying to evade again
)

// mayEvade is false while the fighter is pinned (see evade): kite and dodge are suppressed so it casts instead.
func (f *Fighter) mayEvade() bool { return !f.now().Before(f.noEvadeUntil) }

// evade is move for kite and dodge steps. When two in a row leave the position unchanged the character is pinned in a
// crowd; moving again only stalls the fight (live: Countess, 10 min without a cast), so evading is off for pinHold and
// the fighter fights in place. The state lives on the Fighter because Kill and Clear are called in short slices.
func (f *Fighter) evade(ctx context.Context, from input.Point, to input.Point) error {
	err := f.move(ctx, to)
	if f.snapFn == nil {
		return err
	}
	s, e := f.snapFn()
	if e != nil || s == nil || s.Player == nil {
		return err
	}
	if pdist(from, input.Point{X: s.Player.X, Y: s.Player.Y}) > pinRadius {
		f.pinned = 0
		return err
	}
	if f.pinned++; f.pinned >= pinMoves {
		f.pinned = 0
		f.noEvadeUntil = f.now().Add(pinHold)
		f.info("pinned: evasive moves go nowhere, fighting in place", "at", from, "for", pinHold)
	}
	return err
}

// meleeGuard is the positioning of a melee profile: back out when surrounded and hurt, or when life is low (potions
// come first in the survival step). handled is true when the tick was spent walking away.
func (f *Fighter) meleeGuard(ctx context.Context, snap *d2.Snapshot, m *MeleeParams, st *fightState) (handled bool, err error) {
	t := f.tune()
	p := snap.Player
	now := f.now()
	if now.Sub(st.lastKite) > 10*time.Second {
		st.kites = 0
	}
	if st.kites >= t.KiteMaxSteps || !f.mayEvade() {
		return false, nil
	}
	life := pct(p.Life, p.MaxLife)
	var near, pts []input.Point
	for _, h := range Hostiles(snap, f.hostileCtx(p)) {
		hp := input.Point{X: h.Unit.X, Y: h.Unit.Y}
		if h.Dist <= 3.5 {
			near = append(near, hp)
		}
		if h.Dist <= 8 {
			pts = append(pts, hp)
		}
	}
	surrounded := m.Surround > 0 && len(near) >= m.Surround && life < m.SurroundLifePct
	low := m.RetreatLifePct > 0 && life <= m.RetreatLifePct && len(pts) > 0
	if !surrounded && !low {
		return false, nil
	}
	me := input.Point{X: p.X, Y: p.Y}
	pt, ok := awayPoint(me, pts, hostileMissiles(snap), t.KiteRetreat, f.valid)
	if !ok {
		return false, nil
	}
	st.kites++
	st.lastKite = now
	f.info("melee: backing off", "surrounded", surrounded, "life", life, "near", len(near), "to", pt)
	return true, f.evade(ctx, me, pt)
}

// lifeDropping reports whether life fell over the last couple of seconds (it also refreshes the reference).
func (st *fightState) lifeDropping(now time.Time, life int) bool {
	if st.lifeRefAt.IsZero() {
		st.lifeRef, st.lifeRefAt = life, now
		return false
	}
	dropping := life < st.lifeRef
	if now.Sub(st.lifeRefAt) >= 2*time.Second {
		st.lifeRef, st.lifeRefAt = life, now
	}
	return dropping
}

package combat

import (
	"context"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/resource"
)

// This file is the glue between the pure policy in internal/resource and the fight loop (docs/resource.md):
// PreEngage before a pack, AttackChoice when mana runs short, PotionPolicy for the mana potion, AbortFight when a
// fight is lost.

// meleeRange: the left click walks into reach by itself (live: a target 3 subtiles away was dropped after 12 approach steps
// when the range was 4), so the approach logic only has to get within sight; reach = meleeRange - RangeMargin.
const meleeRange = 10

// otherActFields are the first field of acts 2-5, one short walk from the town (S: standard levels.txt ids, not walked
// live). Act 1 comes from resource.NearTownArea so the fight loop and the trip policy agree (Blood Moor, Den, Cold Plains).
var otherActFields = map[int]bool{
	41:  true, // Rocky Waste (Lut Gholein)
	76:  true, // Spider Forest (Kurast Docks)
	104: true, // Outer Steppes (Pandemonium Fortress)
	110: true, // Bloody Foothills (Harrogath)
}

func nearTown(area int) bool { return resource.NearTownArea(area) || otherActFields[area] }

func meleeSpec() SkillSpec {
	return SkillSpec{ID: 0, Name: "Attack", Kind: KindMelee, Range: meleeRange, Hand: HandLeft}
}

func (f *Fighter) rtune() resource.Tuning { return resource.FromEnv(*f.e.Tune) }

// resSkills lists the attack skills for the policy: the key-bound spells in preference order without the ones the
// target is immune to, plus the basic attack when the left hand skill is "Attack" (id 0).
func (f *Fighter) resSkills(p *d2.Player, tg *Target, order []SkillSpec, weakPack bool) []resource.Skill {
	var out []resource.Skill
	for _, sp := range order {
		if tg != nil && tg.Immune(sp.Elem) {
			continue
		}
		out = append(out, resource.Skill{Name: sp.Name, Cost: sp.Cost, Ready: f.skillReady(sp)})
	}
	// A caster whose left skill is a spell can still swing when the pack is weak: the fight loop rebinds the left skill.
	if p.LeftSkill == 0 || weakPack {
		out = append(out, resource.Skill{Name: meleeSpec().Name, Melee: true, Ready: true})
	}
	return out
}

func (f *Fighter) resBelt(snap *d2.Snapshot) resource.Belt {
	b := BeltOf(snap.Inventory, f.cat)
	rb := resource.Belt{HP: b.Count(PotionHealing), MP: b.Count(PotionMana), RV: b.Count(PotionRejuv)}
	rb.Slots = BeltCapacity(snap.Inventory, f.e.Data)
	if snap.Inventory != nil {
		for _, u := range snap.Inventory.Container(d2.ContInventory) {
			switch f.cat.Item(u.Class).Potion {
			case PotionHealing:
				rb.PackHP++
			case PotionRejuv:
				rb.PackRV++
			}
		}
	}
	for _, c := range b.Cols {
		switch c.Kind {
		case PotionHealing:
			rb.HPTier = max(rb.HPTier, c.Tier)
		case PotionMana:
			rb.MPTier = max(rb.MPTier, c.Tier)
		case PotionRejuv:
			rb.RVTier = max(rb.RVTier, c.Tier)
		}
	}
	return rb
}

// resInput fills the policy input. hs are the hostiles in view (all of them, whatever their distance); pack is how
// many belong to the pack ahead.
func (f *Fighter) resInput(snap *d2.Snapshot, hs []Target, pack int, skills []resource.Skill) resource.Input {
	p := snap.Player
	in := resource.Input{
		Life: p.Life, MaxLife: p.MaxLife, Mana: p.Mana, MaxMana: p.MaxMana,
		Belt: f.resBelt(snap), InTown: p.InTown(), NearTown: nearTown(p.Area), Gold: p.Gold, LastTripGold: -1,
		Level: p.Level, Class: p.Class, Diff: p.Difficulty, Area: p.Area, PackSeen: pack, Skills: skills,
		SinceFight: 999,
	}
	if !f.lastCast.IsZero() {
		in.SinceFight = f.now().Sub(f.lastCast).Seconds()
	}
	if snap.Inventory != nil {
		for _, u := range snap.Inventory.Items {
			if u.Item != nil && f.cat.Item(u.Class).Portal {
				in.HasTP = true
				break
			}
		}
	}
	fr := float64(f.tune().FightRadius)
	for _, h := range hs {
		if h.Dist <= fr {
			in.Hostiles++
			if h.Champion && !h.Boss {
				in.ChampionPack = true
			}
		}
	}
	in.PackDanger = f.packDanger(p, hs)
	in.WeakPack = weakHostiles(p, hs, f.tune().FightRadius)
	return in
}

const weakPackMax = 8 // a bigger pack is never a staff-swing job

// weakHostiles reports whether every awake hostile inside the fight radius is at most one level above the character
// (monster level from monstats, U for areas that override it) and at least one is there. An unknown class is not weak.
func weakHostiles(p *d2.Player, hs []Target, radius int) bool {
	n := 0
	for _, h := range hs {
		if h.Dist > float64(radius) {
			continue
		}
		// Only a monster read as a plain normal one counts: champion/unique levels are +2/+3, minions follow a unique,
		// and an unread kind (offsets U) must not be taken for normal.
		if h.Mon == nil || h.Elite || h.Boss || h.Affixes != 0 || h.Unit.Monster.Kind() != d2.MonKindNormal ||
			h.Mon.Level[p.Difficulty] > p.Level+1 {
			return false
		}
		n++
	}
	return n > 0 && n <= weakPackMax
}

// resInputSnap is resInput for callers without a target (Safety): every skill is offered.
func (f *Fighter) resInputSnap(snap *d2.Snapshot) resource.Input {
	hs := Hostiles(snap, f.hostileCtx(snap.Player))
	return f.resInput(snap, hs, 0, f.resSkills(snap.Player, nil, f.attackSkills(snap.Player), false))
}

// preEngage runs the policy before a pack. handled is true when it spent the tick (rested, drank, retreated); an error
// ends the fight (ErrTownPortal for a free heal in town).
func (f *Fighter) preEngage(ctx context.Context, snap *d2.Snapshot, hs, cand []Target, st *fightState, targeted bool) (handled bool, err error) {
	now := f.now()
	if now.Before(st.skipPre) {
		return false, nil
	}
	p := snap.Player
	in := f.resInput(snap, hs, len(cand), f.resSkills(p, nil, f.attackSkills(p), false))
	// A named target (quest boss, unique) is fought whatever the danger: the poverty limit only thins out Clear.
	if pv := f.povertyNow(); pv.poor && !targeted {
		in.Poor, in.MaxPackDanger = true, pv.maxDanger
	}
	rt := f.rtune()
	d := resource.PreEngage(in, rt)
	// A melee profile ends its attack ladder in the free basic attack, so resting for a cast budget is wasted time
	// (live: 15-20 s per pack on Palli). Life rests and potions still apply.
	if f.profileFor(p).MeleeClass() && d.TargetMana > 0 && (d.Action == resource.RestHere || d.Action == resource.RetreatRest) {
		return false, nil
	}
	if f.armyStands(snap) && d.TargetMana > 0 && (d.Action == resource.RestHere || d.Action == resource.RetreatRest) {
		return false, nil
	}
	if d.Action != resource.RestHere && d.Action != resource.RetreatRest {
		st.restFrom, st.retreated = time.Time{}, false
	}
	// A hostile already in reach and mana for a cast: fight, do not rest (cast-first loop).
	if (d.Action == resource.RestHere || d.Action == resource.RetreatRest) && len(cand) > 0 && cand[0].Dist <= float64(f.castRange(p)) {
		for _, sp := range f.attackSkills(p) {
			if sp.Kind != KindSelf && sp.Kind != KindBuff && sp.Cost <= p.Mana {
				return false, nil
			}
		}
	}
	switch d.Action {
	case resource.Proceed:
		return false, nil
	case resource.RestHere, resource.RetreatRest:
		if st.restFrom.IsZero() {
			st.restFrom = now
			f.info("resting before the pack", "action", d.Action.String(), "mana", p.Mana, "target", d.TargetMana, "waitSec", d.WaitSec, "why", d.Reason)
		}
		// WaitSec is an estimate from the U regeneration figure; live regeneration was slower (docs/bot.md), so the
		// timeout is half as long again, never beyond RestMaxSec.
		limit := min(max(d.WaitSec*3/2+3, 1), rt.RestMaxSec)
		if now.Sub(st.restFrom) > time.Duration(limit)*time.Second {
			f.info("rest timed out, engaging", "mana", p.Mana, "target", d.TargetMana)
			st.skipPre = now.Add(20 * time.Second)
			st.restFrom, st.retreated = time.Time{}, false
			return false, nil
		}
		if d.Action == resource.RetreatRest && !st.retreated && f.mv != nil {
			st.retreated = true
			var pts []input.Point
			for _, c := range cand {
				pts = append(pts, input.Point{X: c.Unit.X, Y: c.Unit.Y})
			}
			me := input.Point{X: p.X, Y: p.Y}
			if pt, ok := awayPoint(me, pts, hostileMissiles(snap), f.tune().KiteRetreat*2, f.valid); ok {
				f.info("retreating to rest", "to", pt)
				return true, f.move(ctx, pt)
			}
		}
		f.restAura(ctx, snap, cand)
		return true, f.sleep(ctx, 1500*time.Millisecond)
	case resource.DrinkMana, resource.DrinkLife:
		a := DrinkMana
		if d.Action == resource.DrinkLife {
			a = DrinkLife
		}
		if aerr := f.Act(ctx, a); aerr != nil {
			f.info("pre-engage drink failed", "err", aerr)
			st.skipPre = now.Add(30 * time.Second)
			return false, nil
		}
		return true, f.sleep(ctx, ms(f.tune().PotionGapMs))
	case resource.TownHeal:
		if aerr := f.Act(ctx, TownPortal); aerr != nil {
			f.info("town heal unavailable", "err", aerr)
			st.skipPre = now.Add(60 * time.Second)
			return false, nil
		}
		f.info("town portal for a free heal", "life", p.Life, "why", d.Reason)
		return true, ErrTownPortal
	case resource.Avoid:
		for _, c := range cand {
			if !st.avoided[c.Unit.ID] {
				st.avoided[c.Unit.ID] = true
				f.ignore(st, c, IgnoreAvoid, "pre-engage: "+d.Reason)
			}
		}
		return true, nil
	}
	return false, nil
}

// fleeMax bounds one unbroken break-away (no potion, no portal).
const fleeMax = 60 * time.Second

// abortFight runs AbortFight after the survival step found nothing to drink. handled is true when it acted.
func (f *Fighter) abortFight(ctx context.Context, snap *d2.Snapshot, hs []Target, st *fightState) (handled bool, err error) {
	p := snap.Player
	in := f.resInput(snap, hs, 0, f.resSkills(p, nil, f.attackSkills(p), false))
	d := resource.AbortFight(in, f.rtune())
	if d.Kind == resource.Flee {
		// Without potion or portal the life barely regenerates: a break-away that never ends only delays the
		// chicken. Past fleeMax of unbroken fleeing, leave the game deliberately.
		now := f.now()
		if st.fleeLast.IsZero() || now.Sub(st.fleeLast) > 10*time.Second {
			st.fleeFrom = now
		}
		st.fleeLast = now
		if now.Sub(st.fleeFrom) > fleeMax {
			d = resource.AbortDecision{Kind: resource.LeaveGame, Reason: "fled too long without a potion or portal"}
		}
	}
	switch d.Kind {
	case resource.PortalOut:
		if aerr := f.Act(ctx, TownPortal); aerr == nil {
			f.info("abort fight: portal", "why", d.Reason)
			return true, ErrTownPortal
		}
	case resource.LeaveGame:
		f.info("abort fight: leaving", "why", d.Reason)
		if aerr := f.Act(ctx, Chicken); aerr != nil {
			return true, aerr
		}
		return true, ErrChicken
	case resource.RetreatOut, resource.Flee:
		step := f.tune().KiteRetreat
		if d.Kind == resource.Flee {
			step *= 3 // a real break-away, not a kite step; it repeats every tick while the hostiles follow
		}
		me := input.Point{X: p.X, Y: p.Y}
		var pts []input.Point
		for _, h := range hs {
			if h.Dist <= float64(f.tune().FightRadius) {
				pts = append(pts, input.Point{X: h.Unit.X, Y: h.Unit.Y})
			}
		}
		if f.mv != nil {
			if pt, ok := awayPoint(me, pts, hostileMissiles(snap), step, f.valid); ok {
				f.info("abort fight: "+d.Kind.String(), "why", d.Reason, "to", pt)
				return true, f.move(ctx, pt)
			}
		}
	}
	return false, nil
}

// lowMana is the AttackChoice step of engage: called when PickAttack found nothing affordable or mana is under the
// economy line. use is the skill to cast (the caller carries on); otherwise the tick was spent here.
func (f *Fighter) lowMana(ctx context.Context, snap *d2.Snapshot, tg Target, order []SkillSpec, hs []Target, tr *track, st *fightState) (use SkillSpec, ok bool, err error) {
	p := snap.Player
	rt := f.rtune()
	capped := rt.BackoffMaxSteps > 0 && st.backoffs >= rt.BackoffMaxSteps
	in := f.resInput(snap, hs, 0, f.resSkills(p, &tg, order, capped || weakHostiles(p, hs, f.tune().FightRadius)))
	in.BackoffSteps = st.backoffs
	d := resource.AttackChoice(in, rt)
	switch d.Kind {
	case resource.Cast, resource.Free:
		for _, sp := range order {
			if sp.Name == d.Skill.Name {
				st.waitFrom = time.Time{}
				st.backoffs = 0
				return sp, true, nil
			}
		}
	case resource.Melee:
		st.waitFrom = time.Time{}
		if capped {
			f.info("out of mana: back-off cap, melee", "steps", st.backoffs, "mana", p.Mana, "hostiles", len(hs))
		}
		return meleeSpec(), true, nil
	case resource.Wait:
		if st.waitFrom.IsZero() {
			st.waitFrom = f.now()
			f.info("out of mana: waiting", "mana", p.Mana, "why", d.Reason)
		}
		if f.now().Sub(st.waitFrom) <= time.Duration(rt.WaitMaxSec)*time.Second {
			if tp := (input.Point{X: tg.Unit.X, Y: tg.Unit.Y}); onScreen(snap, tp) && f.los(input.Point{X: p.X, Y: p.Y}, tp) {
				tr.inShot = true // waiting for mana, not for a way to the target
			}
			return SkillSpec{}, false, f.sleep(ctx, ms(d.WaitMs))
		}
		fallthrough
	case resource.Retreat:
		st.waitFrom = time.Time{}
		me := input.Point{X: p.X, Y: p.Y}
		var pts []input.Point
		for _, h := range hs {
			if h.Dist <= float64(f.tune().FightRadius) {
				pts = append(pts, input.Point{X: h.Unit.X, Y: h.Unit.Y})
			}
		}
		if f.mv != nil {
			if pt, found := awayPoint(me, pts, hostileMissiles(snap), f.tune().KiteRetreat, f.valid); found {
				st.backoffs++
				f.info("out of mana: backing off", "to", pt, "why", d.Reason, "step", st.backoffs)
				return SkillSpec{}, false, f.move(ctx, pt)
			}
		}
	}
	return SkillSpec{}, false, f.sleep(ctx, ms(f.tune().LowManaWaitMs))
}

// poverty is the bot's poverty verdict as the fight loop needs it.
type poverty struct {
	poor      bool
	maxDanger int
}

// SetPoverty tells the fighter that the character is poor (no gold for potions): packs rated above maxPackDanger
// (resource.Input.PackDanger scale) are then skipped while the belt holds no healing potion. Safe to call from the
// bot goroutine while a fight runs.
func (f *Fighter) SetPoverty(poor bool, maxPackDanger int) {
	f.povMu.Lock()
	f.pov = poverty{poor, maxPackDanger}
	f.povMu.Unlock()
}

func (f *Fighter) povertyNow() poverty {
	f.povMu.Lock()
	defer f.povMu.Unlock()
	return f.pov
}

// restAuraAdjacent is the distance (subtiles) under which a hostile counts as adjacent: no resting aura then.
const restAuraAdjacent = 4

// restAura switches to the resting aura (Prayer) while the bot waits for life or mana, unless a hostile is adjacent.
// Aura changes are a key press with no cast time (V, aura.go); the fight tick puts Might back at the first target.
func (f *Fighter) restAura(ctx context.Context, snap *d2.Snapshot, cand []Target) {
	for _, c := range cand {
		if c.Dist <= restAuraAdjacent {
			return
		}
	}
	if _, err := f.keepAura(ctx, snap, nil, true, true); err != nil {
		f.info("rest aura failed", "err", err)
	}
}

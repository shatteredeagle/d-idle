package combat

import (
	"context"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/input"
)

const (
	trapMinDist = 3 // closer than this the trap lands on our own feet (U)
	trapCover   = 5 // a standing trap this close to the target already burns it (U radius)
	trapLaidSec = 3 // after a cast the trap may need a moment to show in the snapshot
	maxTraps    = 5 // V docs/classes/assassin.md (Kolbot): five sentries at once, the sixth replaces the oldest
)

// pickTrapTarget chooses where to lay a trap: an elite or boss first, else the hostile with the most others within
// packRadius, in trap range, above 25% life, not immune to the trap's element, with no standing trap within
// trapCover. Nearest on ties.
func pickTrapTarget(hs []Target, traps []d2.Unit, elem data.DamageType, rng int) (Target, bool) {
	var best Target
	bestScore, found := -1, false
	for _, t := range hs {
		if t.Dist > float64(rng) || t.Dist < trapMinDist || t.LifePct < 25 || t.Immune(elem) {
			continue
		}
		covered := false
		for _, tr := range traps {
			if dist(t.Unit.X, t.Unit.Y, tr.X, tr.Y) <= trapCover {
				covered = true
			}
		}
		if covered {
			continue
		}
		n := 0
		for _, o := range hs {
			if o.Unit.ID != t.Unit.ID && dist(t.Unit.X, t.Unit.Y, o.Unit.X, o.Unit.Y) <= packRadius {
				n++
			}
		}
		score := n*10 - int(t.Dist/4)
		if t.Boss || t.Elite {
			score += 100
		}
		if score > bestScore {
			best, bestScore, found = t, score, true
		}
	}
	return best, found
}

// assassinTick lays one trap per tick (trap.go: planTrap). The trap skills are not in the profile attack list (a
// ground cast per tick would burn mana): the census and the cast memory say what already stands. true = acted.
func (f *Fighter) assassinTick(ctx context.Context, snap *d2.Snapshot) (bool, error) {
	p := snap.Player
	if !isAssassin(p) || p.InTown() {
		return false, nil
	}
	now := f.now()
	standing := f.trap.observe(now, CountAssassinPets(snap, f.cat).Traps)
	hs := Hostiles(snap, f.hostileCtx(p))
	if len(hs) == 0 {
		return false, nil
	}
	me := input.Point{X: p.X, Y: p.Y}
	known := KnownSkills(p, f.cat, f.keys)
	var corpses []d2.Unit
	if _, ok := known["Death Sentry"]; ok {
		corpses = Corpses(snap, f.cat, me, 30, nil)
	}
	ready := func(sp SkillSpec) bool { return sp.Key != "" || p.RightSkill == sp.ID }
	pl, ok := planTrap(me, known, p.Mana, hs, corpses, standing, f.trap.pending(now), f.cat, ready)
	if !ok {
		return false, nil
	}
	if err := f.cast(snap, pl.sp, pl.at); err != nil {
		return false, err
	}
	f.trap.laid = append(f.trap.laid, trapLay{skill: pl.sp.Name, at: now, x: pl.at.X, y: pl.at.Y})
	f.info("trap", "skill", pl.sp.Name, "target", pl.target, "dist", int(pl.dist), "standing", len(standing))
	return true, f.sleep(ctx, ms(f.tune().CastGapMs))
}

// KeepShadow casts Shadow Warrior (or Shadow Master) when no shadow stands: out of a fight, never in town. Shadows
// share the trap cap in the real game (U, docs/classes/assassin.md), so only one is kept. Returns the casts made.
func (f *Fighter) KeepShadow(ctx context.Context) (int, error) {
	snap, err := f.read(ctx)
	if err != nil {
		return 0, err
	}
	p := snap.Player
	if !isAssassin(p) || p.InTown() || isDead(p) || CountAssassinPets(snap, f.cat).Shadows > 0 {
		return 0, nil
	}
	known := KnownSkills(p, f.cat, f.keys)
	for _, name := range []string{"Shadow Master", "Shadow Warrior"} {
		sp, ok := known[name]
		if !ok || (sp.Key == "" && p.RightSkill != sp.ID) || sp.Cost > p.Mana-p.MaxMana/10 {
			continue
		}
		if err := f.cast(snap, sp, input.Point{X: p.X, Y: p.Y}); err != nil {
			return 0, err
		}
		f.info("shadow", "skill", sp.Name)
		return 1, f.sleep(ctx, ms(f.tune().CastGapMs)+ms(500))
	}
	return 0, nil
}

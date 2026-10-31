package combat

import (
	"context"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// Corpse spells of the Necromancer (W5, docs/summon.md): Revive takes the best corpse, Corpse Explosion the corpse
// with the most hostiles around it. Skill ids come from skills.txt through KnownSkills (by name), so an unknown or
// unlearned skill is simply never planned.
const (
	// ceRadius: hostiles within this of a corpse count as hit by its explosion. U: the real radius is a skills.txt
	// calc column that was not read.
	ceRadius = 6
	ceReach  = 20 // fallback cast range when skills.json has none
)

// monLevel ranks a corpse by its monster class (monstats level, highest difficulty entry); unknown class = 0.
func monLevel(c Catalog, class int) int {
	if c == nil {
		return 0
	}
	m := c.Monster(class)
	if m == nil {
		return 0
	}
	return max(m.Level[0], m.Level[1], m.Level[2])
}

// bestRevivable picks the corpse to revive from a nearest-first list: the highest monster level, nearest on ties
// (U: a higher level monster is a better minion; the monstats revive restrictions are not read). cs must not be empty.
func bestRevivable(cs []d2.Unit, c Catalog) d2.Unit {
	best := cs[0]
	for _, u := range cs[1:] {
		if monLevel(c, u.Class) > monLevel(c, best.Class) {
			best = u
		}
	}
	return best
}

// pickExplosionCorpse chooses the corpse to explode: most hostiles within ceRadius (a boss or elite counts as 3),
// at least minHit, level of the dead monster as a tie breaker (U: damage follows the corpse's life). Only corpses
// within reach of me qualify. Pure.
func pickExplosionCorpse(cs []d2.Unit, hs []Target, c Catalog, minHit int) (d2.Unit, bool) {
	var best d2.Unit
	bestScore, found := 0, false
	for _, u := range cs {
		hit := 0
		for _, h := range hs {
			if h.Unit.Mode == d2.MonModeDead || h.Unit.Mode == d2.MonModeDeath || dist(u.X, u.Y, h.Unit.X, h.Unit.Y) > ceRadius {
				continue
			}
			hit++
			if h.Boss || h.Elite {
				hit += 2
			}
		}
		if hit < minHit {
			continue
		}
		if score := hit*100 + monLevel(c, u.Class); score > bestScore {
			best, bestScore, found = u, score, true
		}
	}
	return best, found
}

// corpseExplode casts Corpse Explosion at the best corpse, once. It runs only when no summon is wanted (necroTick
// asks planSummon first), so it never eats a corpse the army needs. A corpse is used once (the id is blacklisted).
func (f *Fighter) corpseExplode(ctx context.Context, snap *d2.Snapshot, hs []Target) (bool, error) {
	p := snap.Player
	f.necro.init()
	sp, ok := KnownSkills(p, f.cat, f.keys)["Corpse Explosion"]
	if !ok || (sp.Key == "" && p.RightSkill != sp.ID) || sp.Cost*2 > p.Mana || len(hs) == 0 {
		return false, nil
	}
	rng := sp.Range
	if rng == 0 {
		rng = ceReach
	}
	cs := Corpses(snap, f.cat, input.Point{X: p.X, Y: p.Y}, float64(rng), f.necro.tried)
	c, ok := pickExplosionCorpse(cs, hs, f.cat, 2)
	if !ok {
		return false, nil
	}
	if err := f.cast(snap, sp, input.Point{X: c.X, Y: c.Y}); err != nil {
		return false, err
	}
	f.necro.tried[c.ID] = true
	f.info("corpse-explosion", "corpse", c.ID, "class", c.Class)
	return true, f.sleep(ctx, ms(f.tune().CastGapMs))
}

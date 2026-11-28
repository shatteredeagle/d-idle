package combat

import (
	"context"
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// Corpse aims (card dlv-aim-corpse; docs/design/fable-review-2026-11-21/combat-engine-d-cards.md, wave 2). Which corpse
// a corpse spell takes is a profile row (Profile.CorpseAims), Go names no skill and no class:
//   - corpseNearest: the nearest corpse in reach (the raise skills).
//   - corpseBest: the corpse of the highest monster level, nearest on ties (Revive).
//   - corpseNearPack: the corpse with the most hostiles within Radius, a boss or elite counting as 3, at least MinHit
//     of them (Corpse Explosion, Death Sentry).
//
// One reservation (corpseRes, Fighter.necro.tried) is shared by raise, revive, explode and Death Sentry: a corpse any of
// them was aimed at is taken and no other cast sees it. Knob combat.deliveryUpkeep (shared with the other drivers): 0 and
// 1 pick as the caller's built-in default row always did (1 also counts where the profile's row would pick another
// corpse), 2 the profile's row picks. A row's refusal (no corpse meets MinHit) skips the cast.
const (
	AimCorpseNearest  = "corpseNearest"
	AimCorpseBest     = "corpseBest"
	AimCorpseNearPack = "corpseNearPack"
)

// CorpseAim is one corpse aim row of a profile: which corpse the named skill takes.
type CorpseAim struct {
	Name string `json:"name"`
	Rule string `json:"rule"`
	// Radius is the area counted around a corpse by corpseNearPack, in subtiles (0: ceRadius).
	Radius int `json:"radius,omitempty"`
	// MinHit is the fewest hostiles in that area that justify the cast (0: 1).
	MinHit int    `json:"minHit,omitempty"`
	Source string `json:"source"` // V, S or U first: where Radius and MinHit come from
}

var corpseRules = map[string]bool{AimCorpseNearest: true, AimCorpseBest: true, AimCorpseNearPack: true}

// validateCorpseAims checks the corpse rows at load time against the skill table.
func validateCorpseAims(ps []Profile, skills map[string]skillMeta) error {
	for i := range ps {
		p := &ps[i]
		seen := map[string]bool{}
		for _, r := range p.CorpseAims {
			switch {
			case !corpseRules[r.Rule]:
				return fmt.Errorf("profile %q: corpse aim row %q: rule %q unknown", p.ProfileID(), r.Name, r.Rule)
			case seen[r.Name]:
				return fmt.Errorf("profile %q: corpse aim row %q twice", p.ProfileID(), r.Name)
			case r.Radius < 0 || r.MinHit < 0:
				return fmt.Errorf("profile %q: corpse aim row %q: radius and minHit must be >= 0", p.ProfileID(), r.Name)
			case !validSource(r.Source):
				return fmt.Errorf("profile %q: corpse aim row %q needs a source tag (V, S or U)", p.ProfileID(), r.Name)
			}
			if _, ok := skills[r.Name]; !ok {
				return fmt.Errorf("profile %q: corpse aim row %q names no skill of the table", p.ProfileID(), r.Name)
			}
			seen[r.Name] = true
		}
	}
	return nil
}

// corpseRes is the shared corpse reservation: a taken corpse is skipped by Corpses (a used corpse vanishes; V a second
// cast spent no mana).
type corpseRes map[uint32]bool

// Defaults of the built-in rows, today's behaviour of each caller.
var (
	corpseRowNearest = CorpseAim{Rule: AimCorpseNearest}
	corpseRowBest    = CorpseAim{Rule: AimCorpseBest}
	corpseRowExplode = CorpseAim{Rule: AimCorpseNearPack, Radius: ceRadius, MinHit: 2}
	corpseRowSentry  = CorpseAim{Rule: AimCorpseNearPack, Radius: ceRadius, MinHit: 1}
)

// corpseHold is how long an aimed-at corpse stays reserved. A cast that landed consumed its corpse, so one still in the
// snapshot after this was not used (the cast was cut or the walk failed) and is offered again. U judgment: a few cast
// times, long enough for the corpse of a landed cast to leave the snapshot.
const corpseHold = 4 * time.Second

// takeCorpse reserves the corpse id for every corpse spell. Id 0 is no corpse (a lay point without one).
func (f *Fighter) takeCorpse(id uint32) {
	if id == 0 {
		return
	}
	f.necro.init()
	f.necro.tried[id] = true
	f.necro.triedAt[id] = f.now()
}

// takenCorpses is the reservation for area: reservations older than corpseHold are released, and another area (or a
// new game, which starts in town) drops them all, since unit ids repeat.
func (f *Fighter) takenCorpses(area int) corpseRes {
	n := &f.necro
	n.init()
	if area != n.triedArea {
		clear(n.tried)
		clear(n.triedAt)
		n.triedArea = area
	}
	now := f.now()
	for id, at := range n.triedAt {
		if now.Sub(at) >= corpseHold {
			delete(n.tried, id)
			delete(n.triedAt, id)
		}
	}
	return n.tried
}

// CorpseCounts are the corpse driver's counters: picks by a profile row (mode 2), refusals, and in shadow (mode 1) the
// casts the row would have aimed at another corpse.
type CorpseCounts struct{ Picked, Refused, ShadowMoved int }

// CorpseCounts reports the corpse driver's counters since the fighter started.
func (f *Fighter) CorpseCounts() CorpseCounts {
	a := &f.aim
	return CorpseCounts{a.corpsePicked, a.corpseRefused, a.corpseShadow}
}

func (f *Fighter) corpseRow(p *d2.Player, name string) *CorpseAim {
	pr := f.profileFor(p)
	for i := range pr.CorpseAims {
		if pr.CorpseAims[i].Name == name {
			return &pr.CorpseAims[i]
		}
	}
	return nil
}

// corpseFor is the corpse skill name aims at: the profile's row at mode 2, def (the caller's built-in row) otherwise.
// cs is the nearest-first list of free corpses in reach. ok false: no corpse qualifies, the cast is skipped.
func (f *Fighter) corpseFor(p *d2.Player, name string, def CorpseAim, cs []d2.Unit, hs []Target) (d2.Unit, bool) {
	r := f.corpseRow(p, name)
	mode := f.upkeepMode()
	if r == nil || mode == 0 {
		return pickCorpse(def, cs, hs, f.cat)
	}
	alt, altOK := pickCorpse(*r, cs, hs, f.cat)
	if mode == 1 {
		c, ok := pickCorpse(def, cs, hs, f.cat)
		if ok != altOK || c.ID != alt.ID {
			f.aim.corpseShadow++
		}
		return c, ok
	}
	if altOK {
		f.aim.corpsePicked++
	} else {
		f.aim.corpseRefused++
	}
	return alt, altOK
}

// pickCorpse applies one row to the free corpses cs (nearest first). Pure.
func pickCorpse(r CorpseAim, cs []d2.Unit, hs []Target, c Catalog) (d2.Unit, bool) {
	if len(cs) == 0 {
		return d2.Unit{}, false
	}
	switch r.Rule {
	case AimCorpseNearest:
		return cs[0], true
	case AimCorpseBest:
		return bestRevivable(cs, c), true
	case AimCorpseNearPack:
		radius, minHit := float64(ceRadius), max(r.MinHit, 1)
		if r.Radius > 0 {
			radius = float64(r.Radius)
		}
		return nearPackCorpse(cs, hs, c, radius, minHit)
	}
	return d2.Unit{}, false
}

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

// nearPackCorpse chooses the corpse with the most hostiles within radius (a boss or elite counts as 3), at least
// minHit, level of the dead monster as a tie breaker (U: damage follows the corpse's life). Pure.
func nearPackCorpse(cs []d2.Unit, hs []Target, c Catalog, radius float64, minHit int) (d2.Unit, bool) {
	var best d2.Unit
	bestScore, found := 0, false
	for _, u := range cs {
		hit := 0
		for _, h := range hs {
			if h.Unit.Mode == d2.MonModeDead || h.Unit.Mode == d2.MonModeDeath || dist(u.X, u.Y, h.Unit.X, h.Unit.Y) > radius {
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
func (f *Fighter) corpseExplode(ctx context.Context, snap *d2.Snapshot, nd *NecroParams, hs []Target) (bool, error) {
	p := snap.Player
	f.necro.init()
	if nd.Explosion == "" {
		return false, nil
	}
	sp, ok := KnownSkills(p, f.cat, f.keys)[nd.Explosion]
	if !ok || (sp.Key == "" && p.RightSkill != sp.ID) || sp.Cost*2 > p.Mana || len(hs) == 0 {
		return false, nil
	}
	rng := sp.Range
	if rng == 0 {
		rng = ceReach
	}
	cs := Corpses(snap, f.cat, input.Point{X: p.X, Y: p.Y}, float64(rng), f.takenCorpses(p.Area))
	c, ok := f.corpseFor(p, nd.Explosion, corpseRowExplode, cs, hs)
	if !ok {
		return false, nil
	}
	if err := f.cast(snap, sp, input.Point{X: c.X, Y: c.Y}); err != nil {
		return false, err
	}
	f.takeCorpse(c.ID)
	f.info("corpse-explosion", "corpse", c.ID, "class", c.Class)
	return true, f.sleep(ctx, ms(f.tune().CastGapMs))
}

// corpseAt is the id of the corpse standing at p (0: none).
func corpseAt(cs []d2.Unit, p input.Point) uint32 {
	for _, c := range cs {
		if c.X == p.X && c.Y == p.Y {
			return c.ID
		}
	}
	return 0
}

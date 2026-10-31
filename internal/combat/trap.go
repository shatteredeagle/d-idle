package combat

import (
	"sort"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// Assassin trap planner (W6, docs/summon.md). Pure planning on top of a small memory (trapMem): the census of
// friendly trap units, per-skill caps, a lay point at the pack, Death Sentry on corpses and an age per trap.
//
// UNVERIFIED LIVE (no Assassin ever laid a trap): which monstats class each sentry skill produces, the trap lifetime,
// and that the sentries show in snap.Monsters at all. Everything that depends on them is isolated here and degrades:
//   - trapClassGuess is empty on purpose. The class of a skill is learned at run time (a new trap unit next to our own
//     cast); a trap of an unlearned class counts only for coverage, never for a skill's cap.
//   - when no cast was ever matched by a unit (ids wrong, sentries not listed), casts are remembered for trapLifeSec
//     instead of the 3 s spawn grace, so the cap of five per skill still holds.
const (
	trapLifeSec  = 30 // U: how long a sentry stands (the real value is a skills.txt duration, not read)
	trapRenewSec = 4  // a trap this close to its end no longer counts as standing
	trapMatch    = 3  // a unit this close to a remembered cast is that cast's trap (U radius)
	trapPackMax  = 8  // hostiles within this of the target form the pack the lay point is centred on
)

// trapClassGuess maps a monstats class id (410-418 are the AssassinSentry family, V friendly.go) to the skill that
// makes it. Empty: UNVERIFIED, filled by learning. A live run fills this from `trap learned` log lines.
var trapClassGuess = map[int]string{}

// trapLadder is the tie-break order of trap skills of equal level (learned and bound ones only).
var trapLadder = []string{"Death Sentry", "Wake of Fire Sentry", "Lightning Sentry", "Charged Bolt Sentry"}

type trapLay struct {
	skill string
	at    time.Time
	x, y  int
}

// trapMem is the Fighter's memory of its traps (zero value usable after init).
type trapMem struct {
	classSkill map[int]string
	seen       map[uint32]time.Time // first sighting of a trap unit: its age
	laid       []trapLay            // casts not yet matched by a unit
	visible    bool                 // a trap unit was matched to a cast of ours: the census can be trusted
}

func (m *trapMem) init() {
	if m.classSkill == nil {
		m.classSkill, m.seen = map[int]string{}, map[uint32]time.Time{}
		for c, s := range trapClassGuess {
			m.classSkill[c] = s
		}
	}
}

type standingTrap struct {
	u     d2.Unit
	skill string // "" = class not known
	age   time.Duration
}

// observe folds one census into the memory: first-seen stamps, learned classes, matched casts dropped.
func (m *trapMem) observe(now time.Time, traps []d2.Unit) []standingTrap {
	m.init()
	live := map[uint32]bool{}
	var out []standingTrap
	for _, u := range traps {
		live[u.ID] = true
		first, old := m.seen[u.ID]
		if !old {
			m.seen[u.ID], first = now, now
			m.match(u)
		}
		out = append(out, standingTrap{u: u, skill: m.classSkill[u.Class], age: now.Sub(first)})
	}
	for id := range m.seen {
		if !live[id] {
			delete(m.seen, id)
		}
	}
	keep := m.laid[:0]
	for _, l := range m.laid {
		if now.Sub(l.at) < trapLifeSec*time.Second {
			keep = append(keep, l)
		}
	}
	m.laid = keep
	return out
}

// match pairs a newly seen trap unit with the one cast of ours it stands next to: the cast is dropped (the unit
// counts now) and the unit's class is learned for that skill when still unknown. Two casts within reach of one
// unit are ambiguous: nothing is learned and the nearer one is dropped.
func (m *trapMem) match(u d2.Unit) {
	var near []int
	for i, l := range m.laid {
		if dist(u.X, u.Y, l.x, l.y) <= trapMatch {
			near = append(near, i)
		}
	}
	if len(near) == 0 {
		return
	}
	m.visible = true
	sort.Slice(near, func(a, b int) bool {
		la, lb := m.laid[near[a]], m.laid[near[b]]
		return dist(u.X, u.Y, la.x, la.y) < dist(u.X, u.Y, lb.x, lb.y)
	})
	i := near[0]
	if _, known := m.classSkill[u.Class]; !known && len(near) == 1 {
		m.classSkill[u.Class] = m.laid[i].skill
	}
	m.laid = append(m.laid[:i], m.laid[i+1:]...)
}

// pendingFor lists the unmatched casts that still count as a trap for skill: within the spawn grace, or for the
// whole lifetime while no trap was ever seen (census untrusted).
func (m *trapMem) pending(now time.Time) []trapLay {
	var out []trapLay
	for _, l := range m.laid {
		if a := now.Sub(l.at); a < trapLaidSec*time.Second || (!m.visible && a < trapLifeSec*time.Second) {
			out = append(out, l)
		}
	}
	return out
}

func expiring(age time.Duration) bool { return age >= (trapLifeSec-trapRenewSec)*time.Second }

// trapPlan is one lay: the skill, where to cast and the target it serves.
type trapPlan struct {
	sp     SkillSpec
	at     input.Point
	target string
	dist   float64
}

// trapState counts for a skill: standing traps of that skill (not about to expire), the unmatched casts, and the
// covering points (same skill, unknown class, pending).
func trapCounts(skill string, st []standingTrap, pend []trapLay) (n int, cover []d2.Unit) {
	for _, s := range st {
		if s.skill != "" && s.skill != skill {
			continue
		}
		if expiring(s.age) {
			continue
		}
		cover = append(cover, s.u)
		if s.skill == skill {
			n++
		}
	}
	for _, l := range pend {
		if l.skill == skill {
			n++
			cover = append(cover, d2.Unit{X: l.x, Y: l.y})
		}
	}
	return n, cover
}

// packCentre is the centroid of the hostiles within trapPackMax of t (t included), clamped to t when the centre
// would fall outside [trapMinDist, rng] from me.
func packCentre(me input.Point, t Target, hs []Target, rng int) input.Point {
	sx, sy, n := t.Unit.X, t.Unit.Y, 1
	for _, o := range hs {
		if o.Unit.ID != t.Unit.ID && o.Unit.Mode != d2.MonModeDead && dist(t.Unit.X, t.Unit.Y, o.Unit.X, o.Unit.Y) <= trapPackMax {
			sx, sy, n = sx+o.Unit.X, sy+o.Unit.Y, n+1
		}
	}
	c := input.Point{X: sx / n, Y: sy / n}
	if d := dist(me.X, me.Y, c.X, c.Y); d > float64(rng) || d < trapMinDist {
		return input.Point{X: t.Unit.X, Y: t.Unit.Y}
	}
	return c
}

// planTrap picks the next trap. Skills are tried by skill level (highest first, ladder order on ties); one at its
// cap of five is skipped. Death Sentry needs a corpse next to hostiles and is laid on the corpse. Others go to the
// pack (pickTrapTarget) and are cast at its centre. Pure.
func planTrap(me input.Point, known map[string]SkillSpec, mana int, hs []Target, corpses []d2.Unit,
	st []standingTrap, pend []trapLay, cat Catalog, ready func(SkillSpec) bool) (trapPlan, bool) {
	var order []SkillSpec
	for _, name := range trapLadder {
		if sp, ok := known[name]; ok && ready(sp) && sp.Cost*2 <= mana {
			order = append(order, sp)
		}
	}
	sort.SliceStable(order, func(i, j int) bool { return order[i].Level > order[j].Level })
	for _, sp := range order {
		n, cover := trapCounts(sp.Name, st, pend)
		if n >= maxTraps {
			continue
		}
		if sp.Name == "Death Sentry" {
			var cs []d2.Unit
			for _, c := range corpses {
				d := dist(me.X, me.Y, c.X, c.Y)
				if d >= trapMinDist && d <= float64(sp.Range) {
					cs = append(cs, c)
				}
			}
			if c, ok := pickExplosionCorpse(cs, hs, cat, 1); ok && !coveredBy(c.X, c.Y, cover) {
				return trapPlan{sp: sp, at: input.Point{X: c.X, Y: c.Y}, target: "corpse", dist: dist(me.X, me.Y, c.X, c.Y)}, true
			}
			continue
		}
		if tg, ok := pickTrapTarget(hs, cover, sp.Elem, sp.Range); ok {
			return trapPlan{sp: sp, at: packCentre(me, tg, hs, sp.Range), target: tg.Name, dist: tg.Dist}, true
		}
	}
	return trapPlan{}, false
}

func coveredBy(x, y int, traps []d2.Unit) bool {
	for _, t := range traps {
		if dist(x, y, t.X, t.Y) <= trapCover {
			return true
		}
	}
	return false
}

package combat

import (
	"fmt"
	"math"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// Point aim (card dlv-aim-point; docs/design/fable-review-2026-11-21/combat-engine-d-cards.md, wave 2). A point skill
// (Blizzard, Meteor, Eruption, a trap) lands where it is aimed, not on a unit. The aim rows are profile data
// (Profile.Aims); Go names no skill and no class:
//   - cluster: the point inside the cast range with the most hostiles within Radius (ties: nearer the target).
//   - lead: the target's cell plus its observed velocity times LeadFrames (25 frames a second), for a skill that hits
//     late (Meteor). No velocity known: the target's cell.
//   - packEdge: Radius subtiles from the pack's front (the pack hostile nearest the hero) toward the pack's centroid,
//     so the near rim of the area sits on the front and the rest covers the pack.
//   - offset: Offset subtiles past the target on the line from the hero (negative: short of it).
//
// Point scoring is a gate: fewer than CrowdAtLeast hostiles within Radius of the point refuses the cast. Line of sight
// is per row: missile (a clear line from the hero), walk (a reachable cell with a clear line), none (falls from the
// sky). A point that fails its line falls back to the target's own cell; both failing refuses.
//
// Knob combat.deliveryUpkeep (the delivery driver knob shared with upkeep, mark and stance): 0 every point skill is
// cast at the target's cell as before, 1 shadow (the old aim acts, the rule's point and refusals are counted and
// logged), 2 the rule aims and a refusal skips the cast in the fight loop. The watchdog (forceCast) aims too but never
// honours a crowd refusal, so the gate cannot stall a fight. No hit-rate claim before the fidelity A5 and A8 verdicts.

// Aim rule kinds.
const (
	AimCluster  = "cluster"
	AimLead     = "lead"
	AimPackEdge = "packEdge"
	AimOffset   = "offset"
)

// Line-of-sight kinds of an aim row.
const (
	AimLOSMissile = "missile"
	AimLOSWalk    = "walk"
	AimLOSNone    = "none"
)

const (
	aimRadiusDefault = 3  // U: area of a point skill without a Radius
	aimPackRange     = 8  // hostiles this close to the target form its pack (the trap planner's trapPackMax)
	aimFPS           = 25 // game frames per second (V: D2 runs 25 logic frames a second, docs/d2-1.14d-memory.md)
	aimVelMaxAge     = time.Second
)

// Aim refusal reasons (counted and logged; the scenario asserts refused_at_most on them).
const (
	aimRefusedCrowd = "crowd"
	aimRefusedLOS   = "los"
)

// AimRule is one aim row of a profile: where the named point skill lands.
type AimRule struct {
	Name string `json:"name"`
	Rule string `json:"rule"`
	// Radius is the skill's area in subtiles (0: aimRadiusDefault); it scores a point and places packEdge.
	Radius     int `json:"radius,omitempty"`
	LeadFrames int `json:"leadFrames,omitempty"` // lead: frames from the cast to the hit
	Offset     int `json:"offset,omitempty"`     // offset: subtiles past the target (negative: short of it)
	// CrowdAtLeast is the gate: hostiles within Radius of the point (0: no gate). A lead row counts the hostiles where
	// they will be after LeadFrames.
	CrowdAtLeast int    `json:"crowdAtLeast,omitempty"`
	LOS          string `json:"los,omitempty"` // missile, walk or none ("": missile)
	Source       string `json:"source"`        // V, S or U first: where Radius, LeadFrames and Offset come from
}

func (r AimRule) radius() float64 {
	if r.Radius > 0 {
		return float64(r.Radius)
	}
	return aimRadiusDefault
}

func (r AimRule) los() string {
	if r.LOS == "" {
		return AimLOSMissile
	}
	return r.LOS
}

var aimRules = map[string]bool{AimCluster: true, AimLead: true, AimPackEdge: true, AimOffset: true}
var aimLOS = map[string]bool{"": true, AimLOSMissile: true, AimLOSWalk: true, AimLOSNone: true}

// validateAims checks the aim rows at load time against the skill table: an error names the profile and the row. A
// row on a melee or self skill is an error (a melee build has no point to aim).
func validateAims(ps []Profile, skills map[string]skillMeta) error {
	for i := range ps {
		p := &ps[i]
		seen := map[string]bool{}
		for _, r := range p.Aims {
			m, ok := skills[r.Name]
			switch {
			case !ok:
				return fmt.Errorf("profile %q: aim row %q names no skill of the table", p.ProfileID(), r.Name)
			case m.Kind == KindMelee || m.Kind == KindSelf || m.Kind == KindBuff:
				return fmt.Errorf("profile %q: aim row %q on a skill with no point to aim", p.ProfileID(), r.Name)
			case seen[r.Name]:
				return fmt.Errorf("profile %q: aim row %q twice", p.ProfileID(), r.Name)
			case !aimRules[r.Rule]:
				return fmt.Errorf("profile %q: aim row %q: rule %q unknown", p.ProfileID(), r.Name, r.Rule)
			case !aimLOS[r.LOS]:
				return fmt.Errorf("profile %q: aim row %q: los %q unknown", p.ProfileID(), r.Name, r.LOS)
			case r.Radius < 0 || r.LeadFrames < 0 || r.CrowdAtLeast < 0:
				return fmt.Errorf("profile %q: aim row %q: radius, leadFrames and crowdAtLeast must be >= 0", p.ProfileID(), r.Name)
			case r.Rule == AimLead && r.LeadFrames == 0, r.Rule == AimOffset && r.Offset == 0:
				return fmt.Errorf("profile %q: aim row %q: rule %s needs its parameter", p.ProfileID(), r.Name, r.Rule)
			case !validSource(r.Source):
				return fmt.Errorf("profile %q: aim row %q needs a source tag (V, S or U)", p.ProfileID(), r.Name)
			}
			seen[r.Name] = true
		}
	}
	return nil
}

// aimIn is what one aim decision reads. Pure: the fighter fills it from the snapshot, its map and its memory.
type aimIn struct {
	Me       input.Point
	Target   Target
	Hostiles []Target // the fight's candidates; the target may or may not be among them
	Range    int      // the skill's cast range (0: no limit)
	// Vel is the observed velocity of a unit in subtiles a second; nil or !ok: unknown.
	Vel  func(id uint32) (vx, vy float64, ok bool)
	LOS  func(a, b input.Point) bool // a clear missile line
	Walk func(p input.Point) bool    // a reachable cell
}

// aimOut is the decision: At is where to cast unless Refused is set; Hits the hostiles within Radius of At;
// Fallback says the rule's point failed its line and At is the target's cell.
type aimOut struct {
	At       input.Point
	Hits     int
	Refused  string
	Fallback bool
}

func unitPoint(u d2.Unit) input.Point { return input.Point{X: u.X, Y: u.Y} }

// alivePool is the target plus every other live hostile of the pool.
func alivePool(tg Target, hs []Target) []input.Point {
	ps := []input.Point{unitPoint(tg.Unit)}
	for _, h := range hs {
		if h.Unit.ID != tg.Unit.ID && h.Unit.Alive() {
			ps = append(ps, unitPoint(h.Unit))
		}
	}
	return ps
}

func within(ps []input.Point, c input.Point, r float64) []input.Point {
	var out []input.Point
	for _, p := range ps {
		if pdist(p, c) <= r {
			out = append(out, p)
		}
	}
	return out
}

func centroid(ps []input.Point) input.Point {
	sx, sy := 0, 0
	for _, p := range ps {
		sx, sy = sx+p.X, sy+p.Y
	}
	n := float64(len(ps))
	return input.Point{X: int(math.Round(float64(sx) / n)), Y: int(math.Round(float64(sy) / n))}
}

// toward is the point d subtiles from a on the line to b (b itself when d reaches past it).
func toward(a, b input.Point, d float64) input.Point {
	l := pdist(a, b)
	if l == 0 || d >= l {
		return b
	}
	return input.Point{X: a.X + int(math.Round(float64(b.X-a.X)*d/l)), Y: a.Y + int(math.Round(float64(b.Y-a.Y)*d/l))}
}

// ahead moves every hostile of the pool by its observed velocity over frames (the pool's order is alivePool's).
func ahead(pool []input.Point, in aimIn, frames int) []input.Point {
	ids := []uint32{in.Target.Unit.ID}
	for _, h := range in.Hostiles {
		if h.Unit.ID != in.Target.Unit.ID && h.Unit.Alive() {
			ids = append(ids, h.Unit.ID)
		}
	}
	s := float64(frames) / aimFPS
	out := make([]input.Point, len(pool))
	for i, p := range pool {
		out[i] = p
		if vx, vy, ok := in.Vel(ids[i]); ok {
			out[i] = input.Point{X: p.X + int(math.Round(vx*s)), Y: p.Y + int(math.Round(vy*s))}
		}
	}
	return out
}

// rulePoint is the point the row's rule picks, before the range clamp and the line check.
func rulePoint(r AimRule, in aimIn, pool []input.Point) input.Point {
	t := unitPoint(in.Target.Unit)
	rad := r.radius()
	switch r.Rule {
	case AimCluster:
		best, bn, bd := t, len(within(pool, t, rad)), 0.0
		for _, h := range pool {
			if in.Range > 0 && pdist(in.Me, h) > float64(in.Range) {
				continue
			}
			c := centroid(within(pool, h, rad))
			n, d := len(within(pool, c, rad)), pdist(c, t)
			if n > bn || n == bn && d < bd {
				best, bn, bd = c, n, d
			}
		}
		return best
	case AimLead:
		if in.Vel == nil {
			return t
		}
		vx, vy, ok := in.Vel(in.Target.Unit.ID)
		if !ok {
			return t
		}
		s := float64(r.LeadFrames) / aimFPS
		return input.Point{X: t.X + int(math.Round(vx*s)), Y: t.Y + int(math.Round(vy*s))}
	case AimPackEdge:
		pack := within(pool, t, aimPackRange)
		c, front := centroid(pack), pack[0]
		for _, p := range pack {
			if pdist(in.Me, p) < pdist(in.Me, front) {
				front = p
			}
		}
		return toward(front, c, rad)
	case AimOffset:
		l := pdist(in.Me, t)
		if l == 0 {
			return t
		}
		k := float64(r.Offset) / l
		return input.Point{X: t.X + int(math.Round(float64(t.X-in.Me.X)*k)), Y: t.Y + int(math.Round(float64(t.Y-in.Me.Y)*k))}
	}
	return t
}

// aimPoint decides where the row's skill lands. Pure.
func aimPoint(r AimRule, in aimIn) aimOut {
	pool := alivePool(in.Target, in.Hostiles)
	t := unitPoint(in.Target.Unit)
	p := rulePoint(r, in, pool)
	if in.Range > 0 && pdist(in.Me, p) > float64(in.Range) {
		p = toward(in.Me, p, float64(in.Range))
	}
	clear := func(q input.Point) bool {
		switch r.los() {
		case AimLOSNone:
			return true
		case AimLOSWalk:
			return (in.Walk == nil || in.Walk(q)) && (in.LOS == nil || in.LOS(in.Me, q))
		}
		return in.LOS == nil || in.LOS(in.Me, q)
	}
	out := aimOut{At: p}
	if !clear(p) {
		if p == t || !clear(t) {
			return aimOut{At: t, Refused: aimRefusedLOS}
		}
		out = aimOut{At: t, Fallback: true}
	}
	if r.Rule == AimLead && in.Vel != nil {
		pool = ahead(pool, in, r.LeadFrames)
	}
	out.Hits = len(within(pool, out.At, r.radius()))
	if out.Hits < r.CrowdAtLeast {
		out.Refused = aimRefusedCrowd
	}
	return out
}

// ---- fighter side ---------------------------------------------------------------------------------------

type aimSeen struct {
	p      input.Point
	at     time.Time
	vx, vy float64
	moved  bool
}

// aimState is the fighter's aim memory and counters (zero value usable).
type aimState struct {
	seen                       map[uint32]aimSeen
	aimed, refused, fallbacks  int
	shadowMoved, shadowRefused int
}

// observe updates the velocity memory from the hostiles of this tick.
func (a *aimState) observe(now time.Time, hs []Target) {
	if a.seen == nil {
		a.seen = map[uint32]aimSeen{}
	}
	for _, h := range hs {
		p := unitPoint(h.Unit)
		o, ok := a.seen[h.Unit.ID]
		n := aimSeen{p: p, at: now}
		if dt := now.Sub(o.at).Seconds(); ok && dt > 0 && now.Sub(o.at) <= aimVelMaxAge {
			n.vx, n.vy, n.moved = float64(p.X-o.p.X)/dt, float64(p.Y-o.p.Y)/dt, true
		} else if ok && o.p == p {
			n.vx, n.vy, n.moved = o.vx, o.vy, o.moved
		}
		a.seen[h.Unit.ID] = n
	}
}

func (a *aimState) vel(id uint32) (float64, float64, bool) {
	s, ok := a.seen[id]
	return s.vx, s.vy, ok && s.moved
}

// AimCounts is the aim driver's counters: casts aimed by a rule, refusals and line fallbacks (mode 2), and in shadow
// (mode 1) the casts the rule would have moved and refused.
type AimCounts struct{ Aimed, Refused, Fallbacks, ShadowMoved, ShadowRefused int }

// AimCounts reports the aim driver's counters since the fighter started.
func (f *Fighter) AimCounts() AimCounts {
	a := &f.aim
	return AimCounts{a.aimed, a.refused, a.fallbacks, a.shadowMoved, a.shadowRefused}
}

// aimRow is the profile's aim row of the skill, nil when it has none.
func (f *Fighter) aimRow(p *d2.Player, name string) *AimRule {
	pr := f.profileFor(p)
	for i := range pr.Aims {
		if pr.Aims[i].Name == name {
			return &pr.Aims[i]
		}
	}
	return nil
}

// aimFor is the point to cast sp at against tg. ok false means the driver (mode 2) refuses the cast; gate false lets
// a crowd refusal through (the watchdog path). Without a row, or with the knob at 0, it is the target's cell.
func (f *Fighter) aimFor(snap *d2.Snapshot, sp SkillSpec, tg Target, hs []Target, gate bool) (input.Point, bool) {
	t := unitPoint(tg.Unit)
	mode := f.upkeepMode()
	if mode == 0 || snap == nil || snap.Player == nil {
		return t, true
	}
	r := f.aimRow(snap.Player, sp.Name)
	if r == nil {
		return t, true
	}
	p := snap.Player
	f.aim.observe(f.now(), hs)
	out := aimPoint(*r, aimIn{
		Me: input.Point{X: p.X, Y: p.Y}, Target: tg, Hostiles: hs, Range: sp.Range,
		Vel: f.aim.vel, LOS: f.los, Walk: f.valid,
	})
	if mode == 1 {
		if out.Refused != "" {
			f.aim.shadowRefused++
		} else if out.At != t {
			f.aim.shadowMoved++
		}
		f.log().Debug("aim shadow", "skill", sp.Name, "rule", r.Rule, "at", out.At, "target", t, "hits", out.Hits, "refused", out.Refused)
		return t, true
	}
	if out.Fallback {
		f.aim.fallbacks++
	}
	switch {
	case out.Refused == aimRefusedLOS, out.Refused == aimRefusedCrowd && gate:
		f.aim.refused++
		f.log().Debug("aim refused", "skill", sp.Name, "rule", r.Rule, "why", out.Refused, "hits", out.Hits)
		return t, out.Refused == aimRefusedLOS && !gate
	}
	f.aim.aimed++
	return out.At, true
}

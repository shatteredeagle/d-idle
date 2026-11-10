package bot

import (
	"strings"

	"d2lod-bot/internal/readiness"
)

// SetReadiness wires the readiness evaluator (internal/readiness). Call it before the loops start; it is read-only
// input for the /api/readiness endpoint and the risk meter's resist gap. No gate or retreat reads it yet (TASK-219).
func (r *Runner) SetReadiness(ev *readiness.Evaluator) { r.readiness = ev }

// ReadinessHero is the hero snapshot for the readiness checks and the name of the area the hero stands in. ok is false
// until the first in-game snapshot. Defence, damage output and gear durability are not read yet (0, 0, -1: the checks
// skip). Resist basis: ResistRaw is today's assumption (combat.PlayerResists.Effective); whether d2.Player.Resist
// already holds the difficulty penalty is U until TASK-216.
func (r *Runner) ReadinessHero() (readiness.Hero, string, bool) {
	s := r.latest()
	w := r.guardWorld(s)
	if w == nil {
		return readiness.Hero{}, "", false
	}
	p := s.Player
	return readiness.Hero{
		Class:      strings.ToLower(p.Class.String()),
		Level:      p.Level,
		Difficulty: strings.ToLower(p.Difficulty.String()),
		Life:       p.Life,
		MaxLife:    p.MaxLife,
		Resist:     readiness.Resists{Fire: p.Resist.Fire, Cold: p.Resist.Cold, Lightning: p.Resist.Lightning, Poison: p.Resist.Poison},
		Basis:      readiness.DefaultResistBasis,
		Potions:    w.Potions,
		RepairPct:  -1,
	}, r.areaName(p.Area), true
}

// readinessResistGap is the risk meter's ResistGap: 0 without an evaluator, an unknown class or a failed read.
func (r *Runner) readinessResistGap() int {
	if r.readiness == nil {
		return 0
	}
	h, _, ok := r.ReadinessHero()
	if !ok {
		return 0
	}
	g, err := r.readiness.ResistGap(h)
	if err != nil {
		return 0
	}
	return g
}

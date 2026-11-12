package bot

import (
	"strings"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
)

// deathWeightSetter is the part of a Fighter that takes death lessons (combat.Fighter.SetDeathWeights).
type deathWeightSetter interface{ SetDeathWeights(combat.DeathWeights) }

// noteDeathWeights is docs/difficulty.md P3 item 27: a Nightmare or Hell death adds weight to the killer's monster
// kind and affixes (DeathInfo.Weights: double for a Hell death 10 or more levels above the character), and the fighter
// avoids a target whose weight reaches 3. Normal deaths add nothing (the spiral breaker's area ban covers them), so a
// Normal run is unchanged. The lessons live for this process only (U: not persisted). Caller holds obMu.
func (r *Runner) noteDeathWeights(di combat.DeathInfo, killer *d2.Unit) {
	if di.Difficulty == d2.Normal {
		return
	}
	key := ""
	if killer != nil && r.e.Data != nil {
		if m := r.e.Data.MonsterByClassID(killer.Class); m != nil && !di.Champion {
			key = strings.ToLower(m.Key) // a champion is not its base kind
		}
	}
	mon, aff := di.Weights(key)
	w := &r.deathW
	if w.Monster == nil {
		w.Monster, w.Affix = map[string]float64{}, map[string]float64{}
	}
	if key != "" {
		w.Monster[key] += mon
	}
	for n, v := range aff {
		w.Affix[n] += v
	}
	if fs, ok := r.d.Fighter.(deathWeightSetter); ok {
		cp := combat.DeathWeights{Monster: map[string]float64{}, Affix: map[string]float64{}}
		for k, v := range w.Monster {
			cp.Monster[k] = v
		}
		for k, v := range w.Affix {
			cp.Affix[k] = v
		}
		fs.SetDeathWeights(cp)
	}
}

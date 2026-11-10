package bot

import (
	"strings"

	"d2lod-bot/internal/ladder"
	"d2lod-bot/internal/watch"
)

// RiskFacts is what the hero risk meter (docs/unlock-risk.md) reads: the kit verdict and potion count the route guard
// uses, the life percentage of the latest snapshot, the engage lines of the resource tuning and the guard's own reason
// for zone ("" asks about the area the hero stands in). It reads the latest snapshot only; nothing is written or asked.
// Ok is false until the first in-game snapshot. ResistGap comes from the readiness model when one is wired (SetReadiness), else 0.
func (r *Runner) RiskFacts(zone string) watch.RiskFacts {
	s := r.latest()
	w := r.guardWorld(s)
	if w == nil {
		return watch.RiskFacts{}
	}
	p := s.Player
	f := watch.RiskFacts{
		Zone:         zone,
		Kit:          ladder.KitFor(w.HasTP, w.Potions),
		Potions:      w.Potions,
		Dead:         isDead(p),
		EngagePct:    r.e.Tune.Resource.EngageLifePct,
		MinEngagePct: r.e.Tune.Resource.MinEngageLifePct,
		ResistGap:    r.readinessResistGap(),
		Ok:           true,
	}
	if p.MaxLife > 0 {
		f.LifePct = p.Life * 100 / p.MaxLife
	}
	if zone == "" {
		f.Zone = r.areaName(p.Area)
	}
	if c := r.d.Catalog; c != nil {
		if id, ok := c.AreaByName(strings.TrimSpace(f.Zone)); ok {
			f.GuardWhy = r.OrderGuardWhy(id)
		}
	}
	return f
}

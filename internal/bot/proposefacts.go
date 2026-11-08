package bot

import (
	"d2lod-bot/internal/ladder"
	"d2lod-bot/internal/watch"
)

// ProposalFacts is what the zone proposer (docs/proposezone.md) reads from the bot: the kit verdict and the healing
// potion count from the latest snapshot (the same count and verdict the route guard uses) and the deaths of the last
// hour. Ok is false until the first in-game snapshot. It reads only; nothing is written or asked.
func (r *Runner) ProposalFacts() watch.ProposalFacts {
	s := r.latest()
	w := r.guardWorld(s)
	if w == nil {
		return watch.ProposalFacts{}
	}
	return watch.ProposalFacts{Kit: ladder.KitFor(w.HasTP, w.Potions), Potions: w.Potions, Deaths: r.DeathReport().LastHour, Ok: true}
}

package loot

import (
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/town/belt"
)

// PotionPlan answers, for a snapshot, how many healing and mana potions the supply plan keeps in the pack (the line's
// max(target, min) minus what the belt already holds), per kind. ok false means no plan applies (town.plan below 4 or
// a plan that cannot be solved); a kind missing from the map has no plan line and keeps today's pack buffer.
type PotionPlan func(s *d2.Snapshot) (keep map[belt.Kind]int, ok bool)

// SetPotionPlan installs the stage 4 pickup rule of the town planner (docs/design/fable-review-2026-11-21/town-planner.md
// 5.3): a ground potion that would land in the pack is wanted while the usable pack count of its kind is under the plan's
// keep, the same number the sell chore protects. The town package installs it on the Looter it shares; nil removes it.
func (l *Looter) SetPotionPlan(f PotionPlan) {
	if f == nil {
		l.potionPlan.Store(nil)
		return
	}
	l.potionPlan.Store(&f)
}

// packTarget is how many pack potions of kind k the pickup wants and the pack count it compares with: the plan's keep
// over the usable pack potions (a potion under the tier floor never counts toward a target, town-planner 3.3), else
// today's buffer over every pack potion of the kind.
func (n needsOf) packTarget(b *data.ItemBase, k belt.Kind) (have, target int) {
	db, inv := n.l.e.Data, n.s.Inventory
	if f := n.l.potionPlan.Load(); f != nil {
		if keep, ok := (*f)(n.s); ok {
			if t, ok := keep[k]; ok {
				p := n.s.Player
				floor, _ := PotionTierFloor(p.Class, p.Level, p.Difficulty)
				for _, u := range inv.Container(d2.ContInventory) {
					if bb := n.l.base(u.Class); bb != nil && belt.PotionKind(bb) == k && belt.Tier(bb) >= floor {
						have++
					}
				}
				return have, t
			}
		}
	}
	return belt.PackCount(db, inv, k), n.l.potionBuffer(b)
}

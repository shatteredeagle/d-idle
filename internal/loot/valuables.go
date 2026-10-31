package loot

import (
	"sort"

	"d2lod-bot/internal/d2"
)

// ValuablesReport splits the unequipped pack items into those that should be stashed before a dangerous trip
// and those that are safe to carry (insurance against death: a corpse run is not worth a rune).
type ValuablesReport struct {
	Stash []d2.Unit
	Carry []d2.Unit // valuables the carry limit still allows, plus nothing else; supplies are never listed
	Value map[uint32]int
}

// Valuables classifies the backpack of s. Supplies (potions, scrolls, keys, gold: Keep verdicts) are always
// safe to carry. A valuable is an item the pickit stashes, or one worth at least ValuableMinGold, or whose rule
// priority is at least ValuableMinPrio (and is not a supply or junk). The cheapest CarryValuables of them may
// stay in the pack; the rest go to the stash. Knobs: loot.valuableMinGold, valuableMinPrio, carryValuables.
func (l *Looter) Valuables(s *d2.Snapshot) ValuablesReport {
	t := &l.e.Tune.Loot
	rep := ValuablesReport{Value: map[uint32]int{}}
	c := l.ctxFrom(s)
	var vals []d2.Unit
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		if u.Item.Flags&d2.FlagStartItem != 0 {
			continue
		}
		v := l.EvaluateWith(u, c)
		val := SellValue(l.base(u.Class), u.Item.Quality)
		if v.Action == Keep || v.Action == Drop && val < t.ValuableMinGold {
			continue
		}
		if v.Action == Stash || val >= t.ValuableMinGold || v.Prio >= t.ValuableMinPrio {
			rep.Value[u.ID] = val
			vals = append(vals, u)
		}
	}
	sort.SliceStable(vals, func(i, j int) bool { return rep.Value[vals[i].ID] < rep.Value[vals[j].ID] })
	n := min(t.CarryValuables, len(vals))
	rep.Carry = vals[:n]
	rep.Stash = vals[n:]
	return rep
}

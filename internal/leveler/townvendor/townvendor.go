// Package townvendor adapts internal/town to leveler.Vendor. It lives apart so that a change in the town API breaks
// this one file only.
package townvendor

import (
	"context"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town"
)

// Vendor wraps a *town.Town.
type Vendor struct{ T *town.Town }

// GearVendors: the repair NPC (weapons and armor smith) first, then the shop NPC (staves, wands, jewelry).
func (Vendor) GearVendors(act int) []int {
	a := town.ActByNo(act)
	if a == nil {
		return nil
	}
	var out []int
	seen := map[int]bool{}
	for _, list := range [][]int{a.Repair, a.Shop} {
		for _, c := range list {
			if !seen[c] {
				seen[c] = true
				out = append(out, c)
			}
		}
	}
	return out
}

func (v Vendor) OpenTrade(ctx context.Context, class int) (d2.Unit, error) {
	npc, err := v.T.OpenForTest(ctx, class)
	if err != nil {
		return npc, err
	}
	return npc, v.T.PickEntryForTest(ctx, npc, town.EntryTrade, d2.UIShop)
}

func (v Vendor) Stock(npc d2.Unit) ([]d2.Unit, error) { return v.T.Stock(npc) }

func (v Vendor) PackFits(s *d2.Snapshot, w, h int) (bool, error) { return v.T.PackFits(s, w, h) }

func (v Vendor) MakeRoom(ctx context.Context, w, h int, junk []d2.Unit) (bool, error) {
	return v.T.MakeRoom(ctx, w, h, junk)
}

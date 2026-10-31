package town

import (
	"context"

	"d2lod-bot/internal/d2"
)

// BASICSFIX (docs/basics-audit.md). Policy numbers below are the d2-sme's EXP rule of thumb (docs/sme/play-economy.md),
// not measured on d2d; each sits behind a town.* knob.

// idScrollWanted is the idScrollNeed knob: an identify scroll is bought only while an unidentified set, rare, unique or
// crafted item sits in the pack (the SME: identify only likely valuables; a plain magic item under ~150 gold is sold as
// it is). 0 = the old rule, always up to IDScrollTarget.
func (t *Town) idScrollWanted(s *d2.Snapshot) bool {
	if t.tune().IDScrollNeed == 0 {
		return true
	}
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		if likelyValuable(u) {
			return true
		}
	}
	return false
}

// likelyValuable: an unidentified set, rare, unique or crafted item.
func likelyValuable(u d2.Unit) bool {
	if u.Item.Identified() {
		return false
	}
	switch u.Item.Quality {
	case d2.QualitySet, d2.QualityRare, d2.QualityUnique, d2.QualityCrafted:
		return true
	}
	return false
}

// sellUnidentified: with idScrollNeed on and no identify tool in the pack, an unidentified plain item is sold as it is
// (the SME: not worth a scroll) instead of waiting for a full pack.
func (t *Town) sellUnidentified(s *d2.Snapshot, u d2.Unit) bool {
	if t.tune().IDScrollNeed == 0 || likelyValuable(u) {
		return false
	}
	_, have := t.idTool(s)
	return !have
}

// identifyNeeded is the identify step's need check: items to identify, and a scroll in the pack or a reason to buy one.
func (t *Town) identifyNeeded(s *d2.Snapshot) bool {
	if len(t.needIdentify(s)) == 0 {
		return false
	}
	if _, have := t.idTool(s); have {
		return true
	}
	return t.idScrollWanted(s)
}

// repairLatch remembers a repair that did not change durability.
type repairLatch struct {
	set       bool
	gold, low int
}

// repairLatched is the repairLatch knob: after a failed repair the chore stays quiet until the lowest durability changed
// or gold grew by FutileGoldStep (the SME: "it did not change, do not retry"; d2d logged 5 failed repairs in one run).
func (t *Town) repairLatched(s *d2.Snapshot, low int) bool {
	l := t.repairFail
	if !l.set || t.tune().RepairLatch == 0 {
		return false
	}
	if low != l.low || s.Player.Gold >= l.gold+max(t.tune().FutileGoldStep, 1) {
		t.repairFail = repairLatch{}
		return false
	}
	return true
}

// kitGold is the gold of the minimal kit: one portal scroll and the first healing potions (resource knobs).
func (t *Town) kitGold() int {
	r := &t.e.Tune.Resource
	return r.KitTPGold + r.HPFirst*r.KitPotionGold
}

// bankKeep is the gold the pack keeps while banking before the shop (goldKeep, never below the minimal kit).
func (t *Town) bankKeep(level int) int { return t.goldKeep(level) }

// needBank is the bankFirst knob: the surplus above the keep line goes to the stash before the shop, so the shop
// (tome, spare scrolls, extra potions) cannot spend it (audit row 11: 40 of 1500 gold banked at clvl 5).
func (t *Town) needBank(s *d2.Snapshot, plan ChorePlan) bool {
	return plan.Gold && t.tune().BankFirst != 0 && s.Player.Gold > t.bankKeep(s.Player.Level)
}

func (t *Town) bank(ctx context.Context, a *Act) error {
	s, err := t.current()
	if err != nil {
		return err
	}
	if err := t.openStash(ctx, a); err != nil {
		return err
	}
	var first error
	if keep := t.bankKeep(s.Player.Level); s.Player.Gold > keep {
		first = t.depositCapped(ctx, s, s.Player.Gold-keep)
	}
	if err := t.h.CloseAll(ctx); err != nil && first == nil {
		first = err
	}
	return first
}

// tpAllowed is the potionsFirst knob (d2-sme order, user decision D3): the first town portal scroll waits for the kit
// potions and is bought only when the gold left covers the scroll plus TPScrollPotions potions. TPFirstLevel stays
// an extra clvl floor (default 0 = none). A held scroll or tome is never affected. Prices are the kit tuning's.
func (t *Town) tpAllowed(s *d2.Snapshot) bool {
	r := t.e.Tune.Resource
	return t.tpAllowedAt(s, r.KitTPGold, r.KitPotionGold)
}

// tpAllowedAt is tpAllowed with the scroll and cheapest healing potion prices of the vendor in front of us.
func (t *Town) tpAllowedAt(s *d2.Snapshot, scroll, potion int) bool {
	tu := t.tune()
	return tu.PotionsFirst == 0 || s.Player.Level >= tu.TPFirstLevel && s.Player.Gold >= scroll+tu.TPScrollPotions*potion
}

// kitPotions is the healing potion count of the kit pass (resource HPFirst, at most a recovery floor).
func (t *Town) kitPotions(plan ChorePlan) int {
	n := t.e.Tune.Resource.HPFirst
	if plan.PotionFloor > 0 {
		n = min(n, plan.PotionFloor)
	}
	return max(n, 1)
}

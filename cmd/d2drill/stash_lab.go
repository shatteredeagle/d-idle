package main

import (
	"d2lod-bot/internal/combat"
)

// The lab side of the stash and tome rechecks (docs/livecheckdrills.md). Every number comes from the run's tuning or
// the item tables; there is no TripLab here (the Runner loop is not wired into the drill lab), so DS.3 reports BLOCKED.

// KeepGold mirrors Town.goldKeep: StashGoldBase + level*StashGoldPerLvl, never below the kit when BankFirst is on.
func (l *lab) KeepGold() int {
	s, err := l.be.Env.Snapshot()
	if err != nil || s.Player == nil {
		return 0
	}
	tu := l.tune.Town
	keep := tu.StashGoldBase + s.Player.Level*tu.StashGoldPerLvl
	if tu.BankFirst != 0 {
		keep = max(keep, l.KitCost())
	}
	return keep
}

// KitCost mirrors Town.kitCost: one portal scroll and max(HPFirst,1) healing potions.
func (l *lab) KitCost() int {
	r := l.tune.Resource
	return r.KitTPGold + max(r.HPFirst, 1)*r.KitPotionGold
}

// RejuvCode is the lowest tier rejuvenation potion in the item table.
func (l *lab) RejuvCode() string {
	cat := combat.NewCatalog(l.db)
	best, tier := "", 0
	for i, it := range l.db.Items {
		c := cat.Item(i)
		if c.Potion == combat.PotionRejuv && (best == "" || c.Tier < tier) {
			best, tier = it.Code, c.Tier
		}
	}
	return best
}

// tomeJunkCode is a plain armor base the drill drops beside the tome (U: that the pickit leaves it on the ground; the
// give cheat is V live in docs/forkcheats.md).
const tomeJunkCode = "cap"

func (l *lab) JunkCode() string { return tomeJunkCode }

func (l *lab) LogMark() int            { return l.rec.Mark() }
func (l *lab) LogSince(m int) []string { return l.rec.Since(m) }

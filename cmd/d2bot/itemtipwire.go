package main

import (
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/status"
)

// itemTips adds the tooltip facts of docs/inventory-tooltip.md to the inventory snapshot: base-type facts and the
// sell price from the game tables (both backends), and, on the d2d backend, the game's own tooltip lines and the socket
// count from the last snapshot's extras. Every part is optional; without it the item keeps today's fields.
type itemTips struct {
	db    *data.DB
	extra func(id uint32) (tip []string, sockets int, ok bool) // nil: the backend has no tooltip text
}

func (t *itemTips) set(db *data.DB, extra func(id uint32) ([]string, int, bool)) {
	t.db, t.extra = db, extra
}

func (t *itemTips) fill(it *status.Item, u d2.Unit) {
	if t.extra != nil {
		if tip, n, ok := t.extra(u.ID); ok {
			it.Tip = tip
			if n > 0 {
				it.Sockets = n
			}
		}
	}
	if t.db == nil || u.Class < 0 || u.Class >= len(t.db.Items) || t.db.Items[u.Class] == nil {
		return
	}
	b := t.db.Items[u.Class]
	it.Base = &status.ItemBase{Name: b.Display, MinAC: b.MinAC, MaxAC: b.MaxAC, Block: b.Block, MinDam: b.MinDam, MaxDam: b.MaxDam,
		TwoMinDam: b.TwoMinDam, TwoMaxDam: b.TwoMaxDam, TwoHandedDam: b.TwoHanded, ReqStr: b.ReqStr, ReqDex: b.ReqDex,
		ReqLvl: b.LevelReq, Speed: b.Speed, MaxSockets: b.Sockets}
	if !b.NoDurability {
		it.Base.Dur = b.Durability
	}
	if u.Item != nil && !b.Quest {
		it.Sell = loot.SellValue(t.db, b, u.Item, 0)
	}
}

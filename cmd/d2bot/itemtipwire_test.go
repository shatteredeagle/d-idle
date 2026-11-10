package main

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/status"
)

func TestItemTipsFill(t *testing.T) {
	db := &data.DB{Items: []*data.ItemBase{{Display: "Cap", MinAC: 3, MaxAC: 5, ReqStr: 0, LevelReq: 0, Durability: 12, Sockets: 2, Cost: 20}}}
	u := d2.Unit{ID: 7, Class: 0, Item: &d2.Item{Quality: d2.QualityNormal, Flags: d2.FlagIdentified}}

	var tips itemTips
	tips.set(db, func(id uint32) ([]string, int, bool) { return []string{"Cap", "Defense: 4"}, 2, id == 7 })
	it := status.Item{ID: 7}
	tips.fill(&it, u)
	if len(it.Tip) != 2 || it.Sockets != 2 || it.Base == nil || it.Base.MaxAC != 5 || it.Base.Dur != 12 || it.Sell < 1 {
		t.Fatalf("filled item %+v base %+v", it, it.Base)
	}

	// memory backend: no extras, the base facts still come
	var mem itemTips
	mem.set(db, nil)
	it = status.Item{ID: 7}
	mem.fill(&it, u)
	if it.Tip != nil || it.Base == nil {
		t.Fatalf("memory backend item %+v", it)
	}

	// no tables, unknown class: the item is left alone
	var none itemTips
	it = status.Item{ID: 7}
	none.fill(&it, d2.Unit{ID: 7, Class: 99, Item: u.Item})
	if it.Base != nil || it.Sell != 0 {
		t.Fatalf("item without tables %+v", it)
	}
}

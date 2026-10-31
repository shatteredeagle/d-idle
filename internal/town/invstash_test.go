package town_test

import (
	"context"
	"d2lod-bot/internal/gear/geartest"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town"
)

// fillPack fills the whole 10x4 pack with 1x1 items of one code and quality, unidentified when unid.
func fillPack(r *rig, code string, quality int, unid bool) {
	for y := 0; y < 4; y++ {
		for x := 0; x < 10; x++ {
			u := r.W.NewItem(code, quality)
			if unid {
				u.Item.Flags &^= d2.FlagIdentified
			}
			r.W.Put(u, d2.ContInventory, x, y)
		}
	}
}

// A full pack of unidentified rings that nobody could identify (no scroll, no gold) is sold, not carried for ever.
func TestFullPackOfUnidentifiedJewelryIsSold(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	dressed(r.W)
	r.W.Player.Gold = 0
	fillPack(r, "rin", 4, true)
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Identify: true, Sell: true})
	if err != nil {
		t.Fatal(err)
	}
	if n := countCode(r, "rin"); n > 0 {
		t.Errorf("%d unidentified rings still in the pack, steps %+v", n, rep.Steps)
	}
	if rep.PackPct >= 70 {
		t.Errorf("pack %d%% after the chores", rep.PackPct)
	}
}

// A pack with room keeps unidentified jewelry: the next visit with gold identifies it.
func TestUnidentifiedJewelryWaitsWhileThePackHasRoom(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	dressed(r.W)
	r.W.Player.Gold = 0
	for x := 0; x < 3; x++ {
		u := r.W.NewItem("rin", 4)
		u.Item.Flags &^= d2.FlagIdentified
		r.W.Put(u, d2.ContInventory, x, 0)
	}
	if _, err := r.T.Chores(context.Background(), town.ChorePlan{Identify: true, Sell: true}); err != nil {
		t.Fatal(err)
	}
	if n := countCode(r, "rin"); n != 3 {
		t.Errorf("%d rings left, want 3", n)
	}
}

// Keys are not bought (KeyTarget 0) and a full pack sells the ones that were picked up; the Cube and quest items stay.
func TestFullPackSellsKeysKeepsCube(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	dressed(r.W)
	r.W.Add("box", d2.ContInventory, 0, 0)
	r.W.Add("msf", d2.ContInventory, 2, 0)
	occ := map[[2]int]bool{}
	for _, u := range r.W.Items {
		if u.Item.Where == d2.ContInventory {
			b := r.W.DB.Items[u.Class]
			for dy := 0; dy < max(1, b.Height); dy++ {
				for dx := 0; dx < max(1, b.Width); dx++ {
					occ[[2]int{u.X + dx, u.Y + dy}] = true
				}
			}
		}
	}
	for y := 0; y < 4; y++ {
		for x := 0; x < 10; x++ {
			if !occ[[2]int{x, y}] {
				r.W.Add("key", d2.ContInventory, x, y)
			}
		}
	}
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Sell: true})
	if err != nil {
		t.Fatal(err)
	}
	if n := countCode(r, "key"); n != 0 {
		t.Errorf("%d keys left (pack %d%%), steps %+v", n, rep.PackPct, rep.Steps)
	}
	if countCode(r, "box") != 1 || countCode(r, "msf") != 1 {
		t.Error("Cube or quest item sold")
	}
}

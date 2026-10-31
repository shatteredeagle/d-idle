package leveler_test

import (
	"context"
	"d2lod-bot/internal/gear/geartest"
	"errors"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/leveler/townvendor"
	"d2lod-bot/internal/town"
	"d2lod-bot/internal/town/sim"
)

type shopRig struct {
	*sim.Rig
	lv *leveler.Leveler
	tv townvendor.Vendor
}

// newShopRig: Charsi sells leather armor (2x3); the character already wears quilted armor, so a bought armor must land
// in the pack. The pack is the live one that failed (i4, 2026-10-28): 37 of 40 cells taken, no 2x3 hole, and two
// magic quilted armors (junk) in the top left. Real item tables are needed (/tmp/d2mpq).
func newShopRig(t *testing.T, junk bool) *shopRig {
	r := &shopRig{Rig: sim.NewRig(t)}
	w := r.W
	r.Env.Tune.Town.SpareKeep = 100 // the scroll filler counts as supplies the pack keeps
	w.Player.Gold = 3000
	w.Player.Strength, w.Player.Dexterity = 40, 30
	charsi := w.AddNPC(154, 5984, 4922)
	w.AddObject(267, 6016, 4934) // stash
	w.Stock(charsi, "lea")
	worn := w.Add("qui", d2.ContEquipment, 0, 0)
	worn.Item.Body = 3
	w.Items[len(w.Items)-1] = worn
	if junk {
		for _, x := range []int{0, 2} {
			qui := w.Add("qui", d2.ContInventory, x, 0)
			qui.Item.Quality = 4
			w.Items[len(w.Items)-1] = qui
		}
	}
	for x := 0; x < 10; x++ {
		for y := 0; y < 4; y++ {
			switch {
			case x < 4 && y < 3: // the armors (or a hole when there is no junk)
				if !junk {
					w.Add("tsc", d2.ContInventory, x, y)
				}
			case y == 3 && x >= 7:
			default:
				w.Add("tsc", d2.ContInventory, x, y)
			}
		}
	}
	tw := town.New(r.Env, r.Mv)
	tw.UseSource(town.Source{Snapshot: w.Snapshot, Items: w.AllItems, Owner: w.Owner, Stats: w.UnitStats})
	r.tv = townvendor.Vendor{T: tw}
	r.lv = leveler.New(r.Env, leveler.Options{Snapshot: w.Snapshot})
	return r
}

func (r *shopRig) buyLeather(t *testing.T) (int, error) {
	ctx := context.Background()
	npc, err := r.tv.OpenTrade(ctx, 154)
	if err != nil {
		t.Fatal(err)
	}
	stock, err := r.tv.Stock(npc)
	if err != nil || len(stock) == 0 {
		t.Fatalf("stock %v %v", stock, err)
	}
	s, _ := r.W.Snapshot()
	return r.lv.BuyUnit(ctx, r.tv, 154, npc, r.lv.Candidate(s.Player, stock[0]))
}

func packHas(w *sim.World, code string) bool {
	for _, u := range w.Items {
		if u.Item.Where == d2.ContInventory && u.Class == w.NewItem(code, 2).Class {
			return true
		}
	}
	return false
}

func TestBuyMakesRoomInAFullFragmentedPack(t *testing.T) {
	geartest.Need(t)
	r := newShopRig(t, true)
	paid, err := r.buyLeather(t)
	if err != nil || paid <= 0 {
		t.Fatalf("paid %d err %v", paid, err)
	}
	if !packHas(r.W, "lea") {
		t.Fatal("leather armor not in the pack")
	}
}

func TestBuyFailsCleanlyWhenNothingCanBeSold(t *testing.T) {
	geartest.Need(t)
	r := newShopRig(t, false)
	g0 := r.W.Player.Gold
	_, err := r.buyLeather(t)
	if !errors.Is(err, leveler.ErrNoRoom) {
		t.Fatalf("want ErrNoRoom, got %v", err)
	}
	if r.W.Player.Gold != g0 {
		t.Fatalf("gold changed %d -> %d", g0, r.W.Player.Gold)
	}
}

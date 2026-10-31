package town_test

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town"
)

// fragmentedPack builds a pack like the one the live leveler failed on (i4, 2026-10-28, 82% full, no 2x3 hole):
// 37 of 40 cells taken, the three free cells are scattered. Two magic quilted armors (2x3, junk) sit in the top left.
func fragmentedPack(r *rig) {
	w := r.W
	dressed(w)
	r.Env.Tune.Town.SpareKeep = 100 // the scroll filler counts as supplies the pack keeps
	for _, x := range []int{0, 2} {
		qui := w.Add("qui", d2.ContInventory, x, 0)
		qui.Item.Quality = 4
		w.Items[len(w.Items)-1] = qui
	}
	for x := 0; x < 10; x++ {
		for y := 0; y < 4; y++ {
			switch {
			case x < 4 && y < 3: // the armors
			case y == 3 && x >= 7: // free cells
			default:
				w.Add("tsc", d2.ContInventory, x, y)
			}
		}
	}
}

func openCharsi(t *testing.T, r *rig) {
	t.Helper()
	npc, err := r.T.OpenForTest(context.Background(), 154)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.T.PickEntryForTest(context.Background(), npc, town.EntryTrade, d2.UIShop); err != nil {
		t.Fatal(err)
	}
}

func TestPackFitsNeedsARealHole(t *testing.T) {
	r := newRig(t)
	fragmentedPack(r)
	s, _ := r.W.Snapshot()
	if ok, err := r.T.PackFits(s, 2, 3); err != nil || ok {
		t.Fatalf("2x3 fits a fragmented pack: %v %v", ok, err)
	}
	if ok, _ := r.T.PackFits(s, 1, 1); !ok {
		t.Fatal("1x1 should fit")
	}
}

func TestMakeRoomSellsJunkUntilTheItemFits(t *testing.T) {
	r := newRig(t)
	fragmentedPack(r)
	openCharsi(t, r)
	ok, err := r.T.MakeRoom(context.Background(), 2, 3, nil)
	if err != nil || !ok {
		t.Fatalf("make room: %v %v", ok, err)
	}
	if n := count(r.W, d2.ContInventory); n != 25+0 {
		t.Errorf("pack has %d items, want both armors sold", n)
	}
}

func TestMakeRoomNothingToSell(t *testing.T) {
	r := newRig(t)
	r.Env.Tune.Town.SpareKeep = 100
	for x := 0; x < 10; x++ { // every cell holds a scroll the pickit keeps: no room, and no stash candidate either
		for y := 0; y < 4; y++ {
			r.W.Add("tsc", d2.ContInventory, x, y)
		}
	}
	openCharsi(t, r)
	ok, _ := r.T.MakeRoom(context.Background(), 2, 3, nil)
	if ok {
		t.Fatal("made room out of nothing")
	}
}

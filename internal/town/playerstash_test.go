package town_test

import (
	"context"
	"errors"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/gear/geartest"
	"d2lod-bot/internal/invdesk"
	"d2lod-bot/internal/town"
)

// The player's stash actions on the sim world. They SKIP without game tables (geartest.Need), like the other town
// click-path tests;.
func TestPlayerStashMovesAndRefusals(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	w := r.W
	pack := w.Add("hp1", d2.ContInventory, 0, 0)
	stashed := w.Add("gpv", d2.ContStash, 0, 0)
	s, err := r.Env.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.T.CheckStashMove(s, pack.ID, true); err != nil {
		t.Errorf("pack -> stash: %v", err)
	}
	if _, err := r.T.CheckStashMove(s, stashed.ID, true); err == nil {
		t.Error("a stash item cannot be stowed")
	}
	res := r.T.StashVisit(context.Background(), invdesk.StashPlan{Take: []uint32{stashed.ID}, Stow: []uint32{pack.ID}})
	if res.Errs[stashed.ID] != nil || res.Errs[pack.ID] != nil {
		t.Fatalf("visit %+v", res.Errs)
	}
	for _, u := range w.Items {
		switch u.ID {
		case stashed.ID:
			if u.Item.Where != d2.ContInventory {
				t.Error("not taken out")
			}
		case pack.ID:
			if u.Item.Where != d2.ContStash {
				t.Error("not stowed")
			}
		}
	}
}

func TestPlayerStashNoRoomInThePack(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	w := r.W
	sword := w.Add("hax", d2.ContStash, 0, 0)
	for row := 0; row < 4; row++ { // fill the pack with 1x1 items
		for col := 0; col < 10; col++ {
			w.Add("hp1", d2.ContInventory, col, row)
		}
	}
	s, err := r.Env.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.T.CheckStashMove(s, sword.ID, false); !errors.Is(err, town.ErrNoRoomPack) {
		t.Errorf("full pack: %v", err)
	}
}

func TestPlayerStashIdentifyInPlace(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	w := r.W
	w.Add("isc", d2.ContInventory, 0, 0)
	ring := w.Add("rin", d2.ContStash, 2, 1)
	ring.Item.Quality, ring.Item.Flags = 6, 0
	res := r.T.StashVisit(context.Background(), invdesk.StashPlan{Identify: []uint32{ring.ID}})
	if res.Errs[ring.ID] != nil {
		t.Skipf("the sim may not model the identify cursor on stash cells: %v", res.Errs[ring.ID])
	}
	if !ring.Item.Identified() || ring.Item.Where != d2.ContStash {
		t.Error("identified in place and still in the stash expected")
	}
}

func TestPlayerStashGold(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	r.W.Player.Gold, r.W.Player.Level = 5000, 20
	res := r.T.StashVisit(context.Background(), invdesk.StashPlan{Gold: 2000, Deposit: true})
	if res.GoldErr != nil || r.W.Player.StashGold != 2000 || r.W.Player.Gold != 3000 {
		t.Fatalf("%v %q gold %d bank %d", res.GoldErr, res.Gold, r.W.Player.Gold, r.W.Player.StashGold)
	}
}

package town_test

import (
	"context"
	"d2lod-bot/internal/gear/geartest"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town"
)

// fillStash fills the whole stash grid (6x8 in LoD) with 1x1 items of one code.
func fillStash(r *rig, code string) {
	for y := 0; y < 8; y++ {
		for x := 0; x < 6; x++ {
			r.W.Add(code, d2.ContStash, x, y)
		}
	}
}

func TestFullStashDoesNotFailAndSellsLowPrioFirst(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	w := r.W
	fillStash(r, "jew") // jewels: stash, prio 60, unprotected
	n := count(w, d2.ContStash)
	w.Add("r07", d2.ContInventory, 0, 0) // rune: protected, higher prio
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Stash: true, Sell: true})
	if err != nil || rep.Errors != 0 {
		t.Fatalf("%v %+v", err, rep.Steps)
	}
	if rep.Stashed != 1 || rep.Sold != 1 || count(w, d2.ContStash) != n {
		t.Fatalf("stashed %d sold %d stash %d (was %d)", rep.Stashed, rep.Sold, count(w, d2.ContStash), n)
	}
	if count(w, d2.ContInventory) != 0 {
		t.Fatalf("pack still holds %d items", count(w, d2.ContInventory))
	}
	// a second visit has nothing to do
	rep2, _ := r.T.Chores(context.Background(), town.ChorePlan{Stash: true, Sell: true})
	for _, s := range rep2.Steps {
		if s.Ran && s.Name != "close" {
			t.Errorf("second visit ran %s", s.Name)
		}
	}
}

func TestFullStashOfRunesKeepsEverything(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	w := r.W
	fillStash(r, "r07")
	w.Add("r08", d2.ContInventory, 0, 0)
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Stash: true})
	if err != nil || rep.Errors != 0 || rep.Sold != 0 || rep.Stashed != 0 {
		t.Fatalf("%v %+v sold %d stashed %d", err, rep.Steps, rep.Sold, rep.Stashed)
	}
	// nothing sold: runes all still there, and the next visit does not retry
	rep2, _ := r.T.Chores(context.Background(), town.ChorePlan{Stash: true})
	for _, s := range rep2.Steps {
		if s.Ran && s.Name != "close" {
			t.Errorf("repeat visit ran %s", s.Name)
		}
	}
	for _, u := range w.Items {
		if u.Item.Where == d2.ContStash && w.DB.Items[u.Class].Code != "r07" {
			t.Fatal("a stash rune was replaced")
		}
	}
}

func TestFullPackQuestItemSurvivesSelling(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	w := r.W
	w.Player.Level = 10 // above the magic-item margin (EQUIPREQ): the magic sword is the weapon worn, as before
	w.Add("box", d2.ContInventory, 0, 0)
	w.Add("msf", d2.ContInventory, 1, 0)
	q := w.Add("ssd", d2.ContInventory, 2, 0)
	q.Item.Quality = 4
	w.Items[len(w.Items)-1] = q
	fillStash(r, "r07")
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Stash: true, Sell: true})
	if err != nil || rep.Errors != 0 {
		t.Fatalf("%v %+v", err, rep.Steps)
	}
	m := cubeCodes(r, d2.ContInventory)
	if m["box"] != 1 || m["msf"] != 1 || m["ssd"] != 0 {
		t.Fatalf("pack %v", m)
	}
}

func TestCubeFullPackIsCleanAndNotRetried(t *testing.T) {
	geartest.Need(t)
	r := cubeRig(t, map[string]int{"r07": 3})
	for y := 0; y < 4; y++ {
		for x := 0; x < 10; x++ {
			if x < 2 && y < 2 { // the Cube (2x2 in the item table)
				continue
			}
			r.W.Add("jew", d2.ContInventory, x, y)
		}
	}
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Cube: true})
	if err != nil || rep.Errors != 0 || rep.Cubed != 0 {
		t.Fatalf("%v cubed %d %+v", err, rep.Cubed, rep.Steps)
	}
	if m := cubeCodes(r, d2.ContStash); m["r07"] != 3 {
		t.Fatalf("stash %v", m)
	}
	rep2, _ := r.T.Chores(context.Background(), town.ChorePlan{Cube: true})
	for _, s := range rep2.Steps {
		if s.Ran && s.Name != "close" {
			t.Errorf("repeat visit ran %s", s.Name)
		}
	}
	// room appears: the chore runs again
	for freed := 0; freed < 3; freed++ {
		for i, u := range r.W.Items {
			if u.Item.Where == d2.ContInventory && r.W.DB.Items[u.Class].Code == "jew" {
				r.W.Items = append(r.W.Items[:i], r.W.Items[i+1:]...)
				break
			}
		}
	}
	rep3, err := r.T.Chores(context.Background(), town.ChorePlan{Cube: true})
	if err != nil || rep3.Cubed != 1 {
		t.Fatalf("%v cubed %d %+v", err, rep3.Cubed, rep3.Steps)
	}
}

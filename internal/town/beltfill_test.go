package town_test

import (
	"context"
	"d2lod-bot/internal/gear/geartest"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town"
)

func addPack(r *rig, code string, n int) {
	placed := 0
	for y := 0; y < 4 && placed < n; y++ {
		for x := 0; x < 10 && placed < n; x++ {
			free := true
			for _, u := range r.W.Items {
				if u.Item.Where == d2.ContInventory && u.X == x && u.Y == y {
					free = false
				}
			}
			if free {
				r.W.Add(code, d2.ContInventory, x, y)
				placed++
			}
		}
	}
}

func bought(rep town.ChoreReport) int {
	n := 0
	for _, v := range rep.Bought {
		n += v
	}
	return n
}

// The Soaky case: a pack of healing potions and an empty belt. The belt is filled from the pack, nothing is bought.
func TestBeltFilledFromPackNoPurchase(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	dressed(r.W)
	addPack(r, "hp1", 12)
	gold := r.W.Player.Gold
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true})
	if err != nil {
		t.Fatal(err)
	}
	if n := count(r.W, d2.ContBelt); n == 0 {
		t.Fatalf("belt still empty: %+v", rep.Steps)
	}
	for name, n := range rep.Bought {
		if strings.Contains(name, "Healing") {
			t.Errorf("bought %d %s (gold %d -> %d); the pack held enough", n, name, gold, r.W.Player.Gold)
		}
	}
}

func TestFillBeltMovesAndReports(t *testing.T) {
	r := newRig(t)
	addPack(r, "hp1", 3)
	rep, err := r.T.FillBelt(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Moved == 0 || rep.Moved != count(r.W, d2.ContBelt) || rep.Failed != 0 || rep.Dragged != 0 {
		t.Errorf("report %+v belt %d", rep, count(r.W, d2.ContBelt))
	}
	if rep.After.Total() <= rep.Before.Total() {
		t.Errorf("belt state did not grow: %+v -> %+v", rep.Before, rep.After)
	}
}

// A belt with no room for the plan moves nothing and clicks nothing.
func TestFullBeltMovesNothing(t *testing.T) {
	r := newRig(t)
	for s := 0; s < 4; s++ {
		r.W.Add("hp1", d2.ContBelt, s, 0)
	}
	addPack(r, "hp1", 5)
	n0 := len(r.W.Clicks)
	rep, err := r.T.FillBelt(context.Background())
	if err != nil || rep.Moved != 0 || len(r.W.Clicks) != n0 {
		t.Errorf("rep %+v err %v clicks %d", rep, err, len(r.W.Clicks)-n0)
	}
	if r.T.NeedsBeltFill(mustSnap(t, r)) {
		t.Error("NeedsBeltFill true with a full belt")
	}
}

func mustSnap(t *testing.T, r *rig) *d2.Snapshot {
	s, err := r.W.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// A swallowed shift-click is retried; when every try is swallowed the drag-and-drop fallback moves the potion.
func TestShiftClickRetryThenDragFallback(t *testing.T) {
	r := newRig(t)
	addPack(r, "hp1", 1)
	r.W.ShiftFails = 1
	rep, err := r.T.FillBelt(context.Background())
	if err != nil || rep.Moved != 1 || rep.Dragged != 0 {
		t.Fatalf("one swallowed click: %+v err %v", rep, err)
	}

	r = newRig(t)
	addPack(r, "hp1", 1)
	r.W.ShiftFails = 99
	rep, err = r.T.FillBelt(context.Background())
	if err != nil || rep.Moved != 1 || rep.Dragged != 1 || count(r.W, d2.ContBelt) != 1 {
		t.Fatalf("fallback: %+v err %v", rep, err)
	}
}

// Shift-click and drag both fail: the failure is reported, the cursor is empty and the potion is back in the pack.
func TestBeltFillGivesUp(t *testing.T) {
	r := newRig(t)
	addPack(r, "hp1", 3)
	r.W.ShiftFails, r.W.DragFails = 99, 99
	rep, err := r.T.FillBelt(context.Background())
	if err == nil || rep.Moved != 0 || rep.Failed == 0 {
		t.Fatalf("rep %+v err %v", rep, err)
	}
	if r.W.Cursor != nil || count(r.W, d2.ContInventory) != 3 {
		t.Errorf("cursor %v pack %d", r.W.Cursor, count(r.W, d2.ContInventory))
	}
}

// Belt fill failed, so the pack still holds the potions: the shop must not buy them again.
func TestShopCountsPackPotions(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	dressed(r.W)
	addPack(r, "hp1", 12)
	addPack(r, "mp1", 12)
	r.W.ShiftFails, r.W.DragFails = 99, 99
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true})
	if err != nil {
		t.Fatal(err)
	}
	if n := bought(rep); n != 0 {
		t.Errorf("bought %v with a full pack", rep.Bought)
	}
}

// REJUVUSE: a pack rejuvenation goes to a free belt slot even without a rejuvenation column (clvl < 31), so the belt
// hotkey can drink it.
func TestPackRejuvIsBelted(t *testing.T) {
	r := newRig(t)
	r.W.Player.Level = 14
	addPack(r, "rvl", 1)
	if _, err := r.T.FillBelt(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := countCode(r, "rvl"); n != 0 || count(r.W, d2.ContBelt) != 1 {
		t.Errorf("rvl %d, belt %d: not moved to the belt", n, count(r.W, d2.ContBelt))
	}
}

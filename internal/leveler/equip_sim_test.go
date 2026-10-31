package leveler_test

import (
	"context"
	"d2lod-bot/internal/gear/geartest"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/town/sim"
)

// equipRig: a clvl 6 Sorceress with the given pack items; the sim places items into every slot class (neck, rings,
// belt, boots, gloves) and refuses the ids in w.EquipRefuse. Real item tables are needed (/tmp/d2mpq).
func equipRig(t *testing.T, codes ...string) (*sim.Rig, *leveler.Leveler, []d2.Unit) {
	r := sim.NewRig(t)
	w := r.W
	w.Player.Level = 6
	w.Player.Strength, w.Player.Dexterity = 40, 30
	var us []d2.Unit
	for i, c := range codes {
		u := w.Add(c, d2.ContInventory, i, 0)
		u.Item.Quality = 4
		u.Item.Flags |= d2.FlagIdentified
		w.Items[len(w.Items)-1] = u
		us = append(us, u)
	}
	return r, leveler.New(r.Env, leveler.Options{Snapshot: w.Snapshot}), us
}

func worn(w *sim.World, id uint32) (int, bool) {
	for _, u := range w.Items {
		if u.ID == id && u.Item.Where == d2.ContEquipment {
			return u.Item.Body, true
		}
	}
	return 0, false
}

func inPack(w *sim.World, id uint32) bool {
	for _, u := range w.Items {
		if u.ID == id && u.Item.Where == d2.ContInventory {
			return true
		}
	}
	return false
}

// The neck slot (and the others) accept a pack item through the same click as every slot.
func TestEquipFillsSmallSlots(t *testing.T) {
	geartest.Need(t)
	for _, c := range []struct {
		code string
		body int
	}{{"amu", d2.BodyNeck}, {"rin", d2.BodyRRing}, {"lgl", d2.BodyGloves}, {"lbt", d2.BodyFeet}, {"lbl", d2.BodyBelt}} {
		r, lv, us := equipRig(t, c.code)
		rep := lv.Equip(context.Background())
		body, ok := worn(r.W, us[0].ID)
		if !ok || rep.Err != nil || len(rep.Failed) != 0 {
			t.Errorf("%s: not worn (rep %+v)", c.code, rep)
			continue
		}
		if c.body == d2.BodyRRing {
			if body != d2.BodyRRing && body != d2.BodyLRing {
				t.Errorf("ring at body %d", body)
			}
		} else if body != c.body {
			t.Errorf("%s at body %d want %d", c.code, body, c.body)
		}
		if r.W.Cursor != nil {
			t.Errorf("%s: cursor not empty", c.code)
		}
	}
}

// A refused amulet goes back to the pack, the cursor is empty, and further Equip calls at the same level do not click again.
func TestEquipRefusedStaysInPackAndBacksOff(t *testing.T) {
	geartest.Need(t)
	r, lv, us := equipRig(t, "amu", "amu", "amu")
	for _, u := range us {
		r.W.EquipRefuse[u.ID] = true
	}
	ctx := context.Background()
	rep := lv.Equip(ctx)
	if len(rep.Failed) == 0 {
		t.Fatalf("no failure reported: %+v", rep)
	}
	for _, u := range us {
		if _, ok := worn(r.W, u.ID); ok {
			t.Errorf("item %d worn", u.ID)
		}
	}
	if r.W.Cursor != nil {
		t.Fatalf("cursor holds %d", r.W.Cursor.ID)
	}
	for _, u := range us {
		if !inPack(r.W, u.ID) {
			t.Errorf("item %d not in the pack", u.ID)
		}
	}
	for _, f := range rep.Failed {
		if !strings.Contains(f, "did not reach slot neck") {
			t.Errorf("failure text %q", f)
		}
	}
	n := len(r.W.Clicks)
	if rep2 := lv.Equip(ctx); len(rep2.Failed) != 0 || len(r.W.Clicks) != n {
		t.Fatalf("second visit at the same level clicked again: %+v, %d new clicks", rep2, len(r.W.Clicks)-n)
	}
	// two items failed into the slot: it stays blocked for two levels, then is tried again
	n = len(r.W.Clicks)
	r.W.Player.Level++
	if rep3 := lv.Equip(ctx); len(rep3.Failed) != 0 || len(r.W.Clicks) != n {
		t.Fatalf("retried one level later: %+v", rep3)
	}
	r.W.Player.Level++
	if rep3 := lv.Equip(ctx); len(rep3.Failed) == 0 {
		t.Fatalf("no retry two levels later: %+v", rep3)
	}
	if r.W.Cursor != nil {
		t.Fatal("cursor not empty after the retry")
	}
}

// One refused amulet must not stop a wearable one.
func TestEquipSkipsRefusedItemForAnother(t *testing.T) {
	geartest.Need(t)
	r, lv, us := equipRig(t, "amu", "amu")
	r.W.EquipRefuse[us[0].ID] = true
	rep := lv.Equip(context.Background())
	if _, ok := worn(r.W, us[1].ID); !ok {
		t.Fatalf("second amulet not worn: %+v", rep)
	}
	if !inPack(r.W, us[0].ID) || r.W.Cursor != nil {
		t.Fatal("refused amulet not back in the pack")
	}
}

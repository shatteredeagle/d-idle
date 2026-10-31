package town_test

import (
	"context"
	"d2lod-bot/internal/gear/geartest"
	"errors"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town"
)

// charsiRig: Charsi's dialog gets the Imbue line (index 2 of 4, the ImbueFlow table).
func charsiRig(t *testing.T) *rig {
	r := newRig(t)
	r.W.Lines = map[int]int{154: 4}
	r.W.Services = map[int]map[int]string{154: {2: "imbue"}}
	return r
}

// larzukRig moves the world to Harrogath; the Talk line (0 of 3) opens the picker.
func larzukRig(t *testing.T) *rig {
	r := newRig(t)
	w := r.W
	w.Player.Area, w.Player.Strength, w.Player.Dexterity = 109, 120, 120
	w.AddNPC(511, 5100, 5030)
	w.Services = map[int]map[int]string{511: {0: "socket"}}
	return r
}

func packItem(r *rig, code string, q, x, y int) d2.Unit {
	u := r.W.NewItem(code, q)
	u.Item.ILvl = 40
	return r.W.Put(u, d2.ContInventory, x, y)
}

func packBy(r *rig, id uint32) *d2.Unit {
	for i := range r.W.Items {
		if r.W.Items[i].ID == id {
			return &r.W.Items[i]
		}
	}
	return nil
}

func TestImbuePicksTheBiggestGain(t *testing.T) {
	geartest.Need(t)
	r := charsiRig(t)
	cap_ := packItem(r, "cap", d2.QualityNormal, 0, 0)
	qui := packItem(r, "qui", d2.QualityNormal, 2, 0)
	rep, err := r.T.Imbue(context.Background(), town.ImbueWant{})
	if err != nil {
		t.Fatalf("%v %+v", err, rep)
	}
	if got := packBy(r, qui.ID); got == nil || got.Item.Quality != d2.QualityRare {
		t.Fatalf("quilted armor not imbued: %+v", rep)
	}
	if packBy(r, cap_.ID).Item.Quality != d2.QualityNormal {
		t.Fatal("the cap was imbued too")
	}
	if r.W.UI != (d2.UIState{}) {
		t.Fatalf("panels left open: %v", r.W.UI)
	}
}

func TestImbueSkipsMagicSocketedAndEthereal(t *testing.T) {
	geartest.Need(t)
	r := charsiRig(t)
	packItem(r, "qui", d2.QualityMagic, 0, 0)
	s := packItem(r, "cap", d2.QualityNormal, 2, 0)
	s.Item.Flags |= d2.FlagSocketed
	packBy(r, s.ID).Item.Flags |= d2.FlagSocketed
	e := packItem(r, "lbt", d2.QualityNormal, 4, 0)
	packBy(r, e.ID).Item.Flags |= d2.FlagEthereal
	_, err := r.T.Imbue(context.Background(), town.ImbueWant{})
	if !errors.Is(err, town.ErrNoTarget) {
		t.Fatalf("want ErrNoTarget, got %v", err)
	}
	if len(r.W.Clicks) != 0 {
		t.Fatalf("the NPC was clicked although nothing qualified: %v", r.W.Clicks)
	}
}

func TestImbueTakesAWornWeaponOffFirst(t *testing.T) {
	geartest.Need(t)
	r := charsiRig(t)
	ssd := r.W.Add("ssd", d2.ContEquipment, 0, 0)
	r.W.Items[len(r.W.Items)-1].Item.Body = d2.BodyRHand
	rep, err := r.T.Imbue(context.Background(), town.ImbueWant{})
	if err != nil {
		t.Fatalf("%v %+v", err, rep)
	}
	got := packBy(r, ssd.ID)
	if !rep.WasWorn || got.Item.Where != d2.ContInventory || got.Item.Quality != d2.QualityRare {
		t.Fatalf("%+v %+v", rep, got.Item)
	}
}

func TestImbueRefusesWhenTheDialogHasNoImbueLine(t *testing.T) {
	geartest.Need(t)
	r := newRig(t) // plain Charsi: the Imbue click lands on Cancel
	packItem(r, "qui", d2.QualityNormal, 0, 0)
	if _, err := r.T.Imbue(context.Background(), town.ImbueWant{}); err == nil {
		t.Fatal("imbue claimed success")
	}
	if q := r.W.Items[len(r.W.Items)-1].Item.Quality; q != d2.QualityNormal {
		t.Fatalf("quality %d", q)
	}
}

func TestSocketPrefersTheRunewordBase(t *testing.T) {
	geartest.Need(t)
	r := larzukRig(t)
	qui := packItem(r, "qui", d2.QualityNormal, 0, 0)
	cap_ := packItem(r, "cap", d2.QualityNormal, 4, 0)
	rep, err := r.T.Socket(context.Background(), town.SocketWant{})
	if err != nil {
		t.Fatalf("%v %+v", err, rep)
	}
	// Spirit and Insight are ladder rows in the 1.14d tables (RunewordAvailableOffline false), so the 2-socket Smoke
	// (body armor, score 55) beats Lore (helm, 30).
	if !strings.Contains(rep.Why, "Smoke") {
		t.Errorf("why %q", rep.Why)
	}
	got := packBy(r, qui.ID)
	if !got.Item.Socketed() || r.W.Stats[got.Addr].Get(194) < 1 {
		t.Fatalf("armor not socketed: %+v", got.Item)
	}
	if packBy(r, cap_.ID).Item.Socketed() {
		t.Fatal("cap socketed too")
	}
}

func TestSocketWantListAndExplicitItem(t *testing.T) {
	r := larzukRig(t)
	qui := packItem(r, "qui", d2.QualityNormal, 0, 0)
	packItem(r, "bsw", d2.QualityNormal, 4, 0)
	if _, err := r.T.Socket(context.Background(), town.SocketWant{Item: qui.ID}); err != nil {
		t.Fatal(err)
	}
	if !packBy(r, qui.ID).Item.Socketed() {
		t.Fatal("explicit item not socketed")
	}
}

func TestSocketNeedsAct5(t *testing.T) {
	r := larzukRig(t)
	r.W.Player.Area = 1
	packItem(r, "qui", d2.QualityNormal, 0, 0)
	if _, err := r.T.Socket(context.Background(), town.SocketWant{}); !errors.Is(err, town.ErrNotInTown) {
		t.Fatalf("%v", err)
	}
}

func TestExpectedSocketsCapsByIlvl(t *testing.T) {
	r := newRig(t)
	db := r.W.DB
	for _, c := range []struct {
		code string
		ilvl int
		min  int
		max  int
	}{{"qui", 40, 2, 2}, {"bsw", 40, 3, 6}, {"bsw", 1, 1, 6}} {
		for _, b := range db.Items {
			if b.Code != c.code {
				continue
			}
			n := town.ExpectedSockets(db, b, c.ilvl)
			t.Logf("%s ilvl %d -> %d (gemsockets %d)", c.code, c.ilvl, n, b.Sockets)
			if n < c.min || n > c.max || n > b.Sockets {
				t.Errorf("%s ilvl %d: %d", c.code, c.ilvl, n)
			}
		}
	}
}

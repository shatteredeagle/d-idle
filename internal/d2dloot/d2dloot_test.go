package d2dloot

import (
	"context"
	"errors"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dsrc"
	"d2lod-bot/internal/data"
)

func db() *data.DB {
	return &data.DB{Items: []*data.ItemBase{
		{Code: "gld", Type: "gold", Display: "Gold"},
		{Code: "ssd", Type: "swor", Display: "Short Sword"},
		{Code: "hp1", Type: "hpot", Display: "Minor Healing Potion", Stackable: true},
	}}
}

func ground(id uint32, class, x, y int) d2.Unit {
	return d2.Unit{Type: d2.TypeItem, ID: id, Class: class, Mode: d2.ItemModeGround, X: x, Y: y, Item: &d2.Item{Where: d2.ContGround}}
}

func result() *d2dsrc.Result {
	snap := &d2.Snapshot{Player: &d2.Player{X: 100, Y: 200}}
	snap.Ground = []d2.Unit{ground(1, 0, 90, 190), ground(2, 1, -1, -1), ground(3, 1, 5, 5), ground(4, 1, 6, 6), ground(5, 1, 7, 7), ground(6, 2, 8, 8)}
	r := &d2dsrc.Result{Snap: snap}
	r.Extras.Items = map[uint32]d2dsrc.ItemExtra{
		1: {Code: "gld", Label: "8800 Gold"},
		2: {Code: "ssd", Label: "Short Sword"},
		3: {Code: "ssd", Label: "Superior Short Sword"},
		4: {Code: "ssd", Label: "Sharp Short Sword of Fire"},
		5: {Code: "ssd", Label: "Beast Claw"},
		6: {Code: "hp1", Label: "Minor Healing Potion (3)"},
	}
	return r
}

func TestEnrich(t *testing.T) {
	s := &Source{DB: db()}
	r := result()
	s.Enrich(r)
	g := r.Snap.Ground
	if g[0].Addr != 1 {
		t.Errorf("Addr %d, want the unit id", g[0].Addr)
	}
	if g[1].X != 100 || g[1].Y != 200 {
		t.Errorf("unknown position not moved to the player: %d,%d", g[1].X, g[1].Y)
	}
	if g[2].X != 5 {
		t.Errorf("known position changed")
	}
	for i, want := range map[int]int{1: d2.QualityNormal, 2: d2.QualitySuperior, 3: d2.QualityMagic, 4: d2.QualityRare, 5: d2.QualityNormal} {
		if got := g[i].Item.Quality; got != want {
			t.Errorf("ground[%d] quality %d, want %d", i, got, want)
		}
	}
	st, err := s.Stats(1)
	if err != nil || st.Get(d2.StatGold) != 8800 {
		t.Errorf("gold stats = %v, %v", st, err)
	}
	if _, err := s.Stats(2); !errors.Is(err, d2.ErrUnsupported) {
		t.Errorf("non-gold stats: %v", err)
	}
}

func TestSnapshotReadError(t *testing.T) {
	boom := errors.New("boom")
	s := &Source{Read: func(context.Context) (*d2dsrc.Result, error) { return nil, boom }}
	if _, err := s.Snapshot(); !errors.Is(err, boom) {
		t.Fatalf("err %v", err)
	}
	s.Read = func(context.Context) (*d2dsrc.Result, error) { return result(), nil }
	s.DB = db()
	snap, err := s.Snapshot()
	if err != nil || snap.Ground[0].Addr != 1 {
		t.Fatalf("snapshot %v %v", snap, err)
	}
}

func TestGoldAmount(t *testing.T) {
	for in, want := range map[string]int{"8800 Gold": 8800, " 3 gold": 3} {
		if n, ok := GoldAmount(in); !ok || n != want {
			t.Errorf("%q = %d %v", in, n, ok)
		}
	}
	if _, ok := GoldAmount("Gold Ring"); ok {
		t.Error("Gold Ring parsed as a pile")
	}
}

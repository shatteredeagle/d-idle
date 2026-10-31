package d2

import (
	"reflect"
	"testing"
)

// Child stat-list chain on a monster: a minion carries state 105, a cursed hostile only state 9 (layout V live).
func TestMonsterStates(t *testing.T) {
	fx := buildInGame()
	w := fx.world
	mk := func(class, id uint32, states ...uint32) {
		m := w.unit(TypeMonster, class, id, 1, 6060, 4960)
		w.set(m, uoData, w.alloc(0x60))
		sl := w.statList([]statKV{{6, 0x8000}, {7, 0x8000}}, nil, false)
		var prev uint32
		for _, s := range states {
			c := w.alloc(0x60)
			w.set(c, slChildState, s)
			w.set(c, slChildPrev, prev)
			prev = c
		}
		w.set(sl, slChild, prev)
		w.set(m, uoStats, sl)
		w.addToTable(TypeMonster, absSrv, m)
	}
	mk(363, 40, 97, 105)
	mk(1, 41, 9)
	mk(1, 42)
	snap, err := reader(fx.f).Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	got := map[uint32][]uint8{}
	for _, u := range snap.Monsters {
		if u.ID >= 40 {
			got[u.ID] = u.Monster.States
			if u.Monster.HasState(StateMinion) != (u.ID == 40) {
				t.Errorf("unit %d HasState(105) wrong", u.ID)
			}
		}
	}
	want := map[uint32][]uint8{40: {105, 97}, 41: {9}, 42: nil}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("states = %v, want %v", got, want)
	}
}

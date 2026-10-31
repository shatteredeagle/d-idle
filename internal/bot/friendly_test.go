package bot

import (
	"testing"

	"d2lod-bot/internal/d2"
)

type aiCatalog struct{ fakeCatalog }

func (aiCatalog) Monster(class int) (MonsterMeta, bool) {
	switch class {
	case 20:
		return MonsterMeta{Name: "Hydra", Killable: true, AI: "Hydra"}, true
	case 21:
		return MonsterMeta{Name: "Hireling", Killable: true, AI: "Hireable"}, true
	}
	return fakeCatalog{}.Monster(class)
}

func TestHostileUsesIsFriendly(t *testing.T) {
	r := &Runner{d: Deps{Catalog: aiCatalog{}}}
	p := &d2.Player{ID: 1}
	mk := func(id uint32, class int, owner uint32) d2.Unit {
		return d2.Unit{ID: id, Type: d2.TypeMonster, Class: class, Mode: d2.MonModeStand, OwnerID: owner}
	}
	s := &d2.Snapshot{Player: p, Monsters: []d2.Unit{mk(5, 10, 0), mk(6, 20, 0), mk(7, 21, 0), mk(8, 10, 1), mk(9, 11, 0)}}
	got := map[uint32]bool{}
	for i := range s.Monsters {
		got[s.Monsters[i].ID] = r.hostile(s, &s.Monsters[i])
	}
	want := map[uint32]bool{5: true, 6: false, 7: false, 8: false, 9: false}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("unit %d hostile=%v want %v", id, got[id], w)
		}
	}
}

type ignFighter struct {
	fakeFighter
	ign map[uint32]bool
}

func (f *ignFighter) IsIgnored(id uint32, _ float64) bool { return f.ign[id] }

// Units the fighter gave up on must not count as hostileNear: a Clear on them kills nothing (live: 'clear killed
// nothing; moving on' with hostileNear 3-4 and remaining 0).
func TestHostileNearSkipsIgnored(t *testing.T) {
	f := &ignFighter{ign: map[uint32]bool{5: true}}
	r := &Runner{d: Deps{Catalog: aiCatalog{}, Fighter: f}}
	s := &d2.Snapshot{Player: &d2.Player{ID: 1, Area: 2}, Monsters: []d2.Unit{
		{ID: 5, Type: d2.TypeMonster, Class: 10, Mode: d2.MonModeStand, X: 10},
		{ID: 6, Type: d2.TypeMonster, Class: 10, Mode: d2.MonModeStand, X: 12},
	}}
	if n := r.hostileNear(s, 30); n != 1 {
		t.Fatalf("hostileNear = %d, want 1", n)
	}
}

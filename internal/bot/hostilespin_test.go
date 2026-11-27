package bot

import (
	"testing"

	"d2lod-bot/internal/d2"
)

// countCatalog counts monster lookups: a hostile scan must cost a bounded number per unit, not one owner-chain pass per unit.
type countCatalog struct {
	aiCatalog
	n *int
}

func (c countCatalog) Monster(class int) (MonsterMeta, bool) {
	*c.n++
	return c.aiCatalog.Monster(class)
}

// crowd is n Fallen around the player plus the owner shapes the hostile check must survive: a self-owned unit, a
// two-unit owner cycle (type 1, so the chain pass follows it) and the player's own pet. featuresoak pass 2 (Sorceress
// 201, arbiter families 255) had 53 hostiles near when the goroutine dump caught the guard in this scan.
func crowd(class d2.Class, n int) *d2.Snapshot {
	p := &d2.Player{ID: 1, Class: class, Area: 2, X: 100, Y: 100}
	mk := func(id uint32, cls int, owner uint32, otype uint32) d2.Unit {
		return d2.Unit{ID: id, Type: d2.TypeMonster, Class: cls, Mode: d2.MonModeStand, X: 101, Y: 101, OwnerID: owner, OwnerType: otype}
	}
	s := &d2.Snapshot{Player: p}
	for i := range n {
		s.Monsters = append(s.Monsters, mk(uint32(100+i), 10, 0, 0))
	}
	s.Monsters = append(s.Monsters,
		mk(7, 10, 7, 1), // owns itself
		mk(8, 10, 9, 1), // 8 <-> 9 owner cycle
		mk(9, 10, 8, 1),
		mk(5, 10, 1, 0), // the player's pet
	)
	return s
}

func TestHostileScanBuildsFriendsOnce(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		const n = 60
		calls := 0
		r := &Runner{d: Deps{Catalog: countCatalog{n: &calls}}}
		s := crowd(class, n)
		got := r.hostileNear(s, 30)
		if want := n + 3; got != want { // the cycle and the self-owner are hostile, the pet is not
			t.Errorf("%v: hostileNear=%d want %d", class, got, want)
		}
		// Per unit: one meta lookup and one Friendlies lookup. Building the set per unit is about n*n (3600+).
		if units := len(s.Monsters); calls > 3*units {
			t.Errorf("%v: %d catalog lookups for %d units, want at most %d (friendly set rebuilt per unit)", class, calls, units, 3*units)
		}
	}
}

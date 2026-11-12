package loot_test

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/town/sim"
)

// memRig puts the hero in the field (area 2) as the given class at the sim's start spot (6020,4930).
func memRig(t *testing.T, class d2.Class, clvl int) (*loot.Looter, *sim.World) {
	l, r := newLooter(t)
	w := r.W
	w.Player.Class, w.Player.Level, w.Player.Area = class, clvl, 2
	return l, w
}

func observe(t *testing.T, l *loot.Looter, w *sim.World) {
	t.Helper()
	s, _ := w.Snapshot()
	l.ObserveGround(s)
}

func pile(w *sim.World, x, y int, amt int32) d2.Unit {
	u := w.AddGround("gld", 2, x, y)
	w.Stats[u.Addr] = d2.Stats{d2.StatGold: amt}
	return u
}

var classes = []struct {
	name  string
	class d2.Class
}{{"sorceress", d2.Sorceress}, {"paladin", d2.Paladin}}

func TestMemoryDropBeyondRadiusIsFetched(t *testing.T) {
	for _, c := range classes {
		t.Run(c.name, func(t *testing.T) {
			l, w := memRig(t, c.class, 10)
			w.Player.Gold = 5000
			pile(w, 6100, 4930, 80) // 80 away: beyond the 30 radius, inside the 120 reach
			observe(t, l, w)
			if len(l.Remembered()) != 1 {
				t.Fatalf("remembered %+v", l.Remembered())
			}
			if rep, _ := l.PickUp(context.Background(), loot.PickOpts{}); len(rep.Picked) != 0 {
				t.Fatalf("ordinary pickup reached it: %+v", rep)
			}
			var told int
			n, err := l.WalkBack(context.Background(), nil, func(n, gold int, first string) { told = gold })
			if err != nil || n != 1 || told != 80 {
				t.Fatalf("trips %d told %d err %v", n, told, err)
			}
			if len(w.Ground) != 0 || len(l.Remembered()) != 0 {
				t.Fatalf("ground %d remembered %+v", len(w.Ground), l.Remembered())
			}
		})
	}
}

func TestMemoryPickedByPursuitDoesNotDuplicate(t *testing.T) {
	l, w := memRig(t, d2.Paladin, 10)
	w.Player.Gold = 5000
	pile(w, 6040, 4930, 80)
	observe(t, l, w)
	if _, err := l.PickUp(context.Background(), loot.PickOpts{}); err != nil { // within 30: the ordinary pickup takes it
		t.Fatal(err)
	}
	observe(t, l, w)
	if len(l.Remembered()) != 0 {
		t.Fatalf("picked drop still remembered: %+v", l.Remembered())
	}
	if n, _ := l.WalkBack(context.Background(), nil, nil); n != 0 {
		t.Fatalf("walked %d trips for nothing", n)
	}
}

func TestMemoryItemThatLeftTheGroundIsForgotten(t *testing.T) {
	l, w := memRig(t, d2.Sorceress, 10)
	u := pile(w, 6050, 4930, 80)
	observe(t, l, w)
	w.Ground = nil // somebody else took it, in view (20 away)
	_ = u
	observe(t, l, w)
	if len(l.Remembered()) != 0 {
		t.Fatalf("%+v", l.Remembered())
	}
	// far away (unloaded room?): kept
	pile(w, 6200, 4930, 80)
	observe(t, l, w)
	w.Ground = nil
	observe(t, l, w)
	if len(l.Remembered()) != 1 {
		t.Fatalf("far entry dropped: %+v", l.Remembered())
	}
}

func TestMemoryCapEvictsLowestValue(t *testing.T) {
	l, w := memRig(t, d2.Paladin, 10)
	l.Env().Tune.Loot.MemCap = 4
	for i := 0; i < 6; i++ {
		pile(w, 6100+i, 4930, int32(30+10*i))
	}
	observe(t, l, w)
	got := l.Remembered()
	if len(got) != 4 {
		t.Fatalf("kept %d", len(got))
	}
	for _, r := range got {
		if r.Gold < 50 {
			t.Fatalf("low pile kept: %+v", got)
		}
	}
}

func TestMemoryResetAndAreas(t *testing.T) {
	l, w := memRig(t, d2.Sorceress, 10)
	pile(w, 6100, 4930, 80)
	observe(t, l, w)
	w.Player.Area = 3 // another area: the entry stays, and is not fetched from there
	w.Ground = nil
	observe(t, l, w)
	if len(l.Remembered()) != 1 {
		t.Fatal("entry lost on a zone change")
	}
	if n, _ := l.WalkBack(context.Background(), nil, nil); n != 0 {
		t.Fatal("walked back across areas")
	}
	l.ResetMemory()
	if len(l.Remembered()) != 0 {
		t.Fatal("new game kept entries")
	}
}

func TestMemoryTrashedNeverEnters(t *testing.T) {
	l, w := memRig(t, d2.Paladin, 10)
	u := pile(w, 6100, 4930, 80)
	l.SetIgnore(func(x d2.Unit) bool { return x.ID == u.ID })
	observe(t, l, w)
	if len(l.Remembered()) != 0 {
		t.Fatal("ignored item remembered")
	}
}

func TestMemoryMinValueAndDistance(t *testing.T) {
	l, w := memRig(t, d2.Sorceress, 20) // bar: 20 * (1 + 1.9) = 58
	w.Player.Gold = 5000
	pile(w, 6100, 4930, 40)  // too small for clvl 20
	pile(w, 6110, 4930, 100) // fine
	pile(w, 6400, 4930, 500) // beyond 120
	observe(t, l, w)
	n, _ := l.WalkBack(context.Background(), nil, nil)
	if n != 1 || len(w.Ground) != 2 {
		t.Fatalf("trips %d ground %d", n, len(w.Ground))
	}
}

func TestMemoryMaxTrips(t *testing.T) {
	l, w := memRig(t, d2.Paladin, 1)
	w.Player.Gold = 5000
	l.Env().Tune.Loot.MemMaxTrips = 2
	l.Env().Tune.Loot.GoldPriority = 0 // gold is exempt from the cap under the priority (goldprio_test.go)
	for i := 0; i < 4; i++ {           // 30 apart: each needs its own trip
		pile(w, 6060+30*i, 4930, 80)
	}
	observe(t, l, w)
	if n, _ := l.WalkBack(context.Background(), nil, nil); n != 2 {
		t.Fatalf("trips %d", n)
	}
}

func TestMemoryNoWalkBackWhenHostile(t *testing.T) {
	l, w := memRig(t, d2.Sorceress, 10)
	w.Player.Gold = 5000
	pile(w, 6100, 4930, 80)
	observe(t, l, w)
	if n, _ := l.WalkBack(context.Background(), func() bool { return true }, nil); n != 0 || len(w.Ground) != 1 {
		t.Fatalf("walked with a hostile near: %d", n)
	}
}

func TestMemoryOff(t *testing.T) {
	l, w := memRig(t, d2.Sorceress, 10)
	l.Env().Tune.Loot.MemEnabled = 0
	pile(w, 6100, 4930, 80)
	observe(t, l, w)
	if len(l.Remembered()) != 0 {
		t.Fatal("recorded while off")
	}
}

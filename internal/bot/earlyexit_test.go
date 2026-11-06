package bot

import (
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

// earlyRig stands a hero of class c in area 3 with n hostiles close by and one portal scroll.
func earlyRig(t *testing.T, c d2.Class, hostiles int) (*rig, *d2.Snapshot) {
	g := newRig(t)
	g.w.set(func(w *world) {
		w.class, w.area, w.level, w.life, w.maxL = c, 3, 4, 40, 69
		w.items = append(w.items, d2.Unit{ID: 950, Type: d2.TypeItem, Class: 5, Item: &d2.Item{Where: d2.ContInventory}})
	})
	for i := 0; i < hostiles; i++ {
		g.w.addMonster(uint32(500+i), 10, 2+i%3, 1)
	}
	s, _ := g.w.snapshot()
	return g, s
}

// Seed 13 of g2f drank 11 potions in about 17 s inside a 12-hostile pack: the drink after the burst is a portal.
func TestPotionBurstBecomesPortal(t *testing.T) {
	for _, c := range []d2.Class{d2.Sorceress, d2.Barbarian} {
		g, s := earlyRig(t, c, 6)
		for i := 0; i < 6; i++ {
			want := SafetyDrinkLife
			if i == 5 {
				want = SafetyTownPortal // the 6th drink in 6 s: more than potionBurstN (5) in the window
			}
			if got := g.r.earlyExit(s, SafetyDrinkLife); got != want {
				t.Fatalf("%v: drink %d = %v, want %v", c, i, got, want)
			}
			g.clock.Sleep(time.Second)
		}
		if a := g.r.stalePackArea(s); a != 3 {
			t.Errorf("%v: the portal exit must leave a stale pack in area 3, got %d", c, a)
		}
	}
}

func TestPotionBurstNeedsPackAndPortal(t *testing.T) {
	g, s := earlyRig(t, d2.Sorceress, 1) // one hostile: keep drinking
	for i := 0; i < 9; i++ {
		if a := g.r.earlyExit(s, SafetyDrinkLife); a != SafetyDrinkLife {
			t.Fatalf("one hostile: drink %d became %v", i, a)
		}
	}
	g, s = earlyRig(t, d2.Sorceress, 6)
	g.e.Tune.Bot.PotionBurstN = 0 // rule off
	for i := 0; i < 9; i++ {
		if a := g.r.earlyExit(s, SafetyDrinkLife); a != SafetyDrinkLife {
			t.Fatalf("rule off: drink %d became %v", i, a)
		}
	}
}

func TestStalePackHold(t *testing.T) {
	for _, c := range []d2.Class{d2.Sorceress, d2.Barbarian} {
		g, s := earlyRig(t, c, 12)
		g.r.earlyExit(s, SafetyChicken)
		if a := g.r.stalePackArea(s); a != 3 {
			t.Fatalf("%v: stale area %d, want 3", c, a)
		}
		ex := map[int]bool{}
		g.r.guardedAreasNoFarm(s, ex)
		if !ex[3] {
			t.Errorf("%v: the planner is not kept out of the stale area", c)
		}
		// a rebuilt kit (5 tier-2 potions) lifts the hold
		g.w.belt(101, 102, 103, 104, 105)
		s2, _ := g.w.snapshot()
		if a := g.r.stalePackArea(s2); a != 0 {
			t.Errorf("%v: rebuilt kit must lift the hold, got %d", c, a)
		}
		// so does time
		g.w.belt(101)
		s3, _ := g.w.snapshot()
		if g.r.stalePackArea(s3) != 3 {
			t.Fatalf("%v: thin kit must keep the hold", c)
		}
		g.clock.Sleep(11 * time.Minute)
		if a := g.r.stalePackArea(s3); a != 0 {
			t.Errorf("%v: hold must expire, got %d", c, a)
		}
	}
}

func TestStalePackSmallPackIgnored(t *testing.T) {
	g, s := earlyRig(t, d2.Sorceress, 2)
	g.r.earlyExit(s, SafetyChicken)
	if a := g.r.stalePackArea(s); a != 0 {
		t.Errorf("a 2-hostile exit must not hold the area, got %d", a)
	}
}

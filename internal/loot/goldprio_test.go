package loot_test

import (
	"context"
	"fmt"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/town/sim"
)

// goldRig is memRig with the tuning in reach.
func goldRig(t *testing.T, class d2.Class, prio int) (*loot.Looter, *sim.World, *env.Tuning) {
	l, r := newLooter(t)
	w := r.W
	w.Player.Class, w.Player.Level, w.Player.Area, w.Player.Gold = class, 10, 2, 5000
	r.Env.Tune.Loot.GoldPriority = prio
	return l, w, r.Env.Tune
}

// A far pile whose click failed stays remembered under goldPriority (it used to be forgotten after two rounds); with
// the knob off the old rule holds. The ordinary pickup (the pile is in radius now) then takes it.
func TestGoldFailedClickStaysRemembered(t *testing.T) {
	for _, c := range classes {
		for _, prio := range []int{1, 0} {
			t.Run(fmt.Sprintf("%s/prio%d", c.name, prio), func(t *testing.T) {
				l, w, _ := goldRig(t, c.class, prio)
				pile(w, 6100, 4930, 80)
				observe(t, l, w)
				w.NoHover = 6 // one whole PickUp call (two rounds of three clicks) fails
				if _, err := l.WalkBack(context.Background(), nil, nil); err != nil {
					t.Fatal(err)
				}
				if kept := len(l.Remembered()); kept != prio {
					t.Fatalf("remembered %d, want %d", kept, prio)
				}
				if _, err := l.PickUp(context.Background(), loot.PickOpts{}); err != nil {
					t.Fatal(err)
				}
				if len(w.Ground) != 0 {
					t.Fatal("pile not taken by the next pickup")
				}
			})
		}
	}
}

// A pile that never works is given up after GoldRetries rounds, not retried forever.
func TestGoldRetriesAreBounded(t *testing.T) {
	l, w, tune := goldRig(t, d2.Paladin, 1)
	tune.Loot.GoldRetries = 2
	pile(w, 6040, 4930, 80)
	w.NoHover = 1000
	for i := 0; i < 4; i++ {
		_, _ = l.PickUp(context.Background(), loot.PickOpts{})
	}
	if len(w.Ground) != 1 {
		t.Fatalf("ground %d", len(w.Ground))
	}
	if l.GoldNear(snap(w), 100) {
		t.Fatal("a given-up pile still counts as near")
	}
}

func snap(w *sim.World) *d2.Snapshot { s, _ := w.Snapshot(); return s }

// GoldNear is the cheap poll: true for a wanted pile in range, false beyond, false with the knob off.
func TestGoldNear(t *testing.T) {
	for _, c := range classes {
		t.Run(c.name, func(t *testing.T) {
			l, w, tune := goldRig(t, c.class, 1)
			pile(w, 6040, 4930, 80)
			s := snap(w)
			if !l.GoldNear(s, 30) {
				t.Fatal("pile 20 away not near")
			}
			if l.GoldNear(s, 10) {
				t.Fatal("pile 20 away is near within 10")
			}
			tune.Loot.GoldPriority = 0
			if l.GoldNear(s, 30) {
				t.Fatal("near with the priority off")
			}
		})
	}
}

// Walk-back for gold is exempt from the item trip cap (up to four times the cap) under goldPriority.
func TestGoldWalkBackExceedsTripCap(t *testing.T) {
	for _, prio := range []int{1, 0} {
		t.Run(fmt.Sprintf("prio%d", prio), func(t *testing.T) {
			l, w, tune := goldRig(t, d2.Sorceress, prio)
			tune.Loot.MemMaxTrips = 2
			tune.Loot.MemMaxDist = 400
			for i := 0; i < 6; i++ {
				pile(w, 6060+30*i, 4930, 80)
			}
			observe(t, l, w)
			n, err := l.WalkBack(context.Background(), nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			want := 2
			if prio == 1 {
				want = 6
			}
			if n != want {
				t.Fatalf("trips %d, want %d", n, want)
			}
		})
	}
}

// The ledger counts piles seen, picked, and those still remembered when the hero leaves the area.
func TestGoldLedger(t *testing.T) {
	l, w, _ := goldRig(t, d2.Paladin, 1)
	pile(w, 6040, 4930, 80)
	pile(w, 6110, 4930, 90)
	observe(t, l, w)
	if _, err := l.PickUp(context.Background(), loot.PickOpts{}); err != nil {
		t.Fatal(err)
	}
	w.Player.Area = 3 // leave with the far pile uncollected
	observe(t, l, w)
	g := l.GoldLedger()
	if g.Seen != 2 || g.Picked != 1 || g.Abandoned != 1 {
		t.Fatalf("ledger %+v", g)
	}
}

// GoldOnly leaves everything else on the ground.
func TestGoldOnlyPickup(t *testing.T) {
	l, w, _ := goldRig(t, d2.Paladin, 1)
	pile(w, 6040, 4930, 80)
	w.AddGround("hp1", 2, 6035, 4930)
	rep, err := l.PickUp(context.Background(), loot.PickOpts{GoldOnly: true, Max: 5})
	if err != nil || rep.GoldGained == 0 {
		t.Fatalf("rep %+v err %v", rep, err)
	}
	if len(w.Ground) != 1 {
		t.Fatalf("ground %d, the potion must stay", len(w.Ground))
	}
}

package town_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"d2lod-bot/internal/town"
	"d2lod-bot/internal/town/sim"
)

// chainMover is a navigator with the level-chain walk (nav.Navigator.ToArea) and a waypoint that was never activated.
type chainMover struct {
	*sim.Mover
	toArea []int
}

func (c *chainMover) TakeWaypoint(_ context.Context, level int) error {
	c.Calls = append(c.Calls, fmt.Sprintf("waypoint %d", level))
	return errors.New("nav: waypoint unavailable: none on level 5")
}

func (c *chainMover) ToArea(_ context.Context, level int) error {
	c.toArea = append(c.toArea, level)
	c.W.Player.Area = level
	return nil
}

// BELTFIX3 (2), Gate 1 20261102 BELT A: Dark Wood, empty tome, waypoint not activated, no hop: the trip failed 62 times.
// Now the level chain leads home.
func TestToTownWalksChainFromDarkWood(t *testing.T) {
	for _, area := range []int{5, 4, 10, 6} {
		r := newRig(t)
		cm := &chainMover{Mover: r.Mv}
		r.T = town.New(r.Env, cm)
		r.T.UseSource(town.Source{Snapshot: r.W.Snapshot, Items: r.W.AllItems, Owner: r.W.Owner, Stats: r.W.UnitStats})
		r.W.Player.Area = area // no scroll in the pack
		if err := r.T.ToTown(context.Background()); err != nil {
			t.Fatalf("area %d: %v (calls %v)", area, err, r.Mv.Calls)
		}
		if r.W.Player.Area != 1 || len(cm.toArea) != 1 || cm.toArea[0] != 1 {
			t.Errorf("area %d: arrived in %d, ToArea calls %v", area, r.W.Player.Area, cm.toArea)
		}
	}
}

// BELTFIX3 (4), Gate 1 NOKILL B: the first click on the town portal was dropped and BackToField gave the portal up
// for a 14 s walk. It retries the portal once first.
func TestBackToFieldRetriesPortalOnce(t *testing.T) {
	r := newRig(t)
	r.W.Player.Area, r.W.PortalLevel = 1, 6
	portalID := r.W.AddObject(59, r.W.Player.X+10, r.W.Player.Y)
	d := &deafPortal{Controller: r.Env.In, n: 1, portal: func(id uint32) bool { return id == portalID.ID }}
	r.Env.In = d
	r.T = town.New(r.Env, r.Mv)
	r.T.UseSource(town.Source{Snapshot: r.W.Snapshot, Items: r.W.AllItems, Owner: r.W.Owner, Stats: r.W.UnitStats})
	if err := r.T.BackToField(context.Background(), 6); err != nil || r.W.Player.Area != 6 {
		t.Fatalf("err %v area %d", err, r.W.Player.Area)
	}
	if d.seen != 2 {
		t.Errorf("%d portal interactions, want 2", d.seen)
	}
	for _, c := range r.Mv.Calls {
		if strings.HasPrefix(c, "exitto") || strings.HasPrefix(c, "waypoint") {
			t.Fatalf("walked or took the waypoint: %v", r.Mv.Calls)
		}
	}
}

// codefix F4: the chain walk is for the quest-route fields only; Catacombs 4 or Andariel's Lair keep ErrNoPortal at once.
func TestToTownNoChainWalkOutsideTheRoute(t *testing.T) {
	for _, area := range []int{7, 37, 29, 35} {
		r := newRig(t)
		cm := &chainMover{Mover: r.Mv}
		r.T = town.New(r.Env, cm)
		r.T.UseSource(town.Source{Snapshot: r.W.Snapshot, Items: r.W.AllItems, Owner: r.W.Owner, Stats: r.W.UnitStats})
		r.W.Player.Area = area
		if err := r.T.ToTown(context.Background()); !errors.Is(err, town.ErrNoPortal) {
			t.Errorf("area %d: %v, want ErrNoPortal", area, err)
		}
		if len(cm.toArea) != 0 {
			t.Errorf("area %d: walked the chain %v", area, cm.toArea)
		}
	}
}

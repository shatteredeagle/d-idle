package town_test

import (
	"context"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/town"
)

// staleOnly ignores every interaction with the listed portal ids (a stale object: the click never changes the level).
type staleOnly struct {
	input.Controller
	ids    map[uint32]bool
	clicks map[uint32]int
}

func (s *staleOnly) Interact(t input.Target) error {
	if s.ids[t.ID] {
		s.clicks[t.ID]++
		return nil
	}
	return s.Controller.Interact(t)
}

func staleRig(t *testing.T, class d2.Class, scrolls int) (*rig, *staleOnly, uint32) {
	r := newRig(t)
	r.W.Player.Class = class
	r.W.Player.Area, r.W.PortalLevel = 1, 6
	stale := r.W.AddObject(59, r.W.Player.X+10, r.W.Player.Y)
	for i := 0; i < scrolls; i++ {
		r.W.Add("tsc", d2.ContInventory, i, 0)
	}
	d := &staleOnly{Controller: r.Env.In, ids: map[uint32]bool{stale.ID: true}, clicks: map[uint32]int{}}
	r.Env.In = d
	r.T = town.New(r.Env, r.Mv)
	r.T.UseSource(town.Source{Snapshot: r.W.Snapshot, Items: r.W.AllItems, Owner: r.W.Owner, Stats: r.W.UnitStats})
	return r, d, stale.ID
}

func walked(r *rig) bool {
	for _, c := range r.Mv.Calls {
		if strings.HasPrefix(c, "exitto") || strings.HasPrefix(c, "waypoint") {
			return true
		}
	}
	return false
}

func TestBackToFieldStalePortalDroppedFreshPortal(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Barbarian} {
		r, d, id := staleRig(t, class, 2)
		if err := r.T.BackToField(context.Background(), 6); err != nil || r.W.Player.Area != 6 {
			t.Fatalf("class %v: err %v area %d", class, err, r.W.Player.Area)
		}
		if d.clicks[id] != 2 {
			t.Errorf("class %v: %d clicks on the stale portal, want 2", class, d.clicks[id])
		}
		if walked(r) {
			t.Errorf("class %v: walked or took the waypoint: %v", class, r.Mv.Calls)
		}
	}
}

func TestBackToFieldStalePortalNoScrollTakesWaypoint(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Barbarian} {
		r, _, _ := staleRig(t, class, 0)
		_ = r.T.BackToField(context.Background(), 6)
		if !strings.Contains(strings.Join(r.Mv.Calls, ","), "waypoint 6") {
			t.Errorf("class %v: moves %v, want the waypoint", class, r.Mv.Calls)
		}
	}
}

func TestBackToFieldFirstClickWorksUnchanged(t *testing.T) {
	r := newRig(t)
	r.W.Player.Area, r.W.PortalLevel = 1, 6
	r.W.AddObject(59, r.W.Player.X+10, r.W.Player.Y)
	if err := r.T.BackToField(context.Background(), 6); err != nil || r.W.Player.Area != 6 || walked(r) {
		t.Fatalf("err %v area %d calls %v", err, r.W.Player.Area, r.Mv.Calls)
	}
}

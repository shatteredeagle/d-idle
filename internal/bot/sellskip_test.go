package bot

import (
	"testing"

	"d2lod-bot/internal/d2"
)

// sellskip (hero14 finding 5): a clvl 14 hero without a town portal was sent to Blood Moor (0.6 xp, 0.26 gold per kill)
// because the first field was the only no-portal choice. With active waypoints it farms the area of its own level band.
func TestNoPortalRecoveryFarmsTheLevelBandViaWaypoints(t *testing.T) {
	g := newRig(t)
	g.w.set(func(w *world) { w.level = 14 })
	g.w.belt()
	s, _ := g.w.snapshot()
	if g.r.resInput(s).HasTP {
		t.Fatal("rig has a town portal")
	}
	if a, ok := g.r.recoveryArea(s); !ok || a != 2 {
		t.Fatalf("no waypoint known: want the first field (Blood Moor), got %d %v", a, ok)
	}
	g.r.markTouched(s, 3, 4, 5)
	a, ok := g.r.recoveryArea(s)
	if !ok || a == 2 || a == 3 {
		t.Fatalf("clvl 14 with waypoints picked %d %v, want a field of its band (Stony Field or Dark Wood)", a, ok)
	}
	g.r.avoid[a] = g.clock.Now().Add(20 * 60e9) // a banned band area falls back to the next reachable one
	if b, ok := g.r.recoveryArea(s); ok && (b == a || b == 3) {
		t.Fatalf("banned %d, got %d", a, b)
	}
}

// A dead merc's revive price is held back from the shop; an alive or unlisted one holds nothing; the knob switches it off.
func TestMercReserveOnlyForADeadMerc(t *testing.T) {
	g := newRig(t)
	g.w.set(func(w *world) { w.level = 14 })
	s, _ := g.w.snapshot()
	if n := g.r.mercReserve(s); n != 0 {
		t.Fatalf("no merc listed: reserve %d", n)
	}
	s.Monsters = append(s.Monsters, d2.Unit{Class: d2.MercClasses[0], Mode: d2.MonModeDead})
	if n := g.r.mercReserve(s); n != 600 {
		t.Fatalf("dead merc at clvl 14: reserve %d, want the quoted 600", n)
	}
	s.Monsters[len(s.Monsters)-1].Mode = 1
	if n := g.r.mercReserve(s); n != 0 {
		t.Fatalf("living merc: reserve %d", n)
	}
	s.Monsters[len(s.Monsters)-1].Mode = d2.MonModeDead
	g.e.Tune.Town.MercReserve = 0
	if n := g.r.mercReserve(s); n != 0 {
		t.Fatalf("knob off: reserve %d", n)
	}
}

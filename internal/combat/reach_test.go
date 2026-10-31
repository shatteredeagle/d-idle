package combat

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

func TestGridLOS(t *testing.T) {
	g := d2map.NewGrid()
	cells := make([]uint16, 40*10)
	for y := 0; y < 10; y++ {
		cells[y*40+20] = d2map.BlockLOS | d2map.BlockWalk // a wall at x=20
	}
	g.SetBlock(d2map.Rect{X: 0, Y: 0, W: 40, H: 10}, cells)
	if GridLOS(g, input.Point{X: 5, Y: 5}, input.Point{X: 35, Y: 5}) {
		t.Fatal("the wall blocks the shot")
	}
	if !GridLOS(g, input.Point{X: 5, Y: 5}, input.Point{X: 15, Y: 7}) {
		t.Fatal("same side is open")
	}
	// water: walk-blocked but not LOS-blocked, shots pass
	cells2 := make([]uint16, 40*10)
	for y := 0; y < 10; y++ {
		cells2[y*40+20] = d2map.BlockWalk
	}
	g2 := d2map.NewGrid()
	g2.SetBlock(d2map.Rect{X: 0, Y: 0, W: 40, H: 10}, cells2)
	if !GridLOS(g2, input.Point{X: 5, Y: 5}, input.Point{X: 35, Y: 5}) {
		t.Fatal("a river does not stop a bolt")
	}
}

func TestFiringPosition(t *testing.T) {
	me, tgt := input.Point{X: 100, Y: 100}, input.Point{X: 130, Y: 100}
	valid := func(p input.Point) bool { return p.X < 125 }
	open := func(a, b input.Point) bool { return true }
	p, ok := FiringPosition(me, tgt, 20, valid, open)
	if !ok || p.X >= 125 || pdist(p, tgt) > 20.5 {
		t.Fatalf("firing position %v %v", p, ok)
	}
	if _, ok := FiringPosition(me, tgt, 20, valid, func(a, b input.Point) bool { return false }); ok {
		t.Fatal("a wall everywhere leaves no firing position")
	}
	if _, ok := FiringPosition(me, tgt, 20, func(input.Point) bool { return false }, open); ok {
		t.Fatal("nothing reachable")
	}
}

// A monster across a river: no path to it, but bolts fly over water. The fighter finds a firing position on its
// own side and kills it from range; nothing is reported as ignored.
func TestShootsAcrossRiver(t *testing.T) {
	h := newHarness(t)
	h.s.noPath = func(p input.Point) bool { return p.X >= 122 }
	h.f.sight = func(a, b input.Point) bool { return true }
	h.s.addMon(10, clsFallen, 135, 100, 24, 0, 0)
	res, err := h.f.Clear(context.Background(), ClearOpts{})
	if err != nil || res.Kills != 1 || len(res.Ignored) != 0 {
		t.Fatalf("%v %v", res, err)
	}
}

// Across a wall: no path and no line of sight from anywhere reachable. One bounded assessment, a reason, and the
// next Clear does not even look at it.
func TestUnreachableIsIgnoredOnceAndRemembered(t *testing.T) {
	h := newHarness(t)
	h.s.noPath = func(p input.Point) bool { return p.X >= 122 }
	h.f.sight = func(a, b input.Point) bool { return !(a.X < 122 && b.X >= 122) }
	h.s.addMon(10, clsFallen, 135, 100, 24, 0, 0)
	res, err := h.f.Clear(context.Background(), ClearOpts{})
	if err != nil || res.Kills != 0 || len(res.Ignored) != 1 || res.Ignored[0].Why != IgnoreUnreachable || res.TimedOut {
		t.Fatalf("%v %v %+v", res, err, res.Ignored)
	}
	if res.Elapsed > 5*time.Second {
		t.Fatalf("one assessment, not a loop: %v", res.Elapsed)
	}
	if got := h.s.casts; got != 0 {
		t.Fatalf("no mana wasted: %d casts", got)
	}
	res2, err := h.f.Clear(context.Background(), ClearOpts{})
	if err != nil || res2.Elapsed > time.Second || len(res2.Ignored) != 1 || res2.Ignored[0].Why != IgnoreUnreachable {
		t.Fatalf("second clear %v %v", res2, err)
	}
	if len(h.f.IgnoredTargets()) != 1 {
		t.Fatal("memory")
	}
	h.f.ForgetIgnored()
	if len(h.f.IgnoredTargets()) != 0 {
		t.Fatal("forget")
	}
}

func TestIgnoreReasons(t *testing.T) {
	h := newHarness(t)
	h.s.addMon(10, clsAllImmun, 115, 100, 50, 0, 0)
	h.s.addMon(11, clsFallen, 120, 100, 500, 0, 0).immune = true
	h.f.SetDeathWeights(DeathWeights{Monster: map[string]float64{"archer": 5}})
	h.s.addMon(12, clsArcher, 125, 100, 30, 0, 0)
	res, err := h.f.Clear(context.Background(), ClearOpts{})
	if err != nil {
		t.Fatal(err)
	}
	why := map[uint32]IgnoreReason{}
	for _, i := range res.Ignored {
		why[i.ID] = i.Why
	}
	if why[10] != IgnoreImmune || why[11] != IgnoreNoProgress || why[12] != IgnoreAvoid || len(res.Ignored) != 3 {
		t.Fatalf("%+v", res.Ignored)
	}
}

// An ignored monster that walks up to the character is fought again.
func TestIgnoredMonsterThatComesCloseIsFought(t *testing.T) {
	h := newHarness(t)
	h.s.noPath = func(p input.Point) bool { return p.X >= 122 }
	h.f.sight = func(a, b input.Point) bool { return !(a.X < 122 && b.X >= 122) }
	m := h.s.addMon(10, clsFallen, 135, 100, 24, 3, 0)
	if res, _ := h.f.Clear(context.Background(), ClearOpts{}); len(res.Ignored) != 1 {
		t.Fatalf("%v", res)
	}
	m.x, m.speed = 104, 6 // it crossed over and attacks
	res, err := h.f.Clear(context.Background(), ClearOpts{})
	if err != nil || res.Kills != 1 {
		t.Fatalf("%v %v", res, err)
	}
}

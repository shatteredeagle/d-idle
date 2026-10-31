package bot

import (
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

// plannedTown is a fakeTown that also answers the trip floor's PlannedSpend.
type plannedTown struct {
	*fakeTown
	spend int
}

func (p *plannedTown) PlannedSpend(*d2.Snapshot) int { return p.spend }

// goldfloor: a pure restock trip needs a planned purchase of Town.TripMinSpend gold unless the belt holds no healing
// potion (docs/goldfloor.md finding 2).
func TestTripFloor(t *testing.T) {
	due := func(belt []uint32, spend int, mutate func(*rig)) (bool, ChorePlan) {
		g := newRig(t)
		g.r.d.Town = &plannedTown{fakeTown: g.r.d.Town.(*fakeTown), spend: spend}
		g.w.set(func(w *world) { w.level = 6; w.area = 2; w.gold0 = 100 })
		g.w.belt(belt...)
		if mutate != nil {
			mutate(g)
		}
		s, _ := g.w.snapshot()
		plan, _, ok := g.r.choresDue(s)
		return ok, plan
	}
	if ok, plan := due([]uint32{101}, 30, nil); ok {
		t.Fatalf("1 healing potion and a 30-gold plan raised a trip: %+v", plan)
	}
	if ok, _ := due([]uint32{101}, 90, nil); !ok {
		t.Fatal("a 90-gold plan raised no trip")
	}
	if ok, _ := due(nil, 0, nil); !ok {
		t.Fatal("an empty belt raised no trip")
	}
	if ok, _ := due([]uint32{101}, 30, func(g *rig) { g.r.e.Tune.Town.TripMinSpend = 0 }); !ok {
		t.Fatal("tripMinSpend 0 must restore the old trip")
	}
}

// Scenario (docs/goldfloor.md: 83 percent of NAVSOAK trips bought under 30 gold): a clvl 6 character with 1 healing
// potion and 100 gold in Blood Moor for 120 virtual minutes raises no trip with the floor (the old behaviour raised one
// every poll, each burning a portal scroll) and still raises it the moment the belt is empty.
func TestScenario_TripFloorPoorLoop(t *testing.T) {
	trips := func(floor int) int {
		g := newRig(t)
		g.r.e.Tune.Town.TripMinSpend = floor
		g.r.d.Town = &plannedTown{fakeTown: g.r.d.Town.(*fakeTown), spend: 30}
		g.w.set(func(w *world) { w.level = 6; w.area = 2; w.gold0 = 100 })
		g.w.belt(101)
		n := 0
		for m := 0; m < 120; m++ {
			g.clock.Sleep(time.Minute)
			g.r.poll()
			if _, _, ok := g.r.choresDue(g.r.latest()); ok {
				n++
			}
		}
		return n
	}
	if n := trips(60); n != 0 {
		t.Fatalf("%d trips with the floor, want 0", n)
	}
	if n := trips(0); n == 0 {
		t.Fatal("floor off raised no trip: the scenario cannot detect the old loop")
	}
}

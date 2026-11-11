package nav

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// follower keeps a monster two subtiles behind the hero, like a pack that runs after it.
func follower(s *sim) {
	p := s.pos()
	s.units = []d2.Unit{{ID: 9, Type: d2.TypeMonster, X: p.X - 3, Y: p.Y, Mode: d2.MonModeStand}}
}

// The nav layer is class free (no class data enters a walk), so the cases vary the facts the rule reads: the
// pursuit knobs and the life share of the hero, which is what separates a full-life melee class from a hurt caster.
func TestWanderTurnsToFightWhenFollowed(t *testing.T) {
	for _, tc := range []struct {
		name    string
		life    int
		radius  float64
		pursuit bool
		yields  int
	}{
		{"full life", 100, 10, true, 1},
		{"hurt, below minimum life", 30, 10, true, 0},
		{"rule off by knob", 100, 0, true, 0},
		{"not a wander walk", 100, 10, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newSim(openField(1000, 1000, 200, 60), input.Point{X: 1010, Y: 1030})
			s.life = tc.life
			s.onStep = func(s *sim) {
				if len(s.units) > 0 {
					follower(s)
				}
			}
			follower(s)
			n := s.navigator()
			n.e.Tune.Nav.PursuitRadius = tc.radius
			yields := 0
			n.Yield = func(ctx context.Context, near []d2.Unit) error {
				yields++
				s.units = nil // the fight killed it
				return nil
			}
			err := n.walk(ctxT(t), func(*obs) (input.Point, float64, error) { return input.Point{X: 1150, Y: 1030}, 3, nil },
				walkOpts{pursuit: tc.pursuit})
			if err != nil {
				t.Fatal(err)
			}
			if yields != tc.yields {
				t.Errorf("yields %d, want %d", yields, tc.yields)
			}
		})
	}
}

// A pack 13 subtiles behind the hero is outside the default radius (10) and inside a raised one (16): the knob decides
// how far the hero looks while it runs (sandbox tuning 2026-11-11). Class free like the rule above.
func TestPursuitRadiusKnobWidensTheLookout(t *testing.T) {
	for _, tc := range []struct {
		radius float64
		yields int
	}{{10, 0}, {16, 1}} {
		s := newSim(openField(1000, 1000, 200, 60), input.Point{X: 1010, Y: 1030})
		behind := func(s *sim) {
			p := s.pos()
			s.units = []d2.Unit{{ID: 9, Type: d2.TypeMonster, X: p.X - 13, Y: p.Y, Mode: d2.MonModeStand}}
		}
		s.onStep = func(s *sim) {
			if len(s.units) > 0 {
				behind(s)
			}
		}
		behind(s)
		n := s.navigator()
		n.e.Tune.Nav.PursuitRadius = tc.radius
		yields := 0
		n.Yield = func(ctx context.Context, near []d2.Unit) error { yields++; s.units = nil; return nil }
		if err := n.walk(ctxT(t), func(*obs) (input.Point, float64, error) { return input.Point{X: 1150, Y: 1030}, 3, nil }, walkOpts{pursuit: true}); err != nil {
			t.Fatal(err)
		}
		if yields != tc.yields {
			t.Errorf("radius %v: yields %d, want %d", tc.radius, yields, tc.yields)
		}
	}
}

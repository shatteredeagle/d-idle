package bot

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
)

// goldLoot is a fakeLoot that knows gold piles (loot.Looter through the adapter in production).
type goldLoot struct {
	fakeLoot
	near       bool
	picked     int
	opts       []PickOpts
	walkBacks  int
	ledgerSeen int
}

func (l *goldLoot) PickUp(ctx context.Context, o PickOpts) (PickReport, error) {
	l.w.call("pickup")
	l.opts = append(l.opts, o)
	if l.near && l.picked < 100 {
		l.near = false
		l.picked++
		return PickReport{Picked: 1, Gold: 30}, nil
	}
	return PickReport{}, nil
}
func (l *goldLoot) GoldNear(s *d2.Snapshot, dist int) bool { return l.near }
func (l *goldLoot) GoldLedger() (int, int, int, int)       { return l.ledgerSeen, l.picked, 30 * l.picked, 0 }
func (l *goldLoot) ObserveGround(*d2.Snapshot)             {}
func (l *goldLoot) ResetMemory()                           {}
func (l *goldLoot) WalkBack(context.Context, func() bool, func(n, gold int, first string)) (int, error) {
	l.walkBacks++
	l.w.call("walkback")
	return 0, nil
}

func goldRig(t *testing.T, class d2.Class) (*rig, *goldLoot) {
	gl := &goldLoot{}
	g := newRig(t, func(_ *Config, d *Deps, _ *env.Tuning) { gl.fakeLoot = fakeLoot{}; d.Loot = gl })
	gl.fakeLoot.w = g.w
	g.w.set(func(w *world) { w.class = class; w.area = 2 })
	g.r.poll()
	return g, gl
}

// The mid-fight grab: takes a pile when healthy, no hostile close and the pack is calm; a hostile close, a dangerous
// pack, low life, or priority off all keep the hero fighting (survival and danger come first).
func TestGoldGrabPolicy(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		for _, c := range []struct {
			name    string
			life    int
			nearest float64
			danger  int
			prio    int
			near    bool
			want    bool
		}{
			{"calm pile", 100, 30, 0, 1, true, true},
			{"hostile close defers", 100, 3, 0, 1, true, false},
			{"dangerous pack defers", 100, 30, 2, 1, true, false},
			{"low life ignores gold", 40, 30, 0, 1, true, false},
			{"priority off", 100, 30, 0, 0, true, false},
			{"no pile", 100, 30, 0, 1, false, false},
		} {
			t.Run(class.String()+"/"+c.name, func(t *testing.T) {
				g, gl := goldRig(t, class)
				g.w.set(func(w *world) { w.life = c.life })
				g.e.Tune.Loot.GoldPriority = c.prio
				gl.near = c.near
				s := g.r.poll()
				got := g.r.goldGrab(context.Background(), s, c.nearest, c.danger)
				if got != c.want {
					t.Fatalf("grab = %v, want %v", got, c.want)
				}
				if c.want && (len(gl.opts) != 1 || !gl.opts[0].GoldOnly || gl.opts[0].Max != 1 || gl.opts[0].Abort == nil) {
					t.Fatalf("pickup opts %+v", gl.opts)
				}
				if !c.want && len(gl.opts) != 0 {
					t.Fatal("picked up anyway")
				}
			})
		}
	}
}

// A grab that takes nothing (aborted or unreachable) must not make the fight loop spin: the interlude pauses itself.
func TestGoldGrabDoesNotSpin(t *testing.T) {
	g, gl := goldRig(t, d2.Paladin)
	gl.picked = 100 // the fake now picks nothing
	gl.near = true
	s := g.r.poll()
	if !g.r.goldGrab(context.Background(), s, 30, 0) {
		t.Fatal("first attempt should act")
	}
	if g.r.goldGrab(context.Background(), s, 30, 0) {
		t.Fatal("second attempt right after an empty one must wait")
	}
}

// A roam leg ends when a pile shows up (the sweep is called again and picks it before roaming on), and a leave
// (explore gave up, area cleared) fetches remembered piles first.
func TestSweepStopsRoamForGoldAndFetchesBeforeLeaving(t *testing.T) {
	g, gl := goldRig(t, d2.Paladin)
	r := g.r
	r.loop.cleared = map[int]bool{}
	var doneCalled bool
	g.mover.exploreFn = func(ctx context.Context, done func() bool) error {
		gl.near = true // a pile came into view during the leg
		doneCalled = done()
		return nil
	}
	if err := r.sweepPass(context.Background(), 2); err != nil {
		t.Fatal(err)
	}
	if !doneCalled {
		t.Fatal("the roam did not stop for a pile in reach")
	}
	if r.loop.cleared[2] {
		t.Fatal("a roam stopped for gold must not count as the area explored")
	}
	// roam finished normally: the area is cleared, and the walk-back ran just before that
	gl.near, gl.walkBacks = false, 0
	g.mover.exploreFn = nil
	if err := r.sweepPass(context.Background(), 2); err != nil {
		t.Fatal(err)
	}
	if gl.walkBacks < 2 || !r.loop.cleared[2] {
		t.Fatalf("walk-backs %d cleared %v", gl.walkBacks, r.loop.cleared)
	}
}

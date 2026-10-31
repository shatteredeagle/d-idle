package bot

import (
	"testing"
	"time"
)

// Leaving an emptied area (KILLFIX): off by default, on after EmptyAreaSec without a kill, restarted by a kill, by
// a pause longer than 90 s and by another area; a quest area is never left this way.
func TestEmptyAreaLeave(t *testing.T) {
	g := newRig(t)
	r := g.r
	r.loop.cleared = map[int]bool{}
	if r.emptyAreaLeave(2) || r.loop.cleared[2] {
		t.Fatal("off by default")
	}
	r.cfg.EmptyAreaSec = 100
	step := func(d time.Duration) bool { g.clock.Sleep(d); return r.emptyAreaLeave(2) }
	if step(0) || step(50*time.Second) {
		t.Fatal("too early")
	}
	r.obMu.Lock()
	r.ob.kills++
	r.obMu.Unlock()
	if step(40*time.Second) || step(50*time.Second) || step(40*time.Second) { // kill restarted the count: 90 s since
		t.Fatal("a kill must restart the count")
	}
	if step(time.Hour) { // a long pause (town trip, new game) restarts it too
		t.Fatal("a pause must restart the count")
	}
	if step(50*time.Second) || !step(55*time.Second) || !r.loop.cleared[2] {
		t.Fatalf("105 s without a kill must clear the area: %v", r.loop.cleared)
	}
	r.loop.cleared = map[int]bool{}
	r.loop.emptySince = time.Time{}
	run := LevelRun{Area: 2}
	r.loop.run = &run
	step(0)
	if step(200 * time.Second) {
		t.Fatal("a quest area is not left")
	}
}

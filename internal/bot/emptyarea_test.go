package bot

import (
	"context"
	"errors"
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

// A roam that keeps failing in a quiet area (the g1 soak: "explore: nav: timeout: roaming for 5m0s" in Cold Plains and
// Blood Moor in turn, nothing ever marked cleared) ends the area after exploreGiveUp failures, counted per area so a
// visit to another area in between does not restart them; a kill does, a quest area never gives up.
func TestExploreGaveUp(t *testing.T) {
	g := newRig(t)
	r := g.r
	ctx := context.Background()
	fail := errors.New("nav: timeout: roaming")
	for _, c := range []struct {
		name  string
		steps []int // areas of successive failures
		kill  bool  // a kill after the first failure
		want  bool  // area 2 cleared at the end
	}{
		{"two failures", []int{2, 2}, false, true},
		{"ping-pong between areas", []int{2, 3, 2, 3}, false, true},
		{"a kill restarts", []int{2, 2}, true, false},
		{"one failure", []int{2}, false, false},
	} {
		r.loop.cleared = map[int]bool{}
		r.loop.exploreFails = nil
		for i, a := range c.steps {
			if i == 1 && c.kill {
				r.obMu.Lock()
				r.ob.kills++
				r.ob.exp += 5000 // a real kill: far above the floor
				r.obMu.Unlock()
			}
			r.exploreGaveUp(ctx, a, fail)
		}
		if r.loop.cleared[2] != c.want {
			t.Errorf("%s: cleared[2] = %v, want %v", c.name, r.loop.cleared[2], c.want)
		}
	}
	r.loop.cleared, r.loop.exploreFails = map[int]bool{}, nil
	for _, k := range []RunKind{RunQuest, RunBoss} {
		r.loop.cleared, r.loop.exploreFails = map[int]bool{}, nil
		r.loop.run = &LevelRun{Area: 2, Kind: k}
		r.exploreGaveUp(ctx, 2, fail)
		if r.exploreGaveUp(ctx, 2, fail) || r.loop.cleared[2] {
			t.Fatalf("a %s run keeps roaming", k)
		}
	}
	// the planner sets a clear run for every leveling step: it must still give up (Gate 1: 20 of 22 timeouts)
	r.loop.cleared, r.loop.exploreFails = map[int]bool{}, nil
	r.loop.run = &LevelRun{Area: 2, Kind: RunClear}
	r.exploreGaveUp(ctx, 2, fail)
	if !r.exploreGaveUp(ctx, 2, fail) || !r.loop.cleared[2] {
		t.Fatal("a clear run in the failing area gives up after two failures")
	}
}

// The 17:27 window of Gate 1: Blood Moor and Cold Plains alternating about every 35 s wall (7 virtual minutes a cycle),
// no kill, no explore timeout. Both become cleared; a kill, a third area or a quest run prevents it.
func TestPingPongGuard(t *testing.T) {
	for _, c := range []struct {
		name  string
		areas []int
		kill  int // index before which a kill happens, -1 none
		run   *LevelRun
		step  time.Duration
		want  bool
	}{
		{"replay", []int{2, 3, 2, 3, 2, 3, 2, 3}, -1, &LevelRun{Area: 3, Kind: RunClear}, 150 * time.Second, true},
		{"a kill restarts", []int{2, 3, 2, 3, 2, 3}, 3, nil, 150 * time.Second, false},
		{"third area", []int{2, 3, 2, 3, 4, 3, 2, 3}, -1, nil, 150 * time.Second, false},
		{"too fast", []int{2, 3, 2, 3, 2}, -1, nil, 10 * time.Second, false},
		{"quest run", []int{2, 3, 2, 3, 2, 3, 2, 3}, -1, &LevelRun{Area: 3, Kind: RunQuest}, 150 * time.Second, false},
	} {
		g := newRig(t)
		r := g.r
		r.loop.cleared = map[int]bool{}
		r.loop.run = c.run
		r.obMu.Lock()
		r.ob.level, r.ob.exp = 14, 1000
		r.obMu.Unlock()
		for i, a := range c.areas {
			if i == c.kill {
				r.obMu.Lock()
				r.ob.kills++
				r.ob.exp += 5000 // a real kill: far above the floor
				r.obMu.Unlock()
			}
			g.clock.Sleep(c.step)
			r.noteArea(a)
		}
		if got := r.loop.cleared[2] && r.loop.cleared[3]; got != c.want {
			t.Errorf("%s: both cleared = %v, want %v", c.name, got, c.want)
		}
	}
}

// Cluster C of gate1r2-20261101: 8 switches over 52 virtual minutes, 26 kills worth 18 XP, clvl 14. A farm that earns
// XP at a normal pace must not fire.
func TestPingPongRate(t *testing.T) {
	for _, c := range []struct {
		name   string
		xpKill uint32
		want   bool
	}{
		{"cluster C", 0, true}, // 26 kills, 18 XP in total: 18 / 26 per kill rounds to 0 or 1
		{"farm pace", 60, false},
	} {
		g := newRig(t)
		r := g.r
		r.loop.cleared = map[int]bool{}
		r.obMu.Lock()
		r.ob.level, r.ob.exp = 14, 100000
		r.obMu.Unlock()
		areas := []int{2, 3, 2, 3, 2, 3, 2, 3, 2}
		kill := 0
		for _, a := range areas {
			g.clock.Sleep(6*time.Minute + 30*time.Second) // 8 switches over 52 minutes
			for k := 0; k < 3 && kill < 26; k++ {         // about 3 kills per pass, 26 in all
				kill++
				r.obMu.Lock()
				r.ob.kills++
				r.ob.exp += c.xpKill
				if c.xpKill == 0 && kill <= 18 {
					r.ob.exp++ // 18 XP over the 26 kills
				}
				r.obMu.Unlock()
			}
			r.noteArea(a)
		}
		if got := r.loop.cleared[2] && r.loop.cleared[3]; got != c.want {
			t.Errorf("%s: both cleared = %v, want %v", c.name, got, c.want)
		}
	}
}

// bot.clearIdleMin (docs/hourfix.md): a leveling slice calls an area cleared after that many idle minutes, other
// sweeps and the value 0 are left alone. Class independent (no class or area data involved).
func TestClearIdleMinLeavesALevelingField(t *testing.T) {
	g := newRig(t)
	r := g.r
	r.loop.cleared = map[int]bool{}
	run := func(levelSlice bool, min int) bool {
		r.e.Tune.Bot.ClearIdleMin = min
		r.loop.levelSlice, r.loop.cleared, r.loop.emptySince = levelSlice, map[int]bool{}, time.Time{}
		r.emptyAreaLeave(2)
		g.clock.Sleep(80 * time.Second)
		r.emptyAreaLeave(2)
		g.clock.Sleep(80 * time.Second)
		r.emptyAreaLeave(2)
		g.clock.Sleep(30 * time.Second) // 190 s idle in all, no gap over 90 s
		return r.emptyAreaLeave(2)
	}
	if !run(true, 3) {
		t.Fatal("190 s idle in a leveling slice with clearIdleMin 3 must clear the area")
	}
	if run(true, 0) {
		t.Fatal("clearIdleMin 0 is off")
	}
	if run(false, 3) {
		t.Fatal("a non-leveling sweep keeps roaming")
	}
}

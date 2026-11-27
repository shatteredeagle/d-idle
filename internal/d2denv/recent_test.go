package d2denv

import (
	"context"
	"d2lod-bot/internal/d2dctl"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dsrc"
)

func TestReaderRecent(t *testing.T) {
	r := &Reader{}
	if _, ok := r.Recent(time.Second); ok {
		t.Fatal("no snapshot read yet: Recent must say no")
	}
	snap := &d2.Snapshot{State: d2.StateInGame}
	r.last, r.lastAt = &d2dsrc.Result{Snap: snap}, time.Now()
	if got, ok := r.Recent(time.Second); !ok || got != snap {
		t.Fatalf("a snapshot just read must be returned: %v %v", got, ok)
	}
	r.lastAt = time.Now().Add(-2 * time.Second)
	if _, ok := r.Recent(time.Second); ok {
		t.Fatal("a snapshot older than maxAge must not be returned")
	}
}

// Recent measures a snapshot's age on the Reader's clock (card pgclock): a guard poll reuses the last snapshot or queues a
// fresh read by VIRTUAL age, so it does not depend on how long the machine took in wall time. Before, Recent used the wall
// clock and a loaded builder re-read where a quiet one reused, which moved the command stream (journal pair, tick 2242).
func TestReaderRecentUsesTheInjectedClock(t *testing.T) {
	for _, cls := range []d2.Class{d2.Sorceress, d2.Paladin} {
		virt := time.Unix(0, 0)
		r := &Reader{Now: func() time.Time { return virt }}
		snap := &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{Class: cls}}
		r.last, r.lastAt = &d2dsrc.Result{Snap: snap}, virt
		time.Sleep(20 * time.Millisecond) // the wall moves, the env clock does not
		if got, ok := r.Recent(10 * time.Millisecond); !ok || got != snap {
			t.Fatalf("class %v: wall time aged a snapshot the env clock calls fresh", cls)
		}
		virt = virt.Add(350 * time.Millisecond)
		if _, ok := r.Recent(300 * time.Millisecond); ok {
			t.Fatalf("class %v: a snapshot older than maxAge on the env clock was reused", cls)
		}
	}
}

// B3 (card pgclock): the soak stop is a function of the stepped ticks. The callback runs on the stepping goroutine,
// once, after the first sleep that reaches the deadline, whichever door the sleep came through; two runs that step the
// same ticks stop at the same command. The old ticker goroutine cancelled on a wall second, wherever the bot was.
func TestStopAfterFiresOnTheSteppingGoroutine(t *testing.T) {
	srv, p := startFake(t)
	_ = srv
	c := d2dctl.New(p, d2dctl.Options{})
	t.Cleanup(func() { c.Close() })
	clk := NewLateClock(context.Background(), c)
	if clk.StopAfter(time.Second, func() {}) {
		t.Fatal("a wall clock must refuse: the caller keeps its timer")
	}
	if !clk.Install(context.Background()) {
		t.Fatal("fake d2d is not lockstep")
	}
	for name, door := range map[string]func(){
		"Sleep": func() { clk.Sleep(400 * time.Millisecond) },
		"Pause": func() { _ = clk.Pause(context.Background(), 400*time.Millisecond) },
		"Poll":  func() { clk.Poll(400 * time.Millisecond) }, // steps 80 ms per call
	} {
		fired := 0
		if !clk.StopAfter(time.Second, func() { fired++ }) {
			t.Fatal("lockstep clock refused StopAfter")
		}
		n := 0
		for fired == 0 && n < 100 {
			door()
			n++
		}
		if fired != 1 {
			t.Fatalf("%s: fired %d after %d sleeps", name, fired, n)
		}
		want := map[string]int{"Sleep": 3, "Pause": 3, "Poll": 13}[name] // first sleep whose end is >= 1 s
		if n != want {
			t.Errorf("%s: stopped after %d sleeps, want %d", name, n, want)
		}
		door()
		if fired != 1 {
			t.Fatalf("%s: fired again", name)
		}
	}
}

// OnSleep (card pgtaps): the hook runs on the stepping goroutine after each lockstep sleep, is not re-entered by a sleep it
// makes, and is refused on a wall clock.
func TestOnSleepRunsAfterEachLockstepSleep(t *testing.T) {
	_, p := startFake(t)
	c := d2dctl.New(p, d2dctl.Options{})
	t.Cleanup(func() { c.Close() })
	clk := NewLateClock(context.Background(), c)
	if clk.OnSleep(func() {}) {
		t.Fatal("a wall clock must refuse")
	}
	if !clk.Install(context.Background()) {
		t.Fatal("fake d2d is not lockstep")
	}
	n, nested := 0, 0
	clk.OnSleep(func() {
		n++
		if n == 1 {
			nested++
			clk.Sleep(80 * time.Millisecond) // steps, must not re-enter
		}
	})
	clk.Sleep(80 * time.Millisecond)
	_ = clk.Pause(context.Background(), 80*time.Millisecond)
	clk.Poll(80 * time.Millisecond)
	if n != 3 || nested != 1 {
		t.Fatalf("hook ran %d times (nested %d), want 3 and 1", n, nested)
	}
}

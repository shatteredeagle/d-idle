package bot

import (
	"testing"
	"time"
)

func TestKillClockBuckets(t *testing.T) {
	var k killClock
	t0 := time.Unix(1000, 0)
	k.Tick(t0, 2, 1, 0, 0)                    // walking from t0
	k.Tick(t0.Add(3*time.Second), 8, 1, 5, 0) // 3 s walk, then cast
	k.Tick(t0.Add(4*time.Second), 1, 1, 5, 0) // 1 s cast, then idle
	k.Tick(t0.Add(10*time.Second), 1, 1, 5, 0)
	s := k.Kill(t0.Add(10 * time.Second))
	if s.Walk != 3*time.Second || s.Cast != time.Second || s.Wait != 6*time.Second || s.Total != 10*time.Second || s.Gap != 10*time.Second {
		t.Fatalf("split %+v", s)
	}
	// out of game time is not charged, the gap still shows it
	k.Tick(t0.Add(11*time.Second), 1, 1, 5, 0)
	k.Gap()
	k.Tick(t0.Add(41*time.Second), 1, 1, 5, 0)
	k.Tick(t0.Add(42*time.Second), 1, 1, 5, 0)
	s = k.Kill(t0.Add(42 * time.Second))
	if s.Total != 2*time.Second || s.Gap != 32*time.Second {
		t.Fatalf("gap split %+v", s)
	}
	if s2 := k.Kill(t0.Add(42 * time.Second)); s2.Total != 0 || s2.Gap != 0 {
		t.Fatalf("second kill same tick %+v", s2)
	}
}

func TestKillClockDisplacementIsWalk(t *testing.T) {
	var k killClock
	t0 := time.Unix(1000, 0)
	k.Tick(t0, 1, 1, 0, 0)
	k.Tick(t0.Add(4*time.Second), 1, 1, 20, 0) // mode idle but moved: walk (blocking MoveTo)
	k.Tick(t0.Add(6*time.Second), 1, 2, 90, 0) // area change: not a walk
	s := k.Kill(t0.Add(6 * time.Second))
	if s.Walk != 4*time.Second || s.Wait != 2*time.Second {
		t.Fatalf("%+v", s)
	}
}

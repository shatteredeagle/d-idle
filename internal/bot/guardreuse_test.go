package bot

import (
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2/fake"
)

// recentRd is a reader with a Recent snapshot (the d2d reader's shape).
type recentRd struct {
	fake.Reader
	recent *d2.Snapshot
}

func (r *recentRd) Recent(time.Duration) (*d2.Snapshot, bool) { return r.recent, r.recent != nil }

// WALKSMOOTH2: the safety guard takes the snapshot the walker just read instead of queueing another full read; a
// plain poll (the main loop, "act then look") always reads fresh.
func TestGuardPollReusesRecentSnapshot(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		g := newRig(t)
		reads := 0
		g.r.d.Snapshot = func() (*d2.Snapshot, error) {
			reads++
			return &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{Class: class, Life: 10, MaxLife: 10}}, nil
		}
		recent := &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{Class: class, Life: 7, MaxLife: 10}}
		g.r.e.Read = &recentRd{recent: recent}

		if s := g.r.pollWith(true); s == nil || s.Player.Life != 7 || reads != 0 {
			t.Fatalf("%v: guard poll must use the recent snapshot (reads %d, snap %+v)", class, reads, s)
		}
		if s := g.r.poll(); s == nil || s.Player.Life != 10 || reads != 1 {
			t.Fatalf("%v: a plain poll must read fresh (reads %d)", class, reads)
		}
		g.r.e.Read.(*recentRd).recent = nil
		if g.r.pollWith(true); reads != 2 {
			t.Fatalf("%v: no recent snapshot: the guard reads fresh (reads %d)", class, reads)
		}
	}
}

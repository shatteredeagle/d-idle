package nav

import (
	"testing"

	"d2lod-bot/internal/d2"
)

// A stale save loads a standing character with life 0 (VERIFIED LIVE, i4 Booti); it must still be able to walk to a healer.
func TestDeadIsModeOnly(t *testing.T) {
	for _, c := range []struct {
		mode, life int
		want       bool
	}{{0, 0, true}, {17, 0, true}, {5, 0, false}, {1, 0, false}, {5, 40, false}} {
		o := &obs{snap: &d2.Snapshot{Player: &d2.Player{Mode: c.mode, Life: c.life, MaxLife: 40}}}
		if got := dead(o); got != c.want {
			t.Errorf("mode %d life %d: dead = %v, want %v", c.mode, c.life, got, c.want)
		}
	}
}

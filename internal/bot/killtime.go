package bot

import (
	"math"
	"time"
)

// killClock splits the time between two kills into three buckets, so the same log line (`killtime`) tells where the
// seconds per kill go on any backend (docs/killgap.md). It is fed once per observed in-game snapshot with the player's
// unit mode of that snapshot; the interval since the previous snapshot is charged to the PREVIOUS snapshot's bucket
// (the mode is what the character was doing during the interval).
//
//	walk: player mode walk or run (2, 3, 6), or the character moved at least 1 unit in the same area since the previous
//	      snapshot = time spent moving (nav, approach, loot walks, town walks). The displacement rule is needed because
//	      nav's blocking MoveTo loops do not feed observe() (V on d2d: the mode alone gave walk=0 in 207 kills, and a 2-unit threshold too because ticks are short).
//	cast: player mode attack or cast (7, 8, 10, 11, 13..16) = time inside the cast/attack animation windows.
//	wait: everything else = standing idle, retarget search, panels, potions, being hit, town, link waits.
//
// total = walk+cast+wait of the in-game time since the previous kill (menu/out-of-game time is not counted);
// gap = time on the runner clock since the previous kill event, so gap - total is time out of game.
// The mode numbers are the Diablo II player modes (docs/d2-1.14d-memory.md); UNVERIFIED that the d2d backend fills
// the mode the same way, which the measurement run checks (walk=0 everywhere would show it).
type killClock struct {
	last             time.Time // previous snapshot time, zero before the first or after a gap
	lastKill         time.Time
	prevMode         int
	prevArea, px, py int
	walk, cast, wait time.Duration
}

type killSplit struct {
	Total, Walk, Cast, Wait, Gap time.Duration
}

func modeBucket(mode int) int {
	switch mode {
	case 2, 3, 6:
		return 0
	case 7, 8, 10, 11, 13, 14, 15, 16:
		return 1
	}
	return 2
}

// Tick charges the time since the previous tick; mode is the mode of this snapshot.
func (k *killClock) Tick(now time.Time, mode, area, x, y int) {
	if !k.last.IsZero() && now.After(k.last) {
		d := now.Sub(k.last)
		b := modeBucket(k.prevMode)
		if b == 2 && area == k.prevArea && abs(x-k.px)+abs(y-k.py) >= 1 {
			b = 0
		}
		switch b {
		case 0:
			k.walk += d
		case 1:
			k.cast += d
		default:
			k.wait += d
		}
	}
	k.last, k.prevMode, k.prevArea, k.px, k.py = now, mode, area, x, y
	if k.lastKill.IsZero() {
		k.lastKill = now
	}
}

// Gap marks time out of game (menu, restart): the interval is not charged to any bucket.
func (k *killClock) Gap() { k.last = time.Time{} }

// Kill returns the split since the previous kill and resets the buckets. Further kills in the same tick get zeros.
func (k *killClock) Kill(now time.Time) killSplit {
	s := killSplit{Walk: k.walk, Cast: k.cast, Wait: k.wait, Gap: now.Sub(k.lastKill)}
	s.Total = k.walk + k.cast + k.wait
	k.walk, k.cast, k.wait = 0, 0, 0
	k.lastKill = now
	return s
}

func secs(d time.Duration) float64 { return math.Round(d.Seconds()*10) / 10 }

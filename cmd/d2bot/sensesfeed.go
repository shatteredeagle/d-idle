package main

import (
	"log/slog"
	"sync"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/senses"
	"d2lod-bot/internal/status"
)

// sensesEvery is how often the hero position goes to the publisher; an area change or a map that grew goes at once.
const sensesEvery = 500 * time.Millisecond

// useSenses feeds the side-panel minimap (docs/senses-map.md). nav calls Env.OnObserve from the goroutine that just ran
// World.Update, so the Tracker is read only there; the Publisher copies what it needs into an immutable snapshot and the
// HTTP side sees only that. Call it after startLadder; no rig or no Publisher (senses.tier 0): nothing.
func useSenses(rig *ladderRig, e *env.Env, log *slog.Logger) {
	if rig == nil || rig.Senses == nil || e == nil {
		return
	}
	var us *unitSource
	if e.Read != nil {
		us = &unitSource{Read: e.Read, Cat: combat.NewCatalog(e.Data)}
	}
	e.OnObserve = sensesFeedUnits(rig.Senses, time.Now, log, us)
}

// sensesFeed builds the OnObserve hook. The hero is State.Pos, the position d2map placed in the same grid (level-relative
// on d2d, absolute on Game.exe), so no Origin is applied (docs/senses-map.md "Coordinates").
func sensesFeed(pub *senses.Publisher, now func() time.Time, log *slog.Logger) func(*d2map.LevelMap, d2map.Point) {
	return sensesFeedUnits(pub, now, log, nil)
}

// sensesFeedUnits is sensesFeed plus the unit list (docs/senses-units.md): after each published observation, at the same
// throttle (about 2 Hz), the units go to the publisher. us may be nil (map only).
func sensesFeedUnits(pub *senses.Publisher, now func() time.Time, log *slog.Logger, us *unitSource) func(*d2map.LevelMap, d2map.Point) {
	var (
		mu     sync.Mutex // OnObserve is one goroutine today; the lock keeps a second caller harmless
		last   time.Time
		lastNo = -1
		warned = map[int]bool{}
		kn     int
	)
	return func(lm *d2map.LevelMap, pos d2map.Point) {
		if lm == nil || lm.Grid == nil {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		t, known := now(), lm.Grid.KnownCount()
		if lm.No == lastNo && known == kn && t.Sub(last) < sensesEvery {
			return
		}
		last, lastNo, kn = t, lm.No, known
		if err := pub.Observe(lm, status.AreaName(lm.No), pos); err != nil && !warned[lm.No] {
			warned[lm.No] = true
			log.Debug("senses: no map yet", "err", err)
		}
		if us != nil {
			pub.SetUnits(us.Units(), senses.Point{X: pos.X, Y: pos.Y})
		}
	}
}

package main

import (
	"log/slog"
	"sync"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/nav"
	"d2lod-bot/internal/senses"
	"d2lod-bot/internal/status"
)

// sensesEvery is how often the hero position goes to the publisher; an area change or a map that grew goes at once.
const sensesEvery = 500 * time.Millisecond

// useSenses feeds the side-panel minimap (docs/senses-map.md). nav calls Env.OnObserve from the goroutine that just ran
// World.Update, so the Tracker is read only there; the Publisher copies what it needs into an immutable snapshot and the
// HTTP side sees only that. Call it after startLadder; no rig or no Publisher (senses.tier 0): nothing.
func useSenses(rig *ladderRig, e *env.Env, f *combat.Fighter, log *slog.Logger) {
	if rig == nil || rig.Senses == nil || e == nil {
		return
	}
	var us *unitSource
	if e.Read != nil {
		us = &unitSource{Read: e.Read, Cat: combat.NewCatalog(e.Data), ModName: decide.ModName}
		if f != nil {
			us.Attacking = f.Attacking
		}
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

// coverageView is what the navigator offers the minimap (nav.Navigator.CoverageView).
type coverageView interface {
	CoverageView() (nav.CoverageView, bool)
}

// useSensesCoverage adds the roam coverage grid to the senses feed (docs/senses-map.md "Coverage"). It chains onto the
// OnObserve hook useSenses set, so it runs on the navigator's goroutine, the only one that may read the roam state.
// mover that is not a *nav.Navigator (stubs) leaves the feed as it was.
func useSensesCoverage(rig *ladderRig, e *env.Env, mover any) {
	nv, ok := mover.(coverageView)
	if rig == nil || rig.Senses == nil || e == nil || e.OnObserve == nil || !ok {
		return
	}
	prev, pub := e.OnObserve, rig.Senses
	e.OnObserve = func(lm *d2map.LevelMap, pos d2map.Point) {
		prev(lm, pos)
		pub.SetCoverage(coverageFor(nv))
	}
	if h, ok := mover.(interface{ SetOnCoverChange(func()) }); ok {
		h.SetOnCoverChange(func() { pub.SetCoverage(coverageFor(nv)) }) // detour positions flip cells at once
	}
}

func coverageFor(nv coverageView) senses.Coverage {
	v, ok := nv.CoverageView()
	if !ok {
		return senses.Coverage{}
	}
	c := senses.Coverage{Area: v.Level, Cell: v.Spacing, Cover: v.Cover, Total: len(v.Pts), Explored: v.Explored, Legs: v.Legs, Held: v.Held}
	for _, p := range v.Pts {
		st := senses.CoverUnseen
		switch p.State {
		case nav.CoverSeen:
			st = senses.CoverSeen
		case nav.CoverFailed:
			st = senses.CoverFailed
		}
		c.Cells = append(c.Cells, [3]int{p.X, p.Y, st})
	}
	if v.Target != nil {
		c.Target = &senses.Point{X: v.Target.X, Y: v.Target.Y}
	}
	return c
}

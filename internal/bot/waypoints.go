package bot

import (
	"context"
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
)

// Waypoint bookkeeping and the opportunistic activation (NAVSTALL). A waypoint only works after the character touched
// it; the save reader knows the state of earlier sessions, the runner remembers what it touched itself.

// WaypointToucher is the optional Mover capability behind the opportunistic activation (nav.ActivateWaypointNear).
type WaypointToucher interface {
	ActivateWaypointNear(ctx context.Context, maxDist float64) (bool, error)
}

const (
	wpTouchMaxDist = 70               // subtiles: only a waypoint this close is worth a detour
	wpTouchGap     = 20 * time.Second // between two looks at the same area
	wpFailedFor    = 10 * time.Minute // a waypoint trip that failed is not repeated for this long
	wpTouchTries   = 3                // failed touches per area and difficulty before the bot stops trying
)

type wpKey struct{ diff, area int }

// wpState is loopState's waypoint memory; it survives new games (activation belongs to the character).
type wpState struct {
	touched map[wpKey]bool      // touched by this runner
	failed  map[wpKey]time.Time // a waypoint trip to this area failed: not before this time
	tries   map[wpKey]int       // failed opportunistic touches
	next    map[wpKey]time.Time // earliest next look
}

func (w *wpState) init() {
	if w.touched == nil {
		w.touched, w.failed, w.tries, w.next = map[wpKey]bool{}, map[wpKey]time.Time{}, map[wpKey]int{}, map[wpKey]time.Time{}
	}
}

func diffOf(s *d2.Snapshot) int {
	if s == nil || s.Player == nil {
		return 0
	}
	return int(s.Player.Difficulty)
}

// activeWaypoints returns the areas whose waypoint is known active and whether the bot has a reader for that at all.
func (r *Runner) activeWaypoints(s *d2.Snapshot) (map[int]bool, bool) {
	out := map[int]bool{}
	known := false
	if td := r.d.Task; td != nil && td.Quests != nil {
		if v, ok := td.Quests.Read(diffOf(s)); ok {
			known = true
			for a, on := range v.WPs {
				if on {
					out[a] = true
				}
			}
		}
	}
	r.syncTaskGame()
	for a, on := range r.loop.task.wp {
		if on {
			out[a] = true
		}
	}
	r.loop.wp.init()
	for k := range r.loop.wp.touched {
		if k.diff == diffOf(s) {
			out[k.area] = true
		}
	}
	return out, known
}

func (r *Runner) markTouched(s *d2.Snapshot, areas ...int) {
	r.loop.wp.init()
	for _, a := range areas {
		r.loop.wp.touched[wpKey{diffOf(s), a}] = true
		delete(r.loop.wp.failed, wpKey{diffOf(s), a})
	}
}

// wpTripWorthTrying: false when a waypoint trip to area would only burn a panel round trip (about 20 s: the click on
// a never touched waypoint does nothing, V live i4 soak 2026-10-29) because it is known to be inactive or failed lately.
func (r *Runner) wpTripWorthTrying(s *d2.Snapshot, area int, active map[int]bool, known bool) bool {
	r.loop.wp.init()
	if r.e.Clock.Now().Before(r.loop.wp.failed[wpKey{diffOf(s), area}]) {
		return false
	}
	return !known || active[area] || r.d.Plan.IsTown(area)
}

// touchWaypointHere activates the waypoint of the area the character stands in when it is close, nothing hostile is
// near and the waypoint is not known active. Cheap when there is nothing to do: one map lookup and a rate limit.
func (r *Runner) touchWaypointHere(ctx context.Context) {
	tc, ok := r.d.Mover.(WaypointToucher)
	s := r.latest()
	if !ok || s == nil || s.Player.InTown() || !r.d.Plan.Waypoints[s.Player.Area] {
		return
	}
	area, key := s.Player.Area, wpKey{diffOf(s), s.Player.Area}
	r.loop.wp.init()
	now := r.e.Clock.Now()
	if r.loop.wp.tries[key] >= wpTouchTries || now.Before(r.loop.wp.next[key]) {
		return
	}
	if active, _ := r.activeWaypoints(s); active[area] {
		return
	}
	if r.hostileNear(s, r.e.Tune.Bot.ClearRadius) > 0 {
		return
	}
	r.loop.wp.next[key] = now.Add(wpTouchGap)
	touched, err := tc.ActivateWaypointNear(ctx, wpTouchMaxDist)
	switch {
	case err != nil:
		if ctx.Err() == nil {
			r.loop.wp.tries[key]++
			r.e.Log.Info("waypoint activation failed", "area", area, "name", r.areaName(area), "err", err)
		}
	case touched:
		r.markTouched(s, area)
		r.e.Log.Info(fmt.Sprintf("wp: activated %d", area), "name", r.areaName(area), "how", "in passing")
	}
}

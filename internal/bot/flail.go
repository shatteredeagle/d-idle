package bot

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goals"
)

// flailGuard catches any area ping-pong whatever the cause (Dark Wood 2026-11-18: Dark Wood, camp, Dark Wood, Cold
// Plains every 7 s for 12 minutes while the quest board, the area-done trip and the waypoint fallback undid each
// other). It looks only at the symptom: FlailMoves area changes within FlailWindowMin minutes, mostly into areas
// already visited in the window, with no real progress (kill, XP, level) in between. Then the bot holds where it is for FlailHoldMin minutes, doubled per repeat.
type flailGuard struct {
	mu        sync.Mutex
	moves     []flailMove
	holdUntil time.Time
	holdEnded time.Time
	streak    int
}

type flailMove struct {
	at   time.Time
	area int
	real uint64 // observer progress counter (kills, XP, levels) when the move happened
}

func (g *flailGuard) noteMove(now time.Time, area int, real uint64) {
	g.mu.Lock()
	g.moves = append(g.moves, flailMove{now, area, real})
	if len(g.moves) > 200 {
		g.moves = g.moves[len(g.moves)-200:]
	}
	g.mu.Unlock()
}

// check returns the cycle text when a flail starts now, and whether the bot is in a hold (new or running).
func (g *flailGuard) check(now time.Time, real uint64, moves int, window, hold time.Duration, name func(int) string) (cycle string, started, holding bool) {
	if moves <= 0 {
		return "", false, false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if now.Before(g.holdUntil) {
		return "", false, true
	}
	if !g.holdUntil.IsZero() {
		g.holdEnded, g.holdUntil = g.holdUntil, time.Time{}
		g.moves = nil // a fresh window after the hold
	}
	cut := 0
	for cut < len(g.moves) && now.Sub(g.moves[cut].at) > window {
		cut++
	}
	g.moves = g.moves[cut:]
	if len(g.moves) < moves || g.moves[0].real != real {
		return "", false, false
	}
	// A walk through new areas is travel, not a loop: a loop revisits, so at most half the moves are distinct areas.
	distinct := map[int]bool{}
	for _, m := range g.moves {
		distinct[m.area] = true
	}
	if len(distinct)*2 > len(g.moves) {
		return "", false, false
	}
	if now.Sub(g.holdEnded) > 12*hold {
		g.streak = 0
	}
	g.streak++
	g.holdUntil = now.Add(hold << min(g.streak-1, 3))
	seen := map[string]int{}
	for _, m := range g.moves {
		seen[name(m.area)]++
	}
	var names []string
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return fmt.Sprintf("%s: %d area changes in %d min, no kill or XP", strings.Join(names, " <-> "), len(g.moves), int(now.Sub(g.moves[0].at)/time.Minute)+1), true, true
}

// flailHold is the one hook in runGoal: while a flail hold runs, stay in the current safe area (fight in the field,
// rest in town) instead of deciding another move. A hurt hero is not held (the heal trip is the way out).
func (r *Runner) flailHold(ctx context.Context, snap *d2.Snapshot, c control) bool {
	t := r.e.Tune.Bot
	r.obMu.Lock()
	real := r.ob.real
	r.obMu.Unlock()
	cycle, started, holding := r.loop.flail.check(r.e.Clock.Now(), real, t.FlailMoves, time.Duration(t.FlailWindowMin)*time.Minute,
		time.Duration(t.FlailHoldMin)*time.Minute, r.areaName)
	if started {
		r.e.Log.Warn("flailing: holding", "cycle", cycle, "holdMin", t.FlailHoldMin)
		r.emit(goals.Event{Type: goals.EvInfo, Message: "flailing: " + cycle})
	}
	if !holding || snap.Player.LifePct() < 50 {
		return false
	}
	if snap.Player.InTown() {
		r.setState(StateInTown, "Holding in town, flailing: area changes without progress")
		r.pause(ctx, 5*time.Second)
		return false
	}
	if a := snap.Player.Area; r.loop.cleared[a] && r.hostileNear(snap, r.e.Tune.Bot.ClearRadius) == 0 {
		// Nothing left to fight here in this game: sweeping again acts in no virtual time, so under lockstep the hold
		// never ran out (cain.chain seed 11, TASK-1103: 'area cleared' every 4 ms for 33 wall minutes). Wait instead.
		r.setState(StateClearing, "Holding in "+r.areaName(a)+", flailing: the area is cleared")
		r.pause(ctx, 5*time.Second)
		return true
	}
	return r.sweep(ctx, snap, c, snap.Player.Area)
}

package bot

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/plandata"
)

// Checkpoint portals (card checkpoint1, docs/design/danger-protocol-2026-11-25.md). A town portal cast on purpose at a
// danger area while the hero is healthy and alone is a checkpoint between two waypoints: the retreat at low life and the
// way back after a death. Off by default (bot.checkpointEnable). One record per game: the engine keeps one portal per
// player, so a new cast anywhere replaces it.

type cpState int

const (
	cpNone   cpState = iota // no checkpoint
	cpSet                   // a portal stands at (area, x, y)
	cpUse                   // the hero is walking to it to retreat
	cpTown                  // the hero went home alive; the portal stands if it entered through it
	cpReturn                // the way back after a death goes through it
	cpDead                  // the hero died with the record set
)

func (s cpState) String() string {
	return [...]string{"none", "set", "use", "town", "return", "dead"}[s]
}

// cpRec is the checkpoint record: area, cell, time, scrolls left after the cast.
type cpRec struct {
	Area    int
	X, Y    int
	At      time.Time
	Scrolls int
}

// checkpointState is the Runner's checkpoint memory. Its own lock: the observer calls into it while holding obMu, so
// nothing here takes obMu.
type checkpointState struct {
	mu       sync.Mutex
	st       cpState
	rec      cpRec
	lastCast time.Time
	arrive   struct { // where the hero entered the area it stands in now
		area, x, y int
	}
	last struct { // the latest field position, for the alive trip home
		area, x, y int
	}
	idle struct { // the last `checkpoint idle` line (noteIdle)
		area int
		why  string
		at   time.Time
	}
}

// cpAction is what the guard asks the main loop to do.
type cpAction int

const (
	cpNothing cpAction = iota
	cpCast             // drop a checkpoint portal here
	cpRetreat          // walk to the checkpoint and go home through it
)

// cpIn is everything cpDecide looks at.
type cpIn struct {
	Now          time.Time
	Area, X, Y   int
	LifePct      int
	ArrX, ArrY   int // where the hero entered this area
	HostilesNear int // within CheckpointSafeRadius
	Adjacent     int // within melee reach
	Scrolls      int // portal casts in the pack
	DangerArea   bool
	NoTP         bool // a step forbids a portal (Ancients)
	Rec          cpRec
	State        cpState
	LastCast     time.Time
}

func dist(ax, ay, bx, by int) float64 { return math.Hypot(float64(ax-bx), float64(ay-by)) }

// cpDecide is the pure rule. Cast: a danger area, healthy, nothing hostile near, near the entry, the gap passed, scrolls
// above the reserve and no record for this area yet. Retreat: a record in this area within reach, life at or below the
// retreat line and not surrounded (a walk away from three attackers is not an escape; the potion and crowd rules act first).
func cpDecide(in cpIn, t cpTune) cpAction {
	if t.Enable == 0 || in.NoTP {
		return cpNothing
	}
	if in.State == cpSet && in.Rec.Area == in.Area && in.LifePct <= t.RetreatPct &&
		in.Adjacent < crowdAdjacentRetreat && dist(in.X, in.Y, in.Rec.X, in.Rec.Y) <= float64(t.Reach) {
		return cpRetreat
	}
	if cpHeld(in, t) != "" {
		return cpNothing
	}
	return cpCast
}

// cpHeld names why no checkpoint is cast here ("" = cast). Only the first reason is named, in the rule's order.
func cpHeld(in cpIn, t cpTune) string {
	switch {
	case !in.DangerArea:
		return "area not in the checkpoint table"
	case in.LifePct < t.MinLifePct:
		return "life under the cast line"
	case in.HostilesNear > 0:
		return "hostiles near"
	case in.Scrolls <= t.MinScrolls:
		return "scroll reserve"
	case in.State == cpSet && in.Rec.Area == in.Area:
		return "checkpoint already set here"
	case !in.LastCast.IsZero() && in.Now.Sub(in.LastCast) < time.Duration(t.GapSec)*time.Second:
		return "cast gap"
	case dist(in.X, in.Y, in.ArrX, in.ArrY) > float64(t.EntryDist):
		return "far from the entry"
	}
	return ""
}

// cpIdleGap is the least time between two `checkpoint idle` lines with the same area and reason.
const cpIdleGap = 10 * time.Minute

// noteIdle writes the evidence line `checkpoint idle` when the enabled rule looked at a field snapshot and cast
// nothing, once per area and reason and cpIdleGap: a window without any checkpoint line then means the rule never
// ran, and one with only idle lines says why it never cast (docs/probes/graduations-2026-11-28.md).
func (r *Runner) noteIdle(in cpIn, t cpTune) {
	why := cpHeld(in, t)
	if why == "" || in.NoTP || r.e.Log == nil {
		return
	}
	c := &r.cp
	c.mu.Lock()
	same := c.idle.area == in.Area && c.idle.why == why && in.Now.Sub(c.idle.at) < cpIdleGap
	if !same {
		c.idle.area, c.idle.why, c.idle.at = in.Area, why, in.Now
	}
	c.mu.Unlock()
	if !same {
		r.e.Log.Info("checkpoint idle", "why", why, "area", in.Area, "life", in.LifePct, "scrolls", in.Scrolls, "hostiles", in.HostilesNear)
	}
}

// crowdAdjacentRetreat is the adjacent count from which the retreat walk is not tried (the crowd rule's surrounded).
const crowdAdjacentRetreat = 3

// cpTune is the checkpoint knobs, copied out of the tuning so cpDecide stays pure.
type cpTune struct {
	Enable, MinLifePct, RetreatPct, GapSec, MinScrolls, SafeRadius, EntryDist, Reach, ArrivalHostiles int
}

func (r *Runner) cpTune() cpTune {
	b := r.e.Tune.Bot
	return cpTune{b.CheckpointEnable, b.CheckpointMinLifePct, b.CheckpointRetreatPct, b.CheckpointGapSec, b.CheckpointMinScrolls,
		b.CheckpointSafeRadius, b.CheckpointEntryDist, b.CheckpointReach, b.CheckpointArrivalHostiles}
}

func (c *checkpointState) toLocked(r *Runner, st cpState, why string) {
	if c.st == st {
		return
	}
	if r != nil && r.e.Log != nil {
		r.e.Log.Info("checkpoint", "from", c.st.String(), "to", st.String(), "why", why, "area", c.rec.Area)
	}
	c.st = st
}

// cpReset drops the record (a new game, a replaced portal, a failed entry).
func (r *Runner) cpReset(why string) {
	c := &r.cp
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.st != cpNone {
		c.toLocked(r, cpNone, why)
	}
	c.rec = cpRec{}
	c.arrive.area = 0
}

// cpScrolls is the number of portal casts in the pack: scrolls plus tome charges (combat.PortalQty).
func (r *Runner) cpScrolls(s *d2.Snapshot) int {
	if r.d.Catalog == nil || s == nil || s.Inventory == nil {
		return 0
	}
	var st combat.Stats
	if r.e.Read != nil {
		st = r.e.Read.UnitStats
	}
	cat := portalClasses{r.d.Catalog}
	n := 0
	for i := range s.Inventory.Items {
		u := &s.Inventory.Items[i]
		if u.Item == nil || u.Item.Where != d2.ContInventory || !cat.Item(u.Class).Portal {
			continue
		}
		n += combat.PortalQty(u, cat, st)
	}
	return n
}

// cpNote follows the hero for the record: where it entered the area and its latest field position. Called from the
// observer on every in-game snapshot.
func (r *Runner) cpNote(s *d2.Snapshot) {
	if r.e.Tune.Bot.CheckpointEnable == 0 || s == nil || s.Player == nil {
		return
	}
	p := s.Player
	c := &r.cp
	c.mu.Lock()
	defer c.mu.Unlock()
	if p.InTown() {
		// alive in town: the field the hero came from decides whether the portal it entered is ours
		if c.st == cpSet || c.st == cpUse {
			if dist(c.last.x, c.last.y, c.rec.X, c.rec.Y) <= cpEntryNear && c.last.area == c.rec.Area {
				c.toLocked(r, cpTown, "went home through the checkpoint")
			} else {
				c.toLocked(r, cpNone, "went home another way: the portal there is not ours any more")
				c.rec = cpRec{}
			}
		}
		c.last.area = 0
		return
	}
	if c.arrive.area != p.Area {
		c.arrive.area, c.arrive.x, c.arrive.y = p.Area, p.X, p.Y
	}
	c.last.area, c.last.x, c.last.y = p.Area, p.X, p.Y
	if c.st == cpTown && c.rec.Area == p.Area {
		c.toLocked(r, cpSet, "back at the checkpoint")
	}
}

// cpDied marks the death in the record.
func (r *Runner) cpDied() {
	c := &r.cp
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.st == cpSet || c.st == cpUse || c.st == cpTown {
		c.toLocked(r, cpDead, "the hero died")
	}
}

// checkpointDue is the guard's look (field only, no hostile fight in progress): what, if anything, the loop should do.
func (r *Runner) checkpointDue(s *d2.Snapshot, now time.Time) cpAction {
	t := r.cpTune()
	if t.Enable == 0 || s == nil || s.Player == nil || s.Player.InTown() || s.Player.MaxLife <= 0 {
		return cpNothing
	}
	p := s.Player
	_, danger := plandata.CheckpointArea(p.Area)
	c := &r.cp
	c.mu.Lock()
	in := cpIn{Now: now, Area: p.Area, X: p.X, Y: p.Y, LifePct: p.Life * 100 / p.MaxLife, DangerArea: danger,
		NoTP: r.noTP.Load(), Rec: c.rec, State: c.st, LastCast: c.lastCast, ArrX: c.arrive.x, ArrY: c.arrive.y}
	if c.arrive.area != p.Area {
		in.ArrX, in.ArrY = p.X, p.Y
	}
	c.mu.Unlock()
	in.HostilesNear = r.hostileNear(s, t.SafeRadius)
	in.Adjacent = r.hostileNear(s, r.e.Tune.Combat.CrowdAdjacent)
	in.Scrolls = r.cpScrolls(s)
	a := cpDecide(in, t)
	if a == cpNothing {
		r.noteIdle(in, t)
	}
	return a
}

// checkpointStep carries the guard's ask out in the main loop: it reports whether it acted.
func (r *Runner) checkpointStep(ctx context.Context, snap *d2.Snapshot, c control) bool {
	if snap == nil || snap.Player == nil || r.e.Tune.Bot.CheckpointEnable == 0 {
		return false
	}
	now := r.e.Clock.Now()
	switch r.checkpointDue(snap, now) {
	case cpCast:
		p := snap.Player
		ok := r.do(ctx, c, StateClearing, "Dropping a checkpoint portal", sec(r.cfg.TownTimeoutSec), "checkpoint", func(ctx context.Context) error {
			return r.d.Fighter.Act(ctx, SafetyTownPortal)
		})
		k := &r.cp
		k.mu.Lock()
		k.lastCast = now // a failed cast waits out the gap too: no scroll storm
		if ok {
			k.rec = cpRec{Area: p.Area, X: p.X, Y: p.Y, At: now, Scrolls: r.cpScrolls(snap) - 1}
			k.toLocked(r, cpSet, fmt.Sprintf("cast at (%d,%d), scrolls left %d", p.X, p.Y, k.rec.Scrolls))
		}
		k.mu.Unlock()
		return true
	case cpRetreat:
		k := &r.cp
		k.mu.Lock()
		rec := k.rec
		k.toLocked(r, cpUse, "life low, walking to the checkpoint")
		k.mu.Unlock()
		r.e.Log.Info("checkpoint retreat", "area", rec.Area, "life", snap.Player.Life, "maxLife", snap.Player.MaxLife)
		r.do(ctx, c, StateClearing, "Retreating to the checkpoint portal", sec(r.cfg.TravelTimeoutSec), "checkpoint retreat", func(ctx context.Context) error {
			return r.d.Mover.GoTo(ctx, input.Point{X: rec.X, Y: rec.Y})
		})
		// ToTown enters the open portal beside the hero (docs/d2d-portal-entry.md), the trip returns through it
		r.loop.trip = &trip{plan: r.fullChores("checkpoint retreat"), returnTo: rec.Area, emergency: true}
		r.setLastTrip(now)
		return true
	}
	return false
}

// cpNewCast is called when a safety action opens a portal other than the checkpoint: the engine keeps one portal.
func (r *Runner) cpNewCast(a SafetyAction) {
	if a == SafetyTownPortal || a == SafetyCrowdAbandon {
		r.cpReset("a new portal replaced the checkpoint")
	}
}

var errCheckpointCrowd = errors.New("bot: crowd at the checkpoint exit")

// cpRoute decides how the way back to the corpse area starts (gotoDeathArea). With the feature on and a usable record in
// that area (not a stale pack, the area is allowed) the standing town portal is the way: portalFrom is set to it. Without
// one the waypoint and the walk go first: the stale portal of an earlier trip is not trusted. It reports whether the
// checkpoint is the route.
func (r *Runner) cpRoute(s *d2.Snapshot, area int) bool {
	if r.e.Tune.Bot.CheckpointEnable == 0 {
		return false
	}
	c := &r.cp
	c.mu.Lock()
	use := (c.st == cpDead || c.st == cpTown || c.st == cpSet) && c.rec.Area == area
	c.mu.Unlock()
	why := ""
	if use && s != nil {
		if r.stalePackArea(s) == area {
			use, why = false, "stale pack at the exit"
		} else if w := r.areaAllowed(s, area); w != "" {
			use, why = false, w
		}
	}
	if use {
		r.obMu.Lock()
		r.ob.portalFrom = area
		r.obMu.Unlock()
		c.mu.Lock()
		c.toLocked(r, cpReturn, "the way back goes through the checkpoint")
		c.mu.Unlock()
		return true
	}
	if why == "" {
		why = "no usable record: waypoint, then walk"
	}
	r.obMu.Lock()
	r.ob.portalFrom = 0
	r.obMu.Unlock()
	r.cpReset(why)
	return false
}

// cpArrival runs after a return through the checkpoint: a crowd near the exit sends the hero straight back to town.
func (r *Runner) cpArrival(ctx context.Context) error {
	t := r.cpTune()
	c := &r.cp
	c.mu.Lock()
	st := c.st
	c.mu.Unlock()
	if st != cpReturn {
		return nil
	}
	s := r.poll()
	if s == nil || s.Player == nil || s.Player.InTown() {
		return nil
	}
	n := r.hostileNear(s, r.e.Tune.Bot.ClearRadius)
	if n < t.ArrivalHostiles {
		c.mu.Lock()
		c.toLocked(r, cpSet, "arrived through the checkpoint")
		c.mu.Unlock()
		return nil
	}
	r.e.Log.Warn("checkpoint exit is crowded: back to town at once", "hostiles", n)
	r.notePack(s, n, r.e.Clock.Now())
	r.cpReset("crowd at the exit")
	if err := r.d.Town.ToTown(ctx); err != nil {
		return fmt.Errorf("%w: %v", errCheckpointCrowd, err)
	}
	return errCheckpointCrowd
}

// cpEntryNear is how close to the portal the hero was when it went home for the trip to count as entering the checkpoint
// (ToTown enters an open portal beside the hero; the retreat walk ends on its cell).
const cpEntryNear = 12

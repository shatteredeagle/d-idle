// Package nav walks the character: grid A* over the d2map collision grid, a click-to-move executor with stuck
// recovery, level exits, waypoints and frontier exploration. See docs/nav.md.
package nav

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
)

var (
	ErrStuck        = errors.New("nav: stuck")
	ErrNoPath       = errors.New("nav: no path")
	ErrUnitGone     = errors.New("nav: unit not found")
	ErrTimeout      = errors.New("nav: timeout")
	ErrNoExit       = errors.New("nav: no exit to that level")
	ErrExplored     = errors.New("nav: nothing left to explore")
	ErrNoWaypoint   = errors.New("nav: waypoint unavailable")
	ErrNotInGame    = errors.New("nav: not in game")
	ErrAborted      = errors.New("nav: aborted")
	ErrDead         = errors.New("nav: character is dead")
	ErrDanger       = errors.New("nav: monsters near")
	ErrLevelChanged = errors.New("nav: the level changed during the walk")
	errLoading      = errors.New("nav: loading")
	errNothingAhead = errors.New("nav: no target")
)

// worldSource is what nav needs from d2map.Tracker.
type worldSource interface {
	Update() (d2map.State, error)
	Current() *d2map.LevelMap
	Level(no int) *d2map.LevelMap
}

// stateSource is what nav needs from d2.Reader.
type stateSource interface {
	Snapshot() (*d2.Snapshot, error)
}

// Navigator implements the Mover contract of docs/archive/wave2.md. It is not safe for concurrent use: one goroutine
// drives the character, and nav is the one calling Tracker.Update.
type Navigator struct {
	cov *coverage // roam state (roam.go); nil on backends that reveal unknown cells
	// Guard, if set, runs on every observation while walking. A non-nil error stops the walk and is returned
	// (wrapped with ErrAborted). The bot uses it for "life low or monster near: stop".
	Guard func(*d2.Snapshot) error
	// Yield, if set, is called when a walk has stalled with living monsters close by (a pack in the way). The bot
	// installs a combat clear here; nav carries on (replanning) when it returns nil, and a non-nil error ends the
	// walk. Without it nav retreats from the pack and plans around it.
	Yield func(ctx context.Context, near []d2.Unit) error
	// TeleportKey, if set, is the hotkey of Teleport: Explore and ExitTo then hop with CastAt (teleport.go).
	TeleportKey   string
	TeleportRange float64 // subtiles per cast, 0: 18
	TeleportMana  int     // mana one cast needs, 0: 24
	// Refill, if set, is called while a cast waits for mana (drink a potion, rest); without it nav sleeps.
	Refill func(ctx context.Context, mana, need int) error

	// ClearCursor, if set, puts a held item into the pack; a waypoint panel does not open with an item on the cursor.
	ClearCursor func(ctx context.Context) error
	// Escape, if set, takes the character out of a place it cannot walk out of (the bot installs a town portal).
	// ExitTo calls it once when a walk to the exit keeps stalling after other approach cells failed; the caller
	// then re-plans from the new level (ErrLevelChanged).
	Escape func(ctx context.Context) error

	// Hint biases informed exploration toward a point (zero value: none). Set with ExploreToward.
	hint     *input.Point
	caution  float64 // multiplier on monster avoidance (1, or 2 during CorpseApproach)
	wantExit int     // level whose exit tiles ExitTo is deliberately walking onto (0: none)
	wpLevel  int     // level of the last observation and when it was first seen: the waypoint cooldown (waypoint.go)
	wpSince  time.Time
	wpClicks int // interactions and row clicks of the current TakeWaypoint

	e          *env.Env
	w          worldSource
	st         stateSource
	in         input.Controller
	clk        env.Clock
	log        *slog.Logger
	inMouth    bool // mouthStep walk in progress (walk.go)
	mouthGuard bool // NearMouth answers (d2d only, see SetMouthGuard)
	tune       func() env.NavTuning
	pl         Planner
	planMu     sync.Mutex // planHard mutates the shared Field (dyn costs, passMask) and n.black: one plan at a time
	rng        *rand.Rand

	field     *Field
	fieldKey  fieldKey
	black     []blocked
	hard      []blocked    // cells the planner must not cross for a while (stuckmem.go)
	spots     []*stuckSpot // recent stalls (stuckmem.go)
	walkSeq   int
	escaping  bool
	teleBad   map[[2]int]int  // hop cells where nothing happened
	teleFails int             // consecutive explore hops that did not move
	visited   map[[2]int]bool // exploration buckets the player has stood in
	failed    map[[2]int]int  // frontier buckets that did not pan out
	badGates  []badGate
	noExit    []noExitMark                                                         // ExitTo gave up exploring for an exit: see exitCooldown
	exploreFn func(ctx context.Context, done func() bool, hint *input.Point) error // test seam, nil: explore
	seen      map[edgeKey]bool                                                     // level crossings made (Route)
	bad       map[edgeKey]bool                                                     // guessed edges that turned out not to exist
}

type fieldKey struct {
	grid     *d2map.Grid
	known    int
	clear    float64
	weight   float64
	lvl      int
	rect     d2map.Rect
	nonce    int
	hasField bool
}

type blocked struct {
	at      pt
	expires time.Time
	r       float64 // hard blocks only: radius in subtiles
}

// New builds a Navigator from the instance Env.
func New(e *env.Env) *Navigator {
	n := &Navigator{e: e, in: e.In, clk: e.Clock, log: e.Log, rng: rand.New(rand.NewSource(1))}
	if e.World != nil {
		n.w = e.World
	}
	if e.Read != nil {
		n.st = e.Read
	}
	n.tune = func() env.NavTuning {
		if e.Tune == nil {
			return env.DefaultNavTuning()
		}
		return e.Tune.Nav
	}
	n.defaults()
	return n
}

// UseWorld replaces the map source New took from env.Env.World (a *d2map.Tracker). It lets a backend with its own
// level maps (d2dworld) drive nav without a Tracker; the Game.exe path never calls it.
func (n *Navigator) UseWorld(w interface {
	Update() (d2map.State, error)
	Current() *d2map.LevelMap
	Level(no int) *d2map.LevelMap
}) {
	n.w = w
}

func (n *Navigator) defaults() {
	if n.clk == nil {
		n.clk = env.RealClock{}
	}
	if n.log == nil {
		n.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if n.rng == nil {
		n.rng = rand.New(rand.NewSource(1))
	}
	n.caution = 1
	n.visited = map[[2]int]bool{}
	n.failed = map[[2]int]int{}
	n.teleBad = map[[2]int]int{}
}

// obs is one observation of the world.
type obs struct {
	snap  *d2.Snapshot
	state d2map.State
	lm    *d2map.LevelMap
	pos   input.Point
	level int
}

// observe reads the game once and advances the map tracker. errLoading means "not in a level right now".
func (n *Navigator) observe() (*obs, error) {
	snap, err := n.st.Snapshot()
	if err != nil {
		return nil, err
	}
	if snap.State != d2.StateInGame || snap.Player == nil {
		return nil, errLoading
	}
	st, err := n.w.Update()
	if err != nil {
		if errors.Is(err, d2map.ErrNotInGame) {
			return nil, errLoading
		}
		return nil, err
	}
	lm := n.w.Current()
	if lm == nil {
		return nil, errLoading
	}
	n.noteLevel(snap)
	return &obs{snap: snap, state: st, lm: lm, pos: input.Point{X: snap.Player.X, Y: snap.Player.Y}, level: snap.Player.Area}, nil
}

// observeWait is observe that waits out loading screens.
func (n *Navigator) observeWait(ctx context.Context) (*obs, error) {
	t := n.tune()
	deadline := n.clk.Now().Add(t.LoadWait())
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		o, err := n.observe()
		if err == nil {
			return o, nil
		}
		if !errors.Is(err, errLoading) {
			return nil, err
		}
		if n.clk.Now().After(deadline) {
			return nil, fmt.Errorf("%w: still loading after %v", ErrNotInGame, t.LoadWait())
		}
		n.clk.Sleep(t.Poll())
	}
}

// dead reports the player's dead or dying state: mode 0 (death) or 17 (dead), VERIFIED LIVE (docs/combat.md
// "Death"). Zero life in a living mode is not death: a stale save loads that way (i4 Booti, 2026-10-28) and the
// character can still walk to a healer, so it must not block navigation.
func dead(o *obs) bool {
	p := o.snap.Player
	return p.Mode == 0 || p.Mode == 17
}

func (n *Navigator) guard(o *obs) error {
	if dead(o) {
		return ErrDead // hard stop: never keep clicking a corpse
	}
	if n.Guard == nil {
		return nil
	}
	if err := n.Guard(o.snap); err != nil {
		return fmt.Errorf("%w: %w", ErrAborted, err)
	}
	return nil
}

func dist(a, b input.Point) float64 { return math.Hypot(float64(a.X-b.X), float64(a.Y-b.Y)) }

// fieldFor returns the planner field for the level's grid, rebuilt when new cells became known.
func (n *Navigator) fieldFor(lm *d2map.LevelMap) *Field {
	t := n.tune()
	k := fieldKey{grid: lm.Grid, known: lm.Grid.KnownCount(), clear: t.WallClearance, weight: t.WallWeight, lvl: lm.No, rect: lm.Rect, nonce: n.fieldKey.nonce, hasField: true}
	if n.field != nil && n.fieldKey == k {
		return n.field
	}
	n.field = buildField(lm.Grid, lm.Rect, t.WallClearance, t.WallWeight)
	n.fieldKey = k
	return n.field
}

// invalidate forces the next plan to re-read the grid (a door opened, for example).
func (n *Navigator) invalidate() { n.fieldKey.nonce++ }

type planResult struct {
	wps        []input.Point // waypoints from the first step to the goal
	partial    bool          // the goal is unreachable or unknown: wps lead to the nearest reachable point
	goal       input.Point   // where the last waypoint is (the snapped goal)
	snapped    bool          // goal was moved to a walkable cell
	cells      int           // length of the raw A* path
	optimistic bool          // the path crosses unknown cells assumed walkable; replan as rooms load
	known      int           // KnownCount when planned
}

// plan finds waypoints from start to goal on lm's current grid. ignore is a unit id left out of the monster
// penalties (the unit being walked to).
func (n *Navigator) plan(lm *d2map.LevelMap, start, goal input.Point, tol float64, units []d2.Unit, ignore uint32) (planResult, error) {
	res, err := n.planHard(lm, start, goal, tol, units, ignore, true)
	if (err != nil || res.partial) && len(n.hard) > 0 {
		// the hard blocks may have cut the only way: better to try the old way again than to give up
		if r2, e2 := n.planHard(lm, start, goal, tol, units, ignore, false); e2 == nil && !r2.partial {
			return r2, nil
		}
	}
	return res, err
}

func (n *Navigator) planHard(lm *d2map.LevelMap, start, goal input.Point, tol float64, units []d2.Unit, ignore uint32, hard bool) (planResult, error) {
	n.planMu.Lock()
	defer n.planMu.Unlock()
	t := n.tune()
	f := n.fieldFor(lm)
	res := planResult{known: lm.Grid.KnownCount()}
	if f.W == 0 {
		return res, ErrNoPath
	}
	f.ClearDyn()
	if t.MonsterRadius > 0 && t.MonsterPenalty > 0 {
		for i := range units {
			u := &units[i]
			if u.ID == ignore || !u.Alive() || u.Type != d2.TypeMonster {
				continue
			}
			f.AddDisc(u.X, u.Y, t.MonsterRadius*n.caution, t.MonsterPenalty*n.caution)
		}
	}
	now := n.clk.Now()
	live := n.black[:0]
	for _, b := range n.black {
		if b.expires.After(now) {
			live = append(live, b)
			f.AddDisc(b.at.x, b.at.y, 2.5, 30)
		}
	}
	n.black = live

	if hard {
		n.applyHard(f, start, goal)
	}
	n.penalizeExits(f, lm, goal)

	sx, sy, ok := f.nearestWalkable(start.X, start.Y, 8)
	if !ok {
		return res, fmt.Errorf("%w: start %v is not on known walkable ground", ErrNoPath, start)
	}
	gx, gy := goal.X, goal.Y
	if f.Known(gx, gy) && !f.Walkable(gx, gy) {
		if x, y, ok := f.nearestWalkable(gx, gy, 12); ok {
			gx, gy, res.snapped = x, y, true
		}
	}
	q := math.Min(1, math.Max(0, t.PathQuality))
	hw := 1 + 3*(1-q)
	reach := math.Max(0, tol*0.8)
	if res.snapped {
		reach = 0
	}
	var cells []int32
	var partial bool
	goalUnknown := f.idx(gx, gy) >= 0 && f.flags[f.idx(gx, gy)]&fKnown == 0
	f.passMask = fWalk
	defer func() { f.passMask = fWalk }()
	if !goalUnknown {
		cells, partial = n.pl.Plan(f, sx, sy, gx, gy, reach, hw)
	}
	if goalUnknown || partial {
		// Nothing known leads to the goal: assume the unknown part of the level is open (extra cost) and walk
		// toward it; the executor replans as rooms load.
		f.passMask = fWalk | fOpt
		if c2, p2 := n.pl.Plan(f, sx, sy, gx, gy, reach, hw); len(c2) > 0 {
			cells, partial, res.optimistic = c2, p2, true
		} else {
			f.passMask = fWalk
		}
	}
	res.partial, res.cells = partial, len(cells)
	if len(cells) == 0 {
		return res, ErrNoPath
	}
	pts := f.points(cells)
	if partial && len(pts) < 2 {
		return res, fmt.Errorf("%w: %v is unreachable from %v", ErrNoPath, goal, start)
	}
	if q >= 0.5 {
		pts = f.smooth(pts, f.pathPrefix(pts))
	}
	end := pts[len(pts)-1]
	if !partial && !res.snapped && tol <= 2.5 && (f.Walkable(goal.X, goal.Y) || res.optimistic) && (end != pt{goal.X, goal.Y}) && f.los(end.x, end.y, goal.X, goal.Y, 0) {
		pts = append(pts, pt{goal.X, goal.Y})
	}
	wps := resample(pts, math.Max(2, t.WaypointSpacing), func(a, b pt) bool { return f.los(a.x, a.y, b.x, b.y, 0) })
	if q < 1 {
		wps = f.jitter(pts[0], wps, (1-q)*t.Sloppiness, n.rng)
	}
	res.wps = toInput(wps)
	if len(res.wps) > 0 {
		res.goal = res.wps[len(res.wps)-1]
	}
	return res, nil
}

// PathTo plans waypoints from the player's position to p on the current grid without moving or advancing the
// map tracker. The path ends at p, or at the nearest walkable cell when p itself is blocked; ErrNoPath if p
// cannot be reached (an unreachable or unknown p yields ErrNoPath too: use GoTo to approach it).
func (n *Navigator) PathTo(p input.Point) ([]input.Point, error) {
	lm := n.w.Current()
	if lm == nil {
		return nil, ErrNotInGame
	}
	pos, err := n.in.Pos()
	if err != nil {
		return nil, err
	}
	var units []d2.Unit
	if snap, err := n.st.Snapshot(); err == nil && snap.State == d2.StateInGame {
		units = snap.Monsters
	}
	res, err := n.plan(lm, pos, p, n.tune().ArriveTolerance, units, 0)
	if err != nil {
		return nil, err
	}
	if res.partial || res.optimistic {
		return nil, fmt.Errorf("%w: %v from %v on the known map", ErrNoPath, p, pos)
	}
	return res.wps, nil
}

// GoTo walks to p on the current level.
func (n *Navigator) GoTo(ctx context.Context, p input.Point) error {
	t := n.tune()
	return n.walk(ctx, func(*obs) (input.Point, float64, error) { return p, t.ArriveTolerance, nil }, walkOpts{})
}

// unitTolerance is how close GoToUnit gets: inside interact range of NPCs and objects.
const unitTolerance = 4.0

// findUnit looks a unit up by id. Ids are only unique per unit type (a town NPC and the waypoint object can both
// be id 11), so typ selects the table; typ < 0 searches monsters, then objects, then ground items.
func findUnit(s *d2.Snapshot, typ int, id uint32) *d2.Unit {
	lists := [][]d2.Unit{s.Monsters, s.Objects, s.Ground}
	switch typ {
	case d2.TypeMonster:
		lists = lists[:1]
	case d2.TypeObject:
		lists = lists[1:2]
	case d2.TypeItem:
		lists = lists[2:]
	}
	for _, list := range lists {
		for i := range list {
			if list[i].ID == id {
				return &list[i]
			}
		}
	}
	return nil
}

// GoToUnit is GoToUnitOf with an unknown unit type: monsters/NPCs win over objects over items on an id clash.
// Prefer GoToUnitOf.
func (n *Navigator) GoToUnit(ctx context.Context, unitID uint32) error {
	return n.GoToUnitOf(ctx, -1, unitID)
}

// GoToUnitOf walks next to the unit of the given d2 type (TypeMonster, TypeObject, TypeItem) and id, following
// it if it moves. ErrUnitGone if it is not in the unit tables for a full second.
func (n *Navigator) GoToUnitOf(ctx context.Context, typ int, unitID uint32) error {
	var missingSince time.Time
	tgt := func(o *obs) (input.Point, float64, error) {
		u := findUnit(o.snap, typ, unitID)
		if u == nil {
			if missingSince.IsZero() {
				missingSince = n.clk.Now()
			}
			if n.clk.Now().Sub(missingSince) > time.Second {
				return input.Point{}, 0, ErrUnitGone
			}
			return o.pos, 1e9, errNothingAhead // wait a tick
		}
		missingSince = time.Time{}
		return input.Point{X: u.X, Y: u.Y}, unitTolerance, nil
	}
	return n.walk(ctx, tgt, walkOpts{ignoreUnit: unitID})
}

// exitAvoidR and exitAvoidPeak shape the penalty around a level-exit tile that is not the destination. V live (D2DLOOP,
// d2d Blood Moor, Den of Evil mouth, `cmd move` from the arrival point): the trigger area reaches about 2 tiles (10
// subtiles) from the warp's position, south and west of it; one tile east, north and the diagonals were safe. The peak
// stays below the 250 cap of the dynamic cost (peak * costUnit 8) so the penalty keeps a gradient: a plateau at the cap
// gave the planner no direction away from the mouth and it walked straight over it.
const (
	exitAvoidR    = 14.0
	exitAvoidPeak = 31.0
)

// penalizeExits makes the cells around every tile exit costly, so a walk to somewhere else does not step on one by
// accident (live: a walk to the Far Oasis waypoint entered Maggot Lair). Exits to wantExit and exits close to the goal
// (the walk's own target) are left alone. It is a cost, not a wall: when the only way leads over an exit the path
// still goes there.
func (n *Navigator) penalizeExits(f *Field, lm *d2map.LevelMap, goal input.Point) {
	for _, e := range lm.Exits {
		if e.Dest == n.wantExit {
			continue
		}
		p := input.Point{X: e.Pos.X, Y: e.Pos.Y}
		if dist(p, goal) <= exitAvoidR+2 {
			continue
		}
		f.AddDisc(p.X, p.Y, exitAvoidR, exitAvoidPeak)
	}
}
